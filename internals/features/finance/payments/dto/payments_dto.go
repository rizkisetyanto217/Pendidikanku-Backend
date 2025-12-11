// file: internals/features/finance/payments/dto/payment_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "madinahsalam_backend/internals/features/finance/payments/model"
	"madinahsalam_backend/internals/helpers/dbtime"
)

/* =========================================================
   PatchField (tetap seperti punyamu)
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
   REQUEST: CreatePayment (HEADER saja)
========================================================= */

type CreatePaymentRequest struct {
	PaymentSchoolID *uuid.UUID `json:"payment_school_id"`
	PaymentUserID   *uuid.UUID `json:"payment_user_id"`
	PaymentNumber   *int64     `json:"payment_number"`

	PaymentAmountIDR int    `json:"payment_amount_idr" validate:"required,min=0"`
	PaymentCurrency  string `json:"payment_currency"    validate:"omitempty,oneof=IDR"`

	PaymentStatus *model.PaymentStatus `json:"payment_status" validate:"omitempty,oneof=initiated pending awaiting_callback paid partially_refunded refunded failed canceled expired"`
	PaymentMethod *model.PaymentMethod `json:"payment_method" validate:"omitempty,oneof=gateway bank_transfer cash qris other"`

	PaymentGatewayProvider  *model.PaymentGatewayProvider `json:"payment_gateway_provider" validate:"omitempty,oneof=midtrans xendit tripay duitku nicepay stripe paypal other"`
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

	PaymentManualChannel          *string    `json:"payment_manual_channel" validate:"omitempty,max=32"`
	PaymentManualReference        *string    `json:"payment_manual_reference" validate:"omitempty,max=120"`
	PaymentManualReceivedByUserID *uuid.UUID `json:"payment_manual_received_by_user_id"`
	PaymentManualVerifiedByUserID *uuid.UUID `json:"payment_manual_verified_by_user_id"`
	PaymentManualVerifiedAt       *time.Time `json:"payment_manual_verified_at"`

	PaymentEntryType *model.PaymentEntryType `json:"payment_entry_type" validate:"omitempty,oneof=charge payment refund adjustment"`

	PaymentSubjectUserID *uuid.UUID `json:"payment_subject_user_id"`

	PaymentUserNameSnapshot     *string `json:"payment_user_name_snapshot"`
	PaymentFullNameSnapshot     *string `json:"payment_full_name_snapshot"`
	PaymentEmailSnapshot        *string `json:"payment_email_snapshot"`
	PaymentDonationNameSnapshot *string `json:"payment_donation_name_snapshot"`

	PaymentChannelSnapshot  *string        `json:"payment_channel_snapshot"`
	PaymentBankSnapshot     *string        `json:"payment_bank_snapshot"`
	PaymentVANumberSnapshot *string        `json:"payment_va_number_snapshot"`
	PaymentVANameSnapshot   *string        `json:"payment_va_name_snapshot"`
	PaymentDescription      *string        `json:"payment_description"`
	PaymentNote             *string        `json:"payment_note"`
	PaymentMeta             datatypes.JSON `json:"payment_meta"`
	PaymentAttachments      datatypes.JSON `json:"payment_attachments"`
}

func (r *CreatePaymentRequest) Validate() error {
	if r.PaymentAmountIDR < 0 {
		return errors.New("payment_amount_idr tidak boleh negatif")
	}
	if r.PaymentCurrency != "" && r.PaymentCurrency != "IDR" {
		return fmt.Errorf("payment_currency hanya mendukung 'IDR'")
	}

	method := model.PaymentMethodGateway
	if r.PaymentMethod != nil {
		method = *r.PaymentMethod
	}

	// Konsistensi method vs provider
	if method == model.PaymentMethodGateway && r.PaymentGatewayProvider == nil {
		return errors.New("payment_method=gateway harus menyertakan payment_gateway_provider")
	}
	if method != model.PaymentMethodGateway && r.PaymentGatewayProvider != nil {
		return errors.New("payment_method manual ('cash','bank_transfer','qris','other') tidak boleh menyertakan payment_gateway_provider")
	}

	return nil
}

// sekarang pakai dbtime, dan controller tidak perlu import dbtime lagi
func (r *CreatePaymentRequest) ToModel(c *fiber.Ctx) *model.PaymentModel {
	now, err := dbtime.GetDBTime(c)
	if err != nil {
		// fallback kalau suatu saat GetDBTime beneran query ke DB dan error
		now = dbtime.NowInSchool(c)
	}

	out := &model.PaymentModel{
		PaymentSchoolID:  r.PaymentSchoolID,
		PaymentUserID:    r.PaymentUserID,
		PaymentNumber:    r.PaymentNumber,
		PaymentAmountIDR: r.PaymentAmountIDR,
		PaymentCurrency:  "IDR",

		PaymentStatus: model.PaymentStatusInitiated,
		PaymentMethod: model.PaymentMethodGateway,

		PaymentGatewayProvider: r.PaymentGatewayProvider,
		PaymentExternalID:      r.PaymentExternalID,
		PaymentGatewayRef:      r.PaymentGatewayReference,
		PaymentCheckoutURL:     r.PaymentCheckoutURL,
		PaymentQRString:        r.PaymentQRString,
		PaymentSignature:       r.PaymentSignature,
		PaymentIdempotencyKey:  r.PaymentIdempotencyKey,
		PaymentRequestedAt:     r.PaymentRequestedAt,
		PaymentExpiresAt:       r.PaymentExpiresAt,
		PaymentPaidAt:          r.PaymentPaidAt,
		PaymentCanceledAt:      r.PaymentCanceledAt,
		PaymentFailedAt:        r.PaymentFailedAt,
		PaymentRefundedAt:      r.PaymentRefundedAt,

		PaymentManualChannel:        r.PaymentManualChannel,
		PaymentManualReference:      r.PaymentManualReference,
		PaymentManualReceivedByUser: r.PaymentManualReceivedByUserID,
		PaymentManualVerifiedByUser: r.PaymentManualVerifiedByUserID,
		PaymentManualVerifiedAt:     r.PaymentManualVerifiedAt,

		PaymentEntryType:            model.PaymentEntryPayment,
		PaymentSubjectUserID:        r.PaymentSubjectUserID,
		PaymentUserNameSnapshot:     r.PaymentUserNameSnapshot,
		PaymentFullNameSnapshot:     r.PaymentFullNameSnapshot,
		PaymentEmailSnapshot:        r.PaymentEmailSnapshot,
		PaymentDonationNameSnapshot: r.PaymentDonationNameSnapshot,

		PaymentChannelSnapshot:  r.PaymentChannelSnapshot,
		PaymentBankSnapshot:     r.PaymentBankSnapshot,
		PaymentVANumberSnapshot: r.PaymentVANumberSnapshot,
		PaymentVANameSnapshot:   r.PaymentVANameSnapshot,

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
   UPDATE (tetap, tidak perlu dbtime)
========================================================= */

type UpdatePaymentRequest struct {
	PaymentSchoolID PatchField[uuid.UUID] `json:"payment_school_id"`
	PaymentUserID   PatchField[uuid.UUID] `json:"payment_user_id"`
	PaymentNumber   PatchField[int64]     `json:"payment_number"`

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

	PaymentEntryType   PatchField[model.PaymentEntryType] `json:"payment_entry_type"`
	PaymentSubjectUser PatchField[uuid.UUID]              `json:"payment_subject_user_id"`

	PaymentUserNameSnapshot     PatchField[string]         `json:"payment_user_name_snapshot"`
	PaymentFullNameSnapshot     PatchField[string]         `json:"payment_full_name_snapshot"`
	PaymentEmailSnapshot        PatchField[string]         `json:"payment_email_snapshot"`
	PaymentDonationNameSnapshot PatchField[string]         `json:"payment_donation_name_snapshot"`
	PaymentChannelSnapshot      PatchField[string]         `json:"payment_channel_snapshot"`
	PaymentBankSnapshot         PatchField[string]         `json:"payment_bank_snapshot"`
	PaymentVANumberSnapshot     PatchField[string]         `json:"payment_va_number_snapshot"`
	PaymentVANameSnapshot       PatchField[string]         `json:"payment_va_name_snapshot"`
	PaymentDescription          PatchField[string]         `json:"payment_description"`
	PaymentNote                 PatchField[string]         `json:"payment_note"`
	PaymentMeta                 PatchField[datatypes.JSON] `json:"payment_meta"`
	PaymentAttachments          PatchField[datatypes.JSON] `json:"payment_attachments"`
}

func (p *UpdatePaymentRequest) Apply(m *model.PaymentModel) error {
	applyPtr(&m.PaymentSchoolID, p.PaymentSchoolID)
	applyPtr(&m.PaymentUserID, p.PaymentUserID)
	applyPtr(&m.PaymentNumber, p.PaymentNumber)

	// amount
	if p.PaymentAmountIDR.Set {
		if p.PaymentAmountIDR.Null || p.PaymentAmountIDR.Value == nil {
			return errors.New("payment_amount_idr tidak boleh null")
		}
		if *p.PaymentAmountIDR.Value < 0 {
			return errors.New("payment_amount_idr tidak boleh negatif")
		}
		m.PaymentAmountIDR = *p.PaymentAmountIDR.Value
	}

	// currency
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
	if p.PaymentStatus.Set {
		if p.PaymentStatus.Null || p.PaymentStatus.Value == nil {
			return errors.New("payment_status tidak boleh null")
		}
		m.PaymentStatus = *p.PaymentStatus.Value
	}
	if p.PaymentMethod.Set {
		if p.PaymentMethod.Null || p.PaymentMethod.Value == nil {
			return errors.New("payment_method tidak boleh null")
		}
		m.PaymentMethod = *p.PaymentMethod.Value
	}
	if p.PaymentEntryType.Set {
		if p.PaymentEntryType.Null || p.PaymentEntryType.Value == nil {
			return errors.New("payment_entry_type tidak boleh null")
		}
		m.PaymentEntryType = *p.PaymentEntryType.Value
	}

	// gateway
	applyPtr(&m.PaymentGatewayProvider, p.PaymentGatewayProvider)
	applyPtr(&m.PaymentExternalID, p.PaymentExternalID)
	applyPtr(&m.PaymentGatewayRef, p.PaymentGatewayReference)
	applyPtr(&m.PaymentCheckoutURL, p.PaymentCheckoutURL)
	applyPtr(&m.PaymentQRString, p.PaymentQRString)
	applyPtr(&m.PaymentSignature, p.PaymentSignature)
	applyPtr(&m.PaymentIdempotencyKey, p.PaymentIdempotencyKey)

	// timestamps (biarkan apa adanya; DB akan simpan sebagai UTC)
	applyPtr(&m.PaymentRequestedAt, p.PaymentRequestedAt)
	applyPtr(&m.PaymentExpiresAt, p.PaymentExpiresAt)
	applyPtr(&m.PaymentPaidAt, p.PaymentPaidAt)
	applyPtr(&m.PaymentCanceledAt, p.PaymentCanceledAt)
	applyPtr(&m.PaymentFailedAt, p.PaymentFailedAt)
	applyPtr(&m.PaymentRefundedAt, p.PaymentRefundedAt)

	// manual ops
	applyPtr(&m.PaymentManualChannel, p.PaymentManualChannel)
	applyPtr(&m.PaymentManualReference, p.PaymentManualReference)
	applyPtr(&m.PaymentManualReceivedByUser, p.PaymentManualReceivedByUserID)
	applyPtr(&m.PaymentManualVerifiedByUser, p.PaymentManualVerifiedByUserID)
	applyPtr(&m.PaymentManualVerifiedAt, p.PaymentManualVerifiedAt)

	// subject & snapshots
	applyPtr(&m.PaymentSubjectUserID, p.PaymentSubjectUser)
	applyPtr(&m.PaymentUserNameSnapshot, p.PaymentUserNameSnapshot)
	applyPtr(&m.PaymentFullNameSnapshot, p.PaymentFullNameSnapshot)
	applyPtr(&m.PaymentEmailSnapshot, p.PaymentEmailSnapshot)
	applyPtr(&m.PaymentDonationNameSnapshot, p.PaymentDonationNameSnapshot)

	applyPtr(&m.PaymentChannelSnapshot, p.PaymentChannelSnapshot)
	applyPtr(&m.PaymentBankSnapshot, p.PaymentBankSnapshot)
	applyPtr(&m.PaymentVANumberSnapshot, p.PaymentVANumberSnapshot)
	applyPtr(&m.PaymentVANameSnapshot, p.PaymentVANameSnapshot)

	applyPtr(&m.PaymentDescription, p.PaymentDescription)
	applyPtr(&m.PaymentNote, p.PaymentNote)
	applyVal(&m.PaymentMeta, p.PaymentMeta)
	applyVal(&m.PaymentAttachments, p.PaymentAttachments)

	// Konsistensi method/provider
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

	PaymentSchoolID *uuid.UUID `json:"payment_school_id"`
	PaymentUserID   *uuid.UUID `json:"payment_user_id"`
	PaymentNumber   *int64     `json:"payment_number"`

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

	PaymentEntryType     model.PaymentEntryType `json:"payment_entry_type"`
	PaymentSubjectUserID *uuid.UUID             `json:"payment_subject_user_id"`

	PaymentUserNameSnapshot     *string `json:"payment_user_name_snapshot"`
	PaymentFullNameSnapshot     *string `json:"payment_full_name_snapshot"`
	PaymentEmailSnapshot        *string `json:"payment_email_snapshot"`
	PaymentDonationNameSnapshot *string `json:"payment_donation_name_snapshot"`

	PaymentChannelSnapshot  *string        `json:"payment_channel_snapshot"`
	PaymentBankSnapshot     *string        `json:"payment_bank_snapshot"`
	PaymentVANumberSnapshot *string        `json:"payment_va_number_snapshot"`
	PaymentVANameSnapshot   *string        `json:"payment_va_name_snapshot"`
	PaymentDescription      *string        `json:"payment_description"`
	PaymentNote             *string        `json:"payment_note"`
	PaymentMeta             datatypes.JSON `json:"payment_meta"`
	PaymentAttachments      datatypes.JSON `json:"payment_attachments"`

	PaymentCreatedAt time.Time  `json:"payment_created_at"`
	PaymentUpdatedAt time.Time  `json:"payment_updated_at"`
	PaymentDeletedAt *time.Time `json:"payment_deleted_at"`
}

// sekarang FromModel pakai dbtime dan butuh *fiber.Ctx
func FromModel(c *fiber.Ctx, m *model.PaymentModel) *PaymentResponse {
	if m == nil {
		return nil
	}
	return &PaymentResponse{
		PaymentID: m.PaymentID,

		PaymentSchoolID: m.PaymentSchoolID,
		PaymentUserID:   m.PaymentUserID,
		PaymentNumber:   m.PaymentNumber,

		PaymentAmountIDR: m.PaymentAmountIDR,
		PaymentCurrency:  m.PaymentCurrency,

		PaymentStatus: m.PaymentStatus,
		PaymentMethod: m.PaymentMethod,

		PaymentGatewayProvider:  m.PaymentGatewayProvider,
		PaymentExternalID:       m.PaymentExternalID,
		PaymentGatewayReference: m.PaymentGatewayRef,
		PaymentCheckoutURL:      m.PaymentCheckoutURL,
		PaymentQRString:         m.PaymentQRString,
		PaymentSignature:        m.PaymentSignature,
		PaymentIdempotencyKey:   m.PaymentIdempotencyKey,

		PaymentRequestedAt: dbtime.ToSchoolTimePtr(c, m.PaymentRequestedAt),
		PaymentExpiresAt:   dbtime.ToSchoolTimePtr(c, m.PaymentExpiresAt),
		PaymentPaidAt:      dbtime.ToSchoolTimePtr(c, m.PaymentPaidAt),
		PaymentCanceledAt:  dbtime.ToSchoolTimePtr(c, m.PaymentCanceledAt),
		PaymentFailedAt:    dbtime.ToSchoolTimePtr(c, m.PaymentFailedAt),
		PaymentRefundedAt:  dbtime.ToSchoolTimePtr(c, m.PaymentRefundedAt),

		PaymentManualChannel:          m.PaymentManualChannel,
		PaymentManualReference:        m.PaymentManualReference,
		PaymentManualReceivedByUserID: m.PaymentManualReceivedByUser,
		PaymentManualVerifiedByUserID: m.PaymentManualVerifiedByUser,
		PaymentManualVerifiedAt:       dbtime.ToSchoolTimePtr(c, m.PaymentManualVerifiedAt),

		PaymentEntryType:     m.PaymentEntryType,
		PaymentSubjectUserID: m.PaymentSubjectUserID,

		PaymentUserNameSnapshot:     m.PaymentUserNameSnapshot,
		PaymentFullNameSnapshot:     m.PaymentFullNameSnapshot,
		PaymentEmailSnapshot:        m.PaymentEmailSnapshot,
		PaymentDonationNameSnapshot: m.PaymentDonationNameSnapshot,

		PaymentChannelSnapshot:  m.PaymentChannelSnapshot,
		PaymentBankSnapshot:     m.PaymentBankSnapshot,
		PaymentVANumberSnapshot: m.PaymentVANumberSnapshot,
		PaymentVANameSnapshot:   m.PaymentVANameSnapshot,

		PaymentDescription: m.PaymentDescription,
		PaymentNote:        m.PaymentNote,
		PaymentMeta:        m.PaymentMeta,
		PaymentAttachments: m.PaymentAttachments,

		PaymentCreatedAt: dbtime.ToSchoolTime(c, m.PaymentCreatedAt),
		PaymentUpdatedAt: dbtime.ToSchoolTime(c, m.PaymentUpdatedAt),
		PaymentDeletedAt: dbtime.ToSchoolTimePtr(c, m.PaymentDeletedAt),
	}
}
