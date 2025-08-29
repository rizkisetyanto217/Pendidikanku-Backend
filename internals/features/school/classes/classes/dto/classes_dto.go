// dto/class_dto.go
package dto

import (
	"strings"
	"time"

	"masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

/* ========== REQUEST DTOs ========== */

// CreateClassRequest: payload saat create
type CreateClassRequest struct {
	ClassMasjidID    uuid.UUID `json:"class_masjid_id"    form:"class_masjid_id"    validate:"required"`
	ClassName        string    `json:"class_name"         form:"class_name"         validate:"required,min=2,max=120"`
	ClassSlug        string    `json:"class_slug"         form:"class_slug"         validate:"omitempty,min=2,max=160"`

	ClassCode        *string   `json:"class_code"         form:"class_code"         validate:"omitempty,max=40"`
	ClassMode        *string   `json:"class_mode"         form:"class_mode"         validate:"omitempty,max=100"` // opsional, tidak ada default

	ClassDescription *string   `json:"class_description"  form:"class_description"`
	ClassLevel       *string   `json:"class_level"        form:"class_level"`
	ClassImageURL    *string   `json:"class_image_url"    form:"class_image_url"    validate:"omitempty,url"`
	ClassIsActive    *bool     `json:"class_is_active"    form:"class_is_active"`
	// trash_url & delete_pending_until biasanya tidak diisi saat create
}

type UpdateClassRequest struct {
	ClassMasjidID           *uuid.UUID `json:"class_masjid_id"            form:"class_masjid_id"` // opsional: pindah masjid
	ClassName               *string    `json:"class_name"                 form:"class_name"                 validate:"omitempty,min=2,max=120"`
	ClassSlug               *string    `json:"class_slug"                 form:"class_slug"                 validate:"omitempty,min=2,max=160"`

	ClassCode               *string    `json:"class_code"                 form:"class_code"                 validate:"omitempty,max=40"`
	ClassMode               *string    `json:"class_mode"                 form:"class_mode"                 validate:"omitempty,max=100"` // opsional

	ClassDescription        *string    `json:"class_description"          form:"class_description"`
	ClassLevel              *string    `json:"class_level"                form:"class_level"`
	ClassImageURL           *string    `json:"class_image_url"            form:"class_image_url"            validate:"omitempty,url"`
	ClassIsActive           *bool      `json:"class_is_active"            form:"class_is_active"`
	ClassTrashURL           *string    `json:"class_trash_url"            form:"class_trash_url"`
	ClassDeletePendingUntil *time.Time `json:"class_delete_pending_until" form:"class_delete_pending_until"` // NULL = tidak pending
}

/* ========== RESPONSE DTO ========== */

type ClassResponse struct {
	ClassID               uuid.UUID  `json:"class_id"`
	ClassMasjidID         uuid.UUID  `json:"class_masjid_id"`

	ClassName             string     `json:"class_name"`
	ClassSlug             string     `json:"class_slug"`
	ClassCode             *string    `json:"class_code,omitempty"`
	ClassMode             string     `json:"class_mode"` // bisa empty string kalau belum diisi

	ClassDescription      *string    `json:"class_description,omitempty"`
	ClassLevel            *string    `json:"class_level,omitempty"`
	ClassImageURL         *string    `json:"class_image_url,omitempty"`
	ClassIsActive         bool       `json:"class_is_active"`

	ClassTrashURL           *string    `json:"class_trash_url,omitempty"`
	ClassDeletePendingUntil *time.Time `json:"class_delete_pending_until,omitempty"`

	ClassCreatedAt        time.Time  `json:"class_created_at"`
	ClassUpdatedAt        time.Time  `json:"class_updated_at"`
}

/* ========== QUERY / FILTER DTO (untuk list) ========== */

type ListClassQuery struct {
	MasjidID   *uuid.UUID `query:"masjid_id"`
	ActiveOnly *bool      `query:"active"`
	Mode       *string    `query:"mode"`   // filter case-insensitive, gunakan LOWER di repo
	Code       *string    `query:"code"`
	Search     *string    `query:"search"` // cari di name/level/desc
	Limit      int        `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset     int        `query:"offset" validate:"omitempty,min=0"`
	Sort       *string    `query:"sort"`   // "created_at_desc", "name_asc", dll.
}

/* ========== HELPERS ========== */

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	return &s
}

/* ========== KONVERSI MODEL <-> DTO ========== */

func NewClassResponse(m *model.ClassModel) *ClassResponse {
	if m == nil {
		return nil
	}
	return &ClassResponse{
		ClassID:                m.ClassID,
		ClassMasjidID:          m.ClassMasjidID,
		ClassName:              m.ClassName,
		ClassSlug:              m.ClassSlug,
		ClassCode:              m.ClassCode,
		ClassMode:              m.ClassMode, // bisa kosong jika belum diisi
		ClassDescription:       m.ClassDescription,
		ClassLevel:             m.ClassLevel,
		ClassImageURL:          m.ClassImageURL,
		ClassIsActive:          m.ClassIsActive,
		ClassTrashURL:          m.ClassTrashURL,
		ClassDeletePendingUntil: m.ClassDeletePendingUntil,
		ClassCreatedAt:         m.ClassCreatedAt,
		ClassUpdatedAt:         m.ClassUpdatedAt,
	}
}

// ToModel: mapping Create -> model (tanpa default untuk class_mode)
func (r *CreateClassRequest) ToModel() *model.ClassModel {
	m := &model.ClassModel{
		ClassMasjidID:  r.ClassMasjidID,
		ClassName:      strings.TrimSpace(r.ClassName),
		ClassSlug:      strings.TrimSpace(r.ClassSlug), // boleh kosong; slugify di service bila perlu
		ClassDescription: trimPtr(r.ClassDescription),
		ClassLevel:       trimPtr(r.ClassLevel),
		ClassImageURL:    trimPtr(r.ClassImageURL),
		ClassIsActive:    true, // default aktif kecuali override
	}
	if r.ClassIsActive != nil {
		m.ClassIsActive = *r.ClassIsActive
	}
	if r.ClassCode != nil {
		m.ClassCode = trimPtr(r.ClassCode)
	}
	// class_mode opsional: hanya set bila dikirim
	if r.ClassMode != nil {
		m.ClassMode = strings.TrimSpace(*r.ClassMode) // bisa jadi "" jika mau dikosongkan
	}
	return m
}

// ApplyToModel: mapping Update -> partial update
// Catatan untuk class_mode:
//   - nil   => abaikan (tidak mengubah nilai di DB)
//   - ""    => set empty string (mengosongkan)
func (r *UpdateClassRequest) ApplyToModel(m *model.ClassModel) {
	if r.ClassMasjidID != nil {
		m.ClassMasjidID = *r.ClassMasjidID
	}
	if r.ClassName != nil {
		m.ClassName = strings.TrimSpace(*r.ClassName)
	}
	if r.ClassSlug != nil {
		m.ClassSlug = strings.TrimSpace(*r.ClassSlug)
	}
	if r.ClassCode != nil {
		m.ClassCode = trimPtr(r.ClassCode) // bisa nil utk clear
	}
	if r.ClassMode != nil {
		m.ClassMode = strings.TrimSpace(*r.ClassMode) // bisa empty string untuk clear
	}
	if r.ClassDescription != nil {
		m.ClassDescription = trimPtr(r.ClassDescription) // bisa nil utk clear
	}
	if r.ClassLevel != nil {
		m.ClassLevel = trimPtr(r.ClassLevel) // bisa nil utk clear
	}
	if r.ClassImageURL != nil {
		m.ClassImageURL = trimPtr(r.ClassImageURL) // bisa nil utk clear
	}
	if r.ClassIsActive != nil {
		m.ClassIsActive = *r.ClassIsActive
	}
	if r.ClassTrashURL != nil {
		m.ClassTrashURL = trimPtr(r.ClassTrashURL) // bisa nil utk clear
	}
	if r.ClassDeletePendingUntil != nil {
		m.ClassDeletePendingUntil = r.ClassDeletePendingUntil // nil = tidak pending
	}
}
