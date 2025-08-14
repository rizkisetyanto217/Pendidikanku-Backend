// internals/features/lembaga/classes/user_class_sections/main/dto/user_class_section_dto.go
package dto

import (
	"time"

	ucsModel "masjidku_backend/internals/features/lembaga/class_sections/main/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

type CreateUserClassSectionRequest struct {
	UserClassSectionsUserClassID   uuid.UUID  `json:"user_class_sections_user_class_id" validate:"required"`
	UserClassSectionsSectionID     uuid.UUID  `json:"user_class_sections_section_id" validate:"required"`
	UserClassSectionsMasjidID      *uuid.UUID `json:"user_class_sections_masjid_id" validate:"omitempty"`
	UserClassSectionsStatus        *string    `json:"user_class_sections_status" validate:"omitempty,oneof=active inactive ended"`
	UserClassSectionsAssignedAt    *time.Time `json:"user_class_sections_assigned_at" validate:"omitempty"`
	UserClassSectionsUnassignedAt  *time.Time `json:"user_class_sections_unassigned_at" validate:"omitempty"`
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

	// AssignedAt default: today/now (DB juga default CURRENT_DATE, tapi set di app untuk konsistensi)
	if r.UserClassSectionsAssignedAt != nil {
		m.UserClassSectionsAssignedAt = *r.UserClassSectionsAssignedAt
	} else {
		m.UserClassSectionsAssignedAt = time.Now()
	}

	// UnassignedAt opsional
	if r.UserClassSectionsUnassignedAt != nil {
		m.UserClassSectionsUnassignedAt = r.UserClassSectionsUnassignedAt
	}
	return m
}

type UpdateUserClassSectionRequest struct {
	UserClassSectionsUserClassID   *uuid.UUID `json:"user_class_sections_user_class_id" validate:"omitempty"`
	UserClassSectionsSectionID     *uuid.UUID `json:"user_class_sections_section_id" validate:"omitempty"`
	UserClassSectionsMasjidID      *uuid.UUID `json:"user_class_sections_masjid_id" validate:"omitempty"`
	UserClassSectionsStatus        *string    `json:"user_class_sections_status" validate:"omitempty,oneof=active inactive ended"`
	UserClassSectionsAssignedAt    *time.Time `json:"user_class_sections_assigned_at" validate:"omitempty"`
	UserClassSectionsUnassignedAt  *time.Time `json:"user_class_sections_unassigned_at" validate:"omitempty"`
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
	m.UserClassSectionsUpdatedAt = &now
}

/* ===================== QUERIES ===================== */

type ListUserClassSectionQuery struct {
	UserClassID *uuid.UUID `query:"user_class_id"`
	SectionID   *uuid.UUID `query:"section_id"`
	MasjidID    *uuid.UUID `query:"masjid_id"`
	Status      *string    `query:"status"`      // active|inactive|ended
	ActiveOnly  *bool      `query:"active_only"` // true => status=active AND unassigned_at IS NULL

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

	UserClassSectionsStatus       string     `json:"user_class_sections_status"`
	UserClassSectionsAssignedAt   time.Time  `json:"user_class_sections_assigned_at"`
	UserClassSectionsUnassignedAt *time.Time `json:"user_class_sections_unassigned_at,omitempty"`

	UserClassSectionsCreatedAt    time.Time  `json:"user_class_sections_created_at"`
	UserClassSectionsUpdatedAt    *time.Time `json:"user_class_sections_updated_at,omitempty"`
}

func NewUserClassSectionResponse(m *ucsModel.UserClassSectionsModel) *UserClassSectionResponse {
	if m == nil {
		return nil
	}
	return &UserClassSectionResponse{
		UserClassSectionsID:           m.UserClassSectionsID,
		UserClassSectionsUserClassID:  m.UserClassSectionsUserClassID,
		UserClassSectionsSectionID:    m.UserClassSectionsSectionID,
		UserClassSectionsMasjidID:     m.UserClassSectionsMasjidID,

		UserClassSectionsAssignedAt:   m.UserClassSectionsAssignedAt,
		UserClassSectionsUnassignedAt: m.UserClassSectionsUnassignedAt,

		UserClassSectionsCreatedAt:    m.UserClassSectionsCreatedAt,
		UserClassSectionsUpdatedAt:    m.UserClassSectionsUpdatedAt,
	}
}
