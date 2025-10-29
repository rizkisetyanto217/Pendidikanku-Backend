// file: internals/features/school/assessments/controller/assessment_type_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	assessSvc "masjidku_backend/internals/features/school/submissions_assesments/assesments/service"
)

type AssessmentTypeController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAssessmentTypeController(db *gorm.DB) *AssessmentTypeController {
	return &AssessmentTypeController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ========================= Helpers ========================= */

func mapToResponse(m *model.AssessmentTypeModel) dto.AssessmentTypeResponse {
	return dto.AssessmentTypeResponse{
		AssessmentTypeID:            m.AssessmentTypeID,
		AssessmentTypeMasjidID:      m.AssessmentTypeMasjidID,
		AssessmentTypeKey:           m.AssessmentTypeKey,
		AssessmentTypeName:          m.AssessmentTypeName,
		AssessmentTypeWeightPercent: m.AssessmentTypeWeightPercent,
		AssessmentTypeIsActive:      m.AssessmentTypeIsActive,
		AssessmentTypeCreatedAt:     m.AssessmentTypeCreatedAt,
		AssessmentTypeUpdatedAt:     m.AssessmentTypeUpdatedAt,
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "sqlstate 23505") ||
		strings.Contains(s, "duplicate key value") ||
		strings.Contains(s, "unique constraint")
}

/* ========================= Handlers ========================= */

/* ========================= Handlers ========================= */

// POST /assessment-types — staff (DKM/Admin/Owner/Superadmin)
func (ctl *AssessmentTypeController) Create(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	var req dto.CreateAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req = req.Normalize()

	// 🔒 Masjid context + ensure DKM/Admin untuk masjid tsb
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Validasi bobot 0..100
	if req.AssessmentTypeWeightPercent < 0 || req.AssessmentTypeWeightPercent > 100 {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity,
			"assessment_type_weight_percent harus di antara 0 hingga 100")
	}

	now := time.Now()
	row := model.AssessmentTypeModel{
		AssessmentTypeID:            uuid.New(),
		AssessmentTypeMasjidID:      masjidID, // ⛔ enforce dari context (anti cross-tenant)
		AssessmentTypeKey:           strings.TrimSpace(req.AssessmentTypeKey),
		AssessmentTypeName:          strings.TrimSpace(req.AssessmentTypeName),
		AssessmentTypeWeightPercent: req.AssessmentTypeWeightPercent,
		AssessmentTypeIsActive:      true,
		AssessmentTypeCreatedAt:     now,
		AssessmentTypeUpdatedAt:     now,
	}
	if req.AssessmentTypeIsActive != nil {
		row.AssessmentTypeIsActive = *req.AssessmentTypeIsActive
	}

	// Validasi agregat aktif ≤ 100
	if row.AssessmentTypeIsActive {
		sum, err := assessSvc.New().SumActiveWeights(ctl.DB.WithContext(c.Context()), masjidID, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total bobot")
		}
		if sum+row.AssessmentTypeWeightPercent > 100.0 {
			remaining := 100.0 - sum
			if remaining < 0 {
				remaining = 0
			}
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				fmt.Sprintf("Total bobot melebihi 100. Sisa yang tersedia: %.2f%%", remaining))
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Key sudah dipakai untuk masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Assessment type dibuat", mapToResponse(&row))
}

// PATCH /assessment-types/:id — staff
func (ctl *AssessmentTypeController) Patch(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_type_id tidak valid")
	}

	var req dto.PatchAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 Masjid context + ensure DKM/Admin
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var existing model.AssessmentTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessment_type_id = ? AND assessment_type_masjid_id = ? AND assessment_type_deleted_at IS NULL", id, masjidID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Hitung nilai akhir utk validasi agregat
	finalActive := existing.AssessmentTypeIsActive
	if req.AssessmentTypeIsActive != nil {
		finalActive = *req.AssessmentTypeIsActive
	}
	finalWeight := existing.AssessmentTypeWeightPercent
	if req.AssessmentTypeWeightPercent != nil {
		if *req.AssessmentTypeWeightPercent < 0 || *req.AssessmentTypeWeightPercent > 100 {
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				"assessment_type_weight_percent harus di antara 0 hingga 100")
		}
		finalWeight = *req.AssessmentTypeWeightPercent
	}

	// Validasi agregat (aktif) ≤ 100 — exclude row ini
	if finalActive {
		sum, err := assessSvc.New().SumActiveWeights(ctl.DB.WithContext(c.Context()), masjidID, &existing.AssessmentTypeID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total bobot")
		}
		if sum+finalWeight > 100.0 {
			remaining := 100.0 - sum
			if remaining < 0 {
				remaining = 0
			}
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				fmt.Sprintf("Total bobot melebihi 100. Sisa yang tersedia: %.2f%%", remaining))
		}
	}

	updates := map[string]any{}
	if req.AssessmentTypeName != nil {
		updates["assessment_type_name"] = strings.TrimSpace(*req.AssessmentTypeName)
	}
	if req.AssessmentTypeWeightPercent != nil {
		updates["assessment_type_weight_percent"] = *req.AssessmentTypeWeightPercent
	}
	if req.AssessmentTypeIsActive != nil {
		updates["assessment_type_is_active"] = *req.AssessmentTypeIsActive
	}
	if len(updates) == 0 {
		return helper.JsonOK(c, "OK", mapToResponse(&existing))
	}
	updates["assessment_type_updated_at"] = time.Now()

	if err := ctl.DB.WithContext(c.Context()).
		Model(&model.AssessmentTypeModel{}).
		Where("assessment_type_id = ? AND assessment_type_masjid_id = ?", id, masjidID).
		Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Key sudah dipakai untuk masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var after model.AssessmentTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessment_type_id = ? AND assessment_type_masjid_id = ?", id, masjidID).
		First(&after).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Assessment type diperbarui", mapToResponse(&after))
}

// DELETE /assessment-types/:id — staff
func (ctl *AssessmentTypeController) Delete(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_type_id tidak valid")
	}

	// 🔒 Masjid context + ensure DKM/Admin
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var row model.AssessmentTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessment_type_id = ? AND assessment_type_masjid_id = ?", id, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Assessment type dihapus", fiber.Map{
		"assessment_type_id": id,
	})
}
