// file: internals/features/school/class_attendance_sessions/dto/class_attendance_session_url_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/class_attendance_result/attendance_sessions/model"
)

/* =========================================================
 * REQUESTS
 * ========================================================= */

// Create (JSON)
type CreateClassAttendanceSessionURLRequest struct {
	// Wajib: id sesi (UUID)
	ClassAttendanceSessionURLSessionID uuid.UUID `json:"class_attendance_session_url_session_id" validate:"required,uuid4"`

	// Opsional: label
	ClassAttendanceSessionURLLabel *string `json:"class_attendance_session_url_label" validate:"omitempty,max=120"`

	// Wajib: URL
	ClassAttendanceSessionURLHref string `json:"class_attendance_session_url_href" validate:"required,url"`
}

// Update (partial JSON)
type UpdateClassAttendanceSessionURLRequest struct {
	ClassAttendanceSessionURLLabel *string    `json:"class_attendance_session_url_label" validate:"omitempty,max=120"`
	ClassAttendanceSessionURLHref  *string    `json:"class_attendance_session_url_href"  validate:"omitempty,url"`
	ClassAttendanceSessionURLTrashURL *string `json:"class_attendance_session_url_trash_url" validate:"omitempty,url"`
	ClassAttendanceSessionURLDeletePendingUntil *time.Time `json:"class_attendance_session_url_delete_pending_until" validate:"omitempty"`
}

// Filter / List (query)
type FilterClassAttendanceSessionURLRequest struct {
	SessionID *uuid.UUID `query:"session_id" validate:"omitempty,uuid4"`
	Search    *string    `query:"search" validate:"omitempty,max=200"`
	OnlyAlive *bool      `query:"only_alive" validate:"omitempty"`

	// Pagination
	Page  *int `query:"page"  validate:"omitempty,min=1"`
	Limit *int `query:"limit" validate:"omitempty,min=1,max=200"`

	// Sorting: created_at_asc | created_at_desc (default) | label_asc | label_desc
	Sort *string `query:"sort" validate:"omitempty,oneof=created_at_asc created_at_desc label_asc label_desc"`
}

/* =========================================================
 * RESPONSE
 * ========================================================= */

type ClassAttendanceSessionURLResponse struct {
	ClassAttendanceSessionURLID  uuid.UUID `json:"class_attendance_session_url_id"`
	ClassAttendanceSessionURLMasjidID  uuid.UUID `json:"class_attendance_session_url_masjid_id"`
	ClassAttendanceSessionURLSessionID uuid.UUID `json:"class_attendance_session_url_session_id"`

	ClassAttendanceSessionURLLabel *string `json:"class_attendance_session_url_label,omitempty"`

	ClassAttendanceSessionURLHref string  `json:"class_attendance_session_url_href"`
	ClassAttendanceSessionURLTrashURL *string `json:"class_attendance_session_url_trash_url,omitempty"`
	ClassAttendanceSessionURLDeletePendingUntil *time.Time `json:"class_attendance_session_url_delete_pending_until,omitempty"`

	ClassAttendanceSessionURLCreatedAt time.Time `json:"class_attendance_session_url_created_at"`
	ClassAttendanceSessionURLUpdatedAt time.Time `json:"class_attendance_session_url_updated_at"`
}

/* =========================================================
 * HELPERS
 * ========================================================= */

func (r CreateClassAttendanceSessionURLRequest) ToModel(masjidID uuid.UUID) m.ClassAttendanceSessionURLModel {
	return m.ClassAttendanceSessionURLModel{
		ClassAttendanceSessionURLMasjidID:  masjidID,
		ClassAttendanceSessionURLSessionID: r.ClassAttendanceSessionURLSessionID,
		ClassAttendanceSessionURLLabel:     r.ClassAttendanceSessionURLLabel,
		ClassAttendanceSessionURLHref:      r.ClassAttendanceSessionURLHref,
	}
}

func NewClassAttendanceSessionURLResponse(mdl m.ClassAttendanceSessionURLModel) ClassAttendanceSessionURLResponse {
	return ClassAttendanceSessionURLResponse{
		ClassAttendanceSessionURLID:                 mdl.ClassAttendanceSessionURLID,
		ClassAttendanceSessionURLMasjidID:          mdl.ClassAttendanceSessionURLMasjidID,
		ClassAttendanceSessionURLSessionID:         mdl.ClassAttendanceSessionURLSessionID,
		ClassAttendanceSessionURLLabel:             mdl.ClassAttendanceSessionURLLabel,
		ClassAttendanceSessionURLHref:              mdl.ClassAttendanceSessionURLHref,
		ClassAttendanceSessionURLTrashURL:          mdl.ClassAttendanceSessionURLTrashURL,
		ClassAttendanceSessionURLDeletePendingUntil: mdl.ClassAttendanceSessionURLDeletePendingUntil,
		ClassAttendanceSessionURLCreatedAt:         mdl.ClassAttendanceSessionURLCreatedAt,
		ClassAttendanceSessionURLUpdatedAt:         mdl.ClassAttendanceSessionURLUpdatedAt,
	}
}
