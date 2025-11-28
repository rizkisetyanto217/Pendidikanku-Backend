// file: internals/features/school/classes/attendance/dto/class_attendance_session_type_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	attendanceModel "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/model"
)

/* ======================================================
   DTO: class_attendance_session_types (Response)
====================================================== */

type ClassAttendanceSessionTypeDTO struct {
	// PK
	ClassAttendanceSessionTypeID uuid.UUID `json:"class_attendance_session_type_id"`

	// tenant
	ClassAttendanceSessionTypeSchoolID uuid.UUID `json:"class_attendance_session_type_school_id"`

	// identitas
	ClassAttendanceSessionTypeSlug        string  `json:"class_attendance_session_type_slug"`
	ClassAttendanceSessionTypeName        string  `json:"class_attendance_session_type_name"`
	ClassAttendanceSessionTypeDescription *string `json:"class_attendance_session_type_description,omitempty"`

	// tampilan
	ClassAttendanceSessionTypeColor *string `json:"class_attendance_session_type_color,omitempty"`
	ClassAttendanceSessionTypeIcon  *string `json:"class_attendance_session_type_icon,omitempty"`

	// control umum
	ClassAttendanceSessionTypeIsActive  bool `json:"class_attendance_session_type_is_active"`
	ClassAttendanceSessionTypeSortOrder int  `json:"class_attendance_session_type_sort_order"`

	// konfigurasi attendance (flag dasar)
	ClassAttendanceSessionTypeAllowStudentSelfAttendance bool `json:"class_attendance_session_type_allow_student_self_attendance"`
	ClassAttendanceSessionTypeAllowTeacherMarkAttendance bool `json:"class_attendance_session_type_allow_teacher_mark_attendance"`
	ClassAttendanceSessionTypeRequireTeacherAttendance   bool `json:"class_attendance_session_type_require_teacher_attendance"`

	// window absensi
	ClassAttendanceSessionTypeAttendanceWindowMode         string `json:"class_attendance_session_type_attendance_window_mode"`
	ClassAttendanceSessionTypeAttendanceOpenOffsetMinutes  *int   `json:"class_attendance_session_type_attendance_open_offset_minutes,omitempty"`
	ClassAttendanceSessionTypeAttendanceCloseOffsetMinutes *int   `json:"class_attendance_session_type_attendance_close_offset_minutes,omitempty"`

	// enum[] di DB → []string di response
	ClassAttendanceSessionTypeRequireAttendanceReason []string `json:"class_attendance_session_type_require_attendance_reason"`

	// meta fleksibel
	ClassAttendanceSessionTypeMeta map[string]any `json:"class_attendance_session_type_meta,omitempty"`

	// audit
	ClassAttendanceSessionTypeCreatedAt time.Time      `json:"class_attendance_session_type_created_at"`
	ClassAttendanceSessionTypeUpdatedAt time.Time      `json:"class_attendance_session_type_updated_at"`
	ClassAttendanceSessionTypeDeletedAt gorm.DeletedAt `json:"class_attendance_session_type_deleted_at,omitempty"`
}

/* ======================================================
   Mapper: Model -> DTO
====================================================== */

func NewClassAttendanceSessionTypeDTO(m *attendanceModel.ClassAttendanceSessionTypeModel) *ClassAttendanceSessionTypeDTO {
	if m == nil {
		return nil
	}

	return &ClassAttendanceSessionTypeDTO{
		ClassAttendanceSessionTypeID:          m.ClassAttendanceSessionTypeID,
		ClassAttendanceSessionTypeSchoolID:    m.ClassAttendanceSessionTypeSchoolID,
		ClassAttendanceSessionTypeSlug:        m.ClassAttendanceSessionTypeSlug,
		ClassAttendanceSessionTypeName:        m.ClassAttendanceSessionTypeName,
		ClassAttendanceSessionTypeDescription: m.ClassAttendanceSessionTypeDescription,
		ClassAttendanceSessionTypeColor:       m.ClassAttendanceSessionTypeColor,
		ClassAttendanceSessionTypeIcon:        m.ClassAttendanceSessionTypeIcon,
		ClassAttendanceSessionTypeIsActive:    m.ClassAttendanceSessionTypeIsActive,
		ClassAttendanceSessionTypeSortOrder:   m.ClassAttendanceSessionTypeSortOrder,

		ClassAttendanceSessionTypeAllowStudentSelfAttendance: m.ClassAttendanceSessionTypeAllowStudentSelfAttendance,
		ClassAttendanceSessionTypeAllowTeacherMarkAttendance: m.ClassAttendanceSessionTypeAllowTeacherMarkAttendance,
		ClassAttendanceSessionTypeRequireTeacherAttendance:   m.ClassAttendanceSessionTypeRequireTeacherAttendance,

		// window mode & offset
		ClassAttendanceSessionTypeAttendanceWindowMode:         string(m.ClassAttendanceSessionTypeAttendanceWindowMode),
		ClassAttendanceSessionTypeAttendanceOpenOffsetMinutes:  m.ClassAttendanceSessionTypeAttendanceOpenOffsetMinutes,
		ClassAttendanceSessionTypeAttendanceCloseOffsetMinutes: m.ClassAttendanceSessionTypeAttendanceCloseOffsetMinutes,

		// pq.StringArray → []string
		ClassAttendanceSessionTypeRequireAttendanceReason: []string(m.ClassAttendanceSessionTypeRequireAttendanceReason),

		// datatypes.JSONMap → map[string]any
		ClassAttendanceSessionTypeMeta: m.ClassAttendanceSessionTypeMeta,

		ClassAttendanceSessionTypeCreatedAt: m.ClassAttendanceSessionTypeCreatedAt,
		ClassAttendanceSessionTypeUpdatedAt: m.ClassAttendanceSessionTypeUpdatedAt,
		ClassAttendanceSessionTypeDeletedAt: m.ClassAttendanceSessionTypeDeletedAt,
	}
}

/* ======================================================
   Mapper: Slice Model -> Slice DTO
====================================================== */

func NewClassAttendanceSessionTypeDTOs(list []*attendanceModel.ClassAttendanceSessionTypeModel) []*ClassAttendanceSessionTypeDTO {
	if len(list) == 0 {
		return []*ClassAttendanceSessionTypeDTO{}
	}

	out := make([]*ClassAttendanceSessionTypeDTO, 0, len(list))
	for _, m := range list {
		out = append(out, NewClassAttendanceSessionTypeDTO(m))
	}
	return out
}
