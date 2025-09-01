// internals/features/lembaga/user_quran_records/dto/user_quran_record_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/attendance_assesment/user_result/user_quran/model"
)

/* ===================== REQUESTS ===================== */

// Create: masjid_id diambil dari token/context (bukan dari body).
type CreateUserQuranRecordRequest struct {
	UserQuranRecordUserID        uuid.UUID  `json:"user_quran_record_user_id" validate:"required"`
	UserQuranRecordSessionID     *uuid.UUID `json:"user_quran_record_session_id" validate:"omitempty"`
	UserQuranRecordTeacherUserID *uuid.UUID `json:"user_quran_record_teacher_user_id" validate:"omitempty"`

	UserQuranRecordSourceKind *string  `json:"user_quran_record_source_kind" validate:"omitempty,max=24"`
	UserQuranRecordScope       *string `json:"user_quran_record_scope" validate:"omitempty"`
	UserQuranRecordUserNote    *string `json:"user_quran_record_user_note" validate:"omitempty"`
	UserQuranRecordTeacherNote *string `json:"user_quran_record_teacher_note" validate:"omitempty"`

	// ✅ baru sesuai skema
	UserQuranRecordScore  *float64 `json:"user_quran_record_score" validate:"omitempty,gte=0,lte=100"`
	UserQuranRecordIsNext *bool    `json:"user_quran_record_is_next" validate:"omitempty"`
}

// ToModel: controller akan menyuplai masjidID dari token
func (r CreateUserQuranRecordRequest) ToModel(masjidID uuid.UUID) *model.UserQuranRecordModel {
	m := &model.UserQuranRecordModel{
		UserQuranRecordMasjidID:      masjidID,
		UserQuranRecordUserID:        r.UserQuranRecordUserID,
		UserQuranRecordSessionID:     r.UserQuranRecordSessionID,
		UserQuranRecordTeacherUserID: r.UserQuranRecordTeacherUserID,
		UserQuranRecordScore:         r.UserQuranRecordScore,
		UserQuranRecordIsNext:        r.UserQuranRecordIsNext,
	}

	// Trim & set bila ada
	if r.UserQuranRecordSourceKind != nil {
		s := strings.TrimSpace(*r.UserQuranRecordSourceKind)
		m.UserQuranRecordSourceKind = &s
	}
	if r.UserQuranRecordScope != nil {
		s := strings.TrimSpace(*r.UserQuranRecordScope)
		m.UserQuranRecordScope = &s
	}
	if r.UserQuranRecordUserNote != nil {
		s := strings.TrimSpace(*r.UserQuranRecordUserNote)
		m.UserQuranRecordUserNote = &s
	}
	if r.UserQuranRecordTeacherNote != nil {
		s := strings.TrimSpace(*r.UserQuranRecordTeacherNote)
		m.UserQuranRecordTeacherNote = &s
	}

	return m
}

/* ===================== UPDATE (partial) ===================== */

type UpdateUserQuranRecordRequest struct {
	UserQuranRecordUserID        *uuid.UUID `json:"user_quran_record_user_id" validate:"omitempty"`
	UserQuranRecordSessionID     *uuid.UUID `json:"user_quran_record_session_id" validate:"omitempty"` // kirim null untuk clear
	UserQuranRecordTeacherUserID *uuid.UUID `json:"user_quran_record_teacher_user_id" validate:"omitempty"`

	UserQuranRecordSourceKind *string  `json:"user_quran_record_source_kind" validate:"omitempty,max=24"`
	UserQuranRecordScope       *string  `json:"user_quran_record_scope" validate:"omitempty"`
	UserQuranRecordUserNote    *string  `json:"user_quran_record_user_note" validate:"omitempty"`
	UserQuranRecordTeacherNote *string  `json:"user_quran_record_teacher_note" validate:"omitempty"`

	// ✅ baru
	UserQuranRecordScore  *float64 `json:"user_quran_record_score" validate:"omitempty,gte=0,lte=100"`
	UserQuranRecordIsNext *bool    `json:"user_quran_record_is_next" validate:"omitempty"`
}

// Terapkan hanya field yang dikirim (nullable jadi bisa clear ke NULL dengan kirim JSON null).
func (r *UpdateUserQuranRecordRequest) ApplyToModel(m *model.UserQuranRecordModel) {
	if r.UserQuranRecordUserID != nil {
		m.UserQuranRecordUserID = *r.UserQuranRecordUserID
	}
	if r.UserQuranRecordSessionID != nil {
		m.UserQuranRecordSessionID = r.UserQuranRecordSessionID
	}
	if r.UserQuranRecordTeacherUserID != nil {
		m.UserQuranRecordTeacherUserID = r.UserQuranRecordTeacherUserID
	}

	if r.UserQuranRecordSourceKind != nil {
		s := strings.TrimSpace(*r.UserQuranRecordSourceKind)
		m.UserQuranRecordSourceKind = &s
	}
	if r.UserQuranRecordScope != nil {
		s := strings.TrimSpace(*r.UserQuranRecordScope)
		m.UserQuranRecordScope = &s
	}
	if r.UserQuranRecordUserNote != nil {
		s := strings.TrimSpace(*r.UserQuranRecordUserNote)
		m.UserQuranRecordUserNote = &s
	}
	if r.UserQuranRecordTeacherNote != nil {
		s := strings.TrimSpace(*r.UserQuranRecordTeacherNote)
		m.UserQuranRecordTeacherNote = &s
	}

	// ✅ baru
	if r.UserQuranRecordScore != nil {
		m.UserQuranRecordScore = r.UserQuranRecordScore
	}
	if r.UserQuranRecordIsNext != nil {
		m.UserQuranRecordIsNext = r.UserQuranRecordIsNext
	}
}

/* ===================== QUERIES (list) ===================== */

type ListUserQuranRecordQuery struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`

	UserID      *uuid.UUID `query:"user_id"`
	SessionID   *uuid.UUID `query:"session_id"`
	TeacherUser *uuid.UUID `query:"teacher_user_id"`

	SourceKind *string `query:"source_kind"`
	Q          *string `query:"q"` // search di scope (ILIKE / trigram oleh layer query)

	// ✅ filter baru selaras skema
	IsNext   *bool    `query:"is_next"`
	ScoreMin *float64 `query:"score_min"` // optional: gte
	ScoreMax *float64 `query:"score_max"` // optional: lte

	CreatedFrom *string `query:"created_from"` // "YYYY-MM-DD"
	CreatedTo   *string `query:"created_to"`   // "YYYY-MM-DD"

	Sort *string `query:"sort"` // e.g. created_at_desc / created_at_asc / score_desc / score_asc
}

/* ===================== RESPONSES ===================== */

type UserQuranRecordResponse struct {
	UserQuranRecordID            uuid.UUID  `json:"user_quran_record_id"`
	UserQuranRecordMasjidID      uuid.UUID  `json:"user_quran_record_masjid_id"`
	UserQuranRecordUserID        uuid.UUID  `json:"user_quran_record_user_id"`
	UserQuranRecordSessionID     *uuid.UUID `json:"user_quran_record_session_id,omitempty"`
	UserQuranRecordTeacherUserID *uuid.UUID `json:"user_quran_record_teacher_user_id,omitempty"`

	UserQuranRecordSourceKind *string `json:"user_quran_record_source_kind,omitempty"`
	UserQuranRecordScope       *string `json:"user_quran_record_scope,omitempty"`

	UserQuranRecordUserNote    *string `json:"user_quran_record_user_note,omitempty"`
	UserQuranRecordTeacherNote *string `json:"user_quran_record_teacher_note,omitempty"`

	// ✅ baru
	UserQuranRecordScore  *float64 `json:"user_quran_record_score,omitempty"`
	UserQuranRecordIsNext *bool    `json:"user_quran_record_is_next,omitempty"`

	UserQuranRecordCreatedAt time.Time `json:"user_quran_record_created_at"`
	UserQuranRecordUpdatedAt time.Time `json:"user_quran_record_updated_at"`
}

// Factory
func NewUserQuranRecordResponse(m *model.UserQuranRecordModel) *UserQuranRecordResponse {
	if m == nil {
		return nil
	}
	return &UserQuranRecordResponse{
		UserQuranRecordID:            m.UserQuranRecordID,
		UserQuranRecordMasjidID:      m.UserQuranRecordMasjidID,
		UserQuranRecordUserID:        m.UserQuranRecordUserID,
		UserQuranRecordSessionID:     m.UserQuranRecordSessionID,
		UserQuranRecordTeacherUserID: m.UserQuranRecordTeacherUserID,

		UserQuranRecordSourceKind: m.UserQuranRecordSourceKind,
		UserQuranRecordScope:      m.UserQuranRecordScope,

		UserQuranRecordUserNote:    m.UserQuranRecordUserNote,
		UserQuranRecordTeacherNote: m.UserQuranRecordTeacherNote,

		UserQuranRecordScore:  m.UserQuranRecordScore,
		UserQuranRecordIsNext: m.UserQuranRecordIsNext,

		UserQuranRecordCreatedAt: m.UserQuranRecordCreatedAt,
		UserQuranRecordUpdatedAt: m.UserQuranRecordUpdatedAt,
	}
}

// Batch mapper
func FromUserQuranRecordModels(rows []model.UserQuranRecordModel) []UserQuranRecordResponse {
	out := make([]UserQuranRecordResponse, 0, len(rows))
	for i := range rows {
		r := NewUserQuranRecordResponse(&rows[i])
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}
