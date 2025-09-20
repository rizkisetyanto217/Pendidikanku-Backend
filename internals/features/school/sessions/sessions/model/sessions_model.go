// file: internals/features/school/class_attendance_sessions/model/class_attendance_session_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionModel struct {
	// PK
	ClassAttendanceSessionId uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_sessions_id" json:"class_attendance_sessions_id"`

	// Tenant
	ClassAttendanceSessionMasjidId uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_masjid_id" json:"class_attendance_sessions_masjid_id"`

	// Relasi utama: schedule (bukan CSST)
	ClassAttendanceSessionScheduleId uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_sessions_schedule_id" json:"class_attendance_sessions_schedule_id"`

	// Opsional: guru
	ClassAttendanceSessionTeacherId *uuid.UUID `gorm:"type:uuid;column:class_attendance_sessions_teacher_id" json:"class_attendance_sessions_teacher_id,omitempty"`

	// Tanggal sesi (DATE). Pointer agar default CURRENT_DATE terpakai bila nil saat insert.
	ClassAttendanceSessionDate *time.Time `gorm:"type:date;not null;default:CURRENT_DATE;column:class_attendance_sessions_date" json:"class_attendance_sessions_date"`

	// Metadata
	ClassAttendanceSessionTitle       *string `gorm:"column:class_attendance_sessions_title" json:"class_attendance_sessions_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string  `gorm:"not null;column:class_attendance_sessions_general_info" json:"class_attendance_sessions_general_info"`
	ClassAttendanceSessionNote        *string `gorm:"column:class_attendance_sessions_note" json:"class_attendance_sessions_note,omitempty"`

	// Rekap kehadiran (nullable di SQL → pointer di model)
	ClassAttendanceSessionPresentCount *int `gorm:"column:class_attendance_sessions_present_count" json:"class_attendance_sessions_present_count,omitempty"`
	ClassAttendanceSessionAbsentCount  *int `gorm:"column:class_attendance_sessions_absent_count" json:"class_attendance_sessions_absent_count,omitempty"`
	ClassAttendanceSessionLateCount    *int `gorm:"column:class_attendance_sessions_late_count" json:"class_attendance_sessions_late_count,omitempty"`
	ClassAttendanceSessionExcusedCount *int `gorm:"column:class_attendance_sessions_excused_count" json:"class_attendance_sessions_excused_count,omitempty"`
	ClassAttendanceSessionSickCount    *int `gorm:"column:class_attendance_sessions_sick_count" json:"class_attendance_sessions_sick_count,omitempty"`
	ClassAttendanceSessionLeaveCount   *int `gorm:"column:class_attendance_sessions_leave_count" json:"class_attendance_sessions_leave_count,omitempty"`

	// Audit & soft delete
	ClassAttendanceSessionCreatedAt time.Time      `gorm:"not null;default:now();autoCreateTime;column:class_attendance_sessions_created_at" json:"class_attendance_sessions_created_at"`
	ClassAttendanceSessionUpdatedAt time.Time      `gorm:"not null;default:now();autoUpdateTime;column:class_attendance_sessions_updated_at" json:"class_attendance_sessions_updated_at"`
	ClassAttendanceSessionDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_sessions_deleted_at;index" json:"class_attendance_sessions_deleted_at,omitempty"`
}

func (ClassAttendanceSessionModel) TableName() string { return "class_attendance_sessions" }
