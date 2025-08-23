// file: internals/features/classes/openings/dto/class_term_opening_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

// ============================
// 📌 CREATE DTO
// ============================

type CreateClassTermOpeningRequest struct {
	ClassTermOpeningsMasjidID   uuid.UUID  `json:"class_term_openings_masjid_id"   validate:"required"`
	ClassTermOpeningsClassID    uuid.UUID  `json:"class_term_openings_class_id"    validate:"required"`
	ClassTermOpeningsTermID     uuid.UUID  `json:"class_term_openings_term_id"     validate:"required"`

	ClassTermOpeningsIsOpen     *bool      `json:"class_term_openings_is_open,omitempty"`

	ClassTermOpeningsRegistrationOpensAt   *time.Time `json:"class_term_openings_registration_opens_at,omitempty"`
	ClassTermOpeningsRegistrationClosesAt  *time.Time `json:"class_term_openings_registration_closes_at,omitempty"`

	ClassTermOpeningsQuotaTotal            *int    `json:"class_term_openings_quota_total,omitempty"`
	ClassTermOpeningsFeeOverrideMonthlyIDR *int    `json:"class_term_openings_fee_override_monthly_idr,omitempty"`

	ClassTermOpeningsNotes                 *string `json:"class_term_openings_notes,omitempty"`
}

// ============================
// 📌 UPDATE DTO
// ============================

type UpdateClassTermOpeningRequest struct {
	ClassTermOpeningsIsOpen     *bool      `json:"class_term_openings_is_open,omitempty"`

	ClassTermOpeningsRegistrationOpensAt   *time.Time `json:"class_term_openings_registration_opens_at,omitempty"`
	ClassTermOpeningsRegistrationClosesAt  *time.Time `json:"class_term_openings_registration_closes_at,omitempty"`

	ClassTermOpeningsQuotaTotal            *int    `json:"class_term_openings_quota_total,omitempty"`
	ClassTermOpeningsFeeOverrideMonthlyIDR *int    `json:"class_term_openings_fee_override_monthly_idr,omitempty"`

	ClassTermOpeningsNotes                 *string `json:"class_term_openings_notes,omitempty"`
}

// ============================
// 📌 RESPONSE DTO
// ============================

type ClassTermOpeningResponse struct {
	ClassTermOpeningsID         uuid.UUID  `json:"class_term_openings_id"`
	ClassTermOpeningsMasjidID   uuid.UUID  `json:"class_term_openings_masjid_id"`
	ClassTermOpeningsClassID    uuid.UUID  `json:"class_term_openings_class_id"`
	ClassTermOpeningsTermID     uuid.UUID  `json:"class_term_openings_term_id"`

	ClassTermOpeningsIsOpen     bool       `json:"class_term_openings_is_open"`

	ClassTermOpeningsRegistrationOpensAt   *time.Time `json:"class_term_openings_registration_opens_at,omitempty"`
	ClassTermOpeningsRegistrationClosesAt  *time.Time `json:"class_term_openings_registration_closes_at,omitempty"`

	ClassTermOpeningsQuotaTotal            *int       `json:"class_term_openings_quota_total,omitempty"`
	ClassTermOpeningsQuotaTaken            int        `json:"class_term_openings_quota_taken"`
	ClassTermOpeningsFeeOverrideMonthlyIDR *int       `json:"class_term_openings_fee_override_monthly_idr,omitempty"`

	ClassTermOpeningsNotes                 *string    `json:"class_term_openings_notes,omitempty"`

	ClassTermOpeningsCreatedAt             time.Time  `json:"class_term_openings_created_at"`
	ClassTermOpeningsUpdatedAt             *time.Time `json:"class_term_openings_updated_at,omitempty"`
	ClassTermOpeningsDeletedAt             *time.Time `json:"class_term_openings_deleted_at,omitempty"`
}



