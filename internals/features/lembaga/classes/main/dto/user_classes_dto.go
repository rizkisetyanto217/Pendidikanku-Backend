// internals/features/lembaga/classes/user_classes/main/dto/user_class_dto.go
package dto

import (
	"time"

	ucModel "masjidku_backend/internals/features/lembaga/classes/main/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

type CreateUserClassRequest struct {
	UserClassesUserID                  uuid.UUID  `json:"user_classes_user_id" validate:"required"`                   // FK -> users(id)
	UserClassesClassID                 uuid.UUID  `json:"user_classes_class_id" validate:"required"`                  // FK -> classes(class_id)
	UserClassesMasjidID                *uuid.UUID `json:"user_classes_masjid_id" validate:"omitempty"`                // (optional) isi dari token lebih aman

	UserClassesStatus                  *string    `json:"user_classes_status" validate:"omitempty,oneof=active inactive ended"`
	UserClassesStartedAt               *time.Time `json:"user_classes_started_at" validate:"omitempty"`
	UserClassesEndedAt                 *time.Time `json:"user_classes_ended_at" validate:"omitempty"`
	UserClassesFeeOverrideMonthlyIDR   *int       `json:"user_classes_fee_override_monthly_idr" validate:"omitempty,gte=0"`
	UserClassesNotes                   *string    `json:"user_classes_notes" validate:"omitempty"`
}

func (r *CreateUserClassRequest) ToModel() *ucModel.UserClassesModel {
	m := &ucModel.UserClassesModel{
		UserClassesUserID:                r.UserClassesUserID,
		UserClassesClassID:               r.UserClassesClassID,
		UserClassesMasjidID:              r.UserClassesMasjidID,
		UserClassesNotes:                 r.UserClassesNotes,
		UserClassesFeeOverrideMonthlyIDR: r.UserClassesFeeOverrideMonthlyIDR,
		UserClassesStatus:                ucModel.UserClassStatusActive, // default
	}

	// status (jika dikirim)
	if r.UserClassesStatus != nil && *r.UserClassesStatus != "" {
		m.UserClassesStatus = *r.UserClassesStatus
	}

	// started_at: pakai request kalau ada, else biarkan nil (biar default DB jalan) atau isi now
	if r.UserClassesStartedAt != nil {
		m.UserClassesStartedAt = r.UserClassesStartedAt
	} else {
		now := time.Now()
		m.UserClassesStartedAt = &now
	}

	// ended_at (opsional)
	if r.UserClassesEndedAt != nil {
		m.UserClassesEndedAt = r.UserClassesEndedAt
	}

	return m
}

type UpdateUserClassRequest struct {
	UserClassesUserID                  *uuid.UUID `json:"user_classes_user_id" validate:"omitempty"`
	UserClassesClassID                 *uuid.UUID `json:"user_classes_class_id" validate:"omitempty"`
	UserClassesMasjidID                *uuid.UUID `json:"user_classes_masjid_id" validate:"omitempty"`

	UserClassesStatus                  *string    `json:"user_classes_status" validate:"omitempty,oneof=active inactive ended"`
	UserClassesStartedAt               *time.Time `json:"user_classes_started_at" validate:"omitempty"`
	UserClassesEndedAt                 *time.Time `json:"user_classes_ended_at" validate:"omitempty"`
	UserClassesFeeOverrideMonthlyIDR   *int       `json:"user_classes_fee_override_monthly_idr" validate:"omitempty,gte=0"`
	UserClassesNotes                   *string    `json:"user_classes_notes" validate:"omitempty"`
}

func (r *UpdateUserClassRequest) ApplyToModel(m *ucModel.UserClassesModel) {
	if r.UserClassesUserID != nil {
		m.UserClassesUserID = *r.UserClassesUserID
	}
	if r.UserClassesClassID != nil {
		m.UserClassesClassID = *r.UserClassesClassID
	}
	if r.UserClassesMasjidID != nil {
		m.UserClassesMasjidID = r.UserClassesMasjidID
	}
	if r.UserClassesStatus != nil {
		m.UserClassesStatus = *r.UserClassesStatus
	}
	if r.UserClassesStartedAt != nil {
		m.UserClassesStartedAt = r.UserClassesStartedAt
	}
	if r.UserClassesEndedAt != nil {
		m.UserClassesEndedAt = r.UserClassesEndedAt
	}
	if r.UserClassesFeeOverrideMonthlyIDR != nil {
		m.UserClassesFeeOverrideMonthlyIDR = r.UserClassesFeeOverrideMonthlyIDR
	}
	if r.UserClassesNotes != nil {
		m.UserClassesNotes = r.UserClassesNotes
	}
	now := time.Now()
	m.UserClassesUpdatedAt = &now
}

/* ===================== QUERIES ===================== */

type ListUserClassQuery struct {
	UserID    *uuid.UUID `query:"user_id"`   // filter by user
	ClassID   *uuid.UUID `query:"class_id"`  // filter by class
	MasjidID  *uuid.UUID `query:"masjid_id"` // filter by masjid (tenant)
	Status    *string    `query:"status"`    // active|inactive|ended
	ActiveNow *bool      `query:"active_now"`

	Limit  int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	Sort   *string `query:"sort"`   // started_at_desc|started_at_asc|created_at_desc|created_at_asc
}

/* ===================== RESPONSES ===================== */

type UserClassResponse struct {
	UserClassesID                    uuid.UUID  `json:"user_classes_id"`
	UserClassesUserID                uuid.UUID  `json:"user_classes_user_id"`
	UserClassesClassID               uuid.UUID  `json:"user_classes_class_id"`
	UserClassesMasjidID              *uuid.UUID `json:"user_classes_masjid_id,omitempty"`

	UserClassesStatus                string      `json:"user_classes_status"`
	UserClassesStartedAt             *time.Time  `json:"user_classes_started_at,omitempty"`
	UserClassesEndedAt               *time.Time  `json:"user_classes_ended_at,omitempty"`

	UserClassesFeeOverrideMonthlyIDR *int        `json:"user_classes_fee_override_monthly_idr,omitempty"`
	UserClassesNotes                 *string     `json:"user_classes_notes,omitempty"`

	UserClassesCreatedAt             time.Time   `json:"user_classes_created_at"`
	UserClassesUpdatedAt             *time.Time  `json:"user_classes_updated_at,omitempty"`
}

func NewUserClassResponse(m *ucModel.UserClassesModel) *UserClassResponse {
	if m == nil {
		return nil
	}
	return &UserClassResponse{
		UserClassesID:                    m.UserClassesID,
		UserClassesUserID:                m.UserClassesUserID,
		UserClassesClassID:               m.UserClassesClassID,
		UserClassesMasjidID:              m.UserClassesMasjidID,

		UserClassesStatus:                m.UserClassesStatus,
		UserClassesStartedAt:             m.UserClassesStartedAt,
		UserClassesEndedAt:               m.UserClassesEndedAt,

		UserClassesFeeOverrideMonthlyIDR: m.UserClassesFeeOverrideMonthlyIDR,
		UserClassesNotes:                 m.UserClassesNotes,

		UserClassesCreatedAt:             m.UserClassesCreatedAt,
		UserClassesUpdatedAt:             m.UserClassesUpdatedAt,
	}
}
