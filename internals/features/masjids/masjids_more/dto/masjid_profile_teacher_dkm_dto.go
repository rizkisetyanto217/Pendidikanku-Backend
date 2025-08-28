package dto

import (
	"time"

	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/google/uuid"
)

// =============================
// REQUEST
// =============================
type MasjidProfileTeacherDkmRequest struct {
	MasjidProfileTeacherDkmMasjidID uuid.UUID  `json:"masjid_profile_teacher_dkm_masjid_id" validate:"required"`
	MasjidProfileTeacherDkmUserID   *uuid.UUID `json:"masjid_profile_teacher_dkm_user_id,omitempty"`
	MasjidProfileTeacherDkmName     string     `json:"masjid_profile_teacher_dkm_name" validate:"required"`
	MasjidProfileTeacherDkmRole     string     `json:"masjid_profile_teacher_dkm_role" validate:"required"`
	// opsional
	MasjidProfileTeacherDkmDescription *string `json:"masjid_profile_teacher_dkm_description,omitempty"`
	MasjidProfileTeacherDkmMessage     *string `json:"masjid_profile_teacher_dkm_message,omitempty"`
	MasjidProfileTeacherDkmImageURL    *string `json:"masjid_profile_teacher_dkm_image_url,omitempty"`
}

type GetProfilesByMasjidRequest struct {
	MasjidProfileTeacherDkmMasjidID string `json:"masjid_profile_teacher_dkm_masjid_id" validate:"required,uuid"`
}

// =============================
// RESPONSE
// =============================
type MasjidProfileTeacherDkmResponse struct {
	MasjidProfileTeacherDkmID       uuid.UUID  `json:"masjid_profile_teacher_dkm_id"`
	MasjidProfileTeacherDkmMasjidID uuid.UUID  `json:"masjid_profile_teacher_dkm_masjid_id"`
	MasjidProfileTeacherDkmUserID   *uuid.UUID `json:"masjid_profile_teacher_dkm_user_id,omitempty"`

	MasjidProfileTeacherDkmName     string  `json:"masjid_profile_teacher_dkm_name"`
	MasjidProfileTeacherDkmRole     string  `json:"masjid_profile_teacher_dkm_role"`
	MasjidProfileTeacherDkmDescription *string `json:"masjid_profile_teacher_dkm_description,omitempty"`
	MasjidProfileTeacherDkmMessage     *string `json:"masjid_profile_teacher_dkm_message,omitempty"`
	MasjidProfileTeacherDkmImageURL    *string `json:"masjid_profile_teacher_dkm_image_url,omitempty"`

	MasjidProfileTeacherDkmCreatedAt time.Time `json:"masjid_profile_teacher_dkm_created_at"`
	MasjidProfileTeacherDkmUpdatedAt time.Time `json:"masjid_profile_teacher_dkm_updated_at"`
}

// =============================
// MAPPER
// =============================
func ToResponse(m *model.MasjidProfileTeacherDkmModel) MasjidProfileTeacherDkmResponse {
	return MasjidProfileTeacherDkmResponse{
		MasjidProfileTeacherDkmID:          m.MasjidProfileTeacherDkmID,
		MasjidProfileTeacherDkmMasjidID:    m.MasjidProfileTeacherDkmMasjidID,
		MasjidProfileTeacherDkmUserID:      m.MasjidProfileTeacherDkmUserID,
		MasjidProfileTeacherDkmName:        m.MasjidProfileTeacherDkmName,
		MasjidProfileTeacherDkmRole:        m.MasjidProfileTeacherDkmRole,
		MasjidProfileTeacherDkmDescription: m.MasjidProfileTeacherDkmDescription,
		MasjidProfileTeacherDkmMessage:     m.MasjidProfileTeacherDkmMessage,
		MasjidProfileTeacherDkmImageURL:    m.MasjidProfileTeacherDkmImageURL,
		MasjidProfileTeacherDkmCreatedAt:   m.MasjidProfileTeacherDkmCreatedAt,
		MasjidProfileTeacherDkmUpdatedAt:   m.MasjidProfileTeacherDkmUpdatedAt,
	}
}

func (r *MasjidProfileTeacherDkmRequest) ToModel() *model.MasjidProfileTeacherDkmModel {
	return &model.MasjidProfileTeacherDkmModel{
		MasjidProfileTeacherDkmMasjidID:    r.MasjidProfileTeacherDkmMasjidID,
		MasjidProfileTeacherDkmUserID:      r.MasjidProfileTeacherDkmUserID,
		MasjidProfileTeacherDkmName:        r.MasjidProfileTeacherDkmName,
		MasjidProfileTeacherDkmRole:        r.MasjidProfileTeacherDkmRole,
		MasjidProfileTeacherDkmDescription: r.MasjidProfileTeacherDkmDescription,
		MasjidProfileTeacherDkmMessage:     r.MasjidProfileTeacherDkmMessage,
		MasjidProfileTeacherDkmImageURL:    r.MasjidProfileTeacherDkmImageURL,
	}
}
