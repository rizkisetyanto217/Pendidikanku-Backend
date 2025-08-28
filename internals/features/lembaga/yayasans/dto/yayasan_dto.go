// internals/features/lembaga/yayasans/dto/yayasan_dto.go
package dto

import (
	"time"

	yModel "masjidku_backend/internals/features/lembaga/yayasans/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

type CreateYayasanRequest struct {
	YayasanName        string     `json:"yayasan_name" validate:"required,min=2,max=150"`
	YayasanLegalNumber *string    `json:"yayasan_legal_number" validate:"omitempty"`
	YayasanLegalDate   *time.Time `json:"yayasan_legal_date" validate:"omitempty"`

	YayasanNPWP *string `json:"yayasan_npwp" validate:"omitempty,max=32"`

	// Kontak & lokasi
	YayasanAddress  *string  `json:"yayasan_address" validate:"omitempty"`
	YayasanCity     *string  `json:"yayasan_city" validate:"omitempty"`
	YayasanProvince *string  `json:"yayasan_province" validate:"omitempty"`
	YayasanLatitude  *float64 `json:"yayasan_latitude" validate:"omitempty"`
	YayasanLongitude *float64 `json:"yayasan_longitude" validate:"omitempty"`

	// Media & maps
	YayasanLogoURL               *string    `json:"yayasan_logo_url" validate:"omitempty,url"`
	YayasanGoogleMapsURL         *string    `json:"yayasan_google_maps_url" validate:"omitempty,url"`
	YayasanLogoDeletePendingUntil *time.Time `json:"yayasan_logo_delete_pending_until,omitempty" validate:"omitempty"`

	// Domain & slug
	YayasanDomain *string `json:"yayasan_domain" validate:"omitempty,max=80"`
	YayasanSlug    string  `json:"yayasan_slug" validate:"required,min=3,max=120"`

	// Status & verifikasi (opsional; biasanya sistem yang set via trigger)
	YayasanIsActive           *bool                          `json:"yayasan_is_active,omitempty" validate:"omitempty"`
	YayasanVerificationStatus *yModel.YayasanVerificationStatus `json:"yayasan_verification_status,omitempty" validate:"omitempty,oneof=pending approved rejected"`
	YayasanVerifiedAt         *time.Time                     `json:"yayasan_verified_at,omitempty" validate:"omitempty"`
	YayasanVerificationNotes  *string                        `json:"yayasan_verification_notes,omitempty" validate:"omitempty"`

	// Sosial
	YayasanWebsiteURL  *string `json:"yayasan_website_url,omitempty" validate:"omitempty,url"`
	YayasanInstagramURL *string `json:"yayasan_instagram_url,omitempty" validate:"omitempty,url"`
	YayasanWhatsappURL  *string `json:"yayasan_whatsapp_url,omitempty" validate:"omitempty,url"`
	YayasanYoutubeURL   *string `json:"yayasan_youtube_url,omitempty" validate:"omitempty,url"`
	YayasanFacebookURL  *string `json:"yayasan_facebook_url,omitempty" validate:"omitempty,url"`
	YayasanTiktokURL    *string `json:"yayasan_tiktok_url,omitempty" validate:"omitempty,url"`
}

func (r *CreateYayasanRequest) ToModel() *yModel.YayasanModel {
	m := &yModel.YayasanModel{
		YayasanName:        r.YayasanName,
		YayasanLegalNumber: r.YayasanLegalNumber,
		YayasanLegalDate:   r.YayasanLegalDate,
		YayasanNPWP:        r.YayasanNPWP,

		YayasanAddress:  r.YayasanAddress,
		YayasanCity:     r.YayasanCity,
		YayasanProvince: r.YayasanProvince,
		YayasanLatitude:  r.YayasanLatitude,
		YayasanLongitude: r.YayasanLongitude,

		YayasanLogoURL:               r.YayasanLogoURL,
		YayasanGoogleMapsURL:         r.YayasanGoogleMapsURL,
		YayasanLogoDeletePendingUntil: r.YayasanLogoDeletePendingUntil,

		YayasanDomain: r.YayasanDomain,
		YayasanSlug:   r.YayasanSlug,

		// default via DB trigger/status; tetapi hormati input jika diisi
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
	if r.YayasanWebsiteURL != nil {
		m.YayasanWebsiteURL = r.YayasanWebsiteURL
	}
	if r.YayasanInstagramURL != nil {
		m.YayasanInstagramURL = r.YayasanInstagramURL
	}
	if r.YayasanWhatsappURL != nil {
		m.YayasanWhatsappURL = r.YayasanWhatsappURL
	}
	if r.YayasanYoutubeURL != nil {
		m.YayasanYoutubeURL = r.YayasanYoutubeURL
	}
	if r.YayasanFacebookURL != nil {
		m.YayasanFacebookURL = r.YayasanFacebookURL
	}
	if r.YayasanTiktokURL != nil {
		m.YayasanTiktokURL = r.YayasanTiktokURL
	}
	return m
}

type UpdateYayasanRequest struct {
	YayasanName        *string    `json:"yayasan_name" validate:"omitempty,min=2,max=150"`
	YayasanLegalNumber *string    `json:"yayasan_legal_number" validate:"omitempty"`
	YayasanLegalDate   *time.Time `json:"yayasan_legal_date" validate:"omitempty"`
	YayasanNPWP        *string    `json:"yayasan_npwp" validate:"omitempty,max=32"`

	YayasanAddress  *string  `json:"yayasan_address" validate:"omitempty"`
	YayasanCity     *string  `json:"yayasan_city" validate:"omitempty"`
	YayasanProvince *string  `json:"yayasan_province" validate:"omitempty"`
	YayasanLatitude  *float64 `json:"yayasan_latitude" validate:"omitempty"`
	YayasanLongitude *float64 `json:"yayasan_longitude" validate:"omitempty"`

	YayasanLogoURL               *string    `json:"yayasan_logo_url" validate:"omitempty,url"`
	YayasanGoogleMapsURL         *string    `json:"yayasan_google_maps_url" validate:"omitempty,url"`
	YayasanLogoDeletePendingUntil *time.Time `json:"yayasan_logo_delete_pending_until,omitempty" validate:"omitempty"`

	YayasanDomain *string `json:"yayasan_domain" validate:"omitempty,max=80"`
	YayasanSlug   *string `json:"yayasan_slug" validate:"omitempty,min=3,max=120"`

	YayasanIsActive           *bool                            `json:"yayasan_is_active,omitempty" validate:"omitempty"`
	YayasanVerificationStatus *yModel.YayasanVerificationStatus `json:"yayasan_verification_status,omitempty" validate:"omitempty,oneof=pending approved rejected"`
	YayasanVerifiedAt         *time.Time                       `json:"yayasan_verified_at,omitempty" validate:"omitempty"`
	YayasanVerificationNotes  *string                          `json:"yayasan_verification_notes,omitempty" validate:"omitempty"`

	YayasanWebsiteURL  *string `json:"yayasan_website_url,omitempty" validate:"omitempty,url"`
	YayasanInstagramURL *string `json:"yayasan_instagram_url,omitempty" validate:"omitempty,url"`
	YayasanWhatsappURL  *string `json:"yayasan_whatsapp_url,omitempty" validate:"omitempty,url"`
	YayasanYoutubeURL   *string `json:"yayasan_youtube_url,omitempty" validate:"omitempty,url"`
	YayasanFacebookURL  *string `json:"yayasan_facebook_url,omitempty" validate:"omitempty,url"`
	YayasanTiktokURL    *string `json:"yayasan_tiktok_url,omitempty" validate:"omitempty,url"`
}

func (r *UpdateYayasanRequest) ApplyToModel(m *yModel.YayasanModel) {
	if r.YayasanName != nil {
		m.YayasanName = *r.YayasanName
	}
	if r.YayasanLegalNumber != nil {
		m.YayasanLegalNumber = r.YayasanLegalNumber
	}
	if r.YayasanLegalDate != nil {
		m.YayasanLegalDate = r.YayasanLegalDate
	}
	if r.YayasanNPWP != nil {
		m.YayasanNPWP = r.YayasanNPWP
	}

	if r.YayasanAddress != nil {
		m.YayasanAddress = r.YayasanAddress
	}
	if r.YayasanCity != nil {
		m.YayasanCity = r.YayasanCity
	}
	if r.YayasanProvince != nil {
		m.YayasanProvince = r.YayasanProvince
	}
	if r.YayasanLatitude != nil {
		m.YayasanLatitude = r.YayasanLatitude
	}
	if r.YayasanLongitude != nil {
		m.YayasanLongitude = r.YayasanLongitude
	}

	if r.YayasanLogoURL != nil {
		m.YayasanLogoURL = r.YayasanLogoURL
	}
	if r.YayasanGoogleMapsURL != nil {
		m.YayasanGoogleMapsURL = r.YayasanGoogleMapsURL
	}
	if r.YayasanLogoDeletePendingUntil != nil {
		m.YayasanLogoDeletePendingUntil = r.YayasanLogoDeletePendingUntil
	}

	if r.YayasanDomain != nil {
		m.YayasanDomain = r.YayasanDomain
	}
	if r.YayasanSlug != nil {
		m.YayasanSlug = *r.YayasanSlug
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

	if r.YayasanWebsiteURL != nil {
		m.YayasanWebsiteURL = r.YayasanWebsiteURL
	}
	if r.YayasanInstagramURL != nil {
		m.YayasanInstagramURL = r.YayasanInstagramURL
	}
	if r.YayasanWhatsappURL != nil {
		m.YayasanWhatsappURL = r.YayasanWhatsappURL
	}
	if r.YayasanYoutubeURL != nil {
		m.YayasanYoutubeURL = r.YayasanYoutubeURL
	}
	if r.YayasanFacebookURL != nil {
		m.YayasanFacebookURL = r.YayasanFacebookURL
	}
	if r.YayasanTiktokURL != nil {
		m.YayasanTiktokURL = r.YayasanTiktokURL
	}

	now := time.Now()
	m.YayasanUpdatedAt = &now
}

/* ===================== QUERIES ===================== */

type ListYayasanQuery struct {
	YayasanID *uuid.UUID `query:"yayasan_id"`
	Slug      *string    `query:"slug"`
	Domain    *string    `query:"domain"`
	City      *string    `query:"city"`
	Province  *string    `query:"province"`
	Active    *bool      `query:"active"`
	Verified  *bool      `query:"verified"`
	VerifStatus *string  `query:"verification_status"` // "pending"|"approved"|"rejected"
	Q         *string    `query:"q"`                   // untuk full-text atau ilike name

	Limit  int     `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	Sort   *string `query:"sort"` // name_asc|name_desc|created_at_desc|created_at_asc|updated_at_desc|updated_at_asc
}

/* ===================== RESPONSES ===================== */

type YayasanResponse struct {
	YayasanID uuid.UUID `json:"yayasan_id"`

	YayasanName        string     `json:"yayasan_name"`
	YayasanLegalNumber *string    `json:"yayasan_legal_number,omitempty"`
	YayasanLegalDate   *time.Time `json:"yayasan_legal_date,omitempty"`
	YayasanNPWP        *string    `json:"yayasan_npwp,omitempty"`

	YayasanAddress  *string  `json:"yayasan_address,omitempty"`
	YayasanCity     *string  `json:"yayasan_city,omitempty"`
	YayasanProvince *string  `json:"yayasan_province,omitempty"`
	YayasanLatitude  *float64 `json:"yayasan_latitude,omitempty"`
	YayasanLongitude *float64 `json:"yayasan_longitude,omitempty"`

	YayasanLogoURL               *string    `json:"yayasan_logo_url,omitempty"`
	YayasanLogoTrashURL          *string    `json:"yayasan_logo_trash_url,omitempty"`
	YayasanLogoDeletePendingUntil *time.Time `json:"yayasan_logo_delete_pending_until,omitempty"`
	YayasanGoogleMapsURL         *string    `json:"yayasan_google_maps_url,omitempty"`

	YayasanDomain *string `json:"yayasan_domain,omitempty"`
	YayasanSlug   string  `json:"yayasan_slug"`

	YayasanIsActive           bool                           `json:"yayasan_is_active"`
	YayasanIsVerified         bool                           `json:"yayasan_is_verified"`
	YayasanVerificationStatus yModel.YayasanVerificationStatus `json:"yayasan_verification_status"`
	YayasanVerifiedAt         *time.Time                     `json:"yayasan_verified_at,omitempty"`
	YayasanVerificationNotes  *string                        `json:"yayasan_verification_notes,omitempty"`

	YayasanWebsiteURL  *string `json:"yayasan_website_url,omitempty"`
	YayasanInstagramURL *string `json:"yayasan_instagram_url,omitempty"`
	YayasanWhatsappURL  *string `json:"yayasan_whatsapp_url,omitempty"`
	YayasanYoutubeURL   *string `json:"yayasan_youtube_url,omitempty"`
	YayasanFacebookURL  *string `json:"yayasan_facebook_url,omitempty"`
	YayasanTiktokURL    *string `json:"yayasan_tiktok_url,omitempty"`

	YayasanCreatedAt time.Time  `json:"yayasan_created_at"`
	YayasanUpdatedAt *time.Time `json:"yayasan_updated_at,omitempty"`
	YayasanDeletedAt *time.Time `json:"yayasan_deleted_at,omitempty"`
}

func NewYayasanResponse(m *yModel.YayasanModel) *YayasanResponse {
	if m == nil {
		return nil
	}
	resp := &YayasanResponse{
		YayasanID: m.YayasanID,

		YayasanName:        m.YayasanName,
		YayasanLegalNumber: m.YayasanLegalNumber,
		YayasanLegalDate:   m.YayasanLegalDate,
		YayasanNPWP:        m.YayasanNPWP,

		YayasanAddress:  m.YayasanAddress,
		YayasanCity:     m.YayasanCity,
		YayasanProvince: m.YayasanProvince,
		YayasanLatitude:  m.YayasanLatitude,
		YayasanLongitude: m.YayasanLongitude,

		YayasanLogoURL:               m.YayasanLogoURL,
		YayasanLogoTrashURL:          m.YayasanLogoTrashURL,
		YayasanLogoDeletePendingUntil: m.YayasanLogoDeletePendingUntil,
		YayasanGoogleMapsURL:         m.YayasanGoogleMapsURL,

		YayasanDomain: m.YayasanDomain,
		YayasanSlug:   m.YayasanSlug,

		YayasanIsActive:           m.YayasanIsActive,
		YayasanIsVerified:         m.YayasanIsVerified,
		YayasanVerificationStatus: m.YayasanVerificationStatus,
		YayasanVerifiedAt:         m.YayasanVerifiedAt,
		YayasanVerificationNotes:  m.YayasanVerificationNotes,

		YayasanWebsiteURL:  m.YayasanWebsiteURL,
		YayasanInstagramURL: m.YayasanInstagramURL,
		YayasanWhatsappURL:  m.YayasanWhatsappURL,
		YayasanYoutubeURL:   m.YayasanYoutubeURL,
		YayasanFacebookURL:  m.YayasanFacebookURL,
		YayasanTiktokURL:    m.YayasanTiktokURL,

		YayasanCreatedAt: m.YayasanCreatedAt,
		YayasanUpdatedAt: m.YayasanUpdatedAt,
	}
	if m.YayasanDeletedAt.Valid {
		t := m.YayasanDeletedAt.Time
		resp.YayasanDeletedAt = &t
	}
	return resp
}
