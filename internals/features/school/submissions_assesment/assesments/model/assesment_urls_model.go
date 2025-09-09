// file: internals/features/assessment/urls/model/assessment_urls_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssessmentUrlsModel merepresentasikan tabel "assessment_urls"
type AssessmentUrlsModel struct {
	AssessmentUrlsID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid();column:assessment_urls_id"`
	AssessmentUrlsAssessmentID uuid.UUID      `gorm:"type:uuid;not null;column:assessment_urls_assessment_id;index"` // FK ke assessments.assessments_id
	AssessmentUrlsLabel        *string        `gorm:"type:varchar(120);column:assessment_urls_label"`
	AssessmentUrlsHref         string         `gorm:"type:text;not null;column:assessment_urls_href"`

	// Trash / delete pending
	AssessmentUrlsTrashURL        *string    `gorm:"type:text;column:assessment_urls_trash_url"`
	AssessmentUrlsDeletePendingAt *time.Time `gorm:"column:assessment_urls_delete_pending_until"`

	// Publish flags
	AssessmentUrlsIsPublished bool       `gorm:"not null;default:false;column:assessment_urls_is_published"`
	AssessmentUrlsIsActive    bool       `gorm:"not null;default:true;column:assessment_urls_is_active"`
	AssessmentUrlsPublishedAt *time.Time `gorm:"column:assessment_urls_published_at"`
	AssessmentUrlsExpiresAt   *time.Time `gorm:"column:assessment_urls_expires_at"`

	// Public accessors
	AssessmentUrlsPublicSlug  *string `gorm:"type:varchar(64);column:assessment_urls_public_slug"`
	AssessmentUrlsPublicToken *string `gorm:"type:varchar(64);column:assessment_urls_public_token"`

	// Timestamps & soft delete
	AssessmentUrlsCreatedAt time.Time      `gorm:"column:assessment_urls_created_at;autoCreateTime"`
	AssessmentUrlsUpdatedAt time.Time      `gorm:"column:assessment_urls_updated_at;autoUpdateTime"`
	AssessmentUrlsDeletedAt gorm.DeletedAt `gorm:"column:assessment_urls_deleted_at;index"`

	// Relasi opsional ke assessments
	Assessment *AssessmentModel `gorm:"foreignKey:AssessmentUrlsAssessmentID;references:AssessmentsID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (AssessmentUrlsModel) TableName() string {
	return "assessment_urls"
}
