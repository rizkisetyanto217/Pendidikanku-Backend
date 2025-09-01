// file: internals/features/assessment/urls/controller/assessment_urls_controller.go
package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/attendance_assesment/assesment/dto"
	model "masjidku_backend/internals/features/school/attendance_assesment/assesment/model"
)

// AssessmentUrlsController mengelola endpoint CRUD assessment_urls
type AssessmentUrlsController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAssessmentUrlsController(db *gorm.DB) *AssessmentUrlsController {
	return &AssessmentUrlsController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ==========================
   Helpers
   ========================== */

func isDuplicateKey(err error) bool {
	// Tanpa pgconn: cek pesan error berisi "duplicate key value"
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key value")
}

/* ==========================
   Routes (contoh):
   g := app.Group("/api/a")
   urls := controller.NewAssessmentUrlsController(db)
   g.Post("/assessment-urls", urls.Create)                           // body contains assessment_id
   g.Get("/assessment-urls", urls.List)                              // ?assessment_id=&q=&is_published=&is_active=&page=&per_page=
   g.Get("/assessment-urls/:id", urls.GetByID)
   g.Patch("/assessment-urls/:id", urls.Update)
   g.Delete("/assessment-urls/:id", urls.Delete)

   // Optional nested (kalau mau):
   g.Post("/assessments/:assessment_id/urls", urls.Create)
   g.Get("/assessments/:assessment_id/urls", urls.List)
   ========================== */

// Create — buat URL untuk sebuah assessment
func (ctl *AssessmentUrlsController) Create(c *fiber.Ctx) error {
	var req dto.CreateAssessmentUrlsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Allow path param override (opsional)
	if pathID := strings.TrimSpace(c.Params("assessment_id")); pathID != "" {
		if id, err := uuid.Parse(pathID); err == nil {
			req.AssessmentUrlsAssessmentID = id
		}
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m := &model.AssessmentUrlsModel{
		AssessmentUrlsAssessmentID:    req.AssessmentUrlsAssessmentID,
		AssessmentUrlsLabel:           req.AssessmentUrlsLabel,
		AssessmentUrlsHref:            req.AssessmentUrlsHref,
		AssessmentUrlsTrashURL:        req.AssessmentUrlsTrashURL,
		AssessmentUrlsDeletePendingAt: req.AssessmentUrlsDeletePendingAt,
		AssessmentUrlsIsPublished:     req.AssessmentUrlsIsPublished,
		AssessmentUrlsIsActive:        req.AssessmentUrlsIsActive,
		AssessmentUrlsPublishedAt:     req.AssessmentUrlsPublishedAt,
		AssessmentUrlsExpiresAt:       req.AssessmentUrlsExpiresAt,
		AssessmentUrlsPublicSlug:      req.AssessmentUrlsPublicSlug,
		AssessmentUrlsPublicToken:     req.AssessmentUrlsPublicToken,
	}

	if err := ctl.DB.Create(m).Error; err != nil {
		if isDuplicateKey(err) {
			return fiber.NewError(fiber.StatusConflict, "URL sudah terdaftar untuk assessment ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusCreated).JSON(dto.ToAssessmentUrlsResponse(m))
}

// Update — patch-like
func (ctl *AssessmentUrlsController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateAssessmentUrlsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var existing model.AssessmentUrlsModel
	if err := ctl.DB.First(&existing, "assessment_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	updates := map[string]interface{}{}
	if req.AssessmentUrlsLabel != nil {
		updates["assessment_urls_label"] = req.AssessmentUrlsLabel
	}
	if req.AssessmentUrlsHref != nil {
		updates["assessment_urls_href"] = *req.AssessmentUrlsHref
	}
	if req.AssessmentUrlsTrashURL != nil {
		updates["assessment_urls_trash_url"] = req.AssessmentUrlsTrashURL
	}
	if req.AssessmentUrlsDeletePendingAt != nil {
		updates["assessment_urls_delete_pending_until"] = req.AssessmentUrlsDeletePendingAt
	}
	if req.AssessmentUrlsIsPublished != nil {
		updates["assessment_urls_is_published"] = *req.AssessmentUrlsIsPublished
	}
	if req.AssessmentUrlsIsActive != nil {
		updates["assessment_urls_is_active"] = *req.AssessmentUrlsIsActive
	}
	if req.AssessmentUrlsPublishedAt != nil {
		updates["assessment_urls_published_at"] = req.AssessmentUrlsPublishedAt
	}
	if req.AssessmentUrlsExpiresAt != nil {
		updates["assessment_urls_expires_at"] = req.AssessmentUrlsExpiresAt
	}
	if req.AssessmentUrlsPublicSlug != nil {
		updates["assessment_urls_public_slug"] = req.AssessmentUrlsPublicSlug
	}
	if req.AssessmentUrlsPublicToken != nil {
		updates["assessment_urls_public_token"] = req.AssessmentUrlsPublicToken
	}

	if len(updates) == 0 {
		// tidak ada perubahan
		return c.JSON(dto.ToAssessmentUrlsResponse(&existing))
	}

	if err := ctl.DB.Model(&existing).
		Where("assessment_urls_id = ?", id).
		Updates(updates).Error; err != nil {
		if isDuplicateKey(err) {
			return fiber.NewError(fiber.StatusConflict, "URL sudah terdaftar untuk assessment ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// reload
	if err := ctl.DB.First(&existing, "assessment_urls_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(dto.ToAssessmentUrlsResponse(&existing))
}

// GetByID
func (ctl *AssessmentUrlsController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	var m model.AssessmentUrlsModel
	if err := ctl.DB.First(&m, "assessment_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(dto.ToAssessmentUrlsResponse(&m))
}

// List dengan filter & pagination
func (ctl *AssessmentUrlsController) List(c *fiber.Ctx) error {
	// Filters
	var (
		assessmentIDStr = strings.TrimSpace(c.Query("assessment_id"))
		q               = strings.TrimSpace(c.Query("q"))
		isPublishedStr  = strings.TrimSpace(c.Query("is_published")) // "true"/"false"
		isActiveStr     = strings.TrimSpace(c.Query("is_active"))    // "true"/"false"
		pageStr         = strings.TrimSpace(c.Query("page", "1"))
		perPageStr      = strings.TrimSpace(c.Query("per_page", "20"))
	)

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(perPageStr)
	if perPage <= 0 || perPage > 200 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	db := ctl.DB.Model(&model.AssessmentUrlsModel{})
	if assessmentIDStr != "" {
		if assessmentID, err := uuid.Parse(assessmentIDStr); err == nil {
			db = db.Where("assessment_urls_assessment_id = ?", assessmentID)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "assessment_id tidak valid")
		}
	}
	if q != "" {
		db = db.Where("(assessment_urls_label ILIKE ? OR assessment_urls_href ILIKE ?)", "%"+q+"%", "%"+q+"%")
	}
	if isPublishedStr != "" {
		if v, err := strconv.ParseBool(isPublishedStr); err == nil {
			db = db.Where("assessment_urls_is_published = ?", v)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "is_published harus boolean")
		}
	}
	if isActiveStr != "" {
		if v, err := strconv.ParseBool(isActiveStr); err == nil {
			db = db.Where("assessment_urls_is_active = ?", v)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "is_active harus boolean")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.AssessmentUrlsModel
	if err := db.
		Order("assessment_urls_created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	out := make([]dto.AssessmentUrlsResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.ToAssessmentUrlsResponse(&rows[i]))
	}

	return c.JSON(fiber.Map{
		"data":       out,
		"page":       page,
		"per_page":   perPage,
		"total":      total,
		"total_page": (total + int64(perPage) - 1) / int64(perPage),
	})
}

// Delete — soft delete
func (ctl *AssessmentUrlsController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	if err := ctl.DB.Delete(&model.AssessmentUrlsModel{}, "assessment_urls_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(http.StatusNoContent)
}
