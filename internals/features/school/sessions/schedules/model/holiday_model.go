// internals/features/school/sessions/holidays/model/holiday_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HolidayModel struct {
	HolidayID       uuid.UUID `json:"holiday_id"                gorm:"column:holiday_id;type:uuid;primaryKey;default:gen_random_uuid()"`
	HolidayMasjidID uuid.UUID `json:"holiday_masjid_id"         gorm:"column:holiday_masjid_id;type:uuid;not null"`

	// NULLable slug (unik per tenant saat alive)
	HolidaySlug *string `json:"holiday_slug,omitempty"    gorm:"column:holiday_slug;type:varchar(160)"`

	HolidayStartDate time.Time `json:"holiday_start_date"        gorm:"column:holiday_start_date;type:date;not null"`
	HolidayEndDate   time.Time `json:"holiday_end_date"          gorm:"column:holiday_end_date;type:date;not null"`

	HolidayTitle  string  `json:"holiday_title"             gorm:"column:holiday_title;type:varchar(200);not null"`
	HolidayReason *string `json:"holiday_reason,omitempty"  gorm:"column:holiday_reason;type:text"`

	HolidayIsActive          bool `json:"holiday_is_active"         gorm:"column:holiday_is_active;not null;default:true"`
	HolidayIsRecurringYearly bool `json:"holiday_is_recurring_yearly" gorm:"column:holiday_is_recurring_yearly;not null;default:false"`

	HolidayCreatedAt time.Time      `json:"holiday_created_at"        gorm:"column:holiday_created_at;type:timestamptz;not null;autoCreateTime"`
	HolidayUpdatedAt time.Time      `json:"holiday_updated_at"        gorm:"column:holiday_updated_at;type:timestamptz;not null;autoUpdateTime"`
	HolidayDeletedAt gorm.DeletedAt `json:"holiday_deleted_at,omitempty" gorm:"column:holiday_deleted_at;index"`
}

func (HolidayModel) TableName() string { return "holidays" }
