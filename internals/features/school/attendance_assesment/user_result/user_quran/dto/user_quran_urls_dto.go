// internals/features/school/attendance_assesment/user_result/user_quran_url/dto/user_quran_url_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/attendance_assesment/user_result/user_quran/model"
)

/* ===================== REQUESTS ===================== */

// Create: masjid_id diambil dari token/context di level controller
type CreateUserQuranURLRequest struct {
	UserQuranURLsRecordID uuid.UUID  `json:"user_quran_urls_record_id" validate:"required"`
	UserQuranURLsLabel    *string    `json:"user_quran_urls_label" validate:"omitempty,max=120"`
	UserQuranURLsHref     string     `json:"user_quran_urls_href" validate:"required,url"`
	UserQuranURLsTrashURL *string    `json:"user_quran_urls_trash_url" validate:"omitempty,url"`
	UserQuranURLsDeletePendingUntil *time.Time `json:"user_quran_urls_delete_pending_until" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	UserQuranURLsUploaderTeacherID  *uuid.UUID `json:"user_quran_urls_uploader_teacher_id" validate:"omitempty"`
	UserQuranURLsUploaderUserID     *uuid.UUID `json:"user_quran_urls_uploader_user_id" validate:"omitempty"`
}

// ToModel: mapping ke model
func (r CreateUserQuranURLRequest) ToModel() *model.UserQuranURLModel {
	m := &model.UserQuranURLModel{
		UserQuranURLsRecordID:        r.UserQuranURLsRecordID,
		UserQuranURLsHref:            strings.TrimSpace(r.UserQuranURLsHref),
		UserQuranURLsTrashURL:        r.UserQuranURLsTrashURL,
		UserQuranURLsDeletePendingUntil: r.UserQuranURLsDeletePendingUntil,
		UserQuranURLsUploaderTeacherID:  r.UserQuranURLsUploaderTeacherID,
		UserQuranURLsUploaderUserID:     r.UserQuranURLsUploaderUserID,
	}
	if r.UserQuranURLsLabel != nil {
		label := strings.TrimSpace(*r.UserQuranURLsLabel)
		m.UserQuranURLsLabel = &label
	}
	return m
}

/* ===================== UPDATE (partial) ===================== */

type UpdateUserQuranURLRequest struct {
	UserQuranURLsLabel    *string    `json:"user_quran_urls_label" validate:"omitempty,max=120"`
	UserQuranURLsHref     *string    `json:"user_quran_urls_href" validate:"omitempty,url"`
	UserQuranURLsTrashURL *string    `json:"user_quran_urls_trash_url" validate:"omitempty,url"`
	UserQuranURLsDeletePendingUntil *time.Time `json:"user_quran_urls_delete_pending_until" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	UserQuranURLsUploaderTeacherID  *uuid.UUID `json:"user_quran_urls_uploader_teacher_id" validate:"omitempty"`
	UserQuranURLsUploaderUserID     *uuid.UUID `json:"user_quran_urls_uploader_user_id" validate:"omitempty"`
}

// Terapkan hanya field yang dikirim
func (r *UpdateUserQuranURLRequest) ApplyToModel(m *model.UserQuranURLModel) {
	if r.UserQuranURLsLabel != nil {
		lbl := strings.TrimSpace(*r.UserQuranURLsLabel)
		m.UserQuranURLsLabel = &lbl
	}
	if r.UserQuranURLsHref != nil {
		href := strings.TrimSpace(*r.UserQuranURLsHref)
		m.UserQuranURLsHref = href
	}
	if r.UserQuranURLsTrashURL != nil {
		url := strings.TrimSpace(*r.UserQuranURLsTrashURL)
		m.UserQuranURLsTrashURL = &url
	}
	if r.UserQuranURLsDeletePendingUntil != nil {
		m.UserQuranURLsDeletePendingUntil = r.UserQuranURLsDeletePendingUntil
	}
	if r.UserQuranURLsUploaderTeacherID != nil {
		m.UserQuranURLsUploaderTeacherID = r.UserQuranURLsUploaderTeacherID
	}
	if r.UserQuranURLsUploaderUserID != nil {
		m.UserQuranURLsUploaderUserID = r.UserQuranURLsUploaderUserID
	}
}

/* ===================== QUERIES (list) ===================== */

type ListUserQuranURLQuery struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`

	RecordID   *uuid.UUID `query:"record_id"`
	TeacherID  *uuid.UUID `query:"uploader_teacher_id"`
	UserID     *uuid.UUID `query:"uploader_user_id"`
	Q          *string    `query:"q"` // search di label/href (ILIKE/trgm di layer query)

	CreatedFrom *string `query:"created_from"` // "YYYY-MM-DD"
	CreatedTo   *string `query:"created_to"`   // "YYYY-MM-DD"

	Sort *string `query:"sort"` // created_at_desc / created_at_asc
}

/* ===================== RESPONSES ===================== */

type UserQuranURLResponse struct {
	UserQuranURLsID     uuid.UUID  `json:"user_quran_urls_id"`
	UserQuranURLsRecordID uuid.UUID `json:"user_quran_urls_record_id"`

	UserQuranURLsLabel *string `json:"user_quran_urls_label,omitempty"`
	UserQuranURLsHref  string  `json:"user_quran_urls_href"`

	UserQuranURLsTrashURL           *string    `json:"user_quran_urls_trash_url,omitempty"`
	UserQuranURLsDeletePendingUntil *time.Time `json:"user_quran_urls_delete_pending_until,omitempty"`

	UserQuranURLsUploaderTeacherID *uuid.UUID `json:"user_quran_urls_uploader_teacher_id,omitempty"`
	UserQuranURLsUploaderUserID    *uuid.UUID `json:"user_quran_urls_uploader_user_id,omitempty"`

	UserQuranURLsCreatedAt time.Time `json:"user_quran_urls_created_at"`
	UserQuranURLsUpdatedAt time.Time `json:"user_quran_urls_updated_at"`
}

// Factory
func NewUserQuranURLResponse(m *model.UserQuranURLModel) *UserQuranURLResponse {
	if m == nil {
		return nil
	}
	return &UserQuranURLResponse{
		UserQuranURLsID:     m.UserQuranURLsID,
		UserQuranURLsRecordID: m.UserQuranURLsRecordID,

		UserQuranURLsLabel: m.UserQuranURLsLabel,
		UserQuranURLsHref:  m.UserQuranURLsHref,

		UserQuranURLsTrashURL:           m.UserQuranURLsTrashURL,
		UserQuranURLsDeletePendingUntil: m.UserQuranURLsDeletePendingUntil,

		UserQuranURLsUploaderTeacherID: m.UserQuranURLsUploaderTeacherID,
		UserQuranURLsUploaderUserID:    m.UserQuranURLsUploaderUserID,

		UserQuranURLsCreatedAt: m.UserQuranURLsCreatedAt,
		UserQuranURLsUpdatedAt: m.UserQuranURLsUpdatedAt,
	}
}

// Batch mapper
func FromUserQuranURLModels(rows []model.UserQuranURLModel) []UserQuranURLResponse {
	out := make([]UserQuranURLResponse, 0, len(rows))
	for i := range rows {
		r := NewUserQuranURLResponse(&rows[i])
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}
