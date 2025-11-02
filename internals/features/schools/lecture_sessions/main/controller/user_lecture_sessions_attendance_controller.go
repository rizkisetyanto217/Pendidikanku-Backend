package controller

import (
	"errors"

	"schoolku_backend/internals/features/schools/lecture_sessions/main/dto"
	"schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	resp "schoolku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserLectureSessionsAttendanceController struct {
	DB *gorm.DB
}

func NewUserLectureSessionsAttendanceController(db *gorm.DB) *UserLectureSessionsAttendanceController {
	return &UserLectureSessionsAttendanceController{DB: db}
}

func (ctrl *UserLectureSessionsAttendanceController) CreateOrUpdate(c *fiber.Ctx) error {
	// 🔐 Ambil user ID dari token
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "user_id tidak ditemukan di token")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid")
	}

	// 📥 Ambil dan validasi payload
	var payload dto.UserLectureSessionsAttendanceRequest
	if err := c.BodyParser(&payload); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	sessionID, err := uuid.Parse(payload.UserLectureSessionsAttendanceLectureSessionID)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Session ID tidak valid")
	}

	// 🔍 Ambil lecture_id berdasarkan session ID (lebih efisien tanpa slice)
	var lectureID uuid.UUID
	if err := ctrl.DB.
		Table("lecture_sessions").
		Select("lecture_session_lecture_id").
		Where("lecture_session_id = ?", sessionID).
		Scan(&lectureID).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil Lecture ID")
	}
	if lectureID == uuid.Nil {
		return resp.JsonError(c, fiber.StatusNotFound, "Lecture ID tidak ditemukan dari session")
	}

	// 🔁 Cek apakah sudah pernah mengisi kehadiran
	var existing model.UserLectureSessionsAttendanceModel
	err = ctrl.DB.
		Where("user_lecture_sessions_attendance_user_id = ? AND user_lecture_sessions_attendance_lecture_session_id = ?", userID, sessionID).
		First(&existing).Error

	if err == nil {
		// ✅ Update
		existing.UserLectureSessionsAttendanceStatus = payload.UserLectureSessionsAttendanceStatus
		existing.UserLectureSessionsAttendanceNotes = payload.UserLectureSessionsAttendanceNotes
		existing.UserLectureSessionsAttendancePersonalNotes = payload.UserLectureSessionsAttendancePersonalNotes
		existing.UserLectureSessionsAttendanceLectureID = lectureID

		if err := ctrl.DB.Save(&existing).Error; err != nil {
			return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui kehadiran")
		}
		return resp.JsonUpdated(c, "Kehadiran berhasil diperbarui", dto.FromModelUserLectureSessionsAttendance(&existing))
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// ➕ Insert baru
		modelData := dto.ToModelUserLectureSessionsAttendance(&payload, userID)
		modelData.UserLectureSessionsAttendanceLectureID = lectureID

		if err := ctrl.DB.Create(&modelData).Error; err != nil {
			return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mencatat kehadiran")
		}
		return resp.JsonCreated(c, "Kehadiran berhasil dicatat", dto.FromModelUserLectureSessionsAttendance(modelData))
	}

	// ❌ Error lain
	return resp.JsonError(c, fiber.StatusInternalServerError, "Terjadi kesalahan saat memproses kehadiran")
}

// ✅ Get attendance by session & user
func (ctrl *UserLectureSessionsAttendanceController) GetByLectureSession(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "user_id tidak ditemukan di token")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid")
	}

	sessionID := c.Params("lecture_session_id")
	if sessionID == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "lecture_session_id tidak boleh kosong")
	}
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "lecture_session_id tidak valid")
	}

	var data model.UserLectureSessionsAttendanceModel
	if err := ctrl.DB.Where(
		"user_lecture_sessions_attendance_user_id = ? AND user_lecture_sessions_attendance_lecture_session_id = ?",
		userID, sessionUUID,
	).First(&data).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Data kehadiran tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data kehadiran")
	}

	return resp.JsonOK(c, "OK", dto.FromModelUserLectureSessionsAttendance(&data))
}

// ✅ Get attendance by session slug & user
func (ctrl *UserLectureSessionsAttendanceController) GetByLectureSessionSlug(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "user_id tidak ditemukan di token")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid")
	}

	slug := c.Params("lecture_session_slug")
	if slug == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "lecture_session_slug tidak boleh kosong")
	}

	// 🔍 Cari lecture_session berdasarkan slug
	var session model.LectureSessionModel
	if err := ctrl.DB.Where("lecture_session_slug = ?", slug).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Sesi kajian dengan slug tersebut tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian")
	}

	// 🔍 Cari data kehadiran user terhadap sesi kajian
	var data model.UserLectureSessionsAttendanceModel
	if err := ctrl.DB.Where(
		"user_lecture_sessions_attendance_user_id = ? AND user_lecture_sessions_attendance_lecture_session_id = ?",
		userID, session.LectureSessionID,
	).First(&data).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Data kehadiran tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data kehadiran")
	}

	return resp.JsonOK(c, "OK", dto.FromModelUserLectureSessionsAttendance(&data))
}

// ✅ Delete attendance
func (ctrl *UserLectureSessionsAttendanceController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctrl.DB.Delete(&model.UserLectureSessionsAttendanceModel{}, "user_lecture_sessions_attendance_id = ?", parsedID).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return resp.JsonDeleted(c, "Data berhasil dihapus", fiber.Map{"id": parsedID})
}
