package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Donation struct {
	DonationID uuid.UUID `gorm:"column:donation_id;type:uuid;default:gen_random_uuid();primaryKey" json:"donation_id"`

	DonationUserID *uuid.UUID `gorm:"column:donation_user_id;type:uuid" json:"donation_user_id,omitempty"`

	DonationName    string  `gorm:"column:donation_name;type:varchar(50);not null" json:"donation_name"`
	DonationAmount  int     `gorm:"column:donation_amount;not null;check:donation_amount > 0" json:"donation_amount"`
	DonationMessage string  `gorm:"column:donation_message;type:text" json:"donation_message"`

	DonationStatus string `gorm:"column:donation_status;type:varchar(20);default:'pending'" json:"donation_status"`

	DonationOrderID string `gorm:"column:donation_order_id;type:varchar(100);not null;unique" json:"donation_order_id"`

	DonationTargetType *int       `gorm:"column:donation_target_type" json:"donation_target_type,omitempty"` // 1 = post, 2 = lecture, etc.
	DonationTargetID   *uuid.UUID `gorm:"column:donation_target_id;type:uuid" json:"donation_target_id,omitempty"`

	DonationPaymentToken   string  `gorm:"column:donation_payment_token;type:text" json:"donation_payment_token"`
	DonationPaymentGateway string  `gorm:"column:donation_payment_gateway;type:varchar(50);default:'midtrans'" json:"donation_payment_gateway"`
	DonationPaymentMethod  string  `gorm:"column:donation_payment_method;type:varchar(50)" json:"donation_payment_method"`

	DonationPaidAt    *time.Time     `gorm:"column:donation_paid_at" json:"donation_paid_at,omitempty"`
	DonationMasjidID  *uuid.UUID     `gorm:"column:donation_masjid_id;type:uuid" json:"donation_masjid_id,omitempty"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

func (Donation) TableName() string {
	return "donations"
}
