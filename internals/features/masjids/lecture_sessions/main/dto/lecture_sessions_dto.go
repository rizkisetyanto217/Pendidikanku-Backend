package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"time"

	"github.com/google/uuid"
)

// =========================
// Struct tambahan
// =========================

// =========================
// Response DTO
// =========================

type LectureSessionDTO struct {
	LectureSessionID                     uuid.UUID    `json:"lecture_session_id"`
	LectureSessionTitle                  string       `json:"lecture_session_title"`
	LectureSessionDescription            string       `json:"lecture_session_description,omitempty"`
	LectureSessionTeacher                JSONBTeacher `json:"lecture_session_teacher"`
	LectureSessionImageURL               *string      `json:"lecture_session_image_url,omitempty"`
	LectureSessionStartTime              time.Time    `json:"lecture_session_start_time"`
	LectureSessionEndTime                time.Time    `json:"lecture_session_end_time"`
	LectureSessionPlace                  *string      `json:"lecture_session_place,omitempty"`
	LectureSessionLectureID              *uuid.UUID   `json:"lecture_session_lecture_id,omitempty"`
	LectureSessionCapacity               *int         `json:"lecture_session_capacity,omitempty"`
	LectureSessionIsPublic               bool         `json:"lecture_session_is_public"`
	LectureSessionIsRegistrationRequired bool         `json:"lecture_session_is_registration_required"`
	LectureSessionIsPaid                 bool         `json:"lecture_session_is_paid"`
	LectureSessionPrice                  *int         `json:"lecture_session_price,omitempty"`
	LectureSessionPaymentDeadline        *time.Time   `json:"lecture_session_payment_deadline,omitempty"`
	LectureSessionCreatedAt              time.Time    `json:"lecture_session_created_at"`
}

// =========================
// Request DTOs
// =========================

type CreateLectureSessionRequest struct {
	LectureSessionTitle                  string       `json:"lecture_session_title" validate:"required,min=3"`
	LectureSessionDescription            string       `json:"lecture_session_description,omitempty"`
	LectureSessionTeacher                JSONBTeacher `json:"lecture_session_teacher" validate:"required"`
	LectureSessionImageURL               *string      `json:"lecture_session_image_url,omitempty"`
	LectureSessionStartTime              time.Time    `json:"lecture_session_start_time" validate:"required"`
	LectureSessionEndTime                time.Time    `json:"lecture_session_end_time" validate:"required"`
	LectureSessionPlace                  *string      `json:"lecture_session_place,omitempty"`
	LectureSessionLectureID              *uuid.UUID   `json:"lecture_session_lecture_id,omitempty"`
	LectureSessionCapacity               *int         `json:"lecture_session_capacity,omitempty"`
	LectureSessionIsPublic               bool         `json:"lecture_session_is_public"`
	LectureSessionIsRegistrationRequired bool         `json:"lecture_session_is_registration_required"`
	LectureSessionIsPaid                 bool         `json:"lecture_session_is_paid"`
	LectureSessionPrice                  *int         `json:"lecture_session_price,omitempty"`
	LectureSessionPaymentDeadline        *time.Time   `json:"lecture_session_payment_deadline,omitempty"`
}

type UpdateLectureSessionRequest = CreateLectureSessionRequest

// =========================
// Request → Model converter
// =========================

func (r CreateLectureSessionRequest) ToModel() model.LectureSessionModel {
	return model.LectureSessionModel{
		LectureSessionTitle:                  r.LectureSessionTitle,
		LectureSessionDescription:            r.LectureSessionDescription,
		LectureSessionTeacher:                r.LectureSessionTeacher.ToModel(), // 👈 Panggil converter eksplisit
		LectureSessionImageURL:               r.LectureSessionImageURL,
		LectureSessionStartTime:              r.LectureSessionStartTime,
		LectureSessionEndTime:                r.LectureSessionEndTime,
		LectureSessionPlace:                  r.LectureSessionPlace,
		LectureSessionLectureID:              r.LectureSessionLectureID,
		LectureSessionCapacity:               r.LectureSessionCapacity,
		LectureSessionIsPublic:               r.LectureSessionIsPublic,
		LectureSessionIsRegistrationRequired: r.LectureSessionIsRegistrationRequired,
		LectureSessionIsPaid:                 r.LectureSessionIsPaid,
		LectureSessionPrice:                  r.LectureSessionPrice,
		LectureSessionPaymentDeadline:        r.LectureSessionPaymentDeadline,
	}
}

// =========================
// Model → Response converter
// =========================

func ToLectureSessionDTO(m model.LectureSessionModel) LectureSessionDTO {
	return LectureSessionDTO{
		LectureSessionID:                     m.LectureSessionID,
		LectureSessionTitle:                  m.LectureSessionTitle,
		LectureSessionDescription:            m.LectureSessionDescription,
		LectureSessionTeacher:                FromModel(m.LectureSessionTeacher),
		LectureSessionImageURL:               m.LectureSessionImageURL,
		LectureSessionStartTime:              m.LectureSessionStartTime,
		LectureSessionEndTime:                m.LectureSessionEndTime,
		LectureSessionPlace:                  m.LectureSessionPlace,
		LectureSessionLectureID:              m.LectureSessionLectureID,
		LectureSessionCapacity:               m.LectureSessionCapacity,
		LectureSessionIsPublic:               m.LectureSessionIsPublic,
		LectureSessionIsRegistrationRequired: m.LectureSessionIsRegistrationRequired,
		LectureSessionIsPaid:                 m.LectureSessionIsPaid,
		LectureSessionPrice:                  m.LectureSessionPrice,
		LectureSessionPaymentDeadline:        m.LectureSessionPaymentDeadline,
		LectureSessionCreatedAt:              m.LectureSessionCreatedAt,
	}
}
