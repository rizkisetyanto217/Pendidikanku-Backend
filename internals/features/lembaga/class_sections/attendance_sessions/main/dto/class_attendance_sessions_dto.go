package dto

import (
	"time"

	attendanceModel "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/model"

	"github.com/google/uuid"
)

/* ===============================
   Request DTO
=============================== */

// Create request
type CreateClassAttendanceSessionRequest struct {
	SectionID     uuid.UUID  `json:"class_attendance_sessions_section_id" validate:"required"`
	MasjidID      uuid.UUID  `json:"class_attendance_sessions_masjid_id" validate:"required"`
	Date          time.Time  `json:"class_attendance_sessions_date" validate:"required"`
	Title         *string    `json:"class_attendance_sessions_title" validate:"omitempty,max=500"`
	GeneralInfo   string     `json:"class_attendance_sessions_general_info" validate:"required"`
	Note          *string    `json:"class_attendance_sessions_note" validate:"omitempty"`
	TeacherUserID *uuid.UUID `json:"class_attendance_sessions_teacher_user_id" validate:"omitempty"`
}

// Update request (pakai pointer untuk partial update)
type UpdateClassAttendanceSessionRequest struct {
	SectionID     *uuid.UUID `json:"class_attendance_sessions_section_id" validate:"omitempty"`
	MasjidID      *uuid.UUID `json:"class_attendance_sessions_masjid_id" validate:"omitempty"`
	Date          *time.Time `json:"class_attendance_sessions_date" validate:"omitempty"`
	Title         *string    `json:"class_attendance_sessions_title" validate:"omitempty,max=500"`
	GeneralInfo   *string    `json:"class_attendance_sessions_general_info" validate:"omitempty"`
	Note          *string    `json:"class_attendance_sessions_note" validate:"omitempty"`
	TeacherUserID *uuid.UUID `json:"class_attendance_sessions_teacher_user_id" validate:"omitempty"`
}

/* ===============================
   Response DTO
=============================== */

type ClassAttendanceSessionResponse struct {
	ID           uuid.UUID  `json:"class_attendance_sessions_id"`
	SectionID    uuid.UUID  `json:"class_attendance_sessions_section_id"`
	MasjidID     uuid.UUID  `json:"class_attendance_sessions_masjid_id"`
	Date         time.Time  `json:"class_attendance_sessions_date"`
	Title        *string    `json:"class_attendance_sessions_title,omitempty"`
	GeneralInfo  string     `json:"class_attendance_sessions_general_info"`
	Note         *string    `json:"class_attendance_sessions_note,omitempty"`
	TeacherUserID *uuid.UUID `json:"class_attendance_sessions_teacher_user_id,omitempty"`
	CreatedAt    time.Time  `json:"class_attendance_sessions_created_at"`
	UpdatedAt    *time.Time `json:"class_attendance_sessions_updated_at,omitempty"`
}

/* ===============================
   Mappers
=============================== */

func (r CreateClassAttendanceSessionRequest) ToModel() attendanceModel.ClassAttendanceSessionModel {
	return attendanceModel.ClassAttendanceSessionModel{
		SectionID:     r.SectionID,
		MasjidID:      r.MasjidID,
		Date:          r.Date,
		Title:         r.Title,
		GeneralInfo:   r.GeneralInfo,
		Note:          r.Note,
		TeacherUserID: r.TeacherUserID,
	}
}

func FromClassAttendanceSessionModel(m attendanceModel.ClassAttendanceSessionModel) ClassAttendanceSessionResponse {
	return ClassAttendanceSessionResponse{
		ID:            m.ClassAttendanceSessionID,
		SectionID:     m.SectionID,
		MasjidID:      m.MasjidID,
		Date:          m.Date,
		Title:         m.Title,
		GeneralInfo:   m.GeneralInfo,
		Note:          m.Note,
		TeacherUserID: m.TeacherUserID,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
