package dto

import (
	"time"

	"masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids_more/model"

	"github.com/google/uuid"
)

const timeLayout = time.RFC3339 // contoh: 2025-08-21T14:03:07Z
func fmtTS(t time.Time) string  { return t.UTC().Format(timeLayout) }

// ==============================
// Request
// ==============================

type MasjidTagRelationRequest struct {
	MasjidTagRelationMasjidID uuid.UUID `json:"masjid_tag_relation_masjid_id"`
	MasjidTagRelationTagID    uuid.UUID `json:"masjid_tag_relation_tag_id"`
}

func (r *MasjidTagRelationRequest) ToModel() *model.MasjidTagRelationModel {
	return &model.MasjidTagRelationModel{
		MasjidTagRelationMasjidID: r.MasjidTagRelationMasjidID,
		MasjidTagRelationTagID:    r.MasjidTagRelationTagID,
	}
}

// ==============================
// Response (simple)
// ==============================

type MasjidTagRelationResponse struct {
	MasjidTagRelationID        uuid.UUID `json:"masjid_tag_relation_id"`
	MasjidTagRelationMasjidID  uuid.UUID `json:"masjid_tag_relation_masjid_id"`
	MasjidTagRelationTagID     uuid.UUID `json:"masjid_tag_relation_tag_id"`
	MasjidTagRelationCreatedAt string    `json:"masjid_tag_relation_created_at"`
}

func ToMasjidTagRelationResponse(m *model.MasjidTagRelationModel) *MasjidTagRelationResponse {
	if m == nil {
		return &MasjidTagRelationResponse{}
	}
	return &MasjidTagRelationResponse{
		MasjidTagRelationID:        m.MasjidTagRelationID,
		MasjidTagRelationMasjidID:  m.MasjidTagRelationMasjidID,
		MasjidTagRelationTagID:     m.MasjidTagRelationTagID,
		MasjidTagRelationCreatedAt: fmtTS(m.MasjidTagRelationCreatedAt),
	}
}

// ==============================
// Response (full, dengan preload)
// ==============================

type MasjidTagRelationFullResponse struct {
	MasjidTagRelationID        uuid.UUID      `json:"masjid_tag_relation_id"`
	MasjidTagRelationMasjidID  uuid.UUID      `json:"masjid_tag_relation_masjid_id"`
	Masjid                     *MiniMasjidDTO `json:"masjid,omitempty"`
	MasjidTagRelationTagID     uuid.UUID      `json:"masjid_tag_relation_tag_id"`
	Tag                        *MiniTagDTO    `json:"tag,omitempty"`
	MasjidTagRelationCreatedAt string         `json:"masjid_tag_relation_created_at"`
}

type MiniMasjidDTO struct {
	MasjidID   uuid.UUID `json:"masjid_id"`
	MasjidName string    `json:"masjid_name"`
}

type MiniTagDTO struct {
	MasjidTagID   uuid.UUID `json:"masjid_tag_id"`
	MasjidTagName string    `json:"masjid_tag_name"`
}

func ToMasjidTagRelationFullResponse(m *model.MasjidTagRelationModel) *MasjidTagRelationFullResponse {
	if m == nil {
		return &MasjidTagRelationFullResponse{}
	}

	var masjidDTO *MiniMasjidDTO
	// Masjid adalah struct (bukan pointer) → cek ID/nama
	if m.Masjid.MasjidID != uuid.Nil || m.Masjid.MasjidName != "" {
		masjidDTO = &MiniMasjidDTO{
			MasjidID:   m.Masjid.MasjidID,
			MasjidName: m.Masjid.MasjidName,
		}
	}

	var tagDTO *MiniTagDTO
	// MasjidTag juga struct → cek ID/nama
	if m.MasjidTag.MasjidTagID != uuid.Nil || m.MasjidTag.MasjidTagName != "" {
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
		MasjidTagRelationCreatedAt: m.MasjidTagRelationCreatedAt.UTC().Format(time.RFC3339),
	}
}


// ==============================
// List converters (efisien, tanpa pointer-range bug)
// ==============================

func ToMasjidTagRelationResponseList(models []model.MasjidTagRelationModel) []MasjidTagRelationResponse {
	out := make([]MasjidTagRelationResponse, len(models))
	for i := range models {
		out[i] = *ToMasjidTagRelationResponse(&models[i])
	}
	return out
}

func ToMasjidTagRelationFullResponseList(models []model.MasjidTagRelationModel) []MasjidTagRelationFullResponse {
	out := make([]MasjidTagRelationFullResponse, len(models))
	for i := range models {
		out[i] = *ToMasjidTagRelationFullResponse(&models[i])
	}
	return out
}
