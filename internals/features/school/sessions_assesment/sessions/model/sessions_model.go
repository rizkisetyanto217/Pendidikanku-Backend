package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionModel struct {
	ClassAttendanceSessionId uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_sessions_id" json:"class_attendance_sessions_id"`

	// Wajib sesuai schema
	ClassAttendanceSessionSectionId     uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_section_id"      json:"class_attendance_sessions_section_id"`
	ClassAttendanceSessionMasjidId      uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_masjid_id"       json:"class_attendance_sessions_masjid_id"`
	ClassAttendanceSessionClassSubjectId uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_class_subject_id" json:"class_attendance_sessions_class_subject_id"`

	// Data sesi
	ClassAttendanceSessionDate        time.Time `gorm:"type:date;not null;column:class_attendance_sessions_date"      json:"class_attendance_sessions_date"`
	ClassAttendanceSessionTitle       *string   `gorm:"column:class_attendance_sessions_title"                         json:"class_attendance_sessions_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string    `gorm:"not null;column:class_attendance_sessions_general_info"         json:"class_attendance_sessions_general_info"`
	ClassAttendanceSessionNote        *string   `gorm:"column:class_attendance_sessions_note"                          json:"class_attendance_sessions_note,omitempty"`

	// Guru yg mengajar (opsional) → FK sudah dibuat di migration SQL
	ClassAttendanceSessionTeacherId *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_teacher_id" json:"class_attendance_sessions_teacher_id,omitempty"`

	// Timestamps (updated_at NOT NULL di DB → gunakan non-pointer)
	ClassAttendanceSessionCreatedAt time.Time      `gorm:"column:class_attendance_sessions_created_at;autoCreateTime" json:"class_attendance_sessions_created_at"`
	ClassAttendanceSessionUpdatedAt time.Time      `gorm:"column:class_attendance_sessions_updated_at;autoUpdateTime" json:"class_attendance_sessions_updated_at"`
	ClassAttendanceSessionDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_sessions_deleted_at;index"          json:"class_attendance_sessions_deleted_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string { return "class_attendance_sessions" }
