// file: internals/features/school/sessions/events/model/class_events_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===================== Enums (Go-side) ===================== */

// DB enum: class_delivery_mode_enum
// (biarkan nilai string mengikuti nilai enum di DB: mis. "onsite","online","hybrid", dll.)
type ClassDeliveryMode string

// Enrollment policy (varchar(16) di DB, dengan CHECK open|invite|closed)
type ClassEnrollmentPolicy string

const (
	EnrollOpen   ClassEnrollmentPolicy = "open"
	EnrollInvite ClassEnrollmentPolicy = "invite"
	EnrollClosed ClassEnrollmentPolicy = "closed"
)

/* ===================== Model ===================== */

type ClassEventModel struct {
	// PK & tenant
	ClassEventsID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_events_id"         json:"class_events_id"`
	ClassEventsMasjidID  uuid.UUID `gorm:"type:uuid;not null;column:class_events_masjid_id"                               json:"class_events_masjid_id"`

	// Optional references
	ClassEventsThemeID    *uuid.UUID `gorm:"type:uuid;column:class_events_theme_id"                                       json:"class_events_theme_id,omitempty"`
	ClassEventsScheduleID *uuid.UUID `gorm:"type:uuid;column:class_events_schedule_id"                                    json:"class_events_schedule_id,omitempty"`

	// Target minimal (opsional, salah satu)
	ClassEventsSectionID       *uuid.UUID `gorm:"type:uuid;column:class_events_section_id"        json:"class_events_section_id,omitempty"`
	ClassEventsClassID         *uuid.UUID `gorm:"type:uuid;column:class_events_class_id"          json:"class_events_class_id,omitempty"`
	ClassEventsClassSubjectID  *uuid.UUID `gorm:"type:uuid;column:class_events_class_subject_id"  json:"class_events_class_subject_id,omitempty"`

	// Info inti
	ClassEventsTitle string  `gorm:"type:varchar(160);not null;column:class_events_title"                                json:"class_events_title"`
	ClassEventsDesc  *string `gorm:"type:text;column:class_events_desc"                                                  json:"class_events_desc,omitempty"`

	// Waktu (DATE wajib, TIME opsional → pakai pointer dan type:time)
	ClassEventsDate      time.Time  `gorm:"type:date;not null;column:class_events_date"                                  json:"class_events_date"`
	ClassEventsEndDate   *time.Time `gorm:"type:date;column:class_events_end_date"                                       json:"class_events_end_date,omitempty"`
	ClassEventsStartTime *time.Time `gorm:"type:time;column:class_events_start_time"                                     json:"class_events_start_time,omitempty"`
	ClassEventsEndTime   *time.Time `gorm:"type:time;column:class_events_end_time"                                       json:"class_events_end_time,omitempty"`

	// Lokasi / delivery mode
	ClassEventsDeliveryMode *ClassDeliveryMode `gorm:"type:class_delivery_mode_enum;column:class_events_delivery_mode"   json:"class_events_delivery_mode,omitempty"`
	ClassEventsRoomID       *uuid.UUID         `gorm:"type:uuid;column:class_events_room_id"                              json:"class_events_room_id,omitempty"`

	// Pengajar (internal / tamu)
	ClassEventsTeacherID   *uuid.UUID `gorm:"type:uuid;column:class_events_teacher_id"                                    json:"class_events_teacher_id,omitempty"`
	ClassEventsTeacherName *string    `gorm:"type:text;column:class_events_teacher_name"                                  json:"class_events_teacher_name,omitempty"`
	ClassEventsTeacherDesc *string    `gorm:"type:text;column:class_events_teacher_desc"                                  json:"class_events_teacher_desc,omitempty"`

	// Kapasitas & RSVP
	ClassEventsCapacity        *int                  `gorm:"type:int;column:class_events_capacity"                        json:"class_events_capacity,omitempty"`
	ClassEventsEnrollmentPolicy *ClassEnrollmentPolicy `gorm:"type:varchar(16);column:class_events_enrollment_policy"     json:"class_events_enrollment_policy,omitempty"`

	// Status aktif
	ClassEventsIsActive bool `gorm:"not null;default:true;column:class_events_is_active"                                  json:"class_events_is_active"`

	// Audit
	ClassEventsCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:class_events_created_at" json:"class_events_created_at"`
	ClassEventsUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:class_events_updated_at" json:"class_events_updated_at"`
	ClassEventsDeletedAt gorm.DeletedAt `gorm:"column:class_events_deleted_at"                                                         json:"class_events_deleted_at,omitempty"`
}

func (ClassEventModel) TableName() string { return "class_events" }
