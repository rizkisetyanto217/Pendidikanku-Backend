package controller

import (
	"errors"

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

	// Validasi minimal
	if req.UserLectureSessionUserID == uuid.Nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "User ID tidak valid")
	}
	if req.UserLectureSessionLectureSessionID == uuid.Nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Lecture Session ID tidak valid")
	}
	if req.UserLectureSessionLectureID == uuid.Nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Lecture ID tidak valid")
	}
	if req.UserLectureSessionMasjidID == uuid.Nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid")
	}
	if req.UserLectureSessionGradeResult != nil {
		g := *req.UserLectureSessionGradeResult
		if g < 0 || g > 100 {
			return resp.JsonError(c, fiber.StatusBadRequest, "Grade harus di antara 0 sampai 100")
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
		if req.UserLectureSessionMasjidID != uuid.Nil {
			existing.UserLectureSessionMasjidID = req.UserLectureSessionMasjidID
		}
		if req.UserLectureSessionLectureID != uuid.Nil {
			existing.UserLectureSessionLectureID = req.UserLectureSessionLectureID
		}
		if req.UserLectureSessionGradeResult != nil {
			existing.UserLectureSessionGradeResult = req.UserLectureSessionGradeResult
		}

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
	result := dto.ToUserLectureSessionDTOList(records)
	return resp.JsonOK(c, "OK", result)
}

// ===================== GET BY ID =====================
func (ctrl *UserLectureSessionController) GetUserLectureSessionByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var record model.UserLectureSessionModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&record, "user_lecture_session_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Record tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil record")
	}
	return resp.JsonOK(c, "OK", dto.ToUserLectureSessionDTO(record))
}

// ===================== UPDATE (partial) =====================
// (masih memakai CreateUserLectureSessionRequest; kalau mau lebih presisi, buat UpdateRequest dengan pointer)
func (ctrl *UserLectureSessionController) UpdateUserLectureSession(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var record model.UserLectureSessionModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&record, "user_lecture_session_id = ?", id).Error; err != nil {
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
		g := *req.UserLectureSessionGradeResult
		if g < 0 || g > 100 {
			return resp.JsonError(c, fiber.StatusBadRequest, "Grade harus di antara 0 sampai 100")
		}
		record.UserLectureSessionGradeResult = req.UserLectureSessionGradeResult
	}
	if req.UserLectureSessionLectureSessionID != uuid.Nil {
		record.UserLectureSessionLectureSessionID = req.UserLectureSessionLectureSessionID
	}
	if req.UserLectureSessionLectureID != uuid.Nil {
		record.UserLectureSessionLectureID = req.UserLectureSessionLectureID
	}
	if req.UserLectureSessionUserID != uuid.Nil {
		record.UserLectureSessionUserID = req.UserLectureSessionUserID
	}
	if req.UserLectureSessionMasjidID != uuid.Nil {
		record.UserLectureSessionMasjidID = req.UserLectureSessionMasjidID
	}

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

	// Soft delete (default; dengan gorm.DeletedAt akan mengisi deleted_at)
	if err := ctrl.DB.WithContext(c.Context()).
		Delete(&model.UserLectureSessionModel{}, "user_lecture_session_id = ?", id).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return resp.JsonDeleted(c, "Data berhasil dihapus", fiber.Map{"id": id})
}
