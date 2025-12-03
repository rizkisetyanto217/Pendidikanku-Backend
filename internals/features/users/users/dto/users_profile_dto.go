package dto

import (
	"errors"
	"strings"
	"time"

	profilemodel "madinahsalam_backend/internals/features/users/users/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

/* ===========================
   Response DTO (JSON diselaraskan dgn model)
   =========================== */

type UsersProfileDTO struct {
	UserProfileID     uuid.UUID `json:"user_profile_id"`
	UserProfileUserID uuid.UUID `json:"user_profile_user_id"`

	// Snapshot dari users
	UserProfileFullNameCache *string `json:"user_profile_full_name_cache,omitempty"`

	// Identitas dasar
	UserProfileSlug         *string    `json:"user_profile_slug,omitempty"`
	UserProfileDonationName *string    `json:"user_profile_donation_name,omitempty"`
	UserProfileDateOfBirth  *time.Time `json:"user_profile_date_of_birth,omitempty"`
	UserProfilePlaceOfBirth *string    `json:"user_profile_place_of_birth,omitempty"`
	UserProfileGender       *string    `json:"user_profile_gender,omitempty"` // "male"/"female"
	UserProfileLocation     *string    `json:"user_profile_location,omitempty"`
	UserProfileCity         *string    `json:"user_profile_city,omitempty"`
	UserProfileBio          *string    `json:"user_profile_bio,omitempty"`

	// Konten panjang & riwayat
	UserProfileBiographyLong  *string `json:"user_profile_biography_long,omitempty"`
	UserProfileExperience     *string `json:"user_profile_experience,omitempty"`
	UserProfileCertifications *string `json:"user_profile_certifications,omitempty"`

	// Sosial media
	UserProfileInstagramURL     *string `json:"user_profile_instagram_url,omitempty"`
	UserProfileWhatsappURL      *string `json:"user_profile_whatsapp_url,omitempty"`
	UserProfileLinkedinURL      *string `json:"user_profile_linkedin_url,omitempty"`
	UserProfileGithubURL        *string `json:"user_profile_github_url,omitempty"`
	UserProfileYoutubeURL       *string `json:"user_profile_youtube_url,omitempty"`
	UserProfileTelegramUsername *string `json:"user_profile_telegram_username,omitempty"`

	// Orang tua / wali
	UserProfileParentName        *string `json:"user_profile_parent_name,omitempty"`
	UserProfileParentWhatsappURL *string `json:"user_profile_parent_whatsapp_url,omitempty"`

	// Avatar (single file, 2-slot + retensi 30 hari)
	UserProfileAvatarURL                *string    `json:"user_profile_avatar_url,omitempty"`
	UserProfileAvatarObjectKey          *string    `json:"user_profile_avatar_object_key,omitempty"`
	UserProfileAvatarURLOld             *string    `json:"user_profile_avatar_url_old,omitempty"`
	UserProfileAvatarObjectKeyOld       *string    `json:"user_profile_avatar_object_key_old,omitempty"`
	UserProfileAvatarDeletePendingUntil *time.Time `json:"user_profile_avatar_delete_pending_until,omitempty"`

	// Privasi & verifikasi profil (oleh platform)
	UserProfileIsPublicProfile bool       `json:"user_profile_is_public_profile"`
	UserProfileIsVerified      bool       `json:"user_profile_is_verified"`
	UserProfileVerifiedAt      *time.Time `json:"user_profile_verified_at,omitempty"`
	UserProfileVerifiedBy      *uuid.UUID `json:"user_profile_verified_by,omitempty"`

	// Pendidikan & pekerjaan
	UserProfileEducation *string `json:"user_profile_education,omitempty"`
	UserProfileCompany   *string `json:"user_profile_company,omitempty"`
	UserProfilePosition  *string `json:"user_profile_position,omitempty"`

	// Arrays
	UserProfileInterests []string `json:"user_profile_interests"`
	UserProfileSkills    []string `json:"user_profile_skills"`

	// Status kelengkapan profil
	UserProfileIsCompleted bool       `json:"user_profile_is_completed"`
	UserProfileCompletedAt *time.Time `json:"user_profile_completed_at,omitempty"`

	// Audit
	UserProfileCreatedAt time.Time  `json:"user_profile_created_at"`
	UserProfileUpdatedAt time.Time  `json:"user_profile_updated_at"`
	UserProfileDeletedAt *time.Time `json:"user_profile_deleted_at,omitempty"`
}

func ToUsersProfileDTO(m profilemodel.UserProfileModel) UsersProfileDTO {
	var genderStr *string
	if m.UserProfileGender != nil {
		g := string(*m.UserProfileGender)
		genderStr = &g
	}

	var deletedAtPtr *time.Time
	if m.UserProfileDeletedAt.Valid {
		t := m.UserProfileDeletedAt.Time
		deletedAtPtr = &t
	}

	return UsersProfileDTO{
		UserProfileID:                       m.UserProfileID,
		UserProfileUserID:                   m.UserProfileUserID,
		UserProfileFullNameCache:            m.UserProfileFullNameCache,
		UserProfileSlug:                     m.UserProfileSlug,
		UserProfileDonationName:             m.UserProfileDonationName,
		UserProfileDateOfBirth:              m.UserProfileDateOfBirth,
		UserProfilePlaceOfBirth:             m.UserProfilePlaceOfBirth,
		UserProfileGender:                   genderStr,
		UserProfileLocation:                 m.UserProfileLocation,
		UserProfileCity:                     m.UserProfileCity,
		UserProfileBio:                      m.UserProfileBio,
		UserProfileBiographyLong:            m.UserProfileBiographyLong,
		UserProfileExperience:               m.UserProfileExperience,
		UserProfileCertifications:           m.UserProfileCertifications,
		UserProfileInstagramURL:             m.UserProfileInstagramURL,
		UserProfileWhatsappURL:              m.UserProfileWhatsappURL,
		UserProfileLinkedinURL:              m.UserProfileLinkedinURL,
		UserProfileGithubURL:                m.UserProfileGithubURL,
		UserProfileYoutubeURL:               m.UserProfileYoutubeURL,
		UserProfileTelegramUsername:         m.UserProfileTelegramUsername,
		UserProfileParentName:               m.UserProfileParentName,
		UserProfileParentWhatsappURL:        m.UserProfileParentWhatsappURL,
		UserProfileAvatarURL:                m.UserProfileAvatarURL,
		UserProfileAvatarObjectKey:          m.UserProfileAvatarObjectKey,
		UserProfileAvatarURLOld:             m.UserProfileAvatarURLOld,
		UserProfileAvatarObjectKeyOld:       m.UserProfileAvatarObjectKeyOld,
		UserProfileAvatarDeletePendingUntil: m.UserProfileAvatarDeletePendingUntil,
		UserProfileIsPublicProfile:          m.UserProfileIsPublicProfile,
		UserProfileIsVerified:               m.UserProfileIsVerified,
		UserProfileVerifiedAt:               m.UserProfileVerifiedAt,
		UserProfileVerifiedBy:               m.UserProfileVerifiedBy,
		UserProfileEducation:                m.UserProfileEducation,
		UserProfileCompany:                  m.UserProfileCompany,
		UserProfilePosition:                 m.UserProfilePosition,
		UserProfileInterests:                []string(m.UserProfileInterests),
		UserProfileSkills:                   []string(m.UserProfileSkills),

		UserProfileIsCompleted: m.UserProfileIsCompleted,
		UserProfileCompletedAt: m.UserProfileCompletedAt,

		UserProfileCreatedAt: m.UserProfileCreatedAt,
		UserProfileUpdatedAt: m.UserProfileUpdatedAt,
		UserProfileDeletedAt: deletedAtPtr,
	}
}

func ToUsersProfileDTOs(list []profilemodel.UserProfileModel) []UsersProfileDTO {
	out := make([]UsersProfileDTO, 0, len(list))
	for _, it := range list {
		out = append(out, ToUsersProfileDTO(it))
	}
	return out
}

/* ===========================
   Request DTOs (JSON diselaraskan dgn model)
   =========================== */

type CreateUsersProfileRequest struct {
	UserProfileSlug *string `json:"user_profile_slug,omitempty" form:"user_profile_slug" validate:"omitempty,max=80"`

	UserProfileDonationName string  `json:"user_profile_donation_name" form:"user_profile_donation_name" validate:"omitempty,max=50"`
	UserProfileDateOfBirth  *string `json:"user_profile_date_of_birth,omitempty" form:"user_profile_date_of_birth" validate:"omitempty"` // "2006-01-02"
	UserProfilePlaceOfBirth *string `json:"user_profile_place_of_birth,omitempty" form:"user_profile_place_of_birth" validate:"omitempty,max=100"`
	UserProfileGender       *string `json:"user_profile_gender,omitempty" form:"user_profile_gender" validate:"omitempty,oneof=male female"`
	UserProfileLocation     *string `json:"user_profile_location,omitempty" form:"user_profile_location" validate:"omitempty,max=100"`
	UserProfileCity         *string `json:"user_profile_city,omitempty" form:"user_profile_city" validate:"omitempty,max=100"`
	UserProfileBio          *string `json:"user_profile_bio,omitempty" form:"user_profile_bio" validate:"omitempty,max=300"`

	UserProfileBiographyLong  *string `json:"user_profile_biography_long,omitempty" form:"user_profile_biography_long" validate:"omitempty"`
	UserProfileExperience     *string `json:"user_profile_experience,omitempty" form:"user_profile_experience" validate:"omitempty"`
	UserProfileCertifications *string `json:"user_profile_certifications,omitempty" form:"user_profile_certifications" validate:"omitempty"`

	UserProfileInstagramURL     *string `json:"user_profile_instagram_url,omitempty" form:"user_profile_instagram_url" validate:"omitempty,url"`
	UserProfileWhatsappURL      *string `json:"user_profile_whatsapp_url,omitempty" form:"user_profile_whatsapp_url" validate:"omitempty,url"`
	UserProfileLinkedinURL      *string `json:"user_profile_linkedin_url,omitempty" form:"user_profile_linkedin_url" validate:"omitempty,url"`
	UserProfileGithubURL        *string `json:"user_profile_github_url,omitempty" form:"user_profile_github_url" validate:"omitempty,url"`
	UserProfileYoutubeURL       *string `json:"user_profile_youtube_url,omitempty" form:"user_profile_youtube_url" validate:"omitempty,url"`
	UserProfileTelegramUsername *string `json:"user_profile_telegram_username,omitempty" form:"user_profile_telegram_username" validate:"omitempty,max=50"`

	// Orang tua / wali
	UserProfileParentName        *string `json:"user_profile_parent_name,omitempty" form:"user_profile_parent_name" validate:"omitempty,max=100"`
	UserProfileParentWhatsappURL *string `json:"user_profile_parent_whatsapp_url,omitempty" form:"user_profile_parent_whatsapp_url" validate:"omitempty,url"`

	UserProfileIsPublicProfile *bool      `json:"user_profile_is_public_profile,omitempty" form:"user_profile_is_public_profile" validate:"omitempty"`
	UserProfileIsVerified      *bool      `json:"user_profile_is_verified,omitempty" form:"user_profile_is_verified" validate:"omitempty"`
	UserProfileVerifiedAt      *string    `json:"user_profile_verified_at,omitempty" form:"user_profile_verified_at" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	UserProfileVerifiedBy      *uuid.UUID `json:"user_profile_verified_by,omitempty" form:"user_profile_verified_by" validate:"omitempty"`

	UserProfileEducation *string `json:"user_profile_education,omitempty" form:"user_profile_education" validate:"omitempty"`
	UserProfileCompany   *string `json:"user_profile_company,omitempty" form:"user_profile_company" validate:"omitempty"`
	UserProfilePosition  *string `json:"user_profile_position,omitempty" form:"user_profile_position" validate:"omitempty"`

	// NOTE: untuk multipart tanpa payload, kirim sebagai JSON string atau berulang (key[])
	UserProfileInterests []string `json:"user_profile_interests,omitempty" form:"user_profile_interests" validate:"omitempty,dive,max=100"`
	UserProfileSkills    []string `json:"user_profile_skills,omitempty" form:"user_profile_skills" validate:"omitempty,dive,max=100"`
}

type UpdateUsersProfileRequest struct {
	UserProfileSlug *string `json:"user_profile_slug" form:"user_profile_slug" validate:"omitempty,max=80"`

	UserProfileDonationName *string `json:"user_profile_donation_name" form:"user_profile_donation_name" validate:"omitempty,max=50"`
	UserProfileDateOfBirth  *string `json:"user_profile_date_of_birth" form:"user_profile_date_of_birth" validate:"omitempty"`
	UserProfilePlaceOfBirth *string `json:"user_profile_place_of_birth" form:"user_profile_place_of_birth" validate:"omitempty,max=100"`
	UserProfileGender       *string `json:"user_profile_gender" form:"user_profile_gender" validate:"omitempty,oneof=male female"`
	UserProfileLocation     *string `json:"user_profile_location" form:"user_profile_location" validate:"omitempty,max=100"`
	UserProfileCity         *string `json:"user_profile_city" form:"user_profile_city" validate:"omitempty,max=100"`
	UserProfileBio          *string `json:"user_profile_bio" form:"user_profile_bio" validate:"omitempty,max=300"`

	UserProfileBiographyLong  *string `json:"user_profile_biography_long" form:"user_profile_biography_long" validate:"omitempty"`
	UserProfileExperience     *string `json:"user_profile_experience" form:"user_profile_experience" validate:"omitempty"`
	UserProfileCertifications *string `json:"user_profile_certifications" form:"user_profile_certifications" validate:"omitempty"`

	UserProfileInstagramURL     *string `json:"user_profile_instagram_url" form:"user_profile_instagram_url" validate:"omitempty,url"`
	UserProfileWhatsappURL      *string `json:"user_profile_whatsapp_url" form:"user_profile_whatsapp_url" validate:"omitempty,url"`
	UserProfileLinkedinURL      *string `json:"user_profile_linkedin_url" form:"user_profile_linkedin_url" validate:"omitempty,url"`
	UserProfileGithubURL        *string `json:"user_profile_github_url" form:"user_profile_github_url" validate:"omitempty,url"`
	UserProfileYoutubeURL       *string `json:"user_profile_youtube_url" form:"user_profile_youtube_url" validate:"omitempty,url"`
	UserProfileTelegramUsername *string `json:"user_profile_telegram_username" form:"user_profile_telegram_username" validate:"omitempty,max=50"`

	// Orang tua / wali
	UserProfileParentName        *string `json:"user_profile_parent_name" form:"user_profile_parent_name" validate:"omitempty,max=100"`
	UserProfileParentWhatsappURL *string `json:"user_profile_parent_whatsapp_url" form:"user_profile_parent_whatsapp_url" validate:"omitempty,url"`

	UserProfileIsPublicProfile *bool      `json:"user_profile_is_public_profile" form:"user_profile_is_public_profile" validate:"omitempty"`
	UserProfileIsVerified      *bool      `json:"user_profile_is_verified" form:"user_profile_is_verified" validate:"omitempty"`
	UserProfileVerifiedAt      *string    `json:"user_profile_verified_at" form:"user_profile_verified_at" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	UserProfileVerifiedBy      *uuid.UUID `json:"user_profile_verified_by" form:"user_profile_verified_by" validate:"omitempty"`

	UserProfileEducation *string `json:"user_profile_education" form:"user_profile_education" validate:"omitempty"`
	UserProfileCompany   *string `json:"user_profile_company" form:"user_profile_company" validate:"omitempty"`
	UserProfilePosition  *string `json:"user_profile_position" form:"user_profile_position" validate:"omitempty"`

	// NOTE: untuk multipart tanpa payload, kirim sebagai JSON string atau berulang (key[])
	UserProfileInterests []string `json:"user_profile_interests" form:"user_profile_interests" validate:"omitempty,dive,max=100"`
	UserProfileSkills    []string `json:"user_profile_skills" form:"user_profile_skills" validate:"omitempty,dive,max=100"`
}

/* ===========================
   Converters / Appliers
   =========================== */

func (r CreateUsersProfileRequest) ToModel(userID uuid.UUID) profilemodel.UserProfileModel {
	m := profilemodel.UserProfileModel{
		UserProfileUserID:       userID,
		UserProfileDonationName: stringsPtrOrNil(strings.TrimSpace(r.UserProfileDonationName)),

		// dasar
		UserProfileSlug:     trimPtr(r.UserProfileSlug),
		UserProfileLocation: trimPtr(r.UserProfileLocation),
		UserProfileCity:     trimPtr(r.UserProfileCity),
		UserProfileBio:      trimPtr(r.UserProfileBio),

		// panjang
		UserProfileBiographyLong:  trimPtr(r.UserProfileBiographyLong),
		UserProfileExperience:     trimPtr(r.UserProfileExperience),
		UserProfileCertifications: trimPtr(r.UserProfileCertifications),

		// sosmed
		UserProfileInstagramURL:     trimPtr(r.UserProfileInstagramURL),
		UserProfileWhatsappURL:      trimPtr(r.UserProfileWhatsappURL),
		UserProfileLinkedinURL:      trimPtr(r.UserProfileLinkedinURL),
		UserProfileGithubURL:        trimPtr(r.UserProfileGithubURL),
		UserProfileYoutubeURL:       trimPtr(r.UserProfileYoutubeURL),
		UserProfileTelegramUsername: trimPtr(r.UserProfileTelegramUsername),

		// Orang tua / wali
		UserProfileParentName:        trimPtr(r.UserProfileParentName),
		UserProfileParentWhatsappURL: trimPtr(r.UserProfileParentWhatsappURL),

		// edu/job
		UserProfileEducation: trimPtr(r.UserProfileEducation),
		UserProfileCompany:   trimPtr(r.UserProfileCompany),
		UserProfilePosition:  trimPtr(r.UserProfilePosition),

		UserProfileInterests: pq.StringArray(CompactStrings(r.UserProfileInterests)),
		UserProfileSkills:    pq.StringArray(CompactStrings(r.UserProfileSkills)),
	}

	// place_of_birth
	m.UserProfilePlaceOfBirth = trimPtr(r.UserProfilePlaceOfBirth)

	// flags
	if r.UserProfileIsPublicProfile != nil {
		m.UserProfileIsPublicProfile = *r.UserProfileIsPublicProfile
	}
	if r.UserProfileIsVerified != nil {
		m.UserProfileIsVerified = *r.UserProfileIsVerified
	}

	// verified_at / verified_by
	if r.UserProfileVerifiedAt != nil && strings.TrimSpace(*r.UserProfileVerifiedAt) != "" {
		if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(*r.UserProfileVerifiedAt)); err == nil {
			m.UserProfileVerifiedAt = &ts
		}
	}
	if r.UserProfileVerifiedBy != nil {
		m.UserProfileVerifiedBy = r.UserProfileVerifiedBy
	}

	// date_of_birth
	if r.UserProfileDateOfBirth != nil && strings.TrimSpace(*r.UserProfileDateOfBirth) != "" {
		if dob, err := time.Parse("2006-01-02", strings.TrimSpace(*r.UserProfileDateOfBirth)); err == nil {
			m.UserProfileDateOfBirth = &dob
		}
	}

	// gender
	if g := parseGenderPtr(r.UserProfileGender); g != nil {
		m.UserProfileGender = g
	}

	return m
}

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
	setStr("user_profile_slug", in.UserProfileSlug)
	setStr("user_profile_donation_name", in.UserProfileDonationName)
	setStr("user_profile_place_of_birth", in.UserProfilePlaceOfBirth)
	setStr("user_profile_location", in.UserProfileLocation)
	setStr("user_profile_city", in.UserProfileCity)
	setStr("user_profile_bio", in.UserProfileBio)

	// Longs
	setStr("user_profile_biography_long", in.UserProfileBiographyLong)
	setStr("user_profile_experience", in.UserProfileExperience)
	setStr("user_profile_certifications", in.UserProfileCertifications)

	// Socials
	setStr("user_profile_instagram_url", in.UserProfileInstagramURL)
	setStr("user_profile_whatsapp_url", in.UserProfileWhatsappURL)
	setStr("user_profile_linkedin_url", in.UserProfileLinkedinURL)
	setStr("user_profile_github_url", in.UserProfileGithubURL)
	setStr("user_profile_youtube_url", in.UserProfileYoutubeURL)
	setStr("user_profile_telegram_username", in.UserProfileTelegramUsername)

	// Orang tua / wali
	setStr("user_profile_parent_name", in.UserProfileParentName)
	setStr("user_profile_parent_whatsapp_url", in.UserProfileParentWhatsappURL)

	// Privacy & verification
	setBool("user_profile_is_public_profile", in.UserProfileIsPublicProfile)
	setBool("user_profile_is_verified", in.UserProfileIsVerified)

	if in.UserProfileVerifiedAt != nil {
		if strings.TrimSpace(*in.UserProfileVerifiedAt) == "" {
			m["user_profile_verified_at"] = nil
		} else {
			ts, err := time.Parse(time.RFC3339, strings.TrimSpace(*in.UserProfileVerifiedAt))
			if err != nil {
				return nil, errors.New("user_profile_verified_at must be RFC3339 timestamp")
			}
			m["user_profile_verified_at"] = ts
		}
	}
	if in.UserProfileVerifiedBy != nil {
		m["user_profile_verified_by"] = *in.UserProfileVerifiedBy
	}

	// Education & job
	setStr("user_profile_education", in.UserProfileEducation)
	setStr("user_profile_company", in.UserProfileCompany)
	setStr("user_profile_position", in.UserProfilePosition)

	// Arrays
	if in.UserProfileInterests != nil {
		m["user_profile_interests"] = pq.StringArray(CompactStrings(in.UserProfileInterests))
	}
	if in.UserProfileSkills != nil {
		m["user_profile_skills"] = pq.StringArray(CompactStrings(in.UserProfileSkills))
	}

	// Date of birth
	if in.UserProfileDateOfBirth != nil {
		if strings.TrimSpace(*in.UserProfileDateOfBirth) == "" {
			m["user_profile_date_of_birth"] = nil
		} else {
			dob, err := time.Parse("2006-01-02", strings.TrimSpace(*in.UserProfileDateOfBirth))
			if err != nil {
				return nil, errors.New("user_profile_date_of_birth must be YYYY-MM-DD")
			}
			m["user_profile_date_of_birth"] = dob
		}
	}

	// Gender
	if in.UserProfileGender != nil {
		g := strings.ToLower(strings.TrimSpace(*in.UserProfileGender))
		if g != string(profilemodel.Male) && g != string(profilemodel.Female) {
			return nil, errors.New("user_profile_gender must be 'male' or 'female'")
		}
		m["user_profile_gender"] = g
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

func CompactStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
