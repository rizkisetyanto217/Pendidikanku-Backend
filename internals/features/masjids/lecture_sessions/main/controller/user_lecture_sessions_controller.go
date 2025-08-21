package controller

import (
	"errors"
	"time"

	"masjidku_backend/internals/features/masjids/lecture_sessions/main/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	resp "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserLectureSessionController struct {
	DB *gorm.DB
}

func NewUserLectureSessionController(db *gorm.DB) *UserLectureSessionController {
	return &UserLectureSessionController{DB: db}
}

// ===================== CREATE or UPDATE (idempotent by (user_id, session_id)) =====================
func (ctrl *UserLectureSessionController) CreateUserLectureSession(c *fiber.Ctx) error {
	var req dto.CreateUserLectureSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Validasi minimal (UUID format)
	if _, err := uuid.Parse(req.UserLectureSessionUserID); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "User ID tidak valid")
	}
	if _, err := uuid.Parse(req.UserLectureSessionLectureSessionID); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Lecture Session ID tidak valid")
	}
	if req.UserLectureSessionMasjidID != "" {
		if _, err := uuid.Parse(req.UserLectureSessionMasjidID); err != nil {
			return resp.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid")
		}
	}

	// Cek existing berdasarkan (user_id, lecture_session_id)
	var existing model.UserLectureSessionModel
	err := ctrl.DB.WithContext(c.Context()).
		Where("user_lecture_session_user_id = ? AND user_lecture_session_lecture_session_id = ?",
			req.UserLectureSessionUserID, req.UserLectureSessionLectureSessionID).
		First(&existing).Error

	if err == nil {
		// Update sebagian field bila dikirim
		if req.UserLectureSessionMasjidID != "" {
			existing.UserLectureSessionMasjidID = req.UserLectureSessionMasjidID
		}
		if req.UserLectureSessionGradeResult != nil {
			existing.UserLectureSessionGradeResult = req.UserLectureSessionGradeResult
		}
		now := time.Now()
		existing.UserLectureSessionUpdatedAt = &now

		if err := ctrl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
			return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui user lecture session")
		}
		return resp.JsonUpdated(c, "User lecture session diperbarui", dto.ToUserLectureSessionDTO(existing))
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Insert baru
		newRecord := req.ToModel()
		if err := ctrl.DB.WithContext(c.Context()).Create(&newRecord).Error; err != nil {
			return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat user lecture session")
		}
		return resp.JsonCreated(c, "User lecture session dibuat", dto.ToUserLectureSessionDTO(newRecord))
	}

	// Error lain
	return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mencari data user lecture session")
}

// ===================== GET ALL =====================
func (ctrl *UserLectureSessionController) GetAllUserLectureSessions(c *fiber.Ctx) error {
	var records []model.UserLectureSessionModel
	if err := ctrl.DB.WithContext(c.Context()).Find(&records).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	result := make([]dto.UserLectureSessionDTO, 0, len(records))
	for _, r := range records {
		result = append(result, dto.ToUserLectureSessionDTO(r))
	}
	return resp.JsonOK(c, "OK", result)
}

// ===================== GET BY ID =====================
func (ctrl *UserLectureSessionController) GetUserLectureSessionByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var record model.UserLectureSessionModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&record, "user_lecture_session_id = ?", idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Record tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil record")
	}
	return resp.JsonOK(c, "OK", dto.ToUserLectureSessionDTO(record))
}

// ===================== UPDATE (partial) =====================
func (ctrl *UserLectureSessionController) UpdateUserLectureSession(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var record model.UserLectureSessionModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&record, "user_lecture_session_id = ?", idStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	var req dto.CreateUserLectureSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Partial update
	if req.UserLectureSessionGradeResult != nil {
		record.UserLectureSessionGradeResult = req.UserLectureSessionGradeResult
	}
	if req.UserLectureSessionLectureSessionID != "" {
		if _, err := uuid.Parse(req.UserLectureSessionLectureSessionID); err != nil {
			return resp.JsonError(c, fiber.StatusBadRequest, "Lecture Session ID tidak valid")
		}
		record.UserLectureSessionLectureSessionID = req.UserLectureSessionLectureSessionID
	}
	if req.UserLectureSessionUserID != "" {
		if _, err := uuid.Parse(req.UserLectureSessionUserID); err != nil {
			return resp.JsonError(c, fiber.StatusBadRequest, "User ID tidak valid")
		}
		record.UserLectureSessionUserID = req.UserLectureSessionUserID
	}
	if req.UserLectureSessionMasjidID != "" {
		if _, err := uuid.Parse(req.UserLectureSessionMasjidID); err != nil {
			return resp.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid")
		}
		record.UserLectureSessionMasjidID = req.UserLectureSessionMasjidID
	}

	now := time.Now()
	record.UserLectureSessionUpdatedAt = &now

	if err := ctrl.DB.WithContext(c.Context()).Save(&record).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data user lecture session")
	}

	return resp.JsonUpdated(c, "User lecture session diperbarui", dto.ToUserLectureSessionDTO(record))
}

// ===================== DELETE =====================
func (ctrl *UserLectureSessionController) DeleteUserLectureSession(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Delete(&model.UserLectureSessionModel{}, "user_lecture_session_id = ?", id).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return resp.JsonDeleted(c, "Data berhasil dihapus", fiber.Map{"id": id})
}
