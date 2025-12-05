// file: internals/features/school/schedules/model/class_schedule_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionStatus string

const (
	SessionStatusScheduled SessionStatus = "scheduled"
	SessionStatusOngoing   SessionStatus = "ongoing"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusCanceled  SessionStatus = "canceled"
)

type ClassScheduleModel struct {
	ClassScheduleID        uuid.UUID     `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_schedule_id" json:"class_schedule_id"`
	ClassScheduleSchoolID  uuid.UUID     `gorm:"type:uuid;not null;column:class_schedule_school_id" json:"class_schedule_school_id"`
	ClassScheduleSlug      *string       `gorm:"type:varchar(160);column:class_schedule_slug" json:"class_schedule_slug,omitempty"`
	ClassScheduleStartDate time.Time     `gorm:"type:date;not null;column:class_schedule_start_date" json:"class_schedule_start_date"`
	ClassScheduleEndDate   time.Time     `gorm:"type:date;not null;column:class_schedule_end_date" json:"class_schedule_end_date"`
	ClassScheduleStatus    SessionStatus `gorm:"type:session_status_enum;not null;default:'scheduled';column:class_schedule_status" json:"class_schedule_status"`
	ClassScheduleIsActive  bool          `gorm:"not null;default:true;column:class_schedule_is_active" json:"class_schedule_is_active"`

	ClassScheduleCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_schedule_created_at" json:"class_schedule_created_at"`
	ClassScheduleUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_schedule_updated_at" json:"class_schedule_updated_at"`
	ClassScheduleDeletedAt gorm.DeletedAt `gorm:"column:class_schedule_deleted_at;index" json:"class_schedule_deleted_at,omitempty"`
}

func (ClassScheduleModel) TableName() string { return "class_schedules" }
