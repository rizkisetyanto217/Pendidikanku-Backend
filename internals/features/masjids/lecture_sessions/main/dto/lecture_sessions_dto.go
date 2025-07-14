package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	"time"

	"github.com/google/uuid"
)

// =========================
// Request DTOs: Create & Update
// =========================

type CreateLectureSessionRequest struct {
	LectureSessionTitle        string     `json:"lecture_session_title" validate:"required,min=3"`
	LectureSessionDescription  string     `json:"lecture_session_description"`
	LectureSessionTeacherID    uuid.UUID  `json:"lecture_session_teacher_id" validate:"required"`
	LectureSessionTeacherName  string     `json:"lecture_session_teacher_name" validate:"required"`
	LectureSessionStartTime    time.Time  `json:"lecture_session_start_time" validate:"required"`
	LectureSessionEndTime      time.Time  `json:"lecture_session_end_time" validate:"required"`
	LectureSessionPlace        *string    `json:"lecture_session_place"`
	LectureSessionImageURL     *string    `json:"lecture_session_image_url"`
	LectureSessionLectureID    *uuid.UUID `json:"lecture_session_lecture_id"`

	// ✅ Jika dikirim, berarti tidak perlu diperiksa oleh guru
	LectureSessionApprovedByTeacherAt *time.Time `json:"lecture_session_approved_by_teacher_at,omitempty"`
}

type UpdateLectureSessionRequest = CreateLectureSessionRequest



// =========================
// Response DTO
// =========================

type LectureSessionDTO struct {
	LectureSessionID           uuid.UUID  `json:"lecture_session_id"`
	LectureSessionTitle        string     `json:"lecture_session_title"`
	LectureSessionDescription  string     `json:"lecture_session_description"`
	LectureSessionTeacherID    uuid.UUID  `json:"lecture_session_teacher_id"`
	LectureSessionTeacherName  string     `json:"lecture_session_teacher_name"`
	LectureSessionStartTime    time.Time  `json:"lecture_session_start_time"`
	LectureSessionEndTime      time.Time  `json:"lecture_session_end_time"`
	LectureSessionPlace        *string    `json:"lecture_session_place"`
	LectureSessionImageURL     *string    `json:"lecture_session_image_url"`
	LectureSessionLectureID    *uuid.UUID `json:"lecture_session_lecture_id"`

	LectureTitle               string     `json:"lecture_title"` // ✅ Tambahan


	// Validasi Role
	LectureSessionApprovedByAdminID   *uuid.UUID `json:"lecture_session_approved_by_admin_id"`
	LectureSessionApprovedByAdminAt   *time.Time `json:"lecture_session_approved_by_admin_at"`
	LectureSessionApprovedByAuthorID  *uuid.UUID `json:"lecture_session_approved_by_author_id"`
	LectureSessionApprovedByAuthorAt  *time.Time `json:"lecture_session_approved_by_author_at"`
	LectureSessionApprovedByTeacherID *uuid.UUID `json:"lecture_session_approved_by_teacher_id"`
	LectureSessionApprovedByTeacherAt *time.Time `json:"lecture_session_approved_by_teacher_at"`
	LectureSessionApprovedByDkmAt     *time.Time `json:"lecture_session_approved_by_dkm_at"`

	// Status DKM
	LectureSessionIsActive bool      `json:"lecture_session_is_active"`

	// Metadata
	LectureSessionCreatedAt time.Time  `json:"lecture_session_created_at"`
	LectureSessionUpdatedAt *time.Time `json:"lecture_session_updated_at"`
}


// =========================
// Request → Model
// =========================

func (r CreateLectureSessionRequest) ToModel() model.LectureSessionModel {
	return model.LectureSessionModel{
		LectureSessionTitle:       r.LectureSessionTitle,
		LectureSessionDescription: r.LectureSessionDescription,
		LectureSessionTeacherID:   r.LectureSessionTeacherID,
		LectureSessionTeacherName: r.LectureSessionTeacherName,
		LectureSessionStartTime:   r.LectureSessionStartTime,
		LectureSessionEndTime:     r.LectureSessionEndTime,
		LectureSessionPlace:       r.LectureSessionPlace,
		LectureSessionImageURL:    r.LectureSessionImageURL,
		LectureSessionLectureID:   r.LectureSessionLectureID,
	}
}


// =========================
// Model → Response
// =========================

func ToLectureSessionDTO(m model.LectureSessionModel) LectureSessionDTO {
	return LectureSessionDTO{
		LectureSessionID:                  m.LectureSessionID,
		LectureSessionTitle:              m.LectureSessionTitle,
		LectureSessionDescription:        m.LectureSessionDescription,
		LectureSessionTeacherID:          m.LectureSessionTeacherID,
		LectureSessionTeacherName:        m.LectureSessionTeacherName,
		LectureSessionStartTime:          m.LectureSessionStartTime,
		LectureSessionEndTime:            m.LectureSessionEndTime,
		LectureSessionPlace:              m.LectureSessionPlace,
		LectureSessionImageURL:           m.LectureSessionImageURL,
		LectureSessionLectureID:          m.LectureSessionLectureID,
		LectureSessionApprovedByAdminID:  m.LectureSessionApprovedByAdminID,
		LectureSessionApprovedByAdminAt:  m.LectureSessionApprovedByAdminAt,
		LectureSessionApprovedByAuthorID: m.LectureSessionApprovedByAuthorID,
		LectureSessionApprovedByAuthorAt: m.LectureSessionApprovedByAuthorAt,
		LectureSessionApprovedByTeacherID: m.LectureSessionApprovedByTeacherID,
		LectureSessionApprovedByTeacherAt: m.LectureSessionApprovedByTeacherAt,
		LectureSessionApprovedByDkmAt:     m.LectureSessionApprovedByDkmAt,
		LectureSessionIsActive:           m.LectureSessionIsActive,
		LectureSessionCreatedAt:          m.LectureSessionCreatedAt,
		LectureSessionUpdatedAt:          m.LectureSessionUpdatedAt,
	}
}


// Untuk Lecture Title
func ToLectureSessionDTOWithLectureTitle(m model.LectureSessionModel, lectureTitle string) LectureSessionDTO {
	dto := ToLectureSessionDTO(m)
	dto.LectureTitle = lectureTitle
	return dto
}


// =========================
// DTO: Approval by Role
// =========================

type ApproveLectureSessionByAdminRequest struct {
	ApprovedByAdminID uuid.UUID `json:"approved_by_admin_id" validate:"required"`
}

type ApproveLectureSessionByAuthorRequest struct {
	ApprovedByAuthorID uuid.UUID `json:"approved_by_author_id" validate:"required"`
}

type ApproveLectureSessionByTeacherRequest struct {
	ApprovedByTeacherID uuid.UUID `json:"approved_by_teacher_id" validate:"required"`
}


// =========================
// DTO: Set Active by DKM
// =========================

type SetLectureSessionActiveRequest struct {
	IsActive bool `json:"is_active" validate:"required"`
}

type SetLectureSessionActiveResponse struct {
	LectureSessionID uuid.UUID `json:"lecture_session_id"`
	IsActive         bool      `json:"is_active"`
}
