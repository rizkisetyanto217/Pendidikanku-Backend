package dto

import (
	"time"

	"schoolku_backend/internals/features/schools/lecture_sessions/main/model"

	"github.com/google/uuid"
)

/*
	=========================
	  Response DTOs

=========================
*/
type UserLectureSessionDTO struct {
	UserLectureSessionID               uuid.UUID `json:"user_lecture_session_id"`
	UserLectureSessionGradeResult      *float64  `json:"user_lecture_session_grade_result,omitempty"`
	UserLectureSessionLectureSessionID uuid.UUID `json:"user_lecture_session_lecture_session_id"`
	UserLectureSessionLectureID        uuid.UUID `json:"user_lecture_session_lecture_id"`
	UserLectureSessionUserID           uuid.UUID `json:"user_lecture_session_user_id"`
	UserLectureSessionSchoolID         uuid.UUID `json:"user_lecture_session_school_id"`
	UserLectureSessionCreatedAt        string    `json:"user_lecture_session_created_at"`
	UserLectureSessionUpdatedAt        string    `json:"user_lecture_session_updated_at"`
}

/*
	=========================
	  Request DTOs

=========================
*/
type CreateUserLectureSessionRequest struct {
	UserLectureSessionGradeResult      *float64  `json:"user_lecture_session_grade_result,omitempty"`
	UserLectureSessionLectureSessionID uuid.UUID `json:"user_lecture_session_lecture_session_id" validate:"required,uuid"`
	UserLectureSessionLectureID        uuid.UUID `json:"user_lecture_session_lecture_id" validate:"required,uuid"`
	UserLectureSessionUserID           uuid.UUID `json:"user_lecture_session_user_id" validate:"required,uuid"`
	UserLectureSessionSchoolID         uuid.UUID `json:"user_lecture_session_school_id" validate:"required,uuid"`
}

/* =========================
   Converters
========================= */

func ToUserLectureSessionDTO(u model.UserLectureSessionModel) UserLectureSessionDTO {
	const fmt = "2006-01-02 15:04:05"
	return UserLectureSessionDTO{
		UserLectureSessionID:               u.UserLectureSessionID,
		UserLectureSessionGradeResult:      u.UserLectureSessionGradeResult,
		UserLectureSessionLectureSessionID: u.UserLectureSessionLectureSessionID,
		UserLectureSessionLectureID:        u.UserLectureSessionLectureID,
		UserLectureSessionUserID:           u.UserLectureSessionUserID,
		UserLectureSessionSchoolID:         u.UserLectureSessionSchoolID,
		UserLectureSessionCreatedAt:        toTimeString(u.UserLectureSessionCreatedAt, fmt),
		UserLectureSessionUpdatedAt:        toTimeString(u.UserLectureSessionUpdatedAt, fmt),
	}
}

func ToUserLectureSessionDTOList(items []model.UserLectureSessionModel) []UserLectureSessionDTO {
	out := make([]UserLectureSessionDTO, 0, len(items))
	for i := range items {
		out = append(out, ToUserLectureSessionDTO(items[i]))
	}
	return out
}

func (r CreateUserLectureSessionRequest) ToModel() model.UserLectureSessionModel {
	return model.UserLectureSessionModel{
		UserLectureSessionGradeResult:      r.UserLectureSessionGradeResult,
		UserLectureSessionLectureSessionID: r.UserLectureSessionLectureSessionID,
		UserLectureSessionLectureID:        r.UserLectureSessionLectureID,
		UserLectureSessionUserID:           r.UserLectureSessionUserID,
		UserLectureSessionSchoolID:         r.UserLectureSessionSchoolID,
	}
}

/* =========================
   Utils
========================= */

func toTimeString(t time.Time, layout string) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(layout)
}
