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
	LectureSessionsQuestionAnswer      string    `json:"lecture_sessions_question_answer"`
	LectureSessionsQuestionCorrect     string    `json:"lecture_sessions_question_correct"`
	LectureSessionsQuestionExplanation string    `json:"lecture_sessions_question_explanation"`
	LectureSessionsQuestionQuizID      *string   `json:"lecture_sessions_question_quiz_id,omitempty"`
	LectureSessionsQuestionExamID      *string   `json:"lecture_sessions_question_exam_id,omitempty"`
	LectureSessionsQuestionCreatedAt   time.Time `json:"lecture_sessions_question_created_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateLectureSessionsQuestionRequest struct {
	LectureSessionsQuestion            string  `json:"lecture_sessions_question" validate:"required"`
	LectureSessionsQuestionAnswer      string  `json:"lecture_sessions_question_answer" validate:"required"`
	LectureSessionsQuestionCorrect     string  `json:"lecture_sessions_question_correct" validate:"required,oneof=A B C D"`
	LectureSessionsQuestionExplanation string  `json:"lecture_sessions_question_explanation" validate:"required"`
	LectureSessionsQuestionQuizID      *string `json:"lecture_sessions_question_quiz_id,omitempty" validate:"omitempty,uuid"`
	LectureSessionsQuestionExamID      *string `json:"lecture_sessions_question_exam_id,omitempty" validate:"omitempty,uuid"`
}

// ============================
// Converter
// ============================

func ToLectureSessionsQuestionDTO(m model.LectureSessionsQuestionModel) LectureSessionsQuestionDTO {
	return LectureSessionsQuestionDTO{
		LectureSessionsQuestionID:          m.LectureSessionsQuestionID,
		LectureSessionsQuestion:            m.LectureSessionsQuestion,
		LectureSessionsQuestionAnswer:      m.LectureSessionsQuestionAnswer,
		LectureSessionsQuestionCorrect:     m.LectureSessionsQuestionCorrect,
		LectureSessionsQuestionExplanation: m.LectureSessionsQuestionExplanation,
		LectureSessionsQuestionQuizID:      m.LectureSessionsQuestionQuizID,
		LectureSessionsQuestionExamID:      m.LectureSessionsQuestionExamID,
		LectureSessionsQuestionCreatedAt:   m.LectureSessionsQuestionCreatedAt,
	}
}
