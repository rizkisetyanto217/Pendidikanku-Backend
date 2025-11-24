package dto

import (
	"time"

	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools_more/model"

	"github.com/google/uuid"
)

// =============================
// REQUEST
// =============================
type SchoolProfileTeacherDkmRequest struct {
	SchoolProfileTeacherDkmSchoolID uuid.UUID  `json:"school_profile_teacher_dkm_school_id" validate:"required"`
	SchoolProfileTeacherDkmUserID   *uuid.UUID `json:"school_profile_teacher_dkm_user_id,omitempty"`
	SchoolProfileTeacherDkmName     string     `json:"school_profile_teacher_dkm_name" validate:"required"`
	SchoolProfileTeacherDkmRole     string     `json:"school_profile_teacher_dkm_role" validate:"required"`
	// opsional
	SchoolProfileTeacherDkmDescription *string `json:"school_profile_teacher_dkm_description,omitempty"`
	SchoolProfileTeacherDkmMessage     *string `json:"school_profile_teacher_dkm_message,omitempty"`
	SchoolProfileTeacherDkmImageURL    *string `json:"school_profile_teacher_dkm_image_url,omitempty"`
}

type GetProfilesBySchoolRequest struct {
	SchoolProfileTeacherDkmSchoolID string `json:"school_profile_teacher_dkm_school_id" validate:"required,uuid"`
}

// =============================
// RESPONSE
// =============================
type SchoolProfileTeacherDkmResponse struct {
	SchoolProfileTeacherDkmID       uuid.UUID  `json:"school_profile_teacher_dkm_id"`
	SchoolProfileTeacherDkmSchoolID uuid.UUID  `json:"school_profile_teacher_dkm_school_id"`
	SchoolProfileTeacherDkmUserID   *uuid.UUID `json:"school_profile_teacher_dkm_user_id,omitempty"`

	SchoolProfileTeacherDkmName        string  `json:"school_profile_teacher_dkm_name"`
	SchoolProfileTeacherDkmRole        string  `json:"school_profile_teacher_dkm_role"`
	SchoolProfileTeacherDkmDescription *string `json:"school_profile_teacher_dkm_description,omitempty"`
	SchoolProfileTeacherDkmMessage     *string `json:"school_profile_teacher_dkm_message,omitempty"`
	SchoolProfileTeacherDkmImageURL    *string `json:"school_profile_teacher_dkm_image_url,omitempty"`

	SchoolProfileTeacherDkmCreatedAt time.Time `json:"school_profile_teacher_dkm_created_at"`
	SchoolProfileTeacherDkmUpdatedAt time.Time `json:"school_profile_teacher_dkm_updated_at"`
}

// =============================
// MAPPER
// =============================
func ToResponse(m *model.SchoolProfileTeacherDkmModel) SchoolProfileTeacherDkmResponse {
	return SchoolProfileTeacherDkmResponse{
		SchoolProfileTeacherDkmID:          m.SchoolProfileTeacherDkmID,
		SchoolProfileTeacherDkmSchoolID:    m.SchoolProfileTeacherDkmSchoolID,
		SchoolProfileTeacherDkmUserID:      m.SchoolProfileTeacherDkmUserID,
		SchoolProfileTeacherDkmName:        m.SchoolProfileTeacherDkmName,
		SchoolProfileTeacherDkmRole:        m.SchoolProfileTeacherDkmRole,
		SchoolProfileTeacherDkmDescription: m.SchoolProfileTeacherDkmDescription,
		SchoolProfileTeacherDkmMessage:     m.SchoolProfileTeacherDkmMessage,
		SchoolProfileTeacherDkmImageURL:    m.SchoolProfileTeacherDkmImageURL,
		SchoolProfileTeacherDkmCreatedAt:   m.SchoolProfileTeacherDkmCreatedAt,
		SchoolProfileTeacherDkmUpdatedAt:   m.SchoolProfileTeacherDkmUpdatedAt,
	}
}

func (r *SchoolProfileTeacherDkmRequest) ToModel() *model.SchoolProfileTeacherDkmModel {
	return &model.SchoolProfileTeacherDkmModel{
		SchoolProfileTeacherDkmSchoolID:    r.SchoolProfileTeacherDkmSchoolID,
		SchoolProfileTeacherDkmUserID:      r.SchoolProfileTeacherDkmUserID,
		SchoolProfileTeacherDkmName:        r.SchoolProfileTeacherDkmName,
		SchoolProfileTeacherDkmRole:        r.SchoolProfileTeacherDkmRole,
		SchoolProfileTeacherDkmDescription: r.SchoolProfileTeacherDkmDescription,
		SchoolProfileTeacherDkmMessage:     r.SchoolProfileTeacherDkmMessage,
		SchoolProfileTeacherDkmImageURL:    r.SchoolProfileTeacherDkmImageURL,
	}
}
