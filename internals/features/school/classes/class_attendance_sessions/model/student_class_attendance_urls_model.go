// file: internals/features/attendance/model/student_class_session_attendance_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StudentClassSessionAttendanceURLModel struct {
	StudentClassSessionAttendanceURLID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_class_session_attendance_url_id"                 json:"student_class_session_attendance_url_id"`
	StudentClassSessionAttendanceURLMasjidID     uuid.UUID `gorm:"type:uuid;not null;column:student_class_session_attendance_url_masjid_id"                                   json:"student_class_session_attendance_url_masjid_id"`
	StudentClassSessionAttendanceURLAttendanceID uuid.UUID `gorm:"type:uuid;not null;column:student_class_session_attendance_url_attendance_id"                               json:"student_class_session_attendance_url_attendance_id"`

	// optional FK ke type
	StudentClassSessionAttendanceTypeID *uuid.UUID `gorm:"type:uuid;column:student_class_session_attendance_type_id"                                                json:"student_class_session_attendance_type_id,omitempty"`

	// data utama
	StudentClassSessionAttendanceURLKind         string  `gorm:"type:varchar(24);not null;column:student_class_session_attendance_url_kind"                                json:"student_class_session_attendance_url_kind"`
	StudentClassSessionAttendanceURLHref         *string `gorm:"type:text;column:student_class_session_attendance_url_href"                                               json:"student_class_session_attendance_url_href,omitempty"`
	StudentClassSessionAttendanceURLObjectKey    *string `gorm:"type:text;column:student_class_session_attendance_url_object_key"                                         json:"student_class_session_attendance_url_object_key,omitempty"`
	StudentClassSessionAttendanceURLObjectKeyOld *string `gorm:"type:text;column:student_class_session_attendance_url_object_key_old"                                     json:"student_class_session_attendance_url_object_key_old,omitempty"`

	// metadata tampilan
	StudentClassSessionAttendanceURLLabel     *string `gorm:"type:varchar(160);column:student_class_session_attendance_url_label"                                     json:"student_class_session_attendance_url_label,omitempty"`
	StudentClassSessionAttendanceURLOrder     int     `gorm:"type:int;not null;default:0;column:student_class_session_attendance_url_order"                            json:"student_class_session_attendance_url_order"`
	StudentClassSessionAttendanceURLIsPrimary bool    `gorm:"not null;default:false;column:student_class_session_attendance_url_is_primary"                            json:"student_class_session_attendance_url_is_primary"`

	// housekeeping / retensi
	StudentClassSessionAttendanceURLTrashURL           *string    `gorm:"type:text;column:student_class_session_attendance_url_trash_url"                                         json:"student_class_session_attendance_url_trash_url,omitempty"`
	StudentClassSessionAttendanceURLDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:student_class_session_attendance_url_delete_pending_until"                        json:"student_class_session_attendance_url_delete_pending_until,omitempty"`

	// uploader (opsional)
	StudentClassSessionAttendanceURLUploaderTeacherID *uuid.UUID `gorm:"type:uuid;column:student_class_session_attendance_url_uploader_teacher_id"                               json:"student_class_session_attendance_url_uploader_teacher_id,omitempty"`
	StudentClassSessionAttendanceURLUploaderStudentID *uuid.UUID `gorm:"type:uuid;column:student_class_session_attendance_url_uploader_student_id"                               json:"student_class_session_attendance_url_uploader_student_id,omitempty"`

	// audit
	StudentClassSessionAttendanceURLCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_class_session_attendance_url_created_at"            json:"student_class_session_attendance_url_created_at"`
	StudentClassSessionAttendanceURLUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_class_session_attendance_url_updated_at"            json:"student_class_session_attendance_url_updated_at"`
	StudentClassSessionAttendanceURLDeletedAt gorm.DeletedAt `gorm:"column:student_class_session_attendance_url_deleted_at;index"                                              json:"student_class_session_attendance_url_deleted_at,omitempty"`
}

func (StudentClassSessionAttendanceURLModel) TableName() string {
	return "student_class_session_attendance_urls"
}
