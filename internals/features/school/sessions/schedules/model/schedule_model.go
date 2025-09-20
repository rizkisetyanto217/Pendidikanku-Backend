// file: internals/features/school/schedules/model/class_schedule.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
   Enum
========================= */

type SessionStatus string

const (
	SessionScheduled SessionStatus = "scheduled"
	SessionOngoing   SessionStatus = "ongoing"
	SessionCompleted SessionStatus = "completed"
	SessionCanceled  SessionStatus = "canceled"
)

/* =========================
   Model: ClassScheduleModel
========================= */

type ClassScheduleModel struct {
	// PK
	ClassScheduleID uuid.UUID `gorm:"type:uuid;primaryKey;column:class_schedule_id"`

	// Tenant
	ClassScheduleMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_schedules_masjid_id;index"`

	// Assignment (CSST) — opsional
	ClassScheduleCSSTID *uuid.UUID                  `gorm:"type:uuid;column:class_schedules_csst_id;index"`
	ClassScheduleCSST   *ClassSectionSubjectTeacher `gorm:"foreignKey:ClassScheduleCSSTID;references:ClassSectionSubjectTeachersID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	// Event — opsional
	ClassScheduleEventID *uuid.UUID  `gorm:"type:uuid;column:class_schedules_event_id;index"`
	ClassScheduleEvent   *ClassEvent `gorm:"foreignKey:ClassScheduleEventID;references:ClassEventsID;constraint:OnDelete:SET NULL"`

	// Pola berulang
	ClassScheduleDayOfWeek int       `gorm:"column:class_schedules_day_of_week;not null"` // 1..7
	ClassScheduleStartTime time.Time `gorm:"column:class_schedules_start_time;type:time;not null"`
	ClassScheduleEndTime   time.Time `gorm:"column:class_schedules_end_time;type:time;not null"`

	// Batas berlaku
	ClassScheduleStartDate time.Time `gorm:"column:class_schedules_start_date;type:date;not null"`
	ClassScheduleEndDate   time.Time `gorm:"column:class_schedules_end_date;type:date;not null"`

	// Status & metadata
	ClassScheduleStatus   SessionStatus `gorm:"type:session_status_enum;default:'scheduled';not null;column:class_schedules_status"`
	ClassScheduleIsActive bool          `gorm:"column:class_schedules_is_active;default:true;not null"`

	// Timestamps
	ClassScheduleCreatedAt time.Time      `gorm:"column:class_schedules_created_at;autoCreateTime"`
	ClassScheduleUpdatedAt time.Time      `gorm:"column:class_schedules_updated_at;autoUpdateTime"`
	ClassScheduleDeletedAt gorm.DeletedAt `gorm:"column:class_schedules_deleted_at;index"`
}

func (ClassScheduleModel) TableName() string { return "class_schedules" }

func (cs *ClassScheduleModel) BeforeCreate(tx *gorm.DB) error {
	if cs.ClassScheduleID == uuid.Nil {
		cs.ClassScheduleID = uuid.New()
	}
	return nil
}

/* =========================
   Referenced models (PK eksplisit)
========================= */

type ClassSectionSubjectTeacher struct {
	ClassSectionSubjectTeachersID uuid.UUID `gorm:"type:uuid;primaryKey;column:class_section_subject_teachers_id"`
}

func (ClassSectionSubjectTeacher) TableName() string {
	return "class_section_subject_teachers"
}

type ClassEvent struct {
	ClassEventsID uuid.UUID `gorm:"type:uuid;primaryKey;column:class_events_id"`
}

func (ClassEvent) TableName() string { return "class_events" }
