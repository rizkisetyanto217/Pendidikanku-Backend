package controller

import (
	masjidAdminModel "masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	"masjidku_backend/internals/features/users/auth/service"
	models "masjidku_backend/internals/features/users/user/model"
	"strings"
	"time"

	userClassModel "masjidku_backend/internals/features/lembaga/classes/main/model"

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

	var user models.UserModel
	if err := ac.DB.First(&user, "id = ?", userUUID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	user.Password = ""

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

	// Admin/DKM
	{
		var rows []masjidAdminModel.MasjidAdminModel
		if err := ac.DB.
			Where("masjid_admins_user_id = ? AND masjid_admins_is_active = true", user.ID).
			Find(&rows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data masjid admin")
		}
		for _, r := range rows {
			adminSet[r.MasjidAdminsMasjidID.String()] = struct{}{}
		}
	}

	// Teacher
	{
		var rows []masjidAdminModel.MasjidTeacher // sesuaikan tipe/package
		if err := ac.DB.
			Where("masjid_teachers_user_id = ?", user.ID).
			Find(&rows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data masjid guru")
		}
		for _, r := range rows {
			teacherSet[r.MasjidTeachersMasjidID] = struct{}{}
		}
	}

	// Student → enrolment aktif (user_classes)
	type enrollRow struct {
		UserClassID uuid.UUID  `gorm:"column:user_classes_id"`
		ClassID     uuid.UUID  `gorm:"column:user_classes_class_id"`
		MasjidID    *uuid.UUID `gorm:"column:user_classes_masjid_id"`
	}

	var activeEnrolls []enrollRow
	{
		if err := ac.DB.
			Model(&userClassModel.UserClassesModel{}).
			Where(`
				user_classes_user_id = ?
				AND user_classes_status = ?
				AND user_classes_deleted_at IS NULL
			`, user.ID, userClassModel.UserClassStatusActive).
			Select("user_classes_id, user_classes_class_id, user_classes_masjid_id").
			Find(&activeEnrolls).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil enrolment student")
		}

		for _, r := range activeEnrolls {
			if r.MasjidID != nil {
				studentSet[r.MasjidID.String()] = struct{}{}
			}
		}
	}


	toSlice := func(set map[string]struct{}) []string {
		out := make([]string, 0, len(set))
		for id := range set { out = append(out, id) }
		return out
	}
	masjidAdminIDs := toSlice(adminSet)
	masjidTeacherIDs := toSlice(teacherSet)
	masjidStudentIDs := toSlice(studentSet)

	// =========================
	// Student: section aktif (mapping class→section + list section)
	// =========================
	// (kita tetap butuh classIDs internal untuk query section, tapi tidak dikirim kecuali diminta)
	classIDsSet := map[string]struct{}{}
	for _, e := range activeEnrolls { classIDsSet[e.ClassID.String()] = struct{}{} }
	internalClassIDs := toSlice(classIDsSet)

	classToSection := map[string]string{}
	classSectionIDsSet := map[string]struct{}{}
	if len(internalClassIDs) > 0 {
		now := time.Now()
		type row struct {
			SectionID uuid.UUID `gorm:"column:class_sections_id"`
			ClassID   uuid.UUID `gorm:"column:class_sections_class_id"`
		}
		var rows []row
		if err := ac.DB.Table("class_sections").
			Where(
				"class_sections_class_id IN ? AND ("+
					"class_sections_is_active = TRUE OR "+
					"(class_sections_start <= ? AND (class_sections_end IS NULL OR class_sections_end >= ?))"+
				")",
				internalClassIDs, now, now,
			).
			Select("class_sections_id, class_sections_class_id").
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

	// Bentuk enrolment minimal
	activeEnrollments := make([]fiber.Map, 0, len(activeEnrolls))
	for _, e := range activeEnrolls {
		row := fiber.Map{
			"user_class_id": e.UserClassID,
			"class_id":      e.ClassID,
			"masjid_id":     e.MasjidID, // pointer → null/string di JSON
		}
		if sid, ok := classToSection[e.ClassID.String()]; ok && sid != "" {
			row["active_section_id"] = sid
		}
		activeEnrollments = append(activeEnrollments, row)
	}

	// ---------- Response dasar (tanpa redundansi) ----------
	respUser := fiber.Map{
		"id":                 user.ID,
		"user_name":          user.UserName,
		"email":              user.Email,
		"role":               user.Role,
		"masjid_admin_ids":   masjidAdminIDs,
		"masjid_teacher_ids": masjidTeacherIDs,
		"masjid_student_ids": masjidStudentIDs,
	}
	respStudent := fiber.Map{
		"active_enrollments": activeEnrollments,
		"class_section_ids":  classSectionIDs,
	}

	// Tambahan opsional bila diminta
	if wantUnion {
		union := map[string]struct{}{}
		for k := range adminSet   { union[k] = struct{}{} }
		for k := range teacherSet { union[k] = struct{}{} }
		for k := range studentSet { union[k] = struct{}{} }
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

	if err := ac.DB.Model(&models.UserModel{}).
		Where("id = ?", userUUID).
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

func (ac *AuthController) CheckSecurityAnswer(c *fiber.Ctx) error {
	return service.CheckSecurityAnswer(ac.DB, c)
}
