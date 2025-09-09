// file: internals/features/lembaga/classes/user_class_sections/main/dto/user_class_section_dto.go
package dto

import (
	"time"

	ucsModel "masjidku_backend/internals/features/school/classes/class_sections/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

type CreateUserClassSectionRequest struct {
	UserClassSectionsUserClassID  uuid.UUID  `json:"user_class_sections_user_class_id" validate:"required"`
	UserClassSectionsSectionID    uuid.UUID  `json:"user_class_sections_section_id" validate:"required"`
	UserClassSectionsMasjidID     *uuid.UUID `json:"user_class_sections_masjid_id" validate:"omitempty"`
	UserClassSectionsAssignedAt   *time.Time `json:"user_class_sections_assigned_at" validate:"omitempty"`   // nil => DEFAULT CURRENT_DATE (DB)
	UserClassSectionsUnassignedAt *time.Time `json:"user_class_sections_unassigned_at" validate:"omitempty"`
}

func (r *CreateUserClassSectionRequest) ToModel() *ucsModel.UserClassSectionsModel {
	m := &ucsModel.UserClassSectionsModel{
		UserClassSectionsUserClassID:  r.UserClassSectionsUserClassID,
		UserClassSectionsSectionID:    r.UserClassSectionsSectionID,
		UserClassSectionsMasjidID:     uuid.Nil,
		UserClassSectionsAssignedAt:   r.UserClassSectionsAssignedAt,
		UserClassSectionsUnassignedAt: r.UserClassSectionsUnassignedAt,
	}
	if r.UserClassSectionsMasjidID != nil {
		m.UserClassSectionsMasjidID = *r.UserClassSectionsMasjidID
	}
	return m
}

type UpdateUserClassSectionRequest struct {
	UserClassSectionsUserClassID  *uuid.UUID `json:"user_class_sections_user_class_id" validate:"omitempty"`
	UserClassSectionsSectionID    *uuid.UUID `json:"user_class_sections_section_id" validate:"omitempty"`
	UserClassSectionsMasjidID     *uuid.UUID `json:"user_class_sections_masjid_id" validate:"omitempty"`
	UserClassSectionsAssignedAt   *time.Time `json:"user_class_sections_assigned_at" validate:"omitempty"`
	UserClassSectionsUnassignedAt *time.Time `json:"user_class_sections_unassigned_at" validate:"omitempty"`
}

func (r *UpdateUserClassSectionRequest) ApplyToModel(m *ucsModel.UserClassSectionsModel) {
	if r.UserClassSectionsUserClassID != nil {
		m.UserClassSectionsUserClassID = *r.UserClassSectionsUserClassID
	}
	if r.UserClassSectionsSectionID != nil {
		m.UserClassSectionsSectionID = *r.UserClassSectionsSectionID
	}
	if r.UserClassSectionsMasjidID != nil {
		m.UserClassSectionsMasjidID = *r.UserClassSectionsMasjidID
	}
	if r.UserClassSectionsAssignedAt != nil {
		m.UserClassSectionsAssignedAt = r.UserClassSectionsAssignedAt
	}
	if r.UserClassSectionsUnassignedAt != nil {
		m.UserClassSectionsUnassignedAt = r.UserClassSectionsUnassignedAt
	}
	m.UserClassSectionsUpdatedAt = time.Now()
}

/* ===================== QUERIES ===================== */

type ListUserClassSectionQuery struct {
	UserClassID *uuid.UUID `query:"user_class_id"`
	SectionID   *uuid.UUID `query:"section_id"`
	MasjidID    *uuid.UUID `query:"masjid_id"`
	ActiveOnly  *bool      `query:"active_only"` // true => unassigned_at IS NULL

	Limit  int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	Sort   *string `query:"sort"` // assigned_at_desc|assigned_at_asc|created_at_desc|created_at_asc
}

/* ===================== RESPONSES ===================== */

type UserClassSectionResponse struct {
	UserClassSectionsID           uuid.UUID  `json:"user_class_sections_id"`
	UserClassSectionsUserClassID  uuid.UUID  `json:"user_class_sections_user_class_id"`
	UserClassSectionsSectionID    uuid.UUID  `json:"user_class_sections_section_id"`
	UserClassSectionsMasjidID     uuid.UUID  `json:"user_class_sections_masjid_id"`
	UserClassSectionsAssignedAt   *time.Time `json:"user_class_sections_assigned_at,omitempty"`
	UserClassSectionsUnassignedAt *time.Time `json:"user_class_sections_unassigned_at,omitempty"`

	// Tambahan dari user_classes (opsional)
	UserClassesStatus string `json:"user_classes_status,omitempty"`

	UserClassSectionsCreatedAt time.Time  `json:"user_class_sections_created_at"`
	UserClassSectionsUpdatedAt time.Time  `json:"user_class_sections_updated_at"`
	UserClassSectionsDeletedAt *time.Time `json:"user_class_sections_deleted_at,omitempty"`

	User    *UcsUser        `json:"user,omitempty"`
	Profile *UcsUserProfile `json:"profile,omitempty"`
}

func NewUserClassSectionResponse(m *ucsModel.UserClassSectionsModel) *UserClassSectionResponse {
	if m == nil {
		return nil
	}
	resp := &UserClassSectionResponse{
		UserClassSectionsID:           m.UserClassSectionsID,
		UserClassSectionsUserClassID:  m.UserClassSectionsUserClassID,
		UserClassSectionsSectionID:    m.UserClassSectionsSectionID,
		UserClassSectionsMasjidID:     m.UserClassSectionsMasjidID,
		UserClassSectionsAssignedAt:   m.UserClassSectionsAssignedAt,
		UserClassSectionsUnassignedAt: m.UserClassSectionsUnassignedAt,
		UserClassSectionsCreatedAt:    m.UserClassSectionsCreatedAt,
		UserClassSectionsUpdatedAt:    m.UserClassSectionsUpdatedAt,
	}
	if m.UserClassSectionsDeletedAt.Valid {
		t := m.UserClassSectionsDeletedAt.Time
		resp.UserClassSectionsDeletedAt = &t
	}
	return resp
}

func (r *UserClassSectionResponse) WithUser(u *UcsUser, p *UcsUserProfile) *UserClassSectionResponse {
	r.User, r.Profile = u, p
	return r
}

/* ========== Tipe ringkas untuk enrichment ========== */

type UcsUser struct {
	ID       uuid.UUID  `json:"id"`
	UserName string     `json:"user_name"`
	FullName *string    `json:"full_name,omitempty"` // dari tabel users
	Email    string     `json:"email"`
	IsActive bool       `json:"is_active"`
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
