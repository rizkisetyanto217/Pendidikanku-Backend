// file: internals/features/finance/payments/controller/payment_controller.go
package controller

import (
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

	// ⬇️ ganti ke package class_enrollments
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
   Handlers
======================================================================= */

func (h *PaymentController) CreatePayment(c *fiber.Ctx) error {
	var req dto.CreatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
	}
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel() // *model.PaymentModel

	// 2) Generate payment_number per sekolah (kalau belum diisi)
	if m.PaymentSchoolID != nil && *m.PaymentSchoolID != uuid.Nil &&
		(m.PaymentNumber == nil || *m.PaymentNumber == 0) {

		if num, er := svc.NextPaymentNumber(c.Context(), h.DB, *m.PaymentSchoolID); er == nil {
			m.PaymentNumber = num
		} else {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal generate payment_number: "+er.Error())
		}
	}

	// 3) Auto-hydrate user snapshots
	if m.PaymentUserID != nil {
		if un, fn, em, dn, er := svc.HydrateUserSnapshots(c.Context(), h.DB, *m.PaymentUserID); er == nil {
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

	// 4) Default provider → midtrans
	if m.PaymentMethod == model.PaymentMethodGateway && m.PaymentGatewayProvider == nil {
		prov := model.GatewayProviderMidtrans
		m.PaymentGatewayProvider = &prov
	}

	// 5) Simpan header
	if err := h.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "create payment failed: "+err.Error())
	}

	// 6) Side-effect dari meta (registration)
	if len(m.PaymentMeta) > 0 {
		meta := svc.ParseRegistrationMeta(m.PaymentMeta)
		if meta.StudentClassEnrollmentID != nil && meta.FeeRuleGBKCategory == "registration" {
			_ = svc.AttachEnrollmentOnCreate(
				c.Context(),
				h.DB,
				m,
				*meta.StudentClassEnrollmentID,
				paymentSnapshot(m),
			)
		}
	}

	// 7) Midtrans
	if m.PaymentMethod == model.PaymentMethodGateway &&
		m.PaymentGatewayProvider != nil && *m.PaymentGatewayProvider == model.GatewayProviderMidtrans {

		if m.PaymentExternalID == nil || strings.TrimSpace(*m.PaymentExternalID) == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "payment_external_id (order_id) is required for midtrans")
		}

		cust := svc.CustomerInput{}
		if len(m.PaymentMeta) > 0 {
			_ = json.Unmarshal(m.PaymentMeta, &cust)
		}

		token, redirectURL, err := svc.GenerateSnapToken(*m, cust, "")
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "midtrans error: "+err.Error())
		}

		now := time.Now()
		m.PaymentCheckoutURL = &redirectURL
		m.PaymentGatewayRef = &token
		m.PaymentStatus = model.PaymentStatusPending
		m.PaymentRequestedAt = &now

		if err := h.DB.WithContext(c.Context()).Save(m).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "update payment after snap failed: "+err.Error())
		}

		_ = svc.ApplyEnrollmentSideEffects(c.Context(), h.DB, m, paymentSnapshot(m))
	}

	// 8) Manual paid
	if m.PaymentMethod != model.PaymentMethodGateway && m.PaymentStatus == model.PaymentStatusPaid {
		_ = svc.ApplyStudentBillSideEffects(c.Context(), h.DB, m)
		_ = svc.ApplyEnrollmentSideEffects(c.Context(), h.DB, m, paymentSnapshot(m))
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

	// 🆕 PaymentModel
	var m model.PaymentModel
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

	// Side effects
	_ = svc.ApplyStudentBillSideEffects(c.Context(), h.DB, &m)
	_ = svc.ApplyEnrollmentSideEffects(c.Context(), h.DB, &m, paymentSnapshot(&m))

	return helper.JsonUpdated(c, "payment updated", dto.FromModel(&m))
}

/* =======================================================================
   Webhook Midtrans
======================================================================= */

type midtransNotif struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	FraudStatus       string `json:"fraud_status"`
	TransactionID     string `json:"transaction_id"`
	SettlementTime    string `json:"settlement_time"`

	// 🔽 Tambahan buat VA / channel / cstore dll
	Bank            string `json:"bank"`              // kadang ada langsung
	PermataVANumber string `json:"permata_va_number"` // khusus permata
	VANumbers       []struct {
		Bank     string `json:"bank"`
		VANumber string `json:"va_number"`
	} `json:"va_numbers"`

	Store       string `json:"store"`        // Indomaret/Alfamart/etc
	PaymentCode string `json:"payment_code"` // kode bayar cstore
}

func (h *PaymentController) MidtransWebhook(c *fiber.Ctx) error {
	// 1) Parse payload
	var notif midtransNotif
	if err := c.BodyParser(&notif); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid payload: "+err.Error())
	}

	// 2) Verify signature — SHA512(order_id + status_code + gross_amount + ServerKey)
	want := strings.ToLower(strings.TrimSpace(notif.SignatureKey))
	raw := notif.OrderID + notif.StatusCode + notif.GrossAmount + h.MidtransServerKey
	got := sha512sum(raw)

	fmt.Printf(
		"[MIDTRANS][WEBHOOK] order_id=%s status_code=%s gross=%s want=%s got=%s\n",
		notif.OrderID,
		notif.StatusCode,
		notif.GrossAmount,
		want,
		got,
	)

	if want == "" || got != want {
		// log event juga biar kelihatan di DB
		_ = h.logGatewayEvent(c, nil, notif, "invalid_signature", "signature mismatch")
		return helper.JsonError(c, fiber.StatusUnauthorized, "invalid signature")
	}

	// 3) Find payment by external_id (order_id)
	var p model.PaymentModel
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
	newStatus, setFields := svc.MapMidtransStatus(p.PaymentStatus, notif.TransactionStatus, notif.FraudStatus, now)

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
		p.PaymentGatewayRef = &ref
	}
	// normalisasi amount gross (string → int)
	if amt, err := strconv.ParseFloat(notif.GrossAmount, 64); err == nil {
		p.PaymentAmountIDR = int(amt + 0.5)
	}

	// ============================
	// 🆕 Isi channel/bank/VA snapshot
	// ============================

	// channel snapshot → simpan payment_type mentah dari Midtrans
	if strings.TrimSpace(notif.PaymentType) != "" {
		ch := strings.TrimSpace(notif.PaymentType)
		p.PaymentChannelSnapshot = &ch
	}

	switch strings.TrimSpace(notif.PaymentType) {
	case "bank_transfer":
		var bank, va string

		// 1) Cek di va_numbers (BCA, BNI, BRI, dll)
		if len(notif.VANumbers) > 0 {
			bank = strings.TrimSpace(notif.VANumbers[0].Bank)
			va = strings.TrimSpace(notif.VANumbers[0].VANumber)
		}

		// 2) Fallback permata_va_number
		if va == "" && strings.TrimSpace(notif.PermataVANumber) != "" {
			va = strings.TrimSpace(notif.PermataVANumber)
		}

		// 3) Fallback bank langsung
		if bank == "" && strings.TrimSpace(notif.Bank) != "" {
			bank = strings.TrimSpace(notif.Bank)
		}

		if bank != "" {
			p.PaymentBankSnapshot = &bank
		}
		if va != "" {
			p.PaymentVANumberSnapshot = &va
		}

	case "cstore":
		// contoh: Indomaret / Alfamart
		if strings.TrimSpace(notif.Store) != "" {
			store := strings.TrimSpace(notif.Store)
			p.PaymentBankSnapshot = &store // atau pakai channel snapshot khusus cstore
		}
		if strings.TrimSpace(notif.PaymentCode) != "" {
			code := strings.TrimSpace(notif.PaymentCode)
			p.PaymentVANumberSnapshot = &code
		}

	// kamu bisa tambahin case lain: "echannel", "qris", "gopay", dll kalau butuh
	default:
		// biarkan default, minimal channel sudah keisi
	}

	p.PaymentUpdatedAt = now

	if err := h.DB.WithContext(c.Context()).Save(&p).Error; err != nil {
		_ = h.updateEventStatus(notif, "failed", err.Error())
		return helper.JsonError(c, fiber.StatusInternalServerError, "update payment failed: "+err.Error())
	}

	// 7) Side effects ke student_bills & enrollment (jika ada target/meta)
	_ = svc.ApplyStudentBillSideEffects(c.Context(), h.DB, &p)
	_ = svc.ApplyEnrollmentSideEffects(c.Context(), h.DB, &p, paymentSnapshot(&p))

	_ = h.updateEventStatus(notif, "processed", "")

	return helper.JsonOK(c, "webhook processed", fiber.Map{
		"payment_id":          p.PaymentID,
		"payment_status":      p.PaymentStatus,
		"transaction_status":  notif.TransactionStatus,
		"fraud_status":        notif.FraudStatus,
		"payment_gateway_ref": p.PaymentGatewayRef,
	})
}

/* =======================================================================
   Helpers: webhook / utils
======================================================================= */

func sha512sum(s string) string {
	h := sha512.Sum512([]byte(s))
	return hex.EncodeToString(h[:])
}

func (h *PaymentController) logGatewayEvent(c *fiber.Ctx, p *model.PaymentModel, notif midtransNotif, status string, errMsg string) error {
	headers := map[string]string{}
	for k, v := range c.GetReqHeaders() { // v: []string
		headers[k] = strings.Join(v, ",")
	}

	headersJSON, _ := json.Marshal(headers)
	payloadJSON, _ := json.Marshal(notif)
	rawQuery := string(c.Request().URI().QueryString())

	now := time.Now().UTC()

	ev := model.PaymentGatewayEventModel{
		// relasi diisi di bawah kalau p != nil
		GatewayEventSchoolID:  nil,
		GatewayEventPaymentID: nil,

		GatewayEventProvider: model.GatewayProviderMidtrans,
		GatewayEventType: func() *string {
			t := strings.TrimSpace(notif.TransactionStatus)
			if t == "" {
				return nil
			}
			return &t
		}(),
		GatewayEventExternalID: func() *string {
			id := strings.TrimSpace(notif.OrderID)
			if id == "" {
				return nil
			}
			return &id
		}(),
		GatewayEventExternalRef: func() *string {
			if strings.TrimSpace(notif.TransactionID) == "" {
				return nil
			}
			id := strings.TrimSpace(notif.TransactionID)
			return &id
		}(),

		GatewayEventHeaders:   datatypes.JSON(headersJSON),
		GatewayEventPayload:   datatypes.JSON(payloadJSON),
		GatewayEventSignature: strPtr(strings.TrimSpace(notif.SignatureKey)),
		GatewayEventRawQuery:  &rawQuery,

		GatewayEventStatus: normalizeGatewayEventStatus(status),
		GatewayEventError: func() *string {
			if strings.TrimSpace(errMsg) == "" {
				return nil
			}
			e := errMsg
			return &e
		}(),
		GatewayEventTryCount: 0,

		GatewayEventReceivedAt:  now,
		GatewayEventProcessedAt: nil,

		GatewayEventCreatedAt: now,
		GatewayEventUpdatedAt: now,
	}

	// Jika payment ada, isi relasi & tenantnya
	if p != nil {
		ev.GatewayEventPaymentID = &p.PaymentID
		ev.GatewayEventSchoolID = p.PaymentSchoolID
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

func normalizeGatewayEventStatus(s string) model.GatewayEventStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", string(model.GatewayEventStatusReceived):
		return model.GatewayEventStatusReceived
	case string(model.GatewayEventStatusProcessing):
		return model.GatewayEventStatusProcessing
	case "processed", "success":
		return model.GatewayEventStatusSuccess
	case "failed", "error":
		return model.GatewayEventStatusFailed
	default:
		// fallback paling aman
		return model.GatewayEventStatusFailed
	}
}

func (h *PaymentController) updateEventStatus(notif midtransNotif, newStatus string, errMsg string) error {
	var ev model.PaymentGatewayEventModel

	// NOTE: kolom di query masih pakai nama DB lama (payment_gateway_event_*)
	// ini aman selama migration-nya memang masih pakai nama itu.
	q := h.DB.
		Where(
			"payment_gateway_event_provider = ? AND COALESCE(payment_gateway_event_external_id,'') = ? AND payment_gateway_event_deleted_at IS NULL",
			model.GatewayProviderMidtrans,
			notif.OrderID,
		).
		Order("payment_gateway_event_created_at DESC").
		Limit(1).
		First(&ev)

	if q.Error != nil {
		return q.Error
	}

	ev.GatewayEventStatus = normalizeGatewayEventStatus(newStatus)

	if strings.TrimSpace(errMsg) != "" {
		ev.GatewayEventError = strPtr(errMsg)
	}

	now := time.Now().UTC()
	ev.GatewayEventProcessedAt = &now
	ev.GatewayEventUpdatedAt = now

	return h.DB.Save(&ev).Error
}

// hasil mapping status: status target + field waktu mana yang perlu di-set
func strPtr(s string) *string { return &s }

func paymentSnapshot(p *model.PaymentModel) datatypes.JSON {
	if p == nil {
		return nil
	}
	b, _ := json.Marshal(dto.FromModel(p))
	return datatypes.JSON(b)
}

// ================= DTO & helpers (request bundle) =================

type CreateRegistrationAndPaymentResponse struct {
	Enrollments []cendto.StudentClassEnrollmentResponse `json:"enrollments"`
	Payment     any                                     `json:"payment"`
}

// Generate NIM untuk siswa baru berdasarkan term (diambil dari salah satu class)
// Format: SSSYYYYAA####
//   - SSS  = school_number (3 digit, zero-padded)
//   - YYYY = tahun awal akademik
//   - AA   = angkatan 2 digit
//   - #### = sequence per sekolah + prefix
//
// ================= HANDLER: POST /payments/registration-enroll =================
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

	// 1) Body + normalisasi items (pakai DTO)
	var req dto.CreateRegistrationAndPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid json: "+err.Error())
	}
	if err := req.NormalizeItems(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
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

	// 🔹 Ambil slug sekolah (dipakai untuk invoice & redirect Midtrans)
	var schoolSlug string
	if err := tx.Raw(`
    SELECT school_slug
      FROM schools
     WHERE school_id = ?
       AND school_deleted_at IS NULL
     LIMIT 1
`, schoolID).Scan(&schoolSlug).Error; err != nil || strings.TrimSpace(schoolSlug) == "" {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil slug sekolah")
	}
	schoolSlug = strings.TrimSpace(schoolSlug)

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
			up.user_profile_user_id        AS user_id,
			u.full_name                    AS full_name,
			u.email                        AS email,
			up.user_profile_full_name_cache AS snap_name
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
				code, er := svc.GenerateStudentCodeForClass(c.Context(), tx, schoolID, baseClassID)
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
                   SET school_student_code       = ?,
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
				code, er := svc.GenerateStudentCodeForClass(c.Context(), tx, schoolID, baseClassID)
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
            school_student_user_profile_name_cache,
            school_student_code,
            school_student_joined_at
        ) VALUES (?, ?, ?, 'active', '[]'::jsonb, ?, ?, NOW())
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
            school_student_code,
            school_student_joined_at
        ) VALUES (?, ?, ?, 'active', '[]'::jsonb, ?, NOW())
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

		// 🆕 snapshot tambahan
		Scope   *string `gorm:"column:fee_rule_scope"`
		Note    *string `gorm:"column:fee_rule_note"`
		GBKCode *string `gorm:"column:fee_rule_gbk_code_snapshot"`
	}

	var fr feeRuleHeader
	if err := tx.Raw(`
    SELECT fee_rule_id,
           fee_rule_school_id,
           fee_rule_general_billing_kind_id,
           fee_rule_gbk_category_snapshot,
           fee_rule_amount_options,
           fee_rule_scope,
           fee_rule_note,
           fee_rule_gbk_code_snapshot
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
	var opts []dto.FeeRuleAmountOption
	if len(fr.AmountOptions) > 0 {
		_ = json.Unmarshal(fr.AmountOptions, &opts)
	}
	if len(opts) == 0 {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule tidak memiliki amount_options")
	}
	minOpt := dto.MinAmountOption(opts)

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
			var chosen *dto.FeeRuleAmountOption

			// per-item override
			if it.FeeRuleOptionCode != nil && strings.TrimSpace(*it.FeeRuleOptionCode) != "" {
				chosen = dto.FindAmountOptionByCode(opts, *it.FeeRuleOptionCode)
				if chosen == nil {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule_option_code tidak valid untuk kelas "+it.ClassID.String())
				}
			} else if req.FeeRuleOptionCode != nil && strings.TrimSpace(*req.FeeRuleOptionCode) != "" {
				// global override
				chosen = dto.FindAmountOptionByCode(opts, *req.FeeRuleOptionCode)
				if chosen == nil {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule_option_code global tidak valid")
				}
			} else {
				// default dari fee_rule
				chosen = dto.FirstDefaultAmountOption(opts)
				if chosen == nil && len(opts) > 1 {
					_ = tx.Rollback()
					return helper.JsonError(c, fiber.StatusBadRequest, "fee_rule punya banyak pilihan; berikan fee_rule_option_code di item")
				}
				if chosen == nil {
					chosen = &opts[0]
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
		Name       *string `gorm:"column:name"`
		Avatar     *string `gorm:"column:avatar"`
		Wa         *string `gorm:"column:wa"`
		ParentName *string `gorm:"column:parent_name"`
		ParentWa   *string `gorm:"column:parent_wa"`
		Gender     *string `gorm:"column:gender"`
		Code       *string `gorm:"column:code"`
		Slug       *string `gorm:"column:slug"`
	}

	if err := tx.Raw(`
	SELECT 
		NULLIF(ss.school_student_user_profile_name_cache,'')              AS name,
		NULLIF(ss.school_student_user_profile_avatar_url_cache,'')       AS avatar,
		NULLIF(ss.school_student_user_profile_whatsapp_url_cache,'')     AS wa,
		NULLIF(ss.school_student_user_profile_parent_name_cache,'')      AS parent_name,
		NULLIF(ss.school_student_user_profile_parent_whatsapp_url_cache,'') AS parent_wa,
		NULLIF(ss.school_student_user_profile_gender_cache,'')           AS gender,
		NULLIF(ss.school_student_code,'')                                AS code,
		NULLIF(ss.school_student_slug,'')                                AS slug
	FROM school_students ss
	WHERE ss.school_student_id = ? AND ss.school_student_school_id = ?
	LIMIT 1
`, schoolStudentID, schoolID).Scan(&stuSnap).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal baca cache siswa: "+err.Error())
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
		ID           uuid.UUID  `gorm:"column:class_id"`
		Name         string     `gorm:"column:class_name"`
		Slug         string     `gorm:"column:class_slug"`
		TermID       *uuid.UUID `gorm:"column:class_academic_term_id"`
		TermYear     *string    `gorm:"column:class_academic_term_academic_year_cache"`
		TermName     *string    `gorm:"column:class_academic_term_name_cache"`
		TermSlug     *string    `gorm:"column:class_academic_term_slug_cache"`
		TermAngkatan *int       `gorm:"column:term_angkatan_int"` // cast dari varchar
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
				class_academic_term_academic_year_cache,
				class_academic_term_name_cache,
				class_academic_term_slug_cache,
				NULLIF(class_academic_term_angkatan_cache,'')::int AS term_angkatan_int
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
			tAngkat = cs.TermAngkatan
		}

		// snapshot siswa (boleh nil → NULL)
		var (
			uName, uAvatar, uWa, uParentName, uParentWa, uGender *string
			sCode, sSlug                                         *string
		)

		if stuSnap.Name != nil && strings.TrimSpace(*stuSnap.Name) != "" {
			s := strings.TrimSpace(*stuSnap.Name)
			uName = &s
		}
		if stuSnap.Avatar != nil && strings.TrimSpace(*stuSnap.Avatar) != "" {
			s := strings.TrimSpace(*stuSnap.Avatar)
			uAvatar = &s
		}
		if stuSnap.Wa != nil && strings.TrimSpace(*stuSnap.Wa) != "" {
			s := strings.TrimSpace(*stuSnap.Wa)
			uWa = &s
		}
		if stuSnap.ParentName != nil && strings.TrimSpace(*stuSnap.ParentName) != "" {
			s := strings.TrimSpace(*stuSnap.ParentName)
			uParentName = &s
		}
		if stuSnap.ParentWa != nil && strings.TrimSpace(*stuSnap.ParentWa) != "" {
			s := strings.TrimSpace(*stuSnap.ParentWa)
			uParentWa = &s
		}
		if stuSnap.Gender != nil && strings.TrimSpace(*stuSnap.Gender) != "" {
			s := strings.TrimSpace(*stuSnap.Gender)
			uGender = &s
		}
		if stuSnap.Code != nil && strings.TrimSpace(*stuSnap.Code) != "" {
			s := strings.TrimSpace(*stuSnap.Code)
			sCode = &s
		}
		if stuSnap.Slug != nil && strings.TrimSpace(*stuSnap.Slug) != "" {
			s := strings.TrimSpace(*stuSnap.Slug)
			sSlug = &s
		}

		// =========================================
		// 5a) LOCK kelas + cek kuota (FOR UPDATE)
		// =========================================
		var quotaRow struct {
			Total *int64 `gorm:"column:class_quota_total"`
			Taken *int64 `gorm:"column:class_quota_taken"`
		}

		if err := tx.Raw(`
			SELECT class_quota_total, class_quota_taken
			  FROM classes
			 WHERE class_id = ?
			   AND class_deleted_at IS NULL
			 FOR UPDATE
		`, it.ClassID).Scan(&quotaRow).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal baca kuota kelas: "+err.Error())
		}

		// Nama kelas buat pesan error
		classNameForErr := it.ClassID.String()
		if cs, ok := clsMap[it.ClassID]; ok && strings.TrimSpace(cs.Name) != "" {
			classNameForErr = strings.TrimSpace(cs.Name)
		}

		// Kalau punya kuota (Total != nil) → cek penuh atau belum
		if quotaRow.Total != nil {
			total := *quotaRow.Total
			taken := int64(0)
			if quotaRow.Taken != nil {
				taken = *quotaRow.Taken
			}

			if taken >= total {
				_ = tx.Rollback()
				return helper.JsonError(
					c,
					fiber.StatusBadRequest,
					fmt.Sprintf("kuota kelas %s sudah penuh", classNameForErr),
				)
			}
		}
		// Kalau Total == nil → unlimited, tapi kita tetap increment nanti

		// =========================================
		// 5b) INSERT enrollment
		// =========================================
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

		-- CACHES (class & student)
		student_class_enrollments_class_name_cache,
		student_class_enrollments_class_slug_cache,

		student_class_enrollments_user_profile_name_cache,
		student_class_enrollments_user_profile_avatar_url_cache,
		student_class_enrollments_user_profile_whatsapp_url_cache,
		student_class_enrollments_user_profile_parent_name_cache,
		student_class_enrollments_user_profile_parent_whatsapp_url_cache,
		student_class_enrollments_user_profile_gender_cache,

		student_class_enrollments_student_code_cache,
		student_class_enrollments_student_slug_cache,

		-- TERM (denormalized, cache)
		student_class_enrollments_term_id,
		student_class_enrollments_term_academic_year_cache,
		student_class_enrollments_term_name_cache,
		student_class_enrollments_term_slug_cache,
		student_class_enrollments_term_angkatan_cache
	)
	VALUES (?, ?, ?, 'initiated', ?, ?::jsonb,
	        ?, ?,         -- class caches
	        ?, ?, ?, ?, ?, ?,  -- user_profile caches
	        ?, ?,         -- student_code + slug
	        ?, ?, ?, ?, ? -- term caches
	)
	RETURNING student_class_enrollments_id
`,
			schoolID,
			schoolStudentID,
			it.ClassID,
			it.AmountIDR,
			datatypes.JSON(prefsJSON),

			cName, cSlug, // class
			uName, uAvatar, uWa, uParentName, uParentWa, uGender, // user_profile
			sCode, sSlug, // student code + slug

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

		// =========================================
		// 5c) Increment kuota_taken (selalu naik)
		// =========================================
		if err := tx.Exec(`
			UPDATE classes
			   SET class_quota_taken = COALESCE(class_quota_taken, 0) + 1,
			       class_updated_at  = NOW()
			 WHERE class_id = ?
			   AND class_deleted_at IS NULL
		`, it.ClassID).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal update kuota kelas: "+err.Error())
		}
	}

	// 6) Buat 1 payment (total)
	// 🔍 pilih satu item untuk dijadikan snapshot option (kalau cocok)
	// aturan: cuma kalau 1 kelas dan sumbernya dari opsi (bukan custom)

	var totalAmount int64
	for _, s := range perShares {
		totalAmount += s
	}
	extID := req.PaymentExternalID
	if extID == nil || strings.TrimSpace(*extID) == "" {
		s := svc.GenOrderID("REG")
		extID = &s
	}

	meta := map[string]any{
		"fee_rule_gbk_category_snapshot": "registration", // dipakai applyEnrollmentSideEffects
		// "fee_rule_id": fr.ID, // sengaja tidak dipakai lagi di meta

		"payer_user_id": userID,
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

	// 🆕 generate payment_number per sekolah (pakai TX)
	var payNum *int64
	if num, er := svc.NextPaymentNumber(c.Context(), tx, schoolID); er == nil {
		payNum = num
	} else {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal generate payment_number: "+er.Error())
	}

	pm := &model.PaymentModel{
		PaymentSchoolID: &schoolID,
		PaymentUserID:   &userID,

		PaymentAmountIDR: int(totalAmount),
		PaymentCurrency:  "IDR",

		PaymentStatus:    model.PaymentStatusInitiated,
		PaymentEntryType: model.PaymentEntryPayment,

		PaymentMeta:   datatypes.JSON(metaJSON),
		PaymentNumber: payNum, // nomor urut per sekolah
		PaymentMethod: method,
	}

	// Gateway-only fields
	if method == model.PaymentMethodGateway {
		pm.PaymentGatewayProvider = &provider // boleh kosong dulu, nanti fallback ke midtrans
		pm.PaymentExternalID = extID
	}

	if un, fn, em, dn, er := svc.HydrateUserSnapshots(c.Context(), tx, userID); er == nil {
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

	// 6a) Buat payment_items per enrollment / class
	for idx, it := range items {
		// index 1-based
		itemIdx := int16(idx + 1)

		// pointer-pointernya jangan ke loop var langsung
		schoolStudentIDCopy := schoolStudentID
		classIDCopy := it.ClassID
		enrollIDCopy := enrollIDs[idx]

		pi := &model.PaymentItemModel{
			PaymentItemID:        uuid.Nil, // biarkan default gen_random_uuid()
			PaymentItemSchoolID:  schoolID,
			PaymentItemPaymentID: pm.PaymentID,
			PaymentItemIndex:     itemIdx,

			// Target per item
			PaymentItemSchoolStudentID: &schoolStudentIDCopy,
			PaymentItemClassID:         &classIDCopy,
			PaymentItemEnrollmentID:    &enrollIDCopy,

			// Nominal per item
			PaymentItemAmountIDR: int(it.AmountIDR),

			// Fee rule snapshots (per item)
			PaymentItemFeeRuleID:            &fr.ID,
			PaymentItemFeeRuleGBKIDSnapshot: &fr.GBKID,
		}

		// scope & note snapshot (kalau ada di fee_rule)
		if fr.Scope != nil && strings.TrimSpace(*fr.Scope) != "" {
			scope := model.FeeScope(strings.TrimSpace(*fr.Scope))
			pi.PaymentItemFeeRuleScopeSnapshot = &scope
		}
		if fr.Note != nil && strings.TrimSpace(*fr.Note) != "" {
			note := strings.TrimSpace(*fr.Note)
			pi.PaymentItemFeeRuleNoteSnapshot = &note
		}

		// option snapshot (kalau source=option)
		if it.Source == "option" {
			code := it.Code
			pi.PaymentItemFeeRuleOptionCodeSnapshot = &code

			amt := int(it.AmountIDR)
			pi.PaymentItemFeeRuleAmountSnapshot = &amt

			idx1 := int16(1) // sementara 1-based; kalau nanti butuh index asli, extend struct itemResolved
			pi.PaymentItemFeeRuleOptionIndexSnapshot = &idx1
		} else {
			// custom amount → tetap boleh isi amount snapshot
			amt := int(it.AmountIDR)
			pi.PaymentItemFeeRuleAmountSnapshot = &amt
		}

		// Snapshot TERM dari kelas (kalau ada)
		if cs, ok := clsMap[it.ClassID]; ok {
			if cs.TermID != nil {
				pi.PaymentItemAcademicTermID = cs.TermID
			}
			pi.PaymentItemAcademicTermAcademicYear = cs.TermYear
			pi.PaymentItemAcademicTermName = cs.TermName
			pi.PaymentItemAcademicTermSlug = cs.TermSlug

			if cs.TermAngkatan != nil {
				angStr := fmt.Sprintf("%d", *cs.TermAngkatan)
				pi.PaymentItemAcademicTermAngkatan = &angStr
			}
		}

		// Invoice per item: due date default +2 hari
		due := time.Now().In(time.Local).AddDate(0, 0, 2)
		dateOnly := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
		pi.PaymentItemInvoiceDue = &dateOnly

		// Title & invoice title (opsional, bisa kamu refine nanti)
		if cs, ok := clsMap[it.ClassID]; ok && strings.TrimSpace(cs.Name) != "" {
			title := fmt.Sprintf("Pendaftaran %s", strings.TrimSpace(cs.Name))
			pi.PaymentItemTitle = &title

			invTitle := title
			pi.PaymentItemInvoiceTitle = &invTitle
		}

		// nomor invoice per item (ini contoh sangat sederhana)
		invNum := fmt.Sprintf(
			"INV/%s/%s/%02d",
			strings.TrimSpace(schoolSlug),
			pm.PaymentID.String()[:8],
			itemIdx,
		)
		pi.PaymentItemInvoiceNumber = &invNum

		if err := tx.Create(pi).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal membuat payment item: "+err.Error())
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
		pm.PaymentGatewayRef = &token
		pm.PaymentStatus = model.PaymentStatusPending
		pm.PaymentRequestedAt = &now

		if err := tx.Save(pm).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "update payment (snap) gagal: "+err.Error())
		}

		if er := svc.ApplyEnrollmentSideEffects(c.Context(), tx, pm, paymentSnapshot(pm)); er != nil {
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
				dtoRow.StudentClassEnrollmentClassNameCache = cs.Name
				dtoRow.StudentClassEnrollmentClassName = cs.Name
			}

			// slug kalau mau ikutan (field DTO bertipe *string)
			if strings.TrimSpace(cs.Slug) != "" {
				slug := cs.Slug
				dtoRow.StudentClassEnrollmentClassSlugCache = &slug
			}

			// ===== TERM snapshots (baru) =====
			dtoRow.StudentClassEnrollmentTermID = cs.TermID
			dtoRow.StudentClassEnrollmentTermAcademicYearCache = cs.TermYear
			dtoRow.StudentClassEnrollmentTermNameCache = cs.TermName
			dtoRow.StudentClassEnrollmentTermSlugCache = cs.TermSlug
			dtoRow.StudentClassEnrollmentTermAngkatanCache = cs.TermAngkatan
		}

		// snapshot siswa
		if stuSnap.Name != nil && strings.TrimSpace(*stuSnap.Name) != "" {
			name := strings.TrimSpace(*stuSnap.Name)
			dtoRow.StudentClassEnrollmentUserProfileNameCache = name
			dtoRow.StudentClassEnrollmentStudentName = name
		}

		if stuSnap.Code != nil && strings.TrimSpace(*stuSnap.Code) != "" {
			code := strings.TrimSpace(*stuSnap.Code)
			dtoRow.StudentClassEnrollmentStudentCodeCache = &code
		}

		if stuSnap.Slug != nil && strings.TrimSpace(*stuSnap.Slug) != "" {
			slug := strings.TrimSpace(*stuSnap.Slug)
			dtoRow.StudentClassEnrollmentStudentSlugCache = &slug
		}

		enrollDTOs = append(enrollDTOs, dtoRow)
	}

	return helper.JsonCreated(c, "registration bundle + payment created", CreateRegistrationAndPaymentResponse{
		Enrollments: enrollDTOs,
		Payment:     dto.FromModel(pm),
	})
}
