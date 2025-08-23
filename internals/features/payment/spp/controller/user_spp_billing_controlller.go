package controller

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/payment/spp/dto"
	model "masjidku_backend/internals/features/payment/spp/model"
	helper "masjidku_backend/internals/helpers"
)

type UserSppBillingItemController struct {
	DB *gorm.DB
}

func NewUserSppBillingItemController(db *gorm.DB) *UserSppBillingItemController {
	return &UserSppBillingItemController{DB: db}
}

/* ======================= HELPERS ======================= */

func (h *UserSppBillingItemController) ensureBillingBelongsToMasjid(billingID, masjidID uuid.UUID) error {
	type row struct{ MasjidID *uuid.UUID `gorm:"column:spp_billing_masjid_id"` }
	var r row
	err := h.DB.Table("spp_billings").
		Select("spp_billing_masjid_id").
		Where("spp_billing_id = ?", billingID).
		Take(&r).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Billing SPP tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// Wajib match tenant
	if r.MasjidID == nil || *r.MasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Billing SPP tidak milik masjid ini")
	}
	return nil
}

// muat satu item by id + tenant safety via join ke header
func (h *UserSppBillingItemController) loadItemByIDTenant(id, masjidID uuid.UUID) (*model.UserSppBillingModel, error) {
	var m model.UserSppBillingModel
	q := h.DB.Table("user_spp_billings AS u").
		Joins("JOIN spp_billings AS b ON b.spp_billing_id = u.user_spp_billing_billing_id").
		Where("u.user_spp_billing_id = ? AND b.spp_billing_masjid_id = ?", id, masjidID).
		Select("u.*")
	if err := q.Take(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Item SPP tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return &m, nil
}



/* ======================== LIST ======================== */
// GET /admin/user-spp-billings?billing_id=...&status=...&user_id=...&limit=...&offset=...
func (h *UserSppBillingItemController) List(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	var q dto.ListUserSppBillingQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit == 0 { q.Limit = 20 }
	if q.Offset < 0 { q.Offset = 0 }

	v := validator.New()
	if err := v.Struct(q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Pastikan header billing milik tenant
	if err := h.ensureBillingBelongsToMasjid(q.BillingID, masjidID); err != nil {
		return err
	}

	db := h.DB.Table("user_spp_billings AS u").
		Joins("JOIN spp_billings AS b ON b.spp_billing_id = u.user_spp_billing_billing_id").
		Where("b.spp_billing_masjid_id = ? AND u.user_spp_billing_billing_id = ?", masjidID, q.BillingID).
		Select("u.*")

	if q.UserID != nil {
		db = db.Where("u.user_spp_billing_user_id = ?", q.UserID)
	}
	if q.Status != nil {
		db = db.Where("u.user_spp_billing_status = ?", *q.Status)
	}
	if q.MinAmount != nil {
		db = db.Where("u.user_spp_billing_amount_idr >= ?", *q.MinAmount)
	}
	if q.MaxAmount != nil {
		db = db.Where("u.user_spp_billing_amount_idr <= ?", *q.MaxAmount)
	}
	if q.PaidFrom != nil {
		db = db.Where("u.user_spp_billing_paid_at >= ?", *q.PaidFrom)
	}
	if q.PaidTo != nil {
		db = db.Where("u.user_spp_billing_paid_at <= ?", *q.PaidTo)
	}
	// (opsional) cari di note
	if q.Q != nil && len(*q.Q) > 0 {
		db = db.Where("u.user_spp_billing_note ILIKE ?", "%"+*q.Q+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.UserSppBillingModel
	if err := db.
		Order("u.user_spp_billing_created_at DESC").
		Limit(q.Limit).Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", dto.FromUserSppBillingModels(rows, total))
}

// GET /admin/user-spp-billings/me
// GET /admin/user-spp-billings/me
func (h *UserSppBillingItemController) ListMine(c *fiber.Ctx) error {
	// masjid dari helper yang sudah ada
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// user_id langsung dari Locals (tanpa helper baru)
	uidRaw := c.Locals("user_id")
	uidStr, ok := uidRaw.(string)
	if !ok || uidStr == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "user_id tidak ditemukan di token")
	}
	userID, err := uuid.Parse(uidStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "user_id pada token tidak valid")
	}

	var q dto.ListMySppBillingQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit == 0 {
		q.Limit = 20
	}

	tx := h.DB.Table("user_spp_billings AS u").
		Joins("JOIN spp_billings AS b ON b.spp_billing_id = u.user_spp_billing_billing_id").
		Where("u.user_spp_billing_user_id = ?", userID).
		Where("b.spp_billing_masjid_id = ?", masjidID)

	if q.Status != nil && *q.Status != "" {
		tx = tx.Where("u.user_spp_billing_status = ?", *q.Status)
	}
	if q.Month != nil {
		tx = tx.Where("b.spp_billing_month = ?", *q.Month)
	}
	if q.Year != nil {
		tx = tx.Where("b.spp_billing_year = ?", *q.Year)
	}
	if q.Q != nil && *q.Q != "" {
		tx = tx.Where("b.spp_billing_title ILIKE ?", "%"+*q.Q+"%")
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var rows []dto.MySppBillingItem
	if err := tx.
		Select(`
			u.user_spp_billing_id,
			u.user_spp_billing_billing_id,
			u.user_spp_billing_amount_idr,
			u.user_spp_billing_status,
			u.user_spp_billing_paid_at,
			b.spp_billing_title    AS billing_title,
			b.spp_billing_month    AS billing_month,
			b.spp_billing_year     AS billing_year,
			b.spp_billing_due_date AS billing_due_date,
			b.spp_billing_class_id AS billing_class_id
		`).
		Order("b.spp_billing_year DESC, b.spp_billing_month DESC, u.user_spp_billing_created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Scan(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", dto.MySppBillingListResponse{
		Items: rows,
		Total: total,
	})
}



/* ====================== GET BY ID ====================== */
// GET /admin/user-spp-billings/:id
func (h *UserSppBillingItemController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.loadItemByIDTenant(id, masjidID)
	if err != nil { return err }

	return helper.JsonOK(c, "OK", dto.FromUserSppBillingModel(*m))
}

/* ======================== UPDATE (PUT, partial) ======================== */
// PUT /admin/user-spp-billings/:id
func (h *UserSppBillingItemController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	id, err := uuid.Parse(c.Params("id"))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	// Pastikan item ada & milik tenant
	curr, err := h.loadItemByIDTenant(id, masjidID)
	if err != nil { return err }

	var req dto.UpdateUserSppBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	patch := map[string]interface{}{}

	if req.UserSppBillingUserID != nil {
		patch["user_spp_billing_user_id"] = *req.UserSppBillingUserID
	}
	if req.UserSppBillingAmountIDR != nil {
		patch["user_spp_billing_amount_idr"] = *req.UserSppBillingAmountIDR
	}
	if req.UserSppBillingStatus != nil {
		patch["user_spp_billing_status"] = *req.UserSppBillingStatus

		// jika status menjadi 'paid' dan paid_at tidak dikirim → isi sekarang
		if *req.UserSppBillingStatus == model.SppPaid {
			if req.UserSppBillingPaidAt != nil {
				patch["user_spp_billing_paid_at"] = *req.UserSppBillingPaidAt
			} else if curr.UserSppBillingPaidAt == nil {
				now := time.Now()
				patch["user_spp_billing_paid_at"] = now
			}
		}
	}

	// paid_at dikirim eksplisit (override)
	if req.UserSppBillingPaidAt != nil {
		patch["user_spp_billing_paid_at"] = *req.UserSppBillingPaidAt
	}

	if req.UserSppBillingNote != nil {
		patch["user_spp_billing_note"] = *req.UserSppBillingNote
	}

	// tidak ada perubahan
	if len(patch) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", dto.FromUserSppBillingModel(*curr))
	}

	// update (trigger BEFORE UPDATE akan set user_spp_billing_updated_at)
	if err := h.DB.Model(&model.UserSppBillingModel{}).
		Where("user_spp_billing_id = ?", id).
		Updates(patch).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui item SPP")
	}

	// muat ulang nilai terbaru
	updated, err := h.loadItemByIDTenant(id, masjidID)
	if err != nil { // fallback kalau gagal reload
		return helper.JsonOK(c, "Item SPP berhasil diperbarui", dto.FromUserSppBillingModel(*curr))
	}
	return helper.JsonOK(c, "Item SPP berhasil diperbarui", dto.FromUserSppBillingModel(*updated))
}



/* ======================== DELETE ======================= */
// DELETE /admin/user-spp-billings/:id
func (h *UserSppBillingItemController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	id, err := uuid.Parse(c.Params("id"))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	m, err := h.loadItemByIDTenant(id, masjidID)
	if err != nil { return err }

	if err := h.DB.Delete(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus item SPP")
	}

	return helper.JsonOK(c, "Item SPP berhasil dihapus", nil)
}
