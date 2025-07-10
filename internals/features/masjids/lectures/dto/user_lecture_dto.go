package dto

import (
	"masjidku_backend/internals/features/masjids/lectures/model"

	"github.com/google/uuid"
)

type UserLectureRequest struct {
	UserLectureUserID                 uuid.UUID `json:"user_lecture_user_id"`
	UserLectureLectureID              uuid.UUID `json:"user_lecture_lecture_id"`
	UserLectureGrade                  int       `json:"user_lecture_grade_result"`
}

type UserLectureResponse struct {
	UserLectureID                     uuid.UUID `json:"user_lecture_id"`
	UserLectureUserID                 uuid.UUID `json:"user_lecture_user_id"`
	UserLectureLectureID              uuid.UUID `json:"user_lecture_lecture_id"`
	UserLectureGrade                  int       `json:"user_lecture_grade_result"`
	UserLectureTotalCompletedSessions int       `json:"user_lecture_total_completed_sessions"`
	UserLectureCreatedAt              string    `json:"user_lecture_created_at"`
}

// Convert request → model
func (r *UserLectureRequest) ToModel() *model.UserLectureModel {
	return &model.UserLectureModel{
		UserLectureUserID:                 r.UserLectureUserID,
		UserLectureLectureID:              r.UserLectureLectureID,
		UserLectureGrade:                  r.UserLectureGrade,
	}
}

// Convert model → response
func ToUserLectureResponse(m *model.UserLectureModel) *UserLectureResponse {
	return &UserLectureResponse{
		UserLectureID:                     m.UserLectureID,
		UserLectureUserID:                 m.UserLectureUserID,
		UserLectureLectureID:              m.UserLectureLectureID,
		UserLectureGrade:                  m.UserLectureGrade,
		UserLectureTotalCompletedSessions: m.UserLectureTotalCompletedSessions,
		UserLectureCreatedAt:              m.UserLectureCreatedAt.Format("2006-01-02 15:04:05"),
	}
}
