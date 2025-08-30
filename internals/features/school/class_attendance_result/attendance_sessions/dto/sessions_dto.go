package dto

import (
	"time"

	attendanceModel "masjidku_backend/internals/features/school/class_attendance_result/attendance_sessions/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =========================================================
   1) REQUEST DTO
   ========================================================= */

// Create
type CreateClassAttendanceSessionRequest struct {
	ClassAttendanceSessionSectionId uuid.UUID  `json:"class_attendance_session_section_id" validate:"required"`
	ClassAttendanceSessionMasjidId  uuid.UUID  `json:"class_attendance_session_masjid_id"  validate:"required"`
	ClassAttendanceSessionDate      time.Time  `json:"class_attendance_session_date"       validate:"required"`
	ClassAttendanceSessionTitle     *string    `json:"class_attendance_session_title"      validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo string   `json:"class_attendance_session_general_info" validate:"required"`
	ClassAttendanceSessionNote      *string    `json:"class_attendance_session_note"       validate:"omitempty"`

	// Integrasi kurikulum & penugasan (opsional)
	ClassAttendanceSessionClassSubjectId *uuid.UUID `json:"class_attendance_session_class_subject_id"              validate:"omitempty"`
	ClassAttendanceSessionClassSectionSubjectTeacherId *uuid.UUID `json:"class_attendance_session_class_section_subject_teacher_id" validate:"omitempty"`
}

// Update (partial)
type UpdateClassAttendanceSessionRequest struct {
	ClassAttendanceSessionSectionId *uuid.UUID `json:"class_attendance_session_section_id" validate:"omitempty"`
	ClassAttendanceSessionMasjidId  *uuid.UUID `json:"class_attendance_session_masjid_id"  validate:"omitempty"`
	ClassAttendanceSessionDate      *time.Time `json:"class_attendance_session_date"       validate:"omitempty"`
	ClassAttendanceSessionTitle     *string    `json:"class_attendance_session_title"      validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo *string  `json:"class_attendance_session_general_info" validate:"omitempty"`
	ClassAttendanceSessionNote      *string    `json:"class_attendance_session_note"       validate:"omitempty"`

	// Integrasi (opsional)
	ClassAttendanceSessionClassSubjectId *uuid.UUID `json:"class_attendance_session_class_subject_id"              validate:"omitempty"`
	ClassAttendanceSessionClassSectionSubjectTeacherId *uuid.UUID `json:"class_attendance_session_class_section_subject_teacher_id" validate:"omitempty"`
}

/*
List query (opsional untuk handler list):
- Limit/Offset default di controller
- Filter umum & sort (whitelist di controller)
*/
type ListClassAttendanceSessionQuery struct {
	Limit    *int       `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset   *int       `query:"offset" validate:"omitempty,min=0"`
	Section  *uuid.UUID `query:"section_id"          validate:"omitempty"`
	Teacher  *uuid.UUID `query:"teacher_user_id"     validate:"omitempty"`
	DateFrom *time.Time `query:"date_from"           validate:"omitempty"`
	DateTo   *time.Time `query:"date_to"             validate:"omitempty"`
	Keyword  *string    `query:"q"                   validate:"omitempty,max=100"`
	OrderBy  *string    `query:"order_by"            validate:"omitempty,oneof=date created_at title"`
	Sort     *string    `query:"sort"                validate:"omitempty,oneof=asc desc"`

	// Tambahan filter (opsional, jika dipakai di controller)
	ClassSubjectId *uuid.UUID `query:"class_subject_id"                  validate:"omitempty"`
	CsstId         *uuid.UUID `query:"class_section_subject_teacher_id"  validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO
   ========================================================= */

// internals/features/lembaga/class_sections/attendance_sessions/dto/class_attendance_session_dto.go

// internals/features/lembaga/class_sections/attendance_sessions/dto/class_attendance_session_dto.go

type ClassAttendanceSessionResponse struct {
    ClassAttendanceSessionId                uuid.UUID  `json:"class_attendance_session_id"`
    ClassAttendanceSessionSectionId         uuid.UUID  `json:"class_attendance_session_section_id"`
    ClassAttendanceSessionMasjidId          uuid.UUID  `json:"class_attendance_session_masjid_id"`
    ClassAttendanceSessionDate              time.Time  `json:"class_attendance_session_date"`
    ClassAttendanceSessionTitle             *string    `json:"class_attendance_session_title,omitempty"`
    ClassAttendanceSessionGeneralInfo       string     `json:"class_attendance_session_general_info"`
    ClassAttendanceSessionNote              *string    `json:"class_attendance_session_note,omitempty"`

    // Teacher info
    ClassAttendanceSessionTeacherUserId     *uuid.UUID `json:"class_attendance_session_teacher_user_id,omitempty"`  // <-- Added this line
    ClassAttendanceSessionTeacherName       *string    `json:"class_attendance_session_teacher_name,omitempty"`
    ClassAttendanceSessionTeacherEmail      *string    `json:"class_attendance_session_teacher_email,omitempty"`

    // Class section info
    ClassAttendanceSessionClassSubjectId                 *uuid.UUID `json:"class_attendance_session_class_subject_id,omitempty"`
    ClassAttendanceSessionClassSectionSubjectTeacherId   *uuid.UUID `json:"class_attendance_session_class_section_subject_teacher_id,omitempty"`

    ClassSectionSlug     *string        `json:"class_sections_slug,omitempty"`
    ClassSectionName     *string        `json:"class_sections_name,omitempty"`
    ClassSectionCode     *string        `json:"class_sections_code,omitempty"`
    ClassSectionCapacity *int           `json:"class_sections_capacity,omitempty"`
    ClassSectionSchedule datatypes.JSON `json:"class_sections_schedule,omitempty"`

    ClassAttendanceSessionCreatedAt time.Time  `json:"class_attendance_session_created_at"`
    ClassAttendanceSessionUpdatedAt *time.Time `json:"class_attendance_session_updated_at,omitempty"`
    ClassAttendanceSessionDeletedAt *time.Time `json:"class_attendance_session_deleted_at,omitempty"`
}


// List response + meta
type ClassAttendanceSessionListResponse struct {
	Items []ClassAttendanceSessionResponse `json:"items"`
	Meta  ListMeta                         `json:"meta"`
}

type ListMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

/* =========================================================
   3) MAPPERS
   ========================================================= */

func (r CreateClassAttendanceSessionRequest) ToModel() attendanceModel.ClassAttendanceSessionModel {
	return attendanceModel.ClassAttendanceSessionModel{
		ClassAttendanceSessionSectionId:   r.ClassAttendanceSessionSectionId,
		ClassAttendanceSessionMasjidId:    r.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionDate:        r.ClassAttendanceSessionDate,
		ClassAttendanceSessionTitle:       r.ClassAttendanceSessionTitle,
		ClassAttendanceSessionGeneralInfo: r.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:        r.ClassAttendanceSessionNote,

		// ✅ konsisten dengan kolom DB: class_attendance_sessions_class_subject_id
		ClassAttendanceSessionClassSubjectId: r.ClassAttendanceSessionClassSubjectId,

		ClassAttendanceSessionClassSectionSubjectTeacherId: r.ClassAttendanceSessionClassSectionSubjectTeacherId,
	}
}

func FromClassAttendanceSessionModel(m attendanceModel.ClassAttendanceSessionModel) ClassAttendanceSessionResponse {
	var deletedAt *time.Time
	if m.ClassAttendanceSessionDeletedAt.Valid {
		deletedAt = &m.ClassAttendanceSessionDeletedAt.Time
	}
	return ClassAttendanceSessionResponse{
		ClassAttendanceSessionId:                m.ClassAttendanceSessionId,
		ClassAttendanceSessionSectionId:         m.ClassAttendanceSessionSectionId,
		ClassAttendanceSessionMasjidId:          m.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionDate:              m.ClassAttendanceSessionDate,
		ClassAttendanceSessionTitle:             m.ClassAttendanceSessionTitle,
		ClassAttendanceSessionGeneralInfo:       m.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:              m.ClassAttendanceSessionNote,
		ClassAttendanceSessionClassSubjectId:    m.ClassAttendanceSessionClassSubjectId,
		ClassAttendanceSessionClassSectionSubjectTeacherId: m.ClassAttendanceSessionClassSectionSubjectTeacherId,
		
		ClassAttendanceSessionCreatedAt:         m.ClassAttendanceSessionCreatedAt,
		ClassAttendanceSessionUpdatedAt:         m.ClassAttendanceSessionUpdatedAt,
		ClassAttendanceSessionDeletedAt:         deletedAt,
	}
}

// Batch mapper
func FromClassAttendanceSessionModels(models []attendanceModel.ClassAttendanceSessionModel) []ClassAttendanceSessionResponse {
	out := make([]ClassAttendanceSessionResponse, 0, len(models))
	for _, m := range models {
		out = append(out, FromClassAttendanceSessionModel(m))
	}
	return out
}

/* =========================================================
   4) APPLY (partial update helper)
   ========================================================= */

func (r UpdateClassAttendanceSessionRequest) Apply(m *attendanceModel.ClassAttendanceSessionModel) {
	if r.ClassAttendanceSessionSectionId != nil {
		m.ClassAttendanceSessionSectionId = *r.ClassAttendanceSessionSectionId
	}
	if r.ClassAttendanceSessionMasjidId != nil {
		m.ClassAttendanceSessionMasjidId = *r.ClassAttendanceSessionMasjidId
	}
	if r.ClassAttendanceSessionDate != nil {
		m.ClassAttendanceSessionDate = *r.ClassAttendanceSessionDate
	}
	if r.ClassAttendanceSessionTitle != nil {
		m.ClassAttendanceSessionTitle = r.ClassAttendanceSessionTitle
	}
	if r.ClassAttendanceSessionGeneralInfo != nil {
		m.ClassAttendanceSessionGeneralInfo = *r.ClassAttendanceSessionGeneralInfo
	}
	if r.ClassAttendanceSessionNote != nil {
		m.ClassAttendanceSessionNote = r.ClassAttendanceSessionNote
	}

	if r.ClassAttendanceSessionClassSubjectId != nil {
		m.ClassAttendanceSessionClassSubjectId = r.ClassAttendanceSessionClassSubjectId
	}
	if r.ClassAttendanceSessionClassSectionSubjectTeacherId != nil {
		m.ClassAttendanceSessionClassSectionSubjectTeacherId = r.ClassAttendanceSessionClassSectionSubjectTeacherId
	}
	// UpdatedAt akan diisi otomatis oleh DB trigger / GORM autoUpdateTime.
}
