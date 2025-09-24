// file: internals/features/school/sessions/events/dto/class_events_dto.go
package dto

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/classes/class_events/model"
)

/* =========================================================
   Helpers
   ========================================================= */
// Tri-state (value only)

func (p Patch[T]) IsSet() bool { return p.Set }

func (p PatchNullable[T]) IsSet() bool { return p.Set }

// parse "YYYY-MM-DD" → time.Time (midnight UTC)
func parseDateYYYYMMDD(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
}

// parse "HH:mm[:ss]" → time.Time (TOD at 2000-01-01 UTC)
func parseTODString(s string) (time.Time, error) {
	ss := strings.TrimSpace(s)
	if ss == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse("15:04:05", ss); err == nil {
		return time.Date(2000, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
	}
	if t, err := time.Parse("15:04", ss); err == nil {
		return time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("invalid time format: %q (use HH:mm or HH:mm:ss)", s)
}

func formatTOD(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04:05")
}

func ptrTimeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// Terima "online|offline|hybrid" (sesuai enum DB).
// Kompatibilitas: "onsite" akan dipetakan ke "offline".
func toDeliveryModePtr(s string) (*m.ClassDeliveryMode, error) {
	raw := strings.ToLower(strings.TrimSpace(s))
	if raw == "" {
		return nil, nil
	}
	if raw == "onsite" {
		raw = "offline"
	}
	v := m.ClassDeliveryMode(raw)
	switch v {
	case m.ClassDeliveryModeOnline, m.ClassDeliveryModeOffline, m.ClassDeliveryModeHybrid:
		return &v, nil
	default:
		return nil, fmt.Errorf("invalid delivery_mode: %s (use online|offline|hybrid)", s)
	}
}

func toEnrollPolicyPtr(s string) (*m.ClassEventEnrollmentPolicy, error) {
	raw := strings.ToLower(strings.TrimSpace(s))
	if raw == "" {
		return nil, nil
	}
	switch raw {
	case string(m.ClassEventPolicyOpen):
		v := m.ClassEventPolicyOpen
		return &v, nil
	case string(m.ClassEventPolicyInvite):
		v := m.ClassEventPolicyInvite
		return &v, nil
	case string(m.ClassEventPolicyClosed):
		v := m.ClassEventPolicyClosed
		return &v, nil
	default:
		return nil, fmt.Errorf("invalid enrollment_policy: %s (use open|invite|closed)", s)
	}
}

/* =========================================================
   Patch types (tri-state)
   ========================================================= */

type Patch[T any] struct {
	Set   bool
	Value T
}

type PatchNullable[T any] struct {
	Set   bool
	Valid bool
	Value T
}

/* =========================================================
   1) REQUESTS
   ========================================================= */

// ---------- CREATE ----------
type CreateClassEventRequest struct {
	// refs (opsional)
	ClassEventThemeID    *uuid.UUID `json:"class_event_theme_id"    validate:"omitempty,uuid"`
	ClassEventScheduleID *uuid.UUID `json:"class_event_schedule_id" validate:"omitempty,uuid"`

	ClassEventSectionID      *uuid.UUID `json:"class_event_section_id"       validate:"omitempty,uuid"`
	ClassEventClassID        *uuid.UUID `json:"class_event_class_id"         validate:"omitempty,uuid"`
	ClassEventClassSubjectID *uuid.UUID `json:"class_event_class_subject_id" validate:"omitempty,uuid"`

	// inti
	ClassEventTitle string  `json:"class_event_title" validate:"required,max=160"`
	ClassEventDesc  *string `json:"class_event_desc"  validate:"omitempty,max=100000"`

	// waktu
	ClassEventDate    string  `json:"class_event_date"     validate:"required,datetime=2006-01-02"`
	ClassEventEndDate *string `json:"class_event_end_date" validate:"omitempty,datetime=2006-01-02"`

	ClassEventStartTime *string `json:"class_event_start_time" validate:"omitempty"` // "HH:mm" / "HH:mm:ss"
	ClassEventEndTime   *string `json:"class_event_end_time"   validate:"omitempty"`

	// lokasi/mode
	ClassEventDeliveryMode *string    `json:"class_event_delivery_mode" validate:"omitempty,oneof=online offline hybrid"`
	ClassEventRoomID       *uuid.UUID `json:"class_event_room_id"        validate:"omitempty,uuid"`

	// pengajar
	ClassEventTeacherID   *uuid.UUID `json:"class_event_teacher_id"   validate:"omitempty,uuid"`
	ClassEventTeacherName *string    `json:"class_event_teacher_name" validate:"omitempty,max=5000"`
	ClassEventTeacherDesc *string    `json:"class_event_teacher_desc" validate:"omitempty,max=100000"`

	// RSVP
	ClassEventCapacity         *int    `json:"class_event_capacity"          validate:"omitempty,gte=0"`
	ClassEventEnrollmentPolicy *string `json:"class_event_enrollment_policy" validate:"omitempty,oneof=open invite closed"`

	// flags
	ClassEventIsActive *bool `json:"class_event_is_active" validate:"omitempty"`
}

var (
	ErrInvalidDate      = errors.New("invalid class_event_date (use YYYY-MM-DD)")
	ErrInvalidEndDate   = errors.New("invalid class_event_end_date (use YYYY-MM-DD)")
	ErrEndBeforeStart   = errors.New("class_event_end_date must be >= class_event_date")
	ErrInvalidStartTime = errors.New("invalid class_event_start_time (use HH:mm or HH:mm:ss)")
	ErrInvalidEndTime   = errors.New("invalid class_event_end_time (use HH:mm or HH:mm:ss)")
	ErrTimeOrder        = errors.New("class_event_end_time must be > class_event_start_time")
	ErrEmptyTitle       = errors.New("class_event_title cannot be empty")
)

func (r CreateClassEventRequest) ToModel(masjidID uuid.UUID) (m.ClassEventModel, error) {
	if strings.TrimSpace(r.ClassEventTitle) == "" {
		return m.ClassEventModel{}, ErrEmptyTitle
	}

	date, ok := parseDateYYYYMMDD(r.ClassEventDate)
	if !ok {
		return m.ClassEventModel{}, ErrInvalidDate
	}

	var endDate *time.Time
	if r.ClassEventEndDate != nil && strings.TrimSpace(*r.ClassEventEndDate) != "" {
		ed, ok := parseDateYYYYMMDD(*r.ClassEventEndDate)
		if !ok {
			return m.ClassEventModel{}, ErrInvalidEndDate
		}
		if ed.Before(date) {
			return m.ClassEventModel{}, ErrEndBeforeStart
		}
		endDate = &ed
	}

	// times
	var stPtr, etPtr *time.Time
	if r.ClassEventStartTime != nil && strings.TrimSpace(*r.ClassEventStartTime) != "" {
		st, err := parseTODString(*r.ClassEventStartTime)
		if err != nil {
			return m.ClassEventModel{}, ErrInvalidStartTime
		}
		stPtr = &st
	}
	if r.ClassEventEndTime != nil && strings.TrimSpace(*r.ClassEventEndTime) != "" {
		et, err := parseTODString(*r.ClassEventEndTime)
		if err != nil {
			return m.ClassEventModel{}, ErrInvalidEndTime
		}
		etPtr = &et
	}
	if stPtr != nil && etPtr != nil && !etPtr.After(*stPtr) {
		return m.ClassEventModel{}, ErrTimeOrder
	}

	// enums
	var mode *m.ClassDeliveryMode
	if r.ClassEventDeliveryMode != nil {
		mv, err := toDeliveryModePtr(*r.ClassEventDeliveryMode)
		if err != nil {
			return m.ClassEventModel{}, err
		}
		mode = mv
	}
	var policy *m.ClassEventEnrollmentPolicy
	if r.ClassEventEnrollmentPolicy != nil {
		pv, err := toEnrollPolicyPtr(*r.ClassEventEnrollmentPolicy)
		if err != nil {
			return m.ClassEventModel{}, err
		}
		policy = pv
	}

	isActive := true
	if r.ClassEventIsActive != nil {
		isActive = *r.ClassEventIsActive
	}

	return m.ClassEventModel{
		ClassEventMasjidID:         masjidID,
		ClassEventThemeID:          r.ClassEventThemeID,
		ClassEventScheduleID:       r.ClassEventScheduleID,
		ClassEventSectionID:        r.ClassEventSectionID,
		ClassEventClassID:          r.ClassEventClassID,
		ClassEventClassSubjectID:   r.ClassEventClassSubjectID,
		ClassEventTitle:            strings.TrimSpace(r.ClassEventTitle),
		ClassEventDesc:             trimPtr(r.ClassEventDesc),
		ClassEventDate:             date,
		ClassEventEndDate:          endDate,
		ClassEventStartTime:        stPtr,
		ClassEventEndTime:          etPtr,
		ClassEventDeliveryMode:     mode,
		ClassEventRoomID:           r.ClassEventRoomID,
		ClassEventTeacherID:        r.ClassEventTeacherID,
		ClassEventTeacherName:      trimPtr(r.ClassEventTeacherName),
		ClassEventTeacherDesc:      trimPtr(r.ClassEventTeacherDesc),
		ClassEventCapacity:         r.ClassEventCapacity,
		ClassEventEnrollmentPolicy: policy,
		ClassEventIsActive:         isActive,
	}, nil
}

// ---------- PATCH ----------
type PatchClassEventRequest struct {
	// refs
	ClassEventThemeID    PatchNullable[uuid.UUID] `json:"class_event_theme_id"`
	ClassEventScheduleID PatchNullable[uuid.UUID] `json:"class_event_schedule_id"`

	ClassEventSectionID      PatchNullable[uuid.UUID] `json:"class_event_section_id"`
	ClassEventClassID        PatchNullable[uuid.UUID] `json:"class_event_class_id"`
	ClassEventClassSubjectID PatchNullable[uuid.UUID] `json:"class_event_class_subject_id"`

	// inti
	ClassEventTitle Patch[string]         `json:"class_event_title"`
	ClassEventDesc  PatchNullable[string] `json:"class_event_desc"`

	// tanggal
	ClassEventDate    Patch[string]         `json:"class_event_date"`     // YYYY-MM-DD
	ClassEventEndDate PatchNullable[string] `json:"class_event_end_date"` // null allowed

	// waktu (TOD)
	ClassEventStartTime PatchNullable[string] `json:"class_event_start_time"` // null → all-day
	ClassEventEndTime   PatchNullable[string] `json:"class_event_end_time"`

	// mode/lokasi
	ClassEventDeliveryMode PatchNullable[string]    `json:"class_event_delivery_mode"`
	ClassEventRoomID       PatchNullable[uuid.UUID] `json:"class_event_room_id"`

	// pengajar
	ClassEventTeacherID   PatchNullable[uuid.UUID] `json:"class_event_teacher_id"`
	ClassEventTeacherName PatchNullable[string]    `json:"class_event_teacher_name"`
	ClassEventTeacherDesc PatchNullable[string]    `json:"class_event_teacher_desc"`

	// RSVP
	ClassEventCapacity         PatchNullable[int]    `json:"class_event_capacity"`
	ClassEventEnrollmentPolicy PatchNullable[string] `json:"class_event_enrollment_policy"`

	// flag
	ClassEventIsActive Patch[bool] `json:"class_event_is_active"`
}

func (p *PatchClassEventRequest) Apply(ev *m.ClassEventModel) error {
	// title
	if p.ClassEventTitle.IsSet() {
		t := strings.TrimSpace(p.ClassEventTitle.Value)
		if t == "" {
			return ErrEmptyTitle
		}
		ev.ClassEventTitle = t
	}

	// desc
	if p.ClassEventDesc.IsSet() {
		if p.ClassEventDesc.Valid {
			ev.ClassEventDesc = trimPtr(&p.ClassEventDesc.Value)
		} else {
			ev.ClassEventDesc = nil
		}
	}

	// refs (nullable)
	if p.ClassEventThemeID.IsSet() {
		if p.ClassEventThemeID.Valid {
			ev.ClassEventThemeID = &p.ClassEventThemeID.Value
		} else {
			ev.ClassEventThemeID = nil
		}
	}
	if p.ClassEventScheduleID.IsSet() {
		if p.ClassEventScheduleID.Valid {
			ev.ClassEventScheduleID = &p.ClassEventScheduleID.Value
		} else {
			ev.ClassEventScheduleID = nil
		}
	}
	if p.ClassEventSectionID.IsSet() {
		if p.ClassEventSectionID.Valid {
			ev.ClassEventSectionID = &p.ClassEventSectionID.Value
		} else {
			ev.ClassEventSectionID = nil
		}
	}
	if p.ClassEventClassID.IsSet() {
		if p.ClassEventClassID.Valid {
			ev.ClassEventClassID = &p.ClassEventClassID.Value
		} else {
			ev.ClassEventClassID = nil
		}
	}
	if p.ClassEventClassSubjectID.IsSet() {
		if p.ClassEventClassSubjectID.Valid {
			ev.ClassEventClassSubjectID = &p.ClassEventClassSubjectID.Value
		} else {
			ev.ClassEventClassSubjectID = nil
		}
	}

	// tanggal
	newDate := ev.ClassEventDate
	newEnd := ev.ClassEventEndDate

	if p.ClassEventDate.IsSet() {
		d, ok := parseDateYYYYMMDD(p.ClassEventDate.Value)
		if !ok {
			return ErrInvalidDate
		}
		newDate = d
	}
	if p.ClassEventEndDate.IsSet() {
		if p.ClassEventEndDate.Valid {
			ed, ok := parseDateYYYYMMDD(p.ClassEventEndDate.Value)
			if !ok {
				return ErrInvalidEndDate
			}
			newEnd = &ed
		} else {
			newEnd = nil
		}
	}
	if p.ClassEventDate.IsSet() || p.ClassEventEndDate.IsSet() {
		if newEnd != nil && newEnd.Before(newDate) {
			return ErrEndBeforeStart
		}
		ev.ClassEventDate = newDate
		ev.ClassEventEndDate = newEnd
	}

	// waktu
	newStart := ev.ClassEventStartTime
	newEndT := ev.ClassEventEndTime

	if p.ClassEventStartTime.IsSet() {
		if p.ClassEventStartTime.Valid {
			st, err := parseTODString(p.ClassEventStartTime.Value)
			if err != nil {
				return ErrInvalidStartTime
			}
			newStart = &st
		} else {
			newStart = nil
		}
	}
	if p.ClassEventEndTime.IsSet() {
		if p.ClassEventEndTime.Valid {
			et, err := parseTODString(p.ClassEventEndTime.Value)
			if err != nil {
				return ErrInvalidEndTime
			}
			newEndT = &et
		} else {
			newEndT = nil
		}
	}
	if newStart != nil && newEndT != nil && !newEndT.After(*newStart) {
		return ErrTimeOrder
	}
	if p.ClassEventStartTime.IsSet() || p.ClassEventEndTime.IsSet() {
		ev.ClassEventStartTime = newStart
		ev.ClassEventEndTime = newEndT
	}

	// mode & room
	if p.ClassEventDeliveryMode.IsSet() {
		if p.ClassEventDeliveryMode.Valid {
			mv, err := toDeliveryModePtr(p.ClassEventDeliveryMode.Value)
			if err != nil {
				return err
			}
			ev.ClassEventDeliveryMode = mv
		} else {
			ev.ClassEventDeliveryMode = nil
		}
	}
	if p.ClassEventRoomID.IsSet() {
		if p.ClassEventRoomID.Valid {
			ev.ClassEventRoomID = &p.ClassEventRoomID.Value
		} else {
			ev.ClassEventRoomID = nil
		}
	}

	// teacher
	if p.ClassEventTeacherID.IsSet() {
		if p.ClassEventTeacherID.Valid {
			ev.ClassEventTeacherID = &p.ClassEventTeacherID.Value
		} else {
			ev.ClassEventTeacherID = nil
		}
	}
	if p.ClassEventTeacherName.IsSet() {
		if p.ClassEventTeacherName.Valid {
			ev.ClassEventTeacherName = trimPtr(&p.ClassEventTeacherName.Value)
		} else {
			ev.ClassEventTeacherName = nil
		}
	}
	if p.ClassEventTeacherDesc.IsSet() {
		if p.ClassEventTeacherDesc.Valid {
			ev.ClassEventTeacherDesc = trimPtr(&p.ClassEventTeacherDesc.Value)
		} else {
			ev.ClassEventTeacherDesc = nil
		}
	}

	// RSVP
	if p.ClassEventCapacity.IsSet() {
		if p.ClassEventCapacity.Valid {
			if p.ClassEventCapacity.Value < 0 {
				return errors.New("class_event_capacity must be >= 0")
			}
			v := p.ClassEventCapacity.Value
			ev.ClassEventCapacity = &v
		} else {
			ev.ClassEventCapacity = nil
		}
	}
	if p.ClassEventEnrollmentPolicy.IsSet() {
		if p.ClassEventEnrollmentPolicy.Valid {
			pv, err := toEnrollPolicyPtr(p.ClassEventEnrollmentPolicy.Value)
			if err != nil {
				return err
			}
			ev.ClassEventEnrollmentPolicy = pv
		} else {
			ev.ClassEventEnrollmentPolicy = nil
		}
	}

	// flags
	if p.ClassEventIsActive.IsSet() {
		ev.ClassEventIsActive = p.ClassEventIsActive.Value
	}

	return nil
}

/* =========================================================
   2) LIST QUERY
   ========================================================= */

type ListClassEventsQuery struct {
	// rentang tanggal (overlap)
	DateFrom *string `query:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo   *string `query:"date_to"   validate:"omitempty,datetime=2006-01-02"`

	OnlyActive *bool `query:"only_active" validate:"omitempty"`

	// filter refs
	ThemeID        *uuid.UUID `query:"theme_id"         validate:"omitempty,uuid"`
	ScheduleID     *uuid.UUID `query:"schedule_id"      validate:"omitempty,uuid"`
	SectionID      *uuid.UUID `query:"section_id"       validate:"omitempty,uuid"`
	ClassID        *uuid.UUID `query:"class_id"         validate:"omitempty,uuid"`
	ClassSubjectID *uuid.UUID `query:"class_subject_id" validate:"omitempty,uuid"`
	RoomID         *uuid.UUID `query:"room_id"          validate:"omitempty,uuid"`
	TeacherID      *uuid.UUID `query:"teacher_id"       validate:"omitempty,uuid"`

	DeliveryMode     *string `query:"delivery_mode"     validate:"omitempty,oneof=online offline hybrid"`
	EnrollmentPolicy *string `query:"enrollment_policy" validate:"omitempty,oneof=open invite closed"`

	// search ringan
	Q *string `query:"q" validate:"omitempty,max=160"`

	// sort
	//   date_asc|date_desc|start_time_asc|start_time_desc|
	//   created_at_asc|created_at_desc|updated_at_asc|updated_at_desc|
	//   title_asc|title_desc
	Sort *string `query:"sort" validate:"omitempty,oneof=date_asc date_desc start_time_asc start_time_desc created_at_asc created_at_desc updated_at_asc updated_at_desc title_asc title_desc"`

	// paging
	Limit  int `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int `query:"offset" validate:"omitempty,min=0"`
}

func (q *ListClassEventsQuery) Normalize() {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Q != nil {
		v := strings.TrimSpace(*q.Q)
		if v == "" {
			q.Q = nil
		} else {
			q.Q = &v
		}
	}
}

/* =========================================================
   3) RESPONSES
   ========================================================= */

type ClassEventResponse struct {
	ClassEventID       uuid.UUID `json:"class_event_id"`
	ClassEventMasjidID uuid.UUID `json:"class_event_masjid_id"`

	ClassEventThemeID    *uuid.UUID `json:"class_event_theme_id,omitempty"`
	ClassEventScheduleID *uuid.UUID `json:"class_event_schedule_id,omitempty"`

	ClassEventSectionID      *uuid.UUID `json:"class_event_section_id,omitempty"`
	ClassEventClassID        *uuid.UUID `json:"class_event_class_id,omitempty"`
	ClassEventClassSubjectID *uuid.UUID `json:"class_event_class_subject_id,omitempty"`

	ClassEventTitle string  `json:"class_event_title"`
	ClassEventDesc  *string `json:"class_event_desc,omitempty"`

	ClassEventDate    time.Time  `json:"class_event_date"`
	ClassEventEndDate *time.Time `json:"class_event_end_date,omitempty"`

	ClassEventStartTime *string `json:"class_event_start_time,omitempty"` // "HH:mm:ss"
	ClassEventEndTime   *string `json:"class_event_end_time,omitempty"`

	ClassEventDeliveryMode *string    `json:"class_event_delivery_mode,omitempty"`
	ClassEventRoomID       *uuid.UUID `json:"class_event_room_id,omitempty"`

	ClassEventTeacherID   *uuid.UUID `json:"class_event_teacher_id,omitempty"`
	ClassEventTeacherName *string    `json:"class_event_teacher_name,omitempty"`
	ClassEventTeacherDesc *string    `json:"class_event_teacher_desc,omitempty"`

	ClassEventCapacity         *int    `json:"class_event_capacity,omitempty"`
	ClassEventEnrollmentPolicy *string `json:"class_event_enrollment_policy,omitempty"`

	ClassEventIsActive bool `json:"class_event_is_active"`

	ClassEventCreatedAt time.Time  `json:"class_event_created_at"`
	ClassEventUpdatedAt time.Time  `json:"class_event_updated_at"`
	ClassEventDeletedAt *time.Time `json:"class_event_deleted_at,omitempty"`
}

type ClassEventListResponse struct {
	Items      []ClassEventResponse `json:"items"`
	Pagination struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	} `json:"pagination"`
}

/* =========================================================
   4) MAPPERS
   ========================================================= */

func FromModelClassEvent(ev m.ClassEventModel) ClassEventResponse {
	var stStr, etStr *string
	if ev.ClassEventStartTime != nil {
		s := formatTOD(*ev.ClassEventStartTime)
		stStr = &s
	}
	if ev.ClassEventEndTime != nil {
		s := formatTOD(*ev.ClassEventEndTime)
		etStr = &s
	}

	var delAt *time.Time
	if ev.ClassEventDeletedAt.Valid {
		t := ev.ClassEventDeletedAt.Time
		delAt = &t
	}

	var modeStr *string
	if ev.ClassEventDeliveryMode != nil {
		s := string(*ev.ClassEventDeliveryMode)
		modeStr = &s
	}
	var polStr *string
	if ev.ClassEventEnrollmentPolicy != nil {
		s := string(*ev.ClassEventEnrollmentPolicy)
		polStr = &s
	}

	return ClassEventResponse{
		ClassEventID:               ev.ClassEventID,
		ClassEventMasjidID:         ev.ClassEventMasjidID,
		ClassEventThemeID:          ev.ClassEventThemeID,
		ClassEventScheduleID:       ev.ClassEventScheduleID,
		ClassEventSectionID:        ev.ClassEventSectionID,
		ClassEventClassID:          ev.ClassEventClassID,
		ClassEventClassSubjectID:   ev.ClassEventClassSubjectID,
		ClassEventTitle:            ev.ClassEventTitle,
		ClassEventDesc:             ev.ClassEventDesc,
		ClassEventDate:             ev.ClassEventDate,
		ClassEventEndDate:          ev.ClassEventEndDate,
		ClassEventStartTime:        stStr,
		ClassEventEndTime:          etStr,
		ClassEventDeliveryMode:     modeStr,
		ClassEventRoomID:           ev.ClassEventRoomID,
		ClassEventTeacherID:        ev.ClassEventTeacherID,
		ClassEventTeacherName:      ev.ClassEventTeacherName,
		ClassEventTeacherDesc:      ev.ClassEventTeacherDesc,
		ClassEventCapacity:         ev.ClassEventCapacity,
		ClassEventEnrollmentPolicy: polStr,
		ClassEventIsActive:         ev.ClassEventIsActive,
		ClassEventCreatedAt:        ev.ClassEventCreatedAt,
		ClassEventUpdatedAt:        ev.ClassEventUpdatedAt,
		ClassEventDeletedAt:        delAt,
	}
}

func FromModelsClassEvent(list []m.ClassEventModel) []ClassEventResponse {
	out := make([]ClassEventResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelClassEvent(list[i]))
	}
	return out
}
