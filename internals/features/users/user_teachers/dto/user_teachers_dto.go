// file: internals/features/users/dto/user_teacher_dto.go
package dto

import (
	"time"

	"madinahsalam_backend/internals/features/users/user_teachers/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

//
// ========== CREATE ==========
//

type CreateUserTeacherRequest struct {
	// FK wajib
	UserTeacherUserID uuid.UUID `json:"user_teacher_user_id" form:"user_teacher_user_id"`

	// Profil ringkas
	UserTeacherUserFullNameCache string `json:"user_teacher_user_full_name_cache" form:"user_teacher_user_full_name_cache" validate:"required,max=100"`
	UserTeacherUserNameCache     string `json:"user_teacher_user_name_cache" form:"user_teacher_user_name_cache" validate:"required,max=50"`
	UserTeacherField             string `json:"user_teacher_field" form:"user_teacher_field" validate:"omitempty,max=80"`
	UserTeacherShortBio          string `json:"user_teacher_short_bio" form:"user_teacher_short_bio" validate:"omitempty,max=300"`
	UserTeacherLongBio           string `json:"user_teacher_long_bio" form:"user_teacher_long_bio" validate:"omitempty"`
	UserTeacherGreeting          string `json:"user_teacher_greeting" form:"user_teacher_greeting" validate:"omitempty"`
	UserTeacherEducation         string `json:"user_teacher_education" form:"user_teacher_education" validate:"omitempty"`
	UserTeacherActivity          string `json:"user_teacher_activity" form:"user_teacher_activity" validate:"omitempty"`
	UserTeacherExperienceYears   *int16 `json:"user_teacher_experience_years" form:"user_teacher_experience_years" validate:"omitempty,min=0,max=80"`

	// Demografis
	UserTeacherGender   string `json:"user_teacher_gender" form:"user_teacher_gender" validate:"omitempty,max=10"`
	UserTeacherLocation string `json:"user_teacher_location" form:"user_teacher_location" validate:"omitempty,max=100"`
	UserTeacherCity     string `json:"user_teacher_city" form:"user_teacher_city" validate:"omitempty,max=100"`

	// Metadata fleksibel (pakai payload JSON saat multipart)
	UserTeacherSpecialties  *datatypes.JSON `json:"user_teacher_specialties" validate:"omitempty"`
	UserTeacherCertificates *datatypes.JSON `json:"user_teacher_certificates" validate:"omitempty"`

	// Sosial (opsional)
	UserTeacherInstagramURL     string `json:"user_teacher_instagram_url" form:"user_teacher_instagram_url" validate:"omitempty,url"`
	UserTeacherWhatsappURL      string `json:"user_teacher_whatsapp_url" form:"user_teacher_whatsapp_url" validate:"omitempty"`
	UserTeacherYoutubeURL       string `json:"user_teacher_youtube_url" form:"user_teacher_youtube_url" validate:"omitempty,url"`
	UserTeacherLinkedinURL      string `json:"user_teacher_linkedin_url" form:"user_teacher_linkedin_url" validate:"omitempty,url"`
	UserTeacherGithubURL        string `json:"user_teacher_github_url" form:"user_teacher_github_url" validate:"omitempty,url"`
	UserTeacherTelegramUsername string `json:"user_teacher_telegram_username" form:"user_teacher_telegram_username" validate:"omitempty,max=50"`

	// Title
	UserTeacherTitlePrefix string `json:"user_teacher_title_prefix" form:"user_teacher_title_prefix" validate:"omitempty,max=60"`
	UserTeacherTitleSuffix string `json:"user_teacher_title_suffix" form:"user_teacher_title_suffix" validate:"omitempty,max=60"`

	// Avatar (opsional di create)
	UserTeacherAvatarURL                string     `json:"user_teacher_avatar_url" form:"user_teacher_avatar_url" validate:"omitempty,max=2048"`
	UserTeacherAvatarObjectKey          string     `json:"user_teacher_avatar_object_key" form:"user_teacher_avatar_object_key" validate:"omitempty,max=2048"`
	UserTeacherAvatarURLOld             string     `json:"user_teacher_avatar_url_old" form:"user_teacher_avatar_url_old" validate:"omitempty,max=2048"`
	UserTeacherAvatarObjectKeyOld       string     `json:"user_teacher_avatar_object_key_old" form:"user_teacher_avatar_object_key_old" validate:"omitempty,max=2048"`
	UserTeacherAvatarDeletePendingUntil *time.Time `json:"user_teacher_avatar_delete_pending_until" form:"user_teacher_avatar_delete_pending_until" validate:"omitempty"`

	// Flags (opsional)
	UserTeacherIsVerified *bool `json:"user_teacher_is_verified" form:"user_teacher_is_verified" validate:"omitempty"`
	UserTeacherIsActive   *bool `json:"user_teacher_is_active" form:"user_teacher_is_active" validate:"omitempty"`
}

// ToModel: mapping Create → model.UserTeacher
func (r CreateUserTeacherRequest) ToModel() model.UserTeacherModel {
	m := model.UserTeacherModel{
		UserTeacherUserID:            r.UserTeacherUserID,
		UserTeacherUserFullNameCache: r.UserTeacherUserFullNameCache,
		UserTeacherUserNameCache:     r.UserTeacherUserNameCache,
		UserTeacherIsVerified:        false,
		UserTeacherIsActive:          true,
	}

	// Optional strings → *string (NULL jika empty)
	if p := nilIfEmpty(r.UserTeacherField); p != nil {
		m.UserTeacherField = p
	}
	if p := nilIfEmpty(r.UserTeacherShortBio); p != nil {
		m.UserTeacherShortBio = p
	}
	if p := nilIfEmpty(r.UserTeacherLongBio); p != nil {
		m.UserTeacherLongBio = p
	}
	if p := nilIfEmpty(r.UserTeacherGreeting); p != nil {
		m.UserTeacherGreeting = p
	}
	if p := nilIfEmpty(r.UserTeacherEducation); p != nil {
		m.UserTeacherEducation = p
	}
	if p := nilIfEmpty(r.UserTeacherActivity); p != nil {
		m.UserTeacherActivity = p
	}
	if r.UserTeacherExperienceYears != nil {
		m.UserTeacherExperienceYears = r.UserTeacherExperienceYears
	}

	// Demografis
	if p := nilIfEmpty(r.UserTeacherGender); p != nil {
		m.UserTeacherGender = p
	}
	if p := nilIfEmpty(r.UserTeacherLocation); p != nil {
		m.UserTeacherLocation = p
	}
	if p := nilIfEmpty(r.UserTeacherCity); p != nil {
		m.UserTeacherCity = p
	}

	// JSONB (pointer → boleh NULL)
	applyJSONCreate(&m.UserTeacherSpecialties, r.UserTeacherSpecialties)
	applyJSONCreate(&m.UserTeacherCertificates, r.UserTeacherCertificates)

	// Sosial
	if p := nilIfEmpty(r.UserTeacherInstagramURL); p != nil {
		m.UserTeacherInstagramURL = p
	}
	if p := nilIfEmpty(r.UserTeacherWhatsappURL); p != nil {
		m.UserTeacherWhatsappURL = p
	}
	if p := nilIfEmpty(r.UserTeacherYoutubeURL); p != nil {
		m.UserTeacherYoutubeURL = p
	}
	if p := nilIfEmpty(r.UserTeacherLinkedinURL); p != nil {
		m.UserTeacherLinkedinURL = p
	}
	if p := nilIfEmpty(r.UserTeacherGithubURL); p != nil {
		m.UserTeacherGithubURL = p
	}
	if p := nilIfEmpty(r.UserTeacherTelegramUsername); p != nil {
		m.UserTeacherTelegramUsername = p
	}

	// Title
	if p := nilIfEmpty(r.UserTeacherTitlePrefix); p != nil {
		m.UserTeacherTitlePrefix = p
	}
	if p := nilIfEmpty(r.UserTeacherTitleSuffix); p != nil {
		m.UserTeacherTitleSuffix = p
	}

	// Avatar
	if p := nilIfEmpty(r.UserTeacherAvatarURL); p != nil {
		m.UserTeacherAvatarURL = p
	}
	if p := nilIfEmpty(r.UserTeacherAvatarObjectKey); p != nil {
		m.UserTeacherAvatarObjectKey = p
	}
	if p := nilIfEmpty(r.UserTeacherAvatarURLOld); p != nil {
		m.UserTeacherAvatarURLOld = p
	}
	if p := nilIfEmpty(r.UserTeacherAvatarObjectKeyOld); p != nil {
		m.UserTeacherAvatarObjectKeyOld = p
	}
	if r.UserTeacherAvatarDeletePendingUntil != nil {
		m.UserTeacherAvatarDeletePendingUntil = r.UserTeacherAvatarDeletePendingUntil
	}

	// Flags
	if r.UserTeacherIsVerified != nil {
		m.UserTeacherIsVerified = *r.UserTeacherIsVerified
	}
	if r.UserTeacherIsActive != nil {
		m.UserTeacherIsActive = *r.UserTeacherIsActive
	}

	return m
}

// ========== UPDATE / PATCH ==========

type UpdateUserTeacherRequest struct {
	// Profil ringkas
	UserTeacherUserFullNameCache *string `json:"user_teacher_user_full_name_cache" form:"user_teacher_user_full_name_cache" validate:"omitempty,max=100"`
	UserTeacherUserNameCache     *string `json:"user_teacher_user_name_cache" form:"user_teacher_user_name_cache" validate:"omitempty,max=50"`
	UserTeacherField             *string `json:"user_teacher_field" form:"user_teacher_field" validate:"omitempty,max=80"`
	UserTeacherShortBio          *string `json:"user_teacher_short_bio" form:"user_teacher_short_bio" validate:"omitempty,max=300"`
	UserTeacherLongBio           *string `json:"user_teacher_long_bio" form:"user_teacher_long_bio" validate:"omitempty"`
	UserTeacherGreeting          *string `json:"user_teacher_greeting" form:"user_teacher_greeting" validate:"omitempty"`
	UserTeacherEducation         *string `json:"user_teacher_education" form:"user_teacher_education" validate:"omitempty"`
	UserTeacherActivity          *string `json:"user_teacher_activity" form:"user_teacher_activity" validate:"omitempty"`
	UserTeacherExperienceYears   *int16  `json:"user_teacher_experience_years" form:"user_teacher_experience_years" validate:"omitempty,min=0,max=80"`

	// Demografis
	UserTeacherGender   *string `json:"user_teacher_gender" form:"user_teacher_gender" validate:"omitempty,max=10"`
	UserTeacherLocation *string `json:"user_teacher_location" form:"user_teacher_location" validate:"omitempty,max=100"`
	UserTeacherCity     *string `json:"user_teacher_city" form:"user_teacher_city" validate:"omitempty,max=100"`

	// Metadata fleksibel
	UserTeacherSpecialties  **datatypes.JSON `json:"user_teacher_specialties" validate:"omitempty"`
	UserTeacherCertificates **datatypes.JSON `json:"user_teacher_certificates" validate:"omitempty"`

	// Sosial
	UserTeacherInstagramURL     *string `json:"user_teacher_instagram_url" form:"user_teacher_instagram_url" validate:"omitempty,url,max=2048"`
	UserTeacherWhatsappURL      *string `json:"user_teacher_whatsapp_url" form:"user_teacher_whatsapp_url" validate:"omitempty,max=2048"`
	UserTeacherYoutubeURL       *string `json:"user_teacher_youtube_url" form:"user_teacher_youtube_url" validate:"omitempty,url,max=2048"`
	UserTeacherLinkedinURL      *string `json:"user_teacher_linkedin_url" form:"user_teacher_linkedin_url" validate:"omitempty,url,max=2048"`
	UserTeacherGithubURL        *string `json:"user_teacher_github_url" form:"user_teacher_github_url" validate:"omitempty,url,max=2048"`
	UserTeacherTelegramUsername *string `json:"user_teacher_telegram_username" form:"user_teacher_telegram_username" validate:"omitempty,max=50"`

	// Title
	UserTeacherTitlePrefix *string `json:"user_teacher_title_prefix" form:"user_teacher_title_prefix" validate:"omitempty,max=60"`
	UserTeacherTitleSuffix *string `json:"user_teacher_title_suffix" form:"user_teacher_title_suffix" validate:"omitempty,max=60"`

	// Avatar (2-slot + retensi)
	UserTeacherAvatarURL                *string    `json:"user_teacher_avatar_url" form:"user_teacher_avatar_url" validate:"omitempty,max=2048"`
	UserTeacherAvatarObjectKey          *string    `json:"user_teacher_avatar_object_key" form:"user_teacher_avatar_object_key" validate:"omitempty,max=2048"`
	UserTeacherAvatarURLOld             *string    `json:"user_teacher_avatar_url_old" form:"user_teacher_avatar_url_old" validate:"omitempty,max=2048"`
	UserTeacherAvatarObjectKeyOld       *string    `json:"user_teacher_avatar_object_key_old" form:"user_teacher_avatar_object_key_old" validate:"omitempty,max=2048"`
	UserTeacherAvatarDeletePendingUntil *time.Time `json:"user_teacher_avatar_delete_pending_until" form:"user_teacher_avatar_delete_pending_until" validate:"omitempty"`

	// Flags
	UserTeacherIsVerified  *bool `json:"user_teacher_is_verified" form:"user_teacher_is_verified" validate:"omitempty"`
	UserTeacherIsActive    *bool `json:"user_teacher_is_active" form:"user_teacher_is_active" validate:"omitempty"`
	UserTeacherIsCompleted *bool `json:"user_teacher_is_completed" form:"user_teacher_is_completed" validate:"omitempty"`

	// Kolom yang ingin DIKOSONGKAN (set NULL) eksplisit
	Clear []string `json:"__clear,omitempty" form:"__clear" validate:"omitempty,dive,oneof=user_teacher_field user_teacher_short_bio user_teacher_long_bio user_teacher_greeting user_teacher_education user_teacher_activity user_teacher_experience_years user_teacher_gender user_teacher_location user_teacher_city user_teacher_specialties user_teacher_certificates user_teacher_instagram_url user_teacher_whatsapp_url user_teacher_youtube_url user_teacher_linkedin_url user_teacher_github_url user_teacher_telegram_username user_teacher_title_prefix user_teacher_title_suffix user_teacher_avatar_url user_teacher_avatar_object_key user_teacher_avatar_url_old user_teacher_avatar_object_key_old user_teacher_avatar_delete_pending_until"`
}

// ApplyPatch: terapkan update parsial ke model.
func (r UpdateUserTeacherRequest) ApplyPatch(m *model.UserTeacherModel) {
	// setter biasa ...
	// (isi sama persis seperti yang kamu punya, tidak perlu diubah)
	// ...
	// (biar jawaban ini nggak kepanjangan, bagian ApplyPatch kamu bisa pakai versi sebelumnya tanpa perubahan)
}

//
// ========== RESPONSE ==========
//

type UserTeacherResponse struct {
	// PK & FK
	UserTeacherID     uuid.UUID `json:"user_teacher_id"`
	UserTeacherUserID uuid.UUID `json:"user_teacher_user_id"`

	// Profil ringkas
	UserTeacherNameSnapshot    string `json:"user_teacher_user_full_name_cache"`
	UserTeacherUserNameCache   string `json:"user_teacher_user_name_cache"`
	UserTeacherField           string `json:"user_teacher_field"`
	UserTeacherShortBio        string `json:"user_teacher_short_bio"`
	UserTeacherLongBio         string `json:"user_teacher_long_bio"`
	UserTeacherGreeting        string `json:"user_teacher_greeting"`
	UserTeacherEducation       string `json:"user_teacher_education"`
	UserTeacherActivity        string `json:"user_teacher_activity"`
	UserTeacherExperienceYears *int16 `json:"user_teacher_experience_years"`

	// Demografis
	UserTeacherGender   string `json:"user_teacher_gender"`
	UserTeacherLocation string `json:"user_teacher_location"`
	UserTeacherCity     string `json:"user_teacher_city"`

	// Metadata fleksibel
	UserTeacherSpecialties  *datatypes.JSON `json:"user_teacher_specialties"`
	UserTeacherCertificates *datatypes.JSON `json:"user_teacher_certificates"`

	// Sosial
	UserTeacherInstagramURL     string `json:"user_teacher_instagram_url"`
	UserTeacherWhatsappURL      string `json:"user_teacher_whatsapp_url"`
	UserTeacherYoutubeURL       string `json:"user_teacher_youtube_url"`
	UserTeacherLinkedinURL      string `json:"user_teacher_linkedin_url"`
	UserTeacherGithubURL        string `json:"user_teacher_github_url"`
	UserTeacherTelegramUsername string `json:"user_teacher_telegram_username"`

	// Title
	UserTeacherTitlePrefix string `json:"user_teacher_title_prefix"`
	UserTeacherTitleSuffix string `json:"user_teacher_title_suffix"`

	// Avatar
	UserTeacherAvatarURL                string     `json:"user_teacher_avatar_url"`
	UserTeacherAvatarObjectKey          string     `json:"user_teacher_avatar_object_key"`
	UserTeacherAvatarURLOld             string     `json:"user_teacher_avatar_url_old"`
	UserTeacherAvatarObjectKeyOld       string     `json:"user_teacher_avatar_object_key_old"`
	UserTeacherAvatarDeletePendingUntil *time.Time `json:"user_teacher_avatar_delete_pending_until"`

	// Status
	UserTeacherIsVerified  bool       `json:"user_teacher_is_verified"`
	UserTeacherIsActive    bool       `json:"user_teacher_is_active"`
	UserTeacherIsCompleted bool       `json:"user_teacher_is_completed"`
	UserTeacherCompletedAt *time.Time `json:"user_teacher_completed_at"`

	// Audit
	UserTeacherCreatedAt string `json:"user_teacher_created_at"`
	UserTeacherUpdatedAt string `json:"user_teacher_updated_at"`
}

// helper: ubah datatypes.JSON → *datatypes.JSON (nil-aware)
func ptrJSON(v datatypes.JSON) *datatypes.JSON {
	if v == nil {
		return nil
	}
	return &v
}

// Mapping model → response
func ToUserTeacherResponse(m model.UserTeacherModel) UserTeacherResponse {
	return UserTeacherResponse{
		UserTeacherID:     m.UserTeacherID,
		UserTeacherUserID: m.UserTeacherUserID,

		UserTeacherNameSnapshot:  m.UserTeacherUserFullNameCache,
		UserTeacherUserNameCache: m.UserTeacherUserNameCache,

		UserTeacherField:           deref(m.UserTeacherField),
		UserTeacherShortBio:        deref(m.UserTeacherShortBio),
		UserTeacherLongBio:         deref(m.UserTeacherLongBio),
		UserTeacherGreeting:        deref(m.UserTeacherGreeting),
		UserTeacherEducation:       deref(m.UserTeacherEducation),
		UserTeacherActivity:        deref(m.UserTeacherActivity),
		UserTeacherExperienceYears: m.UserTeacherExperienceYears,

		UserTeacherGender:   deref(m.UserTeacherGender),
		UserTeacherLocation: deref(m.UserTeacherLocation),
		UserTeacherCity:     deref(m.UserTeacherCity),

		UserTeacherSpecialties:  ptrJSON(m.UserTeacherSpecialties),
		UserTeacherCertificates: ptrJSON(m.UserTeacherCertificates),

		UserTeacherInstagramURL:     deref(m.UserTeacherInstagramURL),
		UserTeacherWhatsappURL:      deref(m.UserTeacherWhatsappURL),
		UserTeacherYoutubeURL:       deref(m.UserTeacherYoutubeURL),
		UserTeacherLinkedinURL:      deref(m.UserTeacherLinkedinURL),
		UserTeacherGithubURL:        deref(m.UserTeacherGithubURL),
		UserTeacherTelegramUsername: deref(m.UserTeacherTelegramUsername),

		UserTeacherTitlePrefix: deref(m.UserTeacherTitlePrefix),
		UserTeacherTitleSuffix: deref(m.UserTeacherTitleSuffix),

		UserTeacherAvatarURL:                deref(m.UserTeacherAvatarURL),
		UserTeacherAvatarObjectKey:          deref(m.UserTeacherAvatarObjectKey),
		UserTeacherAvatarURLOld:             deref(m.UserTeacherAvatarURLOld),
		UserTeacherAvatarObjectKeyOld:       deref(m.UserTeacherAvatarObjectKeyOld),
		UserTeacherAvatarDeletePendingUntil: m.UserTeacherAvatarDeletePendingUntil,

		UserTeacherIsVerified:  m.UserTeacherIsVerified,
		UserTeacherIsActive:    m.UserTeacherIsActive,
		UserTeacherIsCompleted: m.UserTeacherIsCompleted,
		UserTeacherCompletedAt: m.UserTeacherCompletedAt,

		UserTeacherCreatedAt: m.UserTeacherCreatedAt.Format(time.RFC3339),
		UserTeacherUpdatedAt: m.UserTeacherUpdatedAt.Format(time.RFC3339),
	}
}

//
// ========== helpers ==========
//

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// JSON helpers
func applyJSONCreate(dst *datatypes.JSON, src *datatypes.JSON) {
	if src != nil {
		*dst = *src
	}
}
