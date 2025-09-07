// file: internals/features/attendance/model/user_attendance_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserAttendanceTypeModel merepresentasikan tabel user_attendance_type
type UserAttendanceTypeModel struct {
	// Primary Key
	UserAttendanceTypeID uuid.UUID `gorm:"column:user_attendance_type_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"user_attendance_type_id"`

	// Relasi ke masjid (tenant scope)
	UserAttendanceTypeMasjidID uuid.UUID `gorm:"column:user_attendance_type_masjid_id;type:uuid;not null;index" json:"user_attendance_type_masjid_id"`

	// Informasi jenis attendance
	UserAttendanceTypeCode  string  `gorm:"column:user_attendance_type_code;type:varchar(32);not null" json:"user_attendance_type_code"`
	UserAttendanceTypeLabel *string `gorm:"column:user_attendance_type_label;type:varchar(80)" json:"user_attendance_type_label,omitempty"`
	UserAttendanceTypeDesc  *string `gorm:"column:user_attendance_type_desc" json:"user_attendance_type_desc,omitempty"`

	// Status aktif
	UserAttendanceTypeIsActive bool `gorm:"column:user_attendance_type_is_active;not null;default:true" json:"user_attendance_type_is_active"`

	// Timestamps
	UserAttendanceTypeCreatedAt time.Time      `gorm:"column:user_attendance_type_created_at;not null;default:now()" json:"user_attendance_type_created_at"`
	UserAttendanceTypeUpdatedAt time.Time      `gorm:"column:user_attendance_type_updated_at;not null;default:now()" json:"user_attendance_type_updated_at"`
	UserAttendanceTypeDeletedAt gorm.DeletedAt `gorm:"column:user_attendance_type_deleted_at;index" json:"user_attendance_type_deleted_at"`
}

// TableName untuk override nama tabel
func (UserAttendanceTypeModel) TableName() string {
	return "user_attendance_type"
}
