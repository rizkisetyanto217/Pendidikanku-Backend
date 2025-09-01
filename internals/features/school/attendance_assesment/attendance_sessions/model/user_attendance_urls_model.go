package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserAttendanceURLModel struct {
	UserAttendanceURLsID uuid.UUID `json:"user_attendance_urls_id" gorm:"column:user_attendance_urls_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// Tenant scope
	UserAttendanceURLsMasjidID uuid.UUID `json:"user_attendance_urls_masjid_id" gorm:"column:user_attendance_urls_masjid_id;type:uuid;not null;index:idx_user_attendance_urls_masjid_created_at,priority:1"`

	// Relasi ke parent attendance
	UserAttendanceURLsAttendanceID uuid.UUID `json:"user_attendance_urls_attendance_id" gorm:"column:user_attendance_urls_attendance_id;type:uuid;not null;index:idx_user_attendance_urls_attendance;index:idx_uau_attendance_alive,priority:1"`

	// Metadata
	UserAttendanceURLsLabel *string `json:"user_attendance_urls_label,omitempty" gorm:"column:user_attendance_urls_label;type:varchar(120)"`

	// URL aktif (wajib)
	UserAttendanceURLsHref string `json:"user_attendance_urls_href" gorm:"column:user_attendance_urls_href;type:text;not null"`

	// Housekeeping (opsional)
	UserAttendanceURLsTrashURL           *string    `json:"user_attendance_urls_trash_url,omitempty" gorm:"column:user_attendance_urls_trash_url;type:text"`
	UserAttendanceURLsDeletePendingUntil *time.Time `json:"user_attendance_urls_delete_pending_until,omitempty" gorm:"column:user_attendance_urls_delete_pending_until"`

	// Uploader (opsional)
	UserAttendanceURLsUploaderTeacherID *uuid.UUID `json:"user_attendance_urls_uploader_teacher_id,omitempty" gorm:"column:user_attendance_urls_uploader_teacher_id;type:uuid;index:idx_uau_uploader_teacher"`
	UserAttendanceURLsUploaderUserID    *uuid.UUID `json:"user_attendance_urls_uploader_user_id,omitempty" gorm:"column:user_attendance_urls_uploader_user_id;type:uuid;index:idx_uau_uploader_user"`

	// Timestamps & soft delete
	UserAttendanceURLsCreatedAt time.Time      `json:"user_attendance_urls_created_at" gorm:"column:user_attendance_urls_created_at;autoCreateTime"`
	UserAttendanceURLsUpdatedAt time.Time      `json:"user_attendance_urls_updated_at" gorm:"column:user_attendance_urls_updated_at;autoUpdateTime"`
	UserAttendanceURLsDeletedAt gorm.DeletedAt `json:"user_attendance_urls_deleted_at,omitempty" gorm:"column:user_attendance_urls_deleted_at;index"`
}

func (UserAttendanceURLModel) TableName() string {
	return "user_attendance_urls"
}
