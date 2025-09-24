// file: internals/features/lembaga/classes/user_class_sections/main/dto/user_class_section_dto.go
package dto

import (
	"time"

	enrolModel "masjidku_backend/internals/features/school/classes/class_sections/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

type CreateUserClassSectionRequest struct {
	UserClassSectionUserClassID  uuid.UUID  `json:"user_class_section_user_class_id" validate:"required"`
	UserClassSectionSectionID    uuid.UUID  `json:"user_class_section_section_id"    validate:"required"`
	UserClassSectionMasjidID     *uuid.UUID `json:"user_class_section_masjid_id"     validate:"omitempty"`
	UserClassSectionAssignedAt   *time.Time `json:"user_class_section_assigned_at"   validate:"omitempty"` // nil ⇒ pakai DEFAULT CURRENT_DATE (DB)
	UserClassSectionUnassignedAt *time.Time `json:"user_class_section_unassigned_at" validate:"omitempty"`
}

func (r *CreateUserClassSectionRequest) ToModel() *enrolModel.UserClassSection {
	m := &enrolModel.UserClassSection{
		UserClassSectionUserClassID:  r.UserClassSectionUserClassID,
		UserClassSectionSectionID:    r.UserClassSectionSectionID,
		UserClassSectionMasjidID:     uuid.Nil,
		UserClassSectionUnassignedAt: r.UserClassSectionUnassignedAt,
	}
	// AssignedAt di model adalah time.Time (non-pointer). Jika request berikan nilai, set; kalau tidak, biarkan zero value agar DB default jalan.
	if r.UserClassSectionAssignedAt != nil {
		m.UserClassSectionAssignedAt = *r.UserClassSectionAssignedAt
	}
	if r.UserClassSectionMasjidID != nil {
		m.UserClassSectionMasjidID = *r.UserClassSectionMasjidID
	}
	return m
}

type UpdateUserClassSectionRequest struct {
	UserClassSectionUserClassID  *uuid.UUID `json:"user_class_section_user_class_id"  validate:"omitempty"`
	UserClassSectionSectionID    *uuid.UUID `json:"user_class_section_section_id"     validate:"omitempty"`
	UserClassSectionMasjidID     *uuid.UUID `json:"user_class_section_masjid_id"      validate:"omitempty"`
	UserClassSectionAssignedAt   *time.Time `json:"user_class_section_assigned_at"    validate:"omitempty"`
	UserClassSectionUnassignedAt *time.Time `json:"user_class_section_unassigned_at"  validate:"omitempty"`
}

func (r *UpdateUserClassSectionRequest) ApplyToModel(m *enrolModel.UserClassSection) {
	if r.UserClassSectionUserClassID != nil {
		m.UserClassSectionUserClassID = *r.UserClassSectionUserClassID
	}
	if r.UserClassSectionSectionID != nil {
		m.UserClassSectionSectionID = *r.UserClassSectionSectionID
	}
	if r.UserClassSectionMasjidID != nil {
		m.UserClassSectionMasjidID = *r.UserClassSectionMasjidID
	}
	if r.UserClassSectionAssignedAt != nil {
		m.UserClassSectionAssignedAt = *r.UserClassSectionAssignedAt
	}
	if r.UserClassSectionUnassignedAt != nil {
		m.UserClassSectionUnassignedAt = r.UserClassSectionUnassignedAt
	}
	m.UserClassSectionUpdatedAt = time.Now()
}

/* ===================== QUERIES ===================== */

type ListUserClassSectionQuery struct {
	UserClassID *uuid.UUID `query:"user_class_id"`
	SectionID   *uuid.UUID `query:"section_id"`
	MasjidID    *uuid.UUID `query:"masjid_id"`
	ActiveOnly  *bool      `query:"active_only"` // true ⇒ unassigned_at IS NULL

	Limit  int     `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	Sort   *string `query:"sort"` // assigned_at_desc|assigned_at_asc|created_at_desc|created_at_asc
}

/* ===================== RESPONSES ===================== */

type UserClassSectionResponse struct {
	UserClassSectionID           uuid.UUID  `json:"user_class_section_id"`
	UserClassSectionUserClassID  uuid.UUID  `json:"user_class_section_user_class_id"`
	UserClassSectionSectionID    uuid.UUID  `json:"user_class_section_section_id"`
	UserClassSectionMasjidID     uuid.UUID  `json:"user_class_section_masjid_id"`
	UserClassSectionAssignedAt   *time.Time `json:"user_class_section_assigned_at,omitempty"` // pointer agar kompatibel dengan respon lama
	UserClassSectionUnassignedAt *time.Time `json:"user_class_section_unassigned_at,omitempty"`

	// Tambahan dari user_classes (opsional)
	UserClassesStatus string `json:"user_classes_status,omitempty"`

	UserClassSectionCreatedAt time.Time  `json:"user_class_section_created_at"`
	UserClassSectionUpdatedAt time.Time  `json:"user_class_section_updated_at"`
	UserClassSectionDeletedAt *time.Time `json:"user_class_section_deleted_at,omitempty"`

	User    *UcsUser        `json:"user,omitempty"`
	Profile *UcsUserProfile `json:"profile,omitempty"`
}

func NewUserClassSectionResponse(m *enrolModel.UserClassSection) *UserClassSectionResponse {
	if m == nil {
		return nil
	}
	resp := &UserClassSectionResponse{
		UserClassSectionID:           m.UserClassSectionID,
		UserClassSectionUserClassID:  m.UserClassSectionUserClassID,
		UserClassSectionSectionID:    m.UserClassSectionSectionID,
		UserClassSectionMasjidID:     m.UserClassSectionMasjidID,
		UserClassSectionAssignedAt:   func() *time.Time { t := m.UserClassSectionAssignedAt; return &t }(),
		UserClassSectionUnassignedAt: m.UserClassSectionUnassignedAt,
		UserClassSectionCreatedAt:    m.UserClassSectionCreatedAt,
		UserClassSectionUpdatedAt:    m.UserClassSectionUpdatedAt,
	}
	if m.UserClassSectionDeletedAt.Valid {
		t := m.UserClassSectionDeletedAt.Time
		resp.UserClassSectionDeletedAt = &t
	}
	return resp
}

func (r *UserClassSectionResponse) WithUser(u *UcsUser, p *UcsUserProfile) *UserClassSectionResponse {
	r.User, r.Profile = u, p
	return r
}

/* ========== Tipe ringkas untuk enrichment ========== */

type UcsUser struct {
	ID       uuid.UUID `json:"id"`
	UserName string    `json:"user_name"`
	FullName *string   `json:"full_name,omitempty"`
	Email    string    `json:"email"`
	IsActive bool      `json:"is_active"`
}

type UcsUserProfile struct {
	UserID                  uuid.UUID  `json:"user_id"`
	DonationName            *string    `json:"donation_name,omitempty"`
	PhotoURL                *string    `json:"photo_url,omitempty"`
	PhotoTrashURL           *string    `json:"photo_trash_url,omitempty"`
	PhotoDeletePendingUntil *time.Time `json:"photo_delete_pending_until,omitempty"`
	DateOfBirth             *time.Time `json:"date_of_birth,omitempty"`
	Gender                  *string    `json:"gender,omitempty"` // "male" | "female"
	PhoneNumber             *string    `json:"phone_number,omitempty"`
	Bio                     *string    `json:"bio,omitempty"`
	Location                *string    `json:"location,omitempty"`
	Occupation              *string    `json:"occupation,omitempty"`
}
