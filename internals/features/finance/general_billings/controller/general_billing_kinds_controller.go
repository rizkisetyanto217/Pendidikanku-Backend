// file: internals/features/billings/general_billing_kinds/controller/general_billing_kind_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/finance/general_billings/dto"
	m "masjidku_backend/internals/features/finance/general_billings/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
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

/* =========================
   Create
   POST /api/a/:masjid_id/general-billing-kinds
========================= */

func (ctl *GeneralBillingKindController) Create(c *fiber.Ctx) error {
	// 1) Ambil masjid dari path + guard
	masjidID, err := helperAuth.ParseMasjidIDFromPath(c)
	if err != nil {
		return err // helper sudah balikin JSON error
	}
	if er := helperAuth.EnsureDKMOrTeacherMasjid(c, masjidID); er != nil {
		return er
	}
	c.Locals("__masjid_guard_ok", masjidID.String())

	// 2) Body
	var req dto.CreateGeneralBillingKindRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Paksa tenant dari path (endpoint ini khusus per-masjid)
	req.MasjidID = &masjidID

	// Validasi minimal
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	if req.Code == "" || req.Name == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "code and name are required")
	}

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

/* =========================
   Patch (Update)
   PATCH /api/a/:masjid_id/general-billing-kinds/:id
========================= */

func (ctl *GeneralBillingKindController) Patch(c *fiber.Ctx) error {
	// 1) Path + guard
	masjidID, err := helperAuth.ParseMasjidIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureDKMOrTeacherMasjid(c, masjidID); er != nil {
		return er
	}
	c.Locals("__masjid_guard_ok", masjidID.String())

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

	// 3) Load tenant-safe (hanya milik masjid ini & belum terhapus)
	var rec m.GeneralBillingKind
	tx := ctl.DB.WithContext(c.Context()).
		Where(`
			general_billing_kind_id = ?
			AND general_billing_kind_masjid_id = ?
			AND general_billing_kind_deleted_at IS NULL
		`, id, masjidID).
		First(&rec)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return helper.JsonError(c, fiber.StatusNotFound, "record not found or already deleted")
	}
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}

	// 4) Apply + Save
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
   DELETE /api/a/:masjid_id/general-billing-kinds/:id
========================= */

func (ctl *GeneralBillingKindController) Delete(c *fiber.Ctx) error {
	// 1) Path + guard
	masjidID, err := helperAuth.ParseMasjidIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureDKMOrTeacherMasjid(c, masjidID); er != nil {
		return er
	}
	c.Locals("__masjid_guard_ok", masjidID.String())

	idStr := c.Params("id")
	id, err := parseUUID(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid UUID in path")
	}

	// 2) Soft delete (hanya record milik masjid ini)
	res := ctl.DB.WithContext(c.Context()).
		Model(&m.GeneralBillingKind{}).
		Where(`
			general_billing_kind_id = ?
			AND general_billing_kind_masjid_id = ?
			AND general_billing_kind_deleted_at IS NULL
		`, id, masjidID).
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
