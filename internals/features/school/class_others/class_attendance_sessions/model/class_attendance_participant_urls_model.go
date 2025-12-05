// file: internals/features/attendance/model/class_attendance_session_participant_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionParticipantURLModel struct {
	ClassAttendanceSessionParticipantURLID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_session_participant_url_id" json:"class_attendance_session_participant_url_id"`

	// Tenant & owner
	ClassAttendanceSessionParticipantURLSchoolID      uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_participant_url_school_id" json:"class_attendance_session_participant_url_school_id"`
	ClassAttendanceSessionParticipantURLParticipantID uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_participant_url_participant_id" json:"class_attendance_session_participant_url_participant_id"`

	// optional FK ke type
	ClassAttendanceSessionParticipantTypeID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_participant_type_id" json:"class_attendance_session_participant_type_id,omitempty"`

	// data utama
	ClassAttendanceSessionParticipantURLKind string  `gorm:"type:varchar(24);not null;column:class_attendance_session_participant_url_kind" json:"class_attendance_session_participant_url_kind"`
	ClassAttendanceSessionParticipantURL     *string `gorm:"type:text;column:class_attendance_session_participant_url" json:"class_attendance_session_participant_url,omitempty"`

	// dua-slot object storage
	ClassAttendanceSessionParticipantURLObjectKey    *string `gorm:"type:text;column:class_attendance_session_participant_url_object_key" json:"class_attendance_session_participant_url_object_key,omitempty"`
	ClassAttendanceSessionParticipantURLOld          *string `gorm:"type:text;column:class_attendance_session_participant_url_old" json:"class_attendance_session_participant_url_old,omitempty"`
	ClassAttendanceSessionParticipantURLObjectKeyOld *string `gorm:"type:text;column:class_attendance_session_participant_url_object_key_old" json:"class_attendance_session_participant_url_object_key_old,omitempty"`

	// housekeeping / retensi
	ClassAttendanceSessionParticipantURLDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:class_attendance_session_participant_url_delete_pending_until" json:"class_attendance_session_participant_url_delete_pending_until,omitempty"`

	// metadata tampilan
	ClassAttendanceSessionParticipantURLLabel     *string `gorm:"type:varchar(160);column:class_attendance_session_participant_url_label" json:"class_attendance_session_participant_url_label,omitempty"`
	ClassAttendanceSessionParticipantURLOrder     int     `gorm:"type:int;not null;default:0;column:class_attendance_session_participant_url_order" json:"class_attendance_session_participant_url_order"`
	ClassAttendanceSessionParticipantURLIsPrimary bool    `gorm:"not null;default:false;column:class_attendance_session_participant_url_is_primary" json:"class_attendance_session_participant_url_is_primary"`

	// uploader (opsional)
	ClassAttendanceSessionParticipantURLUploaderTeacherID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_participant_url_uploader_teacher_id" json:"class_attendance_session_participant_url_uploader_teacher_id,omitempty"`
	ClassAttendanceSessionParticipantURLUploaderStudentID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_participant_url_uploader_student_id" json:"class_attendance_session_participant_url_uploader_student_id,omitempty"`

	// audit
	ClassAttendanceSessionParticipantURLCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_participant_url_created_at" json:"class_attendance_session_participant_url_created_at"`
	ClassAttendanceSessionParticipantURLUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_participant_url_updated_at" json:"class_attendance_session_participant_url_updated_at"`
	ClassAttendanceSessionParticipantURLDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_participant_url_deleted_at;index" json:"class_attendance_session_participant_url_deleted_at,omitempty"`
}

func (ClassAttendanceSessionParticipantURLModel) TableName() string {
	return "class_attendance_session_participant_urls"
}
