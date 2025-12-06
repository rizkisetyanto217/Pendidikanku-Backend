// file: internals/features/finance/payments/model/payment_group_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PaymentGroup merepresentasikan header checkout / group pembayaran
// mapping ke table: payment_groups
type PaymentGroupModel struct {
	PaymentGroupID              uuid.UUID               `gorm:"column:payment_group_id;type:uuid;default:gen_random_uuid();primaryKey" json:"payment_group_id"`
	PaymentGroupSchoolID        *uuid.UUID              `gorm:"column:payment_group_school_id;type:uuid" json:"payment_group_school_id,omitempty"`
	PaymentGroupUserID          *uuid.UUID              `gorm:"column:payment_group_user_id;type:uuid" json:"payment_group_user_id,omitempty"`
	PaymentGroupSchoolStudentID *uuid.UUID              `gorm:"column:payment_group_school_student_id;type:uuid" json:"payment_group_school_student_id,omitempty"`
	PaymentGroupNumber          *int64                  `gorm:"column:payment_group_number" json:"payment_group_number,omitempty"`
	PaymentGroupAmountIDR       int                     `gorm:"column:payment_group_amount_idr;not null" json:"payment_group_amount_idr"`
	PaymentGroupCurrency        string                  `gorm:"column:payment_group_currency;type:varchar(8);not null;default:IDR" json:"payment_group_currency"`
	PaymentGroupStatus          PaymentStatus           `gorm:"column:payment_group_status;type:payment_status;not null;default:'initiated'" json:"payment_group_status"`
	PaymentGroupMethod          PaymentMethod           `gorm:"column:payment_group_method;type:payment_method;not null;default:'gateway'" json:"payment_group_method"`
	PaymentGroupGatewayProvider *PaymentGatewayProvider `gorm:"column:payment_group_gateway_provider;type:payment_gateway_provider" json:"payment_group_gateway_provider,omitempty"`

	PaymentGroupExternalID       *string        `gorm:"column:payment_group_external_id" json:"payment_group_external_id,omitempty"`
	PaymentGroupGatewayReference *string        `gorm:"column:payment_group_gateway_reference" json:"payment_group_gateway_reference,omitempty"`
	PaymentGroupCheckoutURL      *string        `gorm:"column:payment_group_checkout_url" json:"payment_group_checkout_url,omitempty"`
	PaymentGroupQRString         *string        `gorm:"column:payment_group_qr_string" json:"payment_group_qr_string,omitempty"`
	PaymentGroupSignature        *string        `gorm:"column:payment_group_signature" json:"payment_group_signature,omitempty"`
	PaymentGroupIdempotencyKey   *string        `gorm:"column:payment_group_idempotency_key" json:"payment_group_idempotency_key,omitempty"`
	PaymentGroupRequestedAt      time.Time      `gorm:"column:payment_group_requested_at;not null;default:now()" json:"payment_group_requested_at"`
	PaymentGroupExpiresAt        *time.Time     `gorm:"column:payment_group_expires_at" json:"payment_group_expires_at,omitempty"`
	PaymentGroupPaidAt           *time.Time     `gorm:"column:payment_group_paid_at" json:"payment_group_paid_at,omitempty"`
	PaymentGroupCanceledAt       *time.Time     `gorm:"column:payment_group_canceled_at" json:"payment_group_canceled_at,omitempty"`
	PaymentGroupFailedAt         *time.Time     `gorm:"column:payment_group_failed_at" json:"payment_group_failed_at,omitempty"`
	PaymentGroupRefundedAt       *time.Time     `gorm:"column:payment_group_refunded_at" json:"payment_group_refunded_at,omitempty"`
	PaymentGroupDescription      *string        `gorm:"column:payment_group_description" json:"payment_group_description,omitempty"`
	PaymentGroupNote             *string        `gorm:"column:payment_group_note" json:"payment_group_note,omitempty"`
	PaymentGroupMeta             datatypes.JSON `gorm:"column:payment_group_meta;type:jsonb" json:"payment_group_meta,omitempty"`
	PaymentGroupCreatedAt        time.Time      `gorm:"column:payment_group_created_at;not null;default:now()" json:"payment_group_created_at"`
	PaymentGroupUpdatedAt        time.Time      `gorm:"column:payment_group_updated_at;not null;default:now()" json:"payment_group_updated_at"`
	PaymentGroupDeletedAt        gorm.DeletedAt `gorm:"column:payment_group_deleted_at;index" json:"payment_group_deleted_at,omitempty"`
}

// TableName implements gorm.Tabler
func (PaymentGroupModel) TableName() string {
	return "payment_groups"
}
