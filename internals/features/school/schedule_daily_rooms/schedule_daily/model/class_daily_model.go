// file: internals/features/school/class_daily/model/class_daily_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =======================================================
   Enum status (selaras dengan session_status_enum di DB)
   ======================================================= */
/* =======================================================
   ClassDailyModel — map ke tabel class_daily (occurrence harian)
   ======================================================= */

type ClassDailyModel struct {
	// PK
	ClassDailyID uuid.UUID `json:"class_daily_id" gorm:"type:uuid;primaryKey;column:class_daily_id;default:gen_random_uuid()"`

	// Tenant / scope
	ClassDailyMasjidID uuid.UUID `json:"class_daily_masjid_id" gorm:"type:uuid;not null;column:class_daily_masjid_id"`

	// Tanggal occurrence
	ClassDailyDate time.Time `json:"class_daily_date" gorm:"type:date;not null;column:class_daily_date"`

	// Link sumber (opsional)
	ClassDailyScheduleID   *uuid.UUID `json:"class_daily_schedule_id,omitempty" gorm:"type:uuid;column:class_daily_schedule_id"`
	ClassDailyAttendanceID *uuid.UUID `json:"class_daily_attendance_id,omitempty" gorm:"type:uuid;column:class_daily_attendance_id"`

	// Section wajib
	ClassDailySectionID uuid.UUID `json:"class_daily_section_id" gorm:"type:uuid;not null;column:class_daily_section_id"`

	// Snapshot/override (opsional)
	ClassDailySubjectID       *uuid.UUID `json:"class_daily_subject_id,omitempty" gorm:"type:uuid;column:class_daily_subject_id"`
	ClassDailyAcademicTermsID *uuid.UUID `json:"class_daily_academic_terms_id,omitempty" gorm:"type:uuid;column:class_daily_academic_terms_id"`
	ClassDailyTeacherID       *uuid.UUID `json:"class_daily_teacher_id,omitempty" gorm:"type:uuid;column:class_daily_teacher_id"`
	ClassDailyRoomID          *uuid.UUID `json:"class_daily_room_id,omitempty" gorm:"type:uuid;column:class_daily_room_id"`

	// Waktu pada tanggal tsb
	ClassDailyStartTime time.Time `json:"class_daily_start_time" gorm:"type:time;not null;column:class_daily_start_time"`
	ClassDailyEndTime   time.Time `json:"class_daily_end_time"   gorm:"type:time;not null;column:class_daily_end_time"`

	// Status & metadata
	ClassDailyStatus    SessionStatus `json:"class_daily_status"    gorm:"type:text;not null;default:'scheduled';column:class_daily_status"`
	ClassDailyIsActive  bool          `json:"class_daily_is_active" gorm:"type:boolean;not null;default:true;column:class_daily_is_active"`
	ClassDailyRoomLabel *string       `json:"class_daily_room_label,omitempty" gorm:"type:text;column:class_daily_room_label"`

	// Kolom generated (read-only)
	ClassDailyTimeRange *string `json:"class_daily_time_range,omitempty" gorm:"->;column:class_daily_time_range"`
	ClassDailyDayOfWeek int     `json:"class_daily_day_of_week" gorm:"->;column:class_daily_day_of_week"`

	// Timestamps eksplisit (auto create/update)
	ClassDailyCreatedAt time.Time      `json:"class_daily_created_at" gorm:"column:class_daily_created_at;not null;autoCreateTime"`
	ClassDailyUpdatedAt time.Time      `json:"class_daily_updated_at" gorm:"column:class_daily_updated_at;not null;autoUpdateTime"`
	ClassDailyDeletedAt gorm.DeletedAt `json:"class_daily_deleted_at" gorm:"column:class_daily_deleted_at;index"`
}

/* =======================================================
   Table name
   ======================================================= */

func (ClassDailyModel) TableName() string {
	return "class_daily"
}
