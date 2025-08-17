// internals/features/payment/spp/model/user_spp_billing_item_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Optional: enum kecil biar aman saat pakai di code
type UserSppBillingStatus string

const (
	SppUnpaid   UserSppBillingStatus = "unpaid"
	SppPaid     UserSppBillingStatus = "paid"
	SppCanceled UserSppBillingStatus = "canceled"
)

type UserSppBillingModel struct {
	// PK
	UserSppBillingID uuid.UUID `json:"user_spp_billing_id" gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_spp_billing_id"`

	// FK ke header batch (NOT NULL)
	UserSppBillingBillingID uuid.UUID `json:"user_spp_billing_billing_id" gorm:"type:uuid;not null;column:user_spp_billing_billing_id;index:idx_user_spp_billings_billing"`

	// FK ke users.id (nullable → ON DELETE SET NULL)
	UserSppBillingUserID *uuid.UUID `json:"user_spp_billing_user_id,omitempty" gorm:"type:uuid;column:user_spp_billing_user_id;index:idx_user_spp_billings_user"`

	// Data tagihan per user
	UserSppBillingAmountIDR int                   `json:"user_spp_billing_amount_idr" gorm:"not null;check:user_spp_billing_amount_idr >= 0;column:user_spp_billing_amount_idr"`
	UserSppBillingStatus    UserSppBillingStatus  `json:"user_spp_billing_status"    gorm:"type:varchar(20);not null;default:unpaid;column:user_spp_billing_status"`
	UserSppBillingPaidAt    *time.Time            `json:"user_spp_billing_paid_at,omitempty" gorm:"column:user_spp_billing_paid_at"`
	UserSppBillingNote      *string               `json:"user_spp_billing_note,omitempty"    gorm:"type:text;column:user_spp_billing_note"`

	// Timestamps (DB trigger akan set updated_at)
	UserSppBillingCreatedAt time.Time      `json:"user_spp_billing_created_at" gorm:"autoCreateTime;column:user_spp_billing_created_at"`
	UserSppBillingUpdatedAt *time.Time     `json:"user_spp_billing_updated_at,omitempty" gorm:"autoUpdateTime;column:user_spp_billing_updated_at"`
	UserSppBillingDeletedAt gorm.DeletedAt `json:"user_spp_billing_deleted_at,omitempty" gorm:"index;column:user_spp_billing_deleted_at"`

	// ---- (Opsional) preload relasi; uncomment jika perlu + hindari import cycle ----
	// Billing *SppBillingModel `gorm:"foreignKey:UserSppBillingBillingID;references:SppBillingID" json:"-"`
	// User    *UserModel       `gorm:"foreignKey:UserSppBillingUserID;references:ID"           json:"-"`
}

// Nama tabel
func (UserSppBillingModel) TableName() string { return "user_spp_billings" }
