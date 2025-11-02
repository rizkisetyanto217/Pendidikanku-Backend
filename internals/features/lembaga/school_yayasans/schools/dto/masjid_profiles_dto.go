package dto

import (
	"time"

	"github.com/google/uuid"

	m "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"
)

/* =======================================================
   REQUEST DTOs
   ======================================================= */

// Create: minimal wajib school_id; field lain opsional
type SchoolProfileCreateRequest struct {
	SchoolProfileSchoolID string `json:"school_profile_school_id" validate:"required,uuid4"`

	SchoolProfileDescription *string `json:"school_profile_description,omitempty"`
	SchoolProfileFoundedYear *int    `json:"school_profile_founded_year,omitempty"`

	// Alamat & kontak
	SchoolProfileAddress      *string `json:"school_profile_address,omitempty"`
	SchoolProfileContactPhone *string `json:"school_profile_contact_phone,omitempty"`
	SchoolProfileContactEmail *string `json:"school_profile_contact_email,omitempty"`

	// Sosial/link publik
	SchoolProfileGoogleMapsURL          *string `json:"school_profile_google_maps_url,omitempty"`
	SchoolProfileInstagramURL           *string `json:"school_profile_instagram_url,omitempty"`
	SchoolProfileWhatsappURL            *string `json:"school_profile_whatsapp_url,omitempty"`
	SchoolProfileYoutubeURL             *string `json:"school_profile_youtube_url,omitempty"`
	SchoolProfileFacebookURL            *string `json:"school_profile_facebook_url,omitempty"`
	SchoolProfileTiktokURL              *string `json:"school_profile_tiktok_url,omitempty"`
	SchoolProfileWhatsappGroupIkhwanURL *string `json:"school_profile_whatsapp_group_ikhwan_url,omitempty"`
	SchoolProfileWhatsappGroupAkhwatURL *string `json:"school_profile_whatsapp_group_akhwat_url,omitempty"`
	SchoolProfileWebsiteURL             *string `json:"school_profile_website_url,omitempty"`

	// Koordinat
	SchoolProfileLatitude  *float64 `json:"school_profile_latitude,omitempty"`
	SchoolProfileLongitude *float64 `json:"school_profile_longitude,omitempty"`

	// Profil sekolah (opsional)
	SchoolProfileSchoolNPSN            *string    `json:"school_profile_school_npsn,omitempty"`
	SchoolProfileSchoolNSS             *string    `json:"school_profile_school_nss,omitempty"`
	SchoolProfileSchoolAccreditation   *string    `json:"school_profile_school_accreditation,omitempty"` // A/B/C/Ungraded/-
	SchoolProfileSchoolPrincipalUserID *uuid.UUID `json:"school_profile_school_principal_user_id,omitempty"`
	SchoolProfileSchoolEmail           *string    `json:"school_profile_school_email,omitempty"`
	SchoolProfileSchoolAddress         *string    `json:"school_profile_school_address,omitempty"`
	SchoolProfileSchoolStudentCapacity *int       `json:"school_profile_school_student_capacity,omitempty"`
	SchoolProfileSchoolIsBoarding      *bool      `json:"school_profile_school_is_boarding,omitempty"`
}

// Update (PATCH): semua opsional; hanya yang != nil yang di-apply
type SchoolProfileUpdateRequest struct {
	SchoolProfileDescription *string `json:"school_profile_description,omitempty"`
	SchoolProfileFoundedYear *int    `json:"school_profile_founded_year,omitempty"`

	// Alamat & kontak
	SchoolProfileAddress      *string `json:"school_profile_address,omitempty"`
	SchoolProfileContactPhone *string `json:"school_profile_contact_phone,omitempty"`
	SchoolProfileContactEmail *string `json:"school_profile_contact_email,omitempty"`

	// Sosial/link publik
	SchoolProfileGoogleMapsURL          *string `json:"school_profile_google_maps_url,omitempty"`
	SchoolProfileInstagramURL           *string `json:"school_profile_instagram_url,omitempty"`
	SchoolProfileWhatsappURL            *string `json:"school_profile_whatsapp_url,omitempty"`
	SchoolProfileYoutubeURL             *string `json:"school_profile_youtube_url,omitempty"`
	SchoolProfileFacebookURL            *string `json:"school_profile_facebook_url,omitempty"`
	SchoolProfileTiktokURL              *string `json:"school_profile_tiktok_url,omitempty"`
	SchoolProfileWhatsappGroupIkhwanURL *string `json:"school_profile_whatsapp_group_ikhwan_url,omitempty"`
	SchoolProfileWhatsappGroupAkhwatURL *string `json:"school_profile_whatsapp_group_akhwat_url,omitempty"`
	SchoolProfileWebsiteURL             *string `json:"school_profile_website_url,omitempty"`

	// Koordinat
	SchoolProfileLatitude  *float64 `json:"school_profile_latitude,omitempty"`
	SchoolProfileLongitude *float64 `json:"school_profile_longitude,omitempty"`

	// Profil sekolah
	SchoolProfileSchoolNPSN            *string    `json:"school_profile_school_npsn,omitempty"`
	SchoolProfileSchoolNSS             *string    `json:"school_profile_school_nss,omitempty"`
	SchoolProfileSchoolAccreditation   *string    `json:"school_profile_school_accreditation,omitempty"`
	SchoolProfileSchoolPrincipalUserID *uuid.UUID `json:"school_profile_school_principal_user_id,omitempty"`
	SchoolProfileSchoolEmail           *string    `json:"school_profile_school_email,omitempty"`
	SchoolProfileSchoolAddress         *string    `json:"school_profile_school_address,omitempty"`
	SchoolProfileSchoolStudentCapacity *int       `json:"school_profile_school_student_capacity,omitempty"`
	SchoolProfileSchoolIsBoarding      *bool      `json:"school_profile_school_is_boarding,omitempty"`
}

/* =======================================================
   RESPONSE DTO
   ======================================================= */

type SchoolProfileResponse struct {
	SchoolProfileID       string `json:"school_profile_id"`
	SchoolProfileSchoolID string `json:"school_profile_school_id"`

	SchoolProfileDescription *string `json:"school_profile_description,omitempty"`
	SchoolProfileFoundedYear *int    `json:"school_profile_founded_year,omitempty"`

	// Alamat & kontak
	SchoolProfileAddress      *string `json:"school_profile_address,omitempty"`
	SchoolProfileContactPhone *string `json:"school_profile_contact_phone,omitempty"`
	SchoolProfileContactEmail *string `json:"school_profile_contact_email,omitempty"`

	// Sosial/link publik
	SchoolProfileGoogleMapsURL          *string `json:"school_profile_google_maps_url,omitempty"`
	SchoolProfileInstagramURL           *string `json:"school_profile_instagram_url,omitempty"`
	SchoolProfileWhatsappURL            *string `json:"school_profile_whatsapp_url,omitempty"`
	SchoolProfileYoutubeURL             *string `json:"school_profile_youtube_url,omitempty"`
	SchoolProfileFacebookURL            *string `json:"school_profile_facebook_url,omitempty"`
	SchoolProfileTiktokURL              *string `json:"school_profile_tiktok_url,omitempty"`
	SchoolProfileWhatsappGroupIkhwanURL *string `json:"school_profile_whatsapp_group_ikhwan_url,omitempty"`
	SchoolProfileWhatsappGroupAkhwatURL *string `json:"school_profile_whatsapp_group_akhwat_url,omitempty"`
	SchoolProfileWebsiteURL             *string `json:"school_profile_website_url,omitempty"`

	// Koordinat
	SchoolProfileLatitude  *float64 `json:"school_profile_latitude,omitempty"`
	SchoolProfileLongitude *float64 `json:"school_profile_longitude,omitempty"`

	// Profil sekolah
	SchoolProfileSchoolNPSN            *string    `json:"school_profile_school_npsn,omitempty"`
	SchoolProfileSchoolNSS             *string    `json:"school_profile_school_nss,omitempty"`
	SchoolProfileSchoolAccreditation   *string    `json:"school_profile_school_accreditation,omitempty"`
	SchoolProfileSchoolPrincipalUserID *uuid.UUID `json:"school_profile_school_principal_user_id,omitempty"`
	SchoolProfileSchoolEmail           *string    `json:"school_profile_school_email,omitempty"`
	SchoolProfileSchoolAddress         *string    `json:"school_profile_school_address,omitempty"`
	SchoolProfileSchoolStudentCapacity *int       `json:"school_profile_school_student_capacity,omitempty"`
	SchoolProfileSchoolIsBoarding      bool       `json:"school_profile_school_is_boarding"`

	// Read-only timestamps
	SchoolProfileCreatedAt time.Time `json:"school_profile_created_at"`
	SchoolProfileUpdatedAt time.Time `json:"school_profile_updated_at"`
}

/* =======================================================
   CONVERTERS
   ======================================================= */

func FromModelSchoolProfile(p *m.SchoolProfileModel) SchoolProfileResponse {
	return SchoolProfileResponse{
		SchoolProfileID:       p.SchoolProfileID.String(),
		SchoolProfileSchoolID: p.SchoolProfileSchoolID.String(),

		SchoolProfileDescription: p.SchoolProfileDescription,
		SchoolProfileFoundedYear: p.SchoolProfileFoundedYear,

		// Alamat & kontak
		SchoolProfileAddress:      p.SchoolProfileAddress,
		SchoolProfileContactPhone: p.SchoolProfileContactPhone,
		SchoolProfileContactEmail: p.SchoolProfileContactEmail,

		// Sosial/link
		SchoolProfileGoogleMapsURL:          p.SchoolProfileGoogleMapsURL,
		SchoolProfileInstagramURL:           p.SchoolProfileInstagramURL,
		SchoolProfileWhatsappURL:            p.SchoolProfileWhatsappURL,
		SchoolProfileYoutubeURL:             p.SchoolProfileYoutubeURL,
		SchoolProfileFacebookURL:            p.SchoolProfileFacebookURL,
		SchoolProfileTiktokURL:              p.SchoolProfileTiktokURL,
		SchoolProfileWhatsappGroupIkhwanURL: p.SchoolProfileWhatsappGroupIkhwanURL,
		SchoolProfileWhatsappGroupAkhwatURL: p.SchoolProfileWhatsappGroupAkhwatURL,
		SchoolProfileWebsiteURL:             p.SchoolProfileWebsiteURL,

		// Koordinat
		SchoolProfileLatitude:  p.SchoolProfileLatitude,
		SchoolProfileLongitude: p.SchoolProfileLongitude,

		// Sekolah
		SchoolProfileSchoolNPSN:            p.SchoolProfileSchoolNPSN,
		SchoolProfileSchoolNSS:             p.SchoolProfileSchoolNSS,
		SchoolProfileSchoolAccreditation:   p.SchoolProfileSchoolAccreditation,
		SchoolProfileSchoolPrincipalUserID: p.SchoolProfileSchoolPrincipalUserID,
		SchoolProfileSchoolEmail:           p.SchoolProfileSchoolEmail,
		SchoolProfileSchoolAddress:         p.SchoolProfileSchoolAddress,
		SchoolProfileSchoolStudentCapacity: p.SchoolProfileSchoolStudentCapacity,
		SchoolProfileSchoolIsBoarding:      p.SchoolProfileSchoolIsBoarding,

		SchoolProfileCreatedAt: p.SchoolProfileCreatedAt,
		SchoolProfileUpdatedAt: p.SchoolProfileUpdatedAt,
	}
}

func ToModelSchoolProfileCreate(req *SchoolProfileCreateRequest) *m.SchoolProfileModel {
	schoolID, _ := uuid.Parse(req.SchoolProfileSchoolID)
	model := &m.SchoolProfileModel{
		SchoolProfileSchoolID: schoolID,

		SchoolProfileDescription: req.SchoolProfileDescription,
		SchoolProfileFoundedYear: req.SchoolProfileFoundedYear,

		SchoolProfileAddress:      req.SchoolProfileAddress,
		SchoolProfileContactPhone: req.SchoolProfileContactPhone,
		SchoolProfileContactEmail: req.SchoolProfileContactEmail,

		SchoolProfileGoogleMapsURL:          req.SchoolProfileGoogleMapsURL,
		SchoolProfileInstagramURL:           req.SchoolProfileInstagramURL,
		SchoolProfileWhatsappURL:            req.SchoolProfileWhatsappURL,
		SchoolProfileYoutubeURL:             req.SchoolProfileYoutubeURL,
		SchoolProfileFacebookURL:            req.SchoolProfileFacebookURL,
		SchoolProfileTiktokURL:              req.SchoolProfileTiktokURL,
		SchoolProfileWhatsappGroupIkhwanURL: req.SchoolProfileWhatsappGroupIkhwanURL,
		SchoolProfileWhatsappGroupAkhwatURL: req.SchoolProfileWhatsappGroupAkhwatURL,
		SchoolProfileWebsiteURL:             req.SchoolProfileWebsiteURL,

		SchoolProfileLatitude:  req.SchoolProfileLatitude,
		SchoolProfileLongitude: req.SchoolProfileLongitude,

		SchoolProfileSchoolNPSN:            req.SchoolProfileSchoolNPSN,
		SchoolProfileSchoolNSS:             req.SchoolProfileSchoolNSS,
		SchoolProfileSchoolAccreditation:   req.SchoolProfileSchoolAccreditation,
		SchoolProfileSchoolPrincipalUserID: req.SchoolProfileSchoolPrincipalUserID,
		SchoolProfileSchoolEmail:           req.SchoolProfileSchoolEmail,
		SchoolProfileSchoolAddress:         req.SchoolProfileSchoolAddress,
		SchoolProfileSchoolStudentCapacity: req.SchoolProfileSchoolStudentCapacity,
	}
	// default boolean (false) hanya di-set kalau req menyediakan value
	if req.SchoolProfileSchoolIsBoarding != nil {
		model.SchoolProfileSchoolIsBoarding = *req.SchoolProfileSchoolIsBoarding
	}
	return model
}

// Terapkan PATCH ke model existing (hanya field != nil yang diubah)
func ApplyPatchToModel(p *m.SchoolProfileModel, req *SchoolProfileUpdateRequest) {
	if req.SchoolProfileDescription != nil {
		p.SchoolProfileDescription = req.SchoolProfileDescription
	}
	if req.SchoolProfileFoundedYear != nil {
		p.SchoolProfileFoundedYear = req.SchoolProfileFoundedYear
	}

	if req.SchoolProfileAddress != nil {
		p.SchoolProfileAddress = req.SchoolProfileAddress
	}
	if req.SchoolProfileContactPhone != nil {
		p.SchoolProfileContactPhone = req.SchoolProfileContactPhone
	}
	if req.SchoolProfileContactEmail != nil {
		p.SchoolProfileContactEmail = req.SchoolProfileContactEmail
	}

	if req.SchoolProfileGoogleMapsURL != nil {
		p.SchoolProfileGoogleMapsURL = req.SchoolProfileGoogleMapsURL
	}
	if req.SchoolProfileInstagramURL != nil {
		p.SchoolProfileInstagramURL = req.SchoolProfileInstagramURL
	}
	if req.SchoolProfileWhatsappURL != nil {
		p.SchoolProfileWhatsappURL = req.SchoolProfileWhatsappURL
	}
	if req.SchoolProfileYoutubeURL != nil {
		p.SchoolProfileYoutubeURL = req.SchoolProfileYoutubeURL
	}
	if req.SchoolProfileFacebookURL != nil {
		p.SchoolProfileFacebookURL = req.SchoolProfileFacebookURL
	}
	if req.SchoolProfileTiktokURL != nil {
		p.SchoolProfileTiktokURL = req.SchoolProfileTiktokURL
	}
	if req.SchoolProfileWhatsappGroupIkhwanURL != nil {
		p.SchoolProfileWhatsappGroupIkhwanURL = req.SchoolProfileWhatsappGroupIkhwanURL
	}
	if req.SchoolProfileWhatsappGroupAkhwatURL != nil {
		p.SchoolProfileWhatsappGroupAkhwatURL = req.SchoolProfileWhatsappGroupAkhwatURL
	}
	if req.SchoolProfileWebsiteURL != nil {
		p.SchoolProfileWebsiteURL = req.SchoolProfileWebsiteURL
	}

	if req.SchoolProfileLatitude != nil {
		p.SchoolProfileLatitude = req.SchoolProfileLatitude
	}
	if req.SchoolProfileLongitude != nil {
		p.SchoolProfileLongitude = req.SchoolProfileLongitude
	}

	if req.SchoolProfileSchoolNPSN != nil {
		p.SchoolProfileSchoolNPSN = req.SchoolProfileSchoolNPSN
	}
	if req.SchoolProfileSchoolNSS != nil {
		p.SchoolProfileSchoolNSS = req.SchoolProfileSchoolNSS
	}
	if req.SchoolProfileSchoolAccreditation != nil {
		p.SchoolProfileSchoolAccreditation = req.SchoolProfileSchoolAccreditation
	}
	if req.SchoolProfileSchoolPrincipalUserID != nil {
		p.SchoolProfileSchoolPrincipalUserID = req.SchoolProfileSchoolPrincipalUserID
	}
	if req.SchoolProfileSchoolEmail != nil {
		p.SchoolProfileSchoolEmail = req.SchoolProfileSchoolEmail
	}
	if req.SchoolProfileSchoolAddress != nil {
		p.SchoolProfileSchoolAddress = req.SchoolProfileSchoolAddress
	}
	if req.SchoolProfileSchoolStudentCapacity != nil {
		p.SchoolProfileSchoolStudentCapacity = req.SchoolProfileSchoolStudentCapacity
	}
	if req.SchoolProfileSchoolIsBoarding != nil {
		p.SchoolProfileSchoolIsBoarding = *req.SchoolProfileSchoolIsBoarding
	}
}
