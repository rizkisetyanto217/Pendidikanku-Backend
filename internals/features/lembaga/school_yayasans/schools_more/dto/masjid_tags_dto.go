package dto

import (
	"time"

	"schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/model"

	"github.com/google/uuid"
)

/*
======================

	REQUEST (Create)
	======================
*/
type SchoolTagRequest struct {
	SchoolTagName        string  `json:"school_tag_name" validate:"required,max=50"`
	SchoolTagDescription *string `json:"school_tag_description,omitempty"` // opsional
}

func (r *SchoolTagRequest) ToModel() *model.SchoolTagModel {
	return &model.SchoolTagModel{
		SchoolTagName:        r.SchoolTagName,
		SchoolTagDescription: r.SchoolTagDescription,
	}
}

/*
======================

	REQUEST (Partial Update)
	- Pakai pointer agar nil = tidak diubah
	======================
*/
type SchoolTagUpdateRequest struct {
	SchoolTagName        *string `json:"school_tag_name,omitempty" validate:"omitempty,max=50"`
	SchoolTagDescription *string `json:"school_tag_description,omitempty"`
}

/*
======================

	RESPONSE
	======================
*/
type SchoolTagResponse struct {
	SchoolTagID          uuid.UUID `json:"school_tag_id"`
	SchoolTagName        string    `json:"school_tag_name"`
	SchoolTagDescription string    `json:"school_tag_description"`
	SchoolTagCreatedAt   string    `json:"school_tag_created_at"` // formatted
}

// model -> response
func ToSchoolTagResponse(m *model.SchoolTagModel) *SchoolTagResponse {
	desc := ""
	if m.SchoolTagDescription != nil {
		desc = *m.SchoolTagDescription
	}
	return &SchoolTagResponse{
		SchoolTagID:          m.SchoolTagID,
		SchoolTagName:        m.SchoolTagName,
		SchoolTagDescription: desc,
		SchoolTagCreatedAt:   m.SchoolTagCreatedAt.Format(time.RFC3339), // atau "2006-01-02 15:04:05"
	}
}

// slice model -> slice response
func ToSchoolTagResponseList(models []model.SchoolTagModel) []SchoolTagResponse {
	out := make([]SchoolTagResponse, 0, len(models))
	for i := range models {
		out = append(out, *ToSchoolTagResponse(&models[i]))
	}
	return out
}
