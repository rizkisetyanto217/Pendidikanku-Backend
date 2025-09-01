// file: internals/features/school/assessments/controller/assessment_type_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/attendance_assesment/assesment/dto"
	model "masjidku_backend/internals/features/school/attendance_assesment/assesment/model"
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
	// aman untuk pgx/pq: cari code 23505 atau frasa umum
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "SQLSTATE 23505") ||
		strings.Contains(strings.ToLower(s), "duplicate key value") ||
		strings.Contains(strings.ToLower(s), "unique constraint")
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
	// Ambil tenant dari token (prioritas), kalau tidak ada gunakan query masjid_id (wajib salah satu)
	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)

	var filt dto.ListAssessmentTypeFilter
	// manual parse query agar fleksibel
	if q := c.Query("masjid_id"); q != "" {
		if v, err := uuid.Parse(q); err == nil {
			filt.MasjidID = v
		}
	}
	if masjidIDFromToken != uuid.Nil {
		filt.MasjidID = masjidIDFromToken
	}
	if filt.MasjidID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib (token atau query)")
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

	// validasi ringan
	if err := ctl.Validator.Struct(&filt); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Query builder
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
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.AssessmentTypeModel
	if err := qry.
		Order(getSortClause(filt.SortBy, filt.SortDir)).
		Limit(filt.Limit).Offset(filt.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.AssessmentTypeResponse, 0, len(rows))
	for i := range rows {
		out = append(out, mapToResponse(&rows[i]))
	}

	return c.Status(http.StatusOK).JSON(dto.ListAssessmentTypeResponse{
		Data:   out,
		Total:  total,
		Limit:  filt.Limit,
		Offset: filt.Offset,
	})
}

// GET /assessment-types/:id
func (ctl *AssessmentTypeController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var row model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, masjidIDFromToken).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusOK).JSON(mapToResponse(&row))
}

// POST /assessment-types
func (ctl *AssessmentTypeController) Create(c *fiber.Ctx) error {
	var req dto.CreateAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Ambil masjid_id dari token (override demi keamanan tenant)
	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	req.MasjidID = masjidIDFromToken

	if err := ctl.Validator.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	now := time.Now()
	row := model.AssessmentTypeModel{
		ID:            uuid.Nil, // biarkan DB generate (gen_random_uuid) atau pakai BeforeCreate jika mau app-side
		MasjidID:      req.MasjidID,
		Key:           strings.TrimSpace(req.Key),
		Name:          strings.TrimSpace(req.Name),
		WeightPercent: req.WeightPercent,
		IsActive:      true, // default
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// override IsActive jika dikirim
	if req.IsActive != nil {
		row.IsActive = *req.IsActive
	}

	if err := ctl.DB.Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return fiber.NewError(fiber.StatusConflict, "Key sudah dipakai untuk masjid ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusCreated).JSON(mapToResponse(&row))
}

// PUT /assessment-types/:id  (partial update)
func (ctl *AssessmentTypeController) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	// pastikan ownership
	var existing model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, masjidIDFromToken).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
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
		return c.Status(http.StatusOK).JSON(mapToResponse(&existing))
	}

	updates["assessment_types_updated_at"] = time.Now()

	if err := ctl.DB.Model(&model.AssessmentTypeModel{}).
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, masjidIDFromToken).
		Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return fiber.NewError(fiber.StatusConflict, "Key sudah dipakai untuk masjid ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// reload
	var after model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ?", id).
		First(&after).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusOK).JSON(mapToResponse(&after))
}

// DELETE /assessment-types/:id (soft delete)
func (ctl *AssessmentTypeController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var row model.AssessmentTypeModel
	if err := ctl.DB.
		Where("assessment_types_id = ? AND assessment_types_masjid_id = ?", id, masjidIDFromToken).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.Delete(&row).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(http.StatusNoContent)
}
