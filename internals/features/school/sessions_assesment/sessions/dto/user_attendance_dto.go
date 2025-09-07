// internals/features/school/attendance_assesment/user_result/user_attendance/dto/user_attendance_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"
)

/* ===================== HELPERS ===================== */

/* ===================== REQUESTS ===================== */

// Create: masjid_id diambil dari token/context (bukan dari body)
type CreateUserAttendanceRequest struct {
	UserAttendanceSessionID      uuid.UUID `json:"user_attendance_session_id" validate:"required,uuid"`
	UserAttendanceMasjidStudentID uuid.UUID `json:"user_attendance_masjid_student_id" validate:"required,uuid"`

	// default 'present' jika kosong
	UserAttendanceStatus *string    `json:"user_attendance_status" validate:"omitempty,oneof=present absent excused late"`
	UserAttendanceTypeID *uuid.UUID `json:"user_attendance_type_id" validate:"omitempty,uuid"`

	// Ringkasan catatan Qur'an harian
	UserAttendanceDesc     *string  `json:"user_attendance_desc" validate:"omitempty"`
	UserAttendanceScore    *float64 `json:"user_attendance_score" validate:"omitempty,gte=0,lte=100"`
	UserAttendanceIsPassed *bool    `json:"user_attendance_is_passed" validate:"omitempty"`

	// Notes (nullable)
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
		UserAttendanceMasjidID:       masjidID,
		UserAttendanceSessionID:      r.UserAttendanceSessionID,
		UserAttendanceMasjidStudentID: r.UserAttendanceMasjidStudentID,
		UserAttendanceStatus:         status,
	}

	// Optional fields
	if r.UserAttendanceTypeID != nil {
		m.UserAttendanceTypeID = r.UserAttendanceTypeID
	}
	if r.UserAttendanceDesc != nil {
		m.UserAttendanceDesc = trimPtr(r.UserAttendanceDesc)
	}
	if r.UserAttendanceScore != nil {
		m.UserAttendanceScore = r.UserAttendanceScore
	}
	if r.UserAttendanceIsPassed != nil {
		m.UserAttendanceIsPassed = r.UserAttendanceIsPassed
	}
	if r.UserAttendanceUserNote != nil {
		m.UserAttendanceUserNote = trimPtr(r.UserAttendanceUserNote)
	}
	if r.UserAttendanceTeacherNote != nil {
		m.UserAttendanceTeacherNote = trimPtr(r.UserAttendanceTeacherNote)
	}
	return m
}

/* ===================== UPDATE (partial) ===================== */

type UpdateUserAttendanceRequest struct {
	UserAttendanceSessionID      *uuid.UUID `json:"user_attendance_session_id" validate:"omitempty,uuid"`
	UserAttendanceMasjidStudentID *uuid.UUID `json:"user_attendance_masjid_student_id" validate:"omitempty,uuid"`

	UserAttendanceStatus *string    `json:"user_attendance_status" validate:"omitempty,oneof=present absent excused late"`
	UserAttendanceTypeID *uuid.UUID `json:"user_attendance_type_id" validate:"omitempty,uuid"`

	UserAttendanceDesc     *string  `json:"user_attendance_desc" validate:"omitempty"`
	UserAttendanceScore    *float64 `json:"user_attendance_score" validate:"omitempty,gte=0,lte=100"`
	UserAttendanceIsPassed *bool    `json:"user_attendance_is_passed" validate:"omitempty"`

	UserAttendanceUserNote    *string `json:"user_attendance_user_note" validate:"omitempty"`
	UserAttendanceTeacherNote *string `json:"user_attendance_teacher_note" validate:"omitempty"`
}

// Terapkan hanya field yang dikirim.
// Catatan: pointer nil TIDAK menghapus kolom menjadi NULL (tidak melakukan clear).
func (r *UpdateUserAttendanceRequest) ApplyToModel(m *model.UserAttendanceModel) {
	if r.UserAttendanceSessionID != nil {
		m.UserAttendanceSessionID = *r.UserAttendanceSessionID
	}
	if r.UserAttendanceMasjidStudentID != nil {
		m.UserAttendanceMasjidStudentID = *r.UserAttendanceMasjidStudentID
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
	if r.UserAttendanceTypeID != nil {
		m.UserAttendanceTypeID = r.UserAttendanceTypeID
	}
	if r.UserAttendanceDesc != nil {
		m.UserAttendanceDesc = trimPtr(r.UserAttendanceDesc)
	}
	if r.UserAttendanceScore != nil {
		m.UserAttendanceScore = r.UserAttendanceScore
	}
	if r.UserAttendanceIsPassed != nil {
		m.UserAttendanceIsPassed = r.UserAttendanceIsPassed
	}
	if r.UserAttendanceUserNote != nil {
		m.UserAttendanceUserNote = trimPtr(r.UserAttendanceUserNote)
	}
	if r.UserAttendanceTeacherNote != nil {
		m.UserAttendanceTeacherNote = trimPtr(r.UserAttendanceTeacherNote)
	}
}

/* ===================== QUERIES (list) ===================== */

type ListUserAttendanceQuery struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`

	SessionID *uuid.UUID `query:"session_id"`
	StudentID *uuid.UUID `query:"student_id"`
	TypeID    *uuid.UUID `query:"type_id"`

	Status *string `query:"status"` // present/absent/excused/late

	// filter nilai
	ScoreFrom *float64 `query:"score_from"`
	ScoreTo   *float64 `query:"score_to"`
	IsPassed  *bool    `query:"is_passed"`

	CreatedFrom *string `query:"created_from"` // "YYYY-MM-DD"
	CreatedTo   *string `query:"created_to"`   // "YYYY-MM-DD"

	Sort *string `query:"sort"` // created_at_desc / created_at_asc
}

/* ===================== RESPONSES ===================== */

type UserAttendanceResponse struct {
	UserAttendanceID           uuid.UUID `json:"user_attendance_id"`
	UserAttendanceMasjidID     uuid.UUID `json:"user_attendance_masjid_id"`
	UserAttendanceSessionID    uuid.UUID `json:"user_attendance_session_id"`
	UserAttendanceMasjidStudentID uuid.UUID `json:"user_attendance_masjid_student_id"`

	UserAttendanceStatus string     `json:"user_attendance_status"`
	UserAttendanceTypeID *uuid.UUID `json:"user_attendance_type_id,omitempty"`

	UserAttendanceDesc     *string  `json:"user_attendance_desc,omitempty"`
	UserAttendanceScore    *float64 `json:"user_attendance_score,omitempty"`
	UserAttendanceIsPassed *bool    `json:"user_attendance_is_passed,omitempty"`

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
		UserAttendanceID:           m.UserAttendanceID,
		UserAttendanceMasjidID:     m.UserAttendanceMasjidID,
		UserAttendanceSessionID:    m.UserAttendanceSessionID,
		UserAttendanceMasjidStudentID: m.UserAttendanceMasjidStudentID,

		UserAttendanceStatus: string(m.UserAttendanceStatus),
		UserAttendanceTypeID: m.UserAttendanceTypeID,

		UserAttendanceDesc:     m.UserAttendanceDesc,
		UserAttendanceScore:    m.UserAttendanceScore,
		UserAttendanceIsPassed: m.UserAttendanceIsPassed,

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
		if r := NewUserAttendanceResponse(&rows[i]); r != nil {
			out = append(out, *r)
		}
	}
	return out
}
