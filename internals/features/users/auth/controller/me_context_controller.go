// file: internals/features/users/auth/controller/me_context_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	// ✅ path masjid sesuai struktur terbaru kamu
	masjidModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"
	classModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	userModel "masjidku_backend/internals/features/users/user/model"

	// ✅ helpers
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* =============== Lightweight link models (disesuaikan kolom) =============== */

// user_teachers: user ↔ user_teacher (ID user_teacher dipakai oleh masjid_teachers)
type UserTeacher struct {
	UserTeacherID     uuid.UUID `gorm:"column:user_teacher_id"`
	UserTeacherUserID uuid.UUID `gorm:"column:user_teacher_user_id"`
}

func (UserTeacher) TableName() string { return "user_teachers" }

// masjid_teachers: masjid ↔ user_teacher (kolom benar: masjid_teacher_user_teacher_id)
type MasjidTeacher struct {
	MasjidTeacherMasjidID      uuid.UUID `gorm:"column:masjid_teacher_masjid_id"`
	MasjidTeacherUserTeacherID uuid.UUID `gorm:"column:masjid_teacher_user_teacher_id"`
}

func (MasjidTeacher) TableName() string { return "masjid_teachers" }

// user_profiles: user ↔ user_profile (ID user_profile dipakai oleh masjid_students)
type UserProfile struct {
	UserProfileID     uuid.UUID `gorm:"column:user_profile_id"`
	UserProfileUserID uuid.UUID `gorm:"column:user_profile_user_id"`
}

func (UserProfile) TableName() string { return "user_profiles" }

// masjid_students: masjid ↔ user_profile (kolom benar: masjid_student_user_profile_id)
type MasjidStudent struct {
	MasjidStudentMasjidID      uuid.UUID `gorm:"column:masjid_student_masjid_id"`
	MasjidStudentUserProfileID uuid.UUID `gorm:"column:masjid_student_user_profile_id"`
}

func (MasjidStudent) TableName() string { return "masjid_students" }

/* =============== DTO Response =============== */
type MasjidWithSections struct {
	Masjid        masjidModel.MasjidModel        `json:"masjid"`
	ClassSections []classModel.ClassSectionModel `json:"class_sections"`
}
type MyContextResponse struct {
	User    userModel.UserModel  `json:"user"`
	Masjids []MasjidWithSections `json:"masjids"`
}

/* =============== Controller: GetMyContext (pakai helperAuth) =============== */
func (ac *AuthController) GetMyContext(c *fiber.Ctx) error {
	// 1) Ambil user_id via helperAuth (diisi middleware)
	userUUID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || userUUID == uuid.Nil {
		// Fallback dev: ?user_id=
		if userIDStr := strings.TrimSpace(c.Query("user_id")); userIDStr != "" {
			if parsed, e := uuid.Parse(userIDStr); e == nil {
				userUUID = parsed
			}
		}
		if userUUID == uuid.Nil {
			return fiber.NewError(fiber.StatusUnauthorized, "user_id tidak tersedia pada context")
		}
	}

	// 2) Ambil user (PK default "id")
	var me userModel.UserModel
	if err := ac.DB.WithContext(c.Context()).
		Where("id = ?", userUUID).
		First(&me).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "User tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil user: "+err.Error())
	}

	// 3) Kumpulkan masjid_id (teacher + student) — dedup pakai set
	masjidIDSet := map[uuid.UUID]struct{}{}

	// === 3a) Jalur TEACHER: user_teachers → masjid_teachers
	{
		var myTeacherIDs []uuid.UUID
		// Ambil semua user_teacher_id milik user
		if err := ac.DB.WithContext(c.Context()).
			Model(&UserTeacher{}).
			Where("user_teacher_user_id = ?", userUUID).
			Pluck("user_teacher_id", &myTeacherIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil user_teachers: "+err.Error())
		}
		if len(myTeacherIDs) > 0 {
			// Ambil masjid dari masjid_teachers berdasarkan user_teacher_id
			var mtMasjidIDs []uuid.UUID
			q := ac.DB.WithContext(c.Context()).
				Model(&MasjidTeacher{}).
				Where("masjid_teacher_user_teacher_id IN ?", myTeacherIDs)
			// Soft delete kolom ini ADA di model masjid_teachers
			q = q.Where("masjid_teacher_deleted_at IS NULL")
			if err := q.Pluck("masjid_teacher_masjid_id", &mtMasjidIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil masjid_teachers: "+err.Error())
			}
			for _, id := range mtMasjidIDs {
				masjidIDSet[id] = struct{}{}
			}
		}
	}

	// === 3b) Jalur STUDENT: user_profiles → masjid_students
	{
		var myProfileIDs []uuid.UUID
		// Ambil semua user_profile_id milik user
		if err := ac.DB.WithContext(c.Context()).
			Model(&UserProfile{}).
			Where("user_profile_user_id = ?", userUUID).
			Where("user_profile_deleted_at IS NULL").
			Pluck("user_profile_id", &myProfileIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil user_profiles: "+err.Error())
		}
		if len(myProfileIDs) > 0 {
			// Ambil masjid dari masjid_students berdasarkan user_profile_id
			var msMasjidIDs []uuid.UUID
			q := ac.DB.WithContext(c.Context()).
				Model(&MasjidStudent{}).
				Where("masjid_student_user_profile_id IN ?", myProfileIDs).
				Where("masjid_student_deleted_at IS NULL")
			if err := q.Pluck("masjid_student_masjid_id", &msMasjidIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil masjid_students: "+err.Error())
			}
			for _, id := range msMasjidIDs {
				masjidIDSet[id] = struct{}{}
			}
		}
	}

	// 4) Ambil data Masjid & Class Sections
	masjidIDs := make([]uuid.UUID, 0, len(masjidIDSet))
	for id := range masjidIDSet {
		masjidIDs = append(masjidIDs, id)
	}

	resp := MyContextResponse{User: me, Masjids: []MasjidWithSections{}}
	if len(masjidIDs) == 0 {
		// Tidak tergabung ke masjid mana pun — return user saja
		return c.Status(fiber.StatusOK).JSON(resp)
	}

	// 4a) Masjid (filter aktif + soft delete sesuai model)
	var masjids []masjidModel.MasjidModel
	if err := ac.DB.WithContext(c.Context()).
		Where("masjid_id IN ?", masjidIDs).
		Where("masjid_deleted_at IS NULL").
		Where("masjid_is_active = ?", true).
		Find(&masjids).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil masjid: "+err.Error())
	}

	// 4b) Class sections (filter aktif + soft delete sesuai model)
	var sections []classModel.ClassSectionModel
	if err := ac.DB.WithContext(c.Context()).
		Where("class_section_masjid_id IN ?", masjidIDs).
		Where("class_section_deleted_at IS NULL").
		Where("class_section_is_active = ?", true).
		Find(&sections).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil class_sections: "+err.Error())
	}

	byMasjid := make(map[uuid.UUID][]classModel.ClassSectionModel, len(masjidIDs))
	for _, cs := range sections {
		byMasjid[cs.ClassSectionMasjidID] = append(byMasjid[cs.ClassSectionMasjidID], cs)
	}

	for _, m := range masjids {
		resp.Masjids = append(resp.Masjids, MasjidWithSections{
			Masjid:        m,
			ClassSections: byMasjid[m.MasjidID],
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
