// internals/features/lembaga/class_sections/attendance_sessions/main/dto/user_class_attendance_session_dto.go
package dto

import (
	"time"

	model "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

// Create
type CreateUserClassAttendanceSessionRequest struct {
	UserClassAttendanceSessionsSessionID   uuid.UUID             `json:"user_class_attendance_sessions_session_id" validate:"required"`
	UserClassAttendanceSessionsUserClassID uuid.UUID             `json:"user_class_attendance_sessions_user_class_id" validate:"required"`
	UserClassAttendanceSessionsMasjidID    *uuid.UUID            `json:"user_class_attendance_sessions_masjid_id" validate:"omitempty"`

	// TEXT: "present" | "sick" | "leave" | "absent"
	UserClassAttendanceSessionsAttendanceStatus model.AttendanceStatus `json:"user_class_attendance_sessions_attendance_status" validate:"required,oneof=present sick leave absent"`

	UserClassAttendanceSessionsScore       *int  `json:"user_class_attendance_sessions_score" validate:"omitempty,gte=0,lte=100"`
	UserClassAttendanceSessionsGradePassed *bool `json:"user_class_attendance_sessions_grade_passed" validate:"omitempty"`

	UserClassAttendanceSessionsMaterialPersonal *string `json:"user_class_attendance_sessions_material_personal" validate:"omitempty"`
	UserClassAttendanceSessionsPersonalNote     *string `json:"user_class_attendance_sessions_personal_note" validate:"omitempty"`
	UserClassAttendanceSessionsMemorization     *string `json:"user_class_attendance_sessions_memorization" validate:"omitempty"`
	UserClassAttendanceSessionsHomework         *string `json:"user_class_attendance_sessions_homework" validate:"omitempty"`
}

func (r *CreateUserClassAttendanceSessionRequest) ToModel() *model.UserClassAttendanceSessionModel {
	return &model.UserClassAttendanceSessionModel{
		UserClassAttendanceSessionsSessionID:        r.UserClassAttendanceSessionsSessionID,
		UserClassAttendanceSessionsUserClassID:      r.UserClassAttendanceSessionsUserClassID,
		UserClassAttendanceSessionsMasjidID:         uuidOrNil(r.UserClassAttendanceSessionsMasjidID),

		UserClassAttendanceSessionsAttendanceStatus: r.UserClassAttendanceSessionsAttendanceStatus,
		UserClassAttendanceSessionsScore:            r.UserClassAttendanceSessionsScore,
		UserClassAttendanceSessionsGradePassed:      r.UserClassAttendanceSessionsGradePassed,

		UserClassAttendanceSessionsMaterialPersonal: r.UserClassAttendanceSessionsMaterialPersonal,
		UserClassAttendanceSessionsPersonalNote:     r.UserClassAttendanceSessionsPersonalNote,
		UserClassAttendanceSessionsMemorization:     r.UserClassAttendanceSessionsMemorization,
		UserClassAttendanceSessionsHomework:         r.UserClassAttendanceSessionsHomework,
	}
}

// Update (partial)
type UpdateUserClassAttendanceSessionRequest struct {
	// TEXT: "present" | "sick" | "leave" | "absent"
	UserClassAttendanceSessionsAttendanceStatus *model.AttendanceStatus `json:"user_class_attendance_sessions_attendance_status" validate:"omitempty,oneof=present sick leave absent"`

	UserClassAttendanceSessionsScore       *int  `json:"user_class_attendance_sessions_score" validate:"omitempty,gte=0,lte=100"`
	UserClassAttendanceSessionsGradePassed *bool `json:"user_class_attendance_sessions_grade_passed" validate:"omitempty"`

	UserClassAttendanceSessionsMaterialPersonal *string `json:"user_class_attendance_sessions_material_personal" validate:"omitempty"`
	UserClassAttendanceSessionsPersonalNote     *string `json:"user_class_attendance_sessions_personal_note" validate:"omitempty"`
	UserClassAttendanceSessionsMemorization     *string `json:"user_class_attendance_sessions_memorization" validate:"omitempty"`
	UserClassAttendanceSessionsHomework         *string `json:"user_class_attendance_sessions_homework" validate:"omitempty"`
}

/* ===================== RESPONSES ===================== */

type UserClassAttendanceSessionResponse struct {
	UserClassAttendanceSessionsID               uuid.UUID             `json:"user_class_attendance_sessions_id"`
	UserClassAttendanceSessionsSessionID        uuid.UUID             `json:"user_class_attendance_sessions_session_id"`
	UserClassAttendanceSessionsUserClassID      uuid.UUID             `json:"user_class_attendance_sessions_user_class_id"`
	UserClassAttendanceSessionsMasjidID         uuid.UUID             `json:"user_class_attendance_sessions_masjid_id"`
	UserClassAttendanceSessionsAttendanceStatus model.AttendanceStatus `json:"user_class_attendance_sessions_attendance_status"`
	UserClassAttendanceSessionsScore            *int                  `json:"user_class_attendance_sessions_score,omitempty"`
	UserClassAttendanceSessionsGradePassed      *bool                 `json:"user_class_attendance_sessions_grade_passed,omitempty"`
	UserClassAttendanceSessionsMaterialPersonal *string               `json:"user_class_attendance_sessions_material_personal,omitempty"`
	UserClassAttendanceSessionsPersonalNote     *string               `json:"user_class_attendance_sessions_personal_note,omitempty"`
	UserClassAttendanceSessionsMemorization     *string               `json:"user_class_attendance_sessions_memorization,omitempty"`
	UserClassAttendanceSessionsHomework         *string               `json:"user_class_attendance_sessions_homework,omitempty"`
	UserClassAttendanceSessionsCreatedAt        time.Time             `json:"user_class_attendance_sessions_created_at"`
	UserClassAttendanceSessionsUpdatedAt        *time.Time            `json:"user_class_attendance_sessions_updated_at,omitempty"`
}

func FromUserClassAttendanceSessionModel(m model.UserClassAttendanceSessionModel) UserClassAttendanceSessionResponse {
	return UserClassAttendanceSessionResponse{
		UserClassAttendanceSessionsID:               m.UserClassAttendanceSessionsID,
		UserClassAttendanceSessionsSessionID:        m.UserClassAttendanceSessionsSessionID,
		UserClassAttendanceSessionsUserClassID:      m.UserClassAttendanceSessionsUserClassID,
		UserClassAttendanceSessionsMasjidID:         m.UserClassAttendanceSessionsMasjidID,
		UserClassAttendanceSessionsAttendanceStatus: m.UserClassAttendanceSessionsAttendanceStatus,
		UserClassAttendanceSessionsScore:            m.UserClassAttendanceSessionsScore,
		UserClassAttendanceSessionsGradePassed:      m.UserClassAttendanceSessionsGradePassed,
		UserClassAttendanceSessionsMaterialPersonal: m.UserClassAttendanceSessionsMaterialPersonal,
		UserClassAttendanceSessionsPersonalNote:     m.UserClassAttendanceSessionsPersonalNote,
		UserClassAttendanceSessionsMemorization:     m.UserClassAttendanceSessionsMemorization,
		UserClassAttendanceSessionsHomework:         m.UserClassAttendanceSessionsHomework,
		UserClassAttendanceSessionsCreatedAt:        m.UserClassAttendanceSessionsCreatedAt,
		UserClassAttendanceSessionsUpdatedAt:        m.UserClassAttendanceSessionsUpdatedAt,
	}
}

/* ===================== HELPERS ===================== */

func uuidOrNil(u *uuid.UUID) uuid.UUID {
	if u != nil {
		return *u
	}
	return uuid.Nil
}
