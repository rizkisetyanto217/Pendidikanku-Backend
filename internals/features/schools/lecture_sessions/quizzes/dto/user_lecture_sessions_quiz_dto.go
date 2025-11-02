package dto

import (
	"schoolku_backend/internals/features/schools/lecture_sessions/quizzes/model"
	"time"
)

//
// ============================
// Response DTO
// ============================
//

type UserLectureSessionsQuizDTO struct {
	UserLectureSessionsQuizID               string    `json:"user_lecture_sessions_quiz_id"`
	UserLectureSessionsQuizGrade            float64   `json:"user_lecture_sessions_quiz_grade_result"`
	UserLectureSessionsQuizQuizID           string    `json:"user_lecture_sessions_quiz_quiz_id"`
	UserLectureSessionsQuizUserID           string    `json:"user_lecture_sessions_quiz_user_id"`
	UserLectureSessionsQuizSchoolID         string    `json:"user_lecture_sessions_quiz_school_id"`
	UserLectureSessionsQuizLectureSessionID string    `json:"user_lecture_sessions_quiz_lecture_session_id"`
	UserLectureSessionsQuizAttemptCount     int       `json:"user_lecture_sessions_quiz_attempt_count"`
	UserLectureSessionsQuizDurationSeconds  int       `json:"user_lecture_sessions_quiz_duration_seconds"`
	UserLectureSessionsQuizCreatedAt        time.Time `json:"user_lecture_sessions_quiz_created_at"`
}

//
// ============================
// Create Request DTO
// ============================
//

type CreateUserLectureSessionsQuizRequest struct {
	UserLectureSessionsQuizGrade            float64 `json:"user_lecture_sessions_quiz_grade_result" validate:"required"`
	UserLectureSessionsQuizQuizID           string  `json:"user_lecture_sessions_quiz_quiz_id" validate:"required,uuid"`
	UserLectureSessionsQuizLectureSessionID string  `json:"user_lecture_sessions_quiz_lecture_session_id" validate:"omitempty,uuid"`
	UserLectureSessionsQuizDurationSeconds  int     `json:"user_lecture_sessions_quiz_duration_seconds" validate:"required"`
}

//
// ============================
// Converter Functions
// ============================
//

// Convert Model to DTO
func ToUserLectureSessionsQuizDTO(m model.UserLectureSessionsQuizModel) UserLectureSessionsQuizDTO {
	return UserLectureSessionsQuizDTO{
		UserLectureSessionsQuizID:               m.UserLectureSessionsQuizID,
		UserLectureSessionsQuizGrade:            m.UserLectureSessionsQuizGrade,
		UserLectureSessionsQuizQuizID:           m.UserLectureSessionsQuizQuizID,
		UserLectureSessionsQuizUserID:           m.UserLectureSessionsQuizUserID,
		UserLectureSessionsQuizSchoolID:         m.UserLectureSessionsQuizSchoolID,
		UserLectureSessionsQuizLectureSessionID: m.UserLectureSessionsQuizLectureSessionID,
		UserLectureSessionsQuizAttemptCount:     m.UserLectureSessionsQuizAttemptCount,
		UserLectureSessionsQuizDurationSeconds:  m.UserLectureSessionsQuizDurationSeconds,
		UserLectureSessionsQuizCreatedAt:        m.UserLectureSessionsQuizCreatedAt,
	}
}

// Convert Request to Model
func (r CreateUserLectureSessionsQuizRequest) ToModel(userID string) model.UserLectureSessionsQuizModel {
	return model.UserLectureSessionsQuizModel{
		UserLectureSessionsQuizGrade:            r.UserLectureSessionsQuizGrade,
		UserLectureSessionsQuizQuizID:           r.UserLectureSessionsQuizQuizID,
		UserLectureSessionsQuizUserID:           userID,
		UserLectureSessionsQuizLectureSessionID: r.UserLectureSessionsQuizLectureSessionID,
		UserLectureSessionsQuizDurationSeconds:  r.UserLectureSessionsQuizDurationSeconds,
		// SchoolID & AttemptCount akan diisi oleh controller
	}
}
