// file: internals/features/billings/general_billing_kinds/controller/general_billing_kind_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "schoolku_backend/internals/features/finance/general_billings/dto"
	m "schoolku_backend/internals/features/finance/general_billings/model"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* =========================
   Controller
========================= */

type GeneralBillingKindController struct {
	DB *gorm.DB
}

func NewGeneralBillingKindController(db *gorm.DB) *GeneralBillingKindController {
	return &GeneralBillingKindController{DB: db}
}

/* =========================
   Utils
========================= */

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(s))
}

// isUniqueViolation tanpa driver-specific deps
func isUniqueViolation(err error, constraint string) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	is23505 := strings.Contains(msg, "23505") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint")
	if !is23505 {
		return false
	}
	if constraint == "" {
		return true
	}
	return strings.Contains(msg, strings.ToLower(constraint))
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}

// helper: set flags by category (mirror CHECK constraint) — AMAN utk pointer types
// ========================
// ONE HELPER for Create & Patch (pointer-safe)
// ========================
func normalizeGBKByCategory(
	catPtr **string, // pointer ke *string (req.Category & patch.Category)
	isRecurring **bool, // pointer ke *bool (req.IsRecurring & patch.IsRecurring)
	reqMonthYear **bool, // pointer ke *bool
	reqOptionCode **bool, // pointer ke *bool
	currentCat string, // kategori existing (untuk Patch), "" jika Create
) {
	// pilih kategori efektif: req.Category > currentCat > "mass_student"
	eff := ""
	if catPtr != nil && *catPtr != nil && strings.TrimSpace(**catPtr) != "" {
		eff = strings.ToLower(strings.TrimSpace(**catPtr))
	} else if strings.TrimSpace(currentCat) != "" {
		eff = strings.ToLower(strings.TrimSpace(currentCat))
	} else {
		eff = "mass_student"
	}

	// jika req.Category kosong, set pointer-nya ke eff
	if catPtr != nil && *catPtr == nil {
		v := eff
		*catPtr = &v
	} else if catPtr != nil && *catPtr != nil {
		**catPtr = eff
	}

	// setter aman utk *bool
	setBool := func(dst **bool, v bool) {
		if *dst == nil {
			b := v
			*dst = &b
		} else {
			**dst = v
		}
	}

	// mirror ke aturan CHECK ck_gbk_flags_match_category
	switch eff {
	case "registration":
		setBool(isRecurring, false)
		setBool(reqMonthYear, false)
		setBool(reqOptionCode, false)
	case "spp":
		setBool(isRecurring, true)
		setBool(reqMonthYear, true)
		setBool(reqOptionCode, false)
	case "mass_student":
		setBool(isRecurring, false)
		setBool(reqMonthYear, false)
		setBool(reqOptionCode, true)
	case "donation":
		setBool(isRecurring, false)
		setBool(reqMonthYear, false)
		setBool(reqOptionCode, false)
	default:
		// fallback selalu valid
		v := "mass_student"
		if catPtr != nil {
			*catPtr = &v
		}
		setBool(isRecurring, false)
		setBool(reqMonthYear, false)
		setBool(reqOptionCode, true)
	}
}

/*
	=========================
	  Create
	  POST /api/a/:school_id/general-billing-kinds

=========================
*/
func (ctl *GeneralBillingKindController) Create(c *fiber.Ctx) error {
	// 1) school guard via token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); er != nil {
		return er
	}
	c.Locals("__school_guard_ok", schoolID.String())

	// 2) Body
	var req dto.CreateGeneralBillingKindRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// paksa tenant dari path
	req.SchoolID = &schoolID

	// minimal validation
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Code == "" || req.Name == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "code and name are required")
	}

	// ⬅️ normalize flags berdasar category (abaikan nilai flag yang dikirim client)
	// setelah validasi minimal, sebelum ToModel()
	normalizeGBKByCategory(
		&req.Category,
		&req.IsRecurring,
		&req.RequiresMonthYear,
		&req.RequiresOptionCode,
		"", // currentCat kosong utk Create
	)
	// 3) Persist
	rec := req.ToModel()
	if err := ctl.DB.WithContext(c.Context()).Create(&rec).Error; err != nil {
		if isUniqueViolation(err, "uq_gbk_code_per_tenant_alive") {
			return helper.JsonError(c, fiber.StatusConflict, "code already exists for this tenant (alive)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "created", dto.FromModel(rec))
}

/*
	=========================
	  Patch (Update)
	  PATCH /api/a/:school_id/general-billing-kinds/:id

=========================
*/
func (ctl *GeneralBillingKindController) Patch(c *fiber.Ctx) error {
	// 1) school guard via token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); er != nil {
		return er
	}
	c.Locals("__school_guard_ok", schoolID.String())

	idStr := c.Params("id")
	id, err := parseUUID(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid UUID in path")
	}

	// 2) Body
	var req dto.PatchGeneralBillingKindRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 3) Load tenant-safe (punya school ini & belum terhapus)
	var rec m.GeneralBillingKind
	tx := ctl.DB.WithContext(c.Context()).
		Where(`
			general_billing_kind_id = ?
			AND general_billing_kind_school_id = ?
			AND general_billing_kind_deleted_at IS NULL
		`, id, schoolID).
		First(&rec)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return helper.JsonError(c, fiber.StatusNotFound, "record not found or already deleted")
	}
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}

	// 4) Normalize flags berdasar kategori efektif (request > current)
	normalizeGBKByCategory(
		&req.Category,
		&req.IsRecurring,
		&req.RequiresMonthYear,
		&req.RequiresOptionCode,
		string(rec.GeneralBillingKindCategory), // currentCat
	)

	// 5) Apply + Save
	req.ApplyTo(&rec)
	rec.GeneralBillingKindUpdatedAt = time.Now()

	if err := ctl.DB.WithContext(c.Context()).Save(&rec).Error; err != nil {
		if isUniqueViolation(err, "uq_gbk_code_per_tenant_alive") {
			return helper.JsonError(c, fiber.StatusConflict, "code already exists for this tenant (alive)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "updated", dto.FromModel(rec))
}

/* =========================
   Delete (Soft)
   DELETE /api/a/:school_id/general-billing-kinds/:id
========================= */

func (ctl *GeneralBillingKindController) Delete(c *fiber.Ctx) error {
	// 1) school guard via token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); er != nil {
		return er
	}
	c.Locals("__school_guard_ok", schoolID.String())

	idStr := c.Params("id")
	id, err := parseUUID(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid UUID in path")
	}

	// 2) Soft delete (hanya record milik school ini)
	res := ctl.DB.WithContext(c.Context()).
		Model(&m.GeneralBillingKind{}).
		Where(`
			general_billing_kind_id = ?
			AND general_billing_kind_school_id = ?
			AND general_billing_kind_deleted_at IS NULL
		`, id, schoolID).
		Updates(map[string]any{
			"general_billing_kind_deleted_at": nowPtr(),
			"general_billing_kind_updated_at": time.Now(),
		})

	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "record not found or already deleted")
	}

	return helper.JsonDeleted(c, "deleted", fiber.Map{"id": id})
}
