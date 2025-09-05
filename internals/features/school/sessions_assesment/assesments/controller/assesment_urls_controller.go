// file: internals/features/assessment/urls/controller/assessment_urls_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/sessions_assesment/assesments/dto"
	model "masjidku_backend/internals/features/school/sessions_assesment/assesments/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
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

/* ==========================
   Routes
   ========================== */

// POST /assessment-urls
// POST /assessments/:assessment_id/urls
func (ctl *AssessmentUrlsController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	// 🔐 Gate: hanya Admin/DKM/Teacher
	if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	var req dto.CreateAssessmentUrlsRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Optional: ambil dari path param jika ada
	if s := strings.TrimSpace(c.Params("assessment_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			req.AssessmentUrlsAssessmentID = id
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id pada path tidak valid")
		}
	}

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// (Opsional, jika ingin mengikat ke masjid aktif)
	// activeMasjidID, _ := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	// TODO: validasi bahwa assessment milik masjid tersebut (join ke tabel assessments) bila diperlukan.

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
		if isPGUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "URL sudah terdaftar untuk assessment ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	return helper.JsonCreated(c, "Assessment URL dibuat", dto.ToAssessmentUrlsResponse(m))
}

// PATCH-like: PATCH /assessment-urls/:id
func (ctl *AssessmentUrlsController) Update(c *fiber.Ctx) error {
	ctl.ensureValidator()

	// 🔐 Gate: hanya Admin/DKM/Teacher
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

	var existing model.AssessmentUrlsModel
	if err := ctl.DB.First(&existing, "assessment_urls_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// (Opsional) Pastikan user berhak mengubah data ini berdasarkan masjid assessment
	// activeMasjidID, _ := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	// TODO: join ke assessments untuk cek kepemilikan jika diperlukan.

	updates := map[string]any{}
	if req.AssessmentUrlsLabel != nil {
		updates["assessment_urls_label"] = *req.AssessmentUrlsLabel
	}
	if req.AssessmentUrlsHref != nil {
		updates["assessment_urls_href"] = *req.AssessmentUrlsHref
	}
	if req.AssessmentUrlsTrashURL != nil {
		updates["assessment_urls_trash_url"] = *req.AssessmentUrlsTrashURL
	}
	if req.AssessmentUrlsDeletePendingAt != nil {
		// ✅ perbaikan nama kolom
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
		updates["assessment_urls_public_slug"] = *req.AssessmentUrlsPublicSlug
	}
	if req.AssessmentUrlsPublicToken != nil {
		updates["assessment_urls_public_token"] = *req.AssessmentUrlsPublicToken
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "OK", dto.ToAssessmentUrlsResponse(&existing))
	}

	if err := ctl.DB.Model(&existing).
		Where("assessment_urls_id = ?", id).
		Updates(updates).Error; err != nil {
		if isPGUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "URL sudah terdaftar untuk assessment ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengubah data")
	}

	// reload
	if err := ctl.DB.First(&existing, "assessment_urls_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat ulang data")
	}
	return helper.JsonUpdated(c, "Assessment URL diperbarui", dto.ToAssessmentUrlsResponse(&existing))
}

// GET /assessment-urls/:id
func (ctl *AssessmentUrlsController) GetByID(c *fiber.Ctx) error {
	// (Opsional) batasi hanya Admin/DKM/Teacher
	// if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
	// 	return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	// }

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
	// (Opsional) batasi hanya Admin/DKM/Teacher
	// if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
	// 	return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	// }

	// pagination helper (default: created_at desc)
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

// DELETE /assessment-urls/:id  (soft-delete bila model di-tag gorm.DeletedAt)
func (ctl *AssessmentUrlsController) Delete(c *fiber.Ctx) error {
	// 🔐 Gate: hanya Admin/DKM/Teacher
	if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// (Opsional) validasi kepemilikan by masjid
	// activeMasjidID, _ := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	// TODO: join ke assessments untuk verifikasi.

	if err := ctl.DB.Delete(&model.AssessmentUrlsModel{}, "assessment_urls_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	return helper.JsonDeleted(c, "Assessment URL dihapus", fiber.Map{"deleted": true})
}
