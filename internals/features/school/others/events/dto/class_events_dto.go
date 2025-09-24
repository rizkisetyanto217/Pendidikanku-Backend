// file: internals/features/school/sessions/events/dto/class_events_dto.go
package dto

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/others/events/model"
)

/* =========================================================
   Helpers
   ========================================================= */

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

func toDeliveryModePtr(s string) (*m.ClassDeliveryMode, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	v := m.ClassDeliveryMode(strings.ToLower(strings.TrimSpace(s)))
	// Sesuaikan kalau enum DB-mu berbeda
	switch v {
	case "onsite", "online", "hybrid":
		return &v, nil
	default:
		// biarkan DB yang menolak jika enum lain; atau kembalikan error di sini
		return nil, fmt.Errorf("invalid delivery_mode: %s", s)
	}
}

func toEnrollPolicyPtr(s string) (*m.ClassEnrollmentPolicy, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	v := m.ClassEnrollmentPolicy(strings.ToLower(strings.TrimSpace(s)))
	switch v {
	case m.EnrollOpen, m.EnrollInvite, m.EnrollClosed:
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
	ClassEventsThemeID    *uuid.UUID `json:"class_events_theme_id"    validate:"omitempty,uuid"`
	ClassEventsScheduleID *uuid.UUID `json:"class_events_schedule_id" validate:"omitempty,uuid"`

	ClassEventsSectionID      *uuid.UUID `json:"class_events_section_id"       validate:"omitempty,uuid"`
	ClassEventsClassID        *uuid.UUID `json:"class_events_class_id"         validate:"omitempty,uuid"`
	ClassEventsClassSubjectID *uuid.UUID `json:"class_events_class_subject_id" validate:"omitempty,uuid"`

	// inti
	ClassEventsTitle string  `json:"class_events_title" validate:"required,max=160"`
	ClassEventsDesc  *string `json:"class_events_desc"  validate:"omitempty,max=100000"`

	// waktu
	ClassEventsDate    string  `json:"class_events_date"     validate:"required,datetime=2006-01-02"`
	ClassEventsEndDate *string `json:"class_events_end_date" validate:"omitempty,datetime=2006-01-02"`

	ClassEventsStartTime *string `json:"class_events_start_time" validate:"omitempty"` // "HH:mm" / "HH:mm:ss"
	ClassEventsEndTime   *string `json:"class_events_end_time"   validate:"omitempty"`

	// lokasi/mode
	ClassEventsDeliveryMode *string    `json:"class_events_delivery_mode" validate:"omitempty,oneof=onsite online hybrid"`
	ClassEventsRoomID       *uuid.UUID `json:"class_events_room_id"        validate:"omitempty,uuid"`

	// pengajar
	ClassEventsTeacherID   *uuid.UUID `json:"class_events_teacher_id"   validate:"omitempty,uuid"`
	ClassEventsTeacherName *string    `json:"class_events_teacher_name" validate:"omitempty,max=5000"`
	ClassEventsTeacherDesc *string    `json:"class_events_teacher_desc" validate:"omitempty,max=100000"`

	// RSVP
	ClassEventsCapacity         *int    `json:"class_events_capacity"         validate:"omitempty,gte=0"`
	ClassEventsEnrollmentPolicy *string `json:"class_events_enrollment_policy" validate:"omitempty,oneof=open invite closed"`

	// flags
	ClassEventsIsActive *bool `json:"class_events_is_active" validate:"omitempty"`
}

var (
	ErrInvalidDate      = errors.New("invalid class_events_date (use YYYY-MM-DD)")
	ErrInvalidEndDate   = errors.New("invalid class_events_end_date (use YYYY-MM-DD)")
	ErrEndBeforeStart   = errors.New("class_events_end_date must be >= class_events_date")
	ErrInvalidStartTime = errors.New("invalid class_events_start_time (use HH:mm or HH:mm:ss)")
	ErrInvalidEndTime   = errors.New("invalid class_events_end_time (use HH:mm or HH:mm:ss)")
	ErrTimeOrder        = errors.New("class_events_end_time must be > class_events_start_time")
	ErrEmptyTitle       = errors.New("class_events_title cannot be empty")
)

func (r CreateClassEventRequest) ToModel(masjidID uuid.UUID) (m.ClassEventModel, error) {
	if strings.TrimSpace(r.ClassEventsTitle) == "" {
		return m.ClassEventModel{}, ErrEmptyTitle
	}

	date, ok := parseDateYYYYMMDD(r.ClassEventsDate)
	if !ok {
		return m.ClassEventModel{}, ErrInvalidDate
	}

	var endDate *time.Time
	if r.ClassEventsEndDate != nil && strings.TrimSpace(*r.ClassEventsEndDate) != "" {
		ed, ok := parseDateYYYYMMDD(*r.ClassEventsEndDate)
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
	if r.ClassEventsStartTime != nil && strings.TrimSpace(*r.ClassEventsStartTime) != "" {
		st, err := parseTODString(*r.ClassEventsStartTime)
		if err != nil {
			return m.ClassEventModel{}, ErrInvalidStartTime
		}
		stPtr = &st
	}
	if r.ClassEventsEndTime != nil && strings.TrimSpace(*r.ClassEventsEndTime) != "" {
		et, err := parseTODString(*r.ClassEventsEndTime)
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
	if r.ClassEventsDeliveryMode != nil {
		mv, err := toDeliveryModePtr(*r.ClassEventsDeliveryMode)
		if err != nil {
			return m.ClassEventModel{}, err
		}
		mode = mv
	}
	var policy *m.ClassEnrollmentPolicy
	if r.ClassEventsEnrollmentPolicy != nil {
		pv, err := toEnrollPolicyPtr(*r.ClassEventsEnrollmentPolicy)
		if err != nil {
			return m.ClassEventModel{}, err
		}
		policy = pv
	}

	isActive := true
	if r.ClassEventsIsActive != nil {
		isActive = *r.ClassEventsIsActive
	}

	return m.ClassEventModel{
		ClassEventsMasjidID:         masjidID,
		ClassEventsThemeID:          r.ClassEventsThemeID,
		ClassEventsScheduleID:       r.ClassEventsScheduleID,
		ClassEventsSectionID:        r.ClassEventsSectionID,
		ClassEventsClassID:          r.ClassEventsClassID,
		ClassEventsClassSubjectID:   r.ClassEventsClassSubjectID,
		ClassEventsTitle:            strings.TrimSpace(r.ClassEventsTitle),
		ClassEventsDesc:             trimPtr(r.ClassEventsDesc),
		ClassEventsDate:             date,
		ClassEventsEndDate:          endDate,
		ClassEventsStartTime:        stPtr,
		ClassEventsEndTime:          etPtr,
		ClassEventsDeliveryMode:     mode,
		ClassEventsRoomID:           r.ClassEventsRoomID,
		ClassEventsTeacherID:        r.ClassEventsTeacherID,
		ClassEventsTeacherName:      trimPtr(r.ClassEventsTeacherName),
		ClassEventsTeacherDesc:      trimPtr(r.ClassEventsTeacherDesc),
		ClassEventsCapacity:         r.ClassEventsCapacity,
		ClassEventsEnrollmentPolicy: policy,
		ClassEventsIsActive:         isActive,
	}, nil
}

// ---------- PATCH ----------
type PatchClassEventRequest struct {
	// refs
	ClassEventsThemeID    PatchNullable[uuid.UUID] `json:"class_events_theme_id"`
	ClassEventsScheduleID PatchNullable[uuid.UUID] `json:"class_events_schedule_id"`

	ClassEventsSectionID      PatchNullable[uuid.UUID] `json:"class_events_section_id"`
	ClassEventsClassID        PatchNullable[uuid.UUID] `json:"class_events_class_id"`
	ClassEventsClassSubjectID PatchNullable[uuid.UUID] `json:"class_events_class_subject_id"`

	// inti
	ClassEventsTitle Patch[string]         `json:"class_events_title"`
	ClassEventsDesc  PatchNullable[string] `json:"class_events_desc"`

	// tanggal
	ClassEventsDate    Patch[string]         `json:"class_events_date"`     // YYYY-MM-DD
	ClassEventsEndDate PatchNullable[string] `json:"class_events_end_date"` // null allowed

	// waktu (TOD)
	ClassEventsStartTime PatchNullable[string] `json:"class_events_start_time"` // null → all-day
	ClassEventsEndTime   PatchNullable[string] `json:"class_events_end_time"`

	// mode/lokasi
	ClassEventsDeliveryMode PatchNullable[string]    `json:"class_events_delivery_mode"`
	ClassEventsRoomID       PatchNullable[uuid.UUID] `json:"class_events_room_id"`

	// pengajar
	ClassEventsTeacherID   PatchNullable[uuid.UUID] `json:"class_events_teacher_id"`
	ClassEventsTeacherName PatchNullable[string]    `json:"class_events_teacher_name"`
	ClassEventsTeacherDesc PatchNullable[string]    `json:"class_events_teacher_desc"`

	// RSVP
	ClassEventsCapacity         PatchNullable[int]    `json:"class_events_capacity"`
	ClassEventsEnrollmentPolicy PatchNullable[string] `json:"class_events_enrollment_policy"`

	// flag
	ClassEventsIsActive Patch[bool] `json:"class_events_is_active"`
}

func (p *PatchClassEventRequest) Apply(ev *m.ClassEventModel) error {
	// title
	if p.ClassEventsTitle.IsSet() {
		t := strings.TrimSpace(p.ClassEventsTitle.Value)
		if t == "" {
			return ErrEmptyTitle
		}
		ev.ClassEventsTitle = t
	}

	// desc
	if p.ClassEventsDesc.IsSet() {
		if p.ClassEventsDesc.Valid {
			ev.ClassEventsDesc = trimPtr(&p.ClassEventsDesc.Value)
		} else {
			ev.ClassEventsDesc = nil
		}
	}

	// refs (nullable)
	if p.ClassEventsThemeID.IsSet() {
		if p.ClassEventsThemeID.Valid {
			ev.ClassEventsThemeID = &p.ClassEventsThemeID.Value
		} else {
			ev.ClassEventsThemeID = nil
		}
	}
	if p.ClassEventsScheduleID.IsSet() {
		if p.ClassEventsScheduleID.Valid {
			ev.ClassEventsScheduleID = &p.ClassEventsScheduleID.Value
		} else {
			ev.ClassEventsScheduleID = nil
		}
	}
	if p.ClassEventsSectionID.IsSet() {
		if p.ClassEventsSectionID.Valid {
			ev.ClassEventsSectionID = &p.ClassEventsSectionID.Value
		} else {
			ev.ClassEventsSectionID = nil
		}
	}
	if p.ClassEventsClassID.IsSet() {
		if p.ClassEventsClassID.Valid {
			ev.ClassEventsClassID = &p.ClassEventsClassID.Value
		} else {
			ev.ClassEventsClassID = nil
		}
	}
	if p.ClassEventsClassSubjectID.IsSet() {
		if p.ClassEventsClassSubjectID.Valid {
			ev.ClassEventsClassSubjectID = &p.ClassEventsClassSubjectID.Value
		} else {
			ev.ClassEventsClassSubjectID = nil
		}
	}

	// tanggal
	newDate := ev.ClassEventsDate
	newEnd := ev.ClassEventsEndDate

	if p.ClassEventsDate.IsSet() {
		d, ok := parseDateYYYYMMDD(p.ClassEventsDate.Value)
		if !ok {
			return ErrInvalidDate
		}
		newDate = d
	}
	if p.ClassEventsEndDate.IsSet() {
		if p.ClassEventsEndDate.Valid {
			ed, ok := parseDateYYYYMMDD(p.ClassEventsEndDate.Value)
			if !ok {
				return ErrInvalidEndDate
			}
			newEnd = &ed
		} else {
			newEnd = nil
		}
	}
	if p.ClassEventsDate.IsSet() || p.ClassEventsEndDate.IsSet() {
		if newEnd != nil && newEnd.Before(newDate) {
			return ErrEndBeforeStart
		}
		ev.ClassEventsDate = newDate
		ev.ClassEventsEndDate = newEnd
	}

	// waktu
	newStart := ev.ClassEventsStartTime
	newEndT := ev.ClassEventsEndTime

	if p.ClassEventsStartTime.IsSet() {
		if p.ClassEventsStartTime.Valid {
			st, err := parseTODString(p.ClassEventsStartTime.Value)
			if err != nil {
				return ErrInvalidStartTime
			}
			newStart = &st
		} else {
			newStart = nil
		}
	}
	if p.ClassEventsEndTime.IsSet() {
		if p.ClassEventsEndTime.Valid {
			et, err := parseTODString(p.ClassEventsEndTime.Value)
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
	if p.ClassEventsStartTime.IsSet() || p.ClassEventsEndTime.IsSet() {
		ev.ClassEventsStartTime = newStart
		ev.ClassEventsEndTime = newEndT
	}

	// mode & room
	if p.ClassEventsDeliveryMode.IsSet() {
		if p.ClassEventsDeliveryMode.Valid {
			mv, err := toDeliveryModePtr(p.ClassEventsDeliveryMode.Value)
			if err != nil {
				return err
			}
			ev.ClassEventsDeliveryMode = mv
		} else {
			ev.ClassEventsDeliveryMode = nil
		}
	}
	if p.ClassEventsRoomID.IsSet() {
		if p.ClassEventsRoomID.Valid {
			ev.ClassEventsRoomID = &p.ClassEventsRoomID.Value
		} else {
			ev.ClassEventsRoomID = nil
		}
	}

	// teacher
	if p.ClassEventsTeacherID.IsSet() {
		if p.ClassEventsTeacherID.Valid {
			ev.ClassEventsTeacherID = &p.ClassEventsTeacherID.Value
		} else {
			ev.ClassEventsTeacherID = nil
		}
	}
	if p.ClassEventsTeacherName.IsSet() {
		if p.ClassEventsTeacherName.Valid {
			ev.ClassEventsTeacherName = trimPtr(&p.ClassEventsTeacherName.Value)
		} else {
			ev.ClassEventsTeacherName = nil
		}
	}
	if p.ClassEventsTeacherDesc.IsSet() {
		if p.ClassEventsTeacherDesc.Valid {
			ev.ClassEventsTeacherDesc = trimPtr(&p.ClassEventsTeacherDesc.Value)
		} else {
			ev.ClassEventsTeacherDesc = nil
		}
	}

	// RSVP
	if p.ClassEventsCapacity.IsSet() {
		if p.ClassEventsCapacity.Valid {
			if p.ClassEventsCapacity.Value < 0 {
				return errors.New("class_events_capacity must be >= 0")
			}
			v := p.ClassEventsCapacity.Value
			ev.ClassEventsCapacity = &v
		} else {
			ev.ClassEventsCapacity = nil
		}
	}
	if p.ClassEventsEnrollmentPolicy.IsSet() {
		if p.ClassEventsEnrollmentPolicy.Valid {
			pv, err := toEnrollPolicyPtr(p.ClassEventsEnrollmentPolicy.Value)
			if err != nil {
				return err
			}
			ev.ClassEventsEnrollmentPolicy = pv
		} else {
			ev.ClassEventsEnrollmentPolicy = nil
		}
	}

	// flags
	if p.ClassEventsIsActive.IsSet() {
		ev.ClassEventsIsActive = p.ClassEventsIsActive.Value
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
	ThemeID        *uuid.UUID `query:"theme_id"       validate:"omitempty,uuid"`
	ScheduleID     *uuid.UUID `query:"schedule_id"    validate:"omitempty,uuid"`
	SectionID      *uuid.UUID `query:"section_id"     validate:"omitempty,uuid"`
	ClassID        *uuid.UUID `query:"class_id"       validate:"omitempty,uuid"`
	ClassSubjectID *uuid.UUID `query:"class_subject_id" validate:"omitempty,uuid"`
	RoomID         *uuid.UUID `query:"room_id"        validate:"omitempty,uuid"`
	TeacherID      *uuid.UUID `query:"teacher_id"     validate:"omitempty,uuid"`

	DeliveryMode     *string `query:"delivery_mode"   validate:"omitempty,oneof=onsite online hybrid"`
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
	ClassEventsID       uuid.UUID `json:"class_events_id"`
	ClassEventsMasjidID uuid.UUID `json:"class_events_masjid_id"`

	ClassEventsThemeID    *uuid.UUID `json:"class_events_theme_id,omitempty"`
	ClassEventsScheduleID *uuid.UUID `json:"class_events_schedule_id,omitempty"`

	ClassEventsSectionID      *uuid.UUID `json:"class_events_section_id,omitempty"`
	ClassEventsClassID        *uuid.UUID `json:"class_events_class_id,omitempty"`
	ClassEventsClassSubjectID *uuid.UUID `json:"class_events_class_subject_id,omitempty"`

	ClassEventsTitle string  `json:"class_events_title"`
	ClassEventsDesc  *string `json:"class_events_desc,omitempty"`

	ClassEventsDate    time.Time  `json:"class_events_date"`
	ClassEventsEndDate *time.Time `json:"class_events_end_date,omitempty"`

	ClassEventsStartTime *string `json:"class_events_start_time,omitempty"` // "HH:mm:ss"
	ClassEventsEndTime   *string `json:"class_events_end_time,omitempty"`

	ClassEventsDeliveryMode *string    `json:"class_events_delivery_mode,omitempty"`
	ClassEventsRoomID       *uuid.UUID `json:"class_events_room_id,omitempty"`

	ClassEventsTeacherID   *uuid.UUID `json:"class_events_teacher_id,omitempty"`
	ClassEventsTeacherName *string    `json:"class_events_teacher_name,omitempty"`
	ClassEventsTeacherDesc *string    `json:"class_events_teacher_desc,omitempty"`

	ClassEventsCapacity         *int    `json:"class_events_capacity,omitempty"`
	ClassEventsEnrollmentPolicy *string `json:"class_events_enrollment_policy,omitempty"`

	ClassEventsIsActive bool `json:"class_events_is_active"`

	ClassEventsCreatedAt time.Time  `json:"class_events_created_at"`
	ClassEventsUpdatedAt time.Time  `json:"class_events_updated_at"`
	ClassEventsDeletedAt *time.Time `json:"class_events_deleted_at,omitempty"`
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
	if ev.ClassEventsStartTime != nil {
		s := formatTOD(*ev.ClassEventsStartTime)
		stStr = &s
	}
	if ev.ClassEventsEndTime != nil {
		s := formatTOD(*ev.ClassEventsEndTime)
		etStr = &s
	}

	var delAt *time.Time
	if ev.ClassEventsDeletedAt.Valid {
		t := ev.ClassEventsDeletedAt.Time
		delAt = &t
	}

	var modeStr *string
	if ev.ClassEventsDeliveryMode != nil {
		s := string(*ev.ClassEventsDeliveryMode)
		modeStr = &s
	}
	var polStr *string
	if ev.ClassEventsEnrollmentPolicy != nil {
		s := string(*ev.ClassEventsEnrollmentPolicy)
		polStr = &s
	}

	return ClassEventResponse{
		ClassEventsID:               ev.ClassEventsID,
		ClassEventsMasjidID:         ev.ClassEventsMasjidID,
		ClassEventsThemeID:          ev.ClassEventsThemeID,
		ClassEventsScheduleID:       ev.ClassEventsScheduleID,
		ClassEventsSectionID:        ev.ClassEventsSectionID,
		ClassEventsClassID:          ev.ClassEventsClassID,
		ClassEventsClassSubjectID:   ev.ClassEventsClassSubjectID,
		ClassEventsTitle:            ev.ClassEventsTitle,
		ClassEventsDesc:             ev.ClassEventsDesc,
		ClassEventsDate:             ev.ClassEventsDate,
		ClassEventsEndDate:          ev.ClassEventsEndDate,
		ClassEventsStartTime:        stStr,
		ClassEventsEndTime:          etStr,
		ClassEventsDeliveryMode:     modeStr,
		ClassEventsRoomID:           ev.ClassEventsRoomID,
		ClassEventsTeacherID:        ev.ClassEventsTeacherID,
		ClassEventsTeacherName:      ev.ClassEventsTeacherName,
		ClassEventsTeacherDesc:      ev.ClassEventsTeacherDesc,
		ClassEventsCapacity:         ev.ClassEventsCapacity,
		ClassEventsEnrollmentPolicy: polStr,
		ClassEventsIsActive:         ev.ClassEventsIsActive,
		ClassEventsCreatedAt:        ev.ClassEventsCreatedAt,
		ClassEventsUpdatedAt:        ev.ClassEventsUpdatedAt,
		ClassEventsDeletedAt:        delAt,
	}
}

func FromModelsClassEvent(list []m.ClassEventModel) []ClassEventResponse {
	out := make([]ClassEventResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelClassEvent(list[i]))
	}
	return out
}
