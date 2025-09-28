// file: internals/features/school/academics/classes/model/class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClassModel merepresentasikan tabel `classes` sesuai DDL.
type ClassModel struct {
	// PK & Tenant
	ClassID       uuid.UUID `json:"class_id"        gorm:"column:class_id;type:uuid;default:gen_random_uuid();primaryKey"`
	ClassMasjidID uuid.UUID `json:"class_masjid_id" gorm:"column:class_masjid_id;type:uuid;not null"`

	// Parent (FK komposit di DB dg masjid)
	ClassParentID uuid.UUID `json:"class_parent_id" gorm:"column:class_parent_id;type:uuid;not null"`

	// Identitas (slug)
	ClassSlug string `json:"class_slug" gorm:"column:class_slug;type:varchar(160);not null"`

	// Periode kelas
	ClassStartDate *time.Time `json:"class_start_date,omitempty" gorm:"column:class_start_date;type:date"`
	ClassEndDate   *time.Time `json:"class_end_date,omitempty"   gorm:"column:class_end_date;type:date"`

	// Registrasi / Term
	ClassTermID               *uuid.UUID `json:"class_term_id,omitempty"               gorm:"column:class_term_id;type:uuid"`
	ClassRegistrationOpensAt  *time.Time `json:"class_registration_opens_at,omitempty"  gorm:"column:class_registration_opens_at;type:timestamptz"`
	ClassRegistrationClosesAt *time.Time `json:"class_registration_closes_at,omitempty" gorm:"column:class_registration_closes_at;type:timestamptz"`

	// Kuota
	ClassQuotaTotal *int `json:"class_quota_total,omitempty" gorm:"column:class_quota_total"`
	ClassQuotaTaken int  `json:"class_quota_taken"           gorm:"column:class_quota_taken;not null;default:0"`

	// Pricing / Billing
	ClassRegistrationFeeIDR *int64  `json:"class_registration_fee_idr,omitempty" gorm:"column:class_registration_fee_idr"`
	ClassTuitionFeeIDR      *int64  `json:"class_tuition_fee_idr,omitempty"      gorm:"column:class_tuition_fee_idr"`
	ClassBillingCycle       string  `json:"class_billing_cycle"                  gorm:"column:class_billing_cycle;type:billing_cycle_enum;not null;default:'monthly'"`
	ClassProviderProductID  *string `json:"class_provider_product_id,omitempty"  gorm:"column:class_provider_product_id;type:text"`
	ClassProviderPriceID    *string `json:"class_provider_price_id,omitempty"    gorm:"column:class_provider_price_id;type:text"`

	// Catatan
	ClassNotes *string `json:"class_notes,omitempty" gorm:"column:class_notes;type:text"`

	// Mode & Status
	ClassDeliveryMode *string    `json:"class_delivery_mode,omitempty" gorm:"column:class_delivery_mode;type:class_delivery_mode_enum"`
	ClassStatus       string     `json:"class_status"                  gorm:"column:class_status;type:class_status_enum;not null;default:'active'"`
	ClassCompletedAt  *time.Time `json:"class_completed_at,omitempty"  gorm:"column:class_completed_at;type:timestamptz"`

	// Single image (2-slot + retensi)
	ClassImageURL                *string    `json:"class_image_url,omitempty"                 gorm:"column:class_image_url;type:text"`
	ClassImageObjectKey          *string    `json:"class_image_object_key,omitempty"          gorm:"column:class_image_object_key;type:text"`
	ClassImageURLOld             *string    `json:"class_image_url_old,omitempty"             gorm:"column:class_image_url_old;type:text"`
	ClassImageObjectKeyOld       *string    `json:"class_image_object_key_old,omitempty"      gorm:"column:class_image_object_key_old;type:text"`
	ClassImageDeletePendingUntil *time.Time `json:"class_image_delete_pending_until,omitempty" gorm:"column:class_image_delete_pending_until;type:timestamptz"`

	// Snapshot Class Parent
	ClassCodeParentSnapshot  *string `json:"class_code_parent_snapshot,omitempty"  gorm:"column:class_code_parent_snapshot;type:varchar(40)"`
	ClassNameParentSnapshot  *string `json:"class_name_parent_snapshot,omitempty"  gorm:"column:class_name_parent_snapshot;type:varchar(80)"`
	ClassSlugParentSnapshot  *string `json:"class_slug_parent_snapshot,omitempty"  gorm:"column:class_slug_parent_snapshot;type:varchar(160)"`
	ClassLevelParentSnapshot *int16  `json:"class_level_parent_snapshot,omitempty" gorm:"column:class_level_parent_snapshot;type:smallint"`

	// Snapshot Class Term
	ClassAcademicYearTermSnapshot *string `json:"class_academic_year_term_snapshot,omitempty" gorm:"column:class_academic_year_term_snapshot;type:varchar(40)"`
	ClassNameTermSnapshot         *string `json:"class_name_term_snapshot,omitempty"         gorm:"column:class_name_term_snapshot;type:varchar(100)"`
	ClassSlugTermSnapshot         *string `json:"class_slug_term_snapshot,omitempty"         gorm:"column:class_slug_term_snapshot;type:varchar(160)"`
	ClassAngkatanTermSnapshot     *string `json:"class_angkatan_term_snapshot,omitempty"     gorm:"column:class_angkatan_term_snapshot;type:varchar(40)"`

	// Audit & soft delete
	ClassCreatedAt time.Time      `json:"class_created_at"           gorm:"column:class_created_at;type:timestamptz;not null;default:now()"`
	ClassUpdatedAt time.Time      `json:"class_updated_at"           gorm:"column:class_updated_at;type:timestamptz;not null;default:now()"`
	ClassDeletedAt gorm.DeletedAt `json:"class_deleted_at,omitempty" gorm:"column:class_deleted_at;type:timestamptz;index"`
}

func (ClassModel) TableName() string { return "classes" }

// Enum constants agar konsisten dengan DB
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
