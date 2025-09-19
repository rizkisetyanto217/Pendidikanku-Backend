// file: internals/features/school/sessions_assesment/sessions/dto/class_attendance_session_dto.go
package dto

import (
	"time"

	attendanceModel "masjidku_backend/internals/features/school/sessions/sessions/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =========================================================
   URL DTO (match ke tabel class_attendance_session_url)
   ========================================================= */

type ClassAttendanceSessionURL struct {
	ClassAttendanceSessionURLId        uuid.UUID `json:"class_attendance_session_url_id"`
	ClassAttendanceSessionURLMasjidId  uuid.UUID `json:"class_attendance_session_url_masjid_id"`
	ClassAttendanceSessionURLSessionId uuid.UUID `json:"class_attendance_session_url_session_id"`

	// Properti konten
	ClassAttendanceSessionURLKind         string  `json:"class_attendance_session_url_kind"` // e.g. banner,image,video,attachment,link
	ClassAttendanceSessionURLHref         *string `json:"class_attendance_session_url_href,omitempty"`
	ClassAttendanceSessionURLObjectKey    *string `json:"class_attendance_session_url_object_key,omitempty"`
	ClassAttendanceSessionURLObjectKeyOld *string `json:"class_attendance_session_url_object_key_old,omitempty"`
	ClassAttendanceSessionURLMime         *string `json:"class_attendance_session_url_mime,omitempty"`

	// Tampilan
	ClassAttendanceSessionURLLabel     *string `json:"class_attendance_session_url_label,omitempty"`
	ClassAttendanceSessionURLOrder     int     `json:"class_attendance_session_url_order"`
	ClassAttendanceSessionURLIsPrimary bool    `json:"class_attendance_session_url_is_primary"`

	// Audit & retensi
	ClassAttendanceSessionURLCreatedAt          time.Time  `json:"class_attendance_session_url_created_at"`
	ClassAttendanceSessionURLUpdatedAt          time.Time  `json:"class_attendance_session_url_updated_at"`
	ClassAttendanceSessionURLDeletedAt          *time.Time `json:"class_attendance_session_url_deleted_at,omitempty"`
	ClassAttendanceSessionURLDeletePendingUntil *time.Time `json:"class_attendance_session_url_delete_pending_until,omitempty"`
}

/* =========================================================
   1) REQUEST DTO
   ========================================================= */

// CREATE: pakai schedule_id (bukan CSST)
type CreateClassAttendanceSessionRequest struct {
	ClassAttendanceSessionMasjidId   uuid.UUID `json:"class_attendance_session_masjid_id"  validate:"required"`
	ClassAttendanceSessionScheduleId uuid.UUID `json:"class_attendance_session_schedule_id" validate:"required"`

	ClassAttendanceSessionTeacherId   *uuid.UUID `json:"class_attendance_session_teacher_id"    validate:"omitempty,uuid"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id" validate:"omitempty,uuid"`

	// pointer => bisa nil agar pakai DEFAULT CURRENT_DATE dari DB
	ClassAttendanceSessionDate        *time.Time `json:"class_attendance_session_date"         validate:"omitempty"`
	ClassAttendanceSessionTitle       *string    `json:"class_attendance_session_title"        validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo string     `json:"class_attendance_session_general_info" validate:"required"`
	ClassAttendanceSessionNote        *string    `json:"class_attendance_session_note"         validate:"omitempty"`
}

// UPDATE (partial)
type UpdateClassAttendanceSessionRequest struct {
	ClassAttendanceSessionMasjidId   *uuid.UUID `json:"class_attendance_session_masjid_id"  validate:"omitempty,uuid"`
	ClassAttendanceSessionScheduleId *uuid.UUID `json:"class_attendance_session_schedule_id" validate:"omitempty,uuid"`

	ClassAttendanceSessionTeacherId   *uuid.UUID `json:"class_attendance_session_teacher_id"    validate:"omitempty,uuid"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id" validate:"omitempty,uuid"`

	ClassAttendanceSessionDate        *time.Time `json:"class_attendance_session_date"         validate:"omitempty"`
	ClassAttendanceSessionTitle       *string    `json:"class_attendance_session_title"        validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo *string    `json:"class_attendance_session_general_info" validate:"omitempty"`
	ClassAttendanceSessionNote        *string    `json:"class_attendance_session_note"         validate:"omitempty"`
}

/*
List query:
- Filter utama: teacher/room/schedule/date range/keyword
- Filter turunan section/subject tetap opsional via JOIN dari schedule → csst → section/subject (di layer repo/controller)
*/
type ListClassAttendanceSessionQuery struct {
	Limit  *int `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset *int `query:"offset" validate:"omitempty,min=0"`

	// Filter utama sesuai kolom tabel
	TeacherId  *uuid.UUID `query:"teacher_id"  validate:"omitempty,uuid"`
	RoomId     *uuid.UUID `query:"room_id"     validate:"omitempty,uuid"`
	ScheduleId *uuid.UUID `query:"schedule_id" validate:"omitempty,uuid"`

	// Filter turunan via JOIN (opsional)
	SectionId      *uuid.UUID `query:"section_id"       validate:"omitempty,uuid"`
	ClassSubjectId *uuid.UUID `query:"class_subject_id" validate:"omitempty,uuid"`

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
	ClassAttendanceSessionId       uuid.UUID `json:"class_attendance_session_id"`
	ClassAttendanceSessionMasjidId uuid.UUID `json:"class_attendance_session_masjid_id"`

	// Kolom utama tabel
	ClassAttendanceSessionScheduleId  uuid.UUID  `json:"class_attendance_session_schedule_id"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id,omitempty"`

	// Tanggal sebagai non-pointer untuk tampilan
	ClassAttendanceSessionDate        time.Time `json:"class_attendance_session_date"`
	ClassAttendanceSessionTitle       *string   `json:"class_attendance_session_title,omitempty"`
	ClassAttendanceSessionGeneralInfo string    `json:"class_attendance_session_general_info"`
	ClassAttendanceSessionNote        *string   `json:"class_attendance_session_note,omitempty"`

	// Teacher
	ClassAttendanceSessionTeacherId    *uuid.UUID `json:"class_attendance_session_teacher_id,omitempty"`
	ClassAttendanceSessionTeacherName  *string    `json:"class_attendance_session_teacher_name,omitempty"`
	ClassAttendanceSessionTeacherEmail *string    `json:"class_attendance_session_teacher_email,omitempty"`

	// Rekap kehadiran
	ClassAttendanceSessionPresentCount *int `json:"class_attendance_session_present_count,omitempty"`
	ClassAttendanceSessionAbsentCount  *int `json:"class_attendance_session_absent_count,omitempty"`
	ClassAttendanceSessionLateCount    *int `json:"class_attendance_session_late_count,omitempty"`
	ClassAttendanceSessionExcusedCount *int `json:"class_attendance_session_excused_count,omitempty"`
	ClassAttendanceSessionSickCount    *int `json:"class_attendance_session_sick_count,omitempty"`
	ClassAttendanceSessionLeaveCount   *int `json:"class_attendance_session_leave_count,omitempty"`

	// Enrichment opsional (hasil JOIN, bukan kolom tabel)
	ClassSectionId       *uuid.UUID     `json:"class_sections_id,omitempty"`
	ClassSubjectId       *uuid.UUID     `json:"class_subjects_id,omitempty"`
	ClassSectionSlug     *string        `json:"class_sections_slug,omitempty"`
	ClassSectionName     *string        `json:"class_sections_name,omitempty"`
	ClassSectionCode     *string        `json:"class_sections_code,omitempty"`
	ClassSectionCapacity *int           `json:"class_sections_capacity,omitempty"`
	ClassSectionSchedule datatypes.JSON `json:"class_sections_schedule,omitempty"`

	// Audit & soft delete
	ClassAttendanceSessionCreatedAt time.Time  `json:"class_attendance_session_created_at"`
	ClassAttendanceSessionUpdatedAt time.Time  `json:"class_attendance_session_updated_at"`
	ClassAttendanceSessionDeletedAt *time.Time `json:"class_attendance_session_deleted_at,omitempty"`

	// URLs (opsional)
	ClassAttendanceSessionUrls []ClassAttendanceSessionURL `json:"class_attendance_session_urls,omitempty"`
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
		ClassAttendanceSessionMasjidId:    r.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionScheduleId:  r.ClassAttendanceSessionScheduleId,
		ClassAttendanceSessionTeacherId:   r.ClassAttendanceSessionTeacherId,
		ClassAttendanceSessionClassRoomId: r.ClassAttendanceSessionClassRoomId,
		ClassAttendanceSessionDate:        r.ClassAttendanceSessionDate, // pointer → biar DEFAULT CURRENT_DATE kepakai kalau nil
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

	// date: model *time.Time → response time.Time
	var date time.Time
	if m.ClassAttendanceSessionDate != nil {
		date = *m.ClassAttendanceSessionDate
	}

	return ClassAttendanceSessionResponse{
		ClassAttendanceSessionId:          m.ClassAttendanceSessionId,
		ClassAttendanceSessionMasjidId:    m.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionScheduleId:  m.ClassAttendanceSessionScheduleId,
		ClassAttendanceSessionClassRoomId: m.ClassAttendanceSessionClassRoomId,
		ClassAttendanceSessionDate:        date,
		ClassAttendanceSessionTitle:       m.ClassAttendanceSessionTitle,
		ClassAttendanceSessionGeneralInfo: m.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:        m.ClassAttendanceSessionNote,
		ClassAttendanceSessionTeacherId:   m.ClassAttendanceSessionTeacherId,

		// Rekap
		ClassAttendanceSessionPresentCount: m.ClassAttendanceSessionPresentCount,
		ClassAttendanceSessionAbsentCount:  m.ClassAttendanceSessionAbsentCount,
		ClassAttendanceSessionLateCount:    m.ClassAttendanceSessionLateCount,
		ClassAttendanceSessionExcusedCount: m.ClassAttendanceSessionExcusedCount,
		ClassAttendanceSessionSickCount:    m.ClassAttendanceSessionSickCount,
		ClassAttendanceSessionLeaveCount:   m.ClassAttendanceSessionLeaveCount,

		// Audit
		ClassAttendanceSessionCreatedAt: m.ClassAttendanceSessionCreatedAt,
		ClassAttendanceSessionUpdatedAt: m.ClassAttendanceSessionUpdatedAt,
		ClassAttendanceSessionDeletedAt: deletedAt,

		// Enrichment (Section/Subject, dst) isi di controller bila JOIN
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
	if r.ClassAttendanceSessionScheduleId != nil {
		m.ClassAttendanceSessionScheduleId = *r.ClassAttendanceSessionScheduleId
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
