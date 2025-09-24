// file: internals/features/attendance/model/user_class_session_attendance_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserClassSessionAttendanceTypeModel struct {
	UserClassSessionAttendanceTypeID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_session_attendance_type_id" json:"user_class_session_attendance_type_id"`
	UserClassSessionAttendanceTypeMasjidID  uuid.UUID      `gorm:"type:uuid;not null;column:user_class_session_attendance_type_masjid_id" json:"user_class_session_attendance_type_masjid_id"`

	UserClassSessionAttendanceTypeCode      string         `gorm:"type:varchar(32);not null;column:user_class_session_attendance_type_code"  json:"user_class_session_attendance_type_code"`
	UserClassSessionAttendanceTypeLabel     *string        `gorm:"type:varchar(80);column:user_class_session_attendance_type_label"          json:"user_class_session_attendance_type_label,omitempty"`
	UserClassSessionAttendanceTypeSlug      *string        `gorm:"type:varchar(120);column:user_class_session_attendance_type_slug"          json:"user_class_session_attendance_type_slug,omitempty"`
	UserClassSessionAttendanceTypeColor     *string        `gorm:"type:varchar(20);column:user_class_session_attendance_type_color"          json:"user_class_session_attendance_type_color,omitempty"`
	UserClassSessionAttendanceTypeDesc      *string        `gorm:"type:text;column:user_class_session_attendance_type_desc"                  json:"user_class_session_attendance_type_desc,omitempty"`

	UserClassSessionAttendanceTypeIsActive  bool           `gorm:"not null;default:true;column:user_class_session_attendance_type_is_active" json:"user_class_session_attendance_type_is_active"`

	UserClassSessionAttendanceTypeCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_session_attendance_type_created_at" json:"user_class_session_attendance_type_created_at"`
	UserClassSessionAttendanceTypeUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_session_attendance_type_updated_at" json:"user_class_session_attendance_type_updated_at"`
	UserClassSessionAttendanceTypeDeletedAt gorm.DeletedAt `gorm:"column:user_class_session_attendance_type_deleted_at;index"                                                     json:"user_class_session_attendance_type_deleted_at,omitempty"`
}

func (UserClassSessionAttendanceTypeModel) TableName() string {
	return "user_class_session_attendance_types"
}
