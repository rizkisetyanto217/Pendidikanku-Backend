package dto

import (
	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/google/uuid"
)

type MasjidTagRequest struct {
	MasjidTagName        string `json:"masjid_tag_name"`
	MasjidTagDescription string `json:"masjid_tag_description"`
}

type MasjidTagResponse struct {
	MasjidTagID          uuid.UUID `json:"masjid_tag_id"`
	MasjidTagName        string    `json:"masjid_tag_name"`
	MasjidTagDescription string    `json:"masjid_tag_description"`
	MasjidTagCreatedAt   string    `json:"masjid_tag_created_at"`
}

// Convert request → model
func (r *MasjidTagRequest) ToModel() *model.MasjidTagModel {
	return &model.MasjidTagModel{
		MasjidTagName:        r.MasjidTagName,
		MasjidTagDescription: r.MasjidTagDescription,
	}
}

// Convert model → response
func ToMasjidTagResponse(m *model.MasjidTagModel) *MasjidTagResponse {
	return &MasjidTagResponse{
		MasjidTagID:          m.MasjidTagID,
		MasjidTagName:        m.MasjidTagName,
		MasjidTagDescription: m.MasjidTagDescription,
		MasjidTagCreatedAt:   m.MasjidTagCreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// Convert slice model → slice response
func ToMasjidTagResponseList(models []model.MasjidTagModel) []MasjidTagResponse {
	var result []MasjidTagResponse
	for _, m := range models {
		result = append(result, *ToMasjidTagResponse(&m))
	}
	return result
}
