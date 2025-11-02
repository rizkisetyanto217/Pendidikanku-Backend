// file: internals/features/users/auth/controller/me_context_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	// ✅ path school sesuai struktur terbaru kamu
	schoolModel "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"
	classModel "schoolku_backend/internals/features/school/classes/class_sections/model"
	userModel "schoolku_backend/internals/features/users/users/model"

	// ✅ helpers
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* =============== Lightweight link models (disesuaikan kolom) =============== */

// user_teachers: user ↔ user_teacher (ID user_teacher dipakai oleh school_teachers)
type UserTeacher struct {
	UserTeacherID     uuid.UUID `gorm:"column:user_teacher_id"`
	UserTeacherUserID uuid.UUID `gorm:"column:user_teacher_user_id"`
}

func (UserTeacher) TableName() string { return "user_teachers" }

// school_teachers: school ↔ user_teacher (kolom benar: school_teacher_user_teacher_id)
type SchoolTeacher struct {
	SchoolTeacherSchoolID      uuid.UUID `gorm:"column:school_teacher_school_id"`
	SchoolTeacherUserTeacherID uuid.UUID `gorm:"column:school_teacher_user_teacher_id"`
}

func (SchoolTeacher) TableName() string { return "school_teachers" }

// user_profiles: user ↔ user_profile (ID user_profile dipakai oleh school_students)
type UserProfile struct {
	UserProfileID     uuid.UUID `gorm:"column:user_profile_id"`
	UserProfileUserID uuid.UUID `gorm:"column:user_profile_user_id"`
}

func (UserProfile) TableName() string { return "user_profiles" }

// school_students: school ↔ user_profile (kolom benar: school_student_user_profile_id)
type SchoolStudent struct {
	SchoolStudentSchoolID      uuid.UUID `gorm:"column:school_student_school_id"`
	SchoolStudentUserProfileID uuid.UUID `gorm:"column:school_student_user_profile_id"`
}

func (SchoolStudent) TableName() string { return "school_students" }

/* =============== DTO Response =============== */
type SchoolWithSections struct {
	School        schoolModel.SchoolModel        `json:"school"`
	ClassSections []classModel.ClassSectionModel `json:"class_sections"`
}
type MyContextResponse struct {
	User    userModel.UserModel  `json:"user"`
	Schools []SchoolWithSections `json:"schools"`
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

	// 3) Kumpulkan school_id (teacher + student) — dedup pakai set
	schoolIDSet := map[uuid.UUID]struct{}{}

	// === 3a) Jalur TEACHER: user_teachers → school_teachers
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
			// Ambil school dari school_teachers berdasarkan user_teacher_id
			var mtSchoolIDs []uuid.UUID
			q := ac.DB.WithContext(c.Context()).
				Model(&SchoolTeacher{}).
				Where("school_teacher_user_teacher_id IN ?", myTeacherIDs)
			// Soft delete kolom ini ADA di model school_teachers
			q = q.Where("school_teacher_deleted_at IS NULL")
			if err := q.Pluck("school_teacher_school_id", &mtSchoolIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil school_teachers: "+err.Error())
			}
			for _, id := range mtSchoolIDs {
				schoolIDSet[id] = struct{}{}
			}
		}
	}

	// === 3b) Jalur STUDENT: user_profiles → school_students
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
			// Ambil school dari school_students berdasarkan user_profile_id
			var msSchoolIDs []uuid.UUID
			q := ac.DB.WithContext(c.Context()).
				Model(&SchoolStudent{}).
				Where("school_student_user_profile_id IN ?", myProfileIDs).
				Where("school_student_deleted_at IS NULL")
			if err := q.Pluck("school_student_school_id", &msSchoolIDs).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil school_students: "+err.Error())
			}
			for _, id := range msSchoolIDs {
				schoolIDSet[id] = struct{}{}
			}
		}
	}

	// 4) Ambil data School & Class Sections
	schoolIDs := make([]uuid.UUID, 0, len(schoolIDSet))
	for id := range schoolIDSet {
		schoolIDs = append(schoolIDs, id)
	}

	resp := MyContextResponse{User: me, Schools: []SchoolWithSections{}}
	if len(schoolIDs) == 0 {
		// Tidak tergabung ke school mana pun — return user saja
		return c.Status(fiber.StatusOK).JSON(resp)
	}

	// 4a) School (filter aktif + soft delete sesuai model)
	var schools []schoolModel.SchoolModel
	if err := ac.DB.WithContext(c.Context()).
		Where("school_id IN ?", schoolIDs).
		Where("school_deleted_at IS NULL").
		Where("school_is_active = ?", true).
		Find(&schools).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil school: "+err.Error())
	}

	// 4b) Class sections (filter aktif + soft delete sesuai model)
	var sections []classModel.ClassSectionModel
	if err := ac.DB.WithContext(c.Context()).
		Where("class_section_school_id IN ?", schoolIDs).
		Where("class_section_deleted_at IS NULL").
		Where("class_section_is_active = ?", true).
		Find(&sections).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil class_sections: "+err.Error())
	}

	bySchool := make(map[uuid.UUID][]classModel.ClassSectionModel, len(schoolIDs))
	for _, cs := range sections {
		bySchool[cs.ClassSectionSchoolID] = append(bySchool[cs.ClassSectionSchoolID], cs)
	}

	for _, m := range schools {
		resp.Schools = append(resp.Schools, SchoolWithSections{
			School:        m,
			ClassSections: bySchool[m.SchoolID],
		})
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
