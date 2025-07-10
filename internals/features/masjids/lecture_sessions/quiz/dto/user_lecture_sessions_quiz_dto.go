package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/quiz/model"
	"time"
)

// ============================
// Response DTO
// ============================
type UserLectureSessionsQuizDTO struct {
	UserLectureSessionsQuizID        string    `json:"user_lecture_sessions_quiz_id"`
	UserLectureSessionsQuizGrade     float64   `json:"user_lecture_sessions_quiz_grade_result"`
	UserLectureSessionsQuizQuizID    string    `json:"user_lecture_sessions_quiz_quiz_id"`
	UserLectureSessionsQuizUserID    string    `json:"user_lecture_sessions_quiz_user_id"`
	UserLectureSessionsQuizCreatedAt time.Time `json:"user_lecture_sessions_quiz_created_at"`
}

// ============================
// Create Request DTO
// ============================
type CreateUserLectureSessionsQuizRequest struct {
	UserLectureSessionsQuizGrade  float64 `json:"user_lecture_sessions_quiz_grade_result" validate:"required"`
	UserLectureSessionsQuizQuizID string  `json:"user_lecture_sessions_quiz_quiz_id" validate:"required,uuid"`
}

// ============================
// Converter
// ============================
func ToUserLectureSessionsQuizDTO(m model.UserLectureSessionsQuizModel) UserLectureSessionsQuizDTO {
	return UserLectureSessionsQuizDTO{
		UserLectureSessionsQuizID:        m.UserLectureSessionsQuizID,
		UserLectureSessionsQuizGrade:     m.UserLectureSessionsQuizGrade,
		UserLectureSessionsQuizQuizID:    m.UserLectureSessionsQuizQuizID,
		UserLectureSessionsQuizUserID:    m.UserLectureSessionsQuizUserID,
		UserLectureSessionsQuizCreatedAt: m.UserLectureSessionsQuizCreatedAt,
	}
}
