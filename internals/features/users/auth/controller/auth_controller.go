package controller

import (
	masjidAdminModel "masjidku_backend/internals/features/lembaga/teachers_students/model"
	"masjidku_backend/internals/features/users/auth/service"
	models "masjidku_backend/internals/features/users/user/model"
	"strings"
	"time"

	userClassModel "masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthController struct {
	DB *gorm.DB
}

func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{DB: db}
}

// file: internals/features/auth/controller/auth_controller.go
func (ac *AuthController) Me(c *fiber.Ctx) error {
	// --- Guard user ---
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID in context")
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid UUID format")
	}

	// ===== Users: table plural, column singular =====
	var user models.UserModel
	if err := ac.DB.First(&user, "user_id = ?", userUUID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	user.Password = nil

	// Parse ?include=union,class_ids (opsional)
	includeQ := strings.TrimSpace(c.Query("include"))
	wantUnion := false
	wantClassIDs := false
	if includeQ != "" {
		for _, part := range strings.Split(includeQ, ",") {
			switch strings.ToLower(strings.TrimSpace(part)) {
			case "union", "masjid_union", "masjid_ids":
				wantUnion = true
			case "class_ids":
				wantClassIDs = true
			}
		}
	}

	// =========================
	// Kumpulkan asosiasi peran
	// =========================
	adminSet := map[string]struct{}{}
	teacherSet := map[string]struct{}{}
	studentSet := map[string]struct{}{}

	// ----- Teacher (kolom singular) -----
	{
		var rows []masjidAdminModel.MasjidTeacherModel
		if err := ac.DB.
			Where("masjid_teacher_user_id = ? AND masjid_teacher_deleted_at IS NULL", user.ID).
			Find(&rows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data masjid guru")
		}
		for _, r := range rows {
			teacherSet[r.MasjidTeacherMasjidID.String()] = struct{}{}
		}
	}

	// ----- Student enrolments aktif (kolom singular) -----
	type enrollRow struct {
		UserClassID uuid.UUID `gorm:"column:user_class_id"`
		ClassID     uuid.UUID `gorm:"column:user_class_class_id"`
		MasjidID    uuid.UUID `gorm:"column:user_class_masjid_id"`
	}

	var activeEnrolls []enrollRow
	if err := ac.DB.
		Table("user_classes AS uc").
		Select("uc.user_class_id, uc.user_class_class_id, uc.user_class_masjid_id").
		Joins(`JOIN masjid_students AS ms
			   ON ms.masjid_student_id = uc.user_class_masjid_student_id
			  AND ms.masjid_student_deleted_at IS NULL`).
		Where(`
			ms.masjid_student_user_id = ?
			AND uc.user_class_status = ?
			AND uc.user_class_deleted_at IS NULL
		`, user.ID, userClassModel.UserClassStatusActive).
		Find(&activeEnrolls).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil enrolment student")
	}

	for _, r := range activeEnrolls {
		studentSet[r.MasjidID.String()] = struct{}{}
	}

	toSlice := func(set map[string]struct{}) []string {
		out := make([]string, 0, len(set))
		for id := range set {
			out = append(out, id)
		}
		return out
	}
	masjidAdminIDs := toSlice(adminSet)
	masjidTeacherIDs := toSlice(teacherSet)
	masjidStudentIDs := toSlice(studentSet)

	// =========================
	// Section aktif (mapping class→section)
	// =========================
	classIDsSet := map[string]struct{}{}
	for _, e := range activeEnrolls {
		classIDsSet[e.ClassID.String()] = struct{}{}
	}
	internalClassIDs := toSlice(classIDsSet)

	classToSection := map[string]string{}
	classSectionIDsSet := map[string]struct{}{}
	if len(internalClassIDs) > 0 {
		now := time.Now()
		type row struct {
			SectionID uuid.UUID `gorm:"column:class_section_id"`
			ClassID   uuid.UUID `gorm:"column:class_section_class_id"`
		}
		var rows []row
		if err := ac.DB.Table("class_sections").
			Where(
				"class_section_class_id IN ? AND ("+
					"class_section_is_active = TRUE OR "+
					"(class_section_start <= ? AND (class_section_end IS NULL OR class_section_end >= ?))"+
					") AND class_section_deleted_at IS NULL",
				internalClassIDs, now, now,
			).
			Select("class_section_id, class_section_class_id").
			Find(&rows).Error; err == nil {
			for _, r := range rows {
				cid := r.ClassID.String()
				sid := r.SectionID.String()
				if _, ok := classToSection[cid]; !ok {
					classToSection[cid] = sid // ambil satu per class
				}
				classSectionIDsSet[sid] = struct{}{}
			}
		}
	}
	classSectionIDs := toSlice(classSectionIDsSet)

	// ---------- Response ----------
	respUser := fiber.Map{
		"id":                 user.ID, // kolom singular
		"user_name":          user.UserName,
		"email":              user.Email,
		"masjid_admin_ids":   masjidAdminIDs,
		"masjid_teacher_ids": masjidTeacherIDs,
		"masjid_student_ids": masjidStudentIDs,
	}
	respStudent := fiber.Map{
		"active_enrollments": func() []fiber.Map {
			out := make([]fiber.Map, 0, len(activeEnrolls))
			for _, e := range activeEnrolls {
				row := fiber.Map{
					"user_class_id": e.UserClassID,
					"class_id":      e.ClassID,
					"masjid_id":     e.MasjidID,
				}
				if sid, ok := classToSection[e.ClassID.String()]; ok && sid != "" {
					row["active_section_id"] = sid
				}
				out = append(out, row)
			}
			return out
		}(),
		"class_section_ids": classSectionIDs,
	}

	// Tambahan opsional bila diminta
	if wantUnion {
		union := map[string]struct{}{}
		for k := range adminSet {
			union[k] = struct{}{}
		}
		for k := range teacherSet {
			union[k] = struct{}{}
		}
		for k := range studentSet {
			union[k] = struct{}{}
		}
		respUser["masjid_ids"] = toSlice(union)
	}
	if wantClassIDs {
		respStudent["class_ids"] = internalClassIDs
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    respUser,
		"student": respStudent,
	})
}

func (ac *AuthController) UpdateUserName(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID in context")
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid UUID format")
	}

	var req struct {
		UserName string `json:"user_name" validate:"required,min=3,max=50"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Users: table plural, column singular
	if err := ac.DB.Model(&models.UserModel{}).
		Where("user_id = ?", userUUID).
		Update("user_name", req.UserName).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update user name")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Username berhasil diperbarui",
	})
}

func (ac *AuthController) Register(c *fiber.Ctx) error {
	return service.Register(ac.DB, c)
}

func (ac *AuthController) Login(c *fiber.Ctx) error {
	return service.Login(ac.DB, c)
}

func (ac *AuthController) LoginGoogle(c *fiber.Ctx) error {
	return service.LoginGoogle(ac.DB, c)
}

func (ac *AuthController) Logout(c *fiber.Ctx) error {
	return service.Logout(ac.DB, c)
}

func (pc *AuthController) ChangePassword(c *fiber.Ctx) error {
	return service.ChangePassword(pc.DB, c)
}

func (rc *AuthController) RefreshToken(c *fiber.Ctx) error {
	return service.RefreshToken(rc.DB, c)
}

func (ac *AuthController) ResetPassword(c *fiber.Ctx) error {
	return service.ResetPassword(ac.DB, c)
}
