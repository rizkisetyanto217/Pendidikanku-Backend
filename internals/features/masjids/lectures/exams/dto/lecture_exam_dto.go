package dto

import (
	"masjidku_backend/internals/features/masjids/lectures/exams/model"
	"time"
)

// ========================
// Request DTOs
// ========================

type CreateLectureExamRequest struct {
	LectureExamTitle       string  `json:"lecture_exam_title" validate:"required,min=5"`
	LectureExamDescription *string `json:"lecture_exam_description,omitempty"`
	LectureExamLectureID   string  `json:"lecture_exam_lecture_id" validate:"required,uuid"`
	LectureExamMasjidID    string  `json:"lecture_exam_masjid_id" validate:"required,uuid"` // ✅ baru
}

type UpdateLectureExamRequest struct {
	LectureExamTitle       string  `json:"lecture_exam_title" validate:"required,min=5"`
	LectureExamDescription *string `json:"lecture_exam_description,omitempty"`
}

// ========================
// Response DTO
// ========================

type LectureExamDTO struct {
	LectureExamID          string    `json:"lecture_exam_id"`
	LectureExamTitle       string    `json:"lecture_exam_title"`
	LectureExamDescription *string   `json:"lecture_exam_description,omitempty"`
	LectureExamLectureID   string    `json:"lecture_exam_lecture_id"`
	LectureExamMasjidID    string    `json:"lecture_exam_masjid_id"` // ✅ baru
	LectureExamCreatedAt   time.Time `json:"lecture_exam_created_at"`
}

// ========================
// Converter
// ========================

func ToLectureExamDTO(m model.LectureExamModel) LectureExamDTO {
	return LectureExamDTO{
		LectureExamID:          m.LectureExamID,
		LectureExamTitle:       m.LectureExamTitle,
		LectureExamDescription: m.LectureExamDescription,
		LectureExamLectureID:   m.LectureExamLectureID,
		LectureExamMasjidID:    m.LectureExamMasjidID, // ✅ baru
		LectureExamCreatedAt:   m.LectureExamCreatedAt,
	}
}
