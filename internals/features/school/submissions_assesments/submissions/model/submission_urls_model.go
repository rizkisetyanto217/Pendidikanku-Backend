// file: internals/features/assessments/submissions/model/submission_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubmissionURLModel struct {
	SubmissionURLID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:submission_url_id" json:"submission_url_id"`

	SubmissionURLSchoolID     uuid.UUID `gorm:"type:uuid;not null;column:submission_url_school_id" json:"submission_url_school_id"`
	SubmissionURLSubmissionID uuid.UUID `gorm:"type:uuid;not null;column:submission_url_submission_id" json:"submission_url_submission_id"`

	SubmissionURLKind string `gorm:"type:varchar(24);not null;column:submission_url_kind" json:"submission_url_kind"`

	SubmissionURLHref         *string `gorm:"type:text;column:submission_url_href" json:"submission_url_href,omitempty"`
	SubmissionURLObjectKey    *string `gorm:"type:text;column:submission_url_object_key" json:"submission_url_object_key,omitempty"`
	SubmissionURLObjectKeyOld *string `gorm:"type:text;column:submission_url_object_key_old" json:"submission_url_object_key_old,omitempty"`

	SubmissionURLLabel     *string `gorm:"type:varchar(160);column:submission_url_label" json:"submission_url_label,omitempty"`
	SubmissionURLOrder     int     `gorm:"not null;default:0;column:submission_url_order" json:"submission_url_order"`
	SubmissionURLIsPrimary bool    `gorm:"not null;default:false;column:submission_url_is_primary" json:"submission_url_is_primary"`

	SubmissionURLCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:submission_url_created_at" json:"submission_url_created_at"`
	SubmissionURLUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:submission_url_updated_at" json:"submission_url_updated_at"`
	SubmissionURLDeletedAt gorm.DeletedAt `gorm:"column:submission_url_deleted_at;index" json:"submission_url_deleted_at,omitempty"`

	SubmissionURLDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:submission_url_delete_pending_until" json:"submission_url_delete_pending_until,omitempty"`
}

func (SubmissionURLModel) TableName() string { return "submission_urls" }
