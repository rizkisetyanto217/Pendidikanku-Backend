// file: internals/features/finance/payments/dto/payment_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "masjidku_backend/internals/features/finance/payments/model"
)

/* =========================================================
   PatchField tri-state (Unset / Null / Set(value))
========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"-"`
	Null  bool `json:"-"`
	Value *T   `json:"-"`
}

func (p *PatchField[T]) UnmarshalJSON(b []byte) error {
	p.Set = true
	if string(b) == "null" {
		p.Null = true
		p.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

func Ptr[T any](v T) *T { return &v }

// helpers
func applyPtr[T any](dst **T, f PatchField[T]) {
	if f.Set {
		if f.Null {
			*dst = nil
		} else {
			*dst = f.Value
		}
	}
}
func applyVal[T any](dst *T, f PatchField[T]) {
	if f.Set && !f.Null && f.Value != nil {
		*dst = *f.Value
	}
}

/* =========================================================
   REQUEST: CreatePayment
========================================================= */

type CreatePaymentRequest struct {
	PaymentMasjidID *uuid.UUID `json:"payment_masjid_id" validate:"required"`
	PaymentUserID   *uuid.UUID `json:"payment_user_id"`

	// Target (minimal salah satu WAJIB)
	PaymentStudentBillID        *uuid.UUID `json:"payment_student_bill_id"`
	PaymentGeneralBillingID     *uuid.UUID `json:"payment_general_billing_id"`
	PaymentGeneralBillingKindID *uuid.UUID `json:"payment_general_billing_kind_id"`

	// Konteks (opsional, untuk reporting SPP batch)
	PaymentBillBatchID *uuid.UUID `json:"payment_bill_batch_id"`

	// Nominal
	PaymentAmountIDR int    `json:"payment_amount_idr" validate:"required,min=0"`
	PaymentCurrency  string `json:"payment_currency"    validate:"omitempty,oneof=IDR"`

	// Status & metode (opsional)
	PaymentStatus *model.PaymentStatus `json:"payment_status" validate:"omitempty,oneof=initiated pending awaiting_callback paid partially_refunded refunded failed canceled expired"`
	PaymentMethod *model.PaymentMethod `json:"payment_method" validate:"omitempty,oneof=gateway bank_transfer cash qris other"`

	// Info gateway (opsional kalau manual)
	PaymentGatewayProvider  *model.PaymentGatewayProvider `json:"payment_gateway_provider" validate:"omitempty,oneof=midtrans xendit tripay duitku nicepay stripe paypal other"`
	PaymentExternalID       *string                       `json:"payment_external_id"`
	PaymentGatewayReference *string                       `json:"payment_gateway_reference"`
	PaymentCheckoutURL      *string                       `json:"payment_checkout_url"`
	PaymentQRString         *string                       `json:"payment_qr_string"`
	PaymentSignature        *string                       `json:"payment_signature"`
	PaymentIdempotencyKey   *string                       `json:"payment_idempotency_key"`

	// Timestamps
	PaymentRequestedAt *time.Time `json:"payment_requested_at"`
	PaymentExpiresAt   *time.Time `json:"payment_expires_at"`
	PaymentPaidAt      *time.Time `json:"payment_paid_at"`
	PaymentCanceledAt  *time.Time `json:"payment_canceled_at"`
	PaymentFailedAt    *time.Time `json:"payment_failed_at"`
	PaymentRefundedAt  *time.Time `json:"payment_refunded_at"`

	// Manual ops
	PaymentManualChannel          *string    `json:"payment_manual_channel" validate:"omitempty,max=32"`
	PaymentManualReference        *string    `json:"payment_manual_reference" validate:"omitempty,max=120"`
	PaymentManualReceivedByUserID *uuid.UUID `json:"payment_manual_received_by_user_id"`
	PaymentManualVerifiedByUserID *uuid.UUID `json:"payment_manual_verified_by_user_id"`
	PaymentManualVerifiedAt       *time.Time `json:"payment_manual_verified_at"`

	// Ledger & invoice (opsional)
	PaymentEntryType *model.PaymentEntryType `json:"payment_entry_type" validate:"omitempty,oneof=charge payment refund adjustment"`
	InvoiceNumber    *string                 `json:"invoice_number"`
	InvoiceDueDate   *time.Time              `json:"invoice_due_date"`
	InvoiceTitle     *string                 `json:"invoice_title"`

	// Subject (opsional)
	PaymentSubjectUserID    *uuid.UUID `json:"payment_subject_user_id"`
	PaymentSubjectStudentID *uuid.UUID `json:"payment_subject_student_id"`

	// Meta
	PaymentDescription *string        `json:"payment_description"`
	PaymentNote        *string        `json:"payment_note"`
	PaymentMeta        datatypes.JSON `json:"payment_meta"`
	PaymentAttachments datatypes.JSON `json:"payment_attachments"`
}

func (r *CreatePaymentRequest) Validate() error {
	// Minimal salah satu target harus terisi
	hasTarget := r.PaymentStudentBillID != nil ||
		r.PaymentGeneralBillingID != nil ||
		r.PaymentGeneralBillingKindID != nil
	if !hasTarget {
		return errors.New("wajib menyertakan salah satu target: payment_student_bill_id / payment_general_billing_id / payment_general_billing_kind_id")
	}

	// Konsistensi method vs provider
	method := model.PaymentMethodGateway
	if r.PaymentMethod != nil {
		method = *r.PaymentMethod
	}
	if method == model.PaymentMethodGateway && r.PaymentGatewayProvider == nil {
		return errors.New("payment_method=gateway harus menyertakan payment_gateway_provider")
	}
	if method != model.PaymentMethodGateway && r.PaymentGatewayProvider != nil {
		return errors.New("payment_method manual ('cash','bank_transfer','qris','other') tidak boleh menyertakan payment_gateway_provider")
	}

	// Mata uang
	if r.PaymentCurrency != "" && r.PaymentCurrency != "IDR" {
		return fmt.Errorf("payment_currency hanya mendukung 'IDR'")
	}

	return nil
}

func (r *CreatePaymentRequest) ToModel() *model.Payment {
	now := time.Now()

	out := &model.Payment{
		PaymentMasjidID: r.PaymentMasjidID,
		PaymentUserID:   r.PaymentUserID,

		PaymentStudentBillID:        r.PaymentStudentBillID,
		PaymentGeneralBillingID:     r.PaymentGeneralBillingID,
		PaymentGeneralBillingKindID: r.PaymentGeneralBillingKindID,
		PaymentBillBatchID:          r.PaymentBillBatchID,

		PaymentAmountIDR: r.PaymentAmountIDR,
		PaymentCurrency:  "IDR",

		PaymentStatus: model.PaymentStatusInitiated,
		PaymentMethod: model.PaymentMethodGateway,

		PaymentGatewayProvider:  r.PaymentGatewayProvider,
		PaymentExternalID:       r.PaymentExternalID,
		PaymentGatewayReference: r.PaymentGatewayReference,
		PaymentCheckoutURL:      r.PaymentCheckoutURL,
		PaymentQRString:         r.PaymentQRString,
		PaymentSignature:        r.PaymentSignature,
		PaymentIdempotencyKey:   r.PaymentIdempotencyKey,

		PaymentRequestedAt: r.PaymentRequestedAt,
		PaymentExpiresAt:   r.PaymentExpiresAt,
		PaymentPaidAt:      r.PaymentPaidAt,
		PaymentCanceledAt:  r.PaymentCanceledAt,
		PaymentFailedAt:    r.PaymentFailedAt,
		PaymentRefundedAt:  r.PaymentRefundedAt,

		PaymentManualChannel:          r.PaymentManualChannel,
		PaymentManualReference:        r.PaymentManualReference,
		PaymentManualReceivedByUserID: r.PaymentManualReceivedByUserID,
		PaymentManualVerifiedByUserID: r.PaymentManualVerifiedByUserID,
		PaymentManualVerifiedAt:       r.PaymentManualVerifiedAt,

		PaymentEntryType: model.PaymentEntryPayment,
		InvoiceNumber:    r.InvoiceNumber,
		InvoiceDueDate:   r.InvoiceDueDate,
		InvoiceTitle:     r.InvoiceTitle,

		PaymentSubjectUserID:    r.PaymentSubjectUserID,
		PaymentSubjectStudentID: r.PaymentSubjectStudentID,

		PaymentDescription: r.PaymentDescription,
		PaymentNote:        r.PaymentNote,
		PaymentMeta:        r.PaymentMeta,
		PaymentAttachments: r.PaymentAttachments,

		PaymentCreatedAt: now,
		PaymentUpdatedAt: now,
	}

	if r.PaymentCurrency != "" {
		out.PaymentCurrency = r.PaymentCurrency
	}
	if r.PaymentStatus != nil {
		out.PaymentStatus = *r.PaymentStatus
	}
	if r.PaymentMethod != nil {
		out.PaymentMethod = *r.PaymentMethod
	}
	if r.PaymentEntryType != nil {
		out.PaymentEntryType = *r.PaymentEntryType
	}

	return out
}

/* =========================================================
   REQUEST: Update (PATCH)
========================================================= */

type UpdatePaymentRequest struct {
	PaymentMasjidID PatchField[uuid.UUID] `json:"payment_masjid_id"`
	PaymentUserID   PatchField[uuid.UUID] `json:"payment_user_id"`

	PaymentStudentBillID        PatchField[uuid.UUID] `json:"payment_student_bill_id"`
	PaymentGeneralBillingID     PatchField[uuid.UUID] `json:"payment_general_billing_id"`
	PaymentGeneralBillingKindID PatchField[uuid.UUID] `json:"payment_general_billing_kind_id"`
	PaymentBillBatchID          PatchField[uuid.UUID] `json:"payment_bill_batch_id"`

	PaymentAmountIDR PatchField[int]    `json:"payment_amount_idr"`
	PaymentCurrency  PatchField[string] `json:"payment_currency"`

	PaymentStatus PatchField[model.PaymentStatus] `json:"payment_status"`
	PaymentMethod PatchField[model.PaymentMethod] `json:"payment_method"`

	PaymentGatewayProvider  PatchField[model.PaymentGatewayProvider] `json:"payment_gateway_provider"`
	PaymentExternalID       PatchField[string]                       `json:"payment_external_id"`
	PaymentGatewayReference PatchField[string]                       `json:"payment_gateway_reference"`
	PaymentCheckoutURL      PatchField[string]                       `json:"payment_checkout_url"`
	PaymentQRString         PatchField[string]                       `json:"payment_qr_string"`
	PaymentSignature        PatchField[string]                       `json:"payment_signature"`
	PaymentIdempotencyKey   PatchField[string]                       `json:"payment_idempotency_key"`

	PaymentRequestedAt PatchField[time.Time] `json:"payment_requested_at"`
	PaymentExpiresAt   PatchField[time.Time] `json:"payment_expires_at"`
	PaymentPaidAt      PatchField[time.Time] `json:"payment_paid_at"`
	PaymentCanceledAt  PatchField[time.Time] `json:"payment_canceled_at"`
	PaymentFailedAt    PatchField[time.Time] `json:"payment_failed_at"`
	PaymentRefundedAt  PatchField[time.Time] `json:"payment_refunded_at"`

	PaymentManualChannel          PatchField[string]    `json:"payment_manual_channel"`
	PaymentManualReference        PatchField[string]    `json:"payment_manual_reference"`
	PaymentManualReceivedByUserID PatchField[uuid.UUID] `json:"payment_manual_received_by_user_id"`
	PaymentManualVerifiedByUserID PatchField[uuid.UUID] `json:"payment_manual_verified_by_user_id"`
	PaymentManualVerifiedAt       PatchField[time.Time] `json:"payment_manual_verified_at"`

	PaymentEntryType PatchField[model.PaymentEntryType] `json:"payment_entry_type"`
	InvoiceNumber    PatchField[string]                 `json:"invoice_number"`
	InvoiceDueDate   PatchField[time.Time]              `json:"invoice_due_date"`
	InvoiceTitle     PatchField[string]                 `json:"invoice_title"`

	PaymentSubjectUserID    PatchField[uuid.UUID] `json:"payment_subject_user_id"`
	PaymentSubjectStudentID PatchField[uuid.UUID] `json:"payment_subject_student_id"`

	PaymentDescription PatchField[string]         `json:"payment_description"`
	PaymentNote        PatchField[string]         `json:"payment_note"`
	PaymentMeta        PatchField[datatypes.JSON] `json:"payment_meta"`
	PaymentAttachments PatchField[datatypes.JSON] `json:"payment_attachments"`
}

func (p *UpdatePaymentRequest) Apply(m *model.Payment) error {
	// pointers
	applyPtr(&m.PaymentMasjidID, p.PaymentMasjidID)
	applyPtr(&m.PaymentUserID, p.PaymentUserID)

	applyPtr(&m.PaymentStudentBillID, p.PaymentStudentBillID)
	applyPtr(&m.PaymentGeneralBillingID, p.PaymentGeneralBillingID)
	applyPtr(&m.PaymentGeneralBillingKindID, p.PaymentGeneralBillingKindID)
	applyPtr(&m.PaymentBillBatchID, p.PaymentBillBatchID)

	// scalars
	if p.PaymentAmountIDR.Set && !p.PaymentAmountIDR.Null && p.PaymentAmountIDR.Value != nil {
		if *p.PaymentAmountIDR.Value < 0 {
			return errors.New("payment_amount_idr tidak boleh negatif")
		}
		m.PaymentAmountIDR = *p.PaymentAmountIDR.Value
	}
	if p.PaymentCurrency.Set {
		if p.PaymentCurrency.Null || p.PaymentCurrency.Value == nil {
			return errors.New("payment_currency tidak boleh null")
		}
		if *p.PaymentCurrency.Value != "IDR" {
			return errors.New("payment_currency hanya mendukung 'IDR'")
		}
		m.PaymentCurrency = *p.PaymentCurrency.Value
	}

	// enums
	if p.PaymentMethod.Set {
		if p.PaymentMethod.Null || p.PaymentMethod.Value == nil {
			return errors.New("payment_method tidak boleh null")
		}
		m.PaymentMethod = *p.PaymentMethod.Value
	}
	if p.PaymentStatus.Set {
		if p.PaymentStatus.Null || p.PaymentStatus.Value == nil {
			return errors.New("payment_status tidak boleh null")
		}
		m.PaymentStatus = *p.PaymentStatus.Value
	}
	if p.PaymentEntryType.Set {
		if p.PaymentEntryType.Null || p.PaymentEntryType.Value == nil {
			return errors.New("payment_entry_type tidak boleh null")
		}
		m.PaymentEntryType = *p.PaymentEntryType.Value
	}

	// gateway info
	applyPtr(&m.PaymentGatewayProvider, p.PaymentGatewayProvider)
	applyPtr(&m.PaymentExternalID, p.PaymentExternalID)
	applyPtr(&m.PaymentGatewayReference, p.PaymentGatewayReference)
	applyPtr(&m.PaymentCheckoutURL, p.PaymentCheckoutURL)
	applyPtr(&m.PaymentQRString, p.PaymentQRString)
	applyPtr(&m.PaymentSignature, p.PaymentSignature)
	applyPtr(&m.PaymentIdempotencyKey, p.PaymentIdempotencyKey)

	// timestamps
	applyPtr(&m.PaymentRequestedAt, p.PaymentRequestedAt)
	applyPtr(&m.PaymentExpiresAt, p.PaymentExpiresAt)
	applyPtr(&m.PaymentPaidAt, p.PaymentPaidAt)
	applyPtr(&m.PaymentCanceledAt, p.PaymentCanceledAt)
	applyPtr(&m.PaymentFailedAt, p.PaymentFailedAt)
	applyPtr(&m.PaymentRefundedAt, p.PaymentRefundedAt)

	// manual ops
	applyPtr(&m.PaymentManualChannel, p.PaymentManualChannel)
	applyPtr(&m.PaymentManualReference, p.PaymentManualReference)
	applyPtr(&m.PaymentManualReceivedByUserID, p.PaymentManualReceivedByUserID)
	applyPtr(&m.PaymentManualVerifiedByUserID, p.PaymentManualVerifiedByUserID)
	applyPtr(&m.PaymentManualVerifiedAt, p.PaymentManualVerifiedAt)

	// invoice & subject
	applyPtr(&m.InvoiceNumber, p.InvoiceNumber)
	applyPtr(&m.InvoiceDueDate, p.InvoiceDueDate)
	applyPtr(&m.InvoiceTitle, p.InvoiceTitle)
	applyPtr(&m.PaymentSubjectUserID, p.PaymentSubjectUserID)
	applyPtr(&m.PaymentSubjectStudentID, p.PaymentSubjectStudentID)

	// meta
	applyPtr(&m.PaymentDescription, p.PaymentDescription)
	applyPtr(&m.PaymentNote, p.PaymentNote)
	applyVal(&m.PaymentMeta, p.PaymentMeta)
	applyVal(&m.PaymentAttachments, p.PaymentAttachments)

	// konsistensi method/provider
	if p.PaymentMethod.Set || p.PaymentGatewayProvider.Set {
		if m.PaymentMethod == model.PaymentMethodGateway && m.PaymentGatewayProvider == nil {
			return errors.New("payment_method=gateway harus menyertakan payment_gateway_provider")
		}
		if m.PaymentMethod != model.PaymentMethodGateway && m.PaymentGatewayProvider != nil {
			return errors.New("payment_method manual ('cash','bank_transfer','qris','other') tidak boleh menyertakan payment_gateway_provider")
		}
	}

	return nil
}

/* =========================================================
   RESPONSE
========================================================= */

type PaymentResponse struct {
	PaymentID uuid.UUID `json:"payment_id"`

	PaymentMasjidID *uuid.UUID `json:"payment_masjid_id"`
	PaymentUserID   *uuid.UUID `json:"payment_user_id"`

	PaymentStudentBillID        *uuid.UUID `json:"payment_student_bill_id"`
	PaymentGeneralBillingID     *uuid.UUID `json:"payment_general_billing_id"`
	PaymentGeneralBillingKindID *uuid.UUID `json:"payment_general_billing_kind_id"`
	PaymentBillBatchID          *uuid.UUID `json:"payment_bill_batch_id"`

	PaymentAmountIDR int    `json:"payment_amount_idr"`
	PaymentCurrency  string `json:"payment_currency"`

	PaymentStatus model.PaymentStatus `json:"payment_status"`
	PaymentMethod model.PaymentMethod `json:"payment_method"`

	PaymentGatewayProvider  *model.PaymentGatewayProvider `json:"payment_gateway_provider"`
	PaymentExternalID       *string                       `json:"payment_external_id"`
	PaymentGatewayReference *string                       `json:"payment_gateway_reference"`
	PaymentCheckoutURL      *string                       `json:"payment_checkout_url"`
	PaymentQRString         *string                       `json:"payment_qr_string"`
	PaymentSignature        *string                       `json:"payment_signature"`
	PaymentIdempotencyKey   *string                       `json:"payment_idempotency_key"`

	PaymentRequestedAt *time.Time `json:"payment_requested_at"`
	PaymentExpiresAt   *time.Time `json:"payment_expires_at"`
	PaymentPaidAt      *time.Time `json:"payment_paid_at"`
	PaymentCanceledAt  *time.Time `json:"payment_canceled_at"`
	PaymentFailedAt    *time.Time `json:"payment_failed_at"`
	PaymentRefundedAt  *time.Time `json:"payment_refunded_at"`

	PaymentManualChannel          *string    `json:"payment_manual_channel"`
	PaymentManualReference        *string    `json:"payment_manual_reference"`
	PaymentManualReceivedByUserID *uuid.UUID `json:"payment_manual_received_by_user_id"`
	PaymentManualVerifiedByUserID *uuid.UUID `json:"payment_manual_verified_by_user_id"`
	PaymentManualVerifiedAt       *time.Time `json:"payment_manual_verified_at"`

	PaymentEntryType model.PaymentEntryType `json:"payment_entry_type"`
	InvoiceNumber    *string                `json:"invoice_number"`
	InvoiceDueDate   *time.Time             `json:"invoice_due_date"`
	InvoiceTitle     *string                `json:"invoice_title"`

	PaymentSubjectUserID    *uuid.UUID `json:"payment_subject_user_id"`
	PaymentSubjectStudentID *uuid.UUID `json:"payment_subject_student_id"`

	PaymentDescription *string        `json:"payment_description"`
	PaymentNote        *string        `json:"payment_note"`
	PaymentMeta        datatypes.JSON `json:"payment_meta"`
	PaymentAttachments datatypes.JSON `json:"payment_attachments"`

	PaymentCreatedAt time.Time  `json:"payment_created_at"`
	PaymentUpdatedAt time.Time  `json:"payment_updated_at"`
	PaymentDeletedAt *time.Time `json:"payment_deleted_at"`
}

func FromModel(m *model.Payment) *PaymentResponse {
	if m == nil {
		return nil
	}
	return &PaymentResponse{
		PaymentID: m.PaymentID,

		PaymentMasjidID: m.PaymentMasjidID,
		PaymentUserID:   m.PaymentUserID,

		PaymentStudentBillID:        m.PaymentStudentBillID,
		PaymentGeneralBillingID:     m.PaymentGeneralBillingID,
		PaymentGeneralBillingKindID: m.PaymentGeneralBillingKindID,
		PaymentBillBatchID:          m.PaymentBillBatchID,

		PaymentAmountIDR: m.PaymentAmountIDR,
		PaymentCurrency:  m.PaymentCurrency,

		PaymentStatus: m.PaymentStatus,
		PaymentMethod: m.PaymentMethod,

		PaymentGatewayProvider:  m.PaymentGatewayProvider,
		PaymentExternalID:       m.PaymentExternalID,
		PaymentGatewayReference: m.PaymentGatewayReference,
		PaymentCheckoutURL:      m.PaymentCheckoutURL,
		PaymentQRString:         m.PaymentQRString,
		PaymentSignature:        m.PaymentSignature,
		PaymentIdempotencyKey:   m.PaymentIdempotencyKey,

		PaymentRequestedAt: m.PaymentRequestedAt,
		PaymentExpiresAt:   m.PaymentExpiresAt,
		PaymentPaidAt:      m.PaymentPaidAt,
		PaymentCanceledAt:  m.PaymentCanceledAt,
		PaymentFailedAt:    m.PaymentFailedAt,
		PaymentRefundedAt:  m.PaymentRefundedAt,

		PaymentManualChannel:          m.PaymentManualChannel,
		PaymentManualReference:        m.PaymentManualReference,
		PaymentManualReceivedByUserID: m.PaymentManualReceivedByUserID,
		PaymentManualVerifiedByUserID: m.PaymentManualVerifiedByUserID,
		PaymentManualVerifiedAt:       m.PaymentManualVerifiedAt,

		PaymentEntryType: m.PaymentEntryType,
		InvoiceNumber:    m.InvoiceNumber,
		InvoiceDueDate:   m.InvoiceDueDate,
		InvoiceTitle:     m.InvoiceTitle,

		PaymentSubjectUserID:    m.PaymentSubjectUserID,
		PaymentSubjectStudentID: m.PaymentSubjectStudentID,

		PaymentDescription: m.PaymentDescription,
		PaymentNote:        m.PaymentNote,
		PaymentMeta:        m.PaymentMeta,
		PaymentAttachments: m.PaymentAttachments,

		PaymentCreatedAt: m.PaymentCreatedAt,
		PaymentUpdatedAt: m.PaymentUpdatedAt,
		PaymentDeletedAt: m.PaymentDeletedAt,
	}
}