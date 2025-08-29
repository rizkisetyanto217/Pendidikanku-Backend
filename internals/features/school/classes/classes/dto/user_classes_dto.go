// file: internals/features/lembaga/classes/user_classes/main/dto/user_class_dto.go
package dto

import (
	"time"

	ucModel "masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

type CreateUserClassRequest struct {
	// FK -> users(id)
	UserClassesUserID uuid.UUID `json:"user_classes_user_id" validate:"required"`

	// FK -> classes(class_id)
	UserClassesClassID uuid.UUID `json:"user_classes_class_id" validate:"required"`

	// Tenant. Di handler bisa diisi dari token; tetap optional di payload.
	UserClassesMasjidID *uuid.UUID `json:"user_classes_masjid_id" validate:"omitempty"`

	// FK -> academic_terms(academic_terms_id); di DB juga divalidasi komposit dg masjid_id
	UserClassesTermID uuid.UUID `json:"user_classes_term_id" validate:"required"`

	// (Opsional) jejak opening
	UserClassesOpeningID *uuid.UUID `json:"user_classes_opening_id" validate:"omitempty"`

	// Status dibatasi oleh CHECK ('active','inactive','ended')
	UserClassesStatus *string `json:"user_classes_status" validate:"omitempty,oneof=active inactive ended"`

	// Snapshot biaya per siswa
	UserClassesFeeOverrideMonthlyIDR *int    `json:"user_classes_fee_override_monthly_idr" validate:"omitempty,gte=0"`
	UserClassesNotes                 *string `json:"user_classes_notes" validate:"omitempty"`
}

func (r *CreateUserClassRequest) ToModel(masjidIDFromCtx *uuid.UUID) *ucModel.UserClassesModel {
	// Tentukan masjid_id final (payload > context)
	var masjidID uuid.UUID
	if r.UserClassesMasjidID != nil {
		masjidID = *r.UserClassesMasjidID
	} else if masjidIDFromCtx != nil {
		masjidID = *masjidIDFromCtx
	} else {
		// Biarkan zero-value; sebaiknya handler validasi ini sebelum save
	}

	m := &ucModel.UserClassesModel{
		UserClassesUserID:                r.UserClassesUserID,
		UserClassesClassID:               r.UserClassesClassID,
		UserClassesMasjidID:              masjidID,
		UserClassesTermID:                r.UserClassesTermID,
		UserClassesOpeningID:             r.UserClassesOpeningID,
		UserClassesNotes:                 r.UserClassesNotes,
		UserClassesFeeOverrideMonthlyIDR: r.UserClassesFeeOverrideMonthlyIDR,
		UserClassesStatus:                ucModel.UserClassStatusActive, // default
	}

	if r.UserClassesStatus != nil && *r.UserClassesStatus != "" {
		m.UserClassesStatus = *r.UserClassesStatus
	}

	return m
}

type UpdateUserClassRequest struct {
	UserClassesUserID                *uuid.UUID `json:"user_classes_user_id" validate:"omitempty"`
	UserClassesClassID               *uuid.UUID `json:"user_classes_class_id" validate:"omitempty"`

	// Boleh diubah jika skenario pindah tenant dibuka, tapi hati‑hati dengan FK komposit lain.
	UserClassesMasjidID              *uuid.UUID `json:"user_classes_masjid_id" validate:"omitempty"`

	// Term & Opening
	UserClassesTermID                *uuid.UUID `json:"user_classes_term_id" validate:"omitempty"`
	UserClassesOpeningID             *uuid.UUID `json:"user_classes_opening_id" validate:"omitempty"`

	UserClassesStatus                *string    `json:"user_classes_status" validate:"omitempty,oneof=active inactive ended"`
	UserClassesFeeOverrideMonthlyIDR *int       `json:"user_classes_fee_override_monthly_idr" validate:"omitempty,gte=0"`
	UserClassesNotes                 *string    `json:"user_classes_notes" validate:"omitempty"`
}

func (r *UpdateUserClassRequest) ApplyToModel(m *ucModel.UserClassesModel) {
	if r.UserClassesUserID != nil {
		m.UserClassesUserID = *r.UserClassesUserID
	}
	if r.UserClassesClassID != nil {
		m.UserClassesClassID = *r.UserClassesClassID
	}
	if r.UserClassesMasjidID != nil {
		m.UserClassesMasjidID = *r.UserClassesMasjidID
	}
	if r.UserClassesTermID != nil {
		m.UserClassesTermID = *r.UserClassesTermID
	}
	if r.UserClassesOpeningID != nil {
		m.UserClassesOpeningID = r.UserClassesOpeningID
	}
	if r.UserClassesStatus != nil {
		m.UserClassesStatus = *r.UserClassesStatus
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
	UserID     *uuid.UUID `query:"user_id"`    // filter by user
	ClassID    *uuid.UUID `query:"class_id"`   // filter by class
	MasjidID   *uuid.UUID `query:"masjid_id"`  // tenant
	TermID     *uuid.UUID `query:"term_id"`    // filter by term
	OpeningID  *uuid.UUID `query:"opening_id"` // filter by opening (opsional)
	Status     *string    `query:"status"`     // active|inactive|ended
	ActiveNow  *bool      `query:"active_now"`

	Limit  int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`

	// Disederhanakan: created_at_desc|created_at_asc (karena tidak ada started_at di tabel)
	Sort *string `query:"sort"` // created_at_desc|created_at_asc
}

/* ===================== RESPONSES ===================== */

type UserClassResponse struct {
	UserClassesID                    uuid.UUID  `json:"user_classes_id"`
	UserClassesUserID                uuid.UUID  `json:"user_classes_user_id"`
	UserClassesClassID               uuid.UUID  `json:"user_classes_class_id"`
	UserClassesMasjidID              uuid.UUID  `json:"user_classes_masjid_id"`

	UserClassesTermID                uuid.UUID  `json:"user_classes_term_id"`
	UserClassesOpeningID             *uuid.UUID `json:"user_classes_opening_id,omitempty"`

	UserClassesStatus                string     `json:"user_classes_status"`

	UserClassesFeeOverrideMonthlyIDR *int       `json:"user_classes_fee_override_monthly_idr,omitempty"`
	UserClassesNotes                 *string    `json:"user_classes_notes,omitempty"`

	UserClassesCreatedAt             time.Time  `json:"user_classes_created_at"`
	UserClassesUpdatedAt             *time.Time `json:"user_classes_updated_at,omitempty"`
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

		UserClassesTermID:                m.UserClassesTermID,
		UserClassesOpeningID:             m.UserClassesOpeningID,

		UserClassesStatus:                m.UserClassesStatus,
		UserClassesFeeOverrideMonthlyIDR: m.UserClassesFeeOverrideMonthlyIDR,
		UserClassesNotes:                 m.UserClassesNotes,

		UserClassesCreatedAt:             m.UserClassesCreatedAt,
		UserClassesUpdatedAt:             m.UserClassesUpdatedAt,
	}
}
