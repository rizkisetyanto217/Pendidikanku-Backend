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

// CREATE: tanpa section_id & class_subject_id; pakai CSST (wajib)
type CreateClassAttendanceSessionRequest struct {
	ClassAttendanceSessionMasjidId uuid.UUID `json:"class_attendance_session_masjid_id"  validate:"required"`
	ClassAttendanceSessionCSSTId   uuid.UUID `json:"class_attendance_session_csst_id"    validate:"required"`

	ClassAttendanceSessionTeacherId   *uuid.UUID `json:"class_attendance_session_teacher_id"    validate:"omitempty,uuid"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id" validate:"omitempty,uuid"`

	// pointer => bisa nil agar pakai DEFAULT CURRENT_DATE dari DB
	ClassAttendanceSessionDate        *time.Time `json:"class_attendance_session_date"         validate:"omitempty"`
	ClassAttendanceSessionTitle       *string    `json:"class_attendance_session_title"        validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo string     `json:"class_attendance_session_general_info" validate:"required"`
	ClassAttendanceSessionNote        *string    `json:"class_attendance_session_note"         validate:"omitempty"`
}

// UPDATE (partial): tetap tanpa section_id & class_subject_id
type UpdateClassAttendanceSessionRequest struct {
	ClassAttendanceSessionMasjidId *uuid.UUID `json:"class_attendance_session_masjid_id"  validate:"omitempty,uuid"`
	ClassAttendanceSessionCSSTId   *uuid.UUID `json:"class_attendance_session_csst_id"    validate:"omitempty,uuid"`

	ClassAttendanceSessionTeacherId   *uuid.UUID `json:"class_attendance_session_teacher_id"    validate:"omitempty,uuid"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id" validate:"omitempty,uuid"`

	ClassAttendanceSessionDate        *time.Time `json:"class_attendance_session_date"         validate:"omitempty"`
	ClassAttendanceSessionTitle       *string    `json:"class_attendance_session_title"        validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo *string    `json:"class_attendance_session_general_info" validate:"omitempty"`
	ClassAttendanceSessionNote        *string    `json:"class_attendance_session_note"         validate:"omitempty"`
}

/*
List query (untuk handler list):
- Filter bawaan (tanpa kolom yang dihapus dari tabel).
- section_id & class_subject_id tetap disediakan SEBAGAI FILTER OPSIONAL
  (kalau di controller kamu melakukan JOIN via CSST → section/subject).
*/
type ListClassAttendanceSessionQuery struct {
	Limit  *int `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset *int `query:"offset" validate:"omitempty,min=0"`

	// Filter utama sesuai kolom tabel
	TeacherId *uuid.UUID `query:"teacher_id" validate:"omitempty,uuid"`
	RoomId    *uuid.UUID `query:"room_id"    validate:"omitempty,uuid"`
	CSSTId    *uuid.UUID `query:"csst_id"    validate:"omitempty,uuid"`

	// Filter turunan via JOIN (opsional)
	SectionId      *uuid.UUID `query:"section_id"       validate:"omitempty,uuid"`
	ClassSubjectId *uuid.UUID `query:"class_subject_id"  validate:"omitempty,uuid"`

	DateFrom *time.Time `query:"date_from" validate:"omitempty"`
	DateTo   *time.Time `query:"date_to"   validate:"omitempty"`
	Keyword  *string    `query:"q"         validate:"omitempty,max=100"`

	OrderBy *string `query:"order_by" validate:"omitempty,oneof=date title"`
	Sort    *string `query:"sort"     validate:"omitempty,oneof=asc desc"`
}

/* =========================================================
   2) RESPONSE DTO
   ========================================================= */

type ClassAttendanceSessionResponse struct {
	ClassAttendanceSessionId        uuid.UUID  `json:"class_attendance_session_id"`
	ClassAttendanceSessionMasjidId  uuid.UUID  `json:"class_attendance_session_masjid_id"`

	// Kolom utama tabel
	ClassAttendanceSessionCSSTId      uuid.UUID  `json:"class_attendance_session_csst_id"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id,omitempty"`

	// Tanggal sebagai non-pointer untuk tampilan; (zero jika memang NULL di DB — mestinya tidak, ada DEFAULT)
	ClassAttendanceSessionDate        time.Time  `json:"class_attendance_session_date"`
	ClassAttendanceSessionTitle       *string    `json:"class_attendance_session_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string     `json:"class_attendance_session_general_info"`
	ClassAttendanceSessionNote        *string    `json:"class_attendance_session_note,omitempty"`

	// Teacher
	ClassAttendanceSessionTeacherId    *uuid.UUID `json:"class_attendance_session_teacher_id,omitempty"`
	ClassAttendanceSessionTeacherName  *string    `json:"class_attendance_session_teacher_name,omitempty"`
	ClassAttendanceSessionTeacherEmail *string    `json:"class_attendance_session_teacher_email,omitempty"`

	// Enrichment opsional (hasil JOIN, bukan kolom tabel)
	ClassSectionId        *uuid.UUID     `json:"class_sections_id,omitempty"`
	ClassSubjectId        *uuid.UUID     `json:"class_subjects_id,omitempty"`
	ClassSectionSlug      *string        `json:"class_sections_slug,omitempty"`
	ClassSectionName      *string        `json:"class_sections_name,omitempty"`
	ClassSectionCode      *string        `json:"class_sections_code,omitempty"`
	ClassSectionCapacity  *int           `json:"class_sections_capacity,omitempty"`
	ClassSectionSchedule  datatypes.JSON `json:"class_sections_schedule,omitempty"`

	// Soft delete
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
		ClassAttendanceSessionMasjidId:       r.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionCSSTId:         r.ClassAttendanceSessionCSSTId,
		ClassAttendanceSessionTeacherId:      r.ClassAttendanceSessionTeacherId,
		ClassAttendanceSessionClassRoomId:    r.ClassAttendanceSessionClassRoomId,
		ClassAttendanceSessionDate:           r.ClassAttendanceSessionDate,  // pointer → biar DEFAULT CURRENT_DATE kepakai kalau nil
		ClassAttendanceSessionTitle:          r.ClassAttendanceSessionTitle,
		ClassAttendanceSessionGeneralInfo:    r.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:           r.ClassAttendanceSessionNote,
	}
}

func FromClassAttendanceSessionModel(m attendanceModel.ClassAttendanceSessionModel) ClassAttendanceSessionResponse {
	// deleted_at
	var deletedAt *time.Time
	if m.ClassAttendanceSessionDeletedAt.Valid {
		deletedAt = &m.ClassAttendanceSessionDeletedAt.Time
	}

	// date: model *time.Time → response time.Time
	var date time.Time
	if m.ClassAttendanceSessionDate != nil {
		date = *m.ClassAttendanceSessionDate
	}

	return ClassAttendanceSessionResponse{
		ClassAttendanceSessionId:             m.ClassAttendanceSessionId,
		ClassAttendanceSessionMasjidId:       m.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionCSSTId:         m.ClassAttendanceSessionCSSTId,
		ClassAttendanceSessionClassRoomId:    m.ClassAttendanceSessionClassRoomId,
		ClassAttendanceSessionDate:           date,
		ClassAttendanceSessionTitle:          m.ClassAttendanceSessionTitle,
		ClassAttendanceSessionGeneralInfo:    m.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:           m.ClassAttendanceSessionNote,
		ClassAttendanceSessionTeacherId:      m.ClassAttendanceSessionTeacherId,
		ClassAttendanceSessionDeletedAt:      deletedAt,
		// Enrichment opsional (Section/Subject, Name, dsb) diisi di controller jika melakukan JOIN
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
	if r.ClassAttendanceSessionMasjidId != nil {
		m.ClassAttendanceSessionMasjidId = *r.ClassAttendanceSessionMasjidId
	}
	if r.ClassAttendanceSessionCSSTId != nil {
		m.ClassAttendanceSessionCSSTId = *r.ClassAttendanceSessionCSSTId
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
