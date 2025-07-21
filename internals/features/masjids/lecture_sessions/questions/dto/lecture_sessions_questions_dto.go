package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"
	"time"
)

// ============================
// Response DTO
// ============================

type LectureSessionsQuestionDTO struct {
	LectureSessionsQuestionID          string    `json:"lecture_sessions_question_id"`
	LectureSessionsQuestion            string    `json:"lecture_sessions_question"`
	LectureSessionsQuestionAnswers      []string  `json:"lecture_sessions_question_answers"` // ✅ tetap array
	LectureSessionsQuestionCorrect     string    `json:"lecture_sessions_question_correct"`
	LectureSessionsQuestionExplanation string    `json:"lecture_sessions_question_explanation"`
	LectureSessionsQuestionQuizID      *string   `json:"lecture_sessions_question_quiz_id,omitempty"`
	LectureSessionsQuestionExamID      *string   `json:"lecture_sessions_question_exam_id,omitempty"`
	LectureSessionsQuestionMasjidID    string    `json:"lecture_sessions_question_masjid_id"`
	LectureSessionsQuestionCreatedAt   time.Time `json:"lecture_sessions_question_created_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateLectureSessionsQuestionRequest struct {
	LectureSessionsQuestion            string   `json:"lecture_sessions_question" validate:"required"`
	LectureSessionsQuestionAnswers     []string `json:"lecture_sessions_question_answers" validate:"required,min=1,dive,required"`
	LectureSessionsQuestionCorrect     string   `json:"lecture_sessions_question_correct" validate:"required,oneof=A B C D"`
	LectureSessionsQuestionExplanation string   `json:"lecture_sessions_question_explanation" validate:"required"`
	LectureSessionsQuestionQuizID      *string  `json:"lecture_sessions_question_quiz_id,omitempty" validate:"omitempty,uuid"`
	LectureSessionsQuestionExamID      *string  `json:"lecture_sessions_question_exam_id,omitempty" validate:"omitempty,uuid"`
	LectureSessionsQuestionMasjidID    string   `json:"lecture_sessions_question_masjid_id,omitempty"` // optional untuk fallback
}


// ============================
// Converter
// ============================

func ToLectureSessionsQuestionDTO(m model.LectureSessionsQuestionModel) LectureSessionsQuestionDTO {
	return LectureSessionsQuestionDTO{
		LectureSessionsQuestionID:          m.LectureSessionsQuestionID,
		LectureSessionsQuestion:            m.LectureSessionsQuestion,
		LectureSessionsQuestionAnswers:      m.LectureSessionsQuestionAnswers,
		LectureSessionsQuestionCorrect:     m.LectureSessionsQuestionCorrect,
		LectureSessionsQuestionExplanation: m.LectureSessionsQuestionExplanation,
		LectureSessionsQuestionQuizID:      m.LectureSessionsQuestionQuizID,
		LectureSessionsQuestionExamID:      m.LectureSessionsQuestionExamID,
		LectureSessionsQuestionMasjidID:    m.LectureSessionsQuestionMasjidID,
		LectureSessionsQuestionCreatedAt:   m.LectureSessionsQuestionCreatedAt,
	}
}
