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
	LectureSessionsMaterialTitle            string    `json:"lecture_sessions_material_title"`
	LectureSessionsMaterialSummary          string    `json:"lecture_sessions_material_summary"`
	LectureSessionsMaterialTranscriptFull   string    `json:"lecture_sessions_material_transcript_full"`
	LectureSessionsMaterialLectureSessionID string    `json:"lecture_sessions_material_lecture_session_id"`
	LectureSessionsMaterialCreatedAt        time.Time `json:"lecture_sessions_material_created_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateLectureSessionsMaterialRequest struct {
	LectureSessionsMaterialTitle            string `json:"lecture_sessions_material_title" validate:"required,min=3"`
	LectureSessionsMaterialSummary          string `json:"lecture_sessions_material_summary" validate:"required"`
	LectureSessionsMaterialTranscriptFull   string `json:"lecture_sessions_material_transcript_full" validate:"required"`
	LectureSessionsMaterialLectureSessionID string `json:"lecture_sessions_material_lecture_session_id" validate:"required,uuid"`
}

// ============================
// Converter
// ============================

func ToLectureSessionsMaterialDTO(m model.LectureSessionsMaterialModel) LectureSessionsMaterialDTO {
	return LectureSessionsMaterialDTO{
		LectureSessionsMaterialID:               m.LectureSessionsMaterialID,
		LectureSessionsMaterialTitle:            m.LectureSessionsMaterialTitle,
		LectureSessionsMaterialSummary:          m.LectureSessionsMaterialSummary,
		LectureSessionsMaterialTranscriptFull:   m.LectureSessionsMaterialTranscriptFull,
		LectureSessionsMaterialLectureSessionID: m.LectureSessionsMaterialLectureSessionID,
		LectureSessionsMaterialCreatedAt:        m.LectureSessionsMaterialCreatedAt,
	}
}
