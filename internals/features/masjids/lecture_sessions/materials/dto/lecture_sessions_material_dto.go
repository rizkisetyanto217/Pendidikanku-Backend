package dto

import (
	"time"

	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
)

// ============================
// Response DTO
// ============================

type LectureSessionsMaterialDTO struct {
	LectureSessionsMaterialID               string     `json:"lecture_sessions_material_id"`
	LectureSessionsMaterialSummary          *string    `json:"lecture_sessions_material_summary,omitempty"`
	LectureSessionsMaterialTranscriptFull   *string    `json:"lecture_sessions_material_transcript_full,omitempty"`
	LectureSessionsMaterialLectureSessionID string     `json:"lecture_sessions_material_lecture_session_id"`
	LectureSessionsMaterialMasjidID         string     `json:"lecture_sessions_material_masjid_id"`
	LectureSessionsMaterialCreatedAt        time.Time  `json:"lecture_sessions_material_created_at"`
	LectureSessionsMaterialUpdatedAt        time.Time  `json:"lecture_sessions_material_updated_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateLectureSessionsMaterialRequest struct {
	LectureSessionsMaterialSummary          *string `json:"lecture_sessions_material_summary,omitempty"`
	LectureSessionsMaterialTranscriptFull   *string `json:"lecture_sessions_material_transcript_full,omitempty"`
	LectureSessionsMaterialLectureSessionID string  `json:"lecture_sessions_material_lecture_session_id" validate:"required,uuid"`
	// MasjidID diambil dari token/scope pada controller
}

// Converter: Create -> Model (butuh masjidID dari controller)
func (r *CreateLectureSessionsMaterialRequest) ToModel(masjidID string) *model.LectureSessionsMaterialModel {
	return &model.LectureSessionsMaterialModel{
		LectureSessionsMaterialSummary:          r.LectureSessionsMaterialSummary,
		LectureSessionsMaterialTranscriptFull:   r.LectureSessionsMaterialTranscriptFull,
		LectureSessionsMaterialLectureSessionID: r.LectureSessionsMaterialLectureSessionID,
		LectureSessionsMaterialMasjidID:         masjidID,
	}
}

// ============================
// Update Request DTO (partial)
// ============================

type UpdateLectureSessionsMaterialRequest struct {
	LectureSessionsMaterialSummary          *string `json:"lecture_sessions_material_summary,omitempty"`         // set null untuk clear
	LectureSessionsMaterialTranscriptFull   *string `json:"lecture_sessions_material_transcript_full,omitempty"` // set null untuk clear
	// ID session tidak diubah lewat update normal (hindari migrasi data)
}

// ============================
// Converter
// ============================

func ToLectureSessionsMaterialDTO(m model.LectureSessionsMaterialModel) LectureSessionsMaterialDTO {
	return LectureSessionsMaterialDTO{
		LectureSessionsMaterialID:               m.LectureSessionsMaterialID,
		LectureSessionsMaterialSummary:          m.LectureSessionsMaterialSummary,
		LectureSessionsMaterialTranscriptFull:   m.LectureSessionsMaterialTranscriptFull,
		LectureSessionsMaterialLectureSessionID: m.LectureSessionsMaterialLectureSessionID,
		LectureSessionsMaterialMasjidID:         m.LectureSessionsMaterialMasjidID,
		LectureSessionsMaterialCreatedAt:        m.LectureSessionsMaterialCreatedAt,
		LectureSessionsMaterialUpdatedAt:        m.LectureSessionsMaterialUpdatedAt,
	}
}
