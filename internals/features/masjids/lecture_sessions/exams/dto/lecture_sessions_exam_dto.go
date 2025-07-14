package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/exams/model"
	"time"
)

// ========================
// Request DTOs
// ========================

type CreateLectureSessionsExamRequest struct {
	LectureSessionsExamTitle       string  `json:"lecture_sessions_exam_title" validate:"required,min=5"`
	LectureSessionsExamDescription *string `json:"lecture_sessions_exam_description,omitempty"`
	LectureSessionsExamLectureID   string  `json:"lecture_sessions_exam_lecture_id" validate:"required,uuid"`
	LectureSessionsExamMasjidID    string  `json:"lecture_sessions_exam_masjid_id" validate:"required,uuid"` // ✅ baru
}

type UpdateLectureSessionsExamRequest struct {
	LectureSessionsExamTitle       string  `json:"lecture_sessions_exam_title" validate:"required,min=5"`
	LectureSessionsExamDescription *string `json:"lecture_sessions_exam_description,omitempty"`
}

// ========================
// Response DTO
// ========================

type LectureSessionsExamDTO struct {
	LectureSessionsExamID          string    `json:"lecture_sessions_exam_id"`
	LectureSessionsExamTitle       string    `json:"lecture_sessions_exam_title"`
	LectureSessionsExamDescription *string   `json:"lecture_sessions_exam_description,omitempty"`
	LectureSessionsExamLectureID   string    `json:"lecture_sessions_exam_lecture_id"`
	LectureSessionsExamMasjidID    string    `json:"lecture_sessions_exam_masjid_id"` // ✅ baru
	LectureSessionsExamCreatedAt   time.Time `json:"lecture_sessions_exam_created_at"`
}

// ========================
// Converter
// ========================

func ToLectureSessionsExamDTO(m model.LectureSessionsExamModel) LectureSessionsExamDTO {
	return LectureSessionsExamDTO{
		LectureSessionsExamID:          m.LectureSessionsExamID,
		LectureSessionsExamTitle:       m.LectureSessionsExamTitle,
		LectureSessionsExamDescription: m.LectureSessionsExamDescription,
		LectureSessionsExamLectureID:   m.LectureSessionsExamLectureID,
		LectureSessionsExamMasjidID:    m.LectureSessionsExamMasjidID, // ✅ baru
		LectureSessionsExamCreatedAt:   m.LectureSessionsExamCreatedAt,
	}
}
