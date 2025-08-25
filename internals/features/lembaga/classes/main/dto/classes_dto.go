// dto/class_dto.go
package dto

import (
	"masjidku_backend/internals/features/lembaga/classes/main/model"
	"time"

	"github.com/google/uuid"
)

/* ========== REQUEST DTOs ========== */

// CreateClassRequest: payload saat create
type CreateClassRequest struct {
	ClassMasjidID      *uuid.UUID `json:"class_masjid_id"       form:"class_masjid_id"`
	ClassName          string     `json:"class_name"            form:"class_name"            validate:"required,min=2,max=120"`
	ClassSlug          string     `json:"class_slug"            form:"class_slug"            validate:"omitempty,min=2,max=160"`
	ClassDescription   *string    `json:"class_description"     form:"class_description"`
	ClassLevel         *string    `json:"class_level"           form:"class_level"`
	ClassImageURL      *string    `json:"class_image_url"       form:"class_image_url"       validate:"omitempty,url"`
	ClassIsActive      *bool      `json:"class_is_active"       form:"class_is_active"`
}

type UpdateClassRequest struct {
	ClassMasjidID      *uuid.UUID `json:"class_masjid_id"       form:"class_masjid_id"`
	ClassName          *string    `json:"class_name"            form:"class_name"            validate:"omitempty,min=2,max=120"`
	ClassSlug          *string    `json:"class_slug"            form:"class_slug"            validate:"omitempty,min=2,max=160"`
	ClassDescription   *string    `json:"class_description"     form:"class_description"`
	ClassLevel         *string    `json:"class_level"           form:"class_level"`
	ClassImageURL      *string    `json:"class_image_url"       form:"class_image_url"       validate:"omitempty,url"`
	ClassIsActive      *bool      `json:"class_is_active"       form:"class_is_active"`
}

/* ========== RESPONSE DTO ========== */

type ClassResponse struct {
	ClassID            uuid.UUID  `json:"class_id"`
	ClassMasjidID      *uuid.UUID `json:"class_masjid_id,omitempty"`

	ClassName          string     `json:"class_name"`
	ClassSlug          string     `json:"class_slug"`
	ClassDescription   *string    `json:"class_description,omitempty"`
	ClassLevel         *string    `json:"class_level,omitempty"`
	ClassImageURL      *string    `json:"class_image_url,omitempty"`
	ClassIsActive      bool       `json:"class_is_active"`

	ClassCreatedAt     time.Time  `json:"class_created_at"`
	ClassUpdatedAt     *time.Time `json:"class_updated_at,omitempty"`
}

/* ========== QUERY / FILTER DTO (opsional untuk list) ========== */

type ListClassQuery struct {
	MasjidID   *uuid.UUID `query:"masjid_id"`  // /classes?masjid_id=...
	ActiveOnly *bool      `query:"active"`     // /classes?active=true
	Search     *string    `query:"search"`     // /classes?search=tahfidz (match name/level)
	Limit      int        `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset     int        `query:"offset" validate:"omitempty,min=0"`
	Sort       *string    `query:"sort"` // e.g. "created_at_desc", "name_asc"
}

/* ========== HELPER: KONVERSI MODEL <-> DTO ========== */

func NewClassResponse(m *model.ClassModel) *ClassResponse {
	if m == nil {
		return nil
	}
	return &ClassResponse{
		ClassID:            m.ClassID,
		ClassMasjidID:      m.ClassMasjidID,
		ClassName:          m.ClassName,
		ClassSlug:          m.ClassSlug,
		ClassDescription:   m.ClassDescription,
		ClassLevel:         m.ClassLevel,
		ClassImageURL:      m.ClassImageURL,
		ClassIsActive:      m.ClassIsActive,
		ClassCreatedAt:     m.ClassCreatedAt,
		ClassUpdatedAt:     m.ClassUpdatedAt,
	}
}

// ToModel: mapping CreateClassRequest -> ClassModel (untuk Create)
func (r *CreateClassRequest) ToModel() *model.ClassModel {
	now := time.Now()
	m := &model.ClassModel{
		ClassMasjidID:      r.ClassMasjidID,
		ClassName:          r.ClassName,
		ClassSlug:          r.ClassSlug,
		ClassDescription:   r.ClassDescription,
		ClassLevel:         r.ClassLevel,
		ClassImageURL:      r.ClassImageURL,
		ClassIsActive:      true, // default
		ClassCreatedAt:     now,
	}
	if r.ClassIsActive != nil {
		m.ClassIsActive = *r.ClassIsActive
	}
	return m
}

// ApplyToModel: mapping UpdateClassRequest -> partial update (untuk Update)
func (r *UpdateClassRequest) ApplyToModel(m *model.ClassModel) {
	if r.ClassMasjidID != nil {
		m.ClassMasjidID = r.ClassMasjidID
	}
	if r.ClassName != nil {
		m.ClassName = *r.ClassName
	}
	if r.ClassSlug != nil {
		m.ClassSlug = *r.ClassSlug
	}
	if r.ClassDescription != nil {
		// boleh nil untuk clear description
		m.ClassDescription = r.ClassDescription
	}
	if r.ClassLevel != nil {
		m.ClassLevel = r.ClassLevel
	}
	if r.ClassImageURL != nil {
		// boleh nil untuk clear image
		m.ClassImageURL = r.ClassImageURL
	}
	if r.ClassIsActive != nil {
		m.ClassIsActive = *r.ClassIsActive
	}
	now := time.Now()
	m.ClassUpdatedAt = &now
}
