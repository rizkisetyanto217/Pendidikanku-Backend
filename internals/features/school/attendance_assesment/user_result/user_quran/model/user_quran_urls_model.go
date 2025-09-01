// internals/features/school/attendance_assesment/user_result/user_quran_url/model/user_quran_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserQuranURLModel struct {
	// PK
	UserQuranURLsID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_quran_urls_id" json:"user_quran_urls_id"`

	// FK ke user_quran_records(user_quran_records_id)
	UserQuranURLsRecordID uuid.UUID `gorm:"type:uuid;not null;column:user_quran_urls_record_id;index:idx_user_quran_urls_record" json:"user_quran_urls_record_id"`

	// Metadata
	UserQuranURLsLabel *string `gorm:"type:varchar(120);column:user_quran_urls_label" json:"user_quran_urls_label,omitempty"`

	// URL utama (NOT NULL)
	UserQuranURLsHref string `gorm:"type:text;not null;column:user_quran_urls_href" json:"user_quran_urls_href"`

	// Housekeeping (opsional)
	UserQuranURLsTrashURL           *string    `gorm:"type:text;column:user_quran_urls_trash_url" json:"user_quran_urls_trash_url,omitempty"`
	UserQuranURLsDeletePendingUntil *time.Time `gorm:"column:user_quran_urls_delete_pending_until" json:"user_quran_urls_delete_pending_until,omitempty"`

	// Uploader refs (salah satu boleh NULL)
	UserQuranURLsUploaderTeacherID *uuid.UUID `gorm:"type:uuid;column:user_quran_urls_uploader_teacher_id;index:idx_uqri_uploader_teacher" json:"user_quran_urls_uploader_teacher_id,omitempty"`
	UserQuranURLsUploaderUserID    *uuid.UUID `gorm:"type:uuid;column:user_quran_urls_uploader_user_id;index:idx_uqri_uploader_user" json:"user_quran_urls_uploader_user_id,omitempty"`

	// Timestamps
	UserQuranURLsCreatedAt time.Time      `gorm:"column:user_quran_urls_created_at;autoCreateTime;index:idx_uqri_created_at,sort:desc" json:"user_quran_urls_created_at"`
	UserQuranURLsUpdatedAt time.Time      `gorm:"column:user_quran_urls_updated_at;autoUpdateTime" json:"user_quran_urls_updated_at"`
	UserQuranURLsDeletedAt gorm.DeletedAt `gorm:"column:user_quran_urls_deleted_at;index" json:"user_quran_urls_deleted_at,omitempty"`
}

func (UserQuranURLModel) TableName() string {
	return "user_quran_urls"
}
