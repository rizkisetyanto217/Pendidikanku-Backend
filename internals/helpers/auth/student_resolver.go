package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetSchoolStudentIDFromDB(c *fiber.Ctx, db *gorm.DB, schoolID uuid.UUID) (uuid.UUID, error) {
	if db == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "DB context tidak tersedia")
	}
	if schoolID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id wajib")
	}

	// 1) Ambil user_id dari token
	userID, err := GetUserIDFromToken(c)
	if err != nil || userID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "user_id tidak ditemukan pada token")
	}

	// 2) Ambil semua profile aktif user ini
	var profileIDs []uuid.UUID
	if err := db.WithContext(c.Context()).
		Table("user_profiles").
		Where("user_profile_user_id = ? AND user_profile_deleted_at IS NULL", userID).
		Pluck("user_profile_id", &profileIDs).Error; err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil user_profiles: "+err.Error())
	}
	if len(profileIDs) == 0 {
		return uuid.Nil, nil // belum punya profile → belum jadi student di mana pun
	}

	// 3) Ambil school_student_id sebagai STRING lalu parse
	var sidStr string
	if err := db.WithContext(c.Context()).
		Table("school_students").
		Where("school_student_school_id = ?", schoolID).
		Where("school_student_user_profile_id IN ?", profileIDs).
		Where("school_student_deleted_at IS NULL").
		Limit(1).
		Pluck("school_student_id", &sidStr).Error; err != nil {

		low := strings.ToLower(err.Error())
		if strings.Contains(low, "does not exist") ||
			strings.Contains(low, "no such table") ||
			strings.Contains(low, "undefined") {
			// kalau table belum ada pas awal dev, jangan bikin 500
			return uuid.Nil, nil
		}

		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal ambil school_students: "+err.Error())
	}

	sidStr = strings.TrimSpace(sidStr)
	if sidStr == "" {
		// tidak ada baris yg match
		return uuid.Nil, nil
	}

	sid, perr := uuid.Parse(sidStr)
	if perr != nil {
		// data di DB korup → mending 500 biar ketahuan
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "school_student_id invalid di database")
	}

	return sid, nil
}

// Coba dari token dulu (student_records), kalau nggak ada → fallback ke DB
func GetSchoolStudentIDSmart(c *fiber.Ctx, db *gorm.DB, schoolID uuid.UUID) (uuid.UUID, error) {
	if schoolID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id wajib")
	}

	// 1) coba dari token dulu
	if sid, err := GetSchoolStudentIDForSchool(c, schoolID); err == nil && sid != uuid.Nil {
		return sid, nil
	}

	// 2) fallback ke DB
	return GetSchoolStudentIDFromDB(c, db, schoolID)
}
