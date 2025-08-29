// file: internals/features/classes/openings/controller/class_term_opening_controller.go
package controller

import (
	"strings"
	"time"

	openDTO "masjidku_backend/internals/features/school/academics/academic_terms/dto"
	openModel "masjidku_backend/internals/features/school/academics/academic_terms/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Controller
type ClassTermOpeningController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassTermOpeningController(db *gorm.DB) *ClassTermOpeningController {
	return &ClassTermOpeningController{DB: db, Validate: validator.New()}
}

// ============================
// Helpers
// ============================

func parseUUIDParam(c *fiber.Ctx, key string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(c.Params(key)))
}

func toResponse(m openModel.ClassTermOpeningModel) openDTO.ClassTermOpeningResponse {
	// model.UpdatedAt adalah value (non-pointer). DTO pakai *time.Time → alamatkan salinan.
	updated := m.ClassTermOpeningsUpdatedAt
	var deletedPtr *interface{} // dummy to avoid unused if not needed in mapping example
	_ = deletedPtr


	// Kalau di DTO kamu field deleted_at adalah *time.Time, map sesuai struktur DTO yang kamu punya.
	return openDTO.ClassTermOpeningResponse{
		ClassTermOpeningsID:                    m.ClassTermOpeningsID,
		ClassTermOpeningsMasjidID:              m.ClassTermOpeningsMasjidID,
		ClassTermOpeningsClassID:               m.ClassTermOpeningsClassID,
		ClassTermOpeningsTermID:                m.ClassTermOpeningsTermID,
		ClassTermOpeningsIsOpen:                m.ClassTermOpeningsIsOpen,
		ClassTermOpeningsRegistrationOpensAt:   m.ClassTermOpeningsRegistrationOpensAt,
		ClassTermOpeningsRegistrationClosesAt:  m.ClassTermOpeningsRegistrationClosesAt,
		ClassTermOpeningsQuotaTotal:            m.ClassTermOpeningsQuotaTotal,
		ClassTermOpeningsQuotaTaken:            m.ClassTermOpeningsQuotaTaken,
		ClassTermOpeningsFeeOverrideMonthlyIDR: m.ClassTermOpeningsFeeOverrideMonthlyIDR,
		ClassTermOpeningsNotes:                 m.ClassTermOpeningsNotes,
		ClassTermOpeningsCreatedAt:             m.ClassTermOpeningsCreatedAt,
		ClassTermOpeningsUpdatedAt:             &updated,
		// ClassTermOpeningsDeletedAt:           deletedAt, // isi sesuai tipe DTO-mu
	}
}

// Terapkan perubahan dari DTO PATCH/PUT → model (tanpa sentuh updated_at)
func applyUpdate(dst *openModel.ClassTermOpeningModel, req openDTO.UpdateClassTermOpeningRequest) {
	if req.ClassTermOpeningsIsOpen != nil {
		dst.ClassTermOpeningsIsOpen = *req.ClassTermOpeningsIsOpen
	}
	if req.ClassTermOpeningsRegistrationOpensAt != nil {
		dst.ClassTermOpeningsRegistrationOpensAt = req.ClassTermOpeningsRegistrationOpensAt
	}
	if req.ClassTermOpeningsRegistrationClosesAt != nil {
		dst.ClassTermOpeningsRegistrationClosesAt = req.ClassTermOpeningsRegistrationClosesAt
	}
	if req.ClassTermOpeningsQuotaTotal != nil {
		dst.ClassTermOpeningsQuotaTotal = req.ClassTermOpeningsQuotaTotal
	}
	if req.ClassTermOpeningsFeeOverrideMonthlyIDR != nil {
		dst.ClassTermOpeningsFeeOverrideMonthlyIDR = req.ClassTermOpeningsFeeOverrideMonthlyIDR
	}
	if req.ClassTermOpeningsNotes != nil {
		dst.ClassTermOpeningsNotes = req.ClassTermOpeningsNotes
	}
}

// ============================
// CRUD
// ============================

// POST /api/a/class-term-openings
func (ct *ClassTermOpeningController) CreateClassTermOpening(c *fiber.Ctx) error {
	var req openDTO.CreateClassTermOpeningRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body: "+err.Error())
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// hard-tenant
	if req.ClassTermOpeningsMasjidID != uuid.Nil && req.ClassTermOpeningsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "masjid_id di body tidak sesuai dengan token")
	}
	req.ClassTermOpeningsMasjidID = masjidID

	if err := ct.Validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// (Opsional) validasi window friendly — DB sudah ada CHECK juga
	if req.ClassTermOpeningsRegistrationOpensAt != nil &&
		req.ClassTermOpeningsRegistrationClosesAt != nil &&
		req.ClassTermOpeningsRegistrationClosesAt.Before(*req.ClassTermOpeningsRegistrationOpensAt) {
		return fiber.NewError(fiber.StatusBadRequest, "registration_closes_at harus >= registration_opens_at")
	}

	m := openModel.ClassTermOpeningModel{
		ClassTermOpeningsMasjidID:              req.ClassTermOpeningsMasjidID,
		ClassTermOpeningsClassID:               req.ClassTermOpeningsClassID,
		ClassTermOpeningsTermID:                req.ClassTermOpeningsTermID,
		ClassTermOpeningsIsOpen:                true,
		ClassTermOpeningsRegistrationOpensAt:   req.ClassTermOpeningsRegistrationOpensAt,
		ClassTermOpeningsRegistrationClosesAt:  req.ClassTermOpeningsRegistrationClosesAt,
		ClassTermOpeningsQuotaTotal:            req.ClassTermOpeningsQuotaTotal,
		ClassTermOpeningsQuotaTaken:            0,
		ClassTermOpeningsFeeOverrideMonthlyIDR: req.ClassTermOpeningsFeeOverrideMonthlyIDR,
		ClassTermOpeningsNotes:                 req.ClassTermOpeningsNotes,
	}
	if req.ClassTermOpeningsIsOpen != nil {
		m.ClassTermOpeningsIsOpen = *req.ClassTermOpeningsIsOpen
	}

	// Omit updated_at agar tidak mengirim NULL saat insert (defensive)
	if err := ct.DB.Omit("class_term_openings_updated_at").Create(&m).Error; err != nil {
		// bisa kena error dari CHECK kuota/window, dll.
		return fiber.NewError(fiber.StatusInternalServerError, "create failed: "+err.Error())
	}

	return helper.JsonCreated(c, "Opening berhasil dibuat", toResponse(m))
}

// GET /api/a/class-term-openings
// Query: masjid_id, class_id, term_id, is_open, include_deleted, page, limit, sort
func (ct *ClassTermOpeningController) GetAllClassTermOpenings(c *fiber.Ctx) error {
	db := ct.DB.Model(&openModel.ClassTermOpeningModel{})

	// filters
	if v := strings.TrimSpace(c.Query("masjid_id")); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid masjid_id")
		}
		db = db.Where("class_term_openings_masjid_id = ?", id)
	}
	if v := strings.TrimSpace(c.Query("class_id")); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid class_id")
		}
		db = db.Where("class_term_openings_class_id = ?", id)
	}
	if v := strings.TrimSpace(c.Query("term_id")); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid term_id")
		}
		db = db.Where("class_term_openings_term_id = ?", id)
	}
	if v := strings.TrimSpace(c.Query("is_open")); v != "" {
		switch strings.ToLower(v) {
		case "true", "1":
			db = db.Where("class_term_openings_is_open = true")
		case "false", "0":
			db = db.Where("class_term_openings_is_open = false")
		default:
			return fiber.NewError(fiber.StatusBadRequest, "invalid is_open")
		}
	}

	includeDeleted := strings.EqualFold(c.Query("include_deleted"), "true")
	if includeDeleted {
		db = db.Unscoped()
	} else {
		db = db.Where("class_term_openings_deleted_at IS NULL")
	}

	// pagination
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	// count
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "count failed: "+err.Error())
	}

	// sorting (whitelist)
	sortKey := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	switch sortKey {
	case "created_at asc":
		db = db.Order("class_term_openings_created_at ASC")
	case "created_at desc", "":
		db = db.Order("class_term_openings_created_at DESC")
	case "updated_at asc":
		db = db.Order("class_term_openings_updated_at ASC")
	case "updated_at desc":
		db = db.Order("class_term_openings_updated_at DESC")
	default:
		// fallback aman
		db = db.Order("class_term_openings_created_at DESC")
	}

	// data
	var rows []openModel.ClassTermOpeningModel
	if err := db.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	items := make([]openDTO.ClassTermOpeningResponse, 0, len(rows))
	for i := range rows {
		items = append(items, toResponse(rows[i]))
	}

	return helper.JsonList(c, items, fiber.Map{
		"page":  page,
		"limit": limit,
		"total": int(total),
	})
}

// GET /api/a/class-term-openings/:id
func (ct *ClassTermOpeningController) GetClassTermOpeningByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var m openModel.ClassTermOpeningModel
	if err := ct.DB.First(&m, "class_term_openings_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Opening tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}
	return helper.JsonOK(c, "OK", toResponse(m))
}

// PUT /api/a/class-term-openings/:id  (full/partial update)

// PUT /api/a/class-term-openings/:id  (full/partial update)
func (ct *ClassTermOpeningController) UpdateClassTermOpening(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var req openDTO.UpdateClassTermOpeningRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body: "+err.Error())
	}

	// window validation (opsional)
	if req.ClassTermOpeningsRegistrationOpensAt != nil &&
		req.ClassTermOpeningsRegistrationClosesAt != nil &&
		req.ClassTermOpeningsRegistrationClosesAt.Before(*req.ClassTermOpeningsRegistrationOpensAt) {
		return fiber.NewError(fiber.StatusBadRequest, "registration_closes_at harus >= registration_opens_at")
	}

	// Ambil record (scoped tenant) + lock untuk hindari race
	var m openModel.ClassTermOpeningModel
	if err := ct.DB.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "class_term_openings_id = ? AND class_term_openings_masjid_id = ? AND class_term_openings_deleted_at IS NULL",
			id, masjidID,
		).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Opening tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	// Terapkan perubahan ke struct (tanpa sentuh created_at / updated_at)
	applyUpdateOpening(&m, req)

	// Bangun payload update eksplisit agar:
	// - nilai false/0/nil tetap ter-update
	// - updated_at diset via now()
	updates := map[string]any{
		"class_term_openings_is_open":                  m.ClassTermOpeningsIsOpen,
		"class_term_openings_registration_opens_at":    m.ClassTermOpeningsRegistrationOpensAt,
		"class_term_openings_registration_closes_at":   m.ClassTermOpeningsRegistrationClosesAt,
		"class_term_openings_quota_total":              m.ClassTermOpeningsQuotaTotal,
		"class_term_openings_fee_override_monthly_idr": m.ClassTermOpeningsFeeOverrideMonthlyIDR,
		"class_term_openings_notes":                    m.ClassTermOpeningsNotes,
		"class_term_openings_updated_at":               gorm.Expr("now()"),
	}

	if err := ct.DB.
		Model(&openModel.ClassTermOpeningModel{}).
		Where("class_term_openings_id = ? AND class_term_openings_masjid_id = ? AND class_term_openings_deleted_at IS NULL",
			m.ClassTermOpeningsID, masjidID,
		).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "update failed: "+err.Error())
	}

	// refresh entity untuk response (biar updated_at terambil dari DB)
	if err := ct.DB.First(&m, "class_term_openings_id = ?", m.ClassTermOpeningsID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "reload failed: "+err.Error())
	}

	return helper.JsonUpdated(c, "Opening diperbarui", toResponse(m))
}

// --- ganti applyUpdate lama (yang set pointer updated_at) menjadi ini:
func applyUpdateOpening(dst *openModel.ClassTermOpeningModel, req openDTO.UpdateClassTermOpeningRequest) {
	if req.ClassTermOpeningsIsOpen != nil {
		dst.ClassTermOpeningsIsOpen = *req.ClassTermOpeningsIsOpen
	}
	if req.ClassTermOpeningsRegistrationOpensAt != nil {
		dst.ClassTermOpeningsRegistrationOpensAt = req.ClassTermOpeningsRegistrationOpensAt
	}
	if req.ClassTermOpeningsRegistrationClosesAt != nil {
		dst.ClassTermOpeningsRegistrationClosesAt = req.ClassTermOpeningsRegistrationClosesAt
	}
	if req.ClassTermOpeningsQuotaTotal != nil {
		dst.ClassTermOpeningsQuotaTotal = req.ClassTermOpeningsQuotaTotal
	}
	if req.ClassTermOpeningsFeeOverrideMonthlyIDR != nil {
		dst.ClassTermOpeningsFeeOverrideMonthlyIDR = req.ClassTermOpeningsFeeOverrideMonthlyIDR
	}
	if req.ClassTermOpeningsNotes != nil {
		dst.ClassTermOpeningsNotes = req.ClassTermOpeningsNotes
	}
	// updated_at tidak disentuh di sini — diset di query via now()
	_ = time.Now() // (hindari unused import kalau kamu hapus time)
}

// DELETE /api/a/class-term-openings/:id  (soft delete)
func (ct *ClassTermOpeningController) DeleteClassTermOpening(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var m openModel.ClassTermOpeningModel
	if err := ct.DB.First(&m,
		"class_term_openings_id = ? AND class_term_openings_masjid_id = ?",
		id, masjidID,
	).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Opening tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	// Soft delete (gorm.DeletedAt akan terisi otomatis)
	if err := ct.DB.Delete(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "delete failed: "+err.Error())
	}

	return helper.JsonDeleted(c, "Opening dihapus", fiber.Map{"class_term_openings_id": m.ClassTermOpeningsID})
}

// Optional: RESTORE
// POST /api/a/class-term-openings/:id/restore
func (ct *ClassTermOpeningController) RestoreClassTermOpening(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	// Unscoped untuk bisa menemukan row yang sudah soft-deleted
	var m openModel.ClassTermOpeningModel
	if err := ct.DB.Unscoped().First(&m, "class_term_openings_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Opening tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "query failed: "+err.Error())
	}

	// Clear deleted_at → restore
	if err := ct.DB.Unscoped().
		Model(&openModel.ClassTermOpeningModel{}).
		Where("class_term_openings_id = ?", id).
		Update("class_term_openings_deleted_at", nil).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "restore failed: "+err.Error())
	}

	// Ambil ulang (scoped) untuk response
	if err := ct.DB.First(&m, "class_term_openings_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "reload failed: "+err.Error())
	}
	return helper.JsonOK(c, "Opening dipulihkan", toResponse(m))
}
