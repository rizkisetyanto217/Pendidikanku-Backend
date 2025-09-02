// file: internals/features/assessment/urls/dto/assessment_urls_dto.go
package dto

import (
	"masjidku_backend/internals/features/school/sessions_assesment/assesments/model"
	"time"

	"github.com/google/uuid"
)

// ==== CREATE REQUEST ====
// field wajib: AssessmentUrlsAssessmentID, AssessmentUrlsHref
type CreateAssessmentUrlsRequest struct {
	AssessmentUrlsAssessmentID uuid.UUID `json:"assessment_urls_assessment_id" validate:"required"`
	AssessmentUrlsLabel        *string   `json:"assessment_urls_label"`
	AssessmentUrlsHref         string    `json:"assessment_urls_href" validate:"required,url"`

	AssessmentUrlsTrashURL        *string    `json:"assessment_urls_trash_url"`
	AssessmentUrlsDeletePendingAt *time.Time `json:"assessment_urls_delete_pending_until"`

	AssessmentUrlsIsPublished bool       `json:"assessment_urls_is_published"`
	AssessmentUrlsIsActive    bool       `json:"assessment_urls_is_active"`
	AssessmentUrlsPublishedAt *time.Time `json:"assessment_urls_published_at"`
	AssessmentUrlsExpiresAt   *time.Time `json:"assessment_urls_expires_at"`
	AssessmentUrlsPublicSlug  *string    `json:"assessment_urls_public_slug"`
	AssessmentUrlsPublicToken *string    `json:"assessment_urls_public_token"`
}

// ==== UPDATE REQUEST ====
// patch-like, semua optional
type UpdateAssessmentUrlsRequest struct {
	AssessmentUrlsLabel *string `json:"assessment_urls_label"`
	AssessmentUrlsHref  *string `json:"assessment_urls_href" validate:"omitempty,url"`

	AssessmentUrlsTrashURL        *string    `json:"assessment_urls_trash_url"`
	AssessmentUrlsDeletePendingAt *time.Time `json:"assessment_urls_delete_pending_until"`

	AssessmentUrlsIsPublished *bool      `json:"assessment_urls_is_published"`
	AssessmentUrlsIsActive    *bool      `json:"assessment_urls_is_active"`
	AssessmentUrlsPublishedAt *time.Time `json:"assessment_urls_published_at"`
	AssessmentUrlsExpiresAt   *time.Time `json:"assessment_urls_expires_at"`
	AssessmentUrlsPublicSlug  *string    `json:"assessment_urls_public_slug"`
	AssessmentUrlsPublicToken *string    `json:"assessment_urls_public_token"`
}

// ==== RESPONSE ====
type AssessmentUrlsResponse struct {
	AssessmentUrlsID           uuid.UUID  `json:"assessment_urls_id"`
	AssessmentUrlsAssessmentID uuid.UUID  `json:"assessment_urls_assessment_id"`
	AssessmentUrlsLabel        *string    `json:"assessment_urls_label"`
	AssessmentUrlsHref         string     `json:"assessment_urls_href"`

	AssessmentUrlsTrashURL        *string    `json:"assessment_urls_trash_url,omitempty"`
	AssessmentUrlsDeletePendingAt *time.Time `json:"assessment_urls_delete_pending_until,omitempty"`

	AssessmentUrlsIsPublished bool       `json:"assessment_urls_is_published"`
	AssessmentUrlsIsActive    bool       `json:"assessment_urls_is_active"`
	AssessmentUrlsPublishedAt *time.Time `json:"assessment_urls_published_at,omitempty"`
	AssessmentUrlsExpiresAt   *time.Time `json:"assessment_urls_expires_at,omitempty"`
	AssessmentUrlsPublicSlug  *string    `json:"assessment_urls_public_slug,omitempty"`
	AssessmentUrlsPublicToken *string    `json:"assessment_urls_public_token,omitempty"`

	AssessmentUrlsCreatedAt time.Time  `json:"assessment_urls_created_at"`
	AssessmentUrlsUpdatedAt time.Time  `json:"assessment_urls_updated_at"`
	AssessmentUrlsDeletedAt *time.Time `json:"assessment_urls_deleted_at,omitempty"`
}



// ToAssessmentUrlsResponse memetakan model → response DTO.
func ToAssessmentUrlsResponse(m *model.AssessmentUrlsModel) AssessmentUrlsResponse {
	var deletedAt *time.Time
	if m.AssessmentUrlsDeletedAt.Valid {
		t := m.AssessmentUrlsDeletedAt.Time
		deletedAt = &t
	}
	return AssessmentUrlsResponse{
		AssessmentUrlsID:              m.AssessmentUrlsID,
		AssessmentUrlsAssessmentID:    m.AssessmentUrlsAssessmentID,
		AssessmentUrlsLabel:           m.AssessmentUrlsLabel,
		AssessmentUrlsHref:            m.AssessmentUrlsHref,
		AssessmentUrlsTrashURL:        m.AssessmentUrlsTrashURL,
		AssessmentUrlsDeletePendingAt: m.AssessmentUrlsDeletePendingAt,
		AssessmentUrlsIsPublished:     m.AssessmentUrlsIsPublished,
		AssessmentUrlsIsActive:        m.AssessmentUrlsIsActive,
		AssessmentUrlsPublishedAt:     m.AssessmentUrlsPublishedAt,
		AssessmentUrlsExpiresAt:       m.AssessmentUrlsExpiresAt,
		AssessmentUrlsPublicSlug:      m.AssessmentUrlsPublicSlug,
		AssessmentUrlsPublicToken:     m.AssessmentUrlsPublicToken,
		AssessmentUrlsCreatedAt:       m.AssessmentUrlsCreatedAt,
		AssessmentUrlsUpdatedAt:       m.AssessmentUrlsUpdatedAt,
		AssessmentUrlsDeletedAt:       deletedAt,
	}
}
