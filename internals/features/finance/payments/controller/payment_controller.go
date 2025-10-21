// file: internals/features/finance/payments/controller/payment_controller.go
package controller

import (
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
   Handlers
======================================================================= */

// POST /payments
func (h *PaymentController) CreatePayment(c *fiber.Ctx) error {
	var req dto.CreatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json: "+err.Error())
	}
	// validasi bisnis (XOR target, method/provider, currency)
	if err := req.Validate(); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()

	// default provider → midtrans bila method=gateway & provider kosong
	if m.PaymentMethod == model.PaymentMethodGateway && (m.PaymentGatewayProvider == nil || *m.PaymentGatewayProvider == "") {
		prov := model.GatewayProviderMidtrans
		m.PaymentGatewayProvider = &prov
	}

	// simpan dulu (agar punya payment_id)
	if err := h.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "create payment failed: "+err.Error())
	}

	// Jika gateway Midtrans dan ada external_id (order_id), buat Snap token
	if m.PaymentMethod == model.PaymentMethodGateway &&
		m.PaymentGatewayProvider != nil && *m.PaymentGatewayProvider == model.GatewayProviderMidtrans {

		// external_id W A J I B ada untuk Midtrans (order_id)
		if m.PaymentExternalID == nil || strings.TrimSpace(*m.PaymentExternalID) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "payment_external_id (order_id) is required for midtrans")
		}

		// Build customer info dari meta (best-effort)
		cust := svc.CustomerInput{}
		if m.PaymentMeta != nil {
			_ = json.Unmarshal(m.PaymentMeta, &cust)
		}

		token, redirectURL, err := svc.GenerateSnapToken(*m, cust)
		if err != nil {
			return fiber.NewError(fiber.StatusBadGateway, "midtrans error: "+err.Error())
		}

		// simpan hasil snap
		now := time.Now()
		m.PaymentCheckoutURL = &redirectURL
		m.PaymentGatewayReference = &token
		m.PaymentStatus = model.PaymentStatusPending
		m.PaymentRequestedAt = &now

		if err := h.DB.WithContext(c.Context()).Save(m).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "update payment after snap failed: "+err.Error())
		}
	}

	return c.Status(fiber.StatusCreated).JSON(dto.FromModel(m))
}

// GET /payments/:id
func (h *PaymentController) GetPaymentByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var m model.Payment
	if err := h.DB.WithContext(c.Context()).
		First(&m, "payment_id = ? AND payment_deleted_at IS NULL", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "payment not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(dto.FromModel(&m))
}

// PATCH /payments/:id
func (h *PaymentController) PatchPayment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var m model.Payment
	if err := h.DB.WithContext(c.Context()).
		First(&m, "payment_id = ? AND payment_deleted_at IS NULL", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "payment not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var patch dto.UpdatePaymentRequest
	if err := c.BodyParser(&patch); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json: "+err.Error())
	}
	if err := patch.Apply(&m); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	m.PaymentUpdatedAt = time.Now()

	if err := h.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "save failed: "+err.Error())
	}
	return c.JSON(dto.FromModel(&m))
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
	GrossAmount       string `json:"gross_amount"` // string dari Midtrans
	PaymentType       string `json:"payment_type"`
	FraudStatus       string `json:"fraud_status"` // accept / challenge / deny
	TransactionID     string `json:"transaction_id"`
	SettlementTime    string `json:"settlement_time"`
	// tambahan field lain aman diabaikan
}

func (h *PaymentController) MidtransWebhook(c *fiber.Ctx) error {
	// 1) Parse payload
	var notif midtransNotif
	if err := c.BodyParser(&notif); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload: "+err.Error())
	}

	// 2) Verify signature — SHA512(order_id + status_code + gross_amount + ServerKey)
	want := strings.ToLower(notif.SignatureKey)
	raw := notif.OrderID + notif.StatusCode + notif.GrossAmount + h.MidtransServerKey
	got := sha512sum(raw)
	if want == "" || got != want {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid signature")
	}

	// 3) Find payment by external_id (order_id)
	var p model.Payment
	if err := h.DB.WithContext(c.Context()).
		First(&p, "payment_external_id = ? AND payment_deleted_at IS NULL", notif.OrderID).Error; err != nil {
		// Log event tetap meski payment belum ada (mis-order). Balas 200 agar Midtrans tidak retry terus
		_ = h.logGatewayEvent(c, nil, notif, "received", fmt.Sprintf("payment not found for order_id=%s", notif.OrderID))
		return c.JSON(fiber.Map{"status": "ignored", "reason": "payment not found"})
	}

	// 4) Simpan gateway event (idempotent by unique index provider+external_id jika diinginkan)
	if err := h.logGatewayEvent(c, &p, notif, "received", ""); err != nil {
		// duplicated? lanjutkan saja update status
	}

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
	// selalu update referensi gateway (transaction_id)
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
		return fiber.NewError(fiber.StatusInternalServerError, "update payment failed: "+err.Error())
	}

	_ = h.updateEventStatus(notif, "processed", "")

	return c.JSON(fiber.Map{
		"status":              "ok",
		"payment_id":          p.PaymentID,
		"payment_status":      p.PaymentStatus,
		"transaction_status":  notif.TransactionStatus,
		"fraud_status":        notif.FraudStatus,
		"payment_gateway_ref": p.PaymentGatewayReference,
	})
}

/* =======================================================================
   Helpers: webhook
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
		PaymentGatewayEventStatus:    status, // string (ENUM di DB)
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
