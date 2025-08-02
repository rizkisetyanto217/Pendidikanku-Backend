package controller

import (
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

// ✅ Create or Upsert attendance
func (ctrl *UserLectureSessionsAttendanceController) CreateOrUpdate(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(401).JSON(fiber.Map{"error": "user_id tidak ditemukan di token"})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "user_id tidak valid"})
	}

	var payload dto.UserLectureSessionsAttendanceRequest
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Payload tidak valid"})
	}

	sessionID, err := uuid.Parse(payload.UserLectureSessionsAttendanceLectureSessionID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Session ID tidak valid"})
	}

	var existing model.UserLectureSessionsAttendanceModel
	err = ctrl.DB.Where("user_lecture_sessions_attendance_user_id = ? AND user_lecture_sessions_attendance_lecture_session_id = ?", userID, sessionID).
		First(&existing).Error

	if err == nil {
		// Update existing
		existing.UserLectureSessionsAttendanceStatus = payload.UserLectureSessionsAttendanceStatus
		existing.UserLectureSessionsAttendanceNotes = payload.UserLectureSessionsAttendanceNotes
		existing.UserLectureSessionsAttendancePersonalNotes = payload.UserLectureSessionsAttendancePersonalNotes
		ctrl.DB.Save(&existing)
		return c.JSON(fiber.Map{
			"message": "Kehadiran berhasil diperbarui",
			"data":    dto.FromModelUserLectureSessionsAttendance(&existing),
		})
	}

	if err == gorm.ErrRecordNotFound {
		// Create baru
		modelData := dto.ToModelUserLectureSessionsAttendance(&payload, userID)
		if err := ctrl.DB.Create(&modelData).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan kehadiran"})
		}
		return c.Status(201).JSON(fiber.Map{
			"message": "Kehadiran berhasil dicatat",
			"data":    dto.FromModelUserLectureSessionsAttendance(modelData),
		})
	}

	return c.Status(500).JSON(fiber.Map{"error": "Terjadi kesalahan saat menyimpan data"})
}

// ✅ Get attendance by session & user
func (ctrl *UserLectureSessionsAttendanceController) GetBySession(c *fiber.Ctx) error {
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
