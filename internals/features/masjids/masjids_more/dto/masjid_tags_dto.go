package dto

import (
	"time"

	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/google/uuid"
)

/* ======================
   REQUEST (Create)
   ====================== */
type MasjidTagRequest struct {
	MasjidTagName        string  `json:"masjid_tag_name" validate:"required,max=50"`
	MasjidTagDescription *string `json:"masjid_tag_description,omitempty"` // opsional
}

func (r *MasjidTagRequest) ToModel() *model.MasjidTagModel {
	return &model.MasjidTagModel{
		MasjidTagName:        r.MasjidTagName,
		MasjidTagDescription: r.MasjidTagDescription,
	}
}

/* ======================
   REQUEST (Partial Update)
   - Pakai pointer agar nil = tidak diubah
   ====================== */
type MasjidTagUpdateRequest struct {
	MasjidTagName        *string `json:"masjid_tag_name,omitempty" validate:"omitempty,max=50"`
	MasjidTagDescription *string `json:"masjid_tag_description,omitempty"`
}

/* ======================
   RESPONSE
   ====================== */
type MasjidTagResponse struct {
	MasjidTagID          uuid.UUID `json:"masjid_tag_id"`
	MasjidTagName        string    `json:"masjid_tag_name"`
	MasjidTagDescription string    `json:"masjid_tag_description"`
	MasjidTagCreatedAt   string    `json:"masjid_tag_created_at"` // formatted
}

// model -> response
func ToMasjidTagResponse(m *model.MasjidTagModel) *MasjidTagResponse {
	desc := ""
	if m.MasjidTagDescription != nil {
		desc = *m.MasjidTagDescription
	}
	return &MasjidTagResponse{
		MasjidTagID:          m.MasjidTagID,
		MasjidTagName:        m.MasjidTagName,
		MasjidTagDescription: desc,
		MasjidTagCreatedAt:   m.MasjidTagCreatedAt.Format(time.RFC3339), // atau "2006-01-02 15:04:05"
	}
}

// slice model -> slice response
func ToMasjidTagResponseList(models []model.MasjidTagModel) []MasjidTagResponse {
	out := make([]MasjidTagResponse, 0, len(models))
	for i := range models {
		out = append(out, *ToMasjidTagResponse(&models[i]))
	}
	return out
}
