// file: internals/features/assessment/urls/controller/assessment_urls_controller.go
package controller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mime/multipart"
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
	helperOSS "masjidku_backend/internals/helpers/oss"
)

/* =======================================================
   CONTROLLER
   ======================================================= */

type AssessmentUrlsController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAssessmentUrlsController(db *gorm.DB) *AssessmentUrlsController {
	return &AssessmentUrlsController{
		DB:        db,
		Validator: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (ctl *AssessmentUrlsController) ensureValidator() {
	if ctl.Validator == nil {
		ctl.Validator = validator.New(validator.WithRequiredStructEnabled())
	}
}

/* ==========================
   Utils
   ========================== */

func isPGUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint") || strings.Contains(s, "23505")
}

// trimPtr mengembalikan "" jika pointer nil, kalau tidak di-TrimSpace nilai *p
func trimPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// boolOr: ambil nilai pointer bool; kalau nil pakai default
func boolOr(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}

// strPtr: jadikan *string; kosong -> nil
func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// -- tambahkan import: "database/sql"
func toNullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}


/* ==========================
   Routes
   ========================== */

// POST /masjids/:masjid_id/images/:slot
// Form-Data: image | file | photo | picture | class_parent_image
// POST /assessment-urls
// POST /assessments/:assessment_id/urls
// POST /assessment-urls
// POST /assessments/:assessment_id/urls
func (ctl *AssessmentUrlsController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	var req dto.CreateAssessmentUrlsRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// override assessment_id dari path jika ada
	if s := strings.TrimSpace(c.Params("assessment_id")); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id pada path tidak valid")
		}
		req.AssessmentUrlsAssessmentID = id
	}

	// ==== dukung upload file via form-data ====
	var fh *multipart.FileHeader
	if f, err := c.FormFile("assessment_urls_href"); err == nil && f != nil {
		fh = f
	} else if f2, _ := helperOSS.GetImageFile(c); f2 != nil {
		fh = f2
	}

	// Jika href kosong tapi ada file → upload ke OSS, lalu set href = public URL
	if strings.TrimSpace(req.AssessmentUrlsHref) == "" && fh != nil {
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi: "+err.Error())
		}
		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()

		dir := fmt.Sprintf("assessments/%s/urls", req.AssessmentUrlsAssessmentID.String())
		key, _, err := svc.UploadFromFormFileToDir(ctx, dir, fh)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload ke OSS: "+err.Error())
		}
		req.AssessmentUrlsHref = svc.PublicURL(key)
	}

	// Validasi setelah href mungkin terisi dari upload
	if err := ctl.Validator.Struct(&req); err != nil {
		if strings.Contains(err.Error(), "AssessmentUrlsAssessmentID") || strings.Contains(err.Error(), "AssessmentUrlsHref") {
			return helper.JsonError(c, fiber.StatusBadRequest,
				"Wajib isi salah satu: 'assessment_urls_href' (URL) atau unggah file di field 'assessment_urls_href' / 'file'",
			)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// Nilai boolean dengan default
	isPublished := boolOr(req.AssessmentUrlsIsPublished, false)
	isActive := boolOr(req.AssessmentUrlsIsActive, true)

	// ====== BANGUN MODEL ======
	// >>> A. Jika field di model bertipe string (umum):
	m := &model.AssessmentUrlsModel{
		AssessmentUrlsAssessmentID:    req.AssessmentUrlsAssessmentID,
    	AssessmentUrlsLabel:           strPtr(req.AssessmentUrlsLabel),   // <-- was strings.TrimSpace(...)
		AssessmentUrlsHref:            strings.TrimSpace(req.AssessmentUrlsHref),
		AssessmentUrlsTrashURL:        req.AssessmentUrlsTrashURL,   // *string
		AssessmentUrlsDeletePendingAt: req.AssessmentUrlsDeletePendingAt,
		AssessmentUrlsIsPublished:     isPublished,                  // bool
		AssessmentUrlsIsActive:        isActive,                     // bool
		AssessmentUrlsPublishedAt:     req.AssessmentUrlsPublishedAt,
		AssessmentUrlsExpiresAt:       req.AssessmentUrlsExpiresAt,
		AssessmentUrlsPublicSlug:      req.AssessmentUrlsPublicSlug, // *string
		AssessmentUrlsPublicToken:     req.AssessmentUrlsPublicToken,// *string
	}


	if err := ctl.DB.Create(m).Error; err != nil {
		if isPGUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "URL sudah terdaftar untuk assessment ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	return helper.JsonCreated(c, "Assessment URL dibuat", dto.ToAssessmentUrlsResponse(m))
}


// PATCH /assessment-urls/:id
// PATCH /assessment-urls/:id
func (ctl *AssessmentUrlsController) Update(c *fiber.Ctx) error {
	ctl.ensureValidator()

	// 🔐 Gate: Admin/DKM/Teacher
	if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateAssessmentUrlsRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// Ambil data eksisting
	var existing model.AssessmentUrlsModel
	if err := ctl.DB.WithContext(c.Context()).
		First(&existing, "assessment_urls_id = ?", id).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 🔐 Tenant guard via assessment → masjid_id dari token harus cocok
	if mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && mid != uuid.Nil {
		var cnt int64
		if e := ctl.DB.WithContext(c.Context()).
			Table("assessments").
			Where("assessments_id = ? AND assessments_masjid_id = ?", existing.AssessmentUrlsAssessmentID, mid).
			Count(&cnt).Error; e != nil {

			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal verifikasi tenant")
		}
		if cnt == 0 {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak berhak mengubah resource dari masjid lain")
		}
	}

	// ==== dukung upload file via form-data (menggantikan href) ====
	var uploadedHref string
	if fh, err := c.FormFile("assessment_urls_href"); err == nil && fh != nil {
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi: "+er.Error())
		}
		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()
		dir := fmt.Sprintf("assessments/%s/urls", existing.AssessmentUrlsAssessmentID.String())
		key, _, upErr := svc.UploadFromFormFileToDir(ctx, dir, fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload ke OSS: "+upErr.Error())
		}
		uploadedHref = svc.PublicURL(key)
	}

	// Build updates
	updates := map[string]any{}

	if req.AssessmentUrlsLabel != nil {
		updates["assessment_urls_label"] = strings.TrimSpace(*req.AssessmentUrlsLabel)
	}

	// Sumber href baru:
	//   1) dari body (jika pointer non-nil), atau
	//   2) dari hasil upload file (uploadedHref)
	var newHref *string
	if req.AssessmentUrlsHref != nil {
		h := strings.TrimSpace(*req.AssessmentUrlsHref)
		newHref = &h
	} else if uploadedHref != "" {
		newHref = &uploadedHref
	}

	if newHref != nil {
		updates["assessment_urls_href"] = *newHref
	}

	if req.AssessmentUrlsTrashURL != nil {
		updates["assessment_urls_trash_url"] = strings.TrimSpace(*req.AssessmentUrlsTrashURL)
	}
	if req.AssessmentUrlsDeletePendingAt != nil {
		updates["assessment_urls_delete_pending_at"] = *req.AssessmentUrlsDeletePendingAt
	}
	if req.AssessmentUrlsIsPublished != nil {
		updates["assessment_urls_is_published"] = *req.AssessmentUrlsIsPublished
	}
	if req.AssessmentUrlsIsActive != nil {
		updates["assessment_urls_is_active"] = *req.AssessmentUrlsIsActive
	}
	if req.AssessmentUrlsPublishedAt != nil {
		updates["assessment_urls_published_at"] = *req.AssessmentUrlsPublishedAt
	}
	if req.AssessmentUrlsExpiresAt != nil {
		updates["assessment_urls_expires_at"] = *req.AssessmentUrlsExpiresAt
	}
	if req.AssessmentUrlsPublicSlug != nil {
		updates["assessment_urls_public_slug"] = strings.TrimSpace(*req.AssessmentUrlsPublicSlug)
	}
	if req.AssessmentUrlsPublicToken != nil {
		updates["assessment_urls_public_token"] = strings.TrimSpace(*req.AssessmentUrlsPublicToken)
	}

	// Integrasi OSS:
	// - Jika href berubah & trash_url belum diisi → pindahkan href lama ke spam/
	// - Atau jika delete_pending_at diisi & belum ada trash_url → pindahkan href lama ke spam/
	oldHref := strings.TrimSpace(existing.AssessmentUrlsHref)
	oldTrash := trimPtr(existing.AssessmentUrlsTrashURL)

	newHrefGiven := (newHref != nil) && (strings.TrimSpace(*newHref) != oldHref)
	trashGiven := (req.AssessmentUrlsTrashURL != nil && strings.TrimSpace(*req.AssessmentUrlsTrashURL) != "")
	delPendingGiven := (req.AssessmentUrlsDeletePendingAt != nil)

	needMoveToSpam := false
	if newHrefGiven && oldHref != "" && !trashGiven {
		needMoveToSpam = true
	}
	if delPendingGiven && oldHref != "" && oldTrash == "" && !trashGiven {
		needMoveToSpam = true
	}
	if needMoveToSpam {
		dstURL, moveErr := helperOSS.MoveToSpamByPublicURLENV(oldHref, 20*time.Second)
		if moveErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Gagal memindahkan file lama ke spam: "+moveErr.Error())
		}
		updates["assessment_urls_trash_url"] = dstURL
	}

	// Tidak ada perubahan
	if len(updates) == 0 {
		return helper.JsonOK(c, "OK", dto.ToAssessmentUrlsResponse(&existing))
	}

	// Commit
	if err := ctl.DB.WithContext(c.Context()).
		Model(&existing).
		Where("assessment_urls_id = ?", id).
		Updates(updates).Error; err != nil {

		if isPGUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "URL sudah terdaftar untuk assessment ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengubah data")
	}

	// Reload
	if err := ctl.DB.WithContext(c.Context()).
		First(&existing, "assessment_urls_id = ?", id).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat ulang data")
	}
	return helper.JsonUpdated(c, "Assessment URL diperbarui", dto.ToAssessmentUrlsResponse(&existing))
}


// GET /assessment-urls/:id
func (ctl *AssessmentUrlsController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	var m model.AssessmentUrlsModel
	if err := ctl.DB.First(&m, "assessment_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return helper.JsonOK(c, "OK", dto.ToAssessmentUrlsResponse(&m))
}

// GET /assessment-urls
// GET /assessments/:assessment_id/urls
func (ctl *AssessmentUrlsController) List(c *fiber.Ctx) error {
	// pagination (default: created_at desc)
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// optional filter by path param
	var assessmentID *uuid.UUID
	if s := strings.TrimSpace(c.Params("assessment_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			assessmentID = &id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id pada path tidak valid")
		}
	}

	// query string filters
	if assessmentID == nil {
		if s := strings.TrimSpace(c.Query("assessment_id")); s != "" {
			if id, err := uuid.Parse(s); err == nil {
				assessmentID = &id
			} else {
				return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
			}
		}
	}
	q := strings.TrimSpace(c.Query("q"))
	isPublishedStr := strings.TrimSpace(c.Query("is_published"))
	isActiveStr := strings.TrimSpace(c.Query("is_active"))

	db := ctl.DB.Model(&model.AssessmentUrlsModel{})
	if assessmentID != nil {
		db = db.Where("assessment_urls_assessment_id = ?", *assessmentID)
	}
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		db = db.Where("(LOWER(assessment_urls_label) LIKE ? OR LOWER(assessment_urls_href) LIKE ?)", like, like)
	}
	if isPublishedStr != "" {
		switch strings.ToLower(isPublishedStr) {
		case "true", "1", "t", "yes", "y":
			db = db.Where("assessment_urls_is_published = ?", true)
		case "false", "0", "f", "no", "n":
			db = db.Where("assessment_urls_is_published = ?", false)
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "is_published harus boolean")
		}
	}
	if isActiveStr != "" {
		switch strings.ToLower(isActiveStr) {
		case "true", "1", "t", "yes", "y":
			db = db.Where("assessment_urls_is_active = ?", true)
		case "false", "0", "f", "no", "n":
			db = db.Where("assessment_urls_is_active = ?", false)
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "is_active harus boolean")
		}
	}

	// total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// fetch
	if !p.All {
		db = db.Limit(p.Limit()).Offset(p.Offset())
	}
	var rows []model.AssessmentUrlsModel
	if err := db.Order("assessment_urls_created_at DESC").Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]dto.AssessmentUrlsResponse, 0, len(rows))
	for i := range rows {
		items = append(items, dto.ToAssessmentUrlsResponse(&rows[i]))
	}

	return helper.JsonList(c, items, helper.BuildMeta(total, p))
}

// DELETE /assessment-urls/:id  (soft-delete bila model pakai gorm.DeletedAt)
func (ctl *AssessmentUrlsController) Delete(c *fiber.Ctx) error {
	// 🔐 Gate: Admin/DKM/Teacher
	if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil dulu untuk tahu URL & tenant
	var existing model.AssessmentUrlsModel
	if err := ctl.DB.WithContext(c.Context()).
		First(&existing, "assessment_urls_id = ?", id).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 🔐 Tenant guard via assessment
	if mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && mid != uuid.Nil {
		var cnt int64
		if e := ctl.DB.WithContext(c.Context()).
			Table("assessments").
			Where("assessments_id = ? AND assessments_masjid_id = ?", existing.AssessmentUrlsAssessmentID, mid).
			Count(&cnt).Error; e != nil {

			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal verifikasi tenant")
		}
		if cnt == 0 {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak berhak menghapus resource dari masjid lain")
		}
	}

	// Best-effort hapus objek di OSS (aktif &/atau trash)
	if u := strings.TrimSpace(existing.AssessmentUrlsHref); u != "" {
		_ = helperOSS.DeleteByPublicURLENV(u, 15*time.Second)
	}
	if u := trimPtr(existing.AssessmentUrlsTrashURL); u != "" {
		_ = helperOSS.DeleteByPublicURLENV(u, 15*time.Second)
	}

	// Hapus row (soft delete jika model pakai gorm.DeletedAt)
	if err := ctl.DB.WithContext(c.Context()).
		Delete(&model.AssessmentUrlsModel{}, "assessment_urls_id = ?", id).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	return helper.JsonDeleted(c, "Assessment URL dihapus", fiber.Map{"deleted": true})
}