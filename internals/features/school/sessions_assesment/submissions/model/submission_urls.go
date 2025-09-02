// file: internals/features/school/submissions/model/submission_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubmissionUrlsModel merepresentasikan lampiran URL pada sebuah submission
type SubmissionUrlsModel struct {
	SubmissionUrlsID                  uuid.UUID      `gorm:"column:submission_urls_id;type:uuid;default:gen_random_uuid();primaryKey" json:"submission_urls_id"`
	SubmissionUrlsSubmissionID        uuid.UUID      `gorm:"column:submission_urls_submission_id;type:uuid;not null;index" json:"submission_urls_submission_id"`
	SubmissionUrlsLabel               *string        `gorm:"column:submission_urls_label;type:varchar(120)" json:"submission_urls_label,omitempty"`
	SubmissionUrlsHref                string         `gorm:"column:submission_urls_href;type:text;not null" json:"submission_urls_href"`
	SubmissionUrlsTrashURL            *string        `gorm:"column:submission_urls_trash_url;type:text" json:"submission_urls_trash_url,omitempty"`
	SubmissionUrlsDeletePendingUntil  *time.Time     `gorm:"column:submission_urls_delete_pending_until" json:"submission_urls_delete_pending_until,omitempty"`
	SubmissionUrlsIsActive            bool           `gorm:"column:submission_urls_is_active;not null;default:true" json:"submission_urls_is_active"`
	SubmissionUrlsCreatedAt           time.Time      `gorm:"column:submission_urls_created_at;not null;default:now()" json:"submission_urls_created_at"`
	SubmissionUrlsUpdatedAt           time.Time      `gorm:"column:submission_urls_updated_at;not null;default:now()" json:"submission_urls_updated_at"`
	SubmissionUrlsDeletedAt           gorm.DeletedAt `gorm:"column:submission_urls_deleted_at;index" json:"submission_urls_deleted_at,omitempty"`

	// Relasi
}

// TableName override nama tabel
func (SubmissionUrlsModel) TableName() string {
	return "submission_urls"
}
