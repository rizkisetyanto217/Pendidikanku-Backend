// file: internals/features/finance/general_billings/controller/general_billing_kind_controller.go
package controller

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/finance/general_billings/dto"
	m "masjidku_backend/internals/features/finance/general_billings/model"
	helper "masjidku_backend/internals/helpers" // sesuaikan helper JSON response
	// sesuaikan guard/ACL
)

/* ======================
   GLOBAL ADMIN endpoints
   base path: /admin/general-billing-kinds
====================== */

// POST /admin/general-billing-kinds
func (ctl *GeneralBillingKindController) CreateGlobal(c *fiber.Ctx) error {

	var req dto.CreateGeneralBillingKindRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Code == "" || req.Name == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "code and name are required")
	}

	// Paksa GLOBAL: masjid_id = NULL, is_global = true
	req.MasjidID = nil
	trueVal := true
	req.IsGlobal = &trueVal

	rec := req.ToModel()
	// Pastikan category/visibility sesuai kebutuhan global campaign (opsional)
	// contoh default category campaign bila kosong:
	if rec.GeneralBillingKindCategory == "" {
		rec.GeneralBillingKindCategory = m.GBKCategoryCampaign
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&rec).Error; err != nil {
		if isUniqueViolation(err, "uq_gbk_code_global_alive") {
			return helper.JsonError(c, fiber.StatusConflict, "code already exists (global, alive)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "created", dto.FromModel(rec))
}

// PATCH /admin/general-billing-kinds/:id
func (ctl *GeneralBillingKindController) PatchGlobal(c *fiber.Ctx) error {

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var body dto.PatchGeneralBillingKindRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var rec m.GeneralBillingKind
	tx := ctl.DB.WithContext(c.Context()).
		Where("general_billing_kind_id = ? AND general_billing_kind_masjid_id IS NULL AND general_billing_kind_deleted_at IS NULL", id).
		First(&rec)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}

	body.ApplyTo(&rec)

	if err := ctl.DB.WithContext(c.Context()).Save(&rec).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "updated", dto.FromModel(rec))
}

// DELETE /admin/general-billing-kinds/:id
func (ctl *GeneralBillingKindController) DeleteGlobal(c *fiber.Ctx) error {

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Soft delete
	q := ctl.DB.WithContext(c.Context()).
		Model(&m.GeneralBillingKind{}).
		Where("general_billing_kind_id = ? AND general_billing_kind_masjid_id IS NULL AND general_billing_kind_deleted_at IS NULL", id).
		Update("general_billing_kind_deleted_at", gorm.Expr("NOW()"))
	if q.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, q.Error.Error())
	}
	if q.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "not found")
	}
	return helper.JsonOK(c, "deleted", nil)
}

// GET /admin/general-billing-kinds
func (ctl *GeneralBillingKindController) ListGlobal(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("search")) // ?search=...
	needle := "%" + q + "%"

	var items []m.GeneralBillingKind
	tx := ctl.DB.WithContext(c.Context()).
		Where(`
            general_billing_kind_masjid_id IS NULL
            AND general_billing_kind_deleted_at IS NULL
            AND general_billing_kind_category = ?
            AND general_billing_kind_is_global = TRUE
        `, "campaign")

	// Search by code/name (pilih salah satu blok: A atau B)
	if q != "" {
		// A) Simple ILIKE (tanpa unaccent)
		// tx = tx.Where(
		//     "(general_billing_kind_code ILIKE ? OR general_billing_kind_name ILIKE ?)",
		//     needle, needle,
		// )

		// B) ILIKE + unaccent (aktifkan extension unaccent; sudah ada di migrasi)
		tx = tx.Where(
			"(unaccent(general_billing_kind_code) ILIKE unaccent(?) OR unaccent(general_billing_kind_name) ILIKE unaccent(?))",
			needle, needle,
		)
	}

	tx = tx.Order("general_billing_kind_created_at DESC")

	if err := tx.Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// DTO pakai `omitempty` → masjid_id tidak muncul bila NULL
	return helper.JsonOK(c, "ok", dto.FromModelSlice(items))
}

// GET /admin/general-billing-kinds/:id
func (ctl *GeneralBillingKindController) GetGlobalByID(c *fiber.Ctx) error {

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var rec m.GeneralBillingKind
	tx := ctl.DB.WithContext(c.Context()).
		Where("general_billing_kind_id = ? AND general_billing_kind_masjid_id IS NULL AND general_billing_kind_deleted_at IS NULL", id).
		First(&rec)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	return helper.JsonOK(c, "ok", dto.FromModel(rec))
}

/* ======================
   PUBLIC read-only endpoints
   base path: /public/general-billing-kinds
====================== */

// GET /public/general-billing-kinds?search=&category=campaign
func (ctl *GeneralBillingKindController) ListPublic(c *fiber.Ctx) error {
	// hanya tampilkan global + visibility=public + active
	var items []m.GeneralBillingKind
	q := ctl.DB.WithContext(c.Context()).
		Where("general_billing_kind_masjid_id IS NULL AND general_billing_kind_deleted_at IS NULL").
		Where("general_billing_kind_visibility = ? AND general_billing_kind_is_active = TRUE", m.GBKVisibilityPublic)

	// optional filter cat=campaign
	if cat := strings.TrimSpace(c.Query("category")); cat != "" {
		q = q.Where("general_billing_kind_category = ?", cat)
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		ss := "%" + strings.ToLower(s) + "%"
		q = q.Where("LOWER(general_billing_kind_code) LIKE ? OR LOWER(general_billing_kind_name) LIKE ?", ss, ss)
	}

	if err := q.Order("general_billing_kind_created_at DESC").Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "ok", dto.FromModelSlice(items))
}

// GET /public/general-billing-kinds/:id
func (ctl *GeneralBillingKindController) GetPublicByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var rec m.GeneralBillingKind
	tx := ctl.DB.WithContext(c.Context()).
		Where("general_billing_kind_id = ? AND general_billing_kind_masjid_id IS NULL AND general_billing_kind_deleted_at IS NULL", id).
		Where("general_billing_kind_visibility = ? AND general_billing_kind_is_active = TRUE", m.GBKVisibilityPublic).
		First(&rec)
	if tx.Error != nil {
		if tx.Error == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	return helper.JsonOK(c, "ok", dto.FromModel(rec))
}
