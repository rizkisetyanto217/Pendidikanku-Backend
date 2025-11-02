package dto

import (
	"schoolku_backend/internals/features/schools/lecture_sessions/questions/model"
	"time"
)

// ============================
// Response DTO
// ============================

type LectureSessionsQuestionDTO struct {
	LectureSessionsQuestionID          string    `json:"lecture_sessions_question_id"`
	LectureSessionsQuestion            string    `json:"lecture_sessions_question"`
	LectureSessionsQuestionAnswers     []string  `json:"lecture_sessions_question_answers"` // ✅ tetap array
	LectureSessionsQuestionCorrect     string    `json:"lecture_sessions_question_correct"`
	LectureSessionsQuestionExplanation string    `json:"lecture_sessions_question_explanation"`
	LectureSessionsQuestionQuizID      *string   `json:"lecture_sessions_question_quiz_id,omitempty"`
	LectureQuestionExamID              *string   `json:"lecture_question_exam_id,omitempty"`
	LectureSessionsQuestionSchoolID    string    `json:"lecture_sessions_question_school_id"`
	LectureSessionsQuestionCreatedAt   time.Time `json:"lecture_sessions_question_created_at"`
}

// dto/lecture_sessions_question_dto.go

type UpdateLectureSessionsQuestionDTO struct {
	LectureSessionsQuestion            *string   `json:"lecture_sessions_question"`
	LectureSessionsQuestionAnswers     *[]string `json:"lecture_sessions_question_answers"`
	LectureSessionsQuestionCorrect     *string   `json:"lecture_sessions_question_correct"`
	LectureSessionsQuestionExplanation *string   `json:"lecture_sessions_question_explanation"`
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
	LectureQuestionExamID              *string  `json:"lecture_question_exam_id,omitempty" validate:"omitempty,uuid"`
	LectureSessionsQuestionSchoolID    string   `json:"lecture_sessions_question_school_id,omitempty"` // optional untuk fallback
}

// ============================
// Converter
// ============================

func ToLectureSessionsQuestionDTO(m model.LectureSessionsQuestionModel) LectureSessionsQuestionDTO {
	return LectureSessionsQuestionDTO{
		LectureSessionsQuestionID:          m.LectureSessionsQuestionID,
		LectureSessionsQuestion:            m.LectureSessionsQuestion,
		LectureSessionsQuestionAnswers:     m.LectureSessionsQuestionAnswers,
		LectureSessionsQuestionCorrect:     m.LectureSessionsQuestionCorrect,
		LectureSessionsQuestionExplanation: m.LectureSessionsQuestionExplanation,
		LectureSessionsQuestionQuizID:      m.LectureSessionsQuestionQuizID,
		LectureQuestionExamID:              m.LectureQuestionExamID,
		LectureSessionsQuestionSchoolID:    m.LectureSessionsQuestionSchoolID,
		LectureSessionsQuestionCreatedAt:   m.LectureSessionsQuestionCreatedAt,
	}
}
