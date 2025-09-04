// file: internals/features/school/submissions/controller/submission_urls_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/sessions_assesment/submissions/dto"
	model "masjidku_backend/internals/features/school/sessions_assesment/submissions/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
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
// Create — buat URL untuk sebuah submission (dukung multipart upload ke OSS)
func (ctl *SubmissionUrlsController) Create(c *fiber.Ctx) error {
	ctype := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ========= CABANG MULTIPART =========
	if strings.HasPrefix(ctype, "multipart/form-data") {
		var req dto.CreateSubmissionUrlRequest

		// Path param override
		if pathID := strings.TrimSpace(c.Params("submission_id")); pathID != "" {
			if id, err := uuid.Parse(pathID); err == nil {
				req.SubmissionUrlsSubmissionID = id
			}
		}
		// Jika belum ada dari path, coba dari form field
		if req.SubmissionUrlsSubmissionID == uuid.Nil {
			if s := strings.TrimSpace(c.FormValue("submission_id")); s != "" {
				if id, err := uuid.Parse(s); err == nil {
					req.SubmissionUrlsSubmissionID = id
				}
			}
		}

		// Field text
		req.SubmissionUrlsLabel = func(s string) *string { if s == "" { return nil }; return &s }(strings.TrimSpace(c.FormValue("label")))
		req.SubmissionUrlsHref = strings.TrimSpace(c.FormValue("href")) // boleh kosong, akan diisi dari upload bila ada
		trashURL := strings.TrimSpace(c.FormValue("trash_url"))
		if trashURL != "" {
			req.SubmissionUrlsTrashURL = &trashURL
		}

		// Optional: delete_pending_until (RFC3339)
		if v := strings.TrimSpace(c.FormValue("delete_pending_until")); v != "" {
			if ts, err := time.Parse(time.RFC3339, v); err == nil {
				req.SubmissionUrlsDeletePendingUntil = &ts
			} else {
				return helper.Error(c, http.StatusBadRequest, "delete_pending_until harus RFC3339")
			}
		}

		// Optional: is_active
		if v := strings.TrimSpace(c.FormValue("is_active")); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				req.SubmissionUrlsIsActive = &b
			} else {
				return helper.Error(c, http.StatusBadRequest, "is_active harus boolean")
			}
		}

		// Optional file upload
		fh, _ := c.FormFile("file")
		uploadedURL := ""
		if fh != nil {
			// init OSS service
			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.Error(c, http.StatusBadGateway, "OSS init gagal: "+err.Error())
			}

			// Dir tujuan: submissions/{submission_id}/urls
			if req.SubmissionUrlsSubmissionID == uuid.Nil {
				return helper.Error(c, http.StatusBadRequest, "submission_id wajib (path atau form field)")
			}
			dir := fmt.Sprintf("submissions/%s/urls", req.SubmissionUrlsSubmissionID.String())

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			key, _, err := svc.UploadFromFormFileToDir(ctx, dir, fh) // upload apa adanya, detect content-type otomatis
			if err != nil {
				return helper.Error(c, http.StatusBadGateway, "Upload gagal: "+err.Error())
			}
			uploadedURL = svc.PublicURL(key)
			req.SubmissionUrlsHref = uploadedURL // pakai URL hasil upload
		}

		// Validasi DTO setelah kemungkinan diisi oleh upload
		if err := ctl.Validator.Struct(&req); err != nil {
			// rollback object jika tadi upload sukses tapi payload invalid
			if uploadedURL != "" {
				_ = helperOSS.DeleteByPublicURLENV(uploadedURL, 15*time.Second)
			}
			return helper.ValidationError(c, err)
		}

		row := &model.SubmissionUrlsModel{
			SubmissionUrlsSubmissionID:       req.SubmissionUrlsSubmissionID,
			SubmissionUrlsLabel:              req.SubmissionUrlsLabel,
			SubmissionUrlsHref:               req.SubmissionUrlsHref,
			SubmissionUrlsTrashURL:           req.SubmissionUrlsTrashURL,
			SubmissionUrlsDeletePendingUntil: req.SubmissionUrlsDeletePendingUntil,
			SubmissionUrlsIsActive:           true,
		}
		if req.SubmissionUrlsIsActive != nil {
			row.SubmissionUrlsIsActive = *req.SubmissionUrlsIsActive
		}

		if err := ctl.DB.Create(row).Error; err != nil {
			// rollback object kalau barusan upload
			if uploadedURL != "" {
				_ = helperOSS.DeleteByPublicURLENV(uploadedURL, 15*time.Second)
			}
			if isDuplicateKey(err) {
				return helper.Error(c, http.StatusConflict, "Href sudah terdaftar untuk submission ini")
			}
			return helper.Error(c, http.StatusInternalServerError, err.Error())
		}

		return helper.SuccessWithCode(c, http.StatusCreated, "Submission URL berhasil dibuat", dto.ToSubmissionUrlResponse(row))
	}

	// ========= CABANG JSON (perilaku lama) =========
	var req dto.CreateSubmissionUrlRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, http.StatusBadRequest, "Payload tidak valid")
	}

	// Path param override
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
		SubmissionUrlsIsActive:           true,
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
// Update — patch-like (+ dukung multipart upload ke OSS)
func (ctl *SubmissionUrlsController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "ID tidak valid")
	}

	// Ambil existing untuk dapat submission_id & href lama
	var existing model.SubmissionUrlsModel
	if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}
	oldHref := strings.TrimSpace(existing.SubmissionUrlsHref)

	ctype := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	if strings.HasPrefix(ctype, "multipart/form-data") {
		// ====== CABANG MULTIPART ======
		var req dto.UpdateSubmissionUrlRequest

		// Parse optional fields dari form
		if v := strings.TrimSpace(c.FormValue("label")); v != "" {
			req.SubmissionUrlsLabel = &v
		}
		if v := strings.TrimSpace(c.FormValue("href")); v != "" {
			req.SubmissionUrlsHref = &v
		}
		if v := strings.TrimSpace(c.FormValue("trash_url")); v != "" {
			req.SubmissionUrlsTrashURL = &v
		}
		if v := strings.TrimSpace(c.FormValue("delete_pending_until")); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				req.SubmissionUrlsDeletePendingUntil = &t
			} else {
				return helper.Error(c, http.StatusBadRequest, "delete_pending_until harus RFC3339")
			}
		}
		if v := strings.TrimSpace(c.FormValue("is_active")); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				req.SubmissionUrlsIsActive = &b
			} else {
				return helper.Error(c, http.StatusBadRequest, "is_active harus boolean")
			}
		}

		// Optional file
		fh, _ := c.FormFile("file")
		newUploadedURL := ""
		if fh != nil {
			// init OSS
			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.Error(c, http.StatusBadGateway, "OSS init gagal: "+err.Error())
			}
			dir := fmt.Sprintf("submissions/%s/urls", existing.SubmissionUrlsSubmissionID.String())

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			key, _, err := svc.UploadFromFormFileToDir(ctx, dir, fh) // upload apa adanya
			if err != nil {
				return helper.Error(c, http.StatusBadGateway, "Upload gagal: "+err.Error())
			}
			newUploadedURL = svc.PublicURL(key)
			req.SubmissionUrlsHref = &newUploadedURL // override href ke hasil upload
		}

		// Validasi payload setelah kemungkinan diisi upload
		if err := ctl.Validator.Struct(&req); err != nil {
			// rollback file baru kalau tadi upload
			if newUploadedURL != "" {
				_ = helperOSS.DeleteByPublicURLENV(newUploadedURL, 15*time.Second)
			}
			return helper.ValidationError(c, err)
		}

		updates := dto.BuildSubmissionUrlUpdates(&req)
		if len(updates) == 0 {
			// tidak ada perubahan
			return helper.Success(c, "Tidak ada perubahan", dto.ToSubmissionUrlResponse(&existing))
		}

		// Apply update
		if err := ctl.DB.Model(&existing).
			Where("submission_urls_id = ?", id).
			Updates(updates).Error; err != nil {
			// rollback file baru kalau DB gagal
			if newUploadedURL != "" {
				_ = helperOSS.DeleteByPublicURLENV(newUploadedURL, 15*time.Second)
			}
			if isDuplicateKey(err) {
				return helper.Error(c, http.StatusConflict, "Href sudah terdaftar untuk submission ini")
			}
			return helper.Error(c, http.StatusInternalServerError, err.Error())
		}

		// Reload
		if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
			// rollback file baru kalau gagal reload
			if newUploadedURL != "" {
				_ = helperOSS.DeleteByPublicURLENV(newUploadedURL, 15*time.Second)
			}
			return helper.Error(c, http.StatusInternalServerError, err.Error())
		}

		// Jika href berubah & ada oldHref → pindahkan ke spam/ (best-effort)
		if newUploadedURL != "" && oldHref != "" && oldHref != existing.SubmissionUrlsHref {
			if _, err := helperOSS.MoveToSpamByPublicURLENV(oldHref, 0); err != nil {
				// best-effort: boleh di-log kalau ada logger
				// log.Printf("[WARN] move to spam failed: %v", err)
			}
		}

		return helper.Success(c, "Submission URL berhasil diperbarui", dto.ToSubmissionUrlResponse(&existing))
	}

	// ====== CABANG JSON (perilaku lama) ======
	var req dto.UpdateSubmissionUrlRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, http.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	updates := dto.BuildSubmissionUrlUpdates(&req)
	if len(updates) == 0 {
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
// Delete — soft delete + move object to spam/
func (ctl *SubmissionUrlsController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "ID tidak valid")
	}

	// Load existing untuk ambil href
	var existing model.SubmissionUrlsModel
	if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	// Pindahkan file aktif ke spam/ (best-effort)
	var spamURL string
	var ossWarn string
	href := strings.TrimSpace(existing.SubmissionUrlsHref)
	if href != "" {
		if u, err := helperOSS.MoveToSpamByPublicURLENV(href, 15*time.Second); err == nil {
			spamURL = u
		} else {
			ossWarn = err.Error()
		}
	}

	// Set trash_url & delete_pending_until (default dari ENV RETENTION_DAYS -> 30)
	deletePendingUntil := time.Now().Add(30 * 24 * time.Hour)
	if v := strings.TrimSpace(os.Getenv("RETENTION_DAYS")); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			deletePendingUntil = time.Now().Add(time.Duration(n) * 24 * time.Hour)
		}
	}

	// Update kolom trash sebelum soft delete (kalau ada spamURL)
	if spamURL != "" {
		if err := ctl.DB.Model(&existing).
			Where("submission_urls_id = ?", id).
			Updates(map[string]any{
				"submission_urls_trash_url":            spamURL,
				"submission_urls_delete_pending_until": deletePendingUntil,
			}).Error; err != nil {
			// lanjut soft delete tetap dilakukan, tapi informasikan error update via warning
			ossWarn = strings.TrimSpace(ossWarn + "; db-update-trash-failed: " + err.Error())
		}
	}

	// Soft delete
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	resp := fiber.Map{
		"submission_urls_id": id,
	}
	if spamURL != "" {
		resp["moved_to_spam"] = spamURL
		resp["delete_pending_until"] = deletePendingUntil
	}
	if ossWarn != "" {
		resp["oss_warning"] = ossWarn
	}
	return helper.SuccessWithCode(c, http.StatusOK, "Submission URL dihapus", resp)
}
