package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/exams/model"
	"time"
)

// ====================
// Response DTO
// ====================
type UserLectureSessionsExamDTO struct {
	UserLectureSessionsExamID        string    `json:"user_lecture_sessions_exam_id"`
	UserLectureSessionsExamGrade     *float64  `json:"user_lecture_sessions_exam_grade_result,omitempty"`
	UserLectureSessionsExamExamID    string    `json:"user_lecture_sessions_exam_exam_id"`
	UserLectureSessionsExamUserID    string    `json:"user_lecture_sessions_exam_user_id"`
	UserLectureSessionsExamMasjidID  string    `json:"user_lecture_sessions_exam_masjid_id"` // ✅ masjid_id
	UserLectureSessionsExamCreatedAt time.Time `json:"user_lecture_sessions_exam_created_at"`
}

// ====================
// Request DTO
// ====================
type CreateUserLectureSessionsExamRequest struct {
	UserLectureSessionsExamGrade    *float64 `json:"user_lecture_sessions_exam_grade_result,omitempty"`
	UserLectureSessionsExamExamID   string   `json:"user_lecture_sessions_exam_exam_id" validate:"required,uuid"`
	UserLectureSessionsExamUserID   string   `json:"user_lecture_sessions_exam_user_id" validate:"required,uuid"`
	UserLectureSessionsExamMasjidID string   `json:"user_lecture_sessions_exam_masjid_id" validate:"required,uuid"` // ✅ masjid_id
}

// ====================
// Converter
// ====================
func ToUserLectureSessionsExamDTO(m model.UserLectureSessionsExamModel) UserLectureSessionsExamDTO {
	return UserLectureSessionsExamDTO{
		UserLectureSessionsExamID:        m.UserLectureSessionsExamID,
		UserLectureSessionsExamGrade:     m.UserLectureSessionsExamGrade,
		UserLectureSessionsExamExamID:    m.UserLectureSessionsExamExamID,
		UserLectureSessionsExamUserID:    m.UserLectureSessionsExamUserID,
		UserLectureSessionsExamMasjidID:  m.UserLectureSessionsExamMasjidID,
		UserLectureSessionsExamCreatedAt: m.UserLectureSessionsExamCreatedAt,
	}
}
