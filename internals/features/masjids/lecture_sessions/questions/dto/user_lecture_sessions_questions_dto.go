package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"
	"time"
)

// ============================
// Response DTO
// ============================

type LectureSessionsUserQuestionDTO struct {
	LectureSessionsUserQuestionID         string    `json:"lecture_sessions_user_question_id"`
	LectureSessionsUserQuestionAnswer     string    `json:"lecture_sessions_user_question_answer"`
	LectureSessionsUserQuestionIsCorrect  bool      `json:"lecture_sessions_user_question_is_correct"`
	LectureSessionsUserQuestionQuestionID string    `json:"lecture_sessions_user_question_question_id"`
	LectureSessionsUserQuestionCreatedAt  time.Time `json:"lecture_sessions_user_question_created_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateLectureSessionsUserQuestionRequest struct {
	LectureSessionsUserQuestionAnswer     string `json:"lecture_sessions_user_question_answer" validate:"required,oneof=A B C D"`
	LectureSessionsUserQuestionIsCorrect  bool   `json:"lecture_sessions_user_question_is_correct" validate:"required"`
	LectureSessionsUserQuestionQuestionID string `json:"lecture_sessions_user_question_question_id" validate:"required,uuid"`
}

// ============================
// Converter
// ============================

func ToLectureSessionsUserQuestionDTO(m model.LectureSessionsUserQuestionModel) LectureSessionsUserQuestionDTO {
	return LectureSessionsUserQuestionDTO{
		LectureSessionsUserQuestionID:         m.LectureSessionsUserQuestionID,
		LectureSessionsUserQuestionAnswer:     m.LectureSessionsUserQuestionAnswer,
		LectureSessionsUserQuestionIsCorrect:  m.LectureSessionsUserQuestionIsCorrect,
		LectureSessionsUserQuestionQuestionID: m.LectureSessionsUserQuestionQuestionID,
		LectureSessionsUserQuestionCreatedAt:  m.LectureSessionsUserQuestionCreatedAt,
	}
}
