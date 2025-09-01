// internals/features/school/attendance_assesment/user_result/user_attendance/dto/user_attendance_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/attendance_assesment/attendance_sessions/model"
)

/* ===================== REQUESTS ===================== */

// Create: masjid_id diambil dari token/context (bukan dari body)
type CreateUserAttendanceRequest struct {
	UserAttendanceSessionID uuid.UUID `json:"user_attendance_session_id" validate:"required"`
	UserAttendanceUserID    uuid.UUID `json:"user_attendance_user_id" validate:"required"`

	// default 'present' jika kosong
	UserAttendanceStatus *string `json:"user_attendance_status" validate:"omitempty,oneof=present absent excused late"`

	UserAttendanceUserNote    *string `json:"user_attendance_user_note" validate:"omitempty"`
	UserAttendanceTeacherNote *string `json:"user_attendance_teacher_note" validate:"omitempty"`
}

// ToModel: controller akan menyuplai masjidID dari token
func (r CreateUserAttendanceRequest) ToModel(masjidID uuid.UUID) *model.UserAttendanceModel {
	// status default
	status := model.UserAttendancePresent
	if r.UserAttendanceStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.UserAttendanceStatus)) {
		case "present":
			status = model.UserAttendancePresent
		case "absent":
			status = model.UserAttendanceAbsent
		case "excused":
			status = model.UserAttendanceExcused
		case "late":
			status = model.UserAttendanceLate
		}
	}

	m := &model.UserAttendanceModel{
		UserAttendanceMasjidID:  masjidID,
		UserAttendanceSessionID: r.UserAttendanceSessionID,
		UserAttendanceUserID:    r.UserAttendanceUserID,
		UserAttendanceStatus:    status,
	}

	if r.UserAttendanceUserNote != nil {
		v := strings.TrimSpace(*r.UserAttendanceUserNote)
		m.UserAttendanceUserNote = &v
	}
	if r.UserAttendanceTeacherNote != nil {
		v := strings.TrimSpace(*r.UserAttendanceTeacherNote)
		m.UserAttendanceTeacherNote = &v
	}
	return m
}

/* ===================== UPDATE (partial) ===================== */

type UpdateUserAttendanceRequest struct {
	UserAttendanceSessionID *uuid.UUID `json:"user_attendance_session_id" validate:"omitempty"`
	UserAttendanceUserID    *uuid.UUID `json:"user_attendance_user_id" validate:"omitempty"`

	UserAttendanceStatus *string `json:"user_attendance_status" validate:"omitempty,oneof=present absent excused late"`

	UserAttendanceUserNote    *string `json:"user_attendance_user_note" validate:"omitempty"`
	UserAttendanceTeacherNote *string `json:"user_attendance_teacher_note" validate:"omitempty"`
}

// Terapkan hanya field yang dikirim.
// Catatan: untuk kolom note (pointer), DTO ini TIDAK melakukan clear ke NULL via JSON null,
// mengikuti pola DTO lain. Jika butuh clear ke NULL, bisa ditangani khusus di controller.
func (r *UpdateUserAttendanceRequest) ApplyToModel(m *model.UserAttendanceModel) {
	if r.UserAttendanceSessionID != nil {
		m.UserAttendanceSessionID = *r.UserAttendanceSessionID
	}
	if r.UserAttendanceUserID != nil {
		m.UserAttendanceUserID = *r.UserAttendanceUserID
	}
	if r.UserAttendanceStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.UserAttendanceStatus)) {
		case "present":
			m.UserAttendanceStatus = model.UserAttendancePresent
		case "absent":
			m.UserAttendanceStatus = model.UserAttendanceAbsent
		case "excused":
			m.UserAttendanceStatus = model.UserAttendanceExcused
		case "late":
			m.UserAttendanceStatus = model.UserAttendanceLate
		}
	}
	if r.UserAttendanceUserNote != nil {
		v := strings.TrimSpace(*r.UserAttendanceUserNote)
		m.UserAttendanceUserNote = &v
	}
	if r.UserAttendanceTeacherNote != nil {
		v := strings.TrimSpace(*r.UserAttendanceTeacherNote)
		m.UserAttendanceTeacherNote = &v
	}
}

/* ===================== QUERIES (list) ===================== */

type ListUserAttendanceQuery struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`

	SessionID *uuid.UUID `query:"session_id"`
	UserID    *uuid.UUID `query:"user_id"`
	Status    *string    `query:"status"` // present/absent/excused/late

	CreatedFrom *string `query:"created_from"` // "YYYY-MM-DD"
	CreatedTo   *string `query:"created_to"`   // "YYYY-MM-DD"

	Sort *string `query:"sort"` // created_at_desc / created_at_asc
}

/* ===================== RESPONSES ===================== */

type UserAttendanceResponse struct {
	UserAttendanceID        uuid.UUID `json:"user_attendance_id"`
	UserAttendanceMasjidID  uuid.UUID `json:"user_attendance_masjid_id"`
	UserAttendanceSessionID uuid.UUID `json:"user_attendance_session_id"`
	UserAttendanceUserID    uuid.UUID `json:"user_attendance_user_id"`

	UserAttendanceStatus string  `json:"user_attendance_status"`
	UserAttendanceUserNote    *string `json:"user_attendance_user_note,omitempty"`
	UserAttendanceTeacherNote *string `json:"user_attendance_teacher_note,omitempty"`

	UserAttendanceCreatedAt time.Time `json:"user_attendance_created_at"`
	UserAttendanceUpdatedAt time.Time `json:"user_attendance_updated_at"`
}

// Factory
func NewUserAttendanceResponse(m *model.UserAttendanceModel) *UserAttendanceResponse {
	if m == nil {
		return nil
	}
	return &UserAttendanceResponse{
		UserAttendanceID:        m.UserAttendanceID,
		UserAttendanceMasjidID:  m.UserAttendanceMasjidID,
		UserAttendanceSessionID: m.UserAttendanceSessionID,
		UserAttendanceUserID:    m.UserAttendanceUserID,

		UserAttendanceStatus: string(m.UserAttendanceStatus),
		UserAttendanceUserNote:    m.UserAttendanceUserNote,
		UserAttendanceTeacherNote: m.UserAttendanceTeacherNote,

		UserAttendanceCreatedAt: m.UserAttendanceCreatedAt,
		UserAttendanceUpdatedAt: m.UserAttendanceUpdatedAt,
	}
}

// Batch mapper
func FromUserAttendanceModels(rows []model.UserAttendanceModel) []UserAttendanceResponse {
	out := make([]UserAttendanceResponse, 0, len(rows))
	for i := range rows {
		r := NewUserAttendanceResponse(&rows[i])
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}
