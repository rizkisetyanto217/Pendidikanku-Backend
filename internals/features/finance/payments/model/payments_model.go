package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/*
  payments = HEADER transaksi / VA
  - 1 row = 1 kali transaksi bayar (gateway / manual)
  - total amount (sum of payment_items)
  - info VA / channel / status
  - snapshot payer (user)
*/

type PaymentModel struct {
	PaymentID uuid.UUID `gorm:"column:payment_id;type:uuid;default:gen_random_uuid();primaryKey" json:"payment_id"`

	// Tenant & actor
	PaymentSchoolID *uuid.UUID `gorm:"column:payment_school_id;type:uuid" json:"payment_school_id"`
	PaymentUserID   *uuid.UUID `gorm:"column:payment_user_id;type:uuid" json:"payment_user_id"`
	PaymentNumber   *int64     `gorm:"column:payment_number" json:"payment_number"`

	// Nominal TOTAL transaksi (sum of items)
	PaymentAmountIDR int    `gorm:"column:payment_amount_idr;not null" json:"payment_amount_idr"`
	PaymentCurrency  string `gorm:"column:payment_currency;type:varchar(8);not null;default:IDR" json:"payment_currency"`

	// Status & metode (enum di DB)
	PaymentStatus PaymentStatus `gorm:"column:payment_status;type:payment_status;not null;default:'initiated'" json:"payment_status"`
	PaymentMethod PaymentMethod `gorm:"column:payment_method;type:payment_method;not null;default:'gateway'" json:"payment_method"`

	// Info gateway (NULL jika manual)
	PaymentGatewayProvider *PaymentGatewayProvider `gorm:"column:payment_gateway_provider;type:payment_gateway_provider" json:"payment_gateway_provider"`
	PaymentExternalID      *string                 `gorm:"column:payment_external_id" json:"payment_external_id"`
	PaymentGatewayRef      *string                 `gorm:"column:payment_gateway_reference" json:"payment_gateway_reference"`
	PaymentCheckoutURL     *string                 `gorm:"column:payment_checkout_url" json:"payment_checkout_url"`
	PaymentQRString        *string                 `gorm:"column:payment_qr_string" json:"payment_qr_string"`
	PaymentSignature       *string                 `gorm:"column:payment_signature" json:"payment_signature"`
	PaymentIdempotencyKey  *string                 `gorm:"column:payment_idempotency_key" json:"payment_idempotency_key"`

	// Snapshot channel/bank/VA (hasil dari provider)
	PaymentChannelSnapshot  *string `gorm:"column:payment_channel_snapshot;type:varchar(40)" json:"payment_channel_snapshot"`
	PaymentBankSnapshot     *string `gorm:"column:payment_bank_snapshot;type:varchar(80)" json:"payment_bank_snapshot"`
	PaymentVANumberSnapshot *string `gorm:"column:payment_va_number_snapshot;type:varchar(80)" json:"payment_va_number_snapshot"`
	PaymentVANameSnapshot   *string `gorm:"column:payment_va_name_snapshot;type:varchar(160)" json:"payment_va_name_snapshot"`

	// Timestamps status
	PaymentRequestedAt *time.Time `gorm:"column:payment_requested_at" json:"payment_requested_at"`
	PaymentExpiresAt   *time.Time `gorm:"column:payment_expires_at" json:"payment_expires_at"`
	PaymentPaidAt      *time.Time `gorm:"column:payment_paid_at" json:"payment_paid_at"`
	PaymentCanceledAt  *time.Time `gorm:"column:payment_canceled_at" json:"payment_canceled_at"`
	PaymentFailedAt    *time.Time `gorm:"column:payment_failed_at" json:"payment_failed_at"`
	PaymentRefundedAt  *time.Time `gorm:"column:payment_refunded_at" json:"payment_refunded_at"`

	// Manual ops (kasir/admin)
	PaymentManualChannel        *string    `gorm:"column:payment_manual_channel;type:varchar(32)" json:"payment_manual_channel"`
	PaymentManualReference      *string    `gorm:"column:payment_manual_reference;type:varchar(120)" json:"payment_manual_reference"`
	PaymentManualReceivedByUser *uuid.UUID `gorm:"column:payment_manual_received_by_user_id;type:uuid" json:"payment_manual_received_by_user_id"`
	PaymentManualVerifiedByUser *uuid.UUID `gorm:"column:payment_manual_verified_by_user_id;type:uuid" json:"payment_manual_verified_by_user_id"`
	PaymentManualVerifiedAt     *time.Time `gorm:"column:payment_manual_verified_at" json:"payment_manual_verified_at"`

	// Ledger / tipe entry
	PaymentEntryType PaymentEntryType `gorm:"column:payment_entry_type;type:payment_entry_type;not null;default:'payment'" json:"payment_entry_type"`

	// Subjek pembayaran (payer di level user)
	PaymentSubjectUserID *uuid.UUID `gorm:"column:payment_subject_user_id;type:uuid" json:"payment_subject_user_id"`

	// User snapshots (payer)
	PaymentUserNameSnapshot     *string `gorm:"column:payment_user_name_snapshot" json:"payment_user_name_snapshot"`
	PaymentFullNameSnapshot     *string `gorm:"column:payment_full_name_snapshot" json:"payment_full_name_snapshot"`
	PaymentEmailSnapshot        *string `gorm:"column:payment_email_snapshot" json:"payment_email_snapshot"`
	PaymentDonationNameSnapshot *string `gorm:"column:payment_donation_name_snapshot" json:"payment_donation_name_snapshot"`

	// Meta (header level / bundle)
	PaymentDescription *string        `gorm:"column:payment_description" json:"payment_description"`
	PaymentNote        *string        `gorm:"column:payment_note" json:"payment_note"`
	PaymentMeta        datatypes.JSON `gorm:"column:payment_meta;type:jsonb" json:"payment_meta"`
	PaymentAttachments datatypes.JSON `gorm:"column:payment_attachments;type:jsonb" json:"payment_attachments"`

	// Audit
	PaymentCreatedAt time.Time  `gorm:"column:payment_created_at;not null;default:now()" json:"payment_created_at"`
	PaymentUpdatedAt time.Time  `gorm:"column:payment_updated_at;not null;default:now()" json:"payment_updated_at"`
	PaymentDeletedAt *time.Time `gorm:"column:payment_deleted_at" json:"payment_deleted_at"`

	// OPTIONAL: relation ke items (kalau mau eager load di service)
	// PaymentItems []PaymentItemModel `gorm:"foreignKey:PaymentItemPaymentID;references:PaymentID" json:"payment_items,omitempty"`
}

func (PaymentModel) TableName() string {
	return "payments"
}
