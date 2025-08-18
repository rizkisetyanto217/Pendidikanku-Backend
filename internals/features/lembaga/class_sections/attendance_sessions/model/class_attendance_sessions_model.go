// internals/features/lembaga/classes/attendance/main/model/attendance_models.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionModel struct {
	ClassAttendanceSessionId uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_session_id" json:"class_attendance_session_id"`

	ClassAttendanceSessionSectionId uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_section_id;index:idx_cas_section" json:"class_attendance_session_section_id"`
	ClassAttendanceSessionMasjidId  uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_masjid_id;index:idx_cas_masjid" json:"class_attendance_session_masjid_id"`

	// Integrasi mapel & penugasan
	ClassAttendanceSessionSubjectId *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_subject_id;index:idx_cas_subject" json:"class_attendance_session_subject_id,omitempty"`

	// pakai nama panjang
	ClassAttendanceSessionClassSectionSubjectTeacherId *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_class_section_subject_teacher_id;index:idx_cas_csst" json:"class_attendance_session_class_section_subject_teacher_id,omitempty"`

	ClassAttendanceSessionDate        time.Time  `gorm:"type:date;not null;column:class_attendance_session_date;index:idx_cas_date,sort:desc" json:"class_attendance_session_date"`
	ClassAttendanceSessionTitle       *string    `gorm:"column:class_attendance_session_title" json:"class_attendance_session_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string     `gorm:"not null;column:class_attendance_session_general_info" json:"class_attendance_session_general_info"`
	ClassAttendanceSessionNote        *string    `gorm:"column:class_attendance_session_note" json:"class_attendance_session_note,omitempty"`

	ClassAttendanceSessionTeacherUserId *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_teacher_user_id" json:"class_attendance_session_teacher_user_id,omitempty"`

	ClassAttendanceSessionCreatedAt time.Time      `gorm:"column:class_attendance_session_created_at;autoCreateTime" json:"class_attendance_session_created_at"`
	ClassAttendanceSessionUpdatedAt *time.Time     `gorm:"column:class_attendance_session_updated_at;autoUpdateTime" json:"class_attendance_session_updated_at,omitempty"`
	ClassAttendanceSessionDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_deleted_at;index" json:"class_attendance_session_deleted_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string {
	return "class_attendance_sessions"
}
