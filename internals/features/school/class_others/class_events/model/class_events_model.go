// file: internals/features/school/class_events/model/class_event_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
   ENUMS
========================================================= */

// Mengacu ke tipe PostgreSQL: class_delivery_mode_enum ('online','offline','hybrid')
type ClassDeliveryMode string

const (
	ClassDeliveryModeOnline  ClassDeliveryMode = "online"
	ClassDeliveryModeOffline ClassDeliveryMode = "offline"
	ClassDeliveryModeHybrid  ClassDeliveryMode = "hybrid"
)

// Kolom di DB pakai CHECK ('open'|'invite'|'closed')
type ClassEventEnrollmentPolicy string

const (
	ClassEventPolicyOpen   ClassEventEnrollmentPolicy = "open"
	ClassEventPolicyInvite ClassEventEnrollmentPolicy = "invite"
	ClassEventPolicyClosed ClassEventEnrollmentPolicy = "closed"
)

/* =========================================================
   MODEL: class_events
========================================================= */

type ClassEventModel struct {
	ClassEventID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_event_id" json:"class_event_id"`
	ClassEventSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_event_school_id" json:"class_event_school_id"`

	// optional relations
	ClassEventThemeID    *uuid.UUID `gorm:"type:uuid;column:class_event_theme_id" json:"class_event_theme_id,omitempty"`
	ClassEventScheduleID *uuid.UUID `gorm:"type:uuid;column:class_event_schedule_id" json:"class_event_schedule_id,omitempty"`

	// optional targets (one of them)
	ClassEventSectionID      *uuid.UUID `gorm:"type:uuid;column:class_event_section_id" json:"class_event_section_id,omitempty"`
	ClassEventClassID        *uuid.UUID `gorm:"type:uuid;column:class_event_class_id" json:"class_event_class_id,omitempty"`
	ClassEventClassSubjectID *uuid.UUID `gorm:"type:uuid;column:class_event_class_subject_id" json:"class_event_class_subject_id,omitempty"`

	// core info
	ClassEventTitle string  `gorm:"type:varchar(160);not null;column:class_event_title" json:"class_event_title"`
	ClassEventDesc  *string `gorm:"type:text;column:class_event_desc" json:"class_event_desc,omitempty"`

	// dates & times
	ClassEventDate      time.Time  `gorm:"type:date;not null;column:class_event_date" json:"class_event_date"`
	ClassEventEndDate   *time.Time `gorm:"type:date;column:class_event_end_date" json:"class_event_end_date,omitempty"`
	ClassEventStartTime *time.Time `gorm:"type:time;column:class_event_start_time" json:"class_event_start_time,omitempty"`
	ClassEventEndTime   *time.Time `gorm:"type:time;column:class_event_end_time" json:"class_event_end_time,omitempty"`

	// location / delivery
	ClassEventDeliveryMode *ClassDeliveryMode `gorm:"type:class_delivery_mode_enum;column:class_event_delivery_mode" json:"class_event_delivery_mode,omitempty"`
	ClassEventRoomID       *uuid.UUID         `gorm:"type:uuid;column:class_event_room_id" json:"class_event_room_id,omitempty"`

	// teacher (internal/guest)
	ClassEventTeacherID   *uuid.UUID `gorm:"type:uuid;column:class_event_teacher_id" json:"class_event_teacher_id,omitempty"`
	ClassEventTeacherName *string    `gorm:"type:text;column:class_event_teacher_name" json:"class_event_teacher_name,omitempty"`
	ClassEventTeacherDesc *string    `gorm:"type:text;column:class_event_teacher_desc" json:"class_event_teacher_desc,omitempty"`

	// capacity & RSVP
	ClassEventCapacity         *int                        `gorm:"column:class_event_capacity" json:"class_event_capacity,omitempty"`
	ClassEventEnrollmentPolicy *ClassEventEnrollmentPolicy `gorm:"type:varchar(16);column:class_event_enrollment_policy" json:"class_event_enrollment_policy,omitempty"`

	// status
	ClassEventIsActive bool `gorm:"not null;default:true;column:class_event_is_active" json:"class_event_is_active"`

	// audit
	ClassEventCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:class_event_created_at" json:"class_event_created_at"`
	ClassEventUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:class_event_updated_at" json:"class_event_updated_at"`
	ClassEventDeletedAt gorm.DeletedAt `gorm:"column:class_event_deleted_at;index" json:"class_event_deleted_at,omitempty"`
}

func (ClassEventModel) TableName() string { return "class_events" }
