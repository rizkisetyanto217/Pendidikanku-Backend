// file: internals/features/school/sessions_assesment/sessions/dto/class_attendance_session_dto.go
package dto

import (
	"time"

	attendanceModel "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =========================================================
   1) REQUEST DTO
   ========================================================= */

// Create
type CreateClassAttendanceSessionRequest struct {
	ClassAttendanceSessionSectionId      uuid.UUID  `json:"class_attendance_session_section_id"        validate:"required"`
	ClassAttendanceSessionMasjidId       uuid.UUID  `json:"class_attendance_session_masjid_id"         validate:"required"`
	ClassAttendanceSessionClassSubjectId  uuid.UUID  `json:"class_attendance_session_class_subject_id"   validate:"required"`
	// (baru) CSST optional
	ClassAttendanceSessionCSSTId         *uuid.UUID `json:"class_attendance_session_csst_id"           validate:"omitempty"`

	ClassAttendanceSessionTeacherId   *uuid.UUID `json:"class_attendance_session_teacher_id"         validate:"omitempty"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id"      validate:"omitempty"`

	// pointer => boleh kosong agar pakai DEFAULT CURRENT_DATE di DB
	ClassAttendanceSessionDate        *time.Time `json:"class_attendance_session_date"         validate:"omitempty"`
	ClassAttendanceSessionTitle       *string    `json:"class_attendance_session_title"        validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo string     `json:"class_attendance_session_general_info" validate:"required"`
	ClassAttendanceSessionNote        *string    `json:"class_attendance_session_note"         validate:"omitempty"`
}

// Update (partial)
type UpdateClassAttendanceSessionRequest struct {
	ClassAttendanceSessionSectionId      *uuid.UUID `json:"class_attendance_session_section_id"        validate:"omitempty"`
	ClassAttendanceSessionMasjidId       *uuid.UUID `json:"class_attendance_session_masjid_id"         validate:"omitempty"`
	ClassAttendanceSessionClassSubjectId *uuid.UUID `json:"class_attendance_session_class_subject_id"   validate:"omitempty"`
	// (baru) CSST optional
	ClassAttendanceSessionCSSTId         *uuid.UUID `json:"class_attendance_session_csst_id"           validate:"omitempty"`

	ClassAttendanceSessionTeacherId   *uuid.UUID `json:"class_attendance_session_teacher_id"         validate:"omitempty"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id"      validate:"omitempty"`

	ClassAttendanceSessionDate        *time.Time `json:"class_attendance_session_date"         validate:"omitempty"`
	ClassAttendanceSessionTitle       *string    `json:"class_attendance_session_title"        validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo *string    `json:"class_attendance_session_general_info" validate:"omitempty"`
	ClassAttendanceSessionNote        *string    `json:"class_attendance_session_note"         validate:"omitempty"`
}

/*
List query (opsional untuk handler list):
- Limit/Offset default di controller
- Filter umum & sort (whitelist di controller)
*/
type ListClassAttendanceSessionQuery struct {
	Limit          *int       `query:"limit"            validate:"omitempty,min=1,max=200"`
	Offset         *int       `query:"offset"           validate:"omitempty,min=0"`
	Section        *uuid.UUID `query:"section_id"       validate:"omitempty"`
	TeacherId      *uuid.UUID `query:"teacher_id"       validate:"omitempty"`
	ClassSubjectId *uuid.UUID `query:"class_subject_id" validate:"omitempty"`
	RoomId         *uuid.UUID `query:"room_id"          validate:"omitempty"`
	// (baru) filter CSST
	CSSTId         *uuid.UUID `query:"csst_id"          validate:"omitempty"`

	DateFrom *time.Time `query:"date_from"        validate:"omitempty"`
	DateTo   *time.Time `query:"date_to"          validate:"omitempty"`
	Keyword  *string    `query:"q"                validate:"omitempty,max=100"`

	// created_at dihapus, sesuai skema baru
	OrderBy *string `query:"order_by"         validate:"omitempty,oneof=date title"`
	Sort    *string `query:"sort"             validate:"omitempty,oneof=asc desc"`
}

/* =========================================================
   2) RESPONSE DTO
   ========================================================= */

type ClassAttendanceSessionResponse struct {
	ClassAttendanceSessionId             uuid.UUID  `json:"class_attendance_session_id"`
	ClassAttendanceSessionSectionId      uuid.UUID  `json:"class_attendance_session_section_id"`
	ClassAttendanceSessionMasjidId       uuid.UUID  `json:"class_attendance_session_masjid_id"`
	ClassAttendanceSessionClassSubjectId *uuid.UUID `json:"class_attendance_session_class_subject_id,omitempty"`

	// (baru) CSST optional
	ClassAttendanceSessionCSSTId *uuid.UUID `json:"class_attendance_session_csst_id,omitempty"`

	// Room (opsional)
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id,omitempty"`

	// date dari model adalah *time.Time; response dibuat non-pointer agar konsisten tampil
	ClassAttendanceSessionDate        time.Time  `json:"class_attendance_session_date"`
	ClassAttendanceSessionTitle       *string    `json:"class_attendance_session_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string     `json:"class_attendance_session_general_info"`
	ClassAttendanceSessionNote        *string    `json:"class_attendance_session_note,omitempty"`

	// Teacher info
	ClassAttendanceSessionTeacherId    *uuid.UUID `json:"class_attendance_session_teacher_id,omitempty"`
	ClassAttendanceSessionTeacherName  *string    `json:"class_attendance_session_teacher_name,omitempty"`
	ClassAttendanceSessionTeacherEmail *string    `json:"class_attendance_session_teacher_email,omitempty"`

	// Class section info (opsional)
	ClassSectionSlug     *string        `json:"class_sections_slug,omitempty"`
	ClassSectionName     *string        `json:"class_sections_name,omitempty"`
	ClassSectionCode     *string        `json:"class_sections_code,omitempty"`
	ClassSectionCapacity *int           `json:"class_sections_capacity,omitempty"`
	ClassSectionSchedule datatypes.JSON `json:"class_sections_schedule,omitempty"`

	// timestamps DB-side ditiadakan, hanya soft delete
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
		ClassAttendanceSessionSectionId:      r.ClassAttendanceSessionSectionId,
		ClassAttendanceSessionMasjidId:       r.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionClassSubjectId: r.ClassAttendanceSessionClassSubjectId,

		// (baru) CSST
		ClassAttendanceSessionCSSTId: r.ClassAttendanceSessionCSSTId,

		// optional
		ClassAttendanceSessionTeacherId:   r.ClassAttendanceSessionTeacherId,
		ClassAttendanceSessionClassRoomId: r.ClassAttendanceSessionClassRoomId,

		// model pakai *time.Time → langsung assign pointer dari request
		ClassAttendanceSessionDate:        r.ClassAttendanceSessionDate,
		ClassAttendanceSessionTitle:       r.ClassAttendanceSessionTitle,
		ClassAttendanceSessionGeneralInfo: r.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:        r.ClassAttendanceSessionNote,
	}
}

func FromClassAttendanceSessionModel(m attendanceModel.ClassAttendanceSessionModel) ClassAttendanceSessionResponse {
	// deleted_at
	var deletedAt *time.Time
	if m.ClassAttendanceSessionDeletedAt.Valid {
		deletedAt = &m.ClassAttendanceSessionDeletedAt.Time
	}

	// subject wajib (uuid.UUID) → DTO pointer
	subj := m.ClassAttendanceSessionClassSubjectId

	// date: model *time.Time → response time.Time (zero jika nil)
	var date time.Time
	if m.ClassAttendanceSessionDate != nil {
		date = *m.ClassAttendanceSessionDate
	}

	return ClassAttendanceSessionResponse{
		ClassAttendanceSessionId:             m.ClassAttendanceSessionId,
		ClassAttendanceSessionSectionId:      m.ClassAttendanceSessionSectionId,
		ClassAttendanceSessionMasjidId:       m.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionClassSubjectId: &subj,

		// (baru) CSST
		ClassAttendanceSessionCSSTId: m.ClassAttendanceSessionCSSTId,

		ClassAttendanceSessionClassRoomId: m.ClassAttendanceSessionClassRoomId,

		ClassAttendanceSessionDate:        date,
		ClassAttendanceSessionTitle:       m.ClassAttendanceSessionTitle,
		ClassAttendanceSessionGeneralInfo: m.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:        m.ClassAttendanceSessionNote,

		ClassAttendanceSessionTeacherId: m.ClassAttendanceSessionTeacherId,

		ClassAttendanceSessionDeletedAt: deletedAt,
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
	if r.ClassAttendanceSessionClassSubjectId != nil {
		m.ClassAttendanceSessionClassSubjectId = *r.ClassAttendanceSessionClassSubjectId
	}
	// (baru) CSST
	if r.ClassAttendanceSessionCSSTId != nil {
		m.ClassAttendanceSessionCSSTId = r.ClassAttendanceSessionCSSTId
	}

	if r.ClassAttendanceSessionTeacherId != nil {
		m.ClassAttendanceSessionTeacherId = r.ClassAttendanceSessionTeacherId
	}
	if r.ClassAttendanceSessionClassRoomId != nil {
		m.ClassAttendanceSessionClassRoomId = r.ClassAttendanceSessionClassRoomId
	}

	if r.ClassAttendanceSessionDate != nil {
		m.ClassAttendanceSessionDate = r.ClassAttendanceSessionDate
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
}
