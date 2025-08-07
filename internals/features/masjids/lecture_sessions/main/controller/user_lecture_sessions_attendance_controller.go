package controller

import (
	"errors"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"

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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "user_id tidak ditemukan di token",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id tidak valid",
		})
	}

	// 📥 Ambil dan validasi payload
	var payload dto.UserLectureSessionsAttendanceRequest
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Payload tidak valid",
		})
	}

	sessionID, err := uuid.Parse(payload.UserLectureSessionsAttendanceLectureSessionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID tidak valid",
		})
	}

	// 🔍 Ambil lecture_id berdasarkan session ID
	var lectureIDs []uuid.UUID
	if err := ctrl.DB.
		Table("lecture_sessions").
		Where("lecture_session_id = ?", sessionID).
		Pluck("lecture_session_lecture_id", &lectureIDs).Error; err != nil || len(lectureIDs) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Lecture ID tidak ditemukan dari session",
		})
	}
	lectureID := lectureIDs[0]

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
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal memperbarui kehadiran",
			})
		}

		return c.JSON(fiber.Map{
			"message": "Kehadiran berhasil diperbarui",
			"data":    dto.FromModelUserLectureSessionsAttendance(&existing),
		})
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// ➕ Insert baru
		modelData := dto.ToModelUserLectureSessionsAttendance(&payload, userID)
		modelData.UserLectureSessionsAttendanceLectureID = lectureID

		if err := ctrl.DB.Create(&modelData).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal mencatat kehadiran",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Kehadiran berhasil dicatat",
			"data":    dto.FromModelUserLectureSessionsAttendance(modelData),
		})
	}

	// ❌ Error lain
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "Terjadi kesalahan saat memproses kehadiran",
	})
}


// ✅ Get attendance by session & user
func (ctrl *UserLectureSessionsAttendanceController) GetByLectureSession(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(401).JSON(fiber.Map{"error": "user_id tidak ditemukan di token"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "user_id tidak valid"})
	}

	sessionID := c.Params("lecture_session_id")
	if sessionID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "lecture_session_id tidak boleh kosong"})
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "lecture_session_id tidak valid"})
	}

	var data model.UserLectureSessionsAttendanceModel
	err = ctrl.DB.Where("user_lecture_sessions_attendance_user_id = ? AND user_lecture_sessions_attendance_lecture_session_id = ?", userID, sessionUUID).
		First(&data).Error
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Data kehadiran tidak ditemukan"})
	}

	return c.JSON(dto.FromModelUserLectureSessionsAttendance(&data))
}


// ✅ Get attendance by session slug & user
func (ctrl *UserLectureSessionsAttendanceController) GetByLectureSessionSlug(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(401).JSON(fiber.Map{"error": "user_id tidak ditemukan di token"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "user_id tidak valid"})
	}

	slug := c.Params("lecture_session_slug")
	if slug == "" {
		return c.Status(400).JSON(fiber.Map{"error": "lecture_session_slug tidak boleh kosong"})
	}

	// 🔍 Cari lecture_session berdasarkan slug
	var session model.LectureSessionModel
	err = ctrl.DB.Where("lecture_session_slug = ?", slug).First(&session).Error
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Sesi kajian dengan slug tersebut tidak ditemukan"})
	}

	// 🔍 Cari data kehadiran user terhadap sesi kajian
	var data model.UserLectureSessionsAttendanceModel
	err = ctrl.DB.Where(
		"user_lecture_sessions_attendance_user_id = ? AND user_lecture_sessions_attendance_lecture_session_id = ?",
		userID, session.LectureSessionID,
	).First(&data).Error
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Data kehadiran tidak ditemukan"})
	}

	return c.JSON(dto.FromModelUserLectureSessionsAttendance(&data))
}


// ✅ Delete attendance (optional)
func (ctrl *UserLectureSessionsAttendanceController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "ID tidak valid"})
	}

	if err := ctrl.DB.Delete(&model.UserLectureSessionsAttendanceModel{}, "user_lecture_sessions_attendance_id = ?", parsedID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menghapus data"})
	}

	return c.JSON(fiber.Map{"message": "Data berhasil dihapus"})
}
