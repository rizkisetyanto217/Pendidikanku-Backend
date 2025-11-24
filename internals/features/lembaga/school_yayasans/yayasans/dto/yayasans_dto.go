// internals/features/lembaga/yayasans/dto/yayasan_dto.go
package dto

import (
	"encoding/json"
	"time"

	yModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/yayasans/model"

	"github.com/google/uuid"
)

/* =========================================================
   PATCH FIELD — tri-state detector (absent | null | value)
   ========================================================= */

type PatchField[T any] struct {
	Present bool // true jika key muncul di payload JSON (meski nil)
	Value   *T   // nil = explicit null; non-nil = value
}

func (p *PatchField[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	// null → Value = nil
	if string(b) == "null" {
		p.Value = nil
		return nil
	}
	// decode value
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

// Helper untuk apply sederhana
func applyPatch[T any](dst **T, pf PatchField[T]) {
	if !pf.Present {
		return
	}
	// explicit null → set ke nil
	if pf.Value == nil {
		*dst = nil
		return
	}
	// value → set pointer ke value
	*dst = pf.Value
}

func applyPatchScalar[T any](dst *T, pf PatchField[T]) {
	if !pf.Present || pf.Value == nil {
		return
	}
	*dst = *pf.Value
}

/* ===================== REQUESTS ===================== */

type CreateYayasanRequest struct {
	// Identitas
	YayasanName        string  `json:"yayasan_name" validate:"required,min=2,max=150"`
	YayasanDescription *string `json:"yayasan_description" validate:"omitempty"`
	YayasanBio         *string `json:"yayasan_bio" validate:"omitempty"`

	// Kontak & lokasi
	YayasanAddress  *string `json:"yayasan_address" validate:"omitempty"`
	YayasanCity     *string `json:"yayasan_city" validate:"omitempty"`
	YayasanProvince *string `json:"yayasan_province" validate:"omitempty"`

	// Media & maps
	YayasanGoogleMapsURL *string `json:"yayasan_google_maps_url" validate:"omitempty,url"`

	// Logo (current; _old akan dikelola otomatis saat update)
	YayasanLogoURL       *string `json:"yayasan_logo_url" validate:"omitempty,url"`
	YayasanLogoObjectKey *string `json:"yayasan_logo_object_key" validate:"omitempty"`
	// jangan kirim *_old saat create

	// Domain & slug
	YayasanDomain *string `json:"yayasan_domain" validate:"omitempty,max=80"`
	YayasanSlug   string  `json:"yayasan_slug"   validate:"required,min=3,max=120"`

	// Status & verifikasi (opsional; biasanya sistem yg set)
	YayasanIsActive           *bool                             `json:"yayasan_is_active,omitempty" validate:"omitempty"`
	YayasanVerificationStatus *yModel.YayasanVerificationStatus `json:"yayasan_verification_status,omitempty" validate:"omitempty,oneof=pending approved rejected"`
	YayasanVerifiedAt         *time.Time                        `json:"yayasan_verified_at,omitempty" validate:"omitempty"`
	YayasanVerificationNotes  *string                           `json:"yayasan_verification_notes,omitempty" validate:"omitempty"`
}

func (r *CreateYayasanRequest) ToModel() *yModel.YayasanModel {
	m := &yModel.YayasanModel{
		YayasanName:          r.YayasanName,
		YayasanDescription:   r.YayasanDescription,
		YayasanBio:           r.YayasanBio,
		YayasanAddress:       r.YayasanAddress,
		YayasanCity:          r.YayasanCity,
		YayasanProvince:      r.YayasanProvince,
		YayasanGoogleMapsURL: r.YayasanGoogleMapsURL,

		YayasanLogoURL:       r.YayasanLogoURL,
		YayasanLogoObjectKey: r.YayasanLogoObjectKey,

		YayasanDomain: r.YayasanDomain,
		YayasanSlug:   r.YayasanSlug,
	}
	if r.YayasanIsActive != nil {
		m.YayasanIsActive = *r.YayasanIsActive
	}
	if r.YayasanVerificationStatus != nil {
		m.YayasanVerificationStatus = *r.YayasanVerificationStatus
	}
	if r.YayasanVerifiedAt != nil {
		m.YayasanVerifiedAt = r.YayasanVerifiedAt
	}
	if r.YayasanVerificationNotes != nil {
		m.YayasanVerificationNotes = r.YayasanVerificationNotes
	}
	return m
}

type UpdateYayasanRequest struct {
	// Identitas
	YayasanName        PatchField[string] `json:"yayasan_name"`
	YayasanDescription PatchField[string] `json:"yayasan_description"`
	YayasanBio         PatchField[string] `json:"yayasan_bio"`

	// Kontak & lokasi
	YayasanAddress  PatchField[string] `json:"yayasan_address"`
	YayasanCity     PatchField[string] `json:"yayasan_city"`
	YayasanProvince PatchField[string] `json:"yayasan_province"`

	// Media & maps
	YayasanGoogleMapsURL PatchField[string] `json:"yayasan_google_maps_url"`

	// Logo (2-slot + retensi)
	YayasanLogoURL                PatchField[string]    `json:"yayasan_logo_url"`
	YayasanLogoObjectKey          PatchField[string]    `json:"yayasan_logo_object_key"`
	YayasanLogoURLOld             PatchField[string]    `json:"yayasan_logo_url_old"`
	YayasanLogoObjectKeyOld       PatchField[string]    `json:"yayasan_logo_object_key_old"`
	YayasanLogoDeletePendingUntil PatchField[time.Time] `json:"yayasan_logo_delete_pending_until"`

	// Domain & slug
	YayasanDomain PatchField[string] `json:"yayasan_domain"`
	YayasanSlug   PatchField[string] `json:"yayasan_slug"`

	// Status & verifikasi
	YayasanIsActive           PatchField[bool]                             `json:"yayasan_is_active"`
	YayasanVerificationStatus PatchField[yModel.YayasanVerificationStatus] `json:"yayasan_verification_status"`
	YayasanVerifiedAt         PatchField[time.Time]                        `json:"yayasan_verified_at"`
	YayasanVerificationNotes  PatchField[string]                           `json:"yayasan_verification_notes"`
}

func (r *UpdateYayasanRequest) ApplyToModel(m *yModel.YayasanModel) {
	// string pointers
	applyPatch(&m.YayasanDescription, r.YayasanDescription)
	applyPatch(&m.YayasanBio, r.YayasanBio)
	applyPatch(&m.YayasanAddress, r.YayasanAddress)
	applyPatch(&m.YayasanCity, r.YayasanCity)
	applyPatch(&m.YayasanProvince, r.YayasanProvince)
	applyPatch(&m.YayasanGoogleMapsURL, r.YayasanGoogleMapsURL)
	applyPatch(&m.YayasanDomain, r.YayasanDomain)

	// scalar string (non-pointer): name & slug
	applyPatchScalar(&m.YayasanName, r.YayasanName)
	applyPatchScalar(&m.YayasanSlug, r.YayasanSlug)

	// status & verifikasi
	applyPatchScalar(&m.YayasanIsActive, r.YayasanIsActive)
	if r.YayasanVerificationStatus.Present && r.YayasanVerificationStatus.Value != nil {
		m.YayasanVerificationStatus = *r.YayasanVerificationStatus.Value
	}
	if r.YayasanVerifiedAt.Present {
		if r.YayasanVerifiedAt.Value == nil {
			m.YayasanVerifiedAt = nil
		} else {
			m.YayasanVerifiedAt = r.YayasanVerifiedAt.Value
		}
	}
	applyPatch(&m.YayasanVerificationNotes, r.YayasanVerificationNotes)

	// Logo 2-slot
	applyPatch(&m.YayasanLogoURL, r.YayasanLogoURL)
	applyPatch(&m.YayasanLogoObjectKey, r.YayasanLogoObjectKey)
	applyPatch(&m.YayasanLogoURLOld, r.YayasanLogoURLOld)
	applyPatch(&m.YayasanLogoObjectKeyOld, r.YayasanLogoObjectKeyOld)
	if r.YayasanLogoDeletePendingUntil.Present {
		if r.YayasanLogoDeletePendingUntil.Value == nil {
			m.YayasanLogoDeletePendingUntil = nil
		} else {
			m.YayasanLogoDeletePendingUntil = r.YayasanLogoDeletePendingUntil.Value
		}
	}

	// updated_at (opsional; GORM autoUpdateTime akan set saat Save/Updates)
	now := time.Now()
	m.YayasanUpdatedAt = now
}

/* ===================== QUERIES ===================== */

type ListYayasanQuery struct {
	YayasanID   *uuid.UUID `query:"yayasan_id"`
	Slug        *string    `query:"slug"`
	Domain      *string    `query:"domain"`
	City        *string    `query:"city"`
	Province    *string    `query:"province"`
	Active      *bool      `query:"active"`
	Verified    *bool      `query:"verified"`
	VerifStatus *string    `query:"verification_status"` // "pending"|"approved"|"rejected"
	Q           *string    `query:"q"`                   // full-text / ilike name

	Limit  int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	Sort   *string `query:"sort"` // name_asc|name_desc|created_at_desc|created_at_asc|updated_at_desc|updated_at_asc
}

/* ===================== RESPONSES ===================== */

type YayasanResponse struct {
	YayasanID uuid.UUID `json:"yayasan_id"`

	// Identitas
	YayasanName        string  `json:"yayasan_name"`
	YayasanDescription *string `json:"yayasan_description,omitempty"`
	YayasanBio         *string `json:"yayasan_bio,omitempty"`

	// Kontak & lokasi
	YayasanAddress  *string `json:"yayasan_address,omitempty"`
	YayasanCity     *string `json:"yayasan_city,omitempty"`
	YayasanProvince *string `json:"yayasan_province,omitempty"`

	// Media & maps
	YayasanGoogleMapsURL *string `json:"yayasan_google_maps_url,omitempty"`

	// Logo (2-slot)
	YayasanLogoURL                *string    `json:"yayasan_logo_url,omitempty"`
	YayasanLogoObjectKey          *string    `json:"yayasan_logo_object_key,omitempty"`
	YayasanLogoURLOld             *string    `json:"yayasan_logo_url_old,omitempty"`
	YayasanLogoObjectKeyOld       *string    `json:"yayasan_logo_object_key_old,omitempty"`
	YayasanLogoDeletePendingUntil *time.Time `json:"yayasan_logo_delete_pending_until,omitempty"`

	// Domain & slug
	YayasanDomain *string `json:"yayasan_domain,omitempty"`
	YayasanSlug   string  `json:"yayasan_slug"`

	// Status & verifikasi
	YayasanIsActive           bool                             `json:"yayasan_is_active"`
	YayasanIsVerified         bool                             `json:"yayasan_is_verified"`
	YayasanVerificationStatus yModel.YayasanVerificationStatus `json:"yayasan_verification_status"`
	YayasanVerifiedAt         *time.Time                       `json:"yayasan_verified_at,omitempty"`
	YayasanVerificationNotes  *string                          `json:"yayasan_verification_notes,omitempty"`

	YayasanCreatedAt time.Time  `json:"yayasan_created_at"`
	YayasanUpdatedAt time.Time  `json:"yayasan_updated_at"`
	YayasanDeletedAt *time.Time `json:"yayasan_deleted_at,omitempty"`
}

func NewYayasanResponse(m *yModel.YayasanModel) *YayasanResponse {
	if m == nil {
		return nil
	}
	resp := &YayasanResponse{
		YayasanID:            m.YayasanID,
		YayasanName:          m.YayasanName,
		YayasanDescription:   m.YayasanDescription,
		YayasanBio:           m.YayasanBio,
		YayasanAddress:       m.YayasanAddress,
		YayasanCity:          m.YayasanCity,
		YayasanProvince:      m.YayasanProvince,
		YayasanGoogleMapsURL: m.YayasanGoogleMapsURL,

		YayasanLogoURL:                m.YayasanLogoURL,
		YayasanLogoObjectKey:          m.YayasanLogoObjectKey,
		YayasanLogoURLOld:             m.YayasanLogoURLOld,
		YayasanLogoObjectKeyOld:       m.YayasanLogoObjectKeyOld,
		YayasanLogoDeletePendingUntil: m.YayasanLogoDeletePendingUntil,

		YayasanDomain: m.YayasanDomain,
		YayasanSlug:   m.YayasanSlug,

		YayasanIsActive:           m.YayasanIsActive,
		YayasanIsVerified:         m.YayasanIsVerified,
		YayasanVerificationStatus: m.YayasanVerificationStatus,
		YayasanVerifiedAt:         m.YayasanVerifiedAt,
		YayasanVerificationNotes:  m.YayasanVerificationNotes,

		YayasanCreatedAt: m.YayasanCreatedAt,
		YayasanUpdatedAt: m.YayasanUpdatedAt,
	}
	if m.YayasanDeletedAt.Valid {
		t := m.YayasanDeletedAt.Time
		resp.YayasanDeletedAt = &t
	}
	return resp
}
