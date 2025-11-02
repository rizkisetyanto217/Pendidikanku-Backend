// file: internals/features/finance/payments/model/payment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* ================================
   ENUM mirror (harus cocok dgn DB)
================================ */

type PaymentStatus string
type PaymentMethod string
type PaymentGatewayProvider string
type PaymentEntryType string

const (
	PaymentStatusInitiated         PaymentStatus = "initiated"
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusAwaitingCallback  PaymentStatus = "awaiting_callback"
	PaymentStatusPaid              PaymentStatus = "paid"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
	PaymentStatusRefunded          PaymentStatus = "refunded"
	PaymentStatusFailed            PaymentStatus = "failed"
	PaymentStatusCanceled          PaymentStatus = "canceled"
	PaymentStatusExpired           PaymentStatus = "expired"
)

const (
	PaymentMethodGateway      PaymentMethod = "gateway"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodQRIS         PaymentMethod = "qris"
	PaymentMethodOther        PaymentMethod = "other"
)

const (
	GatewayProviderMidtrans PaymentGatewayProvider = "midtrans"
	GatewayProviderXendit   PaymentGatewayProvider = "xendit"
	GatewayProviderTripay   PaymentGatewayProvider = "tripay"
	GatewayProviderDuitku   PaymentGatewayProvider = "duitku"
	GatewayProviderNicepay  PaymentGatewayProvider = "nicepay"
	GatewayProviderStripe   PaymentGatewayProvider = "stripe"
	GatewayProviderPaypal   PaymentGatewayProvider = "paypal"
	GatewayProviderOther    PaymentGatewayProvider = "other"
)

const (
	PaymentEntryCharge     PaymentEntryType = "charge"
	PaymentEntryPayment    PaymentEntryType = "payment"
	PaymentEntryRefund     PaymentEntryType = "refund"
	PaymentEntryAdjustment PaymentEntryType = "adjustment"
)

/* ================================
   MODEL: payments
================================ */

type Payment struct {
	PaymentID uuid.UUID `json:"payment_id" gorm:"column:payment_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// Tenant & actor
	PaymentSchoolID *uuid.UUID `json:"payment_school_id" gorm:"column:payment_school_id;type:uuid"`
	PaymentUserID   *uuid.UUID `json:"payment_user_id"   gorm:"column:payment_user_id;type:uuid"`

	// ===== Target FK (salah satu WAJIB terisi) =====
	PaymentStudentBillID        *uuid.UUID `json:"payment_student_bill_id"          gorm:"column:payment_student_bill_id;type:uuid"`
	PaymentGeneralBillingID     *uuid.UUID `json:"payment_general_billing_id"       gorm:"column:payment_general_billing_id;type:uuid"`
	PaymentGeneralBillingKindID *uuid.UUID `json:"payment_general_billing_kind_id"  gorm:"column:payment_general_billing_kind_id;type:uuid"`

	// ===== Konteks/report (opsional) =====
	PaymentBillBatchID *uuid.UUID `json:"payment_bill_batch_id" gorm:"column:payment_bill_batch_id;type:uuid"`

	// Nominal
	PaymentAmountIDR int    `json:"payment_amount_idr" gorm:"column:payment_amount_idr;type:int;not null;check:payment_amount_idr>=0"`
	PaymentCurrency  string `json:"payment_currency"   gorm:"column:payment_currency;type:varchar(8);not null;default:IDR"`

	// Status & metode (enum DB)
	PaymentStatus PaymentStatus `json:"payment_status" gorm:"column:payment_status;type:payment_status;not null;default:'initiated'"`
	PaymentMethod PaymentMethod `json:"payment_method" gorm:"column:payment_method;type:payment_method;not null;default:'gateway'"`

	// Info gateway (NULL jika manual)
	PaymentGatewayProvider  *PaymentGatewayProvider `json:"payment_gateway_provider"  gorm:"column:payment_gateway_provider;type:payment_gateway_provider"`
	PaymentExternalID       *string                 `json:"payment_external_id"       gorm:"column:payment_external_id;type:text"`
	PaymentGatewayReference *string                 `json:"payment_gateway_reference" gorm:"column:payment_gateway_reference;type:text"`
	PaymentCheckoutURL      *string                 `json:"payment_checkout_url"      gorm:"column:payment_checkout_url;type:text"`
	PaymentQRString         *string                 `json:"payment_qr_string"         gorm:"column:payment_qr_string;type:text"`
	PaymentSignature        *string                 `json:"payment_signature"         gorm:"column:payment_signature;type:text"`
	PaymentIdempotencyKey   *string                 `json:"payment_idempotency_key"   gorm:"column:payment_idempotency_key;type:text"`

	// Timestamps status
	PaymentRequestedAt *time.Time `json:"payment_requested_at" gorm:"column:payment_requested_at;type:timestamptz;default:now()"`
	PaymentExpiresAt   *time.Time `json:"payment_expires_at"   gorm:"column:payment_expires_at;type:timestamptz"`
	PaymentPaidAt      *time.Time `json:"payment_paid_at"      gorm:"column:payment_paid_at;type:timestamptz"`
	PaymentCanceledAt  *time.Time `json:"payment_canceled_at"  gorm:"column:payment_canceled_at;type:timestamptz"`
	PaymentFailedAt    *time.Time `json:"payment_failed_at"    gorm:"column:payment_failed_at;type:timestamptz"`
	PaymentRefundedAt  *time.Time `json:"payment_refunded_at"  gorm:"column:payment_refunded_at;type:timestamptz"`

	// Manual ops
	PaymentManualChannel          *string    `json:"payment_manual_channel"           gorm:"column:payment_manual_channel;type:varchar(32)"`
	PaymentManualReference        *string    `json:"payment_manual_reference"         gorm:"column:payment_manual_reference;type:varchar(120)"`
	PaymentManualReceivedByUserID *uuid.UUID `json:"payment_manual_received_by_user_id" gorm:"column:payment_manual_received_by_user_id;type:uuid"`
	PaymentManualVerifiedByUserID *uuid.UUID `json:"payment_manual_verified_by_user_id" gorm:"column:payment_manual_verified_by_user_id;type:uuid"`
	PaymentManualVerifiedAt       *time.Time `json:"payment_manual_verified_at"       gorm:"column:payment_manual_verified_at;type:timestamptz"`

	// Ledger & invoice fields
	PaymentEntryType PaymentEntryType `json:"payment_entry_type" gorm:"column:payment_entry_type;type:payment_entry_type;not null;default:'payment'"`
	InvoiceNumber    *string          `json:"invoice_number"     gorm:"column:invoice_number;type:text"`
	InvoiceDueDate   *time.Time       `json:"invoice_due_date"   gorm:"column:invoice_due_date;type:date"`
	InvoiceTitle     *string          `json:"invoice_title"      gorm:"column:invoice_title;type:text"`

	// Subject (opsional)
	PaymentSubjectUserID    *uuid.UUID `json:"payment_subject_user_id"    gorm:"column:payment_subject_user_id;type:uuid"`
	PaymentSubjectStudentID *uuid.UUID `json:"payment_subject_student_id" gorm:"column:payment_subject_student_id;type:uuid"`

	// Meta
	PaymentDescription *string        `json:"payment_description" gorm:"column:payment_description;type:text"`
	PaymentNote        *string        `json:"payment_note"        gorm:"column:payment_note;type:text"`
	PaymentMeta        datatypes.JSON `json:"payment_meta"        gorm:"column:payment_meta;type:jsonb"`
	PaymentAttachments datatypes.JSON `json:"payment_attachments" gorm:"column:payment_attachments;type:jsonb"`

	// Audit
	PaymentCreatedAt time.Time  `json:"payment_created_at" gorm:"column:payment_created_at;type:timestamptz;not null;default:now()"`
	PaymentUpdatedAt time.Time  `json:"payment_updated_at" gorm:"column:payment_updated_at;type:timestamptz;not null;default:now()"`
	PaymentDeletedAt *time.Time `json:"payment_deleted_at" gorm:"column:payment_deleted_at;type:timestamptz"`
}

func (Payment) TableName() string { return "payments" }
