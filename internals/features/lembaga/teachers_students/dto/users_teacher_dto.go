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

type CreateUsersTeacherRequest struct {
	UsersTeacherUserID           uuid.UUID      `json:"users_teacher_user_id" validate:"required"`
	UsersTeacherField            string         `json:"users_teacher_field" validate:"omitempty,max=80"`
	UsersTeacherShortBio         string         `json:"users_teacher_short_bio" validate:"omitempty,max=300"`
	UsersTeacherGreeting         string         `json:"users_teacher_greeting" validate:"omitempty"`
	UsersTeacherEducation        string         `json:"users_teacher_education" validate:"omitempty"`
	UsersTeacherActivity         string         `json:"users_teacher_activity" validate:"omitempty"`
	UsersTeacherExperienceYears  *int16         `json:"users_teacher_experience_years" validate:"omitempty"`
	UsersTeacherSpecialties      datatypes.JSON `json:"users_teacher_specialties" validate:"omitempty"`
	UsersTeacherCertificates     datatypes.JSON `json:"users_teacher_certificates" validate:"omitempty"`
	UsersTeacherLinks            datatypes.JSON `json:"users_teacher_links" validate:"omitempty"`
	UsersTeacherIsVerified       *bool          `json:"users_teacher_is_verified" validate:"omitempty"`
	UsersTeacherIsActive         *bool          `json:"users_teacher_is_active" validate:"omitempty"`
}

// ToModel: mapping Create → model.UserTeacher
func (r CreateUsersTeacherRequest) ToModel() model.UserTeacher {
	m := model.UserTeacher{
		UsersTeacherUserID:         r.UsersTeacherUserID,
		UsersTeacherSpecialties:    r.UsersTeacherSpecialties,
		UsersTeacherCertificates:   r.UsersTeacherCertificates,
		UsersTeacherLinks:          r.UsersTeacherLinks,
		UsersTeacherIsVerified:     false, // default by DB, set eksplisit agar konsisten
		UsersTeacherIsActive:       true,  // default by DB
	}

	// String optional → *string (NULL jika kosong)
	if p := nilIfEmpty(r.UsersTeacherField); p != nil {
		m.UsersTeacherField = p
	}
	if p := nilIfEmpty(r.UsersTeacherShortBio); p != nil {
		m.UsersTeacherShortBio = p
	}
	if p := nilIfEmpty(r.UsersTeacherGreeting); p != nil {
		m.UsersTeacherGreeting = p
	}
	if p := nilIfEmpty(r.UsersTeacherEducation); p != nil {
		m.UsersTeacherEducation = p
	}
	if p := nilIfEmpty(r.UsersTeacherActivity); p != nil {
		m.UsersTeacherActivity = p
	}

	if r.UsersTeacherExperienceYears != nil {
		m.UsersTeacherExperienceYears = r.UsersTeacherExperienceYears
	}
	if r.UsersTeacherIsVerified != nil {
		m.UsersTeacherIsVerified = *r.UsersTeacherIsVerified
	}
	if r.UsersTeacherIsActive != nil {
		m.UsersTeacherIsActive = *r.UsersTeacherIsActive
	}

	return m
}

//
// ========== UPDATE / PATCH ==========
//

// Catatan PATCH:
// - Field pointer: nil = tidak diubah, non-nil = set ke value (termasuk empty string "").
// - Untuk set NULL secara eksplisit, gunakan __clear: ["nama_kolom", ...]
type UpdateUsersTeacherRequest struct {
	UsersTeacherField           *string         `json:"users_teacher_field" validate:"omitempty,max=80"`
	UsersTeacherShortBio        *string         `json:"users_teacher_short_bio" validate:"omitempty,max=300"`
	UsersTeacherGreeting        *string         `json:"users_teacher_greeting" validate:"omitempty"`
	UsersTeacherEducation       *string         `json:"users_teacher_education" validate:"omitempty"`
	UsersTeacherActivity        *string         `json:"users_teacher_activity" validate:"omitempty"`
	UsersTeacherExperienceYears *int16          `json:"users_teacher_experience_years" validate:"omitempty"`
	UsersTeacherSpecialties     *datatypes.JSON `json:"users_teacher_specialties" validate:"omitempty"`
	UsersTeacherCertificates    *datatypes.JSON `json:"users_teacher_certificates" validate:"omitempty"`
	UsersTeacherLinks           *datatypes.JSON `json:"users_teacher_links" validate:"omitempty"`
	UsersTeacherIsVerified      *bool           `json:"users_teacher_is_verified" validate:"omitempty"`
	UsersTeacherIsActive        *bool           `json:"users_teacher_is_active" validate:"omitempty"`

	// Kolom yang ingin DIKOSONGKAN (set NULL) secara eksplisit
	// contoh: "__clear": ["users_teacher_field","users_teacher_specialties"]
	Clear []string `json:"__clear,omitempty" validate:"omitempty,dive,oneof=users_teacher_field users_teacher_short_bio users_teacher_greeting users_teacher_education users_teacher_activity users_teacher_experience_years users_teacher_specialties users_teacher_certificates users_teacher_links"`
}

// ApplyPatch: terapkan update parsial ke model.
// - string: jika pointer non-nil → set; untuk NULL gunakan Clear.
// - JSONB: jika pointer non-nil → replace; untuk NULL gunakan Clear.
// - smallint/bool: jika pointer non-nil → set; untuk NULL gunakan Clear (khusus smallint).
func (r UpdateUsersTeacherRequest) ApplyPatch(m *model.UserTeacher) {
	// 1) Setter biasa (tanpa NULL)
	if r.UsersTeacherField != nil {
		// assignment langsung: empty string "" tetap disimpan (bukan NULL)
		m.UsersTeacherField = r.UsersTeacherField
	}
	if r.UsersTeacherShortBio != nil {
		m.UsersTeacherShortBio = r.UsersTeacherShortBio
	}
	if r.UsersTeacherGreeting != nil {
		m.UsersTeacherGreeting = r.UsersTeacherGreeting
	}
	if r.UsersTeacherEducation != nil {
		m.UsersTeacherEducation = r.UsersTeacherEducation
	}
	if r.UsersTeacherActivity != nil {
		m.UsersTeacherActivity = r.UsersTeacherActivity
	}
	if r.UsersTeacherExperienceYears != nil {
		m.UsersTeacherExperienceYears = r.UsersTeacherExperienceYears
	}
	if r.UsersTeacherSpecialties != nil {
		m.UsersTeacherSpecialties = *r.UsersTeacherSpecialties
	}
	if r.UsersTeacherCertificates != nil {
		m.UsersTeacherCertificates = *r.UsersTeacherCertificates
	}
	if r.UsersTeacherLinks != nil {
		m.UsersTeacherLinks = *r.UsersTeacherLinks
	}
	if r.UsersTeacherIsVerified != nil {
		m.UsersTeacherIsVerified = *r.UsersTeacherIsVerified
	}
	if r.UsersTeacherIsActive != nil {
		m.UsersTeacherIsActive = *r.UsersTeacherIsActive
	}

	// 2) Clear → set NULL eksplisit
	for _, col := range r.Clear {
		switch col {
		case "users_teacher_field":
			m.UsersTeacherField = nil
		case "users_teacher_short_bio":
			m.UsersTeacherShortBio = nil
		case "users_teacher_greeting":
			m.UsersTeacherGreeting = nil
		case "users_teacher_education":
			m.UsersTeacherEducation = nil
		case "users_teacher_activity":
			m.UsersTeacherActivity = nil
		case "users_teacher_experience_years":
			m.UsersTeacherExperienceYears = nil
		case "users_teacher_specialties":
			m.UsersTeacherSpecialties = nil
		case "users_teacher_certificates":
			m.UsersTeacherCertificates = nil
		case "users_teacher_links":
			m.UsersTeacherLinks = nil
		}
	}
}

//
// ========== RESPONSE ==========
//

type UsersTeacherResponse struct {
	UsersTeacherID               uuid.UUID      `json:"users_teacher_id"`
	UsersTeacherUserID           uuid.UUID      `json:"users_teacher_user_id"`
	FullName                     string         `json:"full_name"` // dari join users
	UsersTeacherField            string         `json:"users_teacher_field"`
	UsersTeacherShortBio         string         `json:"users_teacher_short_bio"`
	UsersTeacherGreeting         string         `json:"users_teacher_greeting"`
	UsersTeacherEducation        string         `json:"users_teacher_education"`
	UsersTeacherActivity         string         `json:"users_teacher_activity"`
	UsersTeacherExperienceYears  *int16         `json:"users_teacher_experience_years"`
	UsersTeacherSpecialties      datatypes.JSON `json:"users_teacher_specialties"`
	UsersTeacherCertificates     datatypes.JSON `json:"users_teacher_certificates"`
	UsersTeacherLinks            datatypes.JSON `json:"users_teacher_links"`
	UsersTeacherIsVerified       bool           `json:"users_teacher_is_verified"`
	UsersTeacherIsActive         bool           `json:"users_teacher_is_active"`
	UsersTeacherCreatedAt        string         `json:"users_teacher_created_at"`
	UsersTeacherUpdatedAt        string         `json:"users_teacher_updated_at"`
}

// Mapping model → response
func ToUsersTeacherResponse(m model.UserTeacher, fullName string) UsersTeacherResponse {
	return UsersTeacherResponse{
		UsersTeacherID:              m.UsersTeacherID,
		UsersTeacherUserID:          m.UsersTeacherUserID,
		FullName:                    fullName,
		UsersTeacherField:           deref(m.UsersTeacherField),
		UsersTeacherShortBio:        deref(m.UsersTeacherShortBio),
		UsersTeacherGreeting:        deref(m.UsersTeacherGreeting),
		UsersTeacherEducation:       deref(m.UsersTeacherEducation),
		UsersTeacherActivity:        deref(m.UsersTeacherActivity),
		UsersTeacherExperienceYears: m.UsersTeacherExperienceYears,
		UsersTeacherSpecialties:     m.UsersTeacherSpecialties,
		UsersTeacherCertificates:    m.UsersTeacherCertificates,
		UsersTeacherLinks:           m.UsersTeacherLinks,
		UsersTeacherIsVerified:      m.UsersTeacherIsVerified,
		UsersTeacherIsActive:        m.UsersTeacherIsActive,
		UsersTeacherCreatedAt:       m.UsersTeacherCreatedAt.Format(time.RFC3339),
		UsersTeacherUpdatedAt:       m.UsersTeacherUpdatedAt.Format(time.RFC3339),
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
