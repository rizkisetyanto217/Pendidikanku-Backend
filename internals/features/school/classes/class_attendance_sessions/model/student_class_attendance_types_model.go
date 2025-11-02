// file: internals/features/attendance/model/student_class_session_attendance_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StudentClassSessionAttendanceTypeModel struct {
	StudentClassSessionAttendanceTypeID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_class_session_attendance_type_id" json:"student_class_session_attendance_type_id"`
	StudentClassSessionAttendanceTypeSchoolID uuid.UUID `gorm:"type:uuid;not null;column:student_class_session_attendance_type_school_id" json:"student_class_session_attendance_type_school_id"`

	StudentClassSessionAttendanceTypeCode  string  `gorm:"type:varchar(32);not null;column:student_class_session_attendance_type_code"  json:"student_class_session_attendance_type_code"`
	StudentClassSessionAttendanceTypeLabel *string `gorm:"type:varchar(80);column:student_class_session_attendance_type_label"          json:"student_class_session_attendance_type_label,omitempty"`
	StudentClassSessionAttendanceTypeSlug  *string `gorm:"type:varchar(120);column:student_class_session_attendance_type_slug"          json:"student_class_session_attendance_type_slug,omitempty"`
	StudentClassSessionAttendanceTypeColor *string `gorm:"type:varchar(20);column:student_class_session_attendance_type_color"          json:"student_class_session_attendance_type_color,omitempty"`
	StudentClassSessionAttendanceTypeDesc  *string `gorm:"type:text;column:student_class_session_attendance_type_desc"                  json:"student_class_session_attendance_type_desc,omitempty"`

	StudentClassSessionAttendanceTypeIsActive bool `gorm:"not null;default:true;column:student_class_session_attendance_type_is_active" json:"student_class_session_attendance_type_is_active"`

	StudentClassSessionAttendanceTypeCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_class_session_attendance_type_created_at" json:"student_class_session_attendance_type_created_at"`
	StudentClassSessionAttendanceTypeUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_class_session_attendance_type_updated_at" json:"student_class_session_attendance_type_updated_at"`
	StudentClassSessionAttendanceTypeDeletedAt gorm.DeletedAt `gorm:"column:student_class_session_attendance_type_deleted_at;index"                                                     json:"student_class_session_attendance_type_deleted_at,omitempty"`
}

func (StudentClassSessionAttendanceTypeModel) TableName() string {
	return "student_class_session_attendance_types"
}
