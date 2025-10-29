// file: internals/features/masjid/holidays/model/masjid_holiday_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

/* =====================
   MODEL
   ===================== */

type MasjidHoliday struct {
	// PK
	MasjidHolidayID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:masjid_holiday_id" json:"masjid_holiday_id"`

	// Tenant guard
	MasjidHolidayMasjidID uuid.UUID `gorm:"type:uuid;not null;column:masjid_holiday_masjid_id" json:"masjid_holiday_masjid_id"`

	// Identitas opsional
	MasjidHolidaySlug *string `gorm:"type:varchar(160);column:masjid_holiday_slug" json:"masjid_holiday_slug,omitempty"`

	// Tanggal rentang (wajib)
	MasjidHolidayStartDate time.Time `gorm:"type:date;not null;column:masjid_holiday_start_date" json:"masjid_holiday_start_date"`
	MasjidHolidayEndDate   time.Time `gorm:"type:date;not null;column:masjid_holiday_end_date" json:"masjid_holiday_end_date"`

	// Judul & alasan
	MasjidHolidayTitle  string  `gorm:"type:varchar(200);not null;column:masjid_holiday_title" json:"masjid_holiday_title"`
	MasjidHolidayReason *string `gorm:"type:text;column:masjid_holiday_reason" json:"masjid_holiday_reason,omitempty"`

	// Status
	MasjidHolidayIsActive          bool `gorm:"type:boolean;not null;default:true;column:masjid_holiday_is_active" json:"masjid_holiday_is_active"`
	MasjidHolidayIsRecurringYearly bool `gorm:"type:boolean;not null;default:false;column:masjid_holiday_is_recurring_yearly" json:"masjid_holiday_is_recurring_yearly"`

	// Audit
	MasjidHolidayCreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now();column:masjid_holiday_created_at" json:"masjid_holiday_created_at"`
	MasjidHolidayUpdatedAt time.Time  `gorm:"type:timestamptz;not null;default:now();column:masjid_holiday_updated_at" json:"masjid_holiday_updated_at"`
	MasjidHolidayDeletedAt *time.Time `gorm:"type:timestamptz;column:masjid_holiday_deleted_at" json:"masjid_holiday_deleted_at,omitempty"`
}

/* =====================
   TableName override
   ===================== */

func (MasjidHoliday) TableName() string {
	return "masjid_holidays"
}
