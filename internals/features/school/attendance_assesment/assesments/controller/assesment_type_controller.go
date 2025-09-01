// file: internals/features/school/assessments/controller/assessment_type_controller.go
package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/attendance_assesment/assesments/dto"
	model "masjidku_backend/internals/features/school/attendance_assesment/assesments/model"
	helper "masjidku_backend/internals/helpers"
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

/* =========================
   ROUTES (contoh wiring)
   =========================
r := app.Group("/api/a")
ctrl := controller.NewAssessmentTypeController(db)

r.Get("/assessment-types", ctrl.List)            // ?masjid_id=&active=&q=&limit=&offset=&sort_by=&sort_dir=
r.Get("/assessment-types/:id", ctrl.GetByID)     // path id = assessment_types_id (uuid)
r.Post("/assessment-types", ctrl.Create)
r.Put("/assessment-types/:id", ctrl.Update)      // partial update
r.Delete("/assessment-types/:id", ctrl.Delete)   // soft delete
*/

// ========================= Helpers =========================

func mapToResponse(m *model.AssessmentTypeModel) dto.AssessmentTypeResponse {
	return dto.AssessmentTypeResponse{
		ID:            m.ID,
		MasjidID:      m.MasjidID,
		Key:           m.Key,
		Name:          m.Name,
		WeightPercent: m.WeightPercent,
		IsActive:      m.IsActive,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	slow := strings.ToLower(s)
	return strings.Contains(s, "sqlstate 23505") ||
		strings.Contains(slow, "duplicate key value") ||
		strings.Contains(slow, "unique constraint")
}

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	return uuid.Parse(idStr)
}

func getSortClause(sortBy, sortDir *string) string {
	col := "assessment_types_created_at" // default
	if sortBy != nil {
		switch *sortBy {
		case "name":
			col = "assessment_types_name"
		case "created_at":
			col = "assessment_types_created_at"
		}
	}
	dir := "DESC"
	if sortDir != nil && strings.ToUpper(*sortDir) == "ASC" {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", col, dir)
}

func atoiOr(def int, s string) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// ========================= Handlers =========================

// GET /assessment-types?masjid_id=&active=&q=&limit=&offset=&sort_by=&sort_dir=
func (ctl *AssessmentTypeController) List(c *fiber.Ctx) error {
	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)

	var filt dto.ListAssessmentTypeFilter
	if q := c.Query("masjid_id"); q != "" {
		if v, err := uuid.Parse(q); err == nil {
			filt.MasjidID = v
		}
	}
	if masjidIDFromToken != uuid.Nil {
		filt.MasjidID = masjidIDFromToken
	}
	if filt.MasjidID == uuid.Nil {
		return helper.Error(c, fiber.StatusBadRequest, "masjid_id wajib (token atau query)")
	}

	if v := c.Query("active"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		filt.Active = &b
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		filt.Q = &[]string{q}[0]
	}
	limit := atoiOr(20, c.Query("limit"))
	offset := atoiOr(0, c.Query("offset"))
	filt.Limit, filt.Offset = limit, offset

	if sb := strings.TrimSpace(c.Query("sort_by")); sb != "" {
		filt.SortBy = &[]string{sb}[0] // name|created_at
	}
	if sd := strings.TrimSpace(c.Query("sort_dir")); sd != "" {
		filt.SortDir = &[]string{sd}[0] // asc|desc
	}

	if err := ctl.Validator.Struct(&filt); err != nil {
		return helper.ValidationError(c, err)
	}

	qry := ctl.DB.Model(&model.AssessmentTypeModel{}).
		Where("assessment_types_masjid_id = ?", filt.MasjidID)

	if filt.Active != nil {
		qry = qry.Where("assessment_types_is_active = ?", *filt.Active)
	}
	if filt.Q != nil {
		q := "%" + strings.ToLower(strings.TrimSpace(*filt.Q)) + "%"
		qry = qry.Where(
			"(LOWER(assessment_types_name) LIKE ? OR LOWER(assessment_types_key) LIKE ?)",
			q, q,
		)
	}

	var total int64
	if err := qry.Count(&total).Error; err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.AssessmentTypeModel
	if err := qry.
		Order(getSortClause(filt.SortBy, filt.SortDir)).
		Limit(filt.Limit).Offset(filt.Offset).
		Find(&rows).Error; err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.AssessmentTypeResponse, 0, len(rows))
	for i := range rows {
		out = append(out, mapToResponse(&rows[i]))
	}

	resp := dto.ListAssessmentTypeResponse{
		Data:   out,
		Total:  total,
		Limit:  filt.Limit,
		Offset: filt.Offset,
	}
	return helper.Success(c, "OK", resp)
}

// GET /assessment-types/:id
func (ctl *AssessmentTypeController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var row model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, masjidIDFromToken).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.Success(c, "OK", mapToResponse(&row))
}

// POST /assessment-types
func (ctl *AssessmentTypeController) Create(c *fiber.Ctx) error {
	var req dto.CreateAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	req.MasjidID = masjidIDFromToken

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	now := time.Now()
	row := model.AssessmentTypeModel{
		ID:            uuid.Nil, // biarkan DB generate (gen_random_uuid) atau BeforeCreate
		MasjidID:      req.MasjidID,
		Key:           strings.TrimSpace(req.Key),
		Name:          strings.TrimSpace(req.Name),
		WeightPercent: req.WeightPercent,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if req.IsActive != nil {
		row.IsActive = *req.IsActive
	}

	if err := ctl.DB.Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.Error(c, fiber.StatusConflict, "Key sudah dipakai untuk masjid ini")
		}
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.SuccessWithCode(c, fiber.StatusCreated, "Assessment type dibuat", mapToResponse(&row))
}

// PUT /assessment-types/:id  (partial update)
func (ctl *AssessmentTypeController) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var existing model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, masjidIDFromToken).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["assessment_types_name"] = strings.TrimSpace(*req.Name)
	}
	if req.WeightPercent != nil {
		updates["assessment_types_weight_percent"] = *req.WeightPercent
	}
	if req.IsActive != nil {
		updates["assessment_types_is_active"] = *req.IsActive
	}
	if len(updates) == 0 {
		return helper.Success(c, "OK", mapToResponse(&existing))
	}
	updates["assessment_types_updated_at"] = time.Now()

	if err := ctl.DB.Model(&model.AssessmentTypeModel{}).
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, masjidIDFromToken).
		Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.Error(c, fiber.StatusConflict, "Key sudah dipakai untuk masjid ini")
		}
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	var after model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ?", id).
		First(&after).Error; err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.Success(c, "Assessment type diperbarui", mapToResponse(&after))
}

// DELETE /assessment-types/:id (soft delete)
func (ctl *AssessmentTypeController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var row model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, masjidIDFromToken).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.Delete(&row).Error; err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	// Jika ingin strict 204 tanpa body, ganti ke: return c.SendStatus(fiber.StatusNoContent)
	return helper.SuccessWithCode(c, fiber.StatusNoContent, "Assessment type dihapus", nil)
}
