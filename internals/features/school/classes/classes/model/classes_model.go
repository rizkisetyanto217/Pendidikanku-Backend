// models/class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClassModel merepresentasikan tabel `classes`
type ClassModel struct {
	// PK & tenant
	ClassID       uuid.UUID `gorm:"column:class_id;type:uuid;default:gen_random_uuid();primaryKey" json:"class_id"`
	ClassMasjidID uuid.UUID `gorm:"column:class_masjid_id;type:uuid;not null"                      json:"class_masjid_id"`

	// Parent (FK komposit di DB dg masjid)
	ClassParentID uuid.UUID `gorm:"column:class_parent_id;type:uuid;not null" json:"class_parent_id"`

	// Identitas (slug)
	ClassSlug string `gorm:"column:class_slug;type:varchar(160);not null" json:"class_slug"`

	// Periode kelas (DATE)
	ClassStartDate *time.Time `gorm:"column:class_start_date;type:date" json:"class_start_date,omitempty"`
	ClassEndDate   *time.Time `gorm:"column:class_end_date;type:date"   json:"class_end_date,omitempty"`

	// Registrasi / Term
	ClassTermID               *uuid.UUID `gorm:"column:class_term_id;type:uuid"          json:"class_term_id,omitempty"`
	ClassIsOpen               bool       `gorm:"column:class_is_open;not null;default:true" json:"class_is_open"`
	ClassRegistrationOpensAt  *time.Time `gorm:"column:class_registration_opens_at;type:timestamptz"  json:"class_registration_opens_at,omitempty"`
	ClassRegistrationClosesAt *time.Time `gorm:"column:class_registration_closes_at;type:timestamptz" json:"class_registration_closes_at,omitempty"`

	// Kuota
	ClassQuotaTotal *int `gorm:"column:class_quota_total"                     json:"class_quota_total,omitempty"`
	ClassQuotaTaken int   `gorm:"column:class_quota_taken;not null;default:0" json:"class_quota_taken"`

	// Pricing
	ClassRegistrationFeeIDR *int64  `gorm:"column:class_registration_fee_idr"                                           json:"class_registration_fee_idr,omitempty"`
	ClassTuitionFeeIDR      *int64  `gorm:"column:class_tuition_fee_idr"                                                json:"class_tuition_fee_idr,omitempty"`
	ClassBillingCycle       string  `gorm:"column:class_billing_cycle;type:billing_cycle_enum;not null;default:'monthly'" json:"class_billing_cycle"`
	ClassProviderProductID  *string `gorm:"column:class_provider_product_id"                                            json:"class_provider_product_id,omitempty"`
	ClassProviderPriceID    *string `gorm:"column:class_provider_price_id"                                              json:"class_provider_price_id,omitempty"`

	// Catatan & media
	ClassNotes    *string `gorm:"column:class_notes"     json:"class_notes,omitempty"`
	ClassImageURL *string `gorm:"column:class_image_url" json:"class_image_url,omitempty"`

	// Mode & Status
	ClassDeliveryMode string     `gorm:"column:class_delivery_mode;type:class_delivery_mode_enum"          json:"class_delivery_mode"`
	ClassStatus       string     `gorm:"column:class_status;type:class_status_enum;not null;default:'active'" json:"class_status"`
	ClassCompletedAt  *time.Time `gorm:"column:class_completed_at;type:timestamptz"                         json:"class_completed_at,omitempty"`

	// Penghapusan terjadwal / trash
	ClassTrashURL           *string    `gorm:"column:class_trash_url"                             json:"class_trash_url,omitempty"`
	ClassDeletePendingUntil *time.Time `gorm:"column:class_delete_pending_until;type:timestamptz" json:"class_delete_pending_until,omitempty"`

	// Timestamps (soft delete)
	ClassCreatedAt time.Time      `gorm:"column:class_created_at;type:timestamptz;not null;default:now()" json:"class_created_at"`
	ClassUpdatedAt time.Time      `gorm:"column:class_updated_at;type:timestamptz;not null;default:now()" json:"class_updated_at"`
	ClassDeletedAt gorm.DeletedAt `gorm:"column:class_deleted_at;type:timestamptz;index"                  json:"class_deleted_at,omitempty"`
}

func (ClassModel) TableName() string { return "classes" }

// (opsional) enum helpers — pastikan sesuai enum di DB
const (
	// delivery mode
	ClassDeliveryModeOffline = "offline"
	ClassDeliveryModeOnline  = "online"
	ClassDeliveryModeHybrid  = "hybrid"

	// billing cycle
	BillingCycleOneTime  = "one_time"
	BillingCycleMonthly  = "monthly"
	BillingCycleQuarter  = "quarterly"
	BillingCycleSemester = "semester"
	BillingCycleYearly   = "yearly"

	// status
	ClassStatusActive    = "active"
	ClassStatusInactive  = "inactive"
	ClassStatusCompleted = "completed"
)
