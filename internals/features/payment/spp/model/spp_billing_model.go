package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SppBillingModel struct {
	SppBillingID uuid.UUID `gorm:"column:spp_billing_id;type:uuid;default:gen_random_uuid();primaryKey" json:"spp_billing_id"`

	// FK (nullable → SET NULL)
	SppBillingMasjidID *uuid.UUID `gorm:"column:spp_billing_masjid_id;type:uuid" json:"spp_billing_masjid_id,omitempty"`
	SppBillingClassID  *uuid.UUID `gorm:"column:spp_billing_class_id;type:uuid" json:"spp_billing_class_id,omitempty"`

	// Periode
	SppBillingMonth int16 `gorm:"column:spp_billing_month;type:smallint;not null" json:"spp_billing_month"` // 1..12
	SppBillingYear  int16 `gorm:"column:spp_billing_year;type:smallint;not null"  json:"spp_billing_year"`  // 2000..2100

	// Info batch/tagihan
	SppBillingTitle    string     `gorm:"column:spp_billing_title;type:text;not null" json:"spp_billing_title"`
	SppBillingDueDate  *time.Time `gorm:"column:spp_billing_due_date;type:date"       json:"spp_billing_due_date,omitempty"`
	SppBillingNote     *string    `gorm:"column:spp_billing_note;type:text"           json:"spp_billing_note,omitempty"`

	// Timestamps (DB trigger akan set updated_at)
	SppBillingCreatedAt time.Time       `gorm:"column:spp_billing_created_at;autoCreateTime" json:"spp_billing_created_at"`
	SppBillingUpdatedAt *time.Time      `gorm:"column:spp_billing_updated_at;autoUpdateTime" json:"spp_billing_updated_at,omitempty"`
	SppBillingDeletedAt gorm.DeletedAt  `gorm:"column:spp_billing_deleted_at;index"          json:"spp_billing_deleted_at,omitempty"`

	// (Opsional) preload relasi, kalau perlu:
	// Masjid *MasjidModel `gorm:"foreignKey:SppBillingMasjidID;references:MasjidID" json:"-"`
	// Class  *ClassModel  `gorm:"foreignKey:SppBillingClassID;references:ClassID"   json:"-"`
}

func (SppBillingModel) TableName() string { return "spp_billings" }
