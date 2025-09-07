// file: internals/features/lembaga/classes/user_classes/main/dto/user_class_dto.go
package dto

import (
	"time"

	ucmodel "masjidku_backend/internals/features/school/classes/classes/model"

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

	// FK -> academic_terms(academic_terms_id)
	UserClassesTermID uuid.UUID `json:"user_classes_term_id" validate:"required"`

	// (Opsional) relasi ke masjid_students
	UserClassesMasjidStudentID *uuid.UUID `json:"user_classes_masjid_student_id" validate:"omitempty"`

	// Status dibatasi oleh CHECK ('active','inactive','ended'); default 'active'
	UserClassesStatus *string `json:"user_classes_status" validate:"omitempty,oneof=active inactive ended"`

	// Jejak waktu enrolment
	UserClassesJoinedAt *time.Time `json:"user_classes_joined_at" validate:"omitempty"`
	UserClassesLeftAt   *time.Time `json:"user_classes_left_at" validate:"omitempty"`
}

func (r *CreateUserClassRequest) ToModel(masjidIDFromCtx *uuid.UUID) *ucmodel.UserClassesModel {
	// Tentukan masjid_id final (payload > context)
	var masjidID uuid.UUID
	if r.UserClassesMasjidID != nil {
		masjidID = *r.UserClassesMasjidID
	} else if masjidIDFromCtx != nil {
		masjidID = *masjidIDFromCtx
	}

	m := &ucmodel.UserClassesModel{
		UserClassesUserID:           r.UserClassesUserID,
		UserClassesClassID:          r.UserClassesClassID,
		UserClassesMasjidID:         masjidID,
		UserClassesTermID:           r.UserClassesTermID,
		UserClassesMasjidStudentID:  r.UserClassesMasjidStudentID,
		UserClassesStatus:           ucmodel.UserClassStatusActive, // default
		UserClassesJoinedAt:         r.UserClassesJoinedAt,
		UserClassesLeftAt:           r.UserClassesLeftAt,
	}

	if r.UserClassesStatus != nil && *r.UserClassesStatus != "" {
		m.UserClassesStatus = *r.UserClassesStatus
	}

	return m
}

type UpdateUserClassRequest struct {
	UserClassesUserID           *uuid.UUID `json:"user_classes_user_id" validate:"omitempty"`
	UserClassesClassID          *uuid.UUID `json:"user_classes_class_id" validate:"omitempty"`

	// Boleh diubah jika skenario pindah tenant dibuka (hati-hati dgn FK komposit)
	UserClassesMasjidID         *uuid.UUID `json:"user_classes_masjid_id" validate:"omitempty"`

	// Term
	UserClassesTermID           *uuid.UUID `json:"user_classes_term_id" validate:"omitempty"`

	// Masjid student (opsional)
	UserClassesMasjidStudentID  *uuid.UUID `json:"user_classes_masjid_student_id" validate:"omitempty"`

	UserClassesStatus           *string    `json:"user_classes_status" validate:"omitempty,oneof=active inactive ended"`

	// Jejak waktu enrolment
	UserClassesJoinedAt         *time.Time `json:"user_classes_joined_at" validate:"omitempty"`
	UserClassesLeftAt           *time.Time `json:"user_classes_left_at" validate:"omitempty"`
}

func (r *UpdateUserClassRequest) ApplyToModel(m *ucmodel.UserClassesModel) {
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
	if r.UserClassesMasjidStudentID != nil {
		m.UserClassesMasjidStudentID = r.UserClassesMasjidStudentID
	}
	if r.UserClassesStatus != nil {
		m.UserClassesStatus = *r.UserClassesStatus
	}
	if r.UserClassesJoinedAt != nil {
		m.UserClassesJoinedAt = r.UserClassesJoinedAt
	}
	if r.UserClassesLeftAt != nil {
		m.UserClassesLeftAt = r.UserClassesLeftAt
	}

	// Model pakai non-pointer UpdatedAt; isi manual agar segera berubah.
	m.UserClassesUpdatedAt = time.Now()
}

/* ===================== QUERIES ===================== */

type ListUserClassQuery struct {
	UserID            *uuid.UUID `query:"user_id"`              // filter by user
	ClassID           *uuid.UUID `query:"class_id"`             // filter by class
	MasjidID          *uuid.UUID `query:"masjid_id"`            // tenant
	TermID            *uuid.UUID `query:"term_id"`              // filter by term
	MasjidStudentID   *uuid.UUID `query:"masjid_student_id"`    // filter by masjid_students
	Status            *string    `query:"status"`               // active|inactive|ended
	ActiveNow         *bool      `query:"active_now"`           // bantu flag business logic (status='active' && left_at IS NULL), opsional

	// Filter rentang joined
	JoinedFrom        *time.Time `query:"joined_from"`
	JoinedTo          *time.Time `query:"joined_to"`

	Limit  int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`

	// created_at_desc|created_at_asc|joined_at_desc|joined_at_asc
	Sort *string `query:"sort"`
}

/* ===================== RESPONSES ===================== */

type UserClassResponse struct {
	UserClassesID           uuid.UUID  `json:"user_classes_id"`
	UserClassesUserID       uuid.UUID  `json:"user_classes_user_id"`
	UserClassesClassID      uuid.UUID  `json:"user_classes_class_id"`
	UserClassesMasjidID     uuid.UUID  `json:"user_classes_masjid_id"`

	UserClassesTermID       uuid.UUID  `json:"user_classes_term_id"`
	UserClassesMasjidStudentID *uuid.UUID `json:"user_classes_masjid_student_id,omitempty"`

	UserClassesStatus       string     `json:"user_classes_status"`

	UserClassesJoinedAt     *time.Time `json:"user_classes_joined_at,omitempty"`
	UserClassesLeftAt       *time.Time `json:"user_classes_left_at,omitempty"`

	UserClassesCreatedAt    time.Time  `json:"user_classes_created_at"`
	UserClassesUpdatedAt    time.Time  `json:"user_classes_updated_at"`
}

func NewUserClassResponse(m *ucmodel.UserClassesModel) *UserClassResponse {
	if m == nil {
		return nil
	}
	return &UserClassResponse{
		UserClassesID:            m.UserClassesID,
		UserClassesUserID:        m.UserClassesUserID,
		UserClassesClassID:       m.UserClassesClassID,
		UserClassesMasjidID:      m.UserClassesMasjidID,

		UserClassesTermID:        m.UserClassesTermID,
		UserClassesMasjidStudentID: m.UserClassesMasjidStudentID,

		UserClassesStatus:        m.UserClassesStatus,

		UserClassesJoinedAt:      m.UserClassesJoinedAt,
		UserClassesLeftAt:        m.UserClassesLeftAt,

		UserClassesCreatedAt:     m.UserClassesCreatedAt,
		UserClassesUpdatedAt:     m.UserClassesUpdatedAt,
	}
}
