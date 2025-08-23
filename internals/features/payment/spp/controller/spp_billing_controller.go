package controller

import (
	"errors"
	"fmt"
	"strings"

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

	// Wajib class_id agar bisa auto-generate item per siswa
	if req.SppBillingClassID == nil || *req.SppBillingClassID == uuid.Nil {
	 return fiber.NewError(fiber.StatusBadRequest, "spp_billing_class_id wajib diisi untuk generate item per siswa")
	}

	if err := validator.New().Struct(req); err != nil {
	 return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

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
	 msg := strings.ToLower(err.Error())
	 if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
		return fiber.NewError(fiber.StatusConflict, "Batch SPP untuk kombinasi (masjid, kelas, bulan, tahun) sudah ada")
	 }
	 return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat SPP billing")
	}
	if m.SppBillingID == uuid.Nil {
	 tx.Rollback()
	 return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperoleh ID billing")
	}

	// Generate item per siswa aktif
	res := tx.Exec(`
		INSERT INTO user_spp_billings
			(user_spp_billing_billing_id, user_spp_billing_user_id, user_spp_billing_amount_idr)
		SELECT
			?, uc.user_classes_user_id,
			COALESCE(uc.user_classes_fee_override_monthly_idr, c.class_fee_monthly_idr, 0)
		FROM user_classes uc
		JOIN classes c ON c.class_id = uc.user_classes_class_id
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

	if err := tx.Commit().Error; err != nil {
	 return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

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
// GET /admin/spp/billings?class_id=&term_id=&month=&year=&due_from=&due_to=&q=&limit=&offset=
func (h *SppBillingController) List(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
	 return err
	}

	var q dto.ListSppBillingQuery
	if err := c.QueryParser(&q); err != nil {
	 return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
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
	if q.TermID != nil {
	 base = base.Where("spp_billing_term_id = ?", *q.TermID)
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
	 base = base.Where("(spp_billing_title ILIKE ? OR spp_billing_note ILIKE ?)", like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
	 return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

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

/* ======================== UPDATE (PUT, partial) ======================== */
// PUT /admin/spp/billings/:id
func (h *SppBillingController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	idStr := strings.TrimSpace(c.Params("id"))
	if idStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	var req dto.UpdateSppBillingRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil dulu record untuk guard tenant & eksistensi
	var curr model.SppBillingModel
	if err := h.DB.
		Where("spp_billing_id = ? AND spp_billing_masjid_id = ?", idStr, masjidID).
		First(&curr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Bangun patch hanya dari field yang dikirim (pointer != nil)
	patch := map[string]interface{}{}

	if req.SppBillingMasjidID != nil {
		patch["spp_billing_masjid_id"] = *req.SppBillingMasjidID
	}
	if req.SppBillingClassID != nil {
		patch["spp_billing_class_id"] = *req.SppBillingClassID
	}
	if req.SppBillingTermID != nil {
		patch["spp_billing_term_id"] = *req.SppBillingTermID
	}
	if req.SppBillingMonth != nil {
		patch["spp_billing_month"] = *req.SppBillingMonth
	}
	if req.SppBillingYear != nil {
		patch["spp_billing_year"] = *req.SppBillingYear
	}
	if req.SppBillingTitle != nil {
		patch["spp_billing_title"] = *req.SppBillingTitle
	}
	if req.SppBillingDueDate != nil {
		// catatan: dengan bentuk pointer ini, mengirim null TIDAK akan menghapus nilainya.
		// kalau butuh clear ke NULL, sediakan flag khusus (mis. clear_due_date=true).
		patch["spp_billing_due_date"] = *req.SppBillingDueDate
	}
	if req.SppBillingNote != nil {
		patch["spp_billing_note"] = *req.SppBillingNote
	}

	// Jika tidak ada field yang dikirim, anggap tidak ada perubahan
	if len(patch) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", dto.FromModel(curr))
	}

	// Eksekusi update (trigger BEFORE UPDATE akan set spp_billing_updated_at)
	if err := h.DB.Model(&model.SppBillingModel{}).
		Where("spp_billing_id = ? AND spp_billing_masjid_id = ?", idStr, masjidID).
		Updates(patch).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			return fiber.NewError(fiber.StatusConflict,
				"Batch SPP untuk kombinasi (masjid, kelas, bulan, tahun) sudah ada",
			)
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui SPP billing")
	}

	// Ambil ulang supaya nilai terbaru (termasuk updated_at dari trigger)
	var updated model.SppBillingModel
	if err := h.DB.
		Where("spp_billing_id = ? AND spp_billing_masjid_id = ?", idStr, masjidID).
		First(&updated).Error; err != nil {
		return helper.JsonOK(c, "SPP billing berhasil diperbarui", dto.FromModel(curr)) // fallback
	}

	return helper.JsonOK(c, "SPP billing berhasil diperbarui", dto.FromModel(updated))
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
