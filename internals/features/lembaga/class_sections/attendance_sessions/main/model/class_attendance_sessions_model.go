// internals/features/lembaga/classes/attendance/main/model/attendance_models.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// ==============================
// 1) class_attendance_sessions
// ==============================

type ClassAttendanceSessionModel struct {
	ClassAttendanceSessionID uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_sessions_id" json:"class_attendance_sessions_id"`
	SectionID                uuid.UUID  `gorm:"type:uuid;not null;column:class_attendance_sessions_section_id;index:idx_cas_section;uniqueIndex:uq_cas_section_date,priority:1" json:"class_attendance_sessions_section_id"`
	MasjidID                 uuid.UUID  `gorm:"type:uuid;not null;column:class_attendance_sessions_masjid_id;index:idx_cas_masjid" json:"class_attendance_sessions_masjid_id"`

	Date         time.Time `gorm:"type:date;not null;column:class_attendance_sessions_date;index:idx_cas_date,sort:desc;uniqueIndex:uq_cas_section_date,priority:2" json:"class_attendance_sessions_date"`
	Title        *string   `gorm:"column:class_attendance_sessions_title" json:"class_attendance_sessions_title,omitempty"`
	GeneralInfo  string    `gorm:"not null;column:class_attendance_sessions_general_info" json:"class_attendance_sessions_general_info"`
	Note         *string   `gorm:"column:class_attendance_sessions_note" json:"class_attendance_sessions_note,omitempty"`
	TeacherUserID *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_teacher_user_id" json:"class_attendance_sessions_teacher_user_id,omitempty"`

	CreatedAt time.Time  `gorm:"column:class_attendance_sessions_created_at;autoCreateTime" json:"class_attendance_sessions_created_at"`
	UpdatedAt *time.Time `gorm:"column:class_attendance_sessions_updated_at;autoUpdateTime" json:"class_attendance_sessions_updated_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string {
	return "class_attendance_sessions"
}