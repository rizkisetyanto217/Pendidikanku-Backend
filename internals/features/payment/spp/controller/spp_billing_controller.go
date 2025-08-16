package controller

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/payment/spp/dto"
	model "masjidku_backend/internals/features/payment/spp/model"
	helper "masjidku_backend/internals/helpers"
)

type SppBillingController struct {
	DB *gorm.DB
}

func NewSppBillingController(db *gorm.DB) *SppBillingController {
	return &SppBillingController{DB: db}
}

/* ======================= CREATE ======================= */
// POST /admin/spp/billings
func (h *SppBillingController) Create(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateSppBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Wajibkan tenant dari token
	req.SppBillingMasjidID = &masjidID

	// Wajibkan class_id kalau mau auto-generate item per siswa
	if req.SppBillingClassID == nil || *req.SppBillingClassID == uuid.Nil {
		return fiber.NewError(fiber.StatusBadRequest, "spp_billing_class_id wajib diisi untuk generate item per siswa")
	}

	v := validator.New()
	if err := v.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// === TX START ===
	tx := h.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Insert header
	m := req.ToModel()
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat SPP billing")
	}

	// Pastikan ID header terisi (Postgres biasanya RETURNING otomatis oleh GORM)
	if m.SppBillingID == uuid.Nil {
		// fallback: ambil ulang by unique key (optional)
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperoleh ID billing")
	}

	// Bulk-generate item per siswa aktif di kelas tsb (tenant-safe)
	// - user_classes_status = 'active' dan ended_at IS NULL
	// - amount = override jika ada, else class_fee_monthly_idr, else 0
	res := tx.Exec(`
		INSERT INTO user_spp_billings
			(user_spp_billing_billing_id, user_spp_billing_user_id, user_spp_billing_amount_idr)
		SELECT
			?, uc.user_classes_user_id,
			COALESCE(uc.user_classes_fee_override_monthly_idr, c.class_fee_monthly_idr, 0) AS amount_idr
		FROM user_classes uc
		JOIN classes c
		  ON c.class_id = uc.user_classes_class_id
		WHERE c.class_id = ?
		  AND c.class_masjid_id = ?
		  AND uc.user_classes_status = 'active'
		  AND uc.user_classes_ended_at IS NULL
		ON CONFLICT (user_spp_billing_billing_id, user_spp_billing_user_id) DO NOTHING
	`, m.SppBillingID, *req.SppBillingClassID, masjidID)
	if res.Error != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal generate item SPP per siswa: "+res.Error.Error())
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// === TX END ===

	// (opsional) kalau mau info berapa item yang terbuat:
	// createdN := res.RowsAffected

	return helper.JsonCreated(c, "SPP billing berhasil dibuat & item per siswa digenerate", dto.FromModel(*m))
}


/* ======================== GET BY ID ======================== */
// GET /admin/spp/billings/:id
func (h *SppBillingController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	idStr := c.Params("id")
	if idStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	var row model.SppBillingModel
	// Batasi ke tenant (hanya baris yang punya masjid_id = token)
	if err := h.DB.
		Where("spp_billing_id = ? AND spp_billing_masjid_id = ?", idStr, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", dto.FromModel(row))
}

/* ======================== LIST ======================== */
// GET /admin/spp/billings?class_id=&month=&year=&due_from=&due_to=&q=&limit=&offset=
func (h *SppBillingController) List(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q dto.ListSppBillingQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	// Defaults
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	base := h.DB.Model(&model.SppBillingModel{}).
		Where("spp_billing_masjid_id = ?", masjidID)

	if q.ClassID != nil {
		base = base.Where("spp_billing_class_id = ?", *q.ClassID)
	}
	if q.Month != nil {
		base = base.Where("spp_billing_month = ?", *q.Month)
	}
	if q.Year != nil {
		base = base.Where("spp_billing_year = ?", *q.Year)
	}
	if q.DueFrom != nil {
		base = base.Where("spp_billing_due_date >= ?", *q.DueFrom)
	}
	if q.DueTo != nil {
		base = base.Where("spp_billing_due_date <= ?", *q.DueTo)
	}
	if q.Q != nil && *q.Q != "" {
		like := fmt.Sprintf("%%%s%%", *q.Q)
		base = base.Where(
			"(spp_billing_title ILIKE ? OR spp_billing_note ILIKE ?)",
			like, like,
		)
	}

	// Count
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Data
	var list []model.SppBillingModel
	if err := base.
		Order("spp_billing_year DESC, spp_billing_month DESC, spp_billing_created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "OK", dto.FromModels(list, total))
}

/* ======================== UPDATE (PATCH) ======================== */
// PATCH /admin/spp/billings/:id
func (h *SppBillingController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	idStr := c.Params("id")
	if idStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	var req dto.UpdateSppBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	v := validator.New()
	if err := v.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var row model.SppBillingModel
	if err := h.DB.
		Where("spp_billing_id = ? AND spp_billing_masjid_id = ?", idStr, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	req.ApplyTo(&row)

	if err := h.DB.Save(&row).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui SPP billing")
	}

	return helper.JsonOK(c, "SPP billing berhasil diperbarui", dto.FromModel(row))
}

/* ======================== DELETE (SOFT) ======================== */
// DELETE /admin/spp/billings/:id
func (h *SppBillingController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	idStr := c.Params("id")
	if idStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	// Soft delete + tenant guard
	res := h.DB.
		Where("spp_billing_id = ? AND spp_billing_masjid_id = ?", idStr, masjidID).
		Delete(&model.SppBillingModel{})
	if res.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}

	return helper.JsonOK(c, "SPP billing berhasil dihapus", fiber.Map{"id": idStr})
}
