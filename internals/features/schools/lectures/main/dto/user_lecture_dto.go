package dto

import (
	"time"

	"schoolku_backend/internals/features/schools/lectures/main/model"

	"github.com/google/uuid"
)

/*
======================

	REQUEST
	======================
*/
type UserLectureRequest struct {
	UserLectureUserID       uuid.UUID  `json:"user_lecture_user_id"`
	UserLectureLectureID    uuid.UUID  `json:"user_lecture_lecture_id"`
	UserLectureSchoolID     uuid.UUID  `json:"user_lecture_school_id"`
	UserLectureGrade        *int       `json:"user_lecture_grade_result,omitempty"`
	UserLectureIsRegistered bool       `json:"user_lecture_is_registered,omitempty"`
	UserLectureHasPaid      bool       `json:"user_lecture_has_paid,omitempty"`
	UserLecturePaidAmount   *int       `json:"user_lecture_paid_amount,omitempty"`
	UserLecturePaymentTime  *time.Time `json:"user_lecture_payment_time,omitempty"`
}

func (r *UserLectureRequest) ToModel() *model.UserLectureModel {
	return &model.UserLectureModel{
		UserLectureUserID:       r.UserLectureUserID,
		UserLectureLectureID:    r.UserLectureLectureID,
		UserLectureSchoolID:     r.UserLectureSchoolID,
		UserLectureGradeResult:  r.UserLectureGrade,
		UserLectureIsRegistered: r.UserLectureIsRegistered,
		UserLectureHasPaid:      r.UserLectureHasPaid,
		UserLecturePaidAmount:   r.UserLecturePaidAmount,
		UserLecturePaymentTime:  r.UserLecturePaymentTime,
	}
}

/*
======================

	RESPONSE
	======================
*/
type UserLectureResponse struct {
	UserLectureID                     uuid.UUID  `json:"user_lecture_id"`
	UserLectureUserID                 uuid.UUID  `json:"user_lecture_user_id"`
	UserLectureLectureID              uuid.UUID  `json:"user_lecture_lecture_id"`
	UserLectureSchoolID               uuid.UUID  `json:"user_lecture_school_id"`
	UserLectureGradeResult            *int       `json:"user_lecture_grade_result,omitempty"`
	UserLectureTotalCompletedSessions int        `json:"user_lecture_total_completed_sessions"`
	UserLectureIsRegistered           bool       `json:"user_lecture_is_registered"`
	UserLectureHasPaid                bool       `json:"user_lecture_has_paid"`
	UserLecturePaidAmount             *int       `json:"user_lecture_paid_amount,omitempty"`
	UserLecturePaymentTime            *time.Time `json:"user_lecture_payment_time,omitempty"`
	UserLectureCreatedAt              string     `json:"user_lecture_created_at"`
	UserLectureUpdatedAt              string     `json:"user_lecture_updated_at"`
}

/*
======================

	MAPPER
	======================
*/
func ToUserLectureResponse(m *model.UserLectureModel) *UserLectureResponse {
	return &UserLectureResponse{
		UserLectureID:                     m.UserLectureID,
		UserLectureUserID:                 m.UserLectureUserID,
		UserLectureLectureID:              m.UserLectureLectureID,
		UserLectureSchoolID:               m.UserLectureSchoolID,
		UserLectureGradeResult:            m.UserLectureGradeResult,
		UserLectureTotalCompletedSessions: m.UserLectureTotalCompletedSessions,
		UserLectureIsRegistered:           m.UserLectureIsRegistered,
		UserLectureHasPaid:                m.UserLectureHasPaid,
		UserLecturePaidAmount:             m.UserLecturePaidAmount,
		UserLecturePaymentTime:            m.UserLecturePaymentTime,
		UserLectureCreatedAt:              m.UserLectureCreatedAt.Format("2006-01-02 15:04:05"),
		UserLectureUpdatedAt:              m.UserLectureUpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func ToUserLectureResponseList(models []model.UserLectureModel) []UserLectureResponse {
	out := make([]UserLectureResponse, 0, len(models))
	for i := range models {
		out = append(out, *ToUserLectureResponse(&models[i]))
	}
	return out
}
