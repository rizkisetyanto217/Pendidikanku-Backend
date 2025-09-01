// file: internals/features/school/submissions/dto/submission_urls_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/attendance_assesment/submissions/model"
)

// ==== CREATE REQUEST ====
// Field wajib: SubmissionUrlsSubmissionID, SubmissionUrlsHref
type CreateSubmissionUrlRequest struct {
	SubmissionUrlsSubmissionID       uuid.UUID  `json:"submission_urls_submission_id" validate:"required"`
	SubmissionUrlsLabel              *string    `json:"submission_urls_label,omitempty"`
	SubmissionUrlsHref               string     `json:"submission_urls_href" validate:"required,url"`
	SubmissionUrlsTrashURL           *string    `json:"submission_urls_trash_url,omitempty"`
	SubmissionUrlsDeletePendingUntil *time.Time `json:"submission_urls_delete_pending_until,omitempty"`
	SubmissionUrlsIsActive           *bool      `json:"submission_urls_is_active,omitempty"`
}

// ==== UPDATE/PATCH REQUEST ====
// Patch-like, semua optional (pakai pointer).
type UpdateSubmissionUrlRequest struct {
	SubmissionUrlsLabel              *string    `json:"submission_urls_label,omitempty"`
	SubmissionUrlsHref               *string    `json:"submission_urls_href,omitempty" validate:"omitempty,url"`
	SubmissionUrlsTrashURL           *string    `json:"submission_urls_trash_url,omitempty"`
	SubmissionUrlsDeletePendingUntil *time.Time `json:"submission_urls_delete_pending_until,omitempty"`
	SubmissionUrlsIsActive           *bool      `json:"submission_urls_is_active,omitempty"`
}

// Alias biar eksplisit dipakai untuk endpoint PATCH
type PatchSubmissionUrlRequest = UpdateSubmissionUrlRequest

// ==== RESPONSE ====
// Dipakai untuk response API
type SubmissionUrlResponse struct {
	SubmissionUrlsID                 uuid.UUID  `json:"submission_urls_id"`
	SubmissionUrlsSubmissionID       uuid.UUID  `json:"submission_urls_submission_id"`
	SubmissionUrlsLabel              *string    `json:"submission_urls_label,omitempty"`
	SubmissionUrlsHref               string     `json:"submission_urls_href"`
	SubmissionUrlsTrashURL           *string    `json:"submission_urls_trash_url,omitempty"`
	SubmissionUrlsDeletePendingUntil *time.Time `json:"submission_urls_delete_pending_until,omitempty"`
	SubmissionUrlsIsActive           bool       `json:"submission_urls_is_active"`
	SubmissionUrlsCreatedAt          time.Time  `json:"submission_urls_created_at"`
	SubmissionUrlsUpdatedAt          time.Time  `json:"submission_urls_updated_at"`
	SubmissionUrlsDeletedAt          *time.Time `json:"submission_urls_deleted_at,omitempty"`
}

// ==== HELPERS ====

// ToSubmissionUrlResponse memetakan model → response DTO.
func ToSubmissionUrlResponse(m *model.SubmissionUrlsModel) SubmissionUrlResponse {
	var deletedAt *time.Time
	if m.SubmissionUrlsDeletedAt.Valid {
		t := m.SubmissionUrlsDeletedAt.Time
		deletedAt = &t
	}
	return SubmissionUrlResponse{
		SubmissionUrlsID:                 m.SubmissionUrlsID,
		SubmissionUrlsSubmissionID:       m.SubmissionUrlsSubmissionID,
		SubmissionUrlsLabel:              m.SubmissionUrlsLabel,
		SubmissionUrlsHref:               m.SubmissionUrlsHref,
		SubmissionUrlsTrashURL:           m.SubmissionUrlsTrashURL,
		SubmissionUrlsDeletePendingUntil: m.SubmissionUrlsDeletePendingUntil,
		SubmissionUrlsIsActive:           m.SubmissionUrlsIsActive,
		SubmissionUrlsCreatedAt:          m.SubmissionUrlsCreatedAt,
		SubmissionUrlsUpdatedAt:          m.SubmissionUrlsUpdatedAt,
		SubmissionUrlsDeletedAt:          deletedAt,
	}
}

// BuildSubmissionUrlUpdates menyusun map Updates untuk GORM dari payload PATCH.
// Hanya field non-nil yang akan dimasukkan ke map.
func BuildSubmissionUrlUpdates(req *UpdateSubmissionUrlRequest) map[string]interface{} {
	if req == nil {
		return nil
	}
	updates := make(map[string]interface{})

	if req.SubmissionUrlsLabel != nil {
		updates["submission_urls_label"] = req.SubmissionUrlsLabel
	}
	if req.SubmissionUrlsHref != nil {
		updates["submission_urls_href"] = *req.SubmissionUrlsHref
	}
	if req.SubmissionUrlsTrashURL != nil {
		updates["submission_urls_trash_url"] = req.SubmissionUrlsTrashURL
	}
	if req.SubmissionUrlsDeletePendingUntil != nil {
		updates["submission_urls_delete_pending_until"] = req.SubmissionUrlsDeletePendingUntil
	}
	if req.SubmissionUrlsIsActive != nil {
		updates["submission_urls_is_active"] = *req.SubmissionUrlsIsActive
	}

	return updates
}
