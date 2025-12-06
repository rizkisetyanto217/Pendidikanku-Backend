package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	model "madinahsalam_backend/internals/features/finance/payments/model"
)

// TargetInfo describes the resolved payment target (bill, general billing, or kind).
type TargetInfo struct {
	Kind             string     // "student_bill" | "general_billing" | "kind"
	SchoolID         *uuid.UUID // dapat NULL untuk GLOBAL kind
	AmountSuggestion *int       // nominal saran dari target (boleh nil)
	PayerUserID      *uuid.UUID // payer default jika tersedia
}

// ResolveTarget memastikan target pembayaran valid serta mengembalikan metadata penting.
func ResolveTarget(
	ctx context.Context,
	db *gorm.DB,
	studentBillID,
	generalBillingID,
	generalBillingKindID *uuid.UUID,
) (TargetInfo, error) {
	var ti TargetInfo

	switch {
	case studentBillID != nil:
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
			Where("student_bill_id = ? AND student_bill_deleted_at IS NULL", *studentBillID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "student_bill tidak ditemukan")
		}
		ti = TargetInfo{
			Kind:             "student_bill",
			SchoolID:         &row.SchoolID,
			AmountSuggestion: &row.Amount,
			PayerUserID:      row.PayerUID,
		}

	case generalBillingID != nil:
		type gbRow struct {
			ID       uuid.UUID `gorm:"column:general_billing_id"`
			SchoolID uuid.UUID `gorm:"column:general_billing_school_id"`
			Default  *int      `gorm:"column:general_billing_default_amount_idr"`
		}
		var row gbRow
		if err := db.WithContext(ctx).
			Table("general_billings").
			Select("general_billing_id, general_billing_school_id, general_billing_default_amount_idr").
			Where("general_billing_id = ? AND general_billing_deleted_at IS NULL", *generalBillingID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "general_billing tidak ditemukan")
		}
		ti = TargetInfo{
			Kind:             "general_billing",
			SchoolID:         &row.SchoolID,
			AmountSuggestion: row.Default,
		}

	case generalBillingKindID != nil:
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
			Where("general_billing_kind_id = ? AND general_billing_kind_deleted_at IS NULL", *generalBillingKindID).
			Take(&row).Error; err != nil {
			return ti, fiber.NewError(fiber.StatusNotFound, "general_billing_kind tidak ditemukan")
		}
		if !row.Active {
			return ti, fiber.NewError(fiber.StatusBadRequest, "general_billing_kind tidak aktif")
		}
		ti = TargetInfo{
			Kind:             "kind",
			SchoolID:         row.SchoolID,
			AmountSuggestion: row.Default,
		}

	default:
		return ti, fiber.NewError(fiber.StatusBadRequest, "wajib menyertakan salah satu target: payment_student_bill_id / payment_general_billing_id / payment_general_billing_kind_id")
	}

	return ti, nil
}

// HydrateUserSnapshots mengambil snapshot nama/email/donation dari tabel users dan profiles.
func HydrateUserSnapshots(ctx context.Context, db *gorm.DB, userID uuid.UUID) (userName, fullName, email, donationName *string, err error) {
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
	if err = db.WithContext(ctx).Raw(q, userID).Scan(&row).Error; err != nil {
		return nil, nil, nil, nil, err
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

// RegistrationMeta memudahkan akses metadata registrasi dalam payment_meta.
type RegistrationMeta struct {
	StudentClassEnrollmentID *uuid.UUID `json:"student_class_enrollments_id"`
	FeeRuleGBKCategory       string     `json:"fee_rule_gbk_category_snapshot"`

	FeeRuleID           *uuid.UUID `json:"fee_rule_id"`
	FeeRuleOptionCode   *string    `json:"fee_rule_option_code"`
	FeeRuleOptionLabel  *string    `json:"fee_rule_option_label"`
	FeeRuleOptionAmount *int64     `json:"fee_rule_option_amount_idr"`

	PayerUserID *uuid.UUID `json:"payer_user_id"`
}

// ParseRegistrationMeta membaca json meta registrasi dan menormalkan kategorinya.
func ParseRegistrationMeta(j datatypes.JSON) RegistrationMeta {
	var m RegistrationMeta
	_ = json.Unmarshal(j, &m)
	m.FeeRuleGBKCategory = strings.ToLower(strings.TrimSpace(m.FeeRuleGBKCategory))
	return m
}

func buildEnrollmentPrefPatch(payer *uuid.UUID, meta RegistrationMeta) datatypes.JSON {
	payload := map[string]interface{}{}
	reg := map[string]interface{}{}

	if meta.FeeRuleID != nil {
		reg["fee_rule_id"] = meta.FeeRuleID
	}
	if meta.FeeRuleOptionCode != nil {
		reg["fee_rule_option_code"] = meta.FeeRuleOptionCode
	}
	if meta.FeeRuleOptionLabel != nil {
		reg["fee_rule_option_label"] = meta.FeeRuleOptionLabel
	}
	if meta.FeeRuleOptionAmount != nil {
		reg["fee_rule_option_amount"] = meta.FeeRuleOptionAmount
	}

	if len(reg) > 0 {
		if strings.TrimSpace(meta.FeeRuleGBKCategory) != "" {
			reg["category_snapshot"] = meta.FeeRuleGBKCategory
		}
		payload["registration"] = reg
	}

	if payer != nil {
		payload["payer_user_id"] = payer
	}

	b, _ := json.Marshal(payload)
	return datatypes.JSON(b)
}

type bundleMeta struct {
	EnrollmentIDs []uuid.UUID `json:"enrollment_ids"`
}

func extractEnrollmentIDs(j datatypes.JSON) []uuid.UUID {
	ids := []uuid.UUID{}

	var r RegistrationMeta
	_ = json.Unmarshal(j, &r)
	if r.StudentClassEnrollmentID != nil {
		ids = append(ids, *r.StudentClassEnrollmentID)
	}

	var b struct {
		Bundle bundleMeta `json:"bundle"`
	}
	if err := json.Unmarshal(j, &b); err == nil && len(b.Bundle.EnrollmentIDs) > 0 {
		ids = append(ids, b.Bundle.EnrollmentIDs...)
	}

	return ids
}

// AttachEnrollmentOnCreate mengikat payment baru dengan enrollment registrasi.
func AttachEnrollmentOnCreate(
	ctx context.Context,
	db *gorm.DB,
	p *model.Payment,
	enrollmentID uuid.UUID,
	paymentSnapshot datatypes.JSON,
) error {
	meta := RegistrationMeta{}
	if p.PaymentMeta != nil {
		meta = ParseRegistrationMeta(p.PaymentMeta)
	}

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
	`, p.PaymentID, paymentSnapshot, prefPatch, p.PaymentAmountIDR, enrollmentID).Error
}

// ApplyEnrollmentSideEffects menyinkronkan status enrollment berdasarkan status payment.
func ApplyEnrollmentSideEffects(ctx context.Context, db *gorm.DB, p *model.Payment, paymentSnapshot datatypes.JSON) error {
	if p == nil || p.PaymentMeta == nil {
		return nil
	}

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

	meta := ParseRegistrationMeta(p.PaymentMeta)
	payer := meta.PayerUserID
	if payer == nil && p.PaymentUserID != nil {
		payer = p.PaymentUserID
	}
	prefPatch := buildEnrollmentPrefPatch(payer, meta)

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
			`, p.PaymentID, paymentSnapshot, prefPatch, p.PaymentAmountIDR, eid).Error; err != nil {
				return err
			}
		}

	case model.PaymentStatusCanceled,
		model.PaymentStatusFailed,
		model.PaymentStatusExpired,
		model.PaymentStatusRefunded,
		model.PaymentStatusPartiallyRefunded:
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

	default:
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
			`, p.PaymentID, paymentSnapshot, prefPatch, p.PaymentAmountIDR, eid).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// ApplyStudentBillSideEffects menyelaraskan status student_bills dengan payment.
func ApplyStudentBillSideEffects(ctx context.Context, db *gorm.DB, p *model.Payment) error {
	if p == nil || p.PaymentStudentBillID == nil {
		return nil
	}

	switch p.PaymentStatus {
	case model.PaymentStatusPaid:
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

	case model.PaymentStatusCanceled,
		model.PaymentStatusFailed,
		model.PaymentStatusExpired,
		model.PaymentStatusRefunded:
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

// NextPaymentNumber menghasilkan nomor pembayaran incremental per sekolah.
func NextPaymentNumber(ctx context.Context, db *gorm.DB, schoolID uuid.UUID) (*int64, error) {
	if schoolID == uuid.Nil {
		return nil, nil
	}

	var next int64
	if err := db.WithContext(ctx).Raw(`
		SELECT COALESCE(MAX(payment_number), 0) + 1
		FROM payments
		WHERE payment_school_id = ?
	`, schoolID).Scan(&next).Error; err != nil {
		return nil, err
	}

	return &next, nil
}

// MappedFields menyimpan field waktu yang perlu diperbarui saat map status Midtrans.
type MappedFields struct {
	PaidAt     *time.Time
	CanceledAt *time.Time
	FailedAt   *time.Time
	RefundedAt *time.Time
}

// MapMidtransStatus mengonversi status Midtrans menjadi status internal.
func MapMidtransStatus(current model.PaymentStatus, transactionStatus, fraudStatus string, now time.Time) (model.PaymentStatus, MappedFields) {
	ts := strings.ToLower(transactionStatus)
	fraud := strings.ToLower(fraudStatus)

	switch ts {
	case "capture":
		if fraud == "accept" {
			return model.PaymentStatusPaid, MappedFields{PaidAt: &now}
		}
		if fraud == "challenge" {
			return model.PaymentStatusAwaitingCallback, MappedFields{}
		}
		return model.PaymentStatusFailed, MappedFields{FailedAt: &now}

	case "settlement":
		return model.PaymentStatusPaid, MappedFields{PaidAt: &now}

	case "pending":
		return model.PaymentStatusPending, MappedFields{}

	case "deny":
		return model.PaymentStatusFailed, MappedFields{FailedAt: &now}

	case "cancel":
		return model.PaymentStatusCanceled, MappedFields{CanceledAt: &now}

	case "expire":
		return model.PaymentStatusExpired, MappedFields{}

	case "refund":
		return model.PaymentStatusRefunded, MappedFields{RefundedAt: &now}

	case "partial_refund":
		return model.PaymentStatusPartiallyRefunded, MappedFields{RefundedAt: &now}

	case "failure":
		return model.PaymentStatusFailed, MappedFields{FailedAt: &now}
	}

	return current, MappedFields{}
}

// GenerateStudentCodeForClass membentuk kode siswa berdasarkan snapshot kelas & sekolah.
func GenerateStudentCodeForClass(ctx context.Context, tx *gorm.DB, schoolID uuid.UUID, classID uuid.UUID) (string, error) {
	if classID == uuid.Nil {
		return "", fmt.Errorf("class_id kosong saat generate NIM")
	}

	var schoolNumber int64
	if err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(school_number, 0)
		FROM schools
		WHERE school_id = ?
		  AND school_deleted_at IS NULL
		LIMIT 1
	`, schoolID).Scan(&schoolNumber).Error; err != nil {
		return "", fmt.Errorf("gagal ambil school_number: %w", err)
	}
	if schoolNumber < 0 {
		schoolNumber = 0
	}
	schoolNumStr := fmt.Sprintf("%03d", schoolNumber)

	var row struct {
		YearRaw     *string `gorm:"column:year"`
		AngkatanRaw *string `gorm:"column:angkatan"`
	}
	if err := tx.WithContext(ctx).Raw(`
		SELECT 
			NULLIF(class_academic_term_academic_year_cache,'') AS year,
			NULLIF(class_academic_term_angkatan_cache,'')      AS angkatan
		FROM classes
		WHERE class_id = ?
		  AND class_school_id = ?
		  AND class_deleted_at IS NULL
		LIMIT 1
	`, classID, schoolID).Scan(&row).Error; err != nil {
		return "", fmt.Errorf("gagal select term snapshot: %w", err)
	}

	var year4 string
	if row.YearRaw != nil && strings.TrimSpace(*row.YearRaw) != "" {
		y := strings.TrimSpace(*row.YearRaw)
		if len(y) >= 4 {
			year4 = y[:4]
		} else {
			year4 = fmt.Sprintf("%04d", time.Now().Year())
		}
	} else {
		year4 = fmt.Sprintf("%04d", time.Now().Year())
	}

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

	prefix := fmt.Sprintf("%s%s%02d", schoolNumStr, year4, angkatanInt)

	var lastSeq int
	if err := tx.WithContext(ctx).Raw(`
		SELECT COALESCE(MAX(RIGHT(school_student_code, 4)::int), 0)
		FROM school_students
		WHERE school_student_school_id = ?
		  AND school_student_code IS NOT NULL
		  AND school_student_code ~ '^[0-9]+$'
	`, schoolID).Scan(&lastSeq).Error; err != nil {
		return "", fmt.Errorf("gagal hitung sequence NIM: %w", err)
	}

	next := lastSeq + 1
	code := fmt.Sprintf("%s%04d", prefix, next)
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("kode hasil generate kosong (prefix=%s, lastSeq=%d)", prefix, lastSeq)
	}

	return code, nil
}

// GenOrderID membuat order_id dengan prefix tertentu (dipakai di Midtrans).
func GenOrderID(prefix string) string {
	now := time.Now().In(time.Local).Format("20060102-150405")
	u := uuid.New().String()
	if len(u) > 8 {
		u = u[:8]
	}
	return prefix + "-" + now + "-" + strings.ToUpper(u)
}
