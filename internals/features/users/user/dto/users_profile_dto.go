package dto

import (
	"errors"
	"strings"
	"time"

	profilemodel "masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

/* ===========================
   Response DTO (semua eksplisit)
   =========================== */

type UsersProfileDTO struct {
	UsersProfileID             uuid.UUID  `json:"users_profile_id"`
	UsersProfileUserID         uuid.UUID  `json:"users_profile_user_id"`

	// Identitas dasar
	UsersProfileDonationName   string     `json:"users_profile_donation_name"`
	UsersProfileDateOfBirth    *time.Time `json:"users_profile_date_of_birth,omitempty"`
	UserProfilePlaceOfBirth    *string    `json:"user_profile_place_of_birth,omitempty"`
	UsersProfileGender         *string    `json:"users_profile_gender,omitempty"` // "male"/"female"
	UsersProfileLocation       *string    `json:"users_profile_location,omitempty"`
	UsersProfileCity           *string    `json:"users_profile_city,omitempty"`
	UsersProfilePhoneNumber    *string    `json:"users_profile_phone_number,omitempty"`
	UsersProfileBio            *string    `json:"users_profile_bio,omitempty"`

	// Konten panjang & riwayat
	UsersProfileBiographyLong  *string    `json:"users_profile_biography_long,omitempty"`
	UsersProfileExperience     *string    `json:"users_profile_experience,omitempty"`
	UsersProfileCertifications *string    `json:"users_profile_certifications,omitempty"`

	// Sosial media utama (sesuai tabel)
	UsersProfileInstagramURL   *string    `json:"users_profile_instagram_url,omitempty"`
	UsersProfileWhatsappURL    *string    `json:"users_profile_whatsapp_url,omitempty"`
	UsersProfileLinkedinURL    *string    `json:"users_profile_linkedin_url,omitempty"`
	UsersProfileGithubURL      *string    `json:"users_profile_github_url,omitempty"`
	UsersProfileYoutubeURL     *string    `json:"users_profile_youtube_url,omitempty"` // ← NEW

	// Privasi & verifikasi profil (bukan verifikasi dokumen)
	UsersProfileIsPublicProfile bool       `json:"users_profile_is_public_profile"`
	UsersProfileIsVerified      bool       `json:"users_profile_is_verified"`
	UsersProfileVerifiedAt      *time.Time `json:"users_profile_verified_at,omitempty"`
	UsersProfileVerifiedBy      *uuid.UUID `json:"users_profile_verified_by,omitempty"`

	// Pendidikan & pekerjaan
	UsersProfileEducation *string  `json:"users_profile_education,omitempty"`
	UsersProfileCompany   *string  `json:"users_profile_company,omitempty"`
	UsersProfilePosition  *string  `json:"users_profile_position,omitempty"`

	// Arrays
	UsersProfileInterests []string `json:"users_profile_interests"`
	UsersProfileSkills    []string `json:"users_profile_skills"`

	// Audit
	UsersProfileCreatedAt time.Time  `json:"users_profile_created_at"`
	UsersProfileUpdatedAt time.Time  `json:"users_profile_updated_at"`
	UsersProfileDeletedAt *time.Time `json:"users_profile_deleted_at,omitempty"`
}

func ToUsersProfileDTO(m profilemodel.UsersProfileModel) UsersProfileDTO {
	var genderStr *string
	if m.UsersProfileGender != nil {
		g := string(*m.UsersProfileGender)
		genderStr = &g
	}

	var deletedAtPtr *time.Time
	if m.UsersProfileDeletedAt.Valid {
		t := m.UsersProfileDeletedAt.Time
		deletedAtPtr = &t
	}

	// Donation name di model pointer → defaultkan ke "" di response
	donationName := ""
	if m.UsersProfileDonationName != nil {
		donationName = *m.UsersProfileDonationName
	}

	return UsersProfileDTO{
		UsersProfileID:              m.UsersProfileID,
		UsersProfileUserID:          m.UsersProfileUserID,

		UsersProfileDonationName:    donationName,
		UsersProfileDateOfBirth:     m.UsersProfileDateOfBirth,
		UserProfilePlaceOfBirth:     m.UserProfilePlaceOfBirth,
		UsersProfileGender:          genderStr,
		UsersProfileLocation:        m.UsersProfileLocation,
		UsersProfileCity:            m.UsersProfileCity,
		UsersProfilePhoneNumber:     m.UsersProfilePhoneNumber,
		UsersProfileBio:             m.UsersProfileBio,

		UsersProfileBiographyLong:   m.UsersProfileBiographyLong,
		UsersProfileExperience:      m.UsersProfileExperience,
		UsersProfileCertifications:  m.UsersProfileCertifications,

		UsersProfileInstagramURL:    m.UsersProfileInstagramURL,
		UsersProfileWhatsappURL:     m.UsersProfileWhatsappURL,
		UsersProfileLinkedinURL:     m.UsersProfileLinkedinURL,
		UsersProfileGithubURL:       m.UsersProfileGithubURL,
		UsersProfileYoutubeURL:      m.UsersProfileYoutubeURL, // ← NEW

		UsersProfileIsPublicProfile: m.UsersProfileIsPublicProfile,
		UsersProfileIsVerified:      m.UsersProfileIsVerified,
		UsersProfileVerifiedAt:      m.UsersProfileVerifiedAt,
		UsersProfileVerifiedBy:      m.UsersProfileVerifiedBy,

		UsersProfileEducation: m.UsersProfileEducation,
		UsersProfileCompany:   m.UsersProfileCompany,
		UsersProfilePosition:  m.UsersProfilePosition,

		UsersProfileInterests: []string(m.UsersProfileInterests),
		UsersProfileSkills:    []string(m.UsersProfileSkills),

		UsersProfileCreatedAt: m.UsersProfileCreatedAt,
		UsersProfileUpdatedAt: m.UsersProfileUpdatedAt,
		UsersProfileDeletedAt: deletedAtPtr,
	}
}

func ToUsersProfileDTOs(list []profilemodel.UsersProfileModel) []UsersProfileDTO {
	out := make([]UsersProfileDTO, 0, len(list))
	for _, it := range list {
		out = append(out, ToUsersProfileDTO(it))
	}
	return out
}

/* ===========================
   Request DTOs
   =========================== */

// Create
type CreateUsersProfileRequest struct {
	UsersProfileDonationName  string   `json:"users_profile_donation_name" validate:"omitempty,max=50"`
	UsersProfileDateOfBirth   *string  `json:"users_profile_date_of_birth,omitempty" validate:"omitempty"` // "2006-01-02"
	UserProfilePlaceOfBirth   *string  `json:"user_profile_place_of_birth,omitempty" validate:"omitempty,max=100"`
	UsersProfileGender        *string  `json:"users_profile_gender,omitempty" validate:"omitempty,oneof=male female"`
	UsersProfileLocation      *string  `json:"users_profile_location,omitempty" validate:"omitempty,max=100"`
	UsersProfileCity          *string  `json:"users_profile_city,omitempty" validate:"omitempty,max=100"`
	UsersProfilePhoneNumber   *string  `json:"users_profile_phone_number,omitempty" validate:"omitempty,max=20"`
	UsersProfileBio           *string  `json:"users_profile_bio,omitempty" validate:"omitempty,max=300"`

	UsersProfileBiographyLong  *string `json:"users_profile_biography_long,omitempty" validate:"omitempty"`
	UsersProfileExperience     *string `json:"users_profile_experience,omitempty" validate:"omitempty"`
	UsersProfileCertifications *string `json:"users_profile_certifications,omitempty" validate:"omitempty"`

	UsersProfileInstagramURL *string `json:"users_profile_instagram_url,omitempty" validate:"omitempty,url"`
	UsersProfileWhatsappURL  *string `json:"users_profile_whatsapp_url,omitempty" validate:"omitempty,url"`
	UsersProfileLinkedinURL  *string `json:"users_profile_linkedin_url,omitempty" validate:"omitempty,url"`
	UsersProfileGithubURL    *string `json:"users_profile_github_url,omitempty" validate:"omitempty,url"`
	UsersProfileYoutubeURL   *string `json:"users_profile_youtube_url,omitempty" validate:"omitempty,url"` // ← NEW

	UsersProfileIsPublicProfile *bool      `json:"users_profile_is_public_profile,omitempty" validate:"omitempty"`
	UsersProfileIsVerified      *bool      `json:"users_profile_is_verified,omitempty" validate:"omitempty"`
	UsersProfileVerifiedAt      *string    `json:"users_profile_verified_at,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	UsersProfileVerifiedBy      *uuid.UUID `json:"users_profile_verified_by,omitempty" validate:"omitempty"`

	UsersProfileEducation *string  `json:"users_profile_education,omitempty" validate:"omitempty"`
	UsersProfileCompany   *string  `json:"users_profile_company,omitempty" validate:"omitempty"`
	UsersProfilePosition  *string  `json:"users_profile_position,omitempty" validate:"omitempty"`

	UsersProfileInterests []string `json:"users_profile_interests,omitempty" validate:"omitempty,dive,max=100"`
	UsersProfileSkills    []string `json:"users_profile_skills,omitempty" validate:"omitempty,dive,max=100"`
}

// Update (PATCH)
type UpdateUsersProfileRequest struct {
	UsersProfileDonationName  *string `json:"users_profile_donation_name" validate:"omitempty,max=50"`
	UsersProfileDateOfBirth   *string `json:"users_profile_date_of_birth" validate:"omitempty"` // "2006-01-02"
	UserProfilePlaceOfBirth   *string `json:"user_profile_place_of_birth" validate:"omitempty,max=100"`
	UsersProfileGender        *string `json:"users_profile_gender" validate:"omitempty,oneof=male female"`
	UsersProfileLocation      *string `json:"users_profile_location" validate:"omitempty,max=100"`
	UsersProfileCity          *string `json:"users_profile_city" validate:"omitempty,max=100"`
	UsersProfilePhoneNumber   *string `json:"users_profile_phone_number" validate:"omitempty,max=20"`
	UsersProfileBio           *string `json:"users_profile_bio" validate:"omitempty,max=300"`

	UsersProfileBiographyLong  *string `json:"users_profile_biography_long" validate:"omitempty"`
	UsersProfileExperience     *string `json:"users_profile_experience" validate:"omitempty"`
	UsersProfileCertifications *string `json:"users_profile_certifications" validate:"omitempty"`

	UsersProfileInstagramURL *string `json:"users_profile_instagram_url" validate:"omitempty,url"`
	UsersProfileWhatsappURL  *string `json:"users_profile_whatsapp_url" validate:"omitempty,url"`
	UsersProfileLinkedinURL  *string `json:"users_profile_linkedin_url" validate:"omitempty,url"`
	UsersProfileGithubURL    *string `json:"users_profile_github_url" validate:"omitempty,url"`
	UsersProfileYoutubeURL   *string `json:"users_profile_youtube_url" validate:"omitempty,url"` // ← NEW

	UsersProfileIsPublicProfile *bool      `json:"users_profile_is_public_profile" validate:"omitempty"`
	UsersProfileIsVerified      *bool      `json:"users_profile_is_verified" validate:"omitempty"`
	UsersProfileVerifiedAt      *string    `json:"users_profile_verified_at" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	UsersProfileVerifiedBy      *uuid.UUID `json:"users_profile_verified_by" validate:"omitempty"`

	UsersProfileEducation *string  `json:"users_profile_education" validate:"omitempty"`
	UsersProfileCompany   *string  `json:"users_profile_company" validate:"omitempty"`
	UsersProfilePosition  *string  `json:"users_profile_position" validate:"omitempty"`

	UsersProfileInterests []string `json:"users_profile_interests" validate:"omitempty,dive,max=100"`
	UsersProfileSkills    []string `json:"users_profile_skills" validate:"omitempty,dive,max=100"`
}

/* ===========================
   Converters / Appliers
   =========================== */

func (r CreateUsersProfileRequest) ToModel(userID uuid.UUID) profilemodel.UsersProfileModel {
	m := profilemodel.UsersProfileModel{
		UsersProfileUserID:       userID,
		UsersProfileDonationName: stringsPtrOrNil(strings.TrimSpace(r.UsersProfileDonationName)),

		UsersProfileLocation:    trimPtr(r.UsersProfileLocation),
		UsersProfileCity:        trimPtr(r.UsersProfileCity),
		UsersProfilePhoneNumber: trimPtr(r.UsersProfilePhoneNumber),
		UsersProfileBio:         trimPtr(r.UsersProfileBio),

		UsersProfileBiographyLong:  trimPtr(r.UsersProfileBiographyLong),
		UsersProfileExperience:     trimPtr(r.UsersProfileExperience),
		UsersProfileCertifications: trimPtr(r.UsersProfileCertifications),

		UsersProfileInstagramURL: trimPtr(r.UsersProfileInstagramURL),
		UsersProfileWhatsappURL:  trimPtr(r.UsersProfileWhatsappURL),
		UsersProfileLinkedinURL:  trimPtr(r.UsersProfileLinkedinURL),
		UsersProfileGithubURL:    trimPtr(r.UsersProfileGithubURL),
		UsersProfileYoutubeURL:   trimPtr(r.UsersProfileYoutubeURL), // ← NEW

		UsersProfileEducation: trimPtr(r.UsersProfileEducation),
		UsersProfileCompany:   trimPtr(r.UsersProfileCompany),
		UsersProfilePosition:  trimPtr(r.UsersProfilePosition),

		UsersProfileInterests: pq.StringArray(compactStrings(r.UsersProfileInterests)),
		UsersProfileSkills:    pq.StringArray(compactStrings(r.UsersProfileSkills)),
	}

	// place_of_birth
	m.UserProfilePlaceOfBirth = trimPtr(r.UserProfilePlaceOfBirth)

	// flags
	if r.UsersProfileIsPublicProfile != nil {
		m.UsersProfileIsPublicProfile = *r.UsersProfileIsPublicProfile
	}
	if r.UsersProfileIsVerified != nil {
		m.UsersProfileIsVerified = *r.UsersProfileIsVerified
	}

	// verified_at / verified_by
	if r.UsersProfileVerifiedAt != nil && strings.TrimSpace(*r.UsersProfileVerifiedAt) != "" {
		if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(*r.UsersProfileVerifiedAt)); err == nil {
			m.UsersProfileVerifiedAt = &ts
		}
	}
	if r.UsersProfileVerifiedBy != nil {
		m.UsersProfileVerifiedBy = r.UsersProfileVerifiedBy
	}

	// date_of_birth
	if r.UsersProfileDateOfBirth != nil && strings.TrimSpace(*r.UsersProfileDateOfBirth) != "" {
		if dob, err := time.Parse("2006-01-02", strings.TrimSpace(*r.UsersProfileDateOfBirth)); err == nil {
			m.UsersProfileDateOfBirth = &dob
		}
	}

	// gender
	if g := parseGenderPtr(r.UsersProfileGender); g != nil {
		m.UsersProfileGender = g
	}

	return m
}

// ToUpdateMap: selalu pakai nama kolom DB sebagai key
func (in *UpdateUsersProfileRequest) ToUpdateMap() (map[string]interface{}, error) {
	m := map[string]interface{}{}

	setStr := func(key string, v *string) {
		if v != nil {
			m[key] = strings.TrimSpace(*v)
		}
	}
	setBool := func(key string, v *bool) {
		if v != nil {
			m[key] = *v
		}
	}

	// Basic
	setStr("users_profile_donation_name", in.UsersProfileDonationName)
	setStr("user_profile_place_of_birth", in.UserProfilePlaceOfBirth) // per DDL, tanpa 's'
	setStr("users_profile_location", in.UsersProfileLocation)
	setStr("users_profile_city", in.UsersProfileCity)
	setStr("users_profile_phone_number", in.UsersProfilePhoneNumber)
	setStr("users_profile_bio", in.UsersProfileBio)

	// Longs
	setStr("users_profile_biography_long", in.UsersProfileBiographyLong)
	setStr("users_profile_experience", in.UsersProfileExperience)
	setStr("users_profile_certifications", in.UsersProfileCertifications)

	// Socials (yang ada di tabel)
	setStr("users_profile_instagram_url", in.UsersProfileInstagramURL)
	setStr("users_profile_whatsapp_url", in.UsersProfileWhatsappURL)
	setStr("users_profile_linkedin_url", in.UsersProfileLinkedinURL)
	setStr("users_profile_github_url", in.UsersProfileGithubURL)
	setStr("users_profile_youtube_url", in.UsersProfileYoutubeURL) // ← NEW

	// Privacy & verification
	setBool("users_profile_is_public_profile", in.UsersProfileIsPublicProfile)
	setBool("users_profile_is_verified", in.UsersProfileIsVerified)

	if in.UsersProfileVerifiedAt != nil {
		if strings.TrimSpace(*in.UsersProfileVerifiedAt) == "" {
			m["users_profile_verified_at"] = nil
		} else {
			ts, err := time.Parse(time.RFC3339, strings.TrimSpace(*in.UsersProfileVerifiedAt))
			if err != nil {
				return nil, errors.New("users_profile_verified_at must be RFC3339 timestamp")
			}
			m["users_profile_verified_at"] = ts
		}
	}
	if in.UsersProfileVerifiedBy != nil {
		m["users_profile_verified_by"] = *in.UsersProfileVerifiedBy
	}

	// Education & job
	setStr("users_profile_education", in.UsersProfileEducation)
	setStr("users_profile_company", in.UsersProfileCompany)
	setStr("users_profile_position", in.UsersProfilePosition)

	// Arrays
	if in.UsersProfileInterests != nil {
		m["users_profile_interests"] = pq.StringArray(compactStrings(in.UsersProfileInterests))
	}
	if in.UsersProfileSkills != nil {
		m["users_profile_skills"] = pq.StringArray(compactStrings(in.UsersProfileSkills))
	}

	// Date of birth
	if in.UsersProfileDateOfBirth != nil {
		if strings.TrimSpace(*in.UsersProfileDateOfBirth) == "" {
			m["users_profile_date_of_birth"] = nil
		} else {
			dob, err := time.Parse("2006-01-02", strings.TrimSpace(*in.UsersProfileDateOfBirth))
			if err != nil {
				return nil, errors.New("users_profile_date_of_birth must be YYYY-MM-DD")
			}
			m["users_profile_date_of_birth"] = dob
		}
	}

	// Gender
	if in.UsersProfileGender != nil {
		g := strings.ToLower(strings.TrimSpace(*in.UsersProfileGender))
		if g != string(profilemodel.Male) && g != string(profilemodel.Female) {
			return nil, errors.New("users_profile_gender must be 'male' or 'female'")
		}
		m["users_profile_gender"] = g
	}

	return m, nil
}

/* ===========================
   Helpers
   =========================== */

func parseGenderPtr(s *string) *profilemodel.Gender {
	if s == nil {
		return nil
	}
	val := strings.ToLower(strings.TrimSpace(*s))
	switch val {
	case string(profilemodel.Male), string(profilemodel.Female):
		g := profilemodel.Gender(val)
		return &g
	default:
		return nil
	}
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func stringsPtrOrNil(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}

func compactStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		t := strings.TrimSpace(s)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
