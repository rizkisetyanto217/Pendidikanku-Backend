package dto

import (
	"masjidku_backend/internals/features/masjids/masjids_more/model"
	"time"

	"github.com/google/uuid"
)

type MasjidProfileTeacherDkmRequest struct {
	MasjidProfileTeacherDkmMasjidID    uuid.UUID  `json:"masjid_profile_teacher_dkm_masjid_id"`
	MasjidProfileTeacherDkmUserID      *uuid.UUID `json:"masjid_profile_teacher_dkm_user_id,omitempty"`
	MasjidProfileTeacherDkmName        string     `json:"masjid_profile_teacher_dkm_name"`
	MasjidProfileTeacherDkmRole        string     `json:"masjid_profile_teacher_dkm_role"`
	MasjidProfileTeacherDkmDescription string     `json:"masjid_profile_teacher_dkm_description"`
	MasjidProfileTeacherDkmMessage     string     `json:"masjid_profile_teacher_dkm_message"`
	MasjidProfileTeacherDkmImageURL    string     `json:"masjid_profile_teacher_dkm_image_url"`
}

type GetProfilesByMasjidRequest struct {
	MasjidProfileTeacherDkmMasjidID string `json:"masjid_profile_teacher_dkm_masjid_id"`
}
type MasjidProfileTeacherDkmResponse struct {
	MasjidProfileTeacherDkmID            uuid.UUID  `json:"masjid_profile_teacher_dkm_id"`
	MasjidProfileTeacherDkmMasjidID      uuid.UUID  `json:"masjid_profile_teacher_dkm_masjid_id"`
	MasjidProfileTeacherDkmUserID        *uuid.UUID `json:"masjid_profile_teacher_dkm_user_id,omitempty"`
	MasjidProfileTeacherDkmName          string     `json:"masjid_profile_teacher_dkm_name"`
	MasjidProfileTeacherDkmRole          string     `json:"masjid_profile_teacher_dkm_role"`
	MasjidProfileTeacherDkmDescription   string     `json:"masjid_profile_teacher_dkm_description"`
	MasjidProfileTeacherDkmMessage       string     `json:"masjid_profile_teacher_dkm_message"`
	MasjidProfileTeacherDkmImageURL      string     `json:"masjid_profile_teacher_dkm_image_url"`
	MasjidProfileTeacherDkmCreatedAt     time.Time  `json:"masjid_profile_teacher_dkm_created_at"`
}


func ToResponse(model *model.MasjidProfileTeacherDkmModel) MasjidProfileTeacherDkmResponse {
	return MasjidProfileTeacherDkmResponse{
		MasjidProfileTeacherDkmID:            model.MasjidProfileTeacherDkmID,
		MasjidProfileTeacherDkmMasjidID:      model.MasjidProfileTeacherDkmMasjidID,
		MasjidProfileTeacherDkmUserID:        model.MasjidProfileTeacherDkmUserID,
		MasjidProfileTeacherDkmName:          model.MasjidProfileTeacherDkmName,
		MasjidProfileTeacherDkmRole:          model.MasjidProfileTeacherDkmRole,
		MasjidProfileTeacherDkmDescription:   model.MasjidProfileTeacherDkmDescription,
		MasjidProfileTeacherDkmMessage:       model.MasjidProfileTeacherDkmMessage,
		MasjidProfileTeacherDkmImageURL:      model.MasjidProfileTeacherDkmImageURL,
		MasjidProfileTeacherDkmCreatedAt:     model.MasjidProfileTeacherDkmCreatedAt,
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
