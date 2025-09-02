// file: internals/features/school/submissions/controller/submission_urls_controller.go
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

	dto "masjidku_backend/internals/features/school/sessions_assesment/submissions/dto"
	model "masjidku_backend/internals/features/school/sessions_assesment/submissions/model"
	helper "masjidku_backend/internals/helpers"
)

// SubmissionUrlsController mengelola endpoint CRUD submission_urls
type SubmissionUrlsController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewSubmissionUrlsController(db *gorm.DB) *SubmissionUrlsController {
	return &SubmissionUrlsController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ==========================
   Helpers
   ========================== */

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key value") || strings.Contains(s, "unique constraint")
}

/* ==========================
   Routes (contoh wiring)
   ==========================
g := app.Group("/api/a")
ctrl := controller.NewSubmissionUrlsController(db)

g.Post("/submission-urls", ctrl.Create)
g.Get("/submission-urls", ctrl.List) // ?submission_id=&q=&is_active=&page=&per_page=
g.Get("/submission-urls/:id", ctrl.GetByID)
g.Patch("/submission-urls/:id", ctrl.Update)
g.Delete("/submission-urls/:id", ctrl.Delete)

// (opsional nested)
g.Post("/submissions/:submission_id/urls", ctrl.Create)
g.Get("/submissions/:submission_id/urls", ctrl.List)
======================================== */

// Create — buat URL untuk sebuah submission
func (ctl *SubmissionUrlsController) Create(c *fiber.Ctx) error {
	var req dto.CreateSubmissionUrlRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, http.StatusBadRequest, "Payload tidak valid")
	}

	// Allow path param override untuk nested route
	if pathID := strings.TrimSpace(c.Params("submission_id")); pathID != "" {
		if id, err := uuid.Parse(pathID); err == nil {
			req.SubmissionUrlsSubmissionID = id
		}
	}

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	row := &model.SubmissionUrlsModel{
		SubmissionUrlsSubmissionID:       req.SubmissionUrlsSubmissionID,
		SubmissionUrlsLabel:              req.SubmissionUrlsLabel,
		SubmissionUrlsHref:               req.SubmissionUrlsHref,
		SubmissionUrlsTrashURL:           req.SubmissionUrlsTrashURL,
		SubmissionUrlsDeletePendingUntil: req.SubmissionUrlsDeletePendingUntil,
		SubmissionUrlsIsActive:           true, // default
	}

	if req.SubmissionUrlsIsActive != nil {
		row.SubmissionUrlsIsActive = *req.SubmissionUrlsIsActive
	}

	if err := ctl.DB.Create(row).Error; err != nil {
		if isDuplicateKey(err) {
			return helper.Error(c, http.StatusConflict, "Href sudah terdaftar untuk submission ini")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	return helper.SuccessWithCode(c, http.StatusCreated, "Submission URL berhasil dibuat", dto.ToSubmissionUrlResponse(row))
}

// Update — patch-like
func (ctl *SubmissionUrlsController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateSubmissionUrlRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, http.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	var existing model.SubmissionUrlsModel
	if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	updates := dto.BuildSubmissionUrlUpdates(&req)
	if len(updates) == 0 {
		// tidak ada perubahan
		return helper.Success(c, "Tidak ada perubahan", dto.ToSubmissionUrlResponse(&existing))
	}

	if err := ctl.DB.Model(&existing).
		Where("submission_urls_id = ?", id).
		Updates(updates).Error; err != nil {
	 if isDuplicateKey(err) {
			return helper.Error(c, http.StatusConflict, "Href sudah terdaftar untuk submission ini")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	// reload
	if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}
	return helper.Success(c, "Submission URL berhasil diperbarui", dto.ToSubmissionUrlResponse(&existing))
}

// GetByID — detail
func (ctl *SubmissionUrlsController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "ID tidak valid")
	}
	var row model.SubmissionUrlsModel
	if err := ctl.DB.First(&row, "submission_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}
	return helper.Success(c, "OK", dto.ToSubmissionUrlResponse(&row))
}

// List — dengan filter & pagination
func (ctl *SubmissionUrlsController) List(c *fiber.Ctx) error {
	submissionIDStr := strings.TrimSpace(c.Query("submission_id"))
	q := strings.TrimSpace(c.Query("q")) // cari di label/href
	isActiveStr := strings.TrimSpace(c.Query("is_active"))
	pageStr := strings.TrimSpace(c.Query("page", "1"))
	perPageStr := strings.TrimSpace(c.Query("per_page", "20"))

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(perPageStr)
	if perPage <= 0 || perPage > 200 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	db := ctl.DB.Model(&model.SubmissionUrlsModel{})

	if submissionIDStr != "" {
		if sid, err := uuid.Parse(submissionIDStr); err == nil {
			db = db.Where("submission_urls_submission_id = ?", sid)
		} else {
			return helper.Error(c, http.StatusBadRequest, "submission_id tidak valid")
		}
	}

	if q != "" {
		like := "%" + q + "%"
		db = db.Where("(submission_urls_label ILIKE ? OR submission_urls_href ILIKE ?)", like, like)
	}

	if isActiveStr != "" {
		if v, err := strconv.ParseBool(isActiveStr); err == nil {
			db = db.Where("submission_urls_is_active = ?", v)
		} else {
			return helper.Error(c, http.StatusBadRequest, "is_active harus boolean")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	var rows []model.SubmissionUrlsModel
	if err := db.
		Order("submission_urls_created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	out := make([]dto.SubmissionUrlResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.ToSubmissionUrlResponse(&rows[i]))
	}

	return helper.Success(c, "OK", fiber.Map{
		"items":      out,
		"page":       page,
		"per_page":   perPage,
		"total":      total,
		"total_page": (total + int64(perPage) - 1) / int64(perPage),
	})
}

// Delete — soft delete
func (ctl *SubmissionUrlsController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "ID tidak valid")
	}
	if err := ctl.DB.Delete(&model.SubmissionUrlsModel{}, "submission_urls_id = ?", id).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}
	return helper.SuccessWithCode(c, http.StatusNoContent, "Submission URL dihapus", nil)
}
