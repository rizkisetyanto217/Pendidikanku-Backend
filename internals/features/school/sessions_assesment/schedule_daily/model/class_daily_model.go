// file: internals/features/school/class_daily/model/class_daily_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClassDailyModel memetakan ke tabel class_daily
type ClassDailyModel struct {
	// PK
	ClassDailyID uuid.UUID `json:"class_daily_id" gorm:"type:uuid;primaryKey;column:class_daily_id;default:gen_random_uuid()"`

	// Tenant / scope
	ClassDailyMasjidID uuid.UUID `json:"class_daily_masjid_id" gorm:"type:uuid;not null;column:class_daily_masjid_id"`

	// Tanggal occurrence (DATE)
	ClassDailyDate time.Time `json:"class_daily_date" gorm:"type:date;not null;column:class_daily_date"`

	// Section wajib
	ClassDailySectionID uuid.UUID `json:"class_daily_section_id" gorm:"type:uuid;not null;column:class_daily_section_id"`

	// Flag aktif
	ClassDailyIsActive bool `json:"class_daily_is_active" gorm:"type:boolean;not null;default:true;column:class_daily_is_active"`

	// Kolom generated (read-only) 1=Mon .. 7=Sun
	ClassDailyDayOfWeek int `json:"class_daily_day_of_week" gorm:"->;column:class_daily_day_of_week"`

	// Timestamps eksplisit (sesuai skema SQL)
	ClassDailyCreatedAt time.Time      `json:"class_daily_created_at" gorm:"column:class_daily_created_at;not null;autoCreateTime"`
	ClassDailyUpdatedAt time.Time      `json:"class_daily_updated_at" gorm:"column:class_daily_updated_at;not null;autoUpdateTime"`
	ClassDailyDeletedAt gorm.DeletedAt `json:"class_daily_deleted_at,omitempty" gorm:"column:class_daily_deleted_at;index"`
}

// TableName override
func (ClassDailyModel) TableName() string {
	return "class_daily"
}
