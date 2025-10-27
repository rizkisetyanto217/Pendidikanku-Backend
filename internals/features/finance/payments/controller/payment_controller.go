// file: internals/features/finance/payments/controller/payment_controller.go
package controller

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/finance/payments/dto"
	model "masjidku_backend/internals/features/finance/payments/model"
	svc "masjidku_backend/internals/features/finance/payments/service"
	helper "masjidku_backend/internals/helpers"
)

/* =======================================================================
   Controller
======================================================================= */

type PaymentController struct {
	DB                 *gorm.DB
	Validator          *validator.Validate
	MidtransServerKey  string // dipakai untuk verify signature di webhook
	UseMidtransProdEnv bool   // untuk init Snap client di bootstrap
}

func NewPaymentController(db *gorm.DB, midtransServerKey string, useProd bool) *PaymentController {
	// init midtrans snap client (sekali saja saat bootstrap)
	svc.InitMidtrans(midtransServerKey, useProd)
	return &PaymentController{
		DB:                 db,
		Validator:          validator.New(),
		MidtransServerKey:  midtransServerKey,
		UseMidtransProdEnv: useProd,
	}
}

/* =======================================================================
   Target resolver (skema baru)
   - student_bill
   - general_billing (header)
   - general_billing_kind (campaign/global/tenant)
======================================================================= */

type TargetInfo struct {
	Kind             string     // "student_bill" | "general_billing" | "kind"
	MasjidID         *uuid.UUID // bisa NULL untuk GLOBAL kind
	AmountSuggestion *int       // boleh nil
	PayerUserID      *uuid.UUID // tidak dipakai saat ini (reserved)
}

func (h *PaymentController) resolveTarget(ctx context.Context, db *gorm.DB, r *dto.CreatePaymentRequest) (TargetInfo, error) {
	var ti TargetInfo

	switch {
	case r.PaymentStudentBillID != nil:
		// student_bills
		type sbRow struct {
			ID       uuid.UUID  `gorm:"column:student_bill_id"`
			MasjidID uuid.UUID  `gorm:"column:student_bill_masjid_id"`
			Amount   int        `gorm:"column:student_bill_amount_idr"`
			Status   string     `gorm:"column:student_bill_status"`
			PayerUID *uuid.UUID `gorm:"column:student_bill_payer_user_id"`
		}
		var row sbRow
		if err := db.WithContext(ctx).
			Table("student_bills").
			Select("student_bill_id, student_bill_masjid_id, student_bill_amount_idr, student_bill_status, student_bill_payer_user_id").
			Where("student_bill_id = ? AND student_bill_deleted_at IS NULL", *r.PaymentStudentBillID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "student_bill tidak ditemukan")
		}
		ti = TargetInfo{
			Kind:             "student_bill",
			MasjidID:         &row.MasjidID,
			AmountSuggestion: &row.Amount,
			PayerUserID:      row.PayerUID,
		}

	case r.PaymentGeneralBillingID != nil:
		// general_billings (header)
		type gbRow struct {
			ID       uuid.UUID `gorm:"column:general_billing_id"`
			MasjidID uuid.UUID `gorm:"column:general_billing_masjid_id"`
			Default  *int      `gorm:"column:general_billing_default_amount_idr"`
		}
		var row gbRow
		if err := db.WithContext(ctx).
			Table("general_billings").
			Select("general_billing_id, general_billing_masjid_id, general_billing_default_amount_idr").
			Where("general_billing_id = ? AND general_billing_deleted_at IS NULL", *r.PaymentGeneralBillingID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "general_billing tidak ditemukan")
		}
		ti = TargetInfo{
			Kind:             "general_billing",
			MasjidID:         &row.MasjidID,
			AmountSuggestion: row.Default,
		}

	case r.PaymentGeneralBillingKindID != nil:
		// general_billing_kinds (kind/campaign); masjid_id bisa NULL (GLOBAL)
		type kindRow struct {
			ID       uuid.UUID  `gorm:"column:general_billing_kind_id"`
			MasjidID *uuid.UUID `gorm:"column:general_billing_kind_masjid_id"`
			Default  *int       `gorm:"column:general_billing_kind_default_amount_idr"`
			Active   bool       `gorm:"column:general_billing_kind_is_active"`
		}
		var row kindRow
		if err := db.WithContext(ctx).
			Table("general_billing_kinds").
			Select("general_billing_kind_id, general_billing_kind_masjid_id, general_billing_kind_default_amount_idr, general_billing_kind_is_active").
			Where("general_billing_kind_id = ? AND general_billing_kind_deleted_at IS NULL", *r.PaymentGeneralBillingKindID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "general_billing_kind tidak ditemukan")
		}
		if !row.Active {
			return ti, fiber.NewError(fiber.StatusBadRequest, "general_billing_kind tidak aktif")
		}
		ti = TargetInfo{
			Kind:             "kind",
			MasjidID:         row.MasjidID, // NULL = GLOBAL kind
			AmountSuggestion: row.Default,
		}
	default:
		return ti, fiber.NewError(fiber.StatusBadRequest, "wajib menyertakan salah satu target: payment_student_bill_id / payment_general_billing_id / payment_general_billing_kind_id")
	}

	return ti, nil
}

/* =======================================================================
   Handlers
======================================================================= */

// POST /payments
func (h *PaymentController) CreatePayment(c *fiber.Ctx) error {
	var req dto.CreatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
	}
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 1) Resolve target → isi masjid/amount jika kosong
	ti, err := h.resolveTarget(c.Context(), h.DB, &req)
	if err != nil {
		code := fiber.StatusBadRequest
		if fe, ok := err.(*fiber.Error); ok {
			code = fe.Code
		}
		return helper.JsonError(c, code, err.Error())
	}

	m := req.ToModel()

	// Prefill masjid dari target kalau kosong
	if m.PaymentMasjidID == nil && ti.MasjidID != nil {
		m.PaymentMasjidID = ti.MasjidID
	}
	// Prefill user (optional) dari target (kalau ada)
	if m.PaymentUserID == nil && ti.PayerUserID != nil {
		m.PaymentUserID = ti.PayerUserID
	}
	// Prefill nominal dari target jika request 0
	if m.PaymentAmountIDR == 0 && ti.AmountSuggestion != nil && *ti.AmountSuggestion > 0 {
		m.PaymentAmountIDR = *ti.AmountSuggestion
	}

	// 2) Default provider → midtrans bila method=gateway & provider kosong
	if m.PaymentMethod == model.PaymentMethodGateway && (m.PaymentGatewayProvider == nil || *m.PaymentGatewayProvider == "") {
		prov := model.GatewayProviderMidtrans
		m.PaymentGatewayProvider = &prov
	}

	// 3) Simpan dulu untuk dapat payment_id
	if err := h.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "create payment failed: "+err.Error())
	}

	// 4) Jika method gateway Midtrans → butuh external_id (order_id) + generate Snap
	if m.PaymentMethod == model.PaymentMethodGateway &&
		m.PaymentGatewayProvider != nil && *m.PaymentGatewayProvider == model.GatewayProviderMidtrans {

		if m.PaymentExternalID == nil || strings.TrimSpace(*m.PaymentExternalID) == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "payment_external_id (order_id) is required for midtrans")
		}

		cust := svc.CustomerInput{}
		if m.PaymentMeta != nil {
			_ = json.Unmarshal(m.PaymentMeta, &cust)
		}

		token, redirectURL, err := svc.GenerateSnapToken(*m, cust)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "midtrans error: "+err.Error())
		}

		now := time.Now()
		m.PaymentCheckoutURL = &redirectURL
		m.PaymentGatewayReference = &token
		m.PaymentStatus = model.PaymentStatusPending
		m.PaymentRequestedAt = &now

		if err := h.DB.WithContext(c.Context()).Save(m).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "update payment after snap failed: "+err.Error())
		}
	}

	// 5) Jika pembayaran manual dan sudah ditandai paid sejak awal → sync ke student_bills
	if m.PaymentMethod != model.PaymentMethodGateway && m.PaymentStatus == model.PaymentStatusPaid {
		_ = h.applyStudentBillSideEffects(c.Context(), h.DB, m)
	}

	return helper.JsonCreated(c, "payment created", dto.FromModel(m))
}

// GET /payments/:id
func (h *PaymentController) GetPaymentByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}
	var m model.Payment
	if err := h.DB.WithContext(c.Context()).
		First(&m, "payment_id = ? AND payment_deleted_at IS NULL", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "payment not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "ok", dto.FromModel(&m))
}

// PATCH /payments/:id
func (h *PaymentController) PatchPayment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}
	var m model.Payment
	if err := h.DB.WithContext(c.Context()).
		First(&m, "payment_id = ? AND payment_deleted_at IS NULL", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "payment not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var patch dto.UpdatePaymentRequest
	if err := c.BodyParser(&patch); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
	}
	if err := patch.Apply(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	m.PaymentUpdatedAt = time.Now()

	if err := h.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "save failed: "+err.Error())
	}

	// Jika status berubah menjadi paid/failed/canceled/refunded → sinkronkan student_bills
	_ = h.applyStudentBillSideEffects(c.Context(), h.DB, &m)

	return helper.JsonUpdated(c, "payment updated", dto.FromModel(&m))
}

/* =======================================================================
   Webhook Midtrans
======================================================================= */

type midtransNotif struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"` // capture, settlement, pending, deny, cancel, expire, refund, partial_refund, failure
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	FraudStatus       string `json:"fraud_status"` // accept / challenge / deny
	TransactionID     string `json:"transaction_id"`
	SettlementTime    string `json:"settlement_time"`
}

func (h *PaymentController) MidtransWebhook(c *fiber.Ctx) error {
	// 1) Parse payload
	var notif midtransNotif
	if err := c.BodyParser(&notif); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid payload: "+err.Error())
	}

	// 2) Verify signature — SHA512(order_id + status_code + gross_amount + ServerKey)
	want := strings.ToLower(notif.SignatureKey)
	raw := notif.OrderID + notif.StatusCode + notif.GrossAmount + h.MidtransServerKey
	got := sha512sum(raw)
	if want == "" || got != want {
		return helper.JsonError(c, fiber.StatusUnauthorized, "invalid signature")
	}

	// 3) Find payment by external_id (order_id)
	var p model.Payment
	if err := h.DB.WithContext(c.Context()).
		First(&p, "payment_external_id = ? AND payment_deleted_at IS NULL", notif.OrderID).Error; err != nil {

		// Log event tetap meski payment belum ada (mis-order).
		_ = h.logGatewayEvent(c, nil, notif, "received", fmt.Sprintf("payment not found for order_id=%s", notif.OrderID))

		// Balas 200 agar Midtrans tidak retry terus
		return helper.JsonOK(c, "ignored: payment not found", fiber.Map{
			"order_id": notif.OrderID,
			"status":   "ignored",
			"reason":   "payment not found",
		})
	}

	// 4) Simpan gateway event (idempotent)
	_ = h.logGatewayEvent(c, &p, notif, "received", "")

	// 5) Map status midtrans → status internal
	now := time.Now()
	newStatus, setFields := h.mapMidtransStatus(&p, notif, now)

	// 6) Terapkan perubahan ke model
	p.PaymentStatus = newStatus
	if setFields.PaidAt != nil {
		p.PaymentPaidAt = setFields.PaidAt
	}
	if setFields.CanceledAt != nil {
		p.PaymentCanceledAt = setFields.CanceledAt
	}
	if setFields.FailedAt != nil {
		p.PaymentFailedAt = setFields.FailedAt
	}
	if setFields.RefundedAt != nil {
		p.PaymentRefundedAt = setFields.RefundedAt
	}
	// update referensi gateway (transaction_id)
	if notif.TransactionID != "" {
		ref := notif.TransactionID
		p.PaymentGatewayReference = &ref
	}
	// normalisasi amount gross (string → int)
	if amt, err := strconv.ParseFloat(notif.GrossAmount, 64); err == nil {
		p.PaymentAmountIDR = int(amt + 0.5)
	}
	p.PaymentUpdatedAt = now

	if err := h.DB.WithContext(c.Context()).Save(&p).Error; err != nil {
		_ = h.updateEventStatus(notif, "failed", err.Error())
		return helper.JsonError(c, fiber.StatusInternalServerError, "update payment failed: "+err.Error())
	}

	// 7) Side effects ke student_bills (jika ada target)
	_ = h.applyStudentBillSideEffects(c.Context(), h.DB, &p)

	_ = h.updateEventStatus(notif, "processed", "")

	return helper.JsonOK(c, "webhook processed", fiber.Map{
		"payment_id":          p.PaymentID,
		"payment_status":      p.PaymentStatus,
		"transaction_status":  notif.TransactionStatus,
		"fraud_status":        notif.FraudStatus,
		"payment_gateway_ref": p.PaymentGatewayReference,
	})
}

/* =======================================================================
   Helpers: webhook / utils
======================================================================= */

func sha512sum(s string) string {
	h := sha512.Sum512([]byte(s))
	return hex.EncodeToString(h[:])
}

func (h *PaymentController) logGatewayEvent(c *fiber.Ctx, p *model.Payment, notif midtransNotif, status string, errMsg string) error {
	headers := map[string]string{}
	for k, v := range c.GetReqHeaders() { // v: []string
		headers[k] = strings.Join(v, ",")
	}

	headersJSON, _ := json.Marshal(headers)
	payloadJSON, _ := json.Marshal(notif)
	rawQuery := string(c.Request().URI().QueryString())

	ev := model.PaymentGatewayEvent{
		PaymentGatewayEventMasjidID:   nil,
		PaymentGatewayEventPaymentID:  nil,
		PaymentGatewayEventProvider:   string(model.GatewayProviderMidtrans),
		PaymentGatewayEventType:       strPtr(notif.TransactionStatus),
		PaymentGatewayEventExternalID: strPtr(notif.OrderID),
		PaymentGatewayEventExternalRef: func() *string {
			if notif.TransactionID != "" {
				return &notif.TransactionID
			}
			return nil
		}(),
		PaymentGatewayEventHeaders:   datatypes.JSON(headersJSON),
		PaymentGatewayEventPayload:   datatypes.JSON(payloadJSON),
		PaymentGatewayEventSignature: strPtr(notif.SignatureKey),
		PaymentGatewayEventRawQuery:  &rawQuery,
		PaymentGatewayEventStatus:    status,
		PaymentGatewayEventError:     strPtr(errMsg),
		PaymentGatewayEventTryCount:  0,
	}
	// Jika payment ada, isi relasi & tenantnya
	if p != nil {
		ev.PaymentGatewayEventPaymentID = &p.PaymentID
		ev.PaymentGatewayEventMasjidID = p.PaymentMasjidID
	}

	// insert
	if err := h.DB.WithContext(c.Context()).Create(&ev).Error; err != nil {
		lc := strings.ToLower(err.Error())
		if strings.Contains(lc, "duplicate") || strings.Contains(lc, "uq_gw_event_provider_extid_live") {
			return nil
		}
		return err
	}
	return nil
}

func (h *PaymentController) updateEventStatus(notif midtransNotif, newStatus string, errMsg string) error {
	// update by provider+external_id paling mudah
	var ev model.PaymentGatewayEvent
	q := h.DB.Where(
		"payment_gateway_event_provider = ? AND COALESCE(payment_gateway_event_external_id,'') = ? AND payment_gateway_event_deleted_at IS NULL",
		model.GatewayProviderMidtrans, notif.OrderID,
	).Order("payment_gateway_event_created_at DESC").
		Limit(1).
		First(&ev)
	if q.Error != nil {
		return q.Error
	}
	ev.PaymentGatewayEventStatus = newStatus
	ev.PaymentGatewayEventError = strPtr(errMsg)
	now := time.Now()
	ev.PaymentGatewayEventProcessedAt = &now
	return h.DB.Save(&ev).Error
}

// hasil mapping status: status target + field waktu mana yang perlu di-set
type mappedFields struct {
	PaidAt     *time.Time
	CanceledAt *time.Time
	FailedAt   *time.Time
	RefundedAt *time.Time
}

func (h *PaymentController) mapMidtransStatus(p *model.Payment, n midtransNotif, now time.Time) (model.PaymentStatus, mappedFields) {
	ts := strings.ToLower(n.TransactionStatus)
	fraud := strings.ToLower(n.FraudStatus)
	switch ts {
	case "capture":
		// untuk cc: capture + fraud=accept -> paid, fraud=challenge -> awaiting
		if fraud == "accept" {
			return model.PaymentStatusPaid, mappedFields{PaidAt: &now}
		}
		if fraud == "challenge" {
			return model.PaymentStatusAwaitingCallback, mappedFields{}
		}
		return model.PaymentStatusFailed, mappedFields{FailedAt: &now}

	case "settlement":
		return model.PaymentStatusPaid, mappedFields{PaidAt: &now}

	case "pending":
		return model.PaymentStatusPending, mappedFields{}

	case "deny":
		return model.PaymentStatusFailed, mappedFields{FailedAt: &now}

	case "cancel":
		return model.PaymentStatusCanceled, mappedFields{CanceledAt: &now}

	case "expire":
		return model.PaymentStatusExpired, mappedFields{}

	case "refund":
		return model.PaymentStatusRefunded, mappedFields{RefundedAt: &now}

	case "partial_refund":
		return model.PaymentStatusPartiallyRefunded, mappedFields{RefundedAt: &now}

	case "failure":
		return model.PaymentStatusFailed, mappedFields{FailedAt: &now}
	}
	// fallback
	return p.PaymentStatus, mappedFields{}
}

func strPtr(s string) *string { return &s }

/* =======================================================================
   Side effects ke student_bills (sinkronisasi status)
   - dipanggil dari Create (manual paid) dan Webhook/ Patch
======================================================================= */

func (h *PaymentController) applyStudentBillSideEffects(ctx context.Context, db *gorm.DB, p *model.Payment) error {
	if p == nil || p.PaymentStudentBillID == nil {
		return nil
	}

	switch p.PaymentStatus {
	case model.PaymentStatusPaid:
		// tandai student bill paid
		now := time.Now()
		paidAt := p.PaymentPaidAt
		if paidAt == nil {
			paidAt = &now
		}
		return db.WithContext(ctx).
			Exec(`
				UPDATE student_bills
				   SET student_bill_status = 'paid',
				       student_bill_paid_at = COALESCE(student_bill_paid_at, ?),
				       student_bill_updated_at = NOW()
				 WHERE student_bill_id = ?
				   AND student_bill_deleted_at IS NULL
			`, *paidAt, *p.PaymentStudentBillID).Error

	case model.PaymentStatusCanceled, model.PaymentStatusFailed, model.PaymentStatusExpired, model.PaymentStatusRefunded:
		// kembalikan ke unpaid (kebijakan sederhana; sesuaikan jika perlu)
		return db.WithContext(ctx).
			Exec(`
				UPDATE student_bills
				   SET student_bill_status = 'unpaid',
				       student_bill_paid_at = NULL,
				       student_bill_updated_at = NOW()
				 WHERE student_bill_id = ?
				   AND student_bill_deleted_at IS NULL
			`, *p.PaymentStudentBillID).Error
	}

	return nil
}