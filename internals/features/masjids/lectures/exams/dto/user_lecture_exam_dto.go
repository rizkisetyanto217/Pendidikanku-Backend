package dto

import (
	"masjidku_backend/internals/features/masjids/lectures/exams/model"
	"time"
)

// ====================
// Response DTO
// ====================
type UserLectureExamDTO struct {
	UserLectureExamID        string    `json:"user_lecture_exam_id"`
	UserLectureExamGrade     *float64  `json:"user_lecture_exam_grade_result,omitempty"`
	UserLectureExamExamID    string    `json:"user_lecture_exam_exam_id"`
	UserLectureExamUserID    string    `json:"user_lecture_exam_user_id"`
	UserLectureExamMasjidID  string    `json:"user_lecture_exam_masjid_id"` // ✅ masjid_id
	UserLectureExamCreatedAt time.Time `json:"user_lecture_exam_created_at"`
}

// ====================
// Request DTO
// ====================
type CreateUserLectureExamRequest struct {
	UserLectureExamGrade    *float64 `json:"user_lecture_exam_grade_result,omitempty"`
	UserLectureExamExamID   string   `json:"user_lecture_exam_exam_id" validate:"required,uuid"`
	UserLectureExamUserID   string   `json:"user_lecture_exam_user_id" validate:"required,uuid"`
	UserLectureExamMasjidID string   `json:"user_lecture_exam_masjid_id" validate:"required,uuid"` // ✅ masjid_id
}

// ====================
// Converter
// ====================
func ToUserLectureExamDTO(m model.UserLectureExamModel) UserLectureExamDTO {
	return UserLectureExamDTO{
		UserLectureExamID:        m.UserLectureExamID,
		UserLectureExamGrade:     m.UserLectureExamGrade,
		UserLectureExamExamID:    m.UserLectureExamExamID,
		UserLectureExamUserID:    m.UserLectureExamUserID,
		UserLectureExamMasjidID:  m.UserLectureExamMasjidID,
		UserLectureExamCreatedAt: m.UserLectureExamCreatedAt,
	}
}
