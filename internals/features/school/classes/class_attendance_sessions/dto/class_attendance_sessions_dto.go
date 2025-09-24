// file: internals/features/school/attendance/sessions/dto/class_attendance_session_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/classes/class_attendance_sessions/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* ========================================================
   0) PatchFieldSessions[T] — tri-state (absent|null|value)
   ======================================================== */

type PatchFieldSessions[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchFieldSessions[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	if string(b) == "null" {
		p.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}
func (p PatchFieldSessions[T]) Get() (*T, bool) { return p.Value, p.Present }

/* ========================================================
   1) URL DTOs (tabel: class_attendance_session_urls)
   ======================================================== */

type ClassAttendanceSessionURLLite struct {
	ID        uuid.UUID `json:"class_attendance_session_url_id"`
	Label     *string   `json:"class_attendance_session_url_label,omitempty"`
	Href      string    `json:"class_attendance_session_url_href"`
	Kind      string    `json:"class_attendance_session_url_kind"`
	IsPrimary bool      `json:"class_attendance_session_url_is_primary"`
	Order     int       `json:"class_attendance_session_url_order"`
}

func ToClassAttendanceSessionURLLite(m *model.ClassAttendanceSessionURLModel) ClassAttendanceSessionURLLite {
	href := ""
	if m.ClassAttendanceSessionURLHref != nil {
		href = *m.ClassAttendanceSessionURLHref
	}
	return ClassAttendanceSessionURLLite{
		ID:        m.ClassAttendanceSessionURLID,
		Label:     m.ClassAttendanceSessionURLLabel,
		Href:      href,
		Kind:      m.ClassAttendanceSessionURLKind,
		IsPrimary: m.ClassAttendanceSessionURLIsPrimary,
		Order:     m.ClassAttendanceSessionURLOrder,
	}
}

// Upsert
type ClassAttendanceSessionURLUpsert struct {
	Kind      string  `json:"class_attendance_session_url_kind" validate:"required,min=1,max=24"` // banner|image|video|attachment|link|...
	Label     *string `json:"class_attendance_session_url_label,omitempty" validate:"omitempty,max=160"`
	Href      *string `json:"class_attendance_session_url_href,omitempty" validate:"omitempty,url"`
	ObjectKey *string `json:"class_attendance_session_url_object_key,omitempty" validate:"omitempty"`
	Order     int     `json:"class_attendance_session_url_order"`
	IsPrimary bool    `json:"class_attendance_session_url_is_primary"`
}

func (u *ClassAttendanceSessionURLUpsert) Normalize() {
	u.Kind = strings.TrimSpace(u.Kind)
	if u.Kind == "" {
		u.Kind = "attachment"
	}
	if u.Label != nil {
		v := strings.TrimSpace(*u.Label)
		if v == "" {
			u.Label = nil
		} else {
			u.Label = &v
		}
	}
	if u.Href != nil {
		v := strings.TrimSpace(*u.Href)
		if v == "" {
			u.Href = nil
		} else {
			u.Href = &v
		}
	}
	if u.ObjectKey != nil {
		v := strings.TrimSpace(*u.ObjectKey)
		if v == "" {
			u.ObjectKey = nil
		} else {
			u.ObjectKey = &v
		}
	}
}

// Patch
type ClassAttendanceSessionURLPatch struct {
	ID        uuid.UUID `json:"class_attendance_session_url_id" validate:"required"`
	Label     *string   `json:"class_attendance_session_url_label,omitempty" validate:"omitempty,max=160"`
	Order     *int      `json:"class_attendance_session_url_order,omitempty"`
	IsPrimary *bool     `json:"class_attendance_session_url_is_primary,omitempty"`
	Kind      *string   `json:"class_attendance_session_url_kind,omitempty" validate:"omitempty,max=24"`
	Href      *string   `json:"class_attendance_session_url_href,omitempty" validate:"omitempty,url"`
	ObjectKey *string   `json:"class_attendance_session_url_object_key,omitempty" validate:"omitempty"`
}

func (p *ClassAttendanceSessionURLPatch) Normalize() {
	trim := func(s *string) *string {
		if s == nil {
			return nil
		}
		v := strings.TrimSpace(*s)
		if v == "" {
			return nil
		}
		return &v
	}
	p.Label = trim(p.Label)
	p.Kind = trim(p.Kind)
	p.Href = trim(p.Href)
	p.ObjectKey = trim(p.ObjectKey)
}

// Bulk create
type ClassAttendanceSessionURLCreateRequest struct {
	MasjidID  uuid.UUID                         `json:"class_attendance_session_url_masjid_id" validate:"required"`
	SessionID uuid.UUID                         `json:"class_attendance_session_url_session_id" validate:"required"`
	URLs      []ClassAttendanceSessionURLUpsert `json:"urls" validate:"required,dive"`
}

func (r *ClassAttendanceSessionURLCreateRequest) Normalize() {
	for i := range r.URLs {
		r.URLs[i].Normalize()
	}
}
func (r *ClassAttendanceSessionURLCreateRequest) ToModels() []model.ClassAttendanceSessionURLModel {
	out := make([]model.ClassAttendanceSessionURLModel, 0, len(r.URLs))
	for _, u := range r.URLs {
		row := model.ClassAttendanceSessionURLModel{
			ClassAttendanceSessionURLMasjidID:  r.MasjidID,
			ClassAttendanceSessionURLSessionID: r.SessionID,
			ClassAttendanceSessionURLKind:      u.Kind,
			ClassAttendanceSessionURLLabel:     u.Label,
			ClassAttendanceSessionURLHref:      u.Href,
			ClassAttendanceSessionURLObjectKey: u.ObjectKey,
			ClassAttendanceSessionURLOrder:     u.Order,
			ClassAttendanceSessionURLIsPrimary: u.IsPrimary,
		}
		if strings.TrimSpace(row.ClassAttendanceSessionURLKind) == "" {
			row.ClassAttendanceSessionURLKind = "attachment"
		}
		out = append(out, row)
	}
	return out
}

/* ========================================================
   2) SESSION REQUEST DTOs (CREATE / UPDATE / LIST)
   ======================================================== */

// CREATE
type CreateClassAttendanceSessionRequest struct {
	// Required
	ClassAttendanceSessionMasjidId   uuid.UUID `json:"class_attendance_session_masjid_id"  validate:"required"`
	ClassAttendanceSessionScheduleId uuid.UUID `json:"class_attendance_session_schedule_id" validate:"required"`

	// Optional — occurrence
	ClassAttendanceSessionDate     *time.Time `json:"class_attendance_session_date"       validate:"omitempty"`
	ClassAttendanceSessionStartsAt *time.Time `json:"class_attendance_session_starts_at"  validate:"omitempty"`
	ClassAttendanceSessionEndsAt   *time.Time `json:"class_attendance_session_ends_at"    validate:"omitempty"`

	// Optional — identity & meta
	ClassAttendanceSessionSlug        *string `json:"class_attendance_session_slug"         validate:"omitempty,max=160"`
	ClassAttendanceSessionTitle       *string `json:"class_attendance_session_title"        validate:"omitempty,max=500"`
	ClassAttendanceSessionGeneralInfo string  `json:"class_attendance_session_general_info" validate:"required"`
	ClassAttendanceSessionNote        *string `json:"class_attendance_session_note"         validate:"omitempty"`

	// Optional — lifecycle
	ClassAttendanceSessionStatus           *string `json:"class_attendance_session_status"            validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	ClassAttendanceSessionAttendanceStatus *string `json:"class_attendance_session_attendance_status" validate:"omitempty,oneof=open closed"`
	ClassAttendanceSessionLocked           *bool   `json:"class_attendance_session_locked"            validate:"omitempty"`

	// Optional — overrides
	ClassAttendanceSessionIsOverride      *bool      `json:"class_attendance_session_is_override"       validate:"omitempty"`
	ClassAttendanceSessionIsCanceled      *bool      `json:"class_attendance_session_is_canceled"       validate:"omitempty"`
	ClassAttendanceSessionOriginalStartAt *time.Time `json:"class_attendance_session_original_start_at" validate:"omitempty"`
	ClassAttendanceSessionOriginalEndAt   *time.Time `json:"class_attendance_session_original_end_at"   validate:"omitempty"`
	ClassAttendanceSessionKind            *string    `json:"class_attendance_session_kind"              validate:"omitempty"`
	ClassAttendanceSessionOverrideReason  *string    `json:"class_attendance_session_override_reason"   validate:"omitempty"`

	// Optional — single override event
	ClassAttendanceSessionOverrideEventId *uuid.UUID `json:"class_attendance_session_override_event_id" validate:"omitempty,uuid"`

	// Optional — override resources
	ClassAttendanceSessionTeacherId   *uuid.UUID `json:"class_attendance_session_teacher_id"    validate:"omitempty,uuid"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id" validate:"omitempty,uuid"`
	ClassAttendanceSessionCSSTId      *uuid.UUID `json:"class_attendance_session_csst_id"       validate:"omitempty,uuid"`

	// Optional — create URLs together
	URLs []ClassAttendanceSessionURLUpsert `json:"urls" validate:"omitempty,dive"`
}

// UPDATE (PATCH)
type UpdateClassAttendanceSessionRequest struct {
	// Simple
	ClassAttendanceSessionMasjidId   *uuid.UUID `json:"class_attendance_session_masjid_id"  validate:"omitempty,uuid"`
	ClassAttendanceSessionScheduleId *uuid.UUID `json:"class_attendance_session_schedule_id" validate:"omitempty,uuid"`

	// Tri-state time
	ClassAttendanceSessionDate     PatchFieldSessions[time.Time] `json:"class_attendance_session_date"`
	ClassAttendanceSessionStartsAt PatchFieldSessions[time.Time] `json:"class_attendance_session_starts_at"`
	ClassAttendanceSessionEndsAt   PatchFieldSessions[time.Time] `json:"class_attendance_session_ends_at"`

	// Identity & meta
	ClassAttendanceSessionSlug        PatchFieldSessions[string] `json:"class_attendance_session_slug"`
	ClassAttendanceSessionTitle       PatchFieldSessions[string] `json:"class_attendance_session_title"`
	ClassAttendanceSessionGeneralInfo PatchFieldSessions[string] `json:"class_attendance_session_general_info"`
	ClassAttendanceSessionNote        PatchFieldSessions[string] `json:"class_attendance_session_note"`

	// Lifecycle
	ClassAttendanceSessionStatus           PatchFieldSessions[string] `json:"class_attendance_session_status"`
	ClassAttendanceSessionAttendanceStatus PatchFieldSessions[string] `json:"class_attendance_session_attendance_status"`
	ClassAttendanceSessionLocked           PatchFieldSessions[bool]   `json:"class_attendance_session_locked"`

	// Overrides
	ClassAttendanceSessionIsOverride      PatchFieldSessions[bool]      `json:"class_attendance_session_is_override"`
	ClassAttendanceSessionIsCanceled      PatchFieldSessions[bool]      `json:"class_attendance_session_is_canceled"`
	ClassAttendanceSessionOriginalStartAt PatchFieldSessions[time.Time] `json:"class_attendance_session_original_start_at"`
	ClassAttendanceSessionOriginalEndAt   PatchFieldSessions[time.Time] `json:"class_attendance_session_original_end_at"`
	ClassAttendanceSessionKind            PatchFieldSessions[string]    `json:"class_attendance_session_kind"`
	ClassAttendanceSessionOverrideReason  PatchFieldSessions[string]    `json:"class_attendance_session_override_reason"`

	// Single override event
	ClassAttendanceSessionOverrideEventId PatchFieldSessions[uuid.UUID] `json:"class_attendance_session_override_event_id"`

	// Override resources
	ClassAttendanceSessionTeacherId   PatchFieldSessions[uuid.UUID] `json:"class_attendance_session_teacher_id"`
	ClassAttendanceSessionClassRoomId PatchFieldSessions[uuid.UUID] `json:"class_attendance_session_class_room_id"`
	ClassAttendanceSessionCSSTId      PatchFieldSessions[uuid.UUID] `json:"class_attendance_session_csst_id"`

	// URL ops
	URLsAdd    []ClassAttendanceSessionURLUpsert `json:"urls_add" validate:"omitempty,dive"`
	URLsPatch  []ClassAttendanceSessionURLPatch  `json:"urls_patch" validate:"omitempty,dive"`
	URLsDelete []uuid.UUID                       `json:"urls_delete" validate:"omitempty,dive,unique"`
}

// List query
type ListClassAttendanceSessionQuery struct {
	Limit  *int `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset *int `query:"offset" validate:"omitempty,min=0"`

	// Filter utama
	TeacherId  *uuid.UUID `query:"teacher_id"  validate:"omitempty,uuid"`
	ScheduleId *uuid.UUID `query:"schedule_id" validate:"omitempty,uuid"`

	// Filter tambahan
	RoomId  *uuid.UUID `query:"room_id"  validate:"omitempty,uuid"`
	CSSTId  *uuid.UUID `query:"csst_id"  validate:"omitempty,uuid"`
	EventId *uuid.UUID `query:"event_id" validate:"omitempty,uuid"`

	Status           *string `query:"status"            validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	AttendanceStatus *string `query:"attendance_status" validate:"omitempty,oneof=open closed"`
	IsLocked         *bool   `query:"locked"            validate:"omitempty"`
	IsOverride       *bool   `query:"is_override"       validate:"omitempty"`
	IsCanceled       *bool   `query:"is_canceled"       validate:"omitempty"`

	DateFrom *time.Time `query:"date_from" validate:"omitempty"`
	DateTo   *time.Time `query:"date_to"   validate:"omitempty"`
	Keyword  *string    `query:"q"         validate:"omitempty,max=100"`

	OrderBy *string `query:"order_by" validate:"omitempty,oneof=date title created_at"`
	Sort    *string `query:"sort"     validate:"omitempty,oneof=asc desc"`
}

/* ========================================================
   3) SESSION RESPONSE DTOs
   ======================================================== */

type ClassAttendanceSessionResponse struct {
	ClassAttendanceSessionId       uuid.UUID `json:"class_attendance_session_id"`
	ClassAttendanceSessionMasjidId uuid.UUID `json:"class_attendance_session_masjid_id"`

	// Tabel utama
	ClassAttendanceSessionScheduleId uuid.UUID `json:"class_attendance_session_schedule_id"`

	// Identity
	ClassAttendanceSessionSlug  *string `json:"class_attendance_session_slug,omitempty"`
	ClassAttendanceSessionTitle *string `json:"class_attendance_session_title,omitempty"`

	// Occurrence
	ClassAttendanceSessionDate     time.Time  `json:"class_attendance_session_date"`
	ClassAttendanceSessionStartsAt *time.Time `json:"class_attendance_session_starts_at,omitempty"`
	ClassAttendanceSessionEndsAt   *time.Time `json:"class_attendance_session_ends_at,omitempty"`

	// Lifecycle
	ClassAttendanceSessionStatus           string `json:"class_attendance_session_status"`
	ClassAttendanceSessionAttendanceStatus string `json:"class_attendance_session_attendance_status"`
	ClassAttendanceSessionLocked           bool   `json:"class_attendance_session_locked"`

	// Overrides
	ClassAttendanceSessionIsOverride      bool       `json:"class_attendance_session_is_override"`
	ClassAttendanceSessionIsCanceled      bool       `json:"class_attendance_session_is_canceled"`
	ClassAttendanceSessionOriginalStartAt *time.Time `json:"class_attendance_session_original_start_at,omitempty"`
	ClassAttendanceSessionOriginalEndAt   *time.Time `json:"class_attendance_session_original_end_at,omitempty"`
	ClassAttendanceSessionKind            *string    `json:"class_attendance_session_kind,omitempty"`
	ClassAttendanceSessionOverrideReason  *string    `json:"class_attendance_session_override_reason,omitempty"`

	// Single override event
	ClassAttendanceSessionOverrideEventId *uuid.UUID `json:"class_attendance_session_override_event_id,omitempty"`

	// Override resources
	ClassAttendanceSessionTeacherId   *uuid.UUID `json:"class_attendance_session_teacher_id,omitempty"`
	ClassAttendanceSessionClassRoomId *uuid.UUID `json:"class_attendance_session_class_room_id,omitempty"`
	ClassAttendanceSessionCSSTId      *uuid.UUID `json:"class_attendance_session_csst_id,omitempty"`

	// Enrichment guru (opsional)
	ClassAttendanceSessionTeacherName  *string `json:"class_attendance_session_teacher_name,omitempty"`
	ClassAttendanceSessionTeacherEmail *string `json:"class_attendance_session_teacher_email,omitempty"`

	// Info & rekap
	ClassAttendanceSessionGeneralInfo string  `json:"class_attendance_session_general_info"`
	ClassAttendanceSessionNote        *string `json:"class_attendance_session_note,omitempty"`

	ClassAttendanceSessionPresentCount *int `json:"class_attendance_session_present_count,omitempty"`
	ClassAttendanceSessionAbsentCount  *int `json:"class_attendance_session_absent_count,omitempty"`
	ClassAttendanceSessionLateCount    *int `json:"class_attendance_session_late_count,omitempty"`
	ClassAttendanceSessionExcusedCount *int `json:"class_attendance_session_excused_count,omitempty"`
	ClassAttendanceSessionSickCount    *int `json:"class_attendance_session_sick_count,omitempty"`
	ClassAttendanceSessionLeaveCount   *int `json:"class_attendance_session_leave_count,omitempty"`

	// Enrichment opsional (hasil JOIN)
	ClassSectionId       *uuid.UUID     `json:"class_sections_id,omitempty"`
	ClassSubjectId       *uuid.UUID     `json:"class_subjects_id,omitempty"`
	ClassSectionSlug     *string        `json:"class_sections_slug,omitempty"`
	ClassSectionName     *string        `json:"class_sections_name,omitempty"`
	ClassSectionCode     *string        `json:"class_sections_code,omitempty"`
	ClassSectionCapacity *int           `json:"class_sections_capacity,omitempty"`
	ClassSectionSchedule datatypes.JSON `json:"class_sections_schedule,omitempty"`

	// Audit & soft delete (opsional)
	ClassAttendanceSessionCreatedAt time.Time  `json:"class_attendance_session_created_at"`
	ClassAttendanceSessionUpdatedAt time.Time  `json:"class_attendance_session_updated_at"`
	ClassAttendanceSessionDeletedAt *time.Time `json:"class_attendance_session_deleted_at,omitempty"`

	// URLs (opsional)
	ClassAttendanceSessionUrls []ClassAttendanceSessionURLLite `json:"class_attendance_session_urls,omitempty"`
}

type ClassAttendanceSessionListResponse struct {
	Items []ClassAttendanceSessionResponse `json:"items"`
	Meta  ListMeta                         `json:"meta"`
}

type ListMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

/* ========================================================
   4) Mapping: Create/Read
   ======================================================== */

func (r CreateClassAttendanceSessionRequest) ToModel() model.ClassAttendanceSessionModel {
	// date: deref atau fallback ke today (NOT NULL di DB)
	var date time.Time
	if r.ClassAttendanceSessionDate != nil {
		date = *r.ClassAttendanceSessionDate
	} else {
		now := time.Now().In(time.Local)
		date = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	m := model.ClassAttendanceSessionModel{
		ClassAttendanceSessionMasjidID:   r.ClassAttendanceSessionMasjidId,
		ClassAttendanceSessionScheduleID: r.ClassAttendanceSessionScheduleId,

		ClassAttendanceSessionDate:     date,
		ClassAttendanceSessionStartsAt: r.ClassAttendanceSessionStartsAt,
		ClassAttendanceSessionEndsAt:   r.ClassAttendanceSessionEndsAt,

		ClassAttendanceSessionSlug:        r.ClassAttendanceSessionSlug,
		ClassAttendanceSessionTitle:       r.ClassAttendanceSessionTitle,
		ClassAttendanceSessionGeneralInfo: r.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:        r.ClassAttendanceSessionNote,

		// Overrides & resources
		ClassAttendanceSessionOriginalStartAt: r.ClassAttendanceSessionOriginalStartAt,
		ClassAttendanceSessionOriginalEndAt:   r.ClassAttendanceSessionOriginalEndAt,
		ClassAttendanceSessionKind:            r.ClassAttendanceSessionKind,
		ClassAttendanceSessionOverrideReason:  r.ClassAttendanceSessionOverrideReason,

		ClassAttendanceSessionOverrideEventID: r.ClassAttendanceSessionOverrideEventId,

		ClassAttendanceSessionTeacherID:   r.ClassAttendanceSessionTeacherId,
		ClassAttendanceSessionClassRoomID: r.ClassAttendanceSessionClassRoomId,
		ClassAttendanceSessionCSSTID:      r.ClassAttendanceSessionCSSTId,
	}

	// Lifecycle (opsional)
	if r.ClassAttendanceSessionStatus != nil {
		m.ClassAttendanceSessionStatus = model.SessionStatus(*r.ClassAttendanceSessionStatus)
	}
	if r.ClassAttendanceSessionAttendanceStatus != nil {
		m.ClassAttendanceSessionAttendanceStatus = model.AttendanceStatus(*r.ClassAttendanceSessionAttendanceStatus)
	}
	if r.ClassAttendanceSessionLocked != nil {
		m.ClassAttendanceSessionLocked = *r.ClassAttendanceSessionLocked
	}
	if r.ClassAttendanceSessionIsOverride != nil {
		m.ClassAttendanceSessionIsOverride = *r.ClassAttendanceSessionIsOverride
	}
	if r.ClassAttendanceSessionIsCanceled != nil {
		m.ClassAttendanceSessionIsCanceled = *r.ClassAttendanceSessionIsCanceled
	}
	return m
}

func FromClassAttendanceSessionModel(m model.ClassAttendanceSessionModel) ClassAttendanceSessionResponse {
	// deleted_at
	var deletedAt *time.Time
	if m.ClassAttendanceSessionDeletedAt.Valid {
		deletedAt = &m.ClassAttendanceSessionDeletedAt.Time
	}

	return ClassAttendanceSessionResponse{
		ClassAttendanceSessionId:         m.ClassAttendanceSessionID,
		ClassAttendanceSessionMasjidId:   m.ClassAttendanceSessionMasjidID,
		ClassAttendanceSessionScheduleId: m.ClassAttendanceSessionScheduleID,

		ClassAttendanceSessionSlug:  m.ClassAttendanceSessionSlug,
		ClassAttendanceSessionTitle: m.ClassAttendanceSessionTitle,

		ClassAttendanceSessionDate:     m.ClassAttendanceSessionDate,
		ClassAttendanceSessionStartsAt: m.ClassAttendanceSessionStartsAt,
		ClassAttendanceSessionEndsAt:   m.ClassAttendanceSessionEndsAt,

		ClassAttendanceSessionStatus:           string(m.ClassAttendanceSessionStatus),
		ClassAttendanceSessionAttendanceStatus: string(m.ClassAttendanceSessionAttendanceStatus),
		ClassAttendanceSessionLocked:           m.ClassAttendanceSessionLocked,

		ClassAttendanceSessionIsOverride:      m.ClassAttendanceSessionIsOverride,
		ClassAttendanceSessionIsCanceled:      m.ClassAttendanceSessionIsCanceled,
		ClassAttendanceSessionOriginalStartAt: m.ClassAttendanceSessionOriginalStartAt,
		ClassAttendanceSessionOriginalEndAt:   m.ClassAttendanceSessionOriginalEndAt,
		ClassAttendanceSessionKind:            m.ClassAttendanceSessionKind,
		ClassAttendanceSessionOverrideReason:  m.ClassAttendanceSessionOverrideReason,

		ClassAttendanceSessionOverrideEventId: m.ClassAttendanceSessionOverrideEventID,

		ClassAttendanceSessionTeacherId:   m.ClassAttendanceSessionTeacherID,
		ClassAttendanceSessionClassRoomId: m.ClassAttendanceSessionClassRoomID,
		ClassAttendanceSessionCSSTId:      m.ClassAttendanceSessionCSSTID,

		ClassAttendanceSessionGeneralInfo: m.ClassAttendanceSessionGeneralInfo,
		ClassAttendanceSessionNote:        m.ClassAttendanceSessionNote,

		ClassAttendanceSessionPresentCount: m.ClassAttendanceSessionPresentCount,
		ClassAttendanceSessionAbsentCount:  m.ClassAttendanceSessionAbsentCount,
		ClassAttendanceSessionLateCount:    m.ClassAttendanceSessionLateCount,
		ClassAttendanceSessionExcusedCount: m.ClassAttendanceSessionExcusedCount,
		ClassAttendanceSessionSickCount:    m.ClassAttendanceSessionSickCount,
		ClassAttendanceSessionLeaveCount:   m.ClassAttendanceSessionLeaveCount,

		ClassAttendanceSessionCreatedAt: m.ClassAttendanceSessionCreatedAt,
		ClassAttendanceSessionUpdatedAt: m.ClassAttendanceSessionUpdatedAt,
		ClassAttendanceSessionDeletedAt: deletedAt,
	}
}

func FromClassAttendanceSessionModels(models []model.ClassAttendanceSessionModel) []ClassAttendanceSessionResponse {
	out := make([]ClassAttendanceSessionResponse, 0, len(models))
	for _, m := range models {
		out = append(out, FromClassAttendanceSessionModel(m))
	}
	return out
}

/* ========================================================
   5) APPLY (PATCH → Model) + helpers URL ops
   ======================================================== */

func (r UpdateClassAttendanceSessionRequest) Apply(m *model.ClassAttendanceSessionModel) {
	// Simple
	if r.ClassAttendanceSessionMasjidId != nil {
		m.ClassAttendanceSessionMasjidID = *r.ClassAttendanceSessionMasjidId
	}
	if r.ClassAttendanceSessionScheduleId != nil {
		m.ClassAttendanceSessionScheduleID = *r.ClassAttendanceSessionScheduleId
	}

	// Time
	if v, ok := r.ClassAttendanceSessionDate.Get(); ok {
		if v != nil {
			m.ClassAttendanceSessionDate = *v // NOT NULL
		}
	}
	if v, ok := r.ClassAttendanceSessionStartsAt.Get(); ok {
		m.ClassAttendanceSessionStartsAt = v
	}
	if v, ok := r.ClassAttendanceSessionEndsAt.Get(); ok {
		m.ClassAttendanceSessionEndsAt = v
	}

	// Identity & meta
	if v, ok := r.ClassAttendanceSessionSlug.Get(); ok {
		m.ClassAttendanceSessionSlug = v
	}
	if v, ok := r.ClassAttendanceSessionTitle.Get(); ok {
		m.ClassAttendanceSessionTitle = v
	}
	if v, ok := r.ClassAttendanceSessionGeneralInfo.Get(); ok {
		if v == nil {
			empty := ""
			m.ClassAttendanceSessionGeneralInfo = empty
		} else {
			m.ClassAttendanceSessionGeneralInfo = *v
		}
	}
	if v, ok := r.ClassAttendanceSessionNote.Get(); ok {
		m.ClassAttendanceSessionNote = v
	}

	// Lifecycle
	if v, ok := r.ClassAttendanceSessionStatus.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionStatus = model.SessionStatusScheduled
		} else {
			m.ClassAttendanceSessionStatus = model.SessionStatus(*v)
		}
	}
	if v, ok := r.ClassAttendanceSessionAttendanceStatus.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionAttendanceStatus = model.AttendanceStatusOpen
		} else {
			m.ClassAttendanceSessionAttendanceStatus = model.AttendanceStatus(*v)
		}
	}
	if v, ok := r.ClassAttendanceSessionLocked.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionLocked = false
		} else {
			m.ClassAttendanceSessionLocked = *v
		}
	}

	// Overrides
	if v, ok := r.ClassAttendanceSessionIsOverride.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionIsOverride = false
		} else {
			m.ClassAttendanceSessionIsOverride = *v
		}
	}
	if v, ok := r.ClassAttendanceSessionIsCanceled.Get(); ok {
		if v == nil {
			m.ClassAttendanceSessionIsCanceled = false
		} else {
			m.ClassAttendanceSessionIsCanceled = *v
		}
	}
	if v, ok := r.ClassAttendanceSessionOriginalStartAt.Get(); ok {
		m.ClassAttendanceSessionOriginalStartAt = v
	}
	if v, ok := r.ClassAttendanceSessionOriginalEndAt.Get(); ok {
		m.ClassAttendanceSessionOriginalEndAt = v
	}
	if v, ok := r.ClassAttendanceSessionKind.Get(); ok {
		m.ClassAttendanceSessionKind = v
	}
	if v, ok := r.ClassAttendanceSessionOverrideReason.Get(); ok {
		m.ClassAttendanceSessionOverrideReason = v
	}

	// Single override event
	if v, ok := r.ClassAttendanceSessionOverrideEventId.Get(); ok {
		m.ClassAttendanceSessionOverrideEventID = v
	}

	// Override resources
	if v, ok := r.ClassAttendanceSessionTeacherId.Get(); ok {
		m.ClassAttendanceSessionTeacherID = v
	}
	if v, ok := r.ClassAttendanceSessionClassRoomId.Get(); ok {
		m.ClassAttendanceSessionClassRoomID = v
	}
	if v, ok := r.ClassAttendanceSessionCSSTId.Get(); ok {
		m.ClassAttendanceSessionCSSTID = v
	}
}

// Helper: build URL rows untuk URLsAdd
func (r UpdateClassAttendanceSessionRequest) URLsAddToModels(masjidID, sessionID uuid.UUID) []model.ClassAttendanceSessionURLModel {
	if len(r.URLsAdd) == 0 {
		return nil
	}
	out := make([]model.ClassAttendanceSessionURLModel, 0, len(r.URLsAdd))
	for _, u := range r.URLsAdd {
		u.Normalize()
		row := model.ClassAttendanceSessionURLModel{
			ClassAttendanceSessionURLMasjidID:  masjidID,
			ClassAttendanceSessionURLSessionID: sessionID,
			ClassAttendanceSessionURLKind:      u.Kind,
			ClassAttendanceSessionURLLabel:     u.Label,
			ClassAttendanceSessionURLHref:      u.Href,
			ClassAttendanceSessionURLObjectKey: u.ObjectKey,
			ClassAttendanceSessionURLOrder:     u.Order,
			ClassAttendanceSessionURLIsPrimary: u.IsPrimary,
		}
		if strings.TrimSpace(row.ClassAttendanceSessionURLKind) == "" {
			row.ClassAttendanceSessionURLKind = "attachment"
		}
		out = append(out, row)
	}
	return out
}

// Helper: normalisasi arrays URL ops
func (r *UpdateClassAttendanceSessionRequest) NormalizeURLOps() {
	for i := range r.URLsAdd {
		r.URLsAdd[i].Normalize()
	}
	for i := range r.URLsPatch {
		r.URLsPatch[i].Normalize()
	}
}
