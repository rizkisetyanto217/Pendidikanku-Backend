package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===================== Constants ===================== */

const (
	DonationStatusPending   = "pending"
	DonationStatusPaid      = "paid"
	DonationStatusExpired   = "expired"
	DonationStatusCanceled  = "canceled"
	DonationStatusCompleted = "completed"
)

/* Mapping target type */
const (
	DonationTargetPost     = 1
	DonationTargetLecture  = 2
	DonationTargetCampaign = 3
	DonationTargetSPP      = 4
	DonationTargetOther    = 5
)

/* ===================== Model ===================== */

type Donation struct {
	DonationID uuid.UUID `gorm:"column:donation_id;type:uuid;default:gen_random_uuid();primaryKey" json:"donation_id"`

	// FK opsional
	DonationUserID  *uuid.UUID `gorm:"column:donation_user_id;type:uuid" json:"donation_user_id,omitempty"`
	DonationMasjidID *uuid.UUID `gorm:"column:donation_masjid_id;type:uuid" json:"donation_masjid_id,omitempty"`

	// Link ke SPP billing (opsional; pembayaran SPP)
	DonationUserSPPBillingID *uuid.UUID `gorm:"column:donation_user_spp_billing_id;type:uuid" json:"donation_user_spp_billing_id,omitempty"`

	// Grup order (opsional) untuk multi-item checkout
	DonationParentOrderID *string `gorm:"column:donation_parent_order_id;type:varchar(120)" json:"donation_parent_order_id,omitempty"`

	DonationName   string `gorm:"column:donation_name;type:varchar(50);not null" json:"donation_name"`
	DonationAmount int    `gorm:"column:donation_amount;not null;check:donation_amount > 0" json:"donation_amount"`

	// Rincian pembagian (opsional)
	DonationAmountMasjid           *int `gorm:"column:donation_amount_masjid" json:"donation_amount_masjid,omitempty"`
	DonationAmountMasjidku         *int `gorm:"column:donation_amount_masjidku" json:"donation_amount_masjidku,omitempty"`
	DonationAmountMasjidkuToMasjid *int `gorm:"column:donation_amount_masjidku_to_masjid" json:"donation_amount_masjidku_to_masjid,omitempty"`
	DonationAmountMasjidkuToApp    *int `gorm:"column:donation_amount_masjidku_to_app" json:"donation_amount_masjidku_to_app,omitempty"`

	DonationMessage *string `gorm:"column:donation_message;type:text" json:"donation_message,omitempty"`

	DonationStatus  string `gorm:"column:donation_status;type:varchar(20);default:'pending'" json:"donation_status"`
	DonationOrderID string `gorm:"column:donation_order_id;type:varchar(100);not null;unique" json:"donation_order_id"`

	// Target umum (non-SPP). XOR dengan DonationUserSPPBillingID
	DonationTargetType *int       `gorm:"column:donation_target_type" json:"donation_target_type,omitempty"` // 1=post,2=lecture,3=campaign,4=spp,5=other
	DonationTargetID   *uuid.UUID `gorm:"column:donation_target_id;type:uuid" json:"donation_target_id,omitempty"`

	DonationPaymentToken   *string `gorm:"column:donation_payment_token;type:text" json:"donation_payment_token,omitempty"`
	DonationPaymentGateway string  `gorm:"column:donation_payment_gateway;type:varchar(50);default:'midtrans'" json:"donation_payment_gateway"`
	DonationPaymentMethod  *string `gorm:"column:donation_payment_method;type:varchar(50)" json:"donation_payment_method,omitempty"`

	DonationPaidAt *time.Time `gorm:"column:donation_paid_at" json:"donation_paid_at,omitempty"`

	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

func (Donation) TableName() string { return "donations" }

/* ===================== Helpers ===================== */

// IsForSPP true jika ini donasi untuk SPP
func (d *Donation) IsForSPP() bool {
	return d.DonationUserSPPBillingID != nil || (d.DonationTargetType != nil && *d.DonationTargetType == DonationTargetSPP)
}

// IsForGeneralTarget true jika ini donasi target umum (post/lecture/campaign/other)
func (d *Donation) IsForGeneralTarget() bool {
	return d.DonationTargetType != nil &&
		*d.DonationTargetType != DonationTargetSPP &&
		d.DonationTargetID != nil
}
