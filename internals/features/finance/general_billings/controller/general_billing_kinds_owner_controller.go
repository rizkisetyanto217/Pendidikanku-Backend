// file: internals/features/finance/general_billings/controller/general_billing_kind_controller.go
package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/finance/general_billings/dto"
	m "madinahsalam_backend/internals/features/finance/general_billings/model"
	helper "madinahsalam_backend/internals/helpers"
)

/* ======================
   Controller
====================== */

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

	// Paksa GLOBAL
	req.SchoolID = nil
	trueVal := true
	req.IsGlobal = &trueVal

	rec := req.ToModel()

	// Default category bila kosong → "donation" (pengganti "campaign" sebelumnya)
	if string(rec.GeneralBillingKindCategory) == "" {
		rec.GeneralBillingKindCategory = m.GeneralBillingKindCategory("donation")
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
		Where("general_billing_kind_id = ? AND general_billing_kind_school_id IS NULL AND general_billing_kind_deleted_at IS NULL", id).
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
		Where("general_billing_kind_id = ? AND general_billing_kind_school_id IS NULL AND general_billing_kind_deleted_at IS NULL", id).
		Update("general_billing_kind_deleted_at", gorm.Expr("NOW()"))
	if q.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, q.Error.Error())
	}
	if q.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "not found")
	}
	return helper.JsonOK(c, "deleted", nil)
}

// GET /admin/general-billing-kinds?search=&category=&page=&per_page=
func (ctl *GeneralBillingKindController) ListGlobal(c *fiber.Ctx) error {
	// Paging
	page := clampInt(parseInt(c.Query("page"), 1), 1, 1_000_000)
	perPage := clampInt(parseInt(c.Query("per_page"), 50), 1, 200)
	offset := (page - 1) * perPage

	// Filters
	search := strings.TrimSpace(c.Query("search"))
	category := strings.TrimSpace(strings.ToLower(c.Query("category")))
	// defaultkan ke "donation" agar setara “campaign” lama, tapi boleh override via query
	if category == "" {
		category = "donation"
	}

	q := ctl.DB.WithContext(c.Context()).
		Model(&m.GeneralBillingKind{}).
		Where(`
			general_billing_kind_school_id IS NULL
			AND general_billing_kind_deleted_at IS NULL
			AND general_billing_kind_is_global = TRUE
			AND general_billing_kind_category = ?
		`, category)

	if search != "" {
		needle := "%" + search + "%"
		q = q.Where(
			"(unaccent(general_billing_kind_code) ILIKE unaccent(?) OR unaccent(general_billing_kind_name) ILIKE unaccent(?))",
			needle, needle,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var items []m.GeneralBillingKind
	if err := q.Order("general_billing_kind_created_at DESC").Limit(perPage).Offset(offset).Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return jsonList(c, "ok", dto.FromModelSlice(items), page, perPage, total)
}

// GET /admin/general-billing-kinds/:id
func (ctl *GeneralBillingKindController) GetGlobalByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var rec m.GeneralBillingKind
	tx := ctl.DB.WithContext(c.Context()).
		Where("general_billing_kind_id = ? AND general_billing_kind_school_id IS NULL AND general_billing_kind_deleted_at IS NULL", id).
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

// GET /public/general-billing-kinds?search=&category=&page=&per_page=
func (ctl *GeneralBillingKindController) ListPublic(c *fiber.Ctx) error {
	// Paging
	page := clampInt(parseInt(c.Query("page"), 1), 1, 1_000_000)
	perPage := clampInt(parseInt(c.Query("per_page"), 50), 1, 200)
	offset := (page - 1) * perPage

	// Filters
	search := strings.TrimSpace(c.Query("search"))
	category := strings.TrimSpace(strings.ToLower(c.Query("category"))) // optional

	q := ctl.DB.WithContext(c.Context()).
		Model(&m.GeneralBillingKind{}).
		Where(`
			general_billing_kind_school_id IS NULL
			AND general_billing_kind_deleted_at IS NULL
			AND general_billing_kind_is_active = TRUE
			AND general_billing_kind_visibility = ?
		`, m.GBKVisibilityPublic)

	if category != "" {
		q = q.Where("general_billing_kind_category = ?", category)
	}
	if search != "" {
		needle := "%" + strings.ToLower(search) + "%"
		q = q.Where("LOWER(general_billing_kind_code) LIKE ? OR LOWER(general_billing_kind_name) LIKE ?", needle, needle)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var items []m.GeneralBillingKind
	if err := q.Order("general_billing_kind_created_at DESC").Limit(perPage).Offset(offset).Find(&items).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return jsonList(c, "ok", dto.FromModelSlice(items), page, perPage, total)
}

// GET /public/general-billing-kinds/:id
func (ctl *GeneralBillingKindController) GetPublicByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var rec m.GeneralBillingKind
	tx := ctl.DB.WithContext(c.Context()).
		Where(`
			general_billing_kind_id = ?
			AND general_billing_kind_school_id IS NULL
			AND general_billing_kind_deleted_at IS NULL
			AND general_billing_kind_is_active = TRUE
			AND general_billing_kind_visibility = ?
		`, id, m.GBKVisibilityPublic).
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
   Helpers (local, ringan)
====================== */

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	var (
		n   int
		err error
	)
	// simple atoi tanpa import strconv? tetap pakai strconv:
	// (biar pasti) — tapi file ini tidak import strconv; jadi pakai time.ParseDuration trick tidak cocok.
	// Tambah import strconv di header file kalau IDE protes.
	return func() int {
		// shadow import local
		// NOTE: tambahkan "strconv" di import utama file.
		n, err = strconv.Atoi(s)
		if err != nil {
			return def
		}
		return n
	}()
}

// jsonList: balikan list dengan envelope message + data + pagination
func jsonList(c *fiber.Ctx, message string, data any, page, perPage int, total int64) error {
	totalPages := (total + int64(perPage) - 1) / int64(perPage)
	pagination := fiber.Map{
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    int64(page) < totalPages,
		"has_prev":    page > 1,
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":    message,
		"data":       data,
		"pagination": pagination,
		"timestamp":  time.Now().UTC(),
	})
}
