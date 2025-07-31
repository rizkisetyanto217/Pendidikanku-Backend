package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
	"time"
)

// ============================
// Response DTO
// ============================

type LectureSessionsMaterialDTO struct {
	LectureSessionsMaterialID               string    `json:"lecture_sessions_material_id"`
	LectureSessionsMaterialSummary          string    `json:"lecture_sessions_material_summary"`
	LectureSessionsMaterialTranscriptFull   string    `json:"lecture_sessions_material_transcript_full"`
	LectureSessionsMaterialLectureSessionID string    `json:"lecture_sessions_material_lecture_session_id"`
	LectureSessionsMaterialMasjidID         string    `json:"lecture_sessions_material_masjid_id"`
	LectureSessionsMaterialCreatedAt        time.Time `json:"lecture_sessions_material_created_at"`
}

// ============================
// Create Request DTO
// ============================

// dto/lecture_sessions_material_request.go
type CreateLectureSessionsMaterialRequest struct {
	LectureSessionsMaterialSummary         string `json:"lecture_sessions_material_summary"`
	LectureSessionsMaterialTranscriptFull  string `json:"lecture_sessions_material_transcript_full"`
	LectureSessionsMaterialLectureSessionID string `json:"lecture_sessions_material_lecture_session_id" validate:"required"`
	LectureSessionsMaterialMasjidID        string `json:"-"` // diisi manual dari token
}


// ============================
// Update Request DTO
// ============================

type UpdateLectureSessionsMaterialRequest struct {
	LectureSessionsMaterialSummary          string `json:"lecture_sessions_material_summary"`
	LectureSessionsMaterialTranscriptFull   string `json:"lecture_sessions_material_transcript_full"`
	LectureSessionsMaterialLectureSessionID string `json:"lecture_sessions_material_lecture_session_id"`
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
	}
}
