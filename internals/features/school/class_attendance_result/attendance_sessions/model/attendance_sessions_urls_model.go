// file: internals/features/school/class_attendance_sessions/model/class_attendance_session_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
 * CLASS_ATTENDANCE_SESSION_URL (multi URL per sesi)
 * ========================================================= */
type ClassAttendanceSessionURLModel struct {
	ClassAttendanceSessionURLID  uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_session_url_id" json:"class_attendance_session_url_id"`
	ClassAttendanceSessionURLMasjidID  uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_url_masjid_id" json:"class_attendance_session_url_masjid_id"`
	ClassAttendanceSessionURLSessionID uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_url_session_id" json:"class_attendance_session_url_session_id"`

	ClassAttendanceSessionURLLabel *string `gorm:"type:varchar(120);column:class_attendance_session_url_label" json:"class_attendance_session_url_label,omitempty"`

	ClassAttendanceSessionURLHref string  `gorm:"type:text;not null;column:class_attendance_session_url_href" json:"class_attendance_session_url_href"`
	ClassAttendanceSessionURLTrashURL *string   `gorm:"type:text;column:class_attendance_session_url_trash_url" json:"class_attendance_session_url_trash_url,omitempty"`
	ClassAttendanceSessionURLDeletePendingUntil *time.Time `gorm:"column:class_attendance_session_url_delete_pending_until" json:"class_attendance_session_url_delete_pending_until,omitempty"`

	ClassAttendanceSessionURLCreatedAt time.Time      `gorm:"column:class_attendance_session_url_created_at;autoCreateTime" json:"class_attendance_session_url_created_at"`
	ClassAttendanceSessionURLUpdatedAt time.Time      `gorm:"column:class_attendance_session_url_updated_at;autoUpdateTime" json:"class_attendance_session_url_updated_at"`
	ClassAttendanceSessionURLDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_url_deleted_at;index" json:"-"`
}

func (ClassAttendanceSessionURLModel) TableName() string {
	return "class_attendance_session_url"
}
