package dto

import (
	"masjidku_backend/internals/features/users/user/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ========== CREATE ==========
type CreateUsersTeacherRequest struct {
	UserID          uuid.UUID      `json:"users_teacher_user_id" validate:"required"`
	Field           string         `json:"users_teacher_field" validate:"omitempty,max=80"`
	ShortBio        string         `json:"users_teacher_short_bio" validate:"omitempty,max=300"`
	Greeting        string         `json:"users_teacher_greeting" validate:"omitempty"`
	Education       string         `json:"users_teacher_education" validate:"omitempty"`
	Activity        string         `json:"users_teacher_activity" validate:"omitempty"`
	ExperienceYears *int16         `json:"users_teacher_experience_years" validate:"omitempty"`
	Specialties     datatypes.JSON `json:"users_teacher_specialties" validate:"omitempty"`
	Certificates    datatypes.JSON `json:"users_teacher_certificates" validate:"omitempty"`
	Links           datatypes.JSON `json:"users_teacher_links" validate:"omitempty"`
	IsVerified      *bool          `json:"users_teacher_is_verified" validate:"omitempty"`
	IsActive        *bool          `json:"users_teacher_is_active" validate:"omitempty"`
}

// ========== UPDATE (partial) ==========
type UpdateUsersTeacherRequest struct {
	Field           *string         `json:"users_teacher_field" validate:"omitempty,max=80"`
	ShortBio        *string         `json:"users_teacher_short_bio" validate:"omitempty,max=300"`
	Greeting        *string         `json:"users_teacher_greeting" validate:"omitempty"`
	Education       *string         `json:"users_teacher_education" validate:"omitempty"`
	Activity        *string         `json:"users_teacher_activity" validate:"omitempty"`
	ExperienceYears *int16          `json:"users_teacher_experience_years" validate:"omitempty"`
	Specialties     *datatypes.JSON `json:"users_teacher_specialties" validate:"omitempty"`
	Certificates    *datatypes.JSON `json:"users_teacher_certificates" validate:"omitempty"`
	Links           *datatypes.JSON `json:"users_teacher_links" validate:"omitempty"`
	IsVerified      *bool           `json:"users_teacher_is_verified" validate:"omitempty"`
	IsActive        *bool           `json:"users_teacher_is_active" validate:"omitempty"`
}

// ========== RESPONSE ==========
type UsersTeacherResponse struct {
	ID              uuid.UUID      `json:"users_teacher_id"`
	UserID          uuid.UUID      `json:"users_teacher_user_id"`
	FullName        string         `json:"full_name"` // join dari users
	Field           string         `json:"users_teacher_field"`
	ShortBio        string         `json:"users_teacher_short_bio"`
	Greeting        string         `json:"users_teacher_greeting"`
	Education       string         `json:"users_teacher_education"`
	Activity        string         `json:"users_teacher_activity"`
	ExperienceYears *int16         `json:"users_teacher_experience_years"`
	Specialties     datatypes.JSON `json:"users_teacher_specialties"`
	Certificates    datatypes.JSON `json:"users_teacher_certificates"`
	Links           datatypes.JSON `json:"users_teacher_links"`
	IsVerified      bool           `json:"users_teacher_is_verified"`
	IsActive        bool           `json:"users_teacher_is_active"`
	CreatedAt       string         `json:"users_teacher_created_at"`
	UpdatedAt       string         `json:"users_teacher_updated_at"`
}

// ---------- helper convert ----------
func ToUsersTeacherResponse(m model.UsersTeacherModel, fullName string) UsersTeacherResponse {
	return UsersTeacherResponse{
		ID:              m.ID,
		UserID:          m.UserID,
		FullName:        fullName,
		Field:           m.Field,
		ShortBio:        m.ShortBio,
		Greeting:        m.Greeting,
		Education:       m.Education,
		Activity:        m.Activity,
		ExperienceYears: m.ExperienceYears,
		Specialties:     m.Specialties,
		Certificates:    m.Certificates,
		Links:           m.Links,
		IsVerified:      m.IsVerified,
		IsActive:        m.IsActive,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       m.UpdatedAt.Format(time.RFC3339),
	}
}
