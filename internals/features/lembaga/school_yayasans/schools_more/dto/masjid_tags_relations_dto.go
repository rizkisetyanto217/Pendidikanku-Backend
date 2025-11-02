package dto

import (
	"time"

	"schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/model"

	"github.com/google/uuid"
)

const timeLayout = time.RFC3339 // contoh: 2025-08-21T14:03:07Z
func fmtTS(t time.Time) string  { return t.UTC().Format(timeLayout) }

// ==============================
// Request
// ==============================

type SchoolTagRelationRequest struct {
	SchoolTagRelationSchoolID uuid.UUID `json:"school_tag_relation_school_id"`
	SchoolTagRelationTagID    uuid.UUID `json:"school_tag_relation_tag_id"`
}

func (r *SchoolTagRelationRequest) ToModel() *model.SchoolTagRelationModel {
	return &model.SchoolTagRelationModel{
		SchoolTagRelationSchoolID: r.SchoolTagRelationSchoolID,
		SchoolTagRelationTagID:    r.SchoolTagRelationTagID,
	}
}

// ==============================
// Response (simple)
// ==============================

type SchoolTagRelationResponse struct {
	SchoolTagRelationID        uuid.UUID `json:"school_tag_relation_id"`
	SchoolTagRelationSchoolID  uuid.UUID `json:"school_tag_relation_school_id"`
	SchoolTagRelationTagID     uuid.UUID `json:"school_tag_relation_tag_id"`
	SchoolTagRelationCreatedAt string    `json:"school_tag_relation_created_at"`
}

func ToSchoolTagRelationResponse(m *model.SchoolTagRelationModel) *SchoolTagRelationResponse {
	if m == nil {
		return &SchoolTagRelationResponse{}
	}
	return &SchoolTagRelationResponse{
		SchoolTagRelationID:        m.SchoolTagRelationID,
		SchoolTagRelationSchoolID:  m.SchoolTagRelationSchoolID,
		SchoolTagRelationTagID:     m.SchoolTagRelationTagID,
		SchoolTagRelationCreatedAt: fmtTS(m.SchoolTagRelationCreatedAt),
	}
}

// ==============================
// Response (full, dengan preload)
// ==============================

type SchoolTagRelationFullResponse struct {
	SchoolTagRelationID        uuid.UUID      `json:"school_tag_relation_id"`
	SchoolTagRelationSchoolID  uuid.UUID      `json:"school_tag_relation_school_id"`
	School                     *MiniSchoolDTO `json:"school,omitempty"`
	SchoolTagRelationTagID     uuid.UUID      `json:"school_tag_relation_tag_id"`
	Tag                        *MiniTagDTO    `json:"tag,omitempty"`
	SchoolTagRelationCreatedAt string         `json:"school_tag_relation_created_at"`
}

type MiniSchoolDTO struct {
	SchoolID   uuid.UUID `json:"school_id"`
	SchoolName string    `json:"school_name"`
}

type MiniTagDTO struct {
	SchoolTagID   uuid.UUID `json:"school_tag_id"`
	SchoolTagName string    `json:"school_tag_name"`
}

func ToSchoolTagRelationFullResponse(m *model.SchoolTagRelationModel) *SchoolTagRelationFullResponse {
	if m == nil {
		return &SchoolTagRelationFullResponse{}
	}

	var schoolDTO *MiniSchoolDTO
	// School adalah struct (bukan pointer) → cek ID/nama
	if m.School.SchoolID != uuid.Nil || m.School.SchoolName != "" {
		schoolDTO = &MiniSchoolDTO{
			SchoolID:   m.School.SchoolID,
			SchoolName: m.School.SchoolName,
		}
	}

	var tagDTO *MiniTagDTO
	// SchoolTag juga struct → cek ID/nama
	if m.SchoolTag.SchoolTagID != uuid.Nil || m.SchoolTag.SchoolTagName != "" {
		tagDTO = &MiniTagDTO{
			SchoolTagID:   m.SchoolTag.SchoolTagID,
			SchoolTagName: m.SchoolTag.SchoolTagName,
		}
	}

	return &SchoolTagRelationFullResponse{
		SchoolTagRelationID:        m.SchoolTagRelationID,
		SchoolTagRelationSchoolID:  m.SchoolTagRelationSchoolID,
		School:                     schoolDTO,
		SchoolTagRelationTagID:     m.SchoolTagRelationTagID,
		Tag:                        tagDTO,
		SchoolTagRelationCreatedAt: m.SchoolTagRelationCreatedAt.UTC().Format(time.RFC3339),
	}
}

// ==============================
// List converters (efisien, tanpa pointer-range bug)
// ==============================

func ToSchoolTagRelationResponseList(models []model.SchoolTagRelationModel) []SchoolTagRelationResponse {
	out := make([]SchoolTagRelationResponse, len(models))
	for i := range models {
		out[i] = *ToSchoolTagRelationResponse(&models[i])
	}
	return out
}

func ToSchoolTagRelationFullResponseList(models []model.SchoolTagRelationModel) []SchoolTagRelationFullResponse {
	out := make([]SchoolTagRelationFullResponse, len(models))
	for i := range models {
		out[i] = *ToSchoolTagRelationFullResponse(&models[i])
	}
	return out
}
