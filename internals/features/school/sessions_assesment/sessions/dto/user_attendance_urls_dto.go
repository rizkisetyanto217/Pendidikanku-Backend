// file: internal/dto/user_attendance_urls_dto.go
package dto

import (
	model "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"
	"time"

	"github.com/google/uuid"
)

// ===============================
// CREATE DTO
// ===============================
type CreateUserAttendanceURLRequest struct {
	// Parent Attendance
	UserAttendanceURLsAttendanceID uuid.UUID `json:"user_attendance_urls_attendance_id" validate:"required"`

	// Metadata
	UserAttendanceURLsLabel *string `json:"user_attendance_urls_label,omitempty" validate:"omitempty,max=120"`

	// URL aktif (wajib)
	UserAttendanceURLsHref string `json:"user_attendance_urls_href" validate:"required,url"`

	// Opsional uploader
	UserAttendanceURLsUploaderTeacherID *uuid.UUID `json:"user_attendance_urls_uploader_teacher_id,omitempty"`
	UserAttendanceURLsUploaderUserID    *uuid.UUID `json:"user_attendance_urls_uploader_user_id,omitempty"`
}

// ===============================
// UPDATE DTO (partial)
// ===============================
type UpdateUserAttendanceURLRequest struct {
	UserAttendanceURLsLabel              *string    `json:"user_attendance_urls_label,omitempty" validate:"omitempty,max=120"`
	UserAttendanceURLsHref               *string    `json:"user_attendance_urls_href,omitempty" validate:"omitempty,url"`
	UserAttendanceURLsTrashURL           *string    `json:"user_attendance_urls_trash_url,omitempty"`
	UserAttendanceURLsDeletePendingUntil *time.Time `json:"user_attendance_urls_delete_pending_until,omitempty"`
	UserAttendanceURLsUploaderTeacherID  *uuid.UUID `json:"user_attendance_urls_uploader_teacher_id,omitempty"`
	UserAttendanceURLsUploaderUserID     *uuid.UUID `json:"user_attendance_urls_uploader_user_id,omitempty"`
}

// ===============================
// RESPONSE DTO
// ===============================
type UserAttendanceURLResponse struct {
	UserAttendanceURLsID                 uuid.UUID  `json:"user_attendance_urls_id"`
	UserAttendanceURLsMasjidID           uuid.UUID  `json:"user_attendance_urls_masjid_id"`
	UserAttendanceURLsAttendanceID       uuid.UUID  `json:"user_attendance_urls_attendance_id"`
	UserAttendanceURLsLabel              *string    `json:"user_attendance_urls_label,omitempty"`
	UserAttendanceURLsHref               string     `json:"user_attendance_urls_href"`
	UserAttendanceURLsTrashURL           *string    `json:"user_attendance_urls_trash_url,omitempty"`
	UserAttendanceURLsDeletePendingUntil *time.Time `json:"user_attendance_urls_delete_pending_until,omitempty"`
	UserAttendanceURLsUploaderTeacherID  *uuid.UUID `json:"user_attendance_urls_uploader_teacher_id,omitempty"`
	UserAttendanceURLsUploaderUserID     *uuid.UUID `json:"user_attendance_urls_uploader_user_id,omitempty"`
	UserAttendanceURLsCreatedAt          time.Time  `json:"user_attendance_urls_created_at"`
	UserAttendanceURLsUpdatedAt          time.Time  `json:"user_attendance_urls_updated_at"`
}



func NewUserAttendanceURLModelFromCreate(req CreateUserAttendanceURLRequest, masjidID uuid.UUID) model.UserAttendanceURLModel {
	return model.UserAttendanceURLModel{
		UserAttendanceURLsMasjidID:           masjidID,
		UserAttendanceURLsAttendanceID:       req.UserAttendanceURLsAttendanceID,
		UserAttendanceURLsLabel:              req.UserAttendanceURLsLabel,
		UserAttendanceURLsHref:               req.UserAttendanceURLsHref,
		UserAttendanceURLsUploaderTeacherID:  req.UserAttendanceURLsUploaderTeacherID,
		UserAttendanceURLsUploaderUserID:     req.UserAttendanceURLsUploaderUserID,
	}
}

// Apply partial update to model (only if field supplied)
func ApplyUpdateToUserAttendanceURLModel(m *model.UserAttendanceURLModel, req UpdateUserAttendanceURLRequest) {
	now := time.Now()

	if req.UserAttendanceURLsLabel != nil {
		m.UserAttendanceURLsLabel = req.UserAttendanceURLsLabel
	}
	if req.UserAttendanceURLsHref != nil {
		m.UserAttendanceURLsHref = *req.UserAttendanceURLsHref
	}
	if req.UserAttendanceURLsTrashURL != nil {
		m.UserAttendanceURLsTrashURL = req.UserAttendanceURLsTrashURL
	}
	if req.UserAttendanceURLsDeletePendingUntil != nil {
		m.UserAttendanceURLsDeletePendingUntil = req.UserAttendanceURLsDeletePendingUntil
	}
	if req.UserAttendanceURLsUploaderTeacherID != nil {
		m.UserAttendanceURLsUploaderTeacherID = req.UserAttendanceURLsUploaderTeacherID
	}
	if req.UserAttendanceURLsUploaderUserID != nil {
		m.UserAttendanceURLsUploaderUserID = req.UserAttendanceURLsUploaderUserID
	}

	// touch updated_at (walau GORM autoUpdateTime juga set)
	m.UserAttendanceURLsUpdatedAt = now
}

func ToUserAttendanceURLResponse(m model.UserAttendanceURLModel)UserAttendanceURLResponse {
	return UserAttendanceURLResponse{
		UserAttendanceURLsID:                 m.UserAttendanceURLsID,
		UserAttendanceURLsMasjidID:           m.UserAttendanceURLsMasjidID,
		UserAttendanceURLsAttendanceID:       m.UserAttendanceURLsAttendanceID,
		UserAttendanceURLsLabel:              m.UserAttendanceURLsLabel,
		UserAttendanceURLsHref:               m.UserAttendanceURLsHref,
		UserAttendanceURLsTrashURL:           m.UserAttendanceURLsTrashURL,
		UserAttendanceURLsDeletePendingUntil: m.UserAttendanceURLsDeletePendingUntil,
		UserAttendanceURLsUploaderTeacherID:  m.UserAttendanceURLsUploaderTeacherID,
		UserAttendanceURLsUploaderUserID:     m.UserAttendanceURLsUploaderUserID,
		UserAttendanceURLsCreatedAt:          m.UserAttendanceURLsCreatedAt,
		UserAttendanceURLsUpdatedAt:          m.UserAttendanceURLsUpdatedAt,
	}
}
