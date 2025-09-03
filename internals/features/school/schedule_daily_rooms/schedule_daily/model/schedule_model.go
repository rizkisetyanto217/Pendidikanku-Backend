// file: internals/features/school/class_schedules/model/class_schedule_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =======================================================
   Enum status (menyesuaikan session_status_enum)
   ======================================================= */

type SessionStatus string

const (
	SessionScheduled SessionStatus = "scheduled"
	SessionOngoing   SessionStatus = "ongoing"
	SessionFinished  SessionStatus = "finished"
	SessionCanceled  SessionStatus = "canceled"
)

/* =======================================================
   ClassScheduleModel — map ke tabel class_schedules
   ======================================================= */

type ClassScheduleModel struct {
	// PK
	ClassScheduleID uuid.UUID `json:"class_schedule_id" gorm:"type:uuid;primaryKey;column:class_schedule_id;default:gen_random_uuid()"`

	// Tenant / scope
	ClassSchedulesMasjidID uuid.UUID `json:"class_schedules_masjid_id" gorm:"type:uuid;not null;column:class_schedules_masjid_id"`

	// Induk jadwal → section
	ClassSchedulesSectionID uuid.UUID `json:"class_schedules_section_id" gorm:"type:uuid;not null;column:class_schedules_section_id"`

	// Opsional override
	ClassSchedulesSubjectID  *uuid.UUID `json:"class_schedules_subject_id,omitempty" gorm:"type:uuid;column:class_schedules_subject_id"`
	ClassSchedulesSemesterID *uuid.UUID `json:"class_schedules_semester_id,omitempty" gorm:"type:uuid;column:class_schedules_semester_id"`
	ClassSchedulesTeacherID  *uuid.UUID `json:"class_schedules_teacher_id,omitempty" gorm:"type:uuid;column:class_schedules_teacher_id"`

	// Room (nullable)
	ClassSchedulesRoomID *uuid.UUID `json:"class_schedules_room_id,omitempty" gorm:"type:uuid;column:class_schedules_room_id"`

	// Pola berulang
	ClassSchedulesDayOfWeek int       `json:"class_schedules_day_of_week" gorm:"type:int;not null;column:class_schedules_day_of_week"` // 1..7
	ClassSchedulesStartTime time.Time `json:"class_schedules_start_time" gorm:"type:time;not null;column:class_schedules_start_time"`
	ClassSchedulesEndTime   time.Time `json:"class_schedules_end_time" gorm:"type:time;not null;column:class_schedules_end_time"`

	// Batas berlaku
	ClassSchedulesStartDate time.Time `json:"class_schedules_start_date" gorm:"type:date;not null;column:class_schedules_start_date"`
	ClassSchedulesEndDate   time.Time `json:"class_schedules_end_date" gorm:"type:date;not null;column:class_schedules_end_date"`

	// Status & metadata
	ClassSchedulesStatus    SessionStatus `json:"class_schedules_status" gorm:"type:text;not null;default:'scheduled';column:class_schedules_status"`
	ClassSchedulesIsActive  bool          `json:"class_schedules_is_active" gorm:"type:boolean;not null;default:true;column:class_schedules_is_active"`
	ClassSchedulesRoomLabel *string       `json:"class_schedules_room_label,omitempty" gorm:"type:text;column:class_schedules_room_label"`

	// Kolom generated (read-only)
	ClassSchedulesTimeRange *string `json:"class_schedules_time_range,omitempty" gorm:"->;column:class_schedules_time_range"`

	// Timestamps eksplisit (auto create/update)
	ClassSchedulesCreatedAt time.Time      `json:"class_schedules_created_at" gorm:"column:class_schedules_created_at;not null;autoCreateTime"`
	ClassSchedulesUpdatedAt time.Time      `json:"class_schedules_updated_at" gorm:"column:class_schedules_updated_at;not null;autoUpdateTime"`
	ClassSchedulesDeletedAt gorm.DeletedAt `json:"class_schedules_deleted_at" gorm:"column:class_schedules_deleted_at;index"`
}

/* =======================================================
   Table name
   ======================================================= */

func (ClassScheduleModel) TableName() string {
	return "class_schedules"
}
