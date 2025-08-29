// internals/features/lembaga/classes/user_class_sections/main/dto/user_class_section_dto.go
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
	UserClassSectionsAssignedAt   *time.Time `json:"user_class_sections_assigned_at" validate:"omitempty"`
	UserClassSectionsUnassignedAt *time.Time `json:"user_class_sections_unassigned_at" validate:"omitempty"`
}

func (r *CreateUserClassSectionRequest) ToModel() *ucsModel.UserClassSectionsModel {
	m := &ucsModel.UserClassSectionsModel{
		UserClassSectionsUserClassID: r.UserClassSectionsUserClassID,
		UserClassSectionsSectionID:   r.UserClassSectionsSectionID,
		UserClassSectionsMasjidID:    uuid.Nil,
	}
	if r.UserClassSectionsMasjidID != nil {
		m.UserClassSectionsMasjidID = *r.UserClassSectionsMasjidID
	}
	if r.UserClassSectionsAssignedAt != nil {
		m.UserClassSectionsAssignedAt = *r.UserClassSectionsAssignedAt
	} else {
		m.UserClassSectionsAssignedAt = time.Now()
	}
	if r.UserClassSectionsUnassignedAt != nil {
		m.UserClassSectionsUnassignedAt = r.UserClassSectionsUnassignedAt
	}
	return m
}

type UpdateUserClassSectionRequest struct {
	UserClassSectionsUserClassID  *uuid.UUID `json:"user_class_sections_user_class_id" validate:"omitempty"`
	UserClassSectionsSectionID    *uuid.UUID `json:"user_class_sections_section_id" validate:"omitempty"`
	UserClassSectionsMasjidID     *uuid.UUID `json:"user_class_sections_masjid_id" validate:"omitempty"`
	UserClassSectionsAssignedAt   *time.Time `json:"user_class_sections_assigned_at" validate:"omitempty"`
	UserClassSectionsUnassignedAt *time.Time `json:"user_class_sections_unassigned_at" validate:"omitempty"`
	// Tidak expose DeletedAt di update request
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
		m.UserClassSectionsAssignedAt = *r.UserClassSectionsAssignedAt
	}
	if r.UserClassSectionsUnassignedAt != nil {
		m.UserClassSectionsUnassignedAt = r.UserClassSectionsUnassignedAt
	}
	now := time.Now()
	m.UserClassSectionsUpdatedAt = now
}

/* ===================== QUERIES ===================== */

type ListUserClassSectionQuery struct {
	UserClassID *uuid.UUID `query:"user_class_id"`
	SectionID   *uuid.UUID `query:"section_id"`
	MasjidID    *uuid.UUID `query:"masjid_id"`
	Status      *string    `query:"status"`      // active|inactive|ended
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
	UserClassSectionsAssignedAt   time.Time  `json:"user_class_sections_assigned_at"`
	UserClassSectionsUnassignedAt *time.Time `json:"user_class_sections_unassigned_at,omitempty"`

	// Tambahan dari user_classes
	UserClassesStatus string `json:"user_classes_status,omitempty"`

	UserClassSectionsCreatedAt time.Time   `json:"user_class_sections_created_at"`
	UserClassSectionsUpdatedAt time.Time   `json:"user_class_sections_updated_at"`
	UserClassSectionsDeletedAt *time.Time  `json:"user_class_sections_deleted_at,omitempty"`

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
	// DeletedAt opsional: hanya diisi jika ada (gorm.DeletedAt valid)
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

/* ========== Tipe ringkas (rename dari UserLite/UserProfileLite) ========== */

type UcsUser struct {
	ID       uuid.UUID `json:"id"`
	UserName string    `json:"user_name"`
	Email    string    `json:"email"`
	IsActive bool      `json:"is_active"`
}

type UcsUserProfile struct {
	UserID       uuid.UUID  `json:"user_id"`
	DonationName string     `json:"donation_name"`
	FullName     string     `json:"full_name"`
	FatherName   string     `json:"father_name"`
	MotherName   string     `json:"mother_name"`
	DateOfBirth  *time.Time `json:"date_of_birth,omitempty"`
	Gender       *string    `json:"gender,omitempty"` // "male" | "female"
	PhoneNumber  string     `json:"phone_number"`
	Bio          string     `json:"bio"`
	Location     string     `json:"location"`
	Occupation   string     `json:"occupation"`
}
