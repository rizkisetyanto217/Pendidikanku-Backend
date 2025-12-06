// file: internals/features/finance/payments/controller/payment_item_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/finance/payments/dto"
	model "madinahsalam_backend/internals/features/finance/payments/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

// Controller khusus CHILD: payment_items
type PaymentItemController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewPaymentItemController(db *gorm.DB) *PaymentItemController {
	return &PaymentItemController{
		DB:        db,
		Validator: validator.New(),
	}
}

// helper parse UUID dari path
func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := c.Params(name)
	return uuid.Parse(idStr)
}

// ⚠️ pakai nama beda biar ngga tabrakan dengan controller lain
func parseUUIDParamPI(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := c.Params(name)
	return uuid.Parse(idStr)
}

/* =========================================================
   POST /payments/:payment_id/items
   - bikin 1 payment_item baru di bawah payment tertentu
========================================================= */

func (h *PaymentItemController) CreatePaymentItem(c *fiber.Ctx) error {
	// 0) Auth & scope sekolah
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	paymentID, err := parseUUIDParamPI(c, "payment_id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid payment_id")
	}

	// 1) Ambil payment dulu, sekalian cek sekolah & akses
	var pay model.PaymentModel
	if err := h.DB.WithContext(c.Context()).
		First(&pay, "payment_id = ? AND payment_deleted_at IS NULL", paymentID).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "payment tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if pay.PaymentSchoolID == nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "payment tidak memiliki school_id")
	}
	schoolID := *pay.PaymentSchoolID

	// pastikan user memang member sekolah
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}
	_ = userID // sementara belum dipakai, tapi bisa untuk audit/logging

	// 2) Parse body
	var req dto.CreatePaymentItemRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
	}

	// Override supaya aman (jangan percaya client):
	// - school_id ambil dari payment
	// - payment_id ambil dari path
	req.PaymentItemSchoolID = schoolID
	req.PaymentItemPaymentID = paymentID

	// Kalau index 0 → auto next index
	if req.PaymentItemIndex == 0 {
		var maxIdx int16
		if er := h.DB.WithContext(c.Context()).
			Raw(`
				SELECT COALESCE(MAX(payment_item_index), 0)
				  FROM payment_items
				 WHERE payment_item_payment_id = ?
				   AND payment_item_deleted_at IS NULL
			`, paymentID).
			Scan(&maxIdx).Error; er != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal baca index item: "+er.Error())
		}
		req.PaymentItemIndex = maxIdx + 1
	}

	// 3) Validasi business rule di DTO
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 4) Build model dari DTO
	item := req.ToModel()

	// ==============================
	// 🧠 Auto-isi invoice & due date
	// ==============================
	now := time.Now().In(time.Local)

	// 4a) Due date default: +2 hari (date-only)
	if item.PaymentItemInvoiceDue == nil {
		due := now.AddDate(0, 0, 2)
		// normalisasi ke tengah malam
		dateOnly := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
		item.PaymentItemInvoiceDue = &dateOnly
	}

	// 4b) Ambil slug sekolah buat invoice number
	var schoolSlug string
	if err := h.DB.WithContext(c.Context()).
		Raw(`
			SELECT school_slug
			  FROM schools
			 WHERE school_id = ?
			   AND school_deleted_at IS NULL
			 LIMIT 1
		`, schoolID).Scan(&schoolSlug).Error; err != nil {

		// kalau gagal, jangan hard-fail, masih bisa lanjut dengan nomor simple
		schoolSlug = ""
	}
	schoolSlug = strings.TrimSpace(schoolSlug)

	// 4c) Kalau title kosong dan ada class_id → pakai nama kelas
	if item.PaymentItemTitle == nil && item.PaymentItemClassID != nil {
		var className string
		if err := h.DB.WithContext(c.Context()).
			Raw(`
				SELECT class_name
				  FROM classes
				 WHERE class_id = ?
				   AND class_deleted_at IS NULL
				 LIMIT 1
			`, *item.PaymentItemClassID).Scan(&className).Error; err == nil {

			className = strings.TrimSpace(className)
			if className != "" {
				title := fmt.Sprintf("Pembayaran %s", className)
				item.PaymentItemTitle = &title
				// kalau invoice_title juga kosong, ikutkan
				if item.PaymentItemInvoiceTitle == nil {
					invTitle := title
					item.PaymentItemInvoiceTitle = &invTitle
				}
			}
		}
	}

	// Kalau masih belum ada invoice_title tapi ada title → copy
	if item.PaymentItemInvoiceTitle == nil && item.PaymentItemTitle != nil {
		t := strings.TrimSpace(*item.PaymentItemTitle)
		if t != "" {
			invTitle := t
			item.PaymentItemInvoiceTitle = &invTitle
		}
	}

	// 4d) Invoice number: kalau masih kosong, generate pattern
	//     INV/{slug}/{paymentShort}/{index}
	if item.PaymentItemInvoiceNumber == nil || strings.TrimSpace(*item.PaymentItemInvoiceNumber) == "" {
		payShort := pay.PaymentID.String()
		if len(payShort) > 8 {
			payShort = payShort[:8]
		}
		slugPart := schoolSlug
		if slugPart == "" {
			slugPart = "school"
		}

		invNum := fmt.Sprintf(
			"INV/%s/%s/%02d",
			slugPart,
			payShort,
			req.PaymentItemIndex,
		)
		item.PaymentItemInvoiceNumber = &invNum
	}

	// 4e) Timestamps
	if item.PaymentItemCreatedAt.IsZero() {
		item.PaymentItemCreatedAt = now
	}
	item.PaymentItemUpdatedAt = now

	// 5) Save
	if err := h.DB.WithContext(c.Context()).Create(item).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "create payment_item failed: "+err.Error())
	}

	// NOTE:
	// Di sini kamu BISA tambahin side-effect:
	// - update total header payment_amount_idr = sum(items)
	// - atau trigger ApplyStudentBillSideEffects khusus item
	// Untuk sekarang, kita biarin simpel.

	return helper.JsonCreated(c, "payment_item created", dto.FromPaymentItemModel(item))
}

/* =========================================================
   PATCH /payment-items/:id
========================================================= */

func (h *PaymentItemController) PatchPaymentItem(c *fiber.Ctx) error {
	itemID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	// 1) Ambil item + payment (buat cek school & akses)
	var item model.PaymentItemModel
	if err := h.DB.WithContext(c.Context()).
		First(&item, "payment_item_id = ? AND payment_item_deleted_at IS NULL", itemID).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "payment_item tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var pay model.PaymentModel
	if err := h.DB.WithContext(c.Context()).
		First(&pay, "payment_id = ? AND payment_deleted_at IS NULL", item.PaymentItemPaymentID).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal baca payment header: "+err.Error())
	}

	if pay.PaymentSchoolID == nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "payment tidak memiliki school_id")
	}
	schoolID := *pay.PaymentSchoolID

	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}

	// 2) Parse patch
	var patch dto.UpdatePaymentItemRequest
	if err := c.BodyParser(&patch); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
	}

	// 3) Apply ke model
	if err := patch.Apply(&item); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	item.PaymentItemUpdatedAt = time.Now()

	// 4) Save
	if err := h.DB.WithContext(c.Context()).Save(&item).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "save failed: "+err.Error())
	}

	// NOTE: kalau kamu mau sync header amount
	// bisa hitung ulang total di sini.

	return helper.JsonUpdated(c, "payment_item updated", dto.FromPaymentItemModel(&item))
}


/* =========================================================
   DELETE /payment-items/:id
   - soft delete (set deleted_at)
========================================================= */

func (h *PaymentItemController) DeletePaymentItem(c *fiber.Ctx) error {
	itemID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	var item model.PaymentItemModel
	if err := h.DB.WithContext(c.Context()).
		First(&item, "payment_item_id = ? AND payment_item_deleted_at IS NULL", itemID).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "payment_item tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// cek akses via header
	var pay model.PaymentModel
	if err := h.DB.WithContext(c.Context()).
		First(&pay, "payment_id = ? AND payment_deleted_at IS NULL", item.PaymentItemPaymentID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal baca payment: "+err.Error())
	}
	if pay.PaymentSchoolID == nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "payment tidak memiliki school_id")
	}
	schoolID := *pay.PaymentSchoolID
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}

	now := time.Now()
	item.PaymentItemDeletedAt = &now
	item.PaymentItemUpdatedAt = now

	if err := h.DB.WithContext(c.Context()).Save(&item).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal delete payment_item: "+err.Error())
	}

	// kalau mau, di sini bisa re-hit total payment header

	return helper.JsonOK(c, "payment_item deleted", dto.FromPaymentItemModel(&item))
}
