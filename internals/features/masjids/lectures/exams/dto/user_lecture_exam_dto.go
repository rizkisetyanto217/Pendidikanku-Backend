package dto

import (
	"time"

	"masjidku_backend/internals/features/masjids/lectures/exams/model"

	"github.com/google/uuid"
)

//
// ====================
// Response DTO
// ====================
//
type UserLectureExamDTO struct {
	UserLectureExamID        uuid.UUID  `json:"user_lecture_exam_id"`
	UserLectureExamGrade     *float64   `json:"user_lecture_exam_grade_result,omitempty"`
	UserLectureExamUserName  *string    `json:"user_lecture_exam_user_name,omitempty"`
	UserLectureExamExamID    uuid.UUID  `json:"user_lecture_exam_exam_id"`
	UserLectureExamUserID    uuid.UUID  `json:"user_lecture_exam_user_id"`
	UserLectureExamMasjidID  uuid.UUID  `json:"user_lecture_exam_masjid_id"`
	UserLectureExamCreatedAt time.Time  `json:"user_lecture_exam_created_at"`
	UserLectureExamUpdatedAt time.Time  `json:"user_lecture_exam_updated_at"`
}

//
// ====================
// Create Request DTO
// ====================
//
// catatan:
// - user_id diambil dari token (controller), bukan dari body
// - masjid dikirim sebagai slug → di-resolve ke masjid_id di controller
//
type CreateUserLectureExamRequest struct {
	UserLectureExamGrade      *float64  `json:"user_lecture_exam_grade_result,omitempty"`
	UserLectureExamExamID     uuid.UUID `json:"user_lecture_exam_exam_id" validate:"required"`
	UserLectureExamUserName   *string   `json:"user_lecture_exam_user_name,omitempty"` // optional
	UserLectureExamMasjidSlug string    `json:"user_lecture_exam_masjid_slug" validate:"required"`
}

//
// (Opsional) Update Request DTO (partial)
//
type UpdateUserLectureExamRequest struct {
	UserLectureExamGrade    *float64 `json:"user_lecture_exam_grade_result,omitempty"`
	UserLectureExamUserName *string  `json:"user_lecture_exam_user_name,omitempty"` // bisa null untuk clear
}

//
// ====================
// Converters
// ====================
//
func ToUserLectureExamDTO(m model.UserLectureExamModel) UserLectureExamDTO {
	return UserLectureExamDTO{
		UserLectureExamID:        m.UserLectureExamID,
		UserLectureExamGrade:     m.UserLectureExamGrade,
		UserLectureExamExamID:    m.UserLectureExamExamID,
		UserLectureExamUserID:    m.UserLectureExamUserID,
		UserLectureExamMasjidID:  m.UserLectureExamMasjidID,
		UserLectureExamCreatedAt: m.UserLectureExamCreatedAt,
		UserLectureExamUpdatedAt: m.UserLectureExamUpdatedAt,
		UserLectureExamUserName:  m.UserLectureExamUserName,
	}
}

// Helper untuk controller: create request -> model
// - masjidID & userID dipasok dari controller (token/resolve slug)
func (r *CreateUserLectureExamRequest) ToModel(masjidID, userID uuid.UUID) *model.UserLectureExamModel {
	return &model.UserLectureExamModel{
		UserLectureExamGrade:    r.UserLectureExamGrade,
		UserLectureExamExamID:   r.UserLectureExamExamID,
		UserLectureExamUserID:   userID,
		UserLectureExamUserName: r.UserLectureExamUserName,
		UserLectureExamMasjidID: masjidID,
	}
}
