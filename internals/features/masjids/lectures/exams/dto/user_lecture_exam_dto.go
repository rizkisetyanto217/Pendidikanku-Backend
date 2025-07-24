package dto

import (
	"masjidku_backend/internals/features/masjids/lectures/exams/model"
	"time"

	"github.com/google/uuid"
)

// ====================
// Response DTO
// ====================

type UserLectureExamDTO struct {
	UserLectureExamID        uuid.UUID   `json:"user_lecture_exam_id"`
	UserLectureExamGrade     *float64    `json:"user_lecture_exam_grade_result,omitempty"`
	UserLectureExamUserName  string      `json:"user_lecture_exam_user_name,omitempty"`
	UserLectureExamExamID    uuid.UUID   `json:"user_lecture_exam_exam_id"`
	UserLectureExamUserID    *uuid.UUID  `json:"user_lecture_exam_user_id,omitempty"`
	UserLectureExamMasjidID  uuid.UUID   `json:"user_lecture_exam_masjid_id"`
	UserLectureExamCreatedAt time.Time   `json:"user_lecture_exam_created_at"`
}

// ====================
// Request DTO
// ====================
type CreateUserLectureExamRequest struct {
	UserLectureExamGrade     *float64   `json:"user_lecture_exam_grade_result,omitempty"`
	UserLectureExamExamID    uuid.UUID  `json:"user_lecture_exam_exam_id" validate:"required"`
	UserLectureExamUserID    *uuid.UUID `json:"user_lecture_exam_user_id,omitempty"` // optional
	UserLectureExamUserName  string     `json:"user_lecture_exam_user_name" validate:"required,min=2"`
	UserLectureExamMasjidSlug string    `json:"user_lecture_exam_masjid_slug" validate:"required"` // 👈 dikirim dari frontend
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
		UserLectureExamUserName:  m.UserLectureExamUserName,
	}
}
