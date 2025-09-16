package dto

import (
	"masjidku_backend/internals/features/lembaga/teachers_students/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

//
// ========== CREATE ==========
//

type CreateUserTeacherRequest struct {
	UserTeacherUserID          uuid.UUID       `json:"user_teacher_user_id" validate:"required"`
	UserTeacherField           string          `json:"user_teacher_field" validate:"omitempty,max=80"`
	UserTeacherShortBio        string          `json:"user_teacher_short_bio" validate:"omitempty,max=300"`
	UserTeacherLongBio         string          `json:"user_teacher_long_bio" validate:"omitempty"`
	UserTeacherGreeting        string          `json:"user_teacher_greeting" validate:"omitempty"`
	UserTeacherEducation       string          `json:"user_teacher_education" validate:"omitempty"`
	UserTeacherActivity        string          `json:"user_teacher_activity" validate:"omitempty"`
	UserTeacherExperienceYears *int16          `json:"user_teacher_experience_years" validate:"omitempty,min=0,max=80"`
	UserTeacherSpecialties     *datatypes.JSON `json:"user_teacher_specialties" validate:"omitempty"`  // pointer agar bisa NULL
	UserTeacherCertificates    *datatypes.JSON `json:"user_teacher_certificates" validate:"omitempty"` // pointer agar bisa NULL
	UserTeacherIsVerified      *bool           `json:"user_teacher_is_verified" validate:"omitempty"`
	UserTeacherIsActive        *bool           `json:"user_teacher_is_active" validate:"omitempty"`
}

// ToModel: mapping Create → model.UserTeacher
func (r CreateUserTeacherRequest) ToModel() model.UserTeacher {
	m := model.UserTeacher{
		UserTeacherUserID: r.UserTeacherUserID,
		// default (biarkan DB), tapi kita set eksplisit untuk konsistensi respons awal
		UserTeacherIsVerified: false,
		UserTeacherIsActive:   true,
	}

	// String optional → *string (NULL jika kosong)
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
	if r.UserTeacherIsVerified != nil {
		m.UserTeacherIsVerified = *r.UserTeacherIsVerified
	}
	if r.UserTeacherIsActive != nil {
		m.UserTeacherIsActive = *r.UserTeacherIsActive
	}

	// JSONB (pointer → bisa NULL)
	if r.UserTeacherSpecialties != nil {
		m.UserTeacherSpecialties = r.UserTeacherSpecialties
	}
	if r.UserTeacherCertificates != nil {
		m.UserTeacherCertificates = r.UserTeacherCertificates
	}

	return m
}

//
// ========== UPDATE / PATCH ==========
//

// Catatan PATCH:
// - Field pointer: nil = tidak diubah, non-nil = set ke value (termasuk empty string "").
// - Untuk set NULL secara eksplisit, gunakan __clear: ["nama_kolom", ...]
type UpdateUserTeacherRequest struct {
	UserTeacherField           *string          `json:"user_teacher_field" validate:"omitempty,max=80"`
	UserTeacherShortBio        *string          `json:"user_teacher_short_bio" validate:"omitempty,max=300"`
	UserTeacherLongBio         *string          `json:"user_teacher_long_bio" validate:"omitempty"`
	UserTeacherGreeting        *string          `json:"user_teacher_greeting" validate:"omitempty"`
	UserTeacherEducation       *string          `json:"user_teacher_education" validate:"omitempty"`
	UserTeacherActivity        *string          `json:"user_teacher_activity" validate:"omitempty"`
	UserTeacherExperienceYears *int16           `json:"user_teacher_experience_years" validate:"omitempty,min=0,max=80"`
	UserTeacherSpecialties     **datatypes.JSON `json:"user_teacher_specialties" validate:"omitempty"`  // **JSON: bedakan “tak diubah” vs “set ke []/{...}”
	UserTeacherCertificates    **datatypes.JSON `json:"user_teacher_certificates" validate:"omitempty"`
	UserTeacherIsVerified      *bool            `json:"user_teacher_is_verified" validate:"omitempty"`
	UserTeacherIsActive        *bool            `json:"user_teacher_is_active" validate:"omitempty"`

	// Kolom yang ingin DIKOSONGKAN (set NULL) secara eksplisit
	// contoh: "__clear": ["user_teacher_field","user_teacher_specialties"]
	Clear []string `json:"__clear,omitempty" validate:"omitempty,dive,oneof=user_teacher_field user_teacher_short_bio user_teacher_long_bio user_teacher_greeting user_teacher_education user_teacher_activity user_teacher_experience_years user_teacher_specialties user_teacher_certificates"`
}

// ApplyPatch: terapkan update parsial ke model.
func (r UpdateUserTeacherRequest) ApplyPatch(m *model.UserTeacher) {
	// 1) Setter biasa (tanpa NULL)
	if r.UserTeacherField != nil {
		m.UserTeacherField = r.UserTeacherField
	}
	if r.UserTeacherShortBio != nil {
		m.UserTeacherShortBio = r.UserTeacherShortBio
	}
	if r.UserTeacherLongBio != nil {
		m.UserTeacherLongBio = r.UserTeacherLongBio
	}
	if r.UserTeacherGreeting != nil {
		m.UserTeacherGreeting = r.UserTeacherGreeting
	}
	if r.UserTeacherEducation != nil {
		m.UserTeacherEducation = r.UserTeacherEducation
	}
	if r.UserTeacherActivity != nil {
		m.UserTeacherActivity = r.UserTeacherActivity
	}
	if r.UserTeacherExperienceYears != nil {
		m.UserTeacherExperienceYears = r.UserTeacherExperienceYears
	}

	// JSONB: **datatypes.JSON → bisa bedakan “tidak ada field” vs “set ke {} / []”
	if r.UserTeacherSpecialties != nil {
		m.UserTeacherSpecialties = *r.UserTeacherSpecialties // boleh nil (akan ke NULL), atau &json
	}
	if r.UserTeacherCertificates != nil {
		m.UserTeacherCertificates = *r.UserTeacherCertificates
	}

	if r.UserTeacherIsVerified != nil {
		m.UserTeacherIsVerified = *r.UserTeacherIsVerified
	}
	if r.UserTeacherIsActive != nil {
		m.UserTeacherIsActive = *r.UserTeacherIsActive
	}

	// 2) Clear → set NULL eksplisit
	for _, col := range r.Clear {
		switch col {
		case "user_teacher_field":
			m.UserTeacherField = nil
		case "user_teacher_short_bio":
			m.UserTeacherShortBio = nil
		case "user_teacher_long_bio":
			m.UserTeacherLongBio = nil
		case "user_teacher_greeting":
			m.UserTeacherGreeting = nil
		case "user_teacher_education":
			m.UserTeacherEducation = nil
		case "user_teacher_activity":
			m.UserTeacherActivity = nil
		case "user_teacher_experience_years":
			m.UserTeacherExperienceYears = nil
		case "user_teacher_specialties":
			m.UserTeacherSpecialties = nil
		case "user_teacher_certificates":
			m.UserTeacherCertificates = nil
		}
	}
}

//
// ========== RESPONSE ==========
//

type UserTeacherResponse struct {
	UserTeacherID              uuid.UUID       `json:"user_teacher_id"`
	UserTeacherUserID          uuid.UUID       `json:"user_teacher_user_id"`
	FullName                   string          `json:"full_name"` // join ke users.full_name
	UserTeacherField           string          `json:"user_teacher_field"`
	UserTeacherShortBio        string          `json:"user_teacher_short_bio"`
	UserTeacherLongBio         string          `json:"user_teacher_long_bio"`
	UserTeacherGreeting        string          `json:"user_teacher_greeting"`
	UserTeacherEducation       string          `json:"user_teacher_education"`
	UserTeacherActivity        string          `json:"user_teacher_activity"`
	UserTeacherExperienceYears *int16          `json:"user_teacher_experience_years"`
	UserTeacherSpecialties     *datatypes.JSON `json:"user_teacher_specialties"`  // pointer → bisa NULL di respons
	UserTeacherCertificates    *datatypes.JSON `json:"user_teacher_certificates"` // pointer → bisa NULL di respons
	UserTeacherIsVerified      bool            `json:"user_teacher_is_verified"`
	UserTeacherIsActive        bool            `json:"user_teacher_is_active"`
	UserTeacherCreatedAt       string          `json:"user_teacher_created_at"`
	UserTeacherUpdatedAt       string          `json:"user_teacher_updated_at"`
}

// Mapping model → response
func ToUserTeacherResponse(m model.UserTeacher, fullName string) UserTeacherResponse {
	return UserTeacherResponse{
		UserTeacherID:              m.UserTeacherID,
		UserTeacherUserID:          m.UserTeacherUserID,
		FullName:                   fullName,
		UserTeacherField:           deref(m.UserTeacherField),
		UserTeacherShortBio:        deref(m.UserTeacherShortBio),
		UserTeacherLongBio:         deref(m.UserTeacherLongBio),
		UserTeacherGreeting:        deref(m.UserTeacherGreeting),
		UserTeacherEducation:       deref(m.UserTeacherEducation),
		UserTeacherActivity:        deref(m.UserTeacherActivity),
		UserTeacherExperienceYears: m.UserTeacherExperienceYears,
		UserTeacherSpecialties:     m.UserTeacherSpecialties,
		UserTeacherCertificates:    m.UserTeacherCertificates,
		UserTeacherIsVerified:      m.UserTeacherIsVerified,
		UserTeacherIsActive:        m.UserTeacherIsActive,
		UserTeacherCreatedAt:       m.UserTeacherCreatedAt.Format(time.RFC3339),
		UserTeacherUpdatedAt:       m.UserTeacherUpdatedAt.Format(time.RFC3339),
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
