package dto

import (
	"time"

	model "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

// Create
type CreateUserClassAttendanceEntryRequest struct {
	UserClassAttendanceEntriesSessionID    uuid.UUID              `json:"user_class_attendance_entries_session_id" validate:"required"`
	UserClassAttendanceEntriesUserClassID  uuid.UUID              `json:"user_class_attendance_entries_user_class_id" validate:"required"`
	UserClassAttendanceEntriesMasjidID     *uuid.UUID             `json:"user_class_attendance_entries_masjid_id" validate:"omitempty"`

	// SMALLINT: 0=absent,1=present,2=sick,3=leave
	UserClassAttendanceEntriesAttendanceStatus model.AttendanceStatus `json:"user_class_attendance_entries_attendance_status" validate:"required,oneof=0 1 2 3"`

	UserClassAttendanceEntriesScore         *int   `json:"user_class_attendance_entries_score" validate:"omitempty,gte=0,lte=100"`
	UserClassAttendanceEntriesGradePassed   *bool  `json:"user_class_attendance_entries_grade_passed" validate:"omitempty"`

	UserClassAttendanceEntriesMaterialPersonal *string `json:"user_class_attendance_entries_material_personal" validate:"omitempty"`
	UserClassAttendanceEntriesPersonalNote     *string `json:"user_class_attendance_entries_personal_note" validate:"omitempty"`
	UserClassAttendanceEntriesMemorization     *string `json:"user_class_attendance_entries_memorization" validate:"omitempty"`
	UserClassAttendanceEntriesHomework         *string `json:"user_class_attendance_entries_homework" validate:"omitempty"`
}

func (r *CreateUserClassAttendanceEntryRequest) ToModel() *model.UserClassAttendanceEntryModel {
	return &model.UserClassAttendanceEntryModel{
		UserClassAttendanceEntriesSessionID:        r.UserClassAttendanceEntriesSessionID,
		UserClassAttendanceEntriesUserClassID:      r.UserClassAttendanceEntriesUserClassID,
		UserClassAttendanceEntriesMasjidID:         uuidOrNil(r.UserClassAttendanceEntriesMasjidID),

		UserClassAttendanceEntriesAttendanceStatus: r.UserClassAttendanceEntriesAttendanceStatus,
		UserClassAttendanceEntriesScore:            r.UserClassAttendanceEntriesScore,
		UserClassAttendanceEntriesGradePassed:      r.UserClassAttendanceEntriesGradePassed,

		UserClassAttendanceEntriesMaterialPersonal: r.UserClassAttendanceEntriesMaterialPersonal,
		UserClassAttendanceEntriesPersonalNote:     r.UserClassAttendanceEntriesPersonalNote,
		UserClassAttendanceEntriesMemorization:     r.UserClassAttendanceEntriesMemorization,
		UserClassAttendanceEntriesHomework:         r.UserClassAttendanceEntriesHomework,
	}
}

// Update (partial, semua optional)
type UpdateUserClassAttendanceEntryRequest struct {
	// SMALLINT: 0=absent,1=present,2=sick,3=leave
	UserClassAttendanceEntriesAttendanceStatus *model.AttendanceStatus `json:"user_class_attendance_entries_attendance_status" validate:"omitempty,oneof=0 1 2 3"`

	UserClassAttendanceEntriesScore       *int  `json:"user_class_attendance_entries_score" validate:"omitempty,gte=0,lte=100"`
	UserClassAttendanceEntriesGradePassed *bool `json:"user_class_attendance_entries_grade_passed" validate:"omitempty"`

	UserClassAttendanceEntriesMaterialPersonal *string `json:"user_class_attendance_entries_material_personal" validate:"omitempty"`
	UserClassAttendanceEntriesPersonalNote     *string `json:"user_class_attendance_entries_personal_note" validate:"omitempty"`
	UserClassAttendanceEntriesMemorization     *string `json:"user_class_attendance_entries_memorization" validate:"omitempty"`
	UserClassAttendanceEntriesHomework         *string `json:"user_class_attendance_entries_homework" validate:"omitempty"`
}

/* ===================== RESPONSES ===================== */

type UserClassAttendanceEntryResponse struct {
	UserClassAttendanceEntriesID               uuid.UUID              `json:"user_class_attendance_entries_id"`
	UserClassAttendanceEntriesSessionID        uuid.UUID              `json:"user_class_attendance_entries_session_id"`
	UserClassAttendanceEntriesUserClassID      uuid.UUID              `json:"user_class_attendance_entries_user_class_id"`
	UserClassAttendanceEntriesMasjidID         uuid.UUID              `json:"user_class_attendance_entries_masjid_id"`
	UserClassAttendanceEntriesAttendanceStatus model.AttendanceStatus `json:"user_class_attendance_entries_attendance_status"`
	UserClassAttendanceEntriesScore            *int                   `json:"user_class_attendance_entries_score,omitempty"`
	UserClassAttendanceEntriesGradePassed      *bool                  `json:"user_class_attendance_entries_grade_passed,omitempty"`
	UserClassAttendanceEntriesMaterialPersonal *string                `json:"user_class_attendance_entries_material_personal,omitempty"`
	UserClassAttendanceEntriesPersonalNote     *string                `json:"user_class_attendance_entries_personal_note,omitempty"`
	UserClassAttendanceEntriesMemorization     *string                `json:"user_class_attendance_entries_memorization,omitempty"`
	UserClassAttendanceEntriesHomework         *string                `json:"user_class_attendance_entries_homework,omitempty"`
	UserClassAttendanceEntriesCreatedAt        time.Time              `json:"user_class_attendance_entries_created_at"`
	UserClassAttendanceEntriesUpdatedAt        *time.Time             `json:"user_class_attendance_entries_updated_at,omitempty"`
}

func FromUserClassAttendanceEntryModel(m model.UserClassAttendanceEntryModel) UserClassAttendanceEntryResponse {
	return UserClassAttendanceEntryResponse{
		UserClassAttendanceEntriesID:               m.UserClassAttendanceEntriesID,
		UserClassAttendanceEntriesSessionID:        m.UserClassAttendanceEntriesSessionID,
		UserClassAttendanceEntriesUserClassID:      m.UserClassAttendanceEntriesUserClassID,
		UserClassAttendanceEntriesMasjidID:         m.UserClassAttendanceEntriesMasjidID,
		UserClassAttendanceEntriesAttendanceStatus: m.UserClassAttendanceEntriesAttendanceStatus,
		UserClassAttendanceEntriesScore:            m.UserClassAttendanceEntriesScore,
		UserClassAttendanceEntriesGradePassed:      m.UserClassAttendanceEntriesGradePassed,
		UserClassAttendanceEntriesMaterialPersonal: m.UserClassAttendanceEntriesMaterialPersonal,
		UserClassAttendanceEntriesPersonalNote:     m.UserClassAttendanceEntriesPersonalNote,
		UserClassAttendanceEntriesMemorization:     m.UserClassAttendanceEntriesMemorization,
		UserClassAttendanceEntriesHomework:         m.UserClassAttendanceEntriesHomework,
		UserClassAttendanceEntriesCreatedAt:        m.UserClassAttendanceEntriesCreatedAt,
		UserClassAttendanceEntriesUpdatedAt:        m.UserClassAttendanceEntriesUpdatedAt,
	}
}

/* ===================== HELPERS ===================== */

func uuidOrNil(u *uuid.UUID) uuid.UUID {
	if u != nil {
		return *u
	}
	return uuid.Nil
}
