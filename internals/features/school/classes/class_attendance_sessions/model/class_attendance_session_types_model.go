// file: internals/features/school/classes/attendance/model/class_attendance_session_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ======================================================
   Enum: AttendanceWindowMode (mirror enum SQL)
   SQL: attendance_window_mode_enum
====================================================== */

type AttendanceWindowMode string

const (
	AttendanceWindowModeAnytime        AttendanceWindowMode = "anytime"         // bebas kapan saja
	AttendanceWindowModeSameDay        AttendanceWindowMode = "same_day"        // hanya di hari H
	AttendanceWindowModeThreeDays      AttendanceWindowMode = "three_days"      // H-1, H, H+1
	AttendanceWindowModeSessionTime    AttendanceWindowMode = "session_time"    // hanya saat sesi berlangsung
	AttendanceWindowModeRelativeWindow AttendanceWindowMode = "relative_window" // pakai offset menit
)

/* Helper: convert []AttendanceState <-> pq.StringArray
   (tetap aman kalau ada value kosong / tidak dikenal)
*/

func AttendanceStatesToStringArray(states []AttendanceState) pq.StringArray {
	if len(states) == 0 {
		return pq.StringArray{}
	}
	out := make(pq.StringArray, 0, len(states))
	for _, s := range states {
		if s == "" {
			continue
		}
		out = append(out, string(s))
	}
	return out
}

func StringArrayToAttendanceStates(arr pq.StringArray) []AttendanceState {
	if len(arr) == 0 {
		return []AttendanceState{}
	}
	out := make([]AttendanceState, 0, len(arr))
	for _, s := range arr {
		if s == "" {
			continue
		}
		out = append(out, AttendanceState(s))
	}
	return out
}

/* ======================================================
   Model: class_attendance_session_types
   (master tipe sesi, per tenant)
====================================================== */

type ClassAttendanceSessionTypeModel struct {
	// PK
	ClassAttendanceSessionTypeID uuid.UUID `gorm:"column:class_attendance_session_type_id;type:uuid;default:gen_random_uuid();primaryKey" json:"class_attendance_session_type_id"`

	// tenant
	ClassAttendanceSessionTypeSchoolID uuid.UUID `gorm:"column:class_attendance_session_type_school_id;type:uuid;not null" json:"class_attendance_session_type_school_id"`

	// identitas
	ClassAttendanceSessionTypeSlug        string  `gorm:"column:class_attendance_session_type_slug;type:varchar(160);not null" json:"class_attendance_session_type_slug"`
	ClassAttendanceSessionTypeName        string  `gorm:"column:class_attendance_session_type_name;type:text;not null" json:"class_attendance_session_type_name"`
	ClassAttendanceSessionTypeDescription *string `gorm:"column:class_attendance_session_type_description;type:text" json:"class_attendance_session_type_description,omitempty"`

	// tampilan (opsional)
	ClassAttendanceSessionTypeColor *string `gorm:"column:class_attendance_session_type_color;type:text" json:"class_attendance_session_type_color,omitempty"`
	ClassAttendanceSessionTypeIcon  *string `gorm:"column:class_attendance_session_type_icon;type:text" json:"class_attendance_session_type_icon,omitempty"`

	// control umum
	ClassAttendanceSessionTypeIsActive  bool `gorm:"column:class_attendance_session_type_is_active;not null;default:true" json:"class_attendance_session_type_is_active"`
	ClassAttendanceSessionTypeSortOrder int  `gorm:"column:class_attendance_session_type_sort_order;not null;default:0" json:"class_attendance_session_type_sort_order"`

	// konfigurasi attendance (flag dasar)
	ClassAttendanceSessionTypeAllowStudentSelfAttendance bool `gorm:"column:class_attendance_session_type_allow_student_self_attendance;not null;default:true" json:"class_attendance_session_type_allow_student_self_attendance"`
	ClassAttendanceSessionTypeAllowTeacherMarkAttendance bool `gorm:"column:class_attendance_session_type_allow_teacher_mark_attendance;not null;default:true" json:"class_attendance_session_type_allow_teacher_mark_attendance"`
	ClassAttendanceSessionTypeRequireTeacherAttendance   bool `gorm:"column:class_attendance_session_type_require_teacher_attendance;not null;default:true" json:"class_attendance_session_type_require_teacher_attendance"`

	// window absensi (waktu boleh absen)
	// SQL: class_attendance_session_type_attendance_window_mode attendance_window_mode_enum NOT NULL DEFAULT 'same_day'
	ClassAttendanceSessionTypeAttendanceWindowMode         AttendanceWindowMode `gorm:"column:class_attendance_session_type_attendance_window_mode;type:attendance_window_mode_enum;not null;default:same_day" json:"class_attendance_session_type_attendance_window_mode"`
	ClassAttendanceSessionTypeAttendanceOpenOffsetMinutes  *int                 `gorm:"column:class_attendance_session_type_attendance_open_offset_minutes" json:"class_attendance_session_type_attendance_open_offset_minutes,omitempty"`
	ClassAttendanceSessionTypeAttendanceCloseOffsetMinutes *int                 `gorm:"column:class_attendance_session_type_attendance_close_offset_minutes" json:"class_attendance_session_type_attendance_close_offset_minutes,omitempty"`

	// array enum: attendance_state_enum[]
	// default di DB: ARRAY['unmarked']::attendance_state_enum[]
	ClassAttendanceSessionTypeRequireAttendanceReason pq.StringArray `gorm:"column:class_attendance_session_type_require_attendance_reason;type:attendance_state_enum[]" json:"class_attendance_session_type_require_attendance_reason"`

	// meta fleksibel (JSONB)
	ClassAttendanceSessionTypeMeta datatypes.JSONMap `gorm:"column:class_attendance_session_type_meta;type:jsonb" json:"class_attendance_session_type_meta,omitempty"`

	// audit
	ClassAttendanceSessionTypeCreatedAt time.Time      `gorm:"column:class_attendance_session_type_created_at;autoCreateTime" json:"class_attendance_session_type_created_at"`
	ClassAttendanceSessionTypeUpdatedAt time.Time      `gorm:"column:class_attendance_session_type_updated_at;autoUpdateTime" json:"class_attendance_session_type_updated_at"`
	ClassAttendanceSessionTypeDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_type_deleted_at;index" json:"class_attendance_session_type_deleted_at,omitempty"`
}

// TableName overrides the table name used by GORM
func (ClassAttendanceSessionTypeModel) TableName() string {
	return "class_attendance_session_types"
}

/* ======================================================
   Convenience methods di Model
====================================================== */

// GetRequireAttendanceStates balikin dalam bentuk []AttendanceState
func (m *ClassAttendanceSessionTypeModel) GetRequireAttendanceStates() []AttendanceState {
	return StringArrayToAttendanceStates(m.ClassAttendanceSessionTypeRequireAttendanceReason)
}

// SetRequireAttendanceStates set field enum[] dari []AttendanceState
func (m *ClassAttendanceSessionTypeModel) SetRequireAttendanceStates(states []AttendanceState) {
	m.ClassAttendanceSessionTypeRequireAttendanceReason = AttendanceStatesToStringArray(states)
}
