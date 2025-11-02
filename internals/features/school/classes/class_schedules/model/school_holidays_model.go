// file: internals/features/school/holidays/model/school_holiday_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

/* =====================
   MODEL
   ===================== */

type SchoolHoliday struct {
	// PK
	SchoolHolidayID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:school_holiday_id" json:"school_holiday_id"`

	// Tenant guard
	SchoolHolidaySchoolID uuid.UUID `gorm:"type:uuid;not null;column:school_holiday_school_id" json:"school_holiday_school_id"`

	// Identitas opsional
	SchoolHolidaySlug *string `gorm:"type:varchar(160);column:school_holiday_slug" json:"school_holiday_slug,omitempty"`

	// Tanggal rentang (wajib)
	SchoolHolidayStartDate time.Time `gorm:"type:date;not null;column:school_holiday_start_date" json:"school_holiday_start_date"`
	SchoolHolidayEndDate   time.Time `gorm:"type:date;not null;column:school_holiday_end_date" json:"school_holiday_end_date"`

	// Judul & alasan
	SchoolHolidayTitle  string  `gorm:"type:varchar(200);not null;column:school_holiday_title" json:"school_holiday_title"`
	SchoolHolidayReason *string `gorm:"type:text;column:school_holiday_reason" json:"school_holiday_reason,omitempty"`

	// Status
	SchoolHolidayIsActive          bool `gorm:"type:boolean;not null;default:true;column:school_holiday_is_active" json:"school_holiday_is_active"`
	SchoolHolidayIsRecurringYearly bool `gorm:"type:boolean;not null;default:false;column:school_holiday_is_recurring_yearly" json:"school_holiday_is_recurring_yearly"`

	// Audit
	SchoolHolidayCreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now();column:school_holiday_created_at" json:"school_holiday_created_at"`
	SchoolHolidayUpdatedAt time.Time  `gorm:"type:timestamptz;not null;default:now();column:school_holiday_updated_at" json:"school_holiday_updated_at"`
	SchoolHolidayDeletedAt *time.Time `gorm:"type:timestamptz;column:school_holiday_deleted_at" json:"school_holiday_deleted_at,omitempty"`
}

/* =====================
   TableName override
   ===================== */

func (SchoolHoliday) TableName() string {
	return "school_holidays"
}
