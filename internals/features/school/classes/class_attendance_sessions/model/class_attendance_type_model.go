// file: internals/features/school/classes/attendance/model/class_attendance_session_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ======================================================
   Model: class_attendance_session_types
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

	// tampilan
	ClassAttendanceSessionTypeColor *string `gorm:"column:class_attendance_session_type_color;type:text" json:"class_attendance_session_type_color,omitempty"`
	ClassAttendanceSessionTypeIcon  *string `gorm:"column:class_attendance_session_type_icon;type:text" json:"class_attendance_session_type_icon,omitempty"`

	// control
	ClassAttendanceSessionTypeIsActive  bool `gorm:"column:class_attendance_session_type_is_active;not null;default:true" json:"class_attendance_session_type_is_active"`
	ClassAttendanceSessionTypeSortOrder int  `gorm:"column:class_attendance_session_type_sort_order;not null;default:0" json:"class_attendance_session_type_sort_order"`

	// audit
	ClassAttendanceSessionTypeCreatedAt time.Time      `gorm:"column:class_attendance_session_type_created_at;autoCreateTime" json:"class_attendance_session_type_created_at"`
	ClassAttendanceSessionTypeUpdatedAt time.Time      `gorm:"column:class_attendance_session_type_updated_at;autoUpdateTime" json:"class_attendance_session_type_updated_at"`
	ClassAttendanceSessionTypeDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_type_deleted_at;index" json:"class_attendance_session_type_deleted_at,omitempty"`
}

// TableName overrides the table name used by GORM
func (ClassAttendanceSessionTypeModel) TableName() string {
	return "class_attendance_session_types"
}
