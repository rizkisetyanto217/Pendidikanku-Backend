package controller

import (
	"errors"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserLectureSessionController struct {
	DB *gorm.DB
}

func NewUserLectureSessionController(db *gorm.DB) *UserLectureSessionController {
	return &UserLectureSessionController{DB: db}
}


func (ctrl *UserLectureSessionController) CreateUserLectureSession(c *fiber.Ctx) error {
	var req dto.CreateUserLectureSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Cari apakah sudah ada data sebelumnya (berdasarkan user_id dan lecture_session_id)
	var existing model.UserLectureSessionModel
	err := ctrl.DB.
		Where("user_lecture_session_user_id = ? AND user_lecture_session_lecture_session_id = ?",
			req.UserLectureSessionUserID, req.UserLectureSessionLectureSessionID).
		First(&existing).Error

	if err == nil {
		existing.UserLectureSessionMasjidID = req.UserLectureSessionMasjidID

		if err := ctrl.DB.Save(&existing).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui user lecture session")
		}

		return c.Status(fiber.StatusOK).JSON(dto.ToUserLectureSessionDTO(existing))
	}

	// Jika error karena tidak ditemukan → insert baru
	if errors.Is(err, gorm.ErrRecordNotFound) {
		newRecord := req.ToModel()

		if err := ctrl.DB.Create(&newRecord).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat user lecture session")
		}

		return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureSessionDTO(newRecord))
	}

	// Error lain (misalnya koneksi database)
	return fiber.NewError(fiber.StatusInternalServerError, "Gagal mencari data user lecture session")
}

// GET ALL
func (ctrl *UserLectureSessionController) GetAllUserLectureSessions(c *fiber.Ctx) error {
	var records []model.UserLectureSessionModel
	if err := ctrl.DB.Find(&records).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve records")
	}

	var result []dto.UserLectureSessionDTO
	for _, record := range records {
		result = append(result, dto.ToUserLectureSessionDTO(record))
	}

	return c.JSON(result)
}

// GET BY ID
func (ctrl *UserLectureSessionController) GetUserLectureSessionByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var record model.UserLectureSessionModel
	if err := ctrl.DB.First(&record, "user_lecture_session_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Record not found")
	}

	return c.JSON(dto.ToUserLectureSessionDTO(record))
}


func (ctrl *UserLectureSessionController) UpdateUserLectureSession(c *fiber.Ctx) error {
	id := c.Params("id")

	var record model.UserLectureSessionModel
	if err := ctrl.DB.First(&record, "user_lecture_session_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}

	var req dto.CreateUserLectureSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	if req.UserLectureSessionGradeResult != nil {
		record.UserLectureSessionGradeResult = req.UserLectureSessionGradeResult
	}
	if req.UserLectureSessionLectureSessionID != "" {
		record.UserLectureSessionLectureSessionID = req.UserLectureSessionLectureSessionID
	}
	if req.UserLectureSessionUserID != "" {
		record.UserLectureSessionUserID = req.UserLectureSessionUserID
	}
	if req.UserLectureSessionMasjidID != "" {
		record.UserLectureSessionMasjidID = req.UserLectureSessionMasjidID
	}

	now := time.Now()
	record.UserLectureSessionUpdatedAt = &now

	if err := ctrl.DB.Save(&record).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data user lecture session")
	}

	return c.JSON(dto.ToUserLectureSessionDTO(record))
}


// DELETE
func (ctrl *UserLectureSessionController) DeleteUserLectureSession(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.UserLectureSessionModel{}, "user_lecture_session_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete")
	}

	return c.SendStatus(fiber.StatusNoContent)
}