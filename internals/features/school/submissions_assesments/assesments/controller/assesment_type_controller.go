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

func mustMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return uuid.Nil, helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	return mid, nil
}

func mapToResponse(m *model.AssessmentTypeModel) dto.AssessmentTypeResponse {
	return dto.AssessmentTypeResponse{
		AssessmentTypesID:            m.ID,
		AssessmentTypesMasjidID:      m.MasjidID,
		AssessmentTypesKey:           m.Key,
		AssessmentTypesName:          m.Name,
		AssessmentTypesWeightPercent: m.WeightPercent,
		AssessmentTypesIsActive:      m.IsActive,
		AssessmentTypesCreatedAt:     m.CreatedAt,
		AssessmentTypesUpdatedAt:     m.UpdatedAt,
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

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(c.Params(name)))
}

// di assesment_controller.go (controller assessments), bukan assessment type
func getSortClause(sortBy, sortDir *string) string {
    col := "assessments_created_at" // default
    if sortBy != nil {
        switch strings.ToLower(strings.TrimSpace(*sortBy)) {
        case "title":
            col = "assessments_title"
        case "start_at":
            col = "assessments_start_at"
        case "due_at":
            col = "assessments_due_at"
        case "created_at":
            col = "assessments_created_at"
        }
    }
    dir := "DESC"
    if sortDir != nil && strings.EqualFold(strings.TrimSpace(*sortDir), "asc") {
        dir = "ASC"
    }
    return col + " " + dir
}



/* ========================= Handlers ========================= */

// GET /assessment-types?active=&q=&limit=&offset=&sort_by=&sort_dir=
func (ctl *AssessmentTypeController) List(c *fiber.Ctx) error {
	mid, err := mustMasjidID(c)
	if err != nil {
		return err
	}

	var filt dto.ListAssessmentTypeFilter
	filt.AssessmentTypesMasjidID = mid

	// Filters opsional
	if v := strings.TrimSpace(c.Query("active")); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		filt.Active = &b
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		filt.Q = &q
	}

	// Paging & sorting
	filt.Limit = atoiOr(20, c.Query("limit"))
	filt.Offset = atoiOr(0, c.Query("offset"))
	if sb := strings.TrimSpace(c.Query("sort_by")); sb != "" { // name|created_at
		filt.SortBy = &sb
	}
	if sd := strings.TrimSpace(c.Query("sort_dir")); sd != "" { // asc|desc
		filt.SortDir = &sd
	}

	if err := ctl.Validator.Struct(&filt); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Query tenant-scoped
	qry := ctl.DB.Model(&model.AssessmentTypeModel{}).
		Where("assessment_types_masjid_id = ?", filt.AssessmentTypesMasjidID)

	if filt.Active != nil {
		qry = qry.Where("assessment_types_is_active = ?", *filt.Active)
	}
	if filt.Q != nil {
		like := "%" + strings.ToLower(strings.TrimSpace(*filt.Q)) + "%"
		qry = qry.Where(
			"(LOWER(assessment_types_name) LIKE ? OR LOWER(assessment_types_key) LIKE ?)",
			like, like,
		)
	}

	var total int64
	if err := qry.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.AssessmentTypeModel
	if err := qry.
		Order(getSortClause(filt.SortBy, filt.SortDir)).
		Limit(filt.Limit).
		Offset(filt.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.AssessmentTypeResponse, 0, len(rows))
	for i := range rows {
		out = append(out, mapToResponse(&rows[i]))
	}

	// pakai JsonList: data + pagination terpisah
	return helper.JsonList(c, out, fiber.Map{
		"total":  total,
		"limit":  filt.Limit,
		"offset": filt.Offset,
	})
}

// GET /assessment-types/:id
func (ctl *AssessmentTypeController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_types_id tidak valid")
	}

	mid, err := mustMasjidID(c)
	if err != nil {
		return err
	}

	var row model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, mid).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", mapToResponse(&row))
}

// POST /assessment-types
func (ctl *AssessmentTypeController) Create(c *fiber.Ctx) error {
	var req dto.CreateAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	mid, err := mustMasjidID(c)
	if err != nil {
		return err
	}
	req.AssessmentTypesMasjidID = mid

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 VALIDASI: 0..100
	if req.AssessmentTypesWeightPercent < 0 || req.AssessmentTypesWeightPercent > 100 {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity,
			"assessment_types_weight_percent harus di antara 0 hingga 100")
	}

	now := time.Now()
	row := model.AssessmentTypeModel{
		ID:            uuid.New(),
		MasjidID:      req.AssessmentTypesMasjidID,
		Key:           strings.TrimSpace(req.AssessmentTypesKey),
		Name:          strings.TrimSpace(req.AssessmentTypesName),
		WeightPercent: req.AssessmentTypesWeightPercent,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if req.AssessmentTypesIsActive != nil {
		row.IsActive = *req.AssessmentTypesIsActive
	}

	// ✅ VALIDASI AGREGAT: jika AKTIF, total bobot aktif (existing + candidate) ≤ 100
	if row.IsActive {
		currentSum, err := assessSvc.SumActiveWeights(ctl.DB.WithContext(c.Context()), mid, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total bobot")
		}
		if currentSum+float64(row.WeightPercent) > 100.0 {
			remaining := 100.0 - currentSum
			if remaining < 0 {
				remaining = 0
			}
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				fmt.Sprintf("Total bobot melebihi 100. Sisa yang tersedia: %.2f%%", remaining))
		}
	}

	if err := ctl.DB.Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Key sudah dipakai untuk masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Assessment type dibuat", mapToResponse(&row))
}

// PATCH /assessment-types/:id
func (ctl *AssessmentTypeController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_types_id tidak valid")
	}

	var req dto.PatchAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	mid, err := mustMasjidID(c)
	if err != nil {
		return err
	}

	var existing model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, mid).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Tentukan nilai akhir setelah patch (untuk validasi agregat)
	finalActive := existing.IsActive
	if req.AssessmentTypesIsActive != nil {
		finalActive = *req.AssessmentTypesIsActive
	}
	finalWeight := existing.WeightPercent
	if req.AssessmentTypesWeightPercent != nil {
		// 🔒 VALIDASI: 0..100
		if *req.AssessmentTypesWeightPercent < 0 || *req.AssessmentTypesWeightPercent > 100 {
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				"assessment_types_weight_percent harus di antara 0 hingga 100")
		}
		finalWeight = *req.AssessmentTypesWeightPercent
	}

	// ✅ VALIDASI AGREGAT (hanya bila hasil akhirnya AKTIF)
	if finalActive {
		currentSum, err := assessSvc.SumActiveWeights(ctl.DB.WithContext(c.Context()), mid, &existing.ID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total bobot")
		}
		if currentSum+float64(finalWeight) > 100.0 {
			remaining := 100.0 - currentSum
			if remaining < 0 {
				remaining = 0
			}
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				fmt.Sprintf("Total bobot melebihi 100. Sisa yang tersedia: %.2f%%", remaining))
		}
	}

	updates := map[string]interface{}{}
	if req.AssessmentTypesName != nil {
		updates["assessment_types_name"] = strings.TrimSpace(*req.AssessmentTypesName)
	}
	if req.AssessmentTypesWeightPercent != nil {
		updates["assessment_types_weight_percent"] = *req.AssessmentTypesWeightPercent
	}
	if req.AssessmentTypesIsActive != nil {
		updates["assessment_types_is_active"] = *req.AssessmentTypesIsActive
	}
	if len(updates) == 0 {
		return helper.JsonOK(c, "OK", mapToResponse(&existing))
	}
	updates["assessment_types_updated_at"] = time.Now()

	if err := ctl.DB.Model(&model.AssessmentTypeModel{}).
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, mid).
		Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Key sudah dipakai untuk masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var after model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, mid).
		First(&after).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Assessment type diperbarui", mapToResponse(&after))
}



// DELETE /assessment-types/:id (soft delete)
func (ctl *AssessmentTypeController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_types_id tidak valid")
	}

	mid, err := mustMasjidID(c)
	if err != nil {
		return err
	}

	var row model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, mid).
		First(&row).Error; err != nil {
	if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// gunakan 200 OK + body sesuai helper.JsonDeleted
	return helper.JsonDeleted(c, "Assessment type dihapus", fiber.Map{
		"assessment_types_id": id,
	})
}
