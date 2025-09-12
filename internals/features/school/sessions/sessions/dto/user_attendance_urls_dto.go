// file: internal/dto/user_attendance_urls_dto.go
package dto

import (
	model "masjidku_backend/internals/features/school/sessions/sessions/model"
	"time"

	"github.com/google/uuid"
)

// ===============================
// CREATE DTO
// ===============================
type CreateUserAttendanceURLRequest struct {
	// Parent Attendance
	UserAttendanceURLsAttendanceID uuid.UUID  `json:"user_attendance_urls_attendance_id" validate:"required"`

	// (opsional) tipe lampiran (AUDIO/IMAGE/VIDEO/FILE) via user_attendance_type.user_attendance_type_id
	UserAttendanceTypeID *uuid.UUID `json:"user_attendance_type_id,omitempty"`

	// Metadata
	UserAttendanceURLsLabel *string `json:"user_attendance_urls_label,omitempty" validate:"omitempty,max=120"`

	// URL aktif (wajib)
	UserAttendanceURLsHref string `json:"user_attendance_urls_href" validate:"required,url"`

	// Opsional uploader
	UserAttendanceURLsUploaderTeacherID *uuid.UUID `json:"user_attendance_urls_uploader_teacher_id,omitempty"`
	UserAttendanceURLsUploaderStudentID *uuid.UUID `json:"user_attendance_urls_uploader_student_id,omitempty"`
}

// ===============================
// UPDATE DTO (partial)
// ===============================
type UpdateUserAttendanceURLRequest struct {
	UserAttendanceTypeID                 *uuid.UUID `json:"user_attendance_type_id,omitempty"`
	UserAttendanceURLsLabel              *string    `json:"user_attendance_urls_label,omitempty" validate:"omitempty,max=120"`
	UserAttendanceURLsHref               *string    `json:"user_attendance_urls_href,omitempty" validate:"omitempty,url"`
	UserAttendanceURLsTrashURL           *string    `json:"user_attendance_urls_trash_url,omitempty"`
	UserAttendanceURLsDeletePendingUntil *time.Time `json:"user_attendance_urls_delete_pending_until,omitempty"`
	UserAttendanceURLsUploaderTeacherID  *uuid.UUID `json:"user_attendance_urls_uploader_teacher_id,omitempty"`
	UserAttendanceURLsUploaderStudentID  *uuid.UUID `json:"user_attendance_urls_uploader_student_id,omitempty"`
}

// ===============================
// RESPONSE DTO
// ===============================
type UserAttendanceURLResponse struct {
	UserAttendanceURLsID                 uuid.UUID  `json:"user_attendance_urls_id"`
	UserAttendanceURLsMasjidID           uuid.UUID  `json:"user_attendance_urls_masjid_id"`
	UserAttendanceURLsAttendanceID       uuid.UUID  `json:"user_attendance_urls_attendance_id"`
	UserAttendanceTypeID                 *uuid.UUID `json:"user_attendance_type_id,omitempty"`
	UserAttendanceURLsLabel              *string    `json:"user_attendance_urls_label,omitempty"`
	UserAttendanceURLsHref               string     `json:"user_attendance_urls_href"`
	UserAttendanceURLsTrashURL           *string    `json:"user_attendance_urls_trash_url,omitempty"`
	UserAttendanceURLsDeletePendingUntil *time.Time `json:"user_attendance_urls_delete_pending_until,omitempty"`
	UserAttendanceURLsUploaderTeacherID  *uuid.UUID `json:"user_attendance_urls_uploader_teacher_id,omitempty"`
	UserAttendanceURLsUploaderStudentID  *uuid.UUID `json:"user_attendance_urls_uploader_student_id,omitempty"`
	UserAttendanceURLsCreatedAt          time.Time  `json:"user_attendance_urls_created_at"`
	UserAttendanceURLsUpdatedAt          time.Time  `json:"user_attendance_urls_updated_at"`
}

// ===============================
// Mappers
// ===============================
func NewUserAttendanceURLModelFromCreate(req CreateUserAttendanceURLRequest, masjidID uuid.UUID) model.UserAttendanceURLModel {
	return model.UserAttendanceURLModel{
		UserAttendanceURLsMasjidID:          masjidID,
		UserAttendanceURLsAttendanceID:      req.UserAttendanceURLsAttendanceID,
		UserAttendanceTypeID:                req.UserAttendanceTypeID,
		UserAttendanceURLsLabel:             req.UserAttendanceURLsLabel,
		UserAttendanceURLsHref:              req.UserAttendanceURLsHref,
		UserAttendanceURLsUploaderTeacherID: req.UserAttendanceURLsUploaderTeacherID,
		UserAttendanceURLsUploaderStudentID: req.UserAttendanceURLsUploaderStudentID,
	}
}

// Apply partial update to model (only if field supplied)
func ApplyUpdateToUserAttendanceURLModel(m *model.UserAttendanceURLModel, req UpdateUserAttendanceURLRequest) {
	now := time.Now()

	if req.UserAttendanceTypeID != nil {
		m.UserAttendanceTypeID = req.UserAttendanceTypeID
	}
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
	if req.UserAttendanceURLsUploaderStudentID != nil {
		m.UserAttendanceURLsUploaderStudentID = req.UserAttendanceURLsUploaderStudentID
	}

	// touch updated_at (walau GORM autoUpdateTime juga set)
	m.UserAttendanceURLsUpdatedAt = now
}

func ToUserAttendanceURLResponse(m model.UserAttendanceURLModel) UserAttendanceURLResponse {
	return UserAttendanceURLResponse{
		UserAttendanceURLsID:                 m.UserAttendanceURLsID,
		UserAttendanceURLsMasjidID:           m.UserAttendanceURLsMasjidID,
		UserAttendanceURLsAttendanceID:       m.UserAttendanceURLsAttendanceID,
		UserAttendanceTypeID:                 m.UserAttendanceTypeID,
		UserAttendanceURLsLabel:              m.UserAttendanceURLsLabel,
		UserAttendanceURLsHref:               m.UserAttendanceURLsHref,
		UserAttendanceURLsTrashURL:           m.UserAttendanceURLsTrashURL,
		UserAttendanceURLsDeletePendingUntil: m.UserAttendanceURLsDeletePendingUntil,
		UserAttendanceURLsUploaderTeacherID:  m.UserAttendanceURLsUploaderTeacherID,
		UserAttendanceURLsUploaderStudentID:  m.UserAttendanceURLsUploaderStudentID,
		UserAttendanceURLsCreatedAt:          m.UserAttendanceURLsCreatedAt,
		UserAttendanceURLsUpdatedAt:          m.UserAttendanceURLsUpdatedAt,
	}
}
