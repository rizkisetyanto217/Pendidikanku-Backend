// file: internals/features/finance/payments/controller/payment_controller.go
package controller

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	// tambahkan di import:
	cendto "madinahsalam_backend/internals/features/school/classes/classes/dto"
	cenmodel "madinahsalam_backend/internals/features/school/classes/classes/model"

	dto "madinahsalam_backend/internals/features/finance/payments/dto"
	model "madinahsalam_backend/internals/features/finance/payments/model"
	svc "madinahsalam_backend/internals/features/finance/payments/service"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* =======================================================================
   Controller
======================================================================= */

func envOrDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

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
	SchoolID         *uuid.UUID // bisa NULL untuk GLOBAL kind
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
			SchoolID uuid.UUID  `gorm:"column:student_bill_school_id"`
			Amount   int        `gorm:"column:student_bill_amount_idr"`
			Status   string     `gorm:"column:student_bill_status"`
			PayerUID *uuid.UUID `gorm:"column:student_bill_payer_user_id"`
		}
		var row sbRow
		if err := db.WithContext(ctx).
			Table("student_bills").
			Select("student_bill_id, student_bill_school_id, student_bill_amount_idr, student_bill_status, student_bill_payer_user_id").
			Where("student_bill_id = ? AND student_bill_deleted_at IS NULL", *r.PaymentStudentBillID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "student_bill tidak ditemukan")
		}
		ti = TargetInfo{
			Kind:             "student_bill",
			SchoolID:         &row.SchoolID,
			AmountSuggestion: &row.Amount,
			PayerUserID:      row.PayerUID,
		}

	case r.PaymentGeneralBillingID != nil:
		// general_billings (header)
		type gbRow struct {
			ID       uuid.UUID `gorm:"column:general_billing_id"`
			SchoolID uuid.UUID `gorm:"column:general_billing_school_id"`
			Default  *int      `gorm:"column:general_billing_default_amount_idr"`
		}
		var row gbRow
		if err := db.WithContext(ctx).
			Table("general_billings").
			Select("general_billing_id, general_billing_school_id, general_billing_default_amount_idr").
			Where("general_billing_id = ? AND general_billing_deleted_at IS NULL", *r.PaymentGeneralBillingID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "general_billing tidak ditemukan")
		}
		ti = TargetInfo{
			Kind:             "general_billing",
			SchoolID:         &row.SchoolID,
			AmountSuggestion: row.Default,
		}

	case r.PaymentGeneralBillingKindID != nil:
		// general_billing_kinds (kind/campaign); school_id bisa NULL (GLOBAL)
		type kindRow struct {
			ID       uuid.UUID  `gorm:"column:general_billing_kind_id"`
			SchoolID *uuid.UUID `gorm:"column:general_billing_kind_school_id"`
			Default  *int       `gorm:"column:general_billing_kind_default_amount_idr"`
			Active   bool       `gorm:"column:general_billing_kind_is_active"`
		}
		var row kindRow
		if err := db.WithContext(ctx).
			Table("general_billing_kinds").
			Select("general_billing_kind_id, general_billing_kind_school_id, general_billing_kind_default_amount_idr, general_billing_kind_is_active").
			Where("general_billing_kind_id = ? AND general_billing_kind_deleted_at IS NULL", *r.PaymentGeneralBillingKindID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "general_billing_kind tidak ditemukan")
		}
		if !row.Active {
			return ti, fiber.NewError(fiber.StatusBadRequest, "general_billing_kind tidak aktif")
		}
		ti = TargetInfo{
			Kind:             "kind",
			SchoolID:         row.SchoolID, // NULL = GLOBAL kind
			AmountSuggestion: row.Default,
		}
	default:
		return ti, fiber.NewError(fiber.StatusBadRequest, "wajib menyertakan salah satu target: payment_student_bill_id / payment_general_billing_id / payment_general_billing_kind_id")
	}

	return ti, nil
}

/* =======================================================================
   Helpers (snapshots user)
======================================================================= */

// Ambil snapshot user_name/full_name/email/donation name dari users + user_profiles
func (h *PaymentController) hydrateUserSnapshots(ctx context.Context, db *gorm.DB, userID uuid.UUID) (userName, fullName, email, donationName *string, err error) {
	var row struct {
		UserName *string `gorm:"column:user_name"`
		FullName *string `gorm:"column:full_name"`
		Email    *string `gorm:"column:email"`
		Donation *string `gorm:"column:donation_name"`
	}
	q := `
		SELECT 
			COALESCE(NULLIF(u.user_name,''), NULLIF(split_part(u.email,'@',1), ''), NULLIF(up.user_profile_slug,'')) AS user_name,
			NULLIF(u.full_name,'') AS full_name,
			NULLIF(u.email,'') AS email,
			NULLIF(up.user_profile_donation_name,'') AS donation_name
		FROM users u
		LEFT JOIN user_profiles up ON up.user_profile_user_id = u.id AND up.user_profile_deleted_at IS NULL
		WHERE u.id = ?
		LIMIT 1`
	if err2 := db.WithContext(ctx).Raw(q, userID).Scan(&row).Error; err2 != nil {
		return nil, nil, nil, nil, err2
	}
	trim := func(p *string) *string {
		if p == nil {
			return nil
		}
		s := strings.TrimSpace(*p)
		if s == "" {
			return nil
		}
		return &s
	}
	return trim(row.UserName), trim(row.FullName), trim(row.Email), trim(row.Donation), nil
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

	// 1) Resolve target → isi school/amount jika kosong
	ti, err := h.resolveTarget(c.Context(), h.DB, &req)
	if err != nil {
		code := fiber.StatusBadRequest
		if fe, ok := err.(*fiber.Error); ok {
			code = fe.Code
		}
		return helper.JsonError(c, code, err.Error())
	}

	m := req.ToModel()

	// Prefill school dari target kalau kosong
	if m.PaymentSchoolID == nil && ti.SchoolID != nil {
		m.PaymentSchoolID = ti.SchoolID
	}
	// Prefill user (optional) dari target (kalau ada)
	if m.PaymentUserID == nil && ti.PayerUserID != nil {
		m.PaymentUserID = ti.PayerUserID
	}
	// Prefill nominal dari target jika request 0
	if m.PaymentAmountIDR == 0 && ti.AmountSuggestion != nil && *ti.AmountSuggestion > 0 {
		m.PaymentAmountIDR = *ti.AmountSuggestion
	}

	// Auto-hydrate user snapshots (jika ada PaymentUserID)
	if m.PaymentUserID != nil {
		if un, fn, em, dn, er := h.hydrateUserSnapshots(c.Context(), h.DB, *m.PaymentUserID); er == nil {
			if m.PaymentUserNameSnapshot == nil {
				m.PaymentUserNameSnapshot = un
			}
			if m.PaymentFullNameSnapshot == nil {
				m.PaymentFullNameSnapshot = fn
			}
			if m.PaymentEmailSnapshot == nil {
				m.PaymentEmailSnapshot = em
			}
			if m.PaymentDonationNameSnapshot == nil {
				m.PaymentDonationNameSnapshot = dn
			}
		}
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

	// 3a) Jika meta mengandung kategori registration + enrollment id → link & set awaiting_payment
	if m.PaymentMeta != nil {
		meta := parseRegistrationMeta(m.PaymentMeta)
		if meta.StudentClassEnrollmentID != nil && meta.FeeRuleGBKCategory == "registration" {
			_ = h.attachEnrollmentOnCreate(c.Context(), h.DB, m, *meta.StudentClassEnrollmentID)
		}
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

		token, redirectURL, err := svc.GenerateSnapToken(*m, cust, "")
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

		// Setelah status → pending, sinkronkan lagi enrollment → awaiting_payment + snapshot/ID
		_ = h.applyEnrollmentSideEffects(c.Context(), h.DB, m)
	}

	// 5) Jika pembayaran manual dan sudah ditandai paid sejak awal → sync ke student_bills & enrollment
	if m.PaymentMethod != model.PaymentMethodGateway && m.PaymentStatus == model.PaymentStatusPaid {
		_ = h.applyStudentBillSideEffects(c.Context(), h.DB, m)
		_ = h.applyEnrollmentSideEffects(c.Context(), h.DB, m)
	}

	return helper.JsonCreated(c, "payment created", dto.FromModel(m))
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

	// Jika status berubah → sinkronkan student_bills & enrollment
	_ = h.applyStudentBillSideEffects(c.Context(), h.DB, &m)
	_ = h.applyEnrollmentSideEffects(c.Context(), h.DB, &m)

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

	// 7) Side effects ke student_bills & enrollment (jika ada target/meta)
	_ = h.applyStudentBillSideEffects(c.Context(), h.DB, &p)
	_ = h.applyEnrollmentSideEffects(c.Context(), h.DB, &p)

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
		PaymentGatewayEventSchoolID:   nil,
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
		ev.PaymentGatewayEventSchoolID = p.PaymentSchoolID
	}

	// insert
	if err := h.DB.WithContext(c.Context()).Create(&ev).Error; err != nil {
		lc := strings.ToLower(err.Error())
		// index unik opsional
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

/* =======================================================================
   ===== Enrollment integration based on payment_meta (registration) =====
======================================================================= */

// Meta yang kita butuhkan untuk hubungkan payment ↔ enrollment
type registrationMeta struct {
	StudentClassEnrollmentID *uuid.UUID `json:"student_class_enrollments_id"`
	FeeRuleGBKCategory       string     `json:"fee_rule_gbk_category_snapshot"`

	// detail fee rule yang akan disalin ke enrollment.preferences
	FeeRuleID           *uuid.UUID `json:"fee_rule_id"`
	FeeRuleOptionCode   *string    `json:"fee_rule_option_code"`
	FeeRuleOptionLabel  *string    `json:"fee_rule_option_label"`
	FeeRuleOptionAmount *int64     `json:"fee_rule_option_amount_idr"`

	// payer dari token/met
	PayerUserID *uuid.UUID `json:"payer_user_id"`
}

// parser meta → normalisasi category lower-case
func parseRegistrationMeta(j datatypes.JSON) registrationMeta {
	var m registrationMeta
	_ = json.Unmarshal(j, &m)
	m.FeeRuleGBKCategory = strings.ToLower(strings.TrimSpace(m.FeeRuleGBKCategory))
	return m
}

// Build patch JSON untuk enrollment.preferences
func buildEnrollmentPrefPatch(payer *uuid.UUID, meta registrationMeta) datatypes.JSON {
	payload := map[string]interface{}{
		"registration": map[string]interface{}{
			"fee_rule_id":            meta.FeeRuleID,
			"fee_rule_option_code":   meta.FeeRuleOptionCode,
			"fee_rule_option_label":  meta.FeeRuleOptionLabel,
			"fee_rule_option_amount": meta.FeeRuleOptionAmount,
			"category_snapshot":      meta.FeeRuleGBKCategory,
		},
	}
	if payer != nil {
		payload["payer_user_id"] = payer
	}
	b, _ := json.Marshal(payload)
	return datatypes.JSON(b)
}

// letakkan dekat tipe/DTO meta
type bundleMeta struct {
	EnrollmentIDs []uuid.UUID `json:"enrollment_ids"`
}

func extractEnrollmentIDs(j datatypes.JSON) []uuid.UUID {
	ids := []uuid.UUID{}
	// single
	var r registrationMeta
	_ = json.Unmarshal(j, &r)
	if r.StudentClassEnrollmentID != nil {
		ids = append(ids, *r.StudentClassEnrollmentID)
	}
	// bundle
	var b struct {
		Bundle bundleMeta `json:"bundle"`
	}
	if err := json.Unmarshal(j, &b); err == nil && len(b.Bundle.EnrollmentIDs) > 0 {
		ids = append(ids, b.Bundle.EnrollmentIDs...)
	}
	return ids
}

// Dipanggil segera setelah payment dibuat (CreatePayment) untuk set awaiting_payment & snapshot + merge prefs + set total_due (jika 0)
func (h *PaymentController) attachEnrollmentOnCreate(
	ctx context.Context, db *gorm.DB, p *model.Payment, enrollmentID uuid.UUID,
) error {
	snap, _ := json.Marshal(dto.FromModel(p))

	meta := registrationMeta{}
	if p.PaymentMeta != nil {
		meta = parseRegistrationMeta(p.PaymentMeta)
	}
	// tentukan payer: prioritas dari meta.PayerUserID; fallback ke PaymentUserID
	payer := meta.PayerUserID
	if payer == nil && p.PaymentUserID != nil {
		payer = p.PaymentUserID
	}
	prefPatch := buildEnrollmentPrefPatch(payer, meta)

	return db.WithContext(ctx).Exec(`
		UPDATE student_class_enrollments
		   SET student_class_enrollments_payment_id       = ?,
		       student_class_enrollments_payment_snapshot = ?::jsonb,
		       student_class_enrollments_status           = 'awaiting_payment',
		       student_class_enrollments_preferences      = COALESCE(student_class_enrollments_preferences,'{}'::jsonb) || ?::jsonb,
		       student_class_enrollments_total_due_idr    = CASE
		                                                    WHEN COALESCE(student_class_enrollments_total_due_idr,0)=0 THEN ?
		                                                    ELSE student_class_enrollments_total_due_idr
		                                                   END,
		       student_class_enrollments_updated_at       = NOW()
		 WHERE student_class_enrollments_id = ?
		   AND student_class_enrollments_deleted_at IS NULL
	`, p.PaymentID, datatypes.JSON(snap), prefPatch, p.PaymentAmountIDR, enrollmentID).Error
}

// Sinkronkan status enrollment tiap kali status payment berubah (merge prefs juga)
func (h *PaymentController) applyEnrollmentSideEffects(ctx context.Context, db *gorm.DB, p *model.Payment) error {
	if p == nil || p.PaymentMeta == nil {
		return nil
	}

	// wajib kategori registration
	var cat struct {
		FeeRuleGBKCategory string `json:"fee_rule_gbk_category_snapshot"`
	}
	_ = json.Unmarshal(p.PaymentMeta, &cat)
	if strings.ToLower(strings.TrimSpace(cat.FeeRuleGBKCategory)) != "registration" {
		return nil
	}

	ids := extractEnrollmentIDs(p.PaymentMeta)
	if len(ids) == 0 {
		return nil
	}

	// build prefPatch & snap seperti semula ...
	meta := parseRegistrationMeta(p.PaymentMeta)
	payer := meta.PayerUserID
	if payer == nil && p.PaymentUserID != nil {
		payer = p.PaymentUserID
	}
	prefPatch := buildEnrollmentPrefPatch(payer, meta)
	snap, _ := json.Marshal(dto.FromModel(p))

	switch p.PaymentStatus {
	case model.PaymentStatusPaid:
		for _, eid := range ids {
			if err := db.WithContext(ctx).Exec(`
                UPDATE student_class_enrollments
                   SET student_class_enrollments_status           = 'accepted',
                       student_class_enrollments_accepted_at      = COALESCE(student_class_enrollments_accepted_at, NOW()),
                       student_class_enrollments_payment_id       = ?,
                       student_class_enrollments_payment_snapshot = ?::jsonb,
                       student_class_enrollments_preferences      = COALESCE(student_class_enrollments_preferences,'{}'::jsonb) || ?::jsonb,
                       student_class_enrollments_total_due_idr    = CASE
                           WHEN COALESCE(student_class_enrollments_total_due_idr,0)=0 THEN ?
                           ELSE student_class_enrollments_total_due_idr END,
                       student_class_enrollments_updated_at       = NOW()
                 WHERE student_class_enrollments_id = ?
                   AND student_class_enrollments_deleted_at IS NULL
            `, p.PaymentID, datatypes.JSON(snap), prefPatch, p.PaymentAmountIDR, eid).Error; err != nil {
				return err
			}
		}
	case model.PaymentStatusCanceled, model.PaymentStatusFailed, model.PaymentStatusExpired, model.PaymentStatusRefunded, model.PaymentStatusPartiallyRefunded:
		for _, eid := range ids {
			if err := db.WithContext(ctx).Exec(`
                UPDATE student_class_enrollments
                   SET student_class_enrollments_status           = 'awaiting_payment',
                       student_class_enrollments_payment_id       = NULL,
                       student_class_enrollments_payment_snapshot = NULL,
                       student_class_enrollments_preferences      = COALESCE(student_class_enrollments_preferences,'{}'::jsonb) || ?::jsonb,
                       student_class_enrollments_updated_at       = NOW()
                 WHERE student_class_enrollments_id = ?
                   AND student_class_enrollments_deleted_at IS NULL
            `, prefPatch, eid).Error; err != nil {
				return err
			}
		}
	default: // pending / awaiting_callback / initiated
		for _, eid := range ids {
			if err := db.WithContext(ctx).Exec(`
                UPDATE student_class_enrollments
                   SET student_class_enrollments_status           = 'awaiting_payment',
                       student_class_enrollments_payment_id       = ?,
                       student_class_enrollments_payment_snapshot = ?::jsonb,
                       student_class_enrollments_preferences      = COALESCE(student_class_enrollments_preferences,'{}'::jsonb) || ?::jsonb,
                       student_class_enrollments_total_due_idr    = CASE
                           WHEN COALESCE(student_class_enrollments_total_due_idr,0)=0 THEN ?
                           ELSE student_class_enrollments_total_due_idr END,
                       student_class_enrollments_updated_at       = NOW()
                 WHERE student_class_enrollments_id = ?
                   AND student_class_enrollments_deleted_at IS NULL
            `, p.PaymentID, datatypes.JSON(snap), prefPatch, p.PaymentAmountIDR, eid).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// ================= DTO & helpers (request bundle) =================

type BundleItem struct {
	ClassID           uuid.UUID `json:"class_id" validate:"required"`
	FeeRuleOptionCode *string   `json:"fee_rule_option_code,omitempty"` // pilih kode option untuk kelas ini
	CustomAmountIDR   *int64    `json:"custom_amount_idr,omitempty"`    // nominal custom utk kelas ini (≥ min option)
	CustomLabel       *string   `json:"custom_label,omitempty"`         // label custom (opsional)
}

type CreateRegistrationAndPaymentRequest struct {
	// Rekomendasi baru (per-kelas):
	Items []BundleItem `json:"items" validate:"omitempty,min=1,dive"`

	// Backward-compat:
	ClassIDs          []uuid.UUID `json:"class_ids" validate:"omitempty,min=1,dive,required"`
	ClassID           uuid.UUID   `json:"class_id"  validate:"-"`
	FeeRuleID         uuid.UUID   `json:"fee_rule_id" validate:"required"`
	FeeRuleOptionCode *string     `json:"fee_rule_option_code,omitempty"` // fallback untuk semua kelas
	CustomAmountIDR   *int64      `json:"custom_amount_idr,omitempty"`    // TOTAL fallback → dibagi rata
	CustomLabel       *string     `json:"custom_label,omitempty"`

	// Pembayaran
	PaymentMethod          *model.PaymentMethod          `json:"payment_method"`
	PaymentGatewayProvider *model.PaymentGatewayProvider `json:"payment_gateway_provider"`
	PaymentExternalID      *string                       `json:"payment_external_id"`

	Customer *svc.CustomerInput `json:"customer,omitempty"`
	Notes    string             `json:"notes,omitempty"`
}

type CreateRegistrationAndPaymentResponse struct {
	Enrollments []cendto.StudentClassEnrollmentResponse `json:"enrollments"`
	Payment     any                                     `json:"payment"`
}

type optRow struct {
	Code    string `json:"code"`
	Label   string `json:"label"`
	Amount  int64  `json:"amount"`
	Default *bool  `json:"default,omitempty"`
}

func findByCode(opts []optRow, code string) *optRow {
	c := strings.ToUpper(strings.TrimSpace(code))
	for i := range opts {
		if strings.ToUpper(opts[i].Code) == c {
			return &opts[i]
		}
	}
	return nil
}
func firstDefault(opts []optRow) *optRow {
	for i := range opts {
		if opts[i].Default != nil && *opts[i].Default {
			return &opts[i]
		}
	}
	return nil
}
func minAmount(opts []optRow) int64 {
	if len(opts) == 0 {
		return 0
	}
	m := opts[0].Amount
	for i := 1; i < len(opts); i++ {
		if opts[i].Amount < m {
			m = opts[i].Amount
		}
	}
	return m
}
func genOrderID(prefix string) string {
	now := time.Now().In(time.Local).Format("20060102-150405")
	u := uuid.New().String()
	if len(u) > 8 {
		u = u[:8]
	}
	return prefix + "-" + now + "-" + strings.ToUpper(u)
}

// Generate NIM untuk siswa baru berdasarkan term (diambil dari salah satu class)
// Format: YYYYAA####  (YYYY = tahun awal akademik, AA = angkatan 2 digit, #### = sequence per sekolah+prefix)
func (h *PaymentController) generateStudentCodeForClass(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	classID uuid.UUID,
) (string, error) {
	if classID == uuid.Nil {
		return "", fmt.Errorf("class_id kosong saat generate NIM")
	}

	var row struct {
		YearRaw     *string `gorm:"column:year"`
		AngkatanRaw *string `gorm:"column:angkatan"`
	}

	// Ambil snapshot tahun akademik + angkatan dari tabel classes
	if err := tx.WithContext(ctx).Raw(`
		SELECT 
			NULLIF(class_academic_term_academic_year_snapshot,'') AS year,
			NULLIF(class_academic_term_angkatan_snapshot,'')      AS angkatan
		FROM classes
		WHERE class_id = ?
		  AND class_school_id = ?
		  AND class_deleted_at IS NULL
		LIMIT 1
	`, classID, schoolID).Scan(&row).Error; err != nil {
		return "", fmt.Errorf("gagal select term snapshot: %w", err)
	}

	// ===== Normalisasi YEAR =====
	var year4 string
	if row.YearRaw != nil && strings.TrimSpace(*row.YearRaw) != "" {
		y := strings.TrimSpace(*row.YearRaw) // contoh "2024/2025" atau "2025"
		if len(y) >= 4 {
			year4 = y[:4]
		} else {
			year4 = fmt.Sprintf("%04d", time.Now().Year())
		}
	} else {
		year4 = fmt.Sprintf("%04d", time.Now().Year())
	}

	// ===== Normalisasi ANGKATAN =====
	var angkatanInt int
	if row.AngkatanRaw != nil && strings.TrimSpace(*row.AngkatanRaw) != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(*row.AngkatanRaw)); err == nil && n >= 0 {
			angkatanInt = n
		} else {
			angkatanInt = 0
		}
	} else {
		angkatanInt = 0
	}

	// prefix: YYYY + angkatan (2 digit)
	prefix := fmt.Sprintf("%s%02d", year4, angkatanInt)

	// Cari sequence terakhir untuk prefix ini di sekolah tersebut
	var lastSeq int
	if err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(MAX(RIGHT(school_student_code, 4)::int), 0)
		FROM school_students
		WHERE school_student_school_id = ?
		  AND school_student_code LIKE ?
	`, schoolID, prefix+"%").Scan(&lastSeq).Error; err != nil {
		return "", fmt.Errorf("gagal hitung sequence NIM: %w", err)
	}

	next := lastSeq + 1
	code := fmt.Sprintf("%s%04d", prefix, next) // YYYYAA####

	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("kode hasil generate kosong (prefix=%s, lastSeq=%d)", prefix, lastSeq)
	}

	return code, nil
}

// ================= HANDLER: POST /payments/registration-enroll =================
func (h *PaymentController) CreateRegistrationAndPayment(c *fiber.Ctx) error {
	// ✅ penting: supaya GetSchoolIDBySlug & helper lain bisa akses DB (kalau kepake di tempat lain)
	c.Locals("DB", h.DB)

	// 0) Auth
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	// 0a) Resolve school_id murni dari token / context (BUKAN dari path)
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// helper sudah balikin fiber.Error yang rapi
		return err
	}

	// 0b) Pastikan user memang member sekolah itu (student/teacher/dll)
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}
	c.Locals("__school_guard_ok", schoolID.String())

	// 1) Body + normalisasi items
	var req CreateRegistrationAndPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
	}
	if len(req.Items) == 0 {
		ids := req.ClassIDs
		if len(ids) == 0 && req.ClassID != uuid.Nil {
			ids = []uuid.UUID{req.ClassID}
		}
		for _, cid := range ids {
			b := BundleItem{ClassID: cid}
			if req.CustomAmountIDR != nil {
				b.CustomAmountIDR = req.CustomAmountIDR
				b.CustomLabel = req.CustomLabel
			} else if req.FeeRuleOptionCode != nil {
				b.FeeRuleOptionCode = req.FeeRuleOptionCode
			}
			req.Items = append(req.Items, b)
		}
	}
	if len(req.Items) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "items / class_ids wajib")
	}

	// Defaults
	method := model.PaymentMethodGateway
	if req.PaymentMethod != nil {
		method = *req.PaymentMethod
	}
	provider := model.GatewayProviderMidtrans
	if req.PaymentGatewayProvider != nil && *req.PaymentGatewayProvider != "" {
		provider = *req.PaymentGatewayProvider
	}

	// ==== TX ====
	tx := h.DB.WithContext(c.Context()).Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// 2) user → user_profile_id
	var profileID uuid.UUID
	{
		var pidStr string
		if err := tx.Raw(`
			SELECT user_profile_id
			  FROM user_profiles
			 WHERE user_profile_user_id = ?
			   AND user_profile_deleted_at IS NULL
			 LIMIT 1
		`, userID).Scan(&pidStr).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal cek profile: "+err.Error())
		}
		pidStr = strings.TrimSpace(pidStr)
		if pidStr == "" {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "profil pengguna tidak ditemukan")
		}
		id, er := uuid.Parse(pidStr)
		if er != nil || id == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal parse user_profile_id")
		}
		profileID = id
	}

	// 2a) Ambil kandidat nama dari users + snapshot profile
	var prof struct {
		UserID   uuid.UUID `gorm:"column:user_id"`
		FullName *string   `gorm:"column:full_name"`
		Email    *string   `gorm:"column:email"`
		SnapName *string   `gorm:"column:snap_name"`
	}
	if err := tx.Raw(`
		SELECT 
			up.user_profile_user_id            AS user_id,
			u.full_name                        AS full_name,
			u.email                            AS email,
			up.user_profile_full_name_snapshot AS snap_name
		FROM user_profiles up
		JOIN users u ON u.id = up.user_profile_user_id
		WHERE up.user_profile_id = ?
		LIMIT 1
	`, profileID).Scan(&prof).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil profil pengguna")
	}
	pickProfileName := func() *string {
		for _, s := range []*string{prof.FullName, prof.SnapName} {
			if s != nil && strings.TrimSpace(*s) != "" {
				v := strings.TrimSpace(*s)
				return &v
			}
		}
		return nil
	}()

	// ---- sebelum blok 2b ----
	// tentukan class_id referensi untuk NIM (pakai item pertama)
	baseClassID := uuid.Nil
	if len(req.Items) > 0 {
		baseClassID = req.Items[0].ClassID
	} else if len(req.ClassIDs) > 0 {
		baseClassID = req.ClassIDs[0]
	} else if req.ClassID != uuid.Nil {
		baseClassID = req.ClassID
	}

	// 2b) map/auto-provision SchoolStudent (isi snapshot name + NIM/student_code jika baru)
	var schoolStudentID uuid.UUID
	{
		var sidStr string
		if err := tx.Raw(`
        SELECT school_student_id
          FROM school_students
         WHERE school_student_school_id       = ?
           AND school_student_user_profile_id = ?
           AND school_student_deleted_at      IS NULL
         LIMIT 1
    `, schoolID, profileID).Scan(&sidStr).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal cek siswa: "+err.Error())
		}
		if s := strings.TrimSpace(sidStr); s != "" {
			if sid, er := uuid.Parse(s); er == nil {
				schoolStudentID = sid
			}
		}

		// 🔁 Siswa lama: kalau belum punya code dan ada baseClassID → generate WAJIB
		if schoolStudentID != uuid.Nil && baseClassID != uuid.Nil {
			var currentCode *string
			if err := tx.Raw(`
            SELECT school_student_code
              FROM school_students
             WHERE school_student_id = ?
             LIMIT 1
        `, schoolStudentID).Scan(&currentCode).Error; err != nil {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal cek kode siswa: "+err.Error())
			}

			if currentCode == nil || strings.TrimSpace(*currentCode) == "" {
				code, er := h.generateStudentCodeForClass(c.Context(), tx, schoolID, baseClassID)
				if er != nil {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "gagal generate kode siswa: "+er.Error())
				}
				code = strings.TrimSpace(code)
				if code == "" {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "generate kode siswa kosong")
				}

				if err := tx.Exec(`
                UPDATE school_students
                   SET school_student_code      = ?,
                       school_student_updated_at = NOW()
                 WHERE school_student_id        = ?
            `, code, schoolStudentID).Error; err != nil {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "gagal update kode siswa: "+err.Error())
				}
			}
		}

		// Kalau belum ada (aktif maupun restore), cek dulu yang soft-deleted
		if schoolStudentID == uuid.Nil {
			var delStr string
			if er := tx.Raw(`
			SELECT school_student_id
			  FROM school_students
			 WHERE school_student_school_id       = ?
			   AND school_student_user_profile_id = ?
			   AND school_student_deleted_at      IS NOT NULL
			 LIMIT 1
		`, schoolID, profileID).Scan(&delStr).Error; er != nil {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal cek siswa (deleted): "+er.Error())
			}
			if s := strings.TrimSpace(delStr); s != "" {
				if did, er := uuid.Parse(s); er == nil && did != uuid.Nil {
					if er2 := tx.Exec(`
					UPDATE school_students
					   SET school_student_deleted_at = NULL,
					       school_student_status     = COALESCE(school_student_status, 'active'),
					       school_student_updated_at = NOW()
					 WHERE school_student_id = ?
				`, did).Error; er2 != nil {
						_ = tx.Rollback()
						return helper.JsonError(c, fiber.StatusInternalServerError, "gagal restore siswa: "+er2.Error())
					}
					schoolStudentID = did
				}
			}
		}

		// Jika masih belum ada → buat baru + generate NIM (student_code) dari term pertama
		if schoolStudentID == uuid.Nil {
			var newStudentCode *string
			if baseClassID != uuid.Nil {
				code, er := h.generateStudentCodeForClass(c.Context(), tx, schoolID, baseClassID)
				if er != nil {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "gagal generate kode siswa (baru): "+er.Error())
				}
				code = strings.TrimSpace(code)
				if code != "" {
					cc := code
					newStudentCode = &cc
				}
			}

			shortUID := strings.ReplaceAll(userID.String(), "-", "")
			if len(shortUID) > 8 {
				shortUID = shortUID[:8]
			}
			rand4 := strings.ToLower(uuid.New().String()[:4])
			genSlug := fmt.Sprintf("u-%s-%s", shortUID, rand4)

			var newIDStr string
			if pickProfileName != nil {
				if er := tx.Raw(`
				INSERT INTO school_students (
					school_student_school_id,
					school_student_user_profile_id,
					school_student_slug,
					school_student_status,
					school_student_class_sections,
					school_student_user_profile_name_snapshot,
					school_student_code
				) VALUES (?, ?, ?, 'active', '[]'::jsonb, ?, ?)
				RETURNING school_student_id
			`,
					schoolID,
					profileID,
					genSlug,
					*pickProfileName,
					newStudentCode,
				).Scan(&newIDStr).Error; er != nil {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "gagal membuat siswa: "+er.Error())
				}
			} else {
				if er := tx.Raw(`
				INSERT INTO school_students (
					school_student_school_id,
					school_student_user_profile_id,
					school_student_slug,
					school_student_status,
					school_student_class_sections,
					school_student_code
				) VALUES (?, ?, ?, 'active', '[]'::jsonb, ?)
				RETURNING school_student_id
			`,
					schoolID,
					profileID,
					genSlug,
					newStudentCode,
				).Scan(&newIDStr).Error; er != nil {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "gagal membuat siswa: "+er.Error())
				}
			}

			nid, er := uuid.Parse(strings.TrimSpace(newIDStr))
			if er != nil || nid == uuid.Nil {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal parse school_student_id")
			}
			schoolStudentID = nid
		}
	}

	// 2c) Pastikan user punya role "student" di sekolah ini (scoped)
	if err := tx.Exec(`
		SELECT fn_grant_role(?, 'student', ?, ?)
	`, userID, schoolID, userID).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal set role student: "+err.Error())
	}

	// 3) Fee rule + options (1 rule untuk semua item)
	type feeRuleHeader struct {
		ID            uuid.UUID      `gorm:"column:fee_rule_id"`
		SchoolID      uuid.UUID      `gorm:"column:fee_rule_school_id"`
		GBKID         uuid.UUID      `gorm:"column:fee_rule_general_billing_kind_id"`
		GBKCategory   string         `gorm:"column:fee_rule_gbk_category_snapshot"`
		AmountOptions datatypes.JSON `gorm:"column:fee_rule_amount_options"`
	}
	var fr feeRuleHeader
	if err := tx.Raw(`
		SELECT fee_rule_id,
		       fee_rule_school_id,
		       fee_rule_general_billing_kind_id,
		       fee_rule_gbk_category_snapshot,
		       fee_rule_amount_options
		  FROM fee_rules
		 WHERE fee_rule_id = ?
		   AND fee_rule_deleted_at IS NULL
		 LIMIT 1
	`, req.FeeRuleID).Scan(&fr).Error; err != nil || fr.ID == uuid.Nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusNotFound, "fee_rule tidak ditemukan")
	}
	if strings.ToLower(strings.TrimSpace(fr.GBKCategory)) != "registration" {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule bukan kategori registration")
	}
	if fr.SchoolID != schoolID {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule tidak untuk sekolah ini")
	}
	var opts []optRow
	if len(fr.AmountOptions) > 0 {
		_ = json.Unmarshal(fr.AmountOptions, &opts)
	}
	if len(opts) == 0 {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule tidak memiliki amount_options")
	}
	minOpt := minAmount(opts)

	// 4) Validasi semua kelas + hitung nominal per item
	type itemResolved struct {
		ClassID     uuid.UUID
		AmountIDR   int64
		Source      string // "option"|"custom"
		Code        string
		Label       string
		CustomLabel *string
	}
	items := make([]itemResolved, 0, len(req.Items))
	classSeen := map[uuid.UUID]struct{}{}

	for _, it := range req.Items {
		if it.ClassID == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "class_id item tidak valid")
		}
		if _, ok := classSeen[it.ClassID]; ok {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "class_id duplikat pada items")
		}
		classSeen[it.ClassID] = struct{}{}

		// cek kepemilikan class
		var csStr string
		if err := tx.Raw(`SELECT class_school_id FROM classes WHERE class_id = ? AND class_deleted_at IS NULL`, it.ClassID).
			Scan(&csStr).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal cek class: "+err.Error())
		}
		csStr = strings.TrimSpace(csStr)
		if csStr == "" {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "class tidak ditemukan: "+it.ClassID.String())
		}
		cs, er := uuid.Parse(csStr)
		if er != nil || cs == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal parse class_school_id")
		}
		if cs != schoolID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "class tidak valid di sekolah ini: "+it.ClassID.String())
		}

		// nominal & label per item
		switch {
		case it.CustomAmountIDR != nil:
			if *it.CustomAmountIDR < minOpt {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusBadRequest, fmt.Sprintf("custom_amount_idr minimum %d untuk kelas %s", minOpt, it.ClassID.String()))
			}
			lbl := "Custom Amount"
			if it.CustomLabel != nil && strings.TrimSpace(*it.CustomLabel) != "" {
				lbl = strings.TrimSpace(*it.CustomLabel)
			}
			items = append(items, itemResolved{
				ClassID:     it.ClassID,
				AmountIDR:   *it.CustomAmountIDR,
				Source:      "custom",
				Code:        "CUSTOM",
				Label:       lbl,
				CustomLabel: it.CustomLabel,
			})
		default:
			var chosen *optRow
			if it.FeeRuleOptionCode != nil && strings.TrimSpace(*it.FeeRuleOptionCode) != "" {
				chosen = findByCode(opts, *it.FeeRuleOptionCode)
				if chosen == nil {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule_option_code tidak valid untuk kelas "+it.ClassID.String())
				}
			} else {
				if req.FeeRuleOptionCode != nil && strings.TrimSpace(*req.FeeRuleOptionCode) != "" {
					chosen = findByCode(opts, *req.FeeRuleOptionCode)
					if chosen == nil {
						_ = tx.Rollback()
						return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule_option_code global tidak valid")
					}
				} else {
					chosen = firstDefault(opts)
					if chosen == nil && len(opts) > 1 {
						_ = tx.Rollback()
						return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule punya banyak pilihan; berikan fee_rule_option_code di item")
					}
					if chosen == nil {
						chosen = &opts[0]
					}
				}
			}
			items = append(items, itemResolved{
				ClassID:   it.ClassID,
				AmountIDR: chosen.Amount,
				Source:    "option",
				Code:      chosen.Code,
				Label:     chosen.Label,
			})
		}
	}

	// ---------------------------
	// 4a) Ambil SNAPSHOT siswa
	// ---------------------------
	var stuSnap struct {
		Name *string `gorm:"column:name"`
		Code *string `gorm:"column:code"`
		Slug *string `gorm:"column:slug"`
	}
	if err := tx.Raw(`
		SELECT 
			NULLIF(ss.school_student_user_profile_name_snapshot, '') AS name,
			NULLIF(ss.school_student_code, '')                       AS code,
			NULLIF(ss.school_student_slug, '')                       AS slug
		FROM school_students ss
		WHERE ss.school_student_id = ? AND ss.school_student_school_id = ?
		LIMIT 1
	`, schoolStudentID, schoolID).Scan(&stuSnap).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal baca snapshot siswa: "+err.Error())
	}
	// Fallback name bila kosong → pickProfileName
	if (stuSnap.Name == nil || strings.TrimSpace(*stuSnap.Name) == "") && pickProfileName != nil {
		n := strings.TrimSpace(*pickProfileName)
		stuSnap.Name = &n
	}

	// ---------------------------
	// 4b) Ambil SNAPSHOT kelas + TERM untuk semua class_id
	// ---------------------------
	type clsSnap struct {
		ID         uuid.UUID  `gorm:"column:class_id"`
		Name       string     `gorm:"column:class_name"`
		Slug       string     `gorm:"column:class_slug"`
		TermID     *uuid.UUID `gorm:"column:class_academic_term_id"`
		TermYear   *string    `gorm:"column:class_academic_term_academic_year_snapshot"`
		TermName   *string    `gorm:"column:class_academic_term_name_snapshot"`
		TermSlug   *string    `gorm:"column:class_academic_term_slug_snapshot"`
		TermAngkat *int       `gorm:"column:term_angkatan_int"` // cast dari varchar
	}

	classIDs := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		classIDs = append(classIDs, it.ClassID)
	}
	clsMap := make(map[uuid.UUID]clsSnap, len(classIDs))
	if len(classIDs) > 0 {
		var rows []clsSnap
		if err := tx.Table("classes").
			Select(`
				class_id,
				class_name,
				class_slug,
				class_academic_term_id,
				class_academic_term_academic_year_snapshot,
				class_academic_term_name_snapshot,
				class_academic_term_slug_snapshot,
				NULLIF(class_academic_term_angkatan_snapshot,'')::int AS term_angkatan_int
			`).
			Where("class_school_id = ? AND class_id IN ?", schoolID, classIDs).
			Find(&rows).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal baca snapshot kelas: "+err.Error())
		}
		for _, r := range rows {
			clsMap[r.ID] = r
		}
	}

	// 5) Insert enrollments + isi snapshot (kelas, siswa, TERM)
	enrollIDs := make([]uuid.UUID, 0, len(items))
	perShares := make([]int64, 0, len(items))

	for idx, it := range items {
		prefs := map[string]any{
			"payer_user_id": userID,
			"registration": map[string]any{
				"fee_rule_id":            fr.ID,
				"fee_rule_option_code":   it.Code,
				"fee_rule_option_label":  it.Label,
				"fee_rule_option_amount": it.AmountIDR,
				"picked_source":          it.Source,
				"bundle_index":           idx,
				"bundle_count":           len(items),
				"category_snapshot":      "registration",
			},
			"notes": strings.TrimSpace(req.Notes),
		}
		if it.Source == "custom" && it.CustomLabel != nil {
			prefs["registration"].(map[string]any)["custom_label"] = strings.TrimSpace(*it.CustomLabel)
		}
		prefsJSON, _ := json.Marshal(prefs)

		// snapshot kelas utk item ini
		var (
			cName, cSlug *string
			tID          *uuid.UUID
			tYear        *string
			tName        *string
			tSlug        *string
			tAngkat      *int
		)
		if cs, ok := clsMap[it.ClassID]; ok {
			if s := strings.TrimSpace(cs.Name); s != "" {
				cName = &cs.Name
			}
			if s := strings.TrimSpace(cs.Slug); s != "" {
				cSlug = &cs.Slug
			}
			tID = cs.TermID
			tYear = cs.TermYear
			tName = cs.TermName
			tSlug = cs.TermSlug
			tAngkat = cs.TermAngkat
		}

		// snapshot siswa (boleh nil → NULL)
		var sName, sCode, sSlug *string
		if stuSnap.Name != nil && strings.TrimSpace(*stuSnap.Name) != "" {
			s := strings.TrimSpace(*stuSnap.Name)
			sName = &s
		}
		if stuSnap.Code != nil && strings.TrimSpace(*stuSnap.Code) != "" {
			s := strings.TrimSpace(*stuSnap.Code)
			sCode = &s
		}
		if stuSnap.Slug != nil && strings.TrimSpace(*stuSnap.Slug) != "" {
			s := strings.TrimSpace(*stuSnap.Slug)
			sSlug = &s
		}

		var eidStr string
		if err := tx.Raw(`
			INSERT INTO student_class_enrollments
			(
				student_class_enrollments_school_id,
				student_class_enrollments_school_student_id,
				student_class_enrollments_class_id,
				student_class_enrollments_status,
				student_class_enrollments_total_due_idr,
				student_class_enrollments_preferences,

				-- SNAPSHOTS (class & student)
				student_class_enrollments_class_name_snapshot,
				student_class_enrollments_class_slug_snapshot,
				student_class_enrollments_student_name_snapshot,
				student_class_enrollments_student_code_snapshot,
				student_class_enrollments_student_slug_snapshot,

				-- TERM (denormalized)
				student_class_enrollments_term_id,
				student_class_enrollments_term_academic_year_snapshot,
				student_class_enrollments_term_name_snapshot,
				student_class_enrollments_term_slug_snapshot,
				student_class_enrollments_term_angkatan_snapshot
			)
			VALUES (?, ?, ?, 'initiated', ?, ?::jsonb, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING student_class_enrollments_id
		`,
			schoolID,
			schoolStudentID,
			it.ClassID,
			it.AmountIDR,
			datatypes.JSON(prefsJSON),

			cName, cSlug,
			sName, sCode, sSlug,

			tID, tYear, tName, tSlug, tAngkat,
		).Scan(&eidStr).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "gagal membuat enrollment (mungkin duplikat aktif)")
		}

		eid, er := uuid.Parse(strings.TrimSpace(eidStr))
		if er != nil || eid == uuid.Nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal parse enrollment_id")
		}
		enrollIDs = append(enrollIDs, eid)
		perShares = append(perShares, it.AmountIDR)
	}

	// 6) Buat 1 payment (total)
	var totalAmount int64
	for _, s := range perShares {
		totalAmount += s
	}
	extID := req.PaymentExternalID
	if extID == nil || strings.TrimSpace(*extID) == "" {
		s := genOrderID("REG")
		extID = &s
	}

	meta := map[string]any{
		"fee_rule_gbk_category_snapshot": "registration",
		"fee_rule_id":                    fr.ID,
		"payer_user_id":                  userID,
		"bundle": map[string]any{
			"class_ids": func() []uuid.UUID {
				arr := make([]uuid.UUID, 0, len(items))
				for _, it := range items {
					arr = append(arr, it.ClassID)
				}
				return arr
			}(),
			"enrollment_ids":   enrollIDs,
			"per_shares_idr":   perShares,
			"total_amount_idr": totalAmount,
			"per_items": func() []map[string]any {
				out := make([]map[string]any, 0, len(items))
				for i, it := range items {
					out = append(out, map[string]any{
						"idx":        i,
						"class_id":   it.ClassID,
						"source":     it.Source,
						"code":       it.Code,
						"label":      it.Label,
						"amount_idr": it.AmountIDR,
					})
				}
				return out
			}(),
		},
	}
	if req.Customer != nil {
		meta["customer"] = req.Customer
	}
	metaJSON, _ := json.Marshal(meta)

	pm := &model.Payment{
		PaymentSchoolID:             &schoolID,
		PaymentUserID:               &userID,
		PaymentMethod:               method,
		PaymentGatewayProvider:      &provider,
		PaymentExternalID:           extID,
		PaymentAmountIDR:            int(totalAmount),
		PaymentGeneralBillingKindID: &fr.GBKID,
		PaymentMeta:                 datatypes.JSON(metaJSON),
	}
	if un, fn, em, dn, er := h.hydrateUserSnapshots(c.Context(), tx, userID); er == nil {
		pm.PaymentUserNameSnapshot = un
		if fn != nil {
			pm.PaymentFullNameSnapshot = fn
		} else if pickProfileName != nil {
			pm.PaymentFullNameSnapshot = pickProfileName
		}
		pm.PaymentEmailSnapshot = em
		pm.PaymentDonationNameSnapshot = dn
	}
	if pm.PaymentMethod == model.PaymentMethodGateway && (pm.PaymentGatewayProvider == nil || *pm.PaymentGatewayProvider == "") {
		pr := model.GatewayProviderMidtrans
		pm.PaymentGatewayProvider = &pr
	}
	if err := tx.Create(pm).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal membuat payment: "+err.Error())
	}

	// Link payment → semua enrollment
	for _, eid := range enrollIDs {
		if er := h.attachEnrollmentOnCreate(c.Context(), tx, pm, eid); er != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal link enrollment: "+er.Error())
		}
	}

	// Snap (Midtrans)
	if pm.PaymentMethod == model.PaymentMethodGateway &&
		pm.PaymentGatewayProvider != nil && *pm.PaymentGatewayProvider == model.GatewayProviderMidtrans {

		cust := svc.CustomerInput{}
		if req.Customer != nil {
			cust = *req.Customer
		} else if pm.PaymentMeta != nil {
			var tmp struct {
				Customer *svc.CustomerInput `json:"customer"`
			}
			_ = json.Unmarshal(pm.PaymentMeta, &tmp)
			if tmp.Customer != nil {
				cust = *tmp.Customer
			}
		}

		// 🔹 ambil base URL FE dari env, atau fallback ke Railway
		frontendBase := strings.TrimRight(envOrDefault("FRONTEND_BASE_URL", "https://madinahsalam.up.railway.app"), "/")

		// 🔹 ambil slug sekolah dari DB
		var schoolSlug string
		if err := tx.Raw(`
            SELECT school_slug
            FROM schools
            WHERE school_id = ? AND school_deleted_at IS NULL
            LIMIT 1
        `, schoolID).Scan(&schoolSlug).Error; err != nil || strings.TrimSpace(schoolSlug) == "" {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil slug sekolah untuk redirect pembayaran")
		}

		finishURL := fmt.Sprintf(
			"%s/%s/user/pendaftaran/selesai?payment_id=%s",
			frontendBase,
			strings.TrimSpace(schoolSlug),
			pm.PaymentID.String(),
		)

		token, redirectURL, err := svc.GenerateSnapToken(*pm, cust, finishURL)
		if err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadGateway, "midtrans error: "+err.Error())
		}

		now := time.Now()
		pm.PaymentCheckoutURL = &redirectURL
		pm.PaymentGatewayReference = &token
		pm.PaymentStatus = model.PaymentStatusPending
		pm.PaymentRequestedAt = &now

		if err := tx.Save(pm).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "update payment (snap) gagal: "+err.Error())
		}
		if er := h.applyEnrollmentSideEffects(c.Context(), tx, pm); er != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal apply efek enrollment: "+er.Error())
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "commit gagal: "+err.Error())
	}

	// ===================== Response (enrichment ringan) =====================
	enrollDTOs := make([]cendto.StudentClassEnrollmentResponse, 0, len(enrollIDs))
	for i, eid := range enrollIDs {
		dtoRow := cendto.StudentClassEnrollmentResponse{
			StudentClassEnrollmentID:              eid,
			StudentClassEnrollmentSchoolID:        schoolID,
			StudentClassEnrollmentSchoolStudentID: schoolStudentID,
			StudentClassEnrollmentClassID:         items[i].ClassID,

			StudentClassEnrollmentStatus:      cenmodel.ClassEnrollmentAwaitingPayment,
			StudentClassEnrollmentTotalDueIDR: perShares[i],
		}

		// langsung isi dari snapshot yang tadi kita ambil (konsisten dengan DB)
		if cs, ok := clsMap[items[i].ClassID]; ok {
			if strings.TrimSpace(cs.Name) != "" {
				dtoRow.StudentClassEnrollmentClassNameSnapshot = cs.Name
				dtoRow.StudentClassEnrollmentClassName = cs.Name
			}

			// slug kalau mau ikutan (field DTO bertipe *string)
			if strings.TrimSpace(cs.Slug) != "" {
				slug := cs.Slug
				dtoRow.StudentClassEnrollmentClassSlugSnapshot = &slug
			}

			// ===== TERM snapshots (baru) =====
			dtoRow.StudentClassEnrollmentTermID = cs.TermID
			dtoRow.StudentClassEnrollmentTermAcademicYearSnapshot = cs.TermYear
			dtoRow.StudentClassEnrollmentTermNameSnapshot = cs.TermName
			dtoRow.StudentClassEnrollmentTermSlugSnapshot = cs.TermSlug
			dtoRow.StudentClassEnrollmentTermAngkatanSnapshot = cs.TermAngkat
		}

		// snapshot siswa
		if stuSnap.Name != nil && strings.TrimSpace(*stuSnap.Name) != "" {
			name := strings.TrimSpace(*stuSnap.Name)
			dtoRow.StudentClassEnrollmentStudentNameSnapshot = name
			dtoRow.StudentClassEnrollmentStudentName = name
		}

		if stuSnap.Code != nil && strings.TrimSpace(*stuSnap.Code) != "" {
			code := strings.TrimSpace(*stuSnap.Code)
			dtoRow.StudentClassEnrollmentStudentCodeSnapshot = &code
		}

		if stuSnap.Slug != nil && strings.TrimSpace(*stuSnap.Slug) != "" {
			slug := strings.TrimSpace(*stuSnap.Slug)
			dtoRow.StudentClassEnrollmentStudentSlugSnapshot = &slug
		}

		enrollDTOs = append(enrollDTOs, dtoRow)
	}

	return helper.JsonCreated(c, "registration bundle + payment created", CreateRegistrationAndPaymentResponse{
		Enrollments: enrollDTOs,
		Payment:     dto.FromModel(pm),
	})
}
