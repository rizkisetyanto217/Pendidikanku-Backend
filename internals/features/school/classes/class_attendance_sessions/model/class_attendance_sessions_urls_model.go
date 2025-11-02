// file: internals/features/school/class_attendance_sessions/model/class_attendance_session_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionURLModel struct {
	// PK
	ClassAttendanceSessionURLID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_session_url_id" json:"class_attendance_session_url_id"`

	// Tenant & Owner
	ClassAttendanceSessionURLSchoolID  uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_url_school_id" json:"class_attendance_session_url_school_id"`
	ClassAttendanceSessionURLSessionID uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_url_session_id" json:"class_attendance_session_url_session_id"`

	// Jenis/peran aset
	ClassAttendanceSessionURLKind string `gorm:"type:varchar(24);not null;column:class_attendance_session_url_kind" json:"class_attendance_session_url_kind"`

	// Lokasi file/link
	ClassAttendanceSessionURLHref         *string `gorm:"type:text;column:class_attendance_session_url_href" json:"class_attendance_session_url_href,omitempty"`
	ClassAttendanceSessionURLObjectKey    *string `gorm:"type:text;column:class_attendance_session_url_object_key" json:"class_attendance_session_url_object_key,omitempty"`
	ClassAttendanceSessionURLObjectKeyOld *string `gorm:"type:text;column:class_attendance_session_url_object_key_old" json:"class_attendance_session_url_object_key_old,omitempty"`

	// Tampilan
	ClassAttendanceSessionURLLabel     *string `gorm:"type:varchar(160);column:class_attendance_session_url_label" json:"class_attendance_session_url_label,omitempty"`
	ClassAttendanceSessionURLOrder     int     `gorm:"type:int;not null;default:0;column:class_attendance_session_url_order" json:"class_attendance_session_url_order"`
	ClassAttendanceSessionURLIsPrimary bool    `gorm:"type:boolean;not null;default:false;column:class_attendance_session_url_is_primary" json:"class_attendance_session_url_is_primary"`

	// Audit & Retensi
	ClassAttendanceSessionURLCreatedAt          time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_url_created_at" json:"class_attendance_session_url_created_at"`
	ClassAttendanceSessionURLUpdatedAt          time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_url_updated_at" json:"class_attendance_session_url_updated_at"`
	ClassAttendanceSessionURLDeletedAt          gorm.DeletedAt `gorm:"type:timestamptz;index;column:class_attendance_session_url_deleted_at" json:"class_attendance_session_url_deleted_at,omitempty"`
	ClassAttendanceSessionURLDeletePendingUntil *time.Time     `gorm:"type:timestamptz;column:class_attendance_session_url_delete_pending_until" json:"class_attendance_session_url_delete_pending_until,omitempty"`
}

// TableName override
func (ClassAttendanceSessionURLModel) TableName() string {
	return "class_attendance_session_urls"
}
