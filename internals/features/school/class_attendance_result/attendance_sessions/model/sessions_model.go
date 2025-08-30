package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionModel struct {
	ClassAttendanceSessionId uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_sessions_id" json:"class_attendance_sessions_id"`

	ClassAttendanceSessionSectionId uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_section_id" json:"class_attendance_sessions_section_id"`
	ClassAttendanceSessionMasjidId  uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_masjid_id"  json:"class_attendance_sessions_masjid_id"`

	// ✅ gunakan class_subject (bukan subjects langsung)
	ClassAttendanceSessionClassSubjectId *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_class_subject_id" json:"class_attendance_sessions_class_subject_id,omitempty"`

	// CSS Teacher (penugasan per section+subject)
	ClassAttendanceSessionClassSectionSubjectTeacherId *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_class_section_subject_teacher_id" json:"class_attendance_sessions_class_section_subject_teacher_id,omitempty"`

	ClassAttendanceSessionDate        time.Time `gorm:"type:date;not null;column:class_attendance_sessions_date"      json:"class_attendance_sessions_date"`
	ClassAttendanceSessionTitle       *string   `gorm:"column:class_attendance_sessions_title"                         json:"class_attendance_sessions_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string    `gorm:"not null;column:class_attendance_sessions_general_info"         json:"class_attendance_sessions_general_info"`
	ClassAttendanceSessionNote        *string   `gorm:"column:class_attendance_sessions_note"                          json:"class_attendance_sessions_note,omitempty"`

	// GANTI: refer ke masjid_teachers, bukan users
	ClassAttendanceSessionTeacherId *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_teacher_id;references:masjid_teachers(masjid_teacher_id);onDelete:set null" json:"class_attendance_sessions_teacher_id,omitempty"`

	ClassAttendanceSessionCreatedAt time.Time      `gorm:"column:class_attendance_sessions_created_at;autoCreateTime"   json:"class_attendance_sessions_created_at"`
	ClassAttendanceSessionUpdatedAt *time.Time     `gorm:"column:class_attendance_sessions_updated_at;autoUpdateTime"   json:"class_attendance_sessions_updated_at,omitempty"`
	ClassAttendanceSessionDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_sessions_deleted_at;index"            json:"class_attendance_sessions_deleted_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string { return "class_attendance_sessions" }
