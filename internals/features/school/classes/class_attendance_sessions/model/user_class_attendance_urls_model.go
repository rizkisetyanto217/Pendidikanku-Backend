// file: internals/features/attendance/model/user_class_session_attendance_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserClassSessionAttendanceURLModel struct {
	UserClassSessionAttendanceURLID                 uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_session_attendance_url_id"                 json:"user_class_session_attendance_url_id"`
	UserClassSessionAttendanceURLMasjidID           uuid.UUID      `gorm:"type:uuid;not null;column:user_class_session_attendance_url_masjid_id"                                   json:"user_class_session_attendance_url_masjid_id"`
	UserClassSessionAttendanceURLAttendanceID       uuid.UUID      `gorm:"type:uuid;not null;column:user_class_session_attendance_url_attendance_id"                               json:"user_class_session_attendance_url_attendance_id"`

	// optional FK ke type
	UserClassSessionAttendanceTypeID                *uuid.UUID     `gorm:"type:uuid;column:user_class_session_attendance_type_id"                                                json:"user_class_session_attendance_type_id,omitempty"`

	// data utama
	UserClassSessionAttendanceURLKind               string         `gorm:"type:varchar(24);not null;column:user_class_session_attendance_url_kind"                                json:"user_class_session_attendance_url_kind"`
	UserClassSessionAttendanceURLHref               *string        `gorm:"type:text;column:user_class_session_attendance_url_href"                                               json:"user_class_session_attendance_url_href,omitempty"`
	UserClassSessionAttendanceURLObjectKey          *string        `gorm:"type:text;column:user_class_session_attendance_url_object_key"                                         json:"user_class_session_attendance_url_object_key,omitempty"`
	UserClassSessionAttendanceURLObjectKeyOld       *string        `gorm:"type:text;column:user_class_session_attendance_url_object_key_old"                                     json:"user_class_session_attendance_url_object_key_old,omitempty"`

	// metadata tampilan
	UserClassSessionAttendanceURLLabel              *string        `gorm:"type:varchar(160);column:user_class_session_attendance_url_label"                                     json:"user_class_session_attendance_url_label,omitempty"`
	UserClassSessionAttendanceURLOrder              int            `gorm:"type:int;not null;default:0;column:user_class_session_attendance_url_order"                            json:"user_class_session_attendance_url_order"`
	UserClassSessionAttendanceURLIsPrimary          bool           `gorm:"not null;default:false;column:user_class_session_attendance_url_is_primary"                            json:"user_class_session_attendance_url_is_primary"`

	// housekeeping / retensi
	UserClassSessionAttendanceURLTrashURL           *string        `gorm:"type:text;column:user_class_session_attendance_url_trash_url"                                         json:"user_class_session_attendance_url_trash_url,omitempty"`
	UserClassSessionAttendanceURLDeletePendingUntil *time.Time     `gorm:"type:timestamptz;column:user_class_session_attendance_url_delete_pending_until"                        json:"user_class_session_attendance_url_delete_pending_until,omitempty"`

	// uploader (opsional)
	UserClassSessionAttendanceURLUploaderTeacherID  *uuid.UUID     `gorm:"type:uuid;column:user_class_session_attendance_url_uploader_teacher_id"                               json:"user_class_session_attendance_url_uploader_teacher_id,omitempty"`
	UserClassSessionAttendanceURLUploaderStudentID  *uuid.UUID     `gorm:"type:uuid;column:user_class_session_attendance_url_uploader_student_id"                               json:"user_class_session_attendance_url_uploader_student_id,omitempty"`

	// audit
	UserClassSessionAttendanceURLCreatedAt          time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_session_attendance_url_created_at"            json:"user_class_session_attendance_url_created_at"`
	UserClassSessionAttendanceURLUpdatedAt          time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_session_attendance_url_updated_at"            json:"user_class_session_attendance_url_updated_at"`
	UserClassSessionAttendanceURLDeletedAt          gorm.DeletedAt `gorm:"column:user_class_session_attendance_url_deleted_at;index"                                              json:"user_class_session_attendance_url_deleted_at,omitempty"`
}

func (UserClassSessionAttendanceURLModel) TableName() string {
	return "user_class_session_attendance_urls"
}
