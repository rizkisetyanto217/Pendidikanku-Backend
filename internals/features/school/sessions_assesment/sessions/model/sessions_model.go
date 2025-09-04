package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionModel struct {
	ClassAttendanceSessionId uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_sessions_id" json:"class_attendance_sessions_id"`

	// Wajib (FK tenant-safe di DB)
	ClassAttendanceSessionSectionId      uuid.UUID  `gorm:"type:uuid;not null;column:class_attendance_sessions_section_id"       json:"class_attendance_sessions_section_id"`
	ClassAttendanceSessionMasjidId       uuid.UUID  `gorm:"type:uuid;not null;column:class_attendance_sessions_masjid_id"        json:"class_attendance_sessions_masjid_id"`
	ClassAttendanceSessionClassSubjectId  uuid.UUID  `gorm:"type:uuid;not null;column:class_attendance_sessions_class_subject_id"  json:"class_attendance_sessions_class_subject_id"`

	// Optional room (FK → class_rooms)
	ClassAttendanceSessionClassRoomId *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_class_room_id" json:"class_attendance_sessions_class_room_id,omitempty"`

	// Data sesi
	// Saran: pakai pointer agar DB default CURRENT_DATE bisa kepakai saat nil (field di-skip saat insert)
	ClassAttendanceSessionDate        *time.Time `gorm:"type:date;not null;default:CURRENT_DATE;column:class_attendance_sessions_date" json:"class_attendance_sessions_date"`
	ClassAttendanceSessionTitle       *string    `gorm:"column:class_attendance_sessions_title"                                       json:"class_attendance_sessions_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string     `gorm:"not null;column:class_attendance_sessions_general_info"                      json:"class_attendance_sessions_general_info"`
	ClassAttendanceSessionNote        *string    `gorm:"column:class_attendance_sessions_note"                                       json:"class_attendance_sessions_note,omitempty"`

	// Guru (opsional) → masjid_teachers
	ClassAttendanceSessionTeacherId *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_teacher_id" json:"class_attendance_sessions_teacher_id,omitempty"`

	// Soft delete
	ClassAttendanceSessionDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_sessions_deleted_at;index" json:"class_attendance_sessions_deleted_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string { return "class_attendance_sessions" }
