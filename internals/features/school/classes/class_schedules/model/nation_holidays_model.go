// file: internals/features/system/holidays/model/national_holiday_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NationalHolidayModel struct {
	// PK
	NationalHolidayID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:national_holiday_id" json:"national_holiday_id"`

	// Identitas opsional
	NationalHolidaySlug *string `gorm:"type:varchar(160);column:national_holiday_slug" json:"national_holiday_slug,omitempty"`

	// Tanggal (satu hari = start=end) atau rentang
	NationalHolidayStartDate time.Time `gorm:"type:date;not null;column:national_holiday_start_date" json:"national_holiday_start_date"`
	NationalHolidayEndDate   time.Time `gorm:"type:date;not null;column:national_holiday_end_date"   json:"national_holiday_end_date"`

	// Informasi
	NationalHolidayTitle  string  `gorm:"type:varchar(200);not null;column:national_holiday_title"  json:"national_holiday_title"`
	NationalHolidayReason *string `gorm:"type:text;column:national_holiday_reason"                  json:"national_holiday_reason,omitempty"`

	// Flags
	NationalHolidayIsActive          bool `gorm:"not null;default:true;column:national_holiday_is_active"             json:"national_holiday_is_active"`
	NationalHolidayIsRecurringYearly bool `gorm:"not null;default:false;column:national_holiday_is_recurring_yearly"  json:"national_holiday_is_recurring_yearly"`

	// Audit (pakai autoCreate/autoUpdate; soft delete pakai DeletedAt)
	NationalHolidayCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:national_holiday_created_at" json:"national_holiday_created_at"`
	NationalHolidayUpdatedAt time.Time      `gorm:"type:timestamptz;default:now();autoUpdateTime;column:national_holiday_updated_at"          json:"national_holiday_updated_at,omitempty"`
	NationalHolidayDeletedAt gorm.DeletedAt `gorm:"column:national_holiday_deleted_at"                                                       json:"national_holiday_deleted_at,omitempty"`
}

func (NationalHolidayModel) TableName() string { return "national_holidays" }
