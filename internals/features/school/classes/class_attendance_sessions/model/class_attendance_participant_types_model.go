// file: internals/features/attendance/model/class_attendance_session_participant_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionParticipantTypeModel struct {
	ClassAttendanceSessionParticipantTypeID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_session_participant_type_id" json:"class_attendance_session_participant_type_id"`
	ClassAttendanceSessionParticipantTypeSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_participant_type_school_id" json:"class_attendance_session_participant_type_school_id"`

	ClassAttendanceSessionParticipantTypeCode  string  `gorm:"type:varchar(32);not null;column:class_attendance_session_participant_type_code"  json:"class_attendance_session_participant_type_code"`
	ClassAttendanceSessionParticipantTypeLabel *string `gorm:"type:varchar(80);column:class_attendance_session_participant_type_label"          json:"class_attendance_session_participant_type_label,omitempty"`
	ClassAttendanceSessionParticipantTypeSlug  *string `gorm:"type:varchar(120);column:class_attendance_session_participant_type_slug"          json:"class_attendance_session_participant_type_slug,omitempty"`
	ClassAttendanceSessionParticipantTypeColor *string `gorm:"type:varchar(20);column:class_attendance_session_participant_type_color"          json:"class_attendance_session_participant_type_color,omitempty"`
	ClassAttendanceSessionParticipantTypeDesc  *string `gorm:"type:text;column:class_attendance_session_participant_type_desc"                  json:"class_attendance_session_participant_type_desc,omitempty"`

	ClassAttendanceSessionParticipantTypeIsActive bool `gorm:"not null;default:true;column:class_attendance_session_participant_type_is_active" json:"class_attendance_session_participant_type_is_active"`

	ClassAttendanceSessionParticipantTypeCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_participant_type_created_at" json:"class_attendance_session_participant_type_created_at"`
	ClassAttendanceSessionParticipantTypeUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_participant_type_updated_at" json:"class_attendance_session_participant_type_updated_at"`
	ClassAttendanceSessionParticipantTypeDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_participant_type_deleted_at;index"                                        json:"class_attendance_session_participant_type_deleted_at,omitempty"`
}

func (ClassAttendanceSessionParticipantTypeModel) TableName() string {
	return "class_attendance_session_participant_types"
}
