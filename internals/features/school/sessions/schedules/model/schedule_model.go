// internals/features/lembaga/class_schedules/model/class_schedule_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SessionStatusEnum merepresentasikan enum session_status_enum di Postgres.
type SessionStatusEnum string

const (
	SessionScheduled SessionStatusEnum = "scheduled"
	SessionOngoing   SessionStatusEnum = "ongoing"
	SessionCompleted SessionStatusEnum = "completed"
	SessionCanceled  SessionStatusEnum = "canceled"
)

type ClassScheduleModel struct {
	ClassScheduleID uuid.UUID `gorm:"column:class_schedule_id;type:uuid;default:gen_random_uuid();primaryKey" json:"class_schedule_id"`

	// tenant scope
	ClassSchedulesMasjidID uuid.UUID `gorm:"column:class_schedules_masjid_id;type:uuid;not null" json:"class_schedules_masjid_id"`

	// slug (opsional; unik per tenant saat alive, index dibuat di migration)
	ClassSchedulesSlug *string `gorm:"column:class_schedules_slug;type:varchar(160)" json:"class_schedules_slug,omitempty"`

	// masa berlaku
	ClassSchedulesStartDate time.Time `gorm:"column:class_schedules_start_date;type:date;not null" json:"class_schedules_start_date"`
	ClassSchedulesEndDate   time.Time `gorm:"column:class_schedules_end_date;type:date;not null"   json:"class_schedules_end_date"`

	// status & metadata
	ClassSchedulesStatus    SessionStatusEnum `gorm:"column:class_schedules_status;type:session_status_enum;not null;default:'scheduled'" json:"class_schedules_status"`
	ClassSchedulesIsActive  bool              `gorm:"column:class_schedules_is_active;not null;default:true"                                json:"class_schedules_is_active"`

	// audit
	ClassSchedulesCreatedAt time.Time      `gorm:"column:class_schedules_created_at;type:timestamptz;not null;autoCreateTime" json:"class_schedules_created_at"`
	ClassSchedulesUpdatedAt time.Time      `gorm:"column:class_schedules_updated_at;type:timestamptz;not null;autoUpdateTime" json:"class_schedules_updated_at"`
	ClassSchedulesDeletedAt gorm.DeletedAt `gorm:"column:class_schedules_deleted_at;index"                                   json:"class_schedules_deleted_at,omitempty"`
}

func (ClassScheduleModel) TableName() string { return "class_schedules" }
