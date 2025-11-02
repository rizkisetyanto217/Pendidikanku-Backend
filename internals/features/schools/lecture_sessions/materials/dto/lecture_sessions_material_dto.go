package dto

import (
	"time"

	"schoolku_backend/internals/features/schools/lecture_sessions/materials/model"
)

// ============================
// Response DTO
// ============================

type LectureSessionsMaterialDTO struct {
	LectureSessionsMaterialID               string    `json:"lecture_sessions_material_id"`
	LectureSessionsMaterialSummary          *string   `json:"lecture_sessions_material_summary,omitempty"`
	LectureSessionsMaterialTranscriptFull   *string   `json:"lecture_sessions_material_transcript_full,omitempty"`
	LectureSessionsMaterialLectureSessionID string    `json:"lecture_sessions_material_lecture_session_id"`
	LectureSessionsMaterialSchoolID         string    `json:"lecture_sessions_material_school_id"`
	LectureSessionsMaterialCreatedAt        time.Time `json:"lecture_sessions_material_created_at"`
	LectureSessionsMaterialUpdatedAt        time.Time `json:"lecture_sessions_material_updated_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateLectureSessionsMaterialRequest struct {
	LectureSessionsMaterialSummary          *string `json:"lecture_sessions_material_summary,omitempty"`
	LectureSessionsMaterialTranscriptFull   *string `json:"lecture_sessions_material_transcript_full,omitempty"`
	LectureSessionsMaterialLectureSessionID string  `json:"lecture_sessions_material_lecture_session_id" validate:"required,uuid"`
	// SchoolID diambil dari token/scope pada controller
}

// Converter: Create -> Model (butuh schoolID dari controller)
func (r *CreateLectureSessionsMaterialRequest) ToModel(schoolID string) *model.LectureSessionsMaterialModel {
	return &model.LectureSessionsMaterialModel{
		LectureSessionsMaterialSummary:          r.LectureSessionsMaterialSummary,
		LectureSessionsMaterialTranscriptFull:   r.LectureSessionsMaterialTranscriptFull,
		LectureSessionsMaterialLectureSessionID: r.LectureSessionsMaterialLectureSessionID,
		LectureSessionsMaterialSchoolID:         schoolID,
	}
}

// ============================
// Update Request DTO (partial)
// ============================

type UpdateLectureSessionsMaterialRequest struct {
	LectureSessionsMaterialSummary        *string `json:"lecture_sessions_material_summary,omitempty"`         // set null untuk clear
	LectureSessionsMaterialTranscriptFull *string `json:"lecture_sessions_material_transcript_full,omitempty"` // set null untuk clear
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
		LectureSessionsMaterialSchoolID:         m.LectureSessionsMaterialSchoolID,
		LectureSessionsMaterialCreatedAt:        m.LectureSessionsMaterialCreatedAt,
		LectureSessionsMaterialUpdatedAt:        m.LectureSessionsMaterialUpdatedAt,
	}
}
