// file: internals/features/school/submissions/controller/submission_urls_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/submissions_assesment/submissions/dto"
	model "masjidku_backend/internals/features/school/submissions_assesment/submissions/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* =======================================================
   Controller
   ======================================================= */

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
   Utilities
   ========================== */

func stdReqFromFiber(c *fiber.Ctx) *http.Request {
	u := &url.URL{RawQuery: string(c.Request().URI().QueryString())}
	return &http.Request{URL: u}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "23505") || strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint")
}

// --- Scope helpers (tanpa asumsi struct submission model) ---
// Kita anggap tabel utama bernama `submissions` dengan kolom:
//   submissions_id (UUID PK), submissions_masjid_id (UUID nullable), submissions_user_id (UUID nullable)
func assertSubmissionScope(db *gorm.DB, submissionID uuid.UUID, userID uuid.UUID, masjidIDs []uuid.UUID) (bool, error) {
	if submissionID == uuid.Nil {
		return false, fiber.NewError(fiber.StatusBadRequest, "submission_id wajib")
	}
	// Build query: submission dimiliki user ATAU berada di salah satu masjid scope user
	q := db.
		Table("submissions").
		Select("1").
		Where("submissions_id = ?", submissionID).
		Where(db.
			Where("submissions_user_id = ?::uuid", userID).
			Or("submissions_masjid_id = ANY(?::uuid[])", uuidSliceToAnyArray(masjidIDs)),
		).
		Limit(1)

	var ok int
	if err := q.Scan(&ok).Error; err != nil {
		// Jika tabel submissions belum ada (dev mode), fallback: izinkan admin/DKM
		// -> tetapi tanpa info role, balikan error agar caller bisa putuskan
		return false, err
	}
	return ok == 1, nil
}

func uuidSliceToAnyArray(in []uuid.UUID) []uuid.UUID { // GORM psql driver paham uuid[] langsung
	if len(in) == 0 {
		// Agar ANY('{}') tidak error, tetap kembalikan slice kosong
		return []uuid.UUID{}
	}
	return in
}

func mustGetUserAndMasjidScope(c *fiber.Ctx) (userID uuid.UUID, masjidIDs []uuid.UUID, err error) {
	userID, err = helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return
	}
	masjidIDs, err = helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		// Boleh jadi user hanya owner submission (tanpa masjid scope) → kita pakai slice kosong
		masjidIDs = []uuid.UUID{}
		err = nil
	}
	return
}

/* ==========================
   Routes
   ========================== */

// Create — dukung JSON & multipart upload ke OSS
func (ctl *SubmissionUrlsController) Create(c *fiber.Ctx) error {
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ========= MULTIPART FORM-DATA =========
	if strings.HasPrefix(ct, "multipart/form-data") {
		var req dto.CreateSubmissionUrlRequest

		// Path param override
		if pathID := strings.TrimSpace(c.Params("submission_id")); pathID != "" {
			if id, err := uuid.Parse(pathID); err == nil {
				req.SubmissionUrlsSubmissionID = id
			}
		}
		// Jika belum ada dari path → coba dari form field
		if req.SubmissionUrlsSubmissionID == uuid.Nil {
			if s := strings.TrimSpace(c.FormValue("submission_id")); s != "" {
				if id, err := uuid.Parse(s); err == nil {
					req.SubmissionUrlsSubmissionID = id
				}
			}
		}

		// --- SECURITY: pastikan user boleh akses submission ini ---
		userID, masjidIDs, err := mustGetUserAndMasjidScope(c)
		if err != nil {
			return helper.JsonError(c, http.StatusUnauthorized, err.Error())
		}
		if req.SubmissionUrlsSubmissionID == uuid.Nil {
			return helper.JsonError(c, http.StatusBadRequest, "submission_id wajib (path atau form field)")
		}
		if ok, err := assertSubmissionScope(ctl.DB, req.SubmissionUrlsSubmissionID, userID, masjidIDs); err != nil {
			// jika error query, untuk menghindari kebocoran, tolak akses
			return helper.JsonError(c, http.StatusForbidden, "Akses ditolak (scope tidak tervalidasi)")
		} else if !ok {
			return helper.JsonError(c, http.StatusForbidden, "Akses ditolak untuk submission tersebut")
		}

		// Field text
		if v := strings.TrimSpace(c.FormValue("label")); v != "" {
			req.SubmissionUrlsLabel = &v
		}
		req.SubmissionUrlsHref = strings.TrimSpace(c.FormValue("href"))
		if v := strings.TrimSpace(c.FormValue("trash_url")); v != "" {
			req.SubmissionUrlsTrashURL = &v
		}
		if v := strings.TrimSpace(c.FormValue("delete_pending_until")); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return helper.JsonError(c, http.StatusBadRequest, "delete_pending_until harus RFC3339")
			}
			req.SubmissionUrlsDeletePendingUntil = &t
		}
		if v := strings.TrimSpace(c.FormValue("is_active")); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return helper.JsonError(c, http.StatusBadRequest, "is_active harus boolean")
			}
			req.SubmissionUrlsIsActive = &b
		}

		// File (opsional)
		fh, _ := c.FormFile("file")
		uploadedURL := ""
		if fh != nil {
			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, http.StatusBadGateway, "OSS init gagal: "+err.Error())
			}
			dir := fmt.Sprintf("submissions/%s/urls", req.SubmissionUrlsSubmissionID.String())
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			key, _, err := svc.UploadFromFormFileToDir(ctx, dir, fh)
			if err != nil {
				return helper.JsonError(c, http.StatusBadGateway, "Upload gagal: "+err.Error())
			}
			uploadedURL = svc.PublicURL(key)
			req.SubmissionUrlsHref = uploadedURL
		}

		// Validasi
		if err := ctl.Validator.Struct(&req); err != nil {
			// rollback object jika tadi upload sukses
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
			if isUniqueViolation(err) {
				return helper.JsonError(c, http.StatusConflict, "Href sudah terdaftar untuk submission ini")
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		return helper.JsonCreated(c, "Created", dto.ToSubmissionUrlResponse(row))
	}

	// ========= JSON (perilaku lama) =========
	var req dto.CreateSubmissionUrlRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}

	// Path param override
	if pathID := strings.TrimSpace(c.Params("submission_id")); pathID != "" {
		if id, err := uuid.Parse(pathID); err == nil {
			req.SubmissionUrlsSubmissionID = id
		}
	}

	// --- SECURITY ---
	userID, masjidIDs, err := mustGetUserAndMasjidScope(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, err.Error())
	}
	if req.SubmissionUrlsSubmissionID == uuid.Nil {
		return helper.JsonError(c, http.StatusBadRequest, "submission_id wajib")
	}
	if ok, err := assertSubmissionScope(ctl.DB, req.SubmissionUrlsSubmissionID, userID, masjidIDs); err != nil || !ok {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak untuk submission tersebut")
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
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "Href sudah terdaftar untuk submission ini")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Created", dto.ToSubmissionUrlResponse(row))
}

// Update — dukung JSON & multipart (upload baru menimpa href lama)
func (ctl *SubmissionUrlsController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	// Ambil existing
	var existing model.SubmissionUrlsModel
	if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// --- SECURITY: pastikan boleh akses submission induknya ---
	userID, masjidIDs, err := mustGetUserAndMasjidScope(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, err.Error())
	}
	if ok, err := assertSubmissionScope(ctl.DB, existing.SubmissionUrlsSubmissionID, userID, masjidIDs); err != nil || !ok {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak untuk submission tersebut")
	}

	oldHref := strings.TrimSpace(existing.SubmissionUrlsHref)

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	if strings.HasPrefix(ct, "multipart/form-data") {
		// ====== MULTIPART ======
		var req dto.UpdateSubmissionUrlRequest

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
				return helper.JsonError(c, http.StatusBadRequest, "delete_pending_until harus RFC3339")
			}
		}
		if v := strings.TrimSpace(c.FormValue("is_active")); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				req.SubmissionUrlsIsActive = &b
			} else {
				return helper.JsonError(c, http.StatusBadRequest, "is_active harus boolean")
			}
		}

		// File (opsional)
		fh, _ := c.FormFile("file")
		newUploadedURL := ""
		if fh != nil {
			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, http.StatusBadGateway, "OSS init gagal: "+err.Error())
			}
			dir := fmt.Sprintf("submissions/%s/urls", existing.SubmissionUrlsSubmissionID.String())
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			key, _, err := svc.UploadFromFormFileToDir(ctx, dir, fh)
			if err != nil {
				return helper.JsonError(c, http.StatusBadGateway, "Upload gagal: "+err.Error())
			}
			newUploadedURL = svc.PublicURL(key)
			req.SubmissionUrlsHref = &newUploadedURL
		}

		if err := ctl.Validator.Struct(&req); err != nil {
			// rollback file baru jika ada
			if newUploadedURL != "" {
				_ = helperOSS.DeleteByPublicURLENV(newUploadedURL, 15*time.Second)
			}
			return helper.ValidationError(c, err)
		}

		updates := dto.BuildSubmissionUrlUpdates(&req)
		if len(updates) == 0 {
			return helper.JsonOK(c, "Tidak ada perubahan", dto.ToSubmissionUrlResponse(&existing))
		}

		if err := ctl.DB.Model(&existing).
			Where("submission_urls_id = ?", id).
			Updates(updates).Error; err != nil {
			if newUploadedURL != "" {
				_ = helperOSS.DeleteByPublicURLENV(newUploadedURL, 15*time.Second)
			}
			if isUniqueViolation(err) {
				return helper.JsonError(c, http.StatusConflict, "Href sudah terdaftar untuk submission ini")
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		// Reload
		if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
			if newUploadedURL != "" {
				_ = helperOSS.DeleteByPublicURLENV(newUploadedURL, 15*time.Second)
			}
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}

		// Jika href berubah dan ada oldHref → pindahkan ke spam/ (best-effort)
		if newUploadedURL != "" && oldHref != "" && oldHref != existing.SubmissionUrlsHref {
			_, _ = helperOSS.MoveToSpamByPublicURLENV(oldHref, 0)
		}

		return helper.JsonUpdated(c, "Updated", dto.ToSubmissionUrlResponse(&existing))
	}

	// ====== JSON ======
	var req dto.UpdateSubmissionUrlRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	updates := dto.BuildSubmissionUrlUpdates(&req)
	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", dto.ToSubmissionUrlResponse(&existing))
	}

	if err := ctl.DB.Model(&existing).
		Where("submission_urls_id = ?", id).
		Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "Href sudah terdaftar untuk submission ini")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// reload
	if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "Updated", dto.ToSubmissionUrlResponse(&existing))
}

// GetByID — detail (scope-guarded)
func (ctl *SubmissionUrlsController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}
	var row model.SubmissionUrlsModel
	if err := ctl.DB.First(&row, "submission_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// --- SECURITY ---
	userID, masjidIDs, err := mustGetUserAndMasjidScope(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, err.Error())
	}
	if ok, err := assertSubmissionScope(ctl.DB, row.SubmissionUrlsSubmissionID, userID, masjidIDs); err != nil || !ok {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak untuk submission tersebut")
	}

	return helper.JsonOK(c, "OK", dto.ToSubmissionUrlResponse(&row))
}

// List — filter + pagination helper (scope-guarded)
// - Jika ada query submission_id → validasi scope spesifik submission itu
// - Jika TIDAK ada submission_id → otomatis batasi hasil ke submission yang berada di scope user
func (ctl *SubmissionUrlsController) List(c *fiber.Ctx) error {
	// Pagination (default: created_at DESC)
	p := helper.ParseWith(stdReqFromFiber(c), "created_at", "desc", helper.AdminOpts)

	// Whitelist kolom sorting
	allowedSort := map[string]string{
		"label":      "submission_urls_label",
		"href":       "submission_urls_href",
		"created_at": "submission_urls_created_at",
		"updated_at": "submission_urls_updated_at",
	}
	orderCol := allowedSort["created_at"]
	if col, ok := allowedSort[strings.ToLower(p.SortBy)]; ok {
		orderCol = col
	}
	orderDir := "DESC"
	if strings.ToLower(p.SortOrder) == "asc" {
		orderDir = "ASC"
	}

	submissionIDStr := strings.TrimSpace(c.Query("submission_id"))
	q := strings.TrimSpace(c.Query("q"))
	isActiveStr := strings.TrimSpace(c.Query("is_active"))

	userID, masjidIDs, err := mustGetUserAndMasjidScope(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, err.Error())
	}

	db := ctl.DB.Model(&model.SubmissionUrlsModel{})

	// Scope filtering
	if submissionIDStr != "" {
		sid, err := uuid.Parse(submissionIDStr)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "submission_id tidak valid")
		}
		if ok, err := assertSubmissionScope(ctl.DB, sid, userID, masjidIDs); err != nil || !ok {
			return helper.JsonError(c, http.StatusForbidden, "Akses ditolak untuk submission tersebut")
		}
		db = db.Where("submission_urls_submission_id = ?", sid)
	} else {
		// Tanpa submission_id → batasi berdasarkan scope user via subquery ke submissions
		db = db.Where(`
			EXISTS (
				SELECT 1 FROM submissions s
				WHERE s.submissions_id = submission_urls_submission_id
				  AND (
					   s.submissions_user_id = ?::uuid
					OR s.submissions_masjid_id = ANY(?::uuid[])
				  )
			)
		`, userID, uuidSliceToAnyArray(masjidIDs))
	}

	if q != "" {
		like := "%" + q + "%"
		db = db.Where("(submission_urls_label ILIKE ? OR submission_urls_href ILIKE ?)", like, like)
	}

	if isActiveStr != "" {
		if v, err := strconv.ParseBool(isActiveStr); err == nil {
			db = db.Where("submission_urls_is_active = ?", v)
		} else {
			return helper.JsonError(c, http.StatusBadRequest, "is_active harus boolean")
		}
	}

	// Count total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// Sorting & pagination
	db = db.Order(orderCol + " " + orderDir)
	if !p.All {
		db = db.Limit(p.Limit()).Offset(p.Offset())
	}

	var rows []model.SubmissionUrlsModel
	if err := db.Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	out := make([]dto.SubmissionUrlResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.ToSubmissionUrlResponse(&rows[i]))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}

// Delete — soft delete + move object to spam/ (scope-guarded)
func (ctl *SubmissionUrlsController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	// Load existing untuk ambil href + submission_id
	var existing model.SubmissionUrlsModel
	if err := ctl.DB.First(&existing, "submission_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// --- SECURITY ---
	userID, masjidIDs, err := mustGetUserAndMasjidScope(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, err.Error())
	}
	if ok, err := assertSubmissionScope(ctl.DB, existing.SubmissionUrlsSubmissionID, userID, masjidIDs); err != nil || !ok {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak untuk submission tersebut")
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

	// Set trash_url & delete_pending_until (default: 30 hari atau ENV RETENTION_DAYS)
	deletePendingUntil := time.Now().Add(30 * 24 * time.Hour)
	if v := strings.TrimSpace(os.Getenv("RETENTION_DAYS")); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			deletePendingUntil = time.Now().Add(time.Duration(n) * 24 * time.Hour)
		}
	}

	// Update kolom trash sebelum soft delete (jika ada spamURL)
	if spamURL != "" {
		if err := ctl.DB.Model(&existing).
			Where("submission_urls_id = ?", id).
			Updates(map[string]any{
				"submission_urls_trash_url":            spamURL,
				"submission_urls_delete_pending_until": deletePendingUntil,
			}).Error; err != nil {
			// lanjut soft delete tetap dilakukan, tapi catat warning
			ossWarn = strings.TrimSpace(ossWarn + "; db-update-trash-failed: " + err.Error())
		}
	}

	// Soft delete
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	resp := fiber.Map{
		"submission_urls_id":   id,
		"moved_to_spam":        spamURL,
		"delete_pending_until": deletePendingUntil,
	}
	if spamURL == "" {
		delete(resp, "moved_to_spam")
		delete(resp, "delete_pending_until")
	}
	if ossWarn != "" {
		resp["oss_warning"] = ossWarn
	}
	return helper.JsonDeleted(c, "Deleted", resp)
}
