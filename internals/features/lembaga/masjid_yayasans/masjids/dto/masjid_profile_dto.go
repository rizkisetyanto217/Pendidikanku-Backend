package dto

import (
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"
)

/* =======================================================
   REQUEST DTOs
   ======================================================= */

// Create: minimal wajib masjid_id; field lain opsional
type MasjidProfileCreateRequest struct {
	MasjidProfileMasjidID string `json:"masjid_profile_masjid_id" validate:"required,uuid4"`

	MasjidProfileDescription *string `json:"masjid_profile_description,omitempty"`
	MasjidProfileFoundedYear *int    `json:"masjid_profile_founded_year,omitempty"`

	// Alamat & kontak
	MasjidProfileAddress      *string `json:"masjid_profile_address,omitempty"`
	MasjidProfileContactPhone *string `json:"masjid_profile_contact_phone,omitempty"`
	MasjidProfileContactEmail *string `json:"masjid_profile_contact_email,omitempty"`

	// Sosial/link publik
	MasjidProfileGoogleMapsURL          *string `json:"masjid_profile_google_maps_url,omitempty"`
	MasjidProfileInstagramURL           *string `json:"masjid_profile_instagram_url,omitempty"`
	MasjidProfileWhatsappURL            *string `json:"masjid_profile_whatsapp_url,omitempty"`
	MasjidProfileYoutubeURL             *string `json:"masjid_profile_youtube_url,omitempty"`
	MasjidProfileFacebookURL            *string `json:"masjid_profile_facebook_url,omitempty"`
	MasjidProfileTiktokURL              *string `json:"masjid_profile_tiktok_url,omitempty"`
	MasjidProfileWhatsappGroupIkhwanURL *string `json:"masjid_profile_whatsapp_group_ikhwan_url,omitempty"`
	MasjidProfileWhatsappGroupAkhwatURL *string `json:"masjid_profile_whatsapp_group_akhwat_url,omitempty"`
	MasjidProfileWebsiteURL             *string `json:"masjid_profile_website_url,omitempty"`

	// Koordinat
	MasjidProfileLatitude  *float64 `json:"masjid_profile_latitude,omitempty"`
	MasjidProfileLongitude *float64 `json:"masjid_profile_longitude,omitempty"`

	// Profil sekolah (opsional)
	MasjidProfileSchoolNPSN            *string    `json:"masjid_profile_school_npsn,omitempty"`
	MasjidProfileSchoolNSS             *string    `json:"masjid_profile_school_nss,omitempty"`
	MasjidProfileSchoolAccreditation   *string    `json:"masjid_profile_school_accreditation,omitempty"` // A/B/C/Ungraded/-
	MasjidProfileSchoolPrincipalUserID *uuid.UUID `json:"masjid_profile_school_principal_user_id,omitempty"`
	MasjidProfileSchoolEmail           *string    `json:"masjid_profile_school_email,omitempty"`
	MasjidProfileSchoolAddress         *string    `json:"masjid_profile_school_address,omitempty"`
	MasjidProfileSchoolStudentCapacity *int       `json:"masjid_profile_school_student_capacity,omitempty"`
	MasjidProfileSchoolIsBoarding      *bool      `json:"masjid_profile_school_is_boarding,omitempty"`
}

// Update (PATCH): semua opsional; hanya yang != nil yang di-apply
type MasjidProfileUpdateRequest struct {
	MasjidProfileDescription *string `json:"masjid_profile_description,omitempty"`
	MasjidProfileFoundedYear *int    `json:"masjid_profile_founded_year,omitempty"`

	// Alamat & kontak
	MasjidProfileAddress      *string `json:"masjid_profile_address,omitempty"`
	MasjidProfileContactPhone *string `json:"masjid_profile_contact_phone,omitempty"`
	MasjidProfileContactEmail *string `json:"masjid_profile_contact_email,omitempty"`

	// Sosial/link publik
	MasjidProfileGoogleMapsURL          *string `json:"masjid_profile_google_maps_url,omitempty"`
	MasjidProfileInstagramURL           *string `json:"masjid_profile_instagram_url,omitempty"`
	MasjidProfileWhatsappURL            *string `json:"masjid_profile_whatsapp_url,omitempty"`
	MasjidProfileYoutubeURL             *string `json:"masjid_profile_youtube_url,omitempty"`
	MasjidProfileFacebookURL            *string `json:"masjid_profile_facebook_url,omitempty"`
	MasjidProfileTiktokURL              *string `json:"masjid_profile_tiktok_url,omitempty"`
	MasjidProfileWhatsappGroupIkhwanURL *string `json:"masjid_profile_whatsapp_group_ikhwan_url,omitempty"`
	MasjidProfileWhatsappGroupAkhwatURL *string `json:"masjid_profile_whatsapp_group_akhwat_url,omitempty"`
	MasjidProfileWebsiteURL             *string `json:"masjid_profile_website_url,omitempty"`

	// Koordinat
	MasjidProfileLatitude  *float64 `json:"masjid_profile_latitude,omitempty"`
	MasjidProfileLongitude *float64 `json:"masjid_profile_longitude,omitempty"`

	// Profil sekolah
	MasjidProfileSchoolNPSN            *string    `json:"masjid_profile_school_npsn,omitempty"`
	MasjidProfileSchoolNSS             *string    `json:"masjid_profile_school_nss,omitempty"`
	MasjidProfileSchoolAccreditation   *string    `json:"masjid_profile_school_accreditation,omitempty"`
	MasjidProfileSchoolPrincipalUserID *uuid.UUID `json:"masjid_profile_school_principal_user_id,omitempty"`
	MasjidProfileSchoolEmail           *string    `json:"masjid_profile_school_email,omitempty"`
	MasjidProfileSchoolAddress         *string    `json:"masjid_profile_school_address,omitempty"`
	MasjidProfileSchoolStudentCapacity *int       `json:"masjid_profile_school_student_capacity,omitempty"`
	MasjidProfileSchoolIsBoarding      *bool      `json:"masjid_profile_school_is_boarding,omitempty"`
}

/* =======================================================
   RESPONSE DTO
   ======================================================= */

type MasjidProfileResponse struct {
	MasjidProfileID       string `json:"masjid_profile_id"`
	MasjidProfileMasjidID string `json:"masjid_profile_masjid_id"`

	MasjidProfileDescription *string `json:"masjid_profile_description,omitempty"`
	MasjidProfileFoundedYear *int    `json:"masjid_profile_founded_year,omitempty"`

	// Alamat & kontak
	MasjidProfileAddress      *string `json:"masjid_profile_address,omitempty"`
	MasjidProfileContactPhone *string `json:"masjid_profile_contact_phone,omitempty"`
	MasjidProfileContactEmail *string `json:"masjid_profile_contact_email,omitempty"`

	// Sosial/link publik
	MasjidProfileGoogleMapsURL          *string `json:"masjid_profile_google_maps_url,omitempty"`
	MasjidProfileInstagramURL           *string `json:"masjid_profile_instagram_url,omitempty"`
	MasjidProfileWhatsappURL            *string `json:"masjid_profile_whatsapp_url,omitempty"`
	MasjidProfileYoutubeURL             *string `json:"masjid_profile_youtube_url,omitempty"`
	MasjidProfileFacebookURL            *string `json:"masjid_profile_facebook_url,omitempty"`
	MasjidProfileTiktokURL              *string `json:"masjid_profile_tiktok_url,omitempty"`
	MasjidProfileWhatsappGroupIkhwanURL *string `json:"masjid_profile_whatsapp_group_ikhwan_url,omitempty"`
	MasjidProfileWhatsappGroupAkhwatURL *string `json:"masjid_profile_whatsapp_group_akhwat_url,omitempty"`
	MasjidProfileWebsiteURL             *string `json:"masjid_profile_website_url,omitempty"`

	// Koordinat
	MasjidProfileLatitude  *float64 `json:"masjid_profile_latitude,omitempty"`
	MasjidProfileLongitude *float64 `json:"masjid_profile_longitude,omitempty"`

	// Profil sekolah
	MasjidProfileSchoolNPSN            *string    `json:"masjid_profile_school_npsn,omitempty"`
	MasjidProfileSchoolNSS             *string    `json:"masjid_profile_school_nss,omitempty"`
	MasjidProfileSchoolAccreditation   *string    `json:"masjid_profile_school_accreditation,omitempty"`
	MasjidProfileSchoolPrincipalUserID *uuid.UUID `json:"masjid_profile_school_principal_user_id,omitempty"`
	MasjidProfileSchoolEmail           *string    `json:"masjid_profile_school_email,omitempty"`
	MasjidProfileSchoolAddress         *string    `json:"masjid_profile_school_address,omitempty"`
	MasjidProfileSchoolStudentCapacity *int       `json:"masjid_profile_school_student_capacity,omitempty"`
	MasjidProfileSchoolIsBoarding      bool       `json:"masjid_profile_school_is_boarding"`

	// Read-only timestamps
	MasjidProfileCreatedAt time.Time `json:"masjid_profile_created_at"`
	MasjidProfileUpdatedAt time.Time `json:"masjid_profile_updated_at"`
}

/* =======================================================
   CONVERTERS
   ======================================================= */

func FromModelMasjidProfile(p *m.MasjidProfileModel) MasjidProfileResponse {
	return MasjidProfileResponse{
		MasjidProfileID:       p.MasjidProfileID.String(),
		MasjidProfileMasjidID: p.MasjidProfileMasjidID.String(),

		MasjidProfileDescription: p.MasjidProfileDescription,
		MasjidProfileFoundedYear: p.MasjidProfileFoundedYear,

		// Alamat & kontak
		MasjidProfileAddress:      p.MasjidProfileAddress,
		MasjidProfileContactPhone: p.MasjidProfileContactPhone,
		MasjidProfileContactEmail: p.MasjidProfileContactEmail,

		// Sosial/link
		MasjidProfileGoogleMapsURL:          p.MasjidProfileGoogleMapsURL,
		MasjidProfileInstagramURL:           p.MasjidProfileInstagramURL,
		MasjidProfileWhatsappURL:            p.MasjidProfileWhatsappURL,
		MasjidProfileYoutubeURL:             p.MasjidProfileYoutubeURL,
		MasjidProfileFacebookURL:            p.MasjidProfileFacebookURL,
		MasjidProfileTiktokURL:              p.MasjidProfileTiktokURL,
		MasjidProfileWhatsappGroupIkhwanURL: p.MasjidProfileWhatsappGroupIkhwanURL,
		MasjidProfileWhatsappGroupAkhwatURL: p.MasjidProfileWhatsappGroupAkhwatURL,
		MasjidProfileWebsiteURL:             p.MasjidProfileWebsiteURL,

		// Koordinat
		MasjidProfileLatitude:  p.MasjidProfileLatitude,
		MasjidProfileLongitude: p.MasjidProfileLongitude,

		// Sekolah
		MasjidProfileSchoolNPSN:            p.MasjidProfileSchoolNPSN,
		MasjidProfileSchoolNSS:             p.MasjidProfileSchoolNSS,
		MasjidProfileSchoolAccreditation:   p.MasjidProfileSchoolAccreditation,
		MasjidProfileSchoolPrincipalUserID: p.MasjidProfileSchoolPrincipalUserID,
		MasjidProfileSchoolEmail:           p.MasjidProfileSchoolEmail,
		MasjidProfileSchoolAddress:         p.MasjidProfileSchoolAddress,
		MasjidProfileSchoolStudentCapacity: p.MasjidProfileSchoolStudentCapacity,
		MasjidProfileSchoolIsBoarding:      p.MasjidProfileSchoolIsBoarding,

		MasjidProfileCreatedAt: p.MasjidProfileCreatedAt,
		MasjidProfileUpdatedAt: p.MasjidProfileUpdatedAt,
	}
}

func ToModelMasjidProfileCreate(req *MasjidProfileCreateRequest) *m.MasjidProfileModel {
	masjidID, _ := uuid.Parse(req.MasjidProfileMasjidID)
	model := &m.MasjidProfileModel{
		MasjidProfileMasjidID: masjidID,

		MasjidProfileDescription: req.MasjidProfileDescription,
		MasjidProfileFoundedYear: req.MasjidProfileFoundedYear,

		MasjidProfileAddress:      req.MasjidProfileAddress,
		MasjidProfileContactPhone: req.MasjidProfileContactPhone,
		MasjidProfileContactEmail: req.MasjidProfileContactEmail,

		MasjidProfileGoogleMapsURL:          req.MasjidProfileGoogleMapsURL,
		MasjidProfileInstagramURL:           req.MasjidProfileInstagramURL,
		MasjidProfileWhatsappURL:            req.MasjidProfileWhatsappURL,
		MasjidProfileYoutubeURL:             req.MasjidProfileYoutubeURL,
		MasjidProfileFacebookURL:            req.MasjidProfileFacebookURL,
		MasjidProfileTiktokURL:              req.MasjidProfileTiktokURL,
		MasjidProfileWhatsappGroupIkhwanURL: req.MasjidProfileWhatsappGroupIkhwanURL,
		MasjidProfileWhatsappGroupAkhwatURL: req.MasjidProfileWhatsappGroupAkhwatURL,
		MasjidProfileWebsiteURL:             req.MasjidProfileWebsiteURL,

		MasjidProfileLatitude:  req.MasjidProfileLatitude,
		MasjidProfileLongitude: req.MasjidProfileLongitude,

		MasjidProfileSchoolNPSN:            req.MasjidProfileSchoolNPSN,
		MasjidProfileSchoolNSS:             req.MasjidProfileSchoolNSS,
		MasjidProfileSchoolAccreditation:   req.MasjidProfileSchoolAccreditation,
		MasjidProfileSchoolPrincipalUserID: req.MasjidProfileSchoolPrincipalUserID,
		MasjidProfileSchoolEmail:           req.MasjidProfileSchoolEmail,
		MasjidProfileSchoolAddress:         req.MasjidProfileSchoolAddress,
		MasjidProfileSchoolStudentCapacity: req.MasjidProfileSchoolStudentCapacity,
	}
	// default boolean (false) hanya di-set kalau req menyediakan value
	if req.MasjidProfileSchoolIsBoarding != nil {
		model.MasjidProfileSchoolIsBoarding = *req.MasjidProfileSchoolIsBoarding
	}
	return model
}

// Terapkan PATCH ke model existing (hanya field != nil yang diubah)
func ApplyPatchToModel(p *m.MasjidProfileModel, req *MasjidProfileUpdateRequest) {
	if req.MasjidProfileDescription != nil {
		p.MasjidProfileDescription = req.MasjidProfileDescription
	}
	if req.MasjidProfileFoundedYear != nil {
		p.MasjidProfileFoundedYear = req.MasjidProfileFoundedYear
	}

	if req.MasjidProfileAddress != nil {
		p.MasjidProfileAddress = req.MasjidProfileAddress
	}
	if req.MasjidProfileContactPhone != nil {
		p.MasjidProfileContactPhone = req.MasjidProfileContactPhone
	}
	if req.MasjidProfileContactEmail != nil {
		p.MasjidProfileContactEmail = req.MasjidProfileContactEmail
	}

	if req.MasjidProfileGoogleMapsURL != nil {
		p.MasjidProfileGoogleMapsURL = req.MasjidProfileGoogleMapsURL
	}
	if req.MasjidProfileInstagramURL != nil {
		p.MasjidProfileInstagramURL = req.MasjidProfileInstagramURL
	}
	if req.MasjidProfileWhatsappURL != nil {
		p.MasjidProfileWhatsappURL = req.MasjidProfileWhatsappURL
	}
	if req.MasjidProfileYoutubeURL != nil {
		p.MasjidProfileYoutubeURL = req.MasjidProfileYoutubeURL
	}
	if req.MasjidProfileFacebookURL != nil {
		p.MasjidProfileFacebookURL = req.MasjidProfileFacebookURL
	}
	if req.MasjidProfileTiktokURL != nil {
		p.MasjidProfileTiktokURL = req.MasjidProfileTiktokURL
	}
	if req.MasjidProfileWhatsappGroupIkhwanURL != nil {
		p.MasjidProfileWhatsappGroupIkhwanURL = req.MasjidProfileWhatsappGroupIkhwanURL
	}
	if req.MasjidProfileWhatsappGroupAkhwatURL != nil {
		p.MasjidProfileWhatsappGroupAkhwatURL = req.MasjidProfileWhatsappGroupAkhwatURL
	}
	if req.MasjidProfileWebsiteURL != nil {
		p.MasjidProfileWebsiteURL = req.MasjidProfileWebsiteURL
	}

	if req.MasjidProfileLatitude != nil {
		p.MasjidProfileLatitude = req.MasjidProfileLatitude
	}
	if req.MasjidProfileLongitude != nil {
		p.MasjidProfileLongitude = req.MasjidProfileLongitude
	}

	if req.MasjidProfileSchoolNPSN != nil {
		p.MasjidProfileSchoolNPSN = req.MasjidProfileSchoolNPSN
	}
	if req.MasjidProfileSchoolNSS != nil {
		p.MasjidProfileSchoolNSS = req.MasjidProfileSchoolNSS
	}
	if req.MasjidProfileSchoolAccreditation != nil {
		p.MasjidProfileSchoolAccreditation = req.MasjidProfileSchoolAccreditation
	}
	if req.MasjidProfileSchoolPrincipalUserID != nil {
		p.MasjidProfileSchoolPrincipalUserID = req.MasjidProfileSchoolPrincipalUserID
	}
	if req.MasjidProfileSchoolEmail != nil {
		p.MasjidProfileSchoolEmail = req.MasjidProfileSchoolEmail
	}
	if req.MasjidProfileSchoolAddress != nil {
		p.MasjidProfileSchoolAddress = req.MasjidProfileSchoolAddress
	}
	if req.MasjidProfileSchoolStudentCapacity != nil {
		p.MasjidProfileSchoolStudentCapacity = req.MasjidProfileSchoolStudentCapacity
	}
	if req.MasjidProfileSchoolIsBoarding != nil {
		p.MasjidProfileSchoolIsBoarding = *req.MasjidProfileSchoolIsBoarding
	}
}
