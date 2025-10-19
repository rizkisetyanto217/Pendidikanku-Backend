package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ===================== Enums (string) ===================== */
/* Selaras dengan ENUM di PostgreSQL:
   payment_status, payment_method, payment_gateway_provider
*/

const (
	PaymentStatusInitiated         = "initiated"
	PaymentStatusPending           = "pending"
	PaymentStatusAwaitingCallback  = "awaiting_callback"
	PaymentStatusPaid              = "paid"
	PaymentStatusPartiallyRefunded = "partially_refunded"
	PaymentStatusRefunded          = "refunded"
	PaymentStatusFailed            = "failed"
	PaymentStatusCanceled          = "canceled"
	PaymentStatusExpired           = "expired"
)

const (
	PaymentMethodGateway      = "gateway"
	PaymentMethodBankTransfer = "bank_transfer"
	PaymentMethodCash         = "cash"
	PaymentMethodQRIS         = "qris"
	PaymentMethodOther        = "other"
)

const (
	PaymentProviderMidtrans = "midtrans"
	PaymentProviderXendit   = "xendit"
	PaymentProviderTripay   = "tripay"
	PaymentProviderDuitku   = "duitku"
	PaymentProviderNicepay  = "nicepay"
	PaymentProviderStripe   = "stripe"
	PaymentProviderPaypal   = "paypal"
	PaymentProviderOther    = "other"
)

/* ===================== Model ===================== */

type Payment struct {
	PaymentID uuid.UUID `gorm:"column:payment_id;type:uuid;default:gen_random_uuid();primaryKey" json:"payment_id"`

	PaymentMasjidID *uuid.UUID `gorm:"column:payment_masjid_id;type:uuid" json:"payment_masjid_id,omitempty"`
	PaymentUserID   *uuid.UUID `gorm:"column:payment_user_id;type:uuid" json:"payment_user_id,omitempty"`

	// Link ke billing (opsional)
	PaymentUserSPPBillingID *uuid.UUID `gorm:"column:payment_user_spp_billing_id;type:uuid" json:"payment_user_spp_billing_id,omitempty"`
	PaymentSPPBillingID     *uuid.UUID `gorm:"column:payment_spp_billing_id;type:uuid" json:"payment_spp_billing_id,omitempty"`

	// Nominal & mata uang
	PaymentAmountIDR int    `gorm:"column:payment_amount_idr;not null;check:payment_amount_idr >= 0" json:"payment_amount_idr"`
	PaymentCurrency  string `gorm:"column:payment_currency;type:varchar(8);not null;default:IDR" json:"payment_currency"`

	// Status & metode
	PaymentStatus string `gorm:"column:payment_status;type:payment_status;not null;default:'initiated'" json:"payment_status"`
	PaymentMethod string `gorm:"column:payment_method;type:payment_method;not null;default:'gateway'" json:"payment_method"`

	// Info gateway (opsional untuk metode manual)
	PaymentGatewayProvider  *string `gorm:"column:payment_gateway_provider;type:payment_gateway_provider" json:"payment_gateway_provider,omitempty"`
	PaymentExternalID       *string `gorm:"column:payment_external_id" json:"payment_external_id,omitempty"`             // order_id/invoice_id di PSP
	PaymentGatewayReference *string `gorm:"column:payment_gateway_reference" json:"payment_gateway_reference,omitempty"` // VA number/QR ref
	PaymentCheckoutURL      *string `gorm:"column:payment_checkout_url" json:"payment_checkout_url,omitempty"`
	PaymentQRString         *string `gorm:"column:payment_qr_string" json:"payment_qr_string,omitempty"`
	PaymentSignature        *string `gorm:"column:payment_signature" json:"payment_signature,omitempty"`
	PaymentIdempotencyKey   *string `gorm:"column:payment_idempotency_key" json:"payment_idempotency_key,omitempty"`

	// Timestamps penting
	PaymentRequestedAt *time.Time `gorm:"column:payment_requested_at" json:"payment_requested_at,omitempty"`
	PaymentExpiresAt   *time.Time `gorm:"column:payment_expires_at" json:"payment_expires_at,omitempty"`
	PaymentPaidAt      *time.Time `gorm:"column:payment_paid_at" json:"payment_paid_at,omitempty"`
	PaymentCanceledAt  *time.Time `gorm:"column:payment_canceled_at" json:"payment_canceled_at,omitempty"`
	PaymentFailedAt    *time.Time `gorm:"column:payment_failed_at" json:"payment_failed_at,omitempty"`
	PaymentRefundedAt  *time.Time `gorm:"column:payment_refunded_at" json:"payment_refunded_at,omitempty"`

	// Metadata & catatan
	PaymentDescription *string           `gorm:"column:payment_description" json:"payment_description,omitempty"`
	PaymentNote        *string           `gorm:"column:payment_note" json:"payment_note,omitempty"`
	PaymentMeta        datatypes.JSONMap `gorm:"column:payment_meta;type:jsonb" json:"payment_meta,omitempty"`

	// Base timestamps
	CreatedAt time.Time      `gorm:"column:payment_created_at;autoCreateTime" json:"payment_created_at"`
	UpdatedAt time.Time      `gorm:"column:payment_updated_at;autoUpdateTime" json:"payment_updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:payment_deleted_at;index" json:"payment_deleted_at,omitempty"`
}

func (Payment) TableName() string { return "payments" }

/* ===================== Helpers ===================== */

func (p *Payment) IsForSPP() bool {
	return p.PaymentUserSPPBillingID != nil
}

func (p *Payment) IsGateway() bool {
	return p.PaymentMethod == PaymentMethodGateway && p.PaymentGatewayProvider != nil
}

func (p *Payment) IsManual() bool {
	return p.PaymentMethod == PaymentMethodCash || p.PaymentMethod == PaymentMethodBankTransfer || p.PaymentMethod == PaymentMethodQRIS
}

func (p *Payment) IsPaid() bool {
	return p.PaymentStatus == PaymentStatusPaid || p.PaymentStatus == PaymentStatusRefunded || p.PaymentStatus == PaymentStatusPartiallyRefunded
}

func (p *Payment) IsOpen() bool {
	switch p.PaymentStatus {
	case PaymentStatusInitiated, PaymentStatusPending, PaymentStatusAwaitingCallback:
		return true
	default:
		return false
	}
}
