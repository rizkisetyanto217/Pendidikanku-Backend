// file: internals/features/school/class_schedules/dto/class_schedule_dto.go
package dto

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/sessions/schedules/model"
)

/* =======================================================
   Util & parsing
   ======================================================= */

var (
	layoutDate = "2006-01-02" // DATE
	layoutT1   = "15:04"      // TIME (HH:mm)
	layoutT2   = "15:04:05"   // TIME (HH:mm:ss)
)

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty date")
	}
	t, err := time.Parse(layoutDate, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format (want YYYY-MM-DD): %w", err)
	}
	return t, nil
}

func parseTimeOnly(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}
	if t, err := time.Parse(layoutT1, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(layoutT2, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time format (want HH:mm or HH:mm:ss)")
}

func uuidPtrFromString(s *string) (*uuid.UUID, error) {
	if s == nil {
		return nil, nil
	}
	ss := strings.TrimSpace(*s)
	if ss == "" {
		return nil, nil
	}
	id, err := uuid.Parse(ss)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}
	return &id, nil
}

func uuidFromStringRequired(s string) (uuid.UUID, error) {
	ss := strings.TrimSpace(s)
	if ss == "" {
		return uuid.Nil, errors.New("empty uuid")
	}
	return uuid.Parse(ss)
}

/* =======================================================
   Request DTOs
   ======================================================= */

type CreateClassScheduleRequest struct {
	// Required
	ClassSchedulesMasjidID  string `json:"class_schedules_masjid_id"  validate:"required,uuid4"`
	ClassSchedulesDayOfWeek int    `json:"class_schedules_day_of_week" validate:"required,gte=1,lte=7"`
	ClassSchedulesStartTime string `json:"class_schedules_start_time"  validate:"required"`
	ClassSchedulesEndTime   string `json:"class_schedules_end_time"    validate:"required"`
	ClassSchedulesStartDate string `json:"class_schedules_start_date"  validate:"required"`
	ClassSchedulesEndDate   string `json:"class_schedules_end_date"    validate:"required"`

	// Optional
	ClassSchedulesCSSTID   *string          `json:"class_schedules_csst_id,omitempty"   validate:"omitempty,uuid4"`
	ClassSchedulesEventID  *string          `json:"class_schedules_event_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesStatus   *m.SessionStatus `json:"class_schedules_status,omitempty"    validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	ClassSchedulesIsActive *bool            `json:"class_schedules_is_active,omitempty"`
}

type UpdateClassScheduleRequest struct {
	// Required (put-like)
	ClassSchedulesMasjidID  string `json:"class_schedules_masjid_id"  validate:"required,uuid4"`
	ClassSchedulesDayOfWeek int    `json:"class_schedules_day_of_week" validate:"required,gte=1,lte=7"`
	ClassSchedulesStartTime string `json:"class_schedules_start_time"  validate:"required"`
	ClassSchedulesEndTime   string `json:"class_schedules_end_time"    validate:"required"`
	ClassSchedulesStartDate string `json:"class_schedules_start_date"  validate:"required"`
	ClassSchedulesEndDate   string `json:"class_schedules_end_date"    validate:"required"`
	ClassSchedulesStatus    string `json:"class_schedules_status"      validate:"required,oneof=scheduled ongoing completed canceled"`
	ClassSchedulesIsActive  bool   `json:"class_schedules_is_active"`

	// Optional
	ClassSchedulesCSSTID  *string `json:"class_schedules_csst_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesEventID *string `json:"class_schedules_event_id,omitempty" validate:"omitempty,uuid4"`
}

type PatchClassScheduleRequest struct {
	// Semua optional—akan di-apply selectively
	ClassSchedulesMasjidID  *string          `json:"class_schedules_masjid_id,omitempty"   validate:"omitempty,uuid4"`
	ClassSchedulesDayOfWeek *int             `json:"class_schedules_day_of_week,omitempty" validate:"omitempty,gte=1,lte=7"`
	ClassSchedulesStartTime *string          `json:"class_schedules_start_time,omitempty"`
	ClassSchedulesEndTime   *string          `json:"class_schedules_end_time,omitempty"`
	ClassSchedulesStartDate *string          `json:"class_schedules_start_date,omitempty"`
	ClassSchedulesEndDate   *string          `json:"class_schedules_end_date,omitempty"`
	ClassSchedulesStatus    *m.SessionStatus `json:"class_schedules_status,omitempty"      validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	ClassSchedulesIsActive  *bool            `json:"class_schedules_is_active,omitempty"`

	// Optional relasi
	ClassSchedulesCSSTID  *string `json:"class_schedules_csst_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesEventID *string `json:"class_schedules_event_id,omitempty" validate:"omitempty,uuid4"`
}

/* =======================================================
   Validator registrar (opsional)
   ======================================================= */

func RegisterClassScheduleValidators(v *validator.Validate) {
	// Tag validation standar sudah cukup.
}

/* =======================================================
   Convert & Apply (Create / Update)
   ======================================================= */

func (r *CreateClassScheduleRequest) ApplyToModel(dst *m.ClassScheduleModel) error {
	masjidID, err := uuidFromStringRequired(r.ClassSchedulesMasjidID)
	if err != nil {
		return fmt.Errorf("class_schedules_masjid_id: %w", err)
	}

	startDate, err := parseDate(r.ClassSchedulesStartDate)
	if err != nil {
		return err
	}
	endDate, err := parseDate(r.ClassSchedulesEndDate)
	if err != nil {
		return err
	}
	if endDate.Before(startDate) {
		return errors.New("class_schedules_end_date must be >= class_schedules_start_date")
	}

	startTime, err := parseTimeOnly(r.ClassSchedulesStartTime)
	if err != nil {
		return err
	}
	endTime, err := parseTimeOnly(r.ClassSchedulesEndTime)
	if err != nil {
		return err
	}
	if !endTime.After(startTime) {
		return errors.New("class_schedules_end_time must be greater than start_time")
	}

	csstID, err := uuidPtrFromString(r.ClassSchedulesCSSTID)
	if err != nil {
		return fmt.Errorf("class_schedules_csst_id: %w", err)
	}
	eventID, err := uuidPtrFromString(r.ClassSchedulesEventID)
	if err != nil {
		return fmt.Errorf("class_schedules_event_id: %w", err)
	}

	dst.ClassScheduleMasjidID = masjidID
	dst.ClassScheduleCSSTID = csstID
	dst.ClassScheduleEventID = eventID

	dst.ClassScheduleDayOfWeek = r.ClassSchedulesDayOfWeek
	dst.ClassScheduleStartTime = startTime // time.Time → TIME
	dst.ClassScheduleEndTime = endTime     // time.Time → TIME
	dst.ClassScheduleStartDate = startDate // DATE
	dst.ClassScheduleEndDate = endDate     // DATE

	if r.ClassSchedulesStatus != nil {
		dst.ClassScheduleStatus = *r.ClassSchedulesStatus
	} else {
		dst.ClassScheduleStatus = m.SessionScheduled
	}
	if r.ClassSchedulesIsActive != nil {
		dst.ClassScheduleIsActive = *r.ClassSchedulesIsActive
	} else {
		dst.ClassScheduleIsActive = true
	}

	return nil
}

func (r *UpdateClassScheduleRequest) ApplyToModel(dst *m.ClassScheduleModel) error {
	masjidID, err := uuidFromStringRequired(r.ClassSchedulesMasjidID)
	if err != nil {
		return fmt.Errorf("class_schedules_masjid_id: %w", err)
	}

	startDate, err := parseDate(r.ClassSchedulesStartDate)
	if err != nil {
		return err
	}
	endDate, err := parseDate(r.ClassSchedulesEndDate)
	if err != nil {
		return err
	}
	if endDate.Before(startDate) {
		return errors.New("class_schedules_end_date must be >= class_schedules_start_date")
	}

	startTime, err := parseTimeOnly(r.ClassSchedulesStartTime)
	if err != nil {
		return err
	}
	endTime, err := parseTimeOnly(r.ClassSchedulesEndTime)
	if err != nil {
		return err
	}
	if !endTime.After(startTime) {
		return errors.New("class_schedules_end_time must be greater than start_time")
	}

	csstID, err := uuidPtrFromString(r.ClassSchedulesCSSTID)
	if err != nil {
		return fmt.Errorf("class_schedules_csst_id: %w", err)
	}
	eventID, err := uuidPtrFromString(r.ClassSchedulesEventID)
	if err != nil {
		return fmt.Errorf("class_schedules_event_id: %w", err)
	}

	dst.ClassScheduleMasjidID = masjidID
	dst.ClassScheduleCSSTID = csstID
	dst.ClassScheduleEventID = eventID

	dst.ClassScheduleDayOfWeek = r.ClassSchedulesDayOfWeek
	dst.ClassScheduleStartTime = startTime
	dst.ClassScheduleEndTime = endTime
	dst.ClassScheduleStartDate = startDate
	dst.ClassScheduleEndDate = endDate

	dst.ClassScheduleStatus = m.SessionStatus(r.ClassSchedulesStatus)
	dst.ClassScheduleIsActive = r.ClassSchedulesIsActive

	return nil
}

/* =======================================================
   PATCH — apply only non-nil fields
   ======================================================= */

func (p *PatchClassScheduleRequest) ApplyPatch(dst *m.ClassScheduleModel) error {
	// IDs
	if p.ClassSchedulesMasjidID != nil {
		id, err := uuidFromStringRequired(*p.ClassSchedulesMasjidID)
		if err != nil {
			return fmt.Errorf("class_schedules_masjid_id: %w", err)
		}
		dst.ClassScheduleMasjidID = id
	}

	// Relasi
	if p.ClassSchedulesCSSTID != nil {
		idp, err := uuidPtrFromString(p.ClassSchedulesCSSTID)
		if err != nil {
			return fmt.Errorf("class_schedules_csst_id: %w", err)
		}
		dst.ClassScheduleCSSTID = idp
	}
	if p.ClassSchedulesEventID != nil {
		idp, err := uuidPtrFromString(p.ClassSchedulesEventID)
		if err != nil {
			return fmt.Errorf("class_schedules_event_id: %w", err)
		}
		dst.ClassScheduleEventID = idp
	}

	// Day of week
	if p.ClassSchedulesDayOfWeek != nil {
		if *p.ClassSchedulesDayOfWeek < 1 || *p.ClassSchedulesDayOfWeek > 7 {
			return errors.New("class_schedules_day_of_week must be between 1 and 7")
		}
		dst.ClassScheduleDayOfWeek = *p.ClassSchedulesDayOfWeek
	}

	// Time
	if p.ClassSchedulesStartTime != nil {
		t, err := parseTimeOnly(*p.ClassSchedulesStartTime)
		if err != nil {
			return fmt.Errorf("class_schedules_start_time: %w", err)
		}
		dst.ClassScheduleStartTime = t
	}
	if p.ClassSchedulesEndTime != nil {
		t, err := parseTimeOnly(*p.ClassSchedulesEndTime)
		if err != nil {
			return fmt.Errorf("class_schedules_end_time: %w", err)
		}
		dst.ClassScheduleEndTime = t
	}
	if p.ClassSchedulesStartTime != nil || p.ClassSchedulesEndTime != nil {
		st := dst.ClassScheduleStartTime
		et := dst.ClassScheduleEndTime
		if !et.After(st) {
			return errors.New("class_schedules_end_time must be greater than start_time")
		}
	}

	// Dates
	if p.ClassSchedulesStartDate != nil {
		d, err := parseDate(*p.ClassSchedulesStartDate)
		if err != nil {
			return fmt.Errorf("class_schedules_start_date: %w", err)
		}
		dst.ClassScheduleStartDate = d
	}
	if p.ClassSchedulesEndDate != nil {
		d, err := parseDate(*p.ClassSchedulesEndDate)
		if err != nil {
			return fmt.Errorf("class_schedules_end_date: %w", err)
		}
		dst.ClassScheduleEndDate = d
	}
	if p.ClassSchedulesStartDate != nil || p.ClassSchedulesEndDate != nil {
		if dst.ClassScheduleEndDate.Before(dst.ClassScheduleStartDate) {
			return errors.New("class_schedules_end_date must be >= class_schedules_start_date")
		}
	}

	// Status & Active
	if p.ClassSchedulesStatus != nil {
		switch *p.ClassSchedulesStatus {
		case m.SessionScheduled, m.SessionOngoing, m.SessionCompleted, m.SessionCanceled:
			dst.ClassScheduleStatus = *p.ClassSchedulesStatus
		default:
			return errors.New("invalid class_schedules_status")
		}
	}
	if p.ClassSchedulesIsActive != nil {
		dst.ClassScheduleIsActive = *p.ClassSchedulesIsActive
	}

	return nil
}

/* =======================================================
   Response DTO
   ======================================================= */

type ClassScheduleResponse struct {
	ClassScheduleID        uuid.UUID `json:"class_schedule_id"`
	ClassSchedulesMasjidID uuid.UUID `json:"class_schedules_masjid_id"`

	// opsional → pointer
	ClassSchedulesCSSTID  *uuid.UUID `json:"class_schedules_csst_id,omitempty"`
	ClassSchedulesEventID *uuid.UUID `json:"class_schedules_event_id,omitempty"`

	ClassSchedulesDayOfWeek int    `json:"class_schedules_day_of_week"`
	ClassSchedulesStartTime string `json:"class_schedules_start_time"` // HH:mm:ss
	ClassSchedulesEndTime   string `json:"class_schedules_end_time"`
	ClassSchedulesStartDate string `json:"class_schedules_start_date"` // YYYY-MM-DD
	ClassSchedulesEndDate   string `json:"class_schedules_end_date"`

	ClassSchedulesStatus   m.SessionStatus `json:"class_schedules_status"`
	ClassSchedulesIsActive bool            `json:"class_schedules_is_active"`

	ClassSchedulesCreatedAt time.Time  `json:"class_schedules_created_at"`
	ClassSchedulesUpdatedAt time.Time  `json:"class_schedules_updated_at"`
	ClassSchedulesDeletedAt *time.Time `json:"class_schedules_deleted_at,omitempty"`
}

func NewClassScheduleResponse(src *m.ClassScheduleModel) ClassScheduleResponse {
	var deletedAt *time.Time
	if src.ClassScheduleDeletedAt.Valid {
		deletedAt = &src.ClassScheduleDeletedAt.Time
	}
	return ClassScheduleResponse{
		ClassScheduleID:        src.ClassScheduleID,
		ClassSchedulesMasjidID: src.ClassScheduleMasjidID,

		ClassSchedulesCSSTID:  src.ClassScheduleCSSTID,
		ClassSchedulesEventID: src.ClassScheduleEventID,

		ClassSchedulesDayOfWeek: src.ClassScheduleDayOfWeek,
		ClassSchedulesStartTime: src.ClassScheduleStartTime.Format("15:04:05"),
		ClassSchedulesEndTime:   src.ClassScheduleEndTime.Format("15:04:05"),
		ClassSchedulesStartDate: src.ClassScheduleStartDate.Format(layoutDate),
		ClassSchedulesEndDate:   src.ClassScheduleEndDate.Format(layoutDate),

		ClassSchedulesStatus:   src.ClassScheduleStatus,
		ClassSchedulesIsActive: src.ClassScheduleIsActive,

		ClassSchedulesCreatedAt: src.ClassScheduleCreatedAt,
		ClassSchedulesUpdatedAt: src.ClassScheduleUpdatedAt,
		ClassSchedulesDeletedAt: deletedAt,
	}
}

/* =======================================================
   List Query (disederhanakan)
   ======================================================= */

type ListQuery struct {
	// Filter
	MasjidID        string `query:"masjid_id"`
	CSSTID          string `query:"csst_id"`
	EventID         string `query:"event_id"`
	Status          string `query:"status"`
	Active          *bool  `query:"active"`
	DayOfWeek       *int   `query:"dow"`
	OnDate          string `query:"on_date"`
	StartAfter      string `query:"start_after"`
	EndBefore       string `query:"end_before"`
	ClassScheduleID string `query:"class_schedule_id"`  // single
	IDs             string `query:"class_schedule_ids"` // comma-separated

	// Pagination & sort
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
	SortBy string `query:"sort_by"`
	Order  string `query:"order"`
}

/* =======================================================
   Convenience helpers
   ======================================================= */

func (r *CreateClassScheduleRequest) Validate(v *validator.Validate) error {
	if v == nil {
		return nil
	}
	return v.Struct(r)
}

func (r *UpdateClassScheduleRequest) Validate(v *validator.Validate) error {
	if v == nil {
		return nil
	}
	return v.Struct(r)
}

func (r *PatchClassScheduleRequest) Validate(v *validator.Validate) error {
	if v == nil {
		return nil
	}
	return v.Struct(r)
}
