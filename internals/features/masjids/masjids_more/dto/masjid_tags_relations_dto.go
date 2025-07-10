package dto

import (
	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/google/uuid"
)

// 🟩 Input request untuk membuat relasi tag
type MasjidTagRelationRequest struct {
	MasjidTagRelationMasjidID uuid.UUID `json:"masjid_tag_relation_masjid_id"`
	MasjidTagRelationTagID    uuid.UUID `json:"masjid_tag_relation_tag_id"`
}

// 🟦 Response sederhana tanpa preload
type MasjidTagRelationResponse struct {
	MasjidTagRelationID        uuid.UUID `json:"masjid_tag_relation_id"`
	MasjidTagRelationMasjidID  uuid.UUID `json:"masjid_tag_relation_masjid_id"`
	MasjidTagRelationTagID     uuid.UUID `json:"masjid_tag_relation_tag_id"`
	MasjidTagRelationCreatedAt string    `json:"masjid_tag_relation_created_at"`
}

// 🟨 Response lengkap dengan relasi Masjid dan Tag (Preload)
type MasjidTagRelationFullResponse struct {
	MasjidTagRelationID        uuid.UUID      `json:"masjid_tag_relation_id"`
	MasjidTagRelationMasjidID  uuid.UUID      `json:"masjid_tag_relation_masjid_id"`
	Masjid                     *MiniMasjidDTO `json:"masjid,omitempty"`
	MasjidTagRelationTagID     uuid.UUID      `json:"masjid_tag_relation_tag_id"`
	Tag                        *MiniTagDTO    `json:"tag,omitempty"`
	MasjidTagRelationCreatedAt string         `json:"masjid_tag_relation_created_at"`
}

// 🔹 Miniatur Masjid (hanya ID dan Name)
type MiniMasjidDTO struct {
	MasjidID   uuid.UUID `json:"masjid_id"`
	MasjidName string    `json:"masjid_name"`
}

// 🔹 Miniatur Tag (hanya ID dan Name)
type MiniTagDTO struct {
	MasjidTagID   uuid.UUID `json:"masjid_tag_id"`
	MasjidTagName string    `json:"masjid_tag_name"`
}

// 🔁 Convert request → model
func (r *MasjidTagRelationRequest) ToModel() *model.MasjidTagRelationModel {
	return &model.MasjidTagRelationModel{
		MasjidTagRelationMasjidID: r.MasjidTagRelationMasjidID,
		MasjidTagRelationTagID:    r.MasjidTagRelationTagID,
	}
}

// 🔁 Convert model → response biasa
func ToMasjidTagRelationResponse(m *model.MasjidTagRelationModel) *MasjidTagRelationResponse {
	return &MasjidTagRelationResponse{
		MasjidTagRelationID:        m.MasjidTagRelationID,
		MasjidTagRelationMasjidID:  m.MasjidTagRelationMasjidID,
		MasjidTagRelationTagID:     m.MasjidTagRelationTagID,
		MasjidTagRelationCreatedAt: m.MasjidTagRelationCreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// 🔁 Convert model → response dengan relasi masjid dan tag
func ToMasjidTagRelationFullResponse(m *model.MasjidTagRelationModel) *MasjidTagRelationFullResponse {
	var masjidDTO *MiniMasjidDTO
	if m.Masjid.MasjidID != uuid.Nil {
		masjidDTO = &MiniMasjidDTO{
			MasjidID:   m.Masjid.MasjidID,
			MasjidName: m.Masjid.MasjidName,
		}
	}

	var tagDTO *MiniTagDTO
	if m.MasjidTag != nil {
		tagDTO = &MiniTagDTO{
			MasjidTagID:   m.MasjidTag.MasjidTagID,
			MasjidTagName: m.MasjidTag.MasjidTagName,
		}
	}

	return &MasjidTagRelationFullResponse{
		MasjidTagRelationID:        m.MasjidTagRelationID,
		MasjidTagRelationMasjidID:  m.MasjidTagRelationMasjidID,
		Masjid:                     masjidDTO,
		MasjidTagRelationTagID:     m.MasjidTagRelationTagID,
		Tag:                        tagDTO,
		MasjidTagRelationCreatedAt: m.MasjidTagRelationCreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// 🔁 Convert slice → response biasa
func ToMasjidTagRelationResponseList(models []model.MasjidTagRelationModel) []MasjidTagRelationResponse {
	var result []MasjidTagRelationResponse
	for _, m := range models {
		result = append(result, *ToMasjidTagRelationResponse(&m))
	}
	return result
}

// 🔁 Convert slice → response full (dengan preload)
func ToMasjidTagRelationFullResponseList(models []model.MasjidTagRelationModel) []MasjidTagRelationFullResponse {
	var result []MasjidTagRelationFullResponse
	for _, m := range models {
		result = append(result, *ToMasjidTagRelationFullResponse(&m))
	}
	return result
}
