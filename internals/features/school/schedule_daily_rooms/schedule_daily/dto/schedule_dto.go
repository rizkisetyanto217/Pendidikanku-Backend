// file: internals/features/school/class_schedules/dto/class_schedule_dto.go
package dto

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/schedule_daily_rooms/schedule_daily/model"
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

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}
	// coba HH:mm lalu HH:mm:ss
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

func strPtrOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

/* =======================================================
   Request DTOs
   - Gunakan string untuk tanggal & jam agar simpel dari FE
   ======================================================= */

type CreateClassScheduleRequest struct {
	// Required
	ClassSchedulesMasjidID   string `json:"class_schedules_masjid_id"   validate:"required,uuid4"`
	ClassSchedulesSectionID  string `json:"class_schedules_section_id"  validate:"required,uuid4"`
	ClassSchedulesDayOfWeek  int    `json:"class_schedules_day_of_week" validate:"required,gte=1,lte=7"`
	ClassSchedulesStartTime  string `json:"class_schedules_start_time"  validate:"required"` // "HH:mm" / "HH:mm:ss"
	ClassSchedulesEndTime    string `json:"class_schedules_end_time"    validate:"required"`
	ClassSchedulesStartDate  string `json:"class_schedules_start_date"  validate:"required"` // "YYYY-MM-DD"
	ClassSchedulesEndDate    string `json:"class_schedules_end_date"    validate:"required"`

	// Optional
	ClassSchedulesSubjectID  *string              `json:"class_schedules_subject_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesSemesterID *string              `json:"class_schedules_semester_id,omitempty" validate:"omitempty,uuid4"`
	ClassSchedulesTeacherID  *string              `json:"class_schedules_teacher_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesRoomID     *string              `json:"class_schedules_room_id,omitempty"     validate:"omitempty,uuid4"`
	ClassSchedulesStatus     *m.SessionStatus     `json:"class_schedules_status,omitempty"      validate:"omitempty,oneof=scheduled ongoing finished canceled"`
	ClassSchedulesIsActive   *bool                `json:"class_schedules_is_active,omitempty"`
	ClassSchedulesRoomLabel  *string              `json:"class_schedules_room_label,omitempty"`
}

type UpdateClassScheduleRequest struct {
	// Required — update penuh (PUT-like)
	ClassSchedulesMasjidID   string `json:"class_schedules_masjid_id"   validate:"required,uuid4"`
	ClassSchedulesSectionID  string `json:"class_schedules_section_id"  validate:"required,uuid4"`
	ClassSchedulesDayOfWeek  int    `json:"class_schedules_day_of_week" validate:"required,gte=1,lte=7"`
	ClassSchedulesStartTime  string `json:"class_schedules_start_time"  validate:"required"`
	ClassSchedulesEndTime    string `json:"class_schedules_end_time"    validate:"required"`
	ClassSchedulesStartDate  string `json:"class_schedules_start_date"  validate:"required"`
	ClassSchedulesEndDate    string `json:"class_schedules_end_date"    validate:"required"`
	ClassSchedulesStatus     string `json:"class_schedules_status"      validate:"required,oneof=scheduled ongoing finished canceled"`
	ClassSchedulesIsActive   bool   `json:"class_schedules_is_active"`

	// Optional
	ClassSchedulesSubjectID  *string `json:"class_schedules_subject_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesSemesterID *string `json:"class_schedules_semester_id,omitempty" validate:"omitempty,uuid4"`
	ClassSchedulesTeacherID  *string `json:"class_schedules_teacher_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesRoomID     *string `json:"class_schedules_room_id,omitempty"     validate:"omitempty,uuid4"`
	ClassSchedulesRoomLabel  *string `json:"class_schedules_room_label,omitempty"`
}

type PatchClassScheduleRequest struct {
	// Semua optional — hanya field non-nil yang di-apply
	ClassSchedulesMasjidID   *string          `json:"class_schedules_masjid_id,omitempty"   validate:"omitempty,uuid4"`
	ClassSchedulesSectionID  *string          `json:"class_schedules_section_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesDayOfWeek  *int             `json:"class_schedules_day_of_week,omitempty" validate:"omitempty,gte=1,lte=7"`
	ClassSchedulesStartTime  *string          `json:"class_schedules_start_time,omitempty"`
	ClassSchedulesEndTime    *string          `json:"class_schedules_end_time,omitempty"`
	ClassSchedulesStartDate  *string          `json:"class_schedules_start_date,omitempty"`
	ClassSchedulesEndDate    *string          `json:"class_schedules_end_date,omitempty"`
	ClassSchedulesSubjectID  *string          `json:"class_schedules_subject_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesSemesterID *string          `json:"class_schedules_semester_id,omitempty" validate:"omitempty,uuid4"`
	ClassSchedulesTeacherID  *string          `json:"class_schedules_teacher_id,omitempty"  validate:"omitempty,uuid4"`
	ClassSchedulesRoomID     *string          `json:"class_schedules_room_id,omitempty"     validate:"omitempty,uuid4"`
	ClassSchedulesStatus     *m.SessionStatus `json:"class_schedules_status,omitempty"      validate:"omitempty,oneof=scheduled ongoing finished canceled"`
	ClassSchedulesIsActive   *bool            `json:"class_schedules_is_active,omitempty"`
	ClassSchedulesRoomLabel  *string          `json:"class_schedules_room_label,omitempty"`
}

/* =======================================================
   Validator registrar (opsional)
   ======================================================= */

func RegisterClassScheduleValidators(v *validator.Validate) {
	// Enum status sudah ditangani dengan oneof di tag.
	// Tambahan custom bisa disimpan di sini bila perlu.
}

/* =======================================================
   Convert & Apply (Create / Update)
   ======================================================= */

func (r *CreateClassScheduleRequest) ApplyToModel(dst *m.ClassScheduleModel) error {
	masjidID, _ := uuid.Parse(r.ClassSchedulesMasjidID)
	sectionID, _ := uuid.Parse(r.ClassSchedulesSectionID)

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

	startTime, err := parseTime(r.ClassSchedulesStartTime)
	if err != nil {
		return err
	}
	endTime, err := parseTime(r.ClassSchedulesEndTime)
	if err != nil {
		return err
	}
	if !endTime.After(startTime) {
		return errors.New("class_schedules_end_time must be greater than start_time")
	}

	subjectID, err := uuidPtrFromString(r.ClassSchedulesSubjectID)
	if err != nil {
		return err
	}
	semesterID, err := uuidPtrFromString(r.ClassSchedulesSemesterID)
	if err != nil {
		return err
	}
	teacherID, err := uuidPtrFromString(r.ClassSchedulesTeacherID)
	if err != nil {
		return err
	}
	roomID, err := uuidPtrFromString(r.ClassSchedulesRoomID)
	if err != nil {
		return err
	}

	dst.ClassSchedulesMasjidID = masjidID
	dst.ClassSchedulesSectionID = sectionID
	dst.ClassSchedulesDayOfWeek = r.ClassSchedulesDayOfWeek
	dst.ClassSchedulesStartTime = startTime
	dst.ClassSchedulesEndTime = endTime
	dst.ClassSchedulesStartDate = startDate
	dst.ClassSchedulesEndDate = endDate

	if r.ClassSchedulesStatus != nil {
		dst.ClassSchedulesStatus = *r.ClassSchedulesStatus
	} else {
		dst.ClassSchedulesStatus = m.SessionScheduled
	}
	if r.ClassSchedulesIsActive != nil {
		dst.ClassSchedulesIsActive = *r.ClassSchedulesIsActive
	} else {
		dst.ClassSchedulesIsActive = true
	}

	dst.ClassSchedulesSubjectID = subjectID
	dst.ClassSchedulesSemesterID = semesterID
	dst.ClassSchedulesTeacherID = teacherID
	dst.ClassSchedulesRoomID = roomID
	dst.ClassSchedulesRoomLabel = strPtrOrNil(r.ClassSchedulesRoomLabel)

	return nil
}

func (r *UpdateClassScheduleRequest) ApplyToModel(dst *m.ClassScheduleModel) error {
	masjidID, _ := uuid.Parse(r.ClassSchedulesMasjidID)
	sectionID, _ := uuid.Parse(r.ClassSchedulesSectionID)

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

	startTime, err := parseTime(r.ClassSchedulesStartTime)
	if err != nil {
		return err
	}
	endTime, err := parseTime(r.ClassSchedulesEndTime)
	if err != nil {
		return err
	}
	if !endTime.After(startTime) {
		return errors.New("class_schedules_end_time must be greater than start_time")
	}

	subjectID, err := uuidPtrFromString(r.ClassSchedulesSubjectID)
	if err != nil {
		return err
	}
	semesterID, err := uuidPtrFromString(r.ClassSchedulesSemesterID)
	if err != nil {
		return err
	}
	teacherID, err := uuidPtrFromString(r.ClassSchedulesTeacherID)
	if err != nil {
		return err
	}
	roomID, err := uuidPtrFromString(r.ClassSchedulesRoomID)
	if err != nil {
		return err
	}

	dst.ClassSchedulesMasjidID = masjidID
	dst.ClassSchedulesSectionID = sectionID
	dst.ClassSchedulesDayOfWeek = r.ClassSchedulesDayOfWeek
	dst.ClassSchedulesStartTime = startTime
	dst.ClassSchedulesEndTime = endTime
	dst.ClassSchedulesStartDate = startDate
	dst.ClassSchedulesEndDate = endDate

	// status & active
	dst.ClassSchedulesStatus = m.SessionStatus(r.ClassSchedulesStatus)
	dst.ClassSchedulesIsActive = r.ClassSchedulesIsActive

	dst.ClassSchedulesSubjectID = subjectID
	dst.ClassSchedulesSemesterID = semesterID
	dst.ClassSchedulesTeacherID = teacherID
	dst.ClassSchedulesRoomID = roomID
	dst.ClassSchedulesRoomLabel = strPtrOrNil(r.ClassSchedulesRoomLabel)

	return nil
}

/* =======================================================
   PATCH — apply only non-nil fields
   ======================================================= */

func (p *PatchClassScheduleRequest) ApplyPatch(dst *m.ClassScheduleModel) error {
	// IDs
	if p.ClassSchedulesMasjidID != nil {
		id, err := uuid.Parse(strings.TrimSpace(*p.ClassSchedulesMasjidID))
		if err != nil {
			return fmt.Errorf("class_schedules_masjid_id: %w", err)
		}
		dst.ClassSchedulesMasjidID = id
	}
	if p.ClassSchedulesSectionID != nil {
		id, err := uuid.Parse(strings.TrimSpace(*p.ClassSchedulesSectionID))
		if err != nil {
			return fmt.Errorf("class_schedules_section_id: %w", err)
		}
		dst.ClassSchedulesSectionID = id
	}

	// Subject, Term, Teacher, Room
	if p.ClassSchedulesSubjectID != nil {
		idp, err := uuidPtrFromString(p.ClassSchedulesSubjectID)
		if err != nil {
			return fmt.Errorf("class_schedules_subject_id: %w", err)
		}
		dst.ClassSchedulesSubjectID = idp
	}
	if p.ClassSchedulesSemesterID != nil {
		idp, err := uuidPtrFromString(p.ClassSchedulesSemesterID)
		if err != nil {
			return fmt.Errorf("class_schedules_semester_id: %w", err)
		}
		dst.ClassSchedulesSemesterID = idp
	}
	if p.ClassSchedulesTeacherID != nil {
		idp, err := uuidPtrFromString(p.ClassSchedulesTeacherID)
		if err != nil {
			return fmt.Errorf("class_schedules_teacher_id: %w", err)
		}
		dst.ClassSchedulesTeacherID = idp
	}
	if p.ClassSchedulesRoomID != nil {
		idp, err := uuidPtrFromString(p.ClassSchedulesRoomID)
		if err != nil {
			return fmt.Errorf("class_schedules_room_id: %w", err)
		}
		dst.ClassSchedulesRoomID = idp
	}

	// Day of week
	if p.ClassSchedulesDayOfWeek != nil {
		if *p.ClassSchedulesDayOfWeek < 1 || *p.ClassSchedulesDayOfWeek > 7 {
			return errors.New("class_schedules_day_of_week must be between 1 and 7")
		}
		dst.ClassSchedulesDayOfWeek = *p.ClassSchedulesDayOfWeek
	}

	// Time
	if p.ClassSchedulesStartTime != nil {
		t, err := parseTime(*p.ClassSchedulesStartTime)
		if err != nil {
			return fmt.Errorf("class_schedules_start_time: %w", err)
		}
		dst.ClassSchedulesStartTime = t
	}
	if p.ClassSchedulesEndTime != nil {
		t, err := parseTime(*p.ClassSchedulesEndTime)
		if err != nil {
			return fmt.Errorf("class_schedules_end_time: %w", err)
		}
		dst.ClassSchedulesEndTime = t
	}
	// Ensure time validity if any changed
	if p.ClassSchedulesStartTime != nil || p.ClassSchedulesEndTime != nil {
		if !dst.ClassSchedulesEndTime.After(dst.ClassSchedulesStartTime) {
			return errors.New("class_schedules_end_time must be greater than start_time")
		}
	}

	// Date
	if p.ClassSchedulesStartDate != nil {
		d, err := parseDate(*p.ClassSchedulesStartDate)
		if err != nil {
			return fmt.Errorf("class_schedules_start_date: %w", err)
		}
		dst.ClassSchedulesStartDate = d
	}
	if p.ClassSchedulesEndDate != nil {
		d, err := parseDate(*p.ClassSchedulesEndDate)
		if err != nil {
			return fmt.Errorf("class_schedules_end_date: %w", err)
		}
		dst.ClassSchedulesEndDate = d
	}
	// Ensure date validity if any changed
	if p.ClassSchedulesStartDate != nil || p.ClassSchedulesEndDate != nil {
		if dst.ClassSchedulesEndDate.Before(dst.ClassSchedulesStartDate) {
			return errors.New("class_schedules_end_date must be >= class_schedules_start_date")
		}
	}

	// Status & metadata
	if p.ClassSchedulesStatus != nil {
		switch *p.ClassSchedulesStatus {
		case m.SessionScheduled, m.SessionOngoing, m.SessionFinished, m.SessionCanceled:
			dst.ClassSchedulesStatus = *p.ClassSchedulesStatus
		default:
			return errors.New("invalid class_schedules_status")
		}
	}
	if p.ClassSchedulesIsActive != nil {
		dst.ClassSchedulesIsActive = *p.ClassSchedulesIsActive
	}
	if p.ClassSchedulesRoomLabel != nil {
		dst.ClassSchedulesRoomLabel = strPtrOrNil(p.ClassSchedulesRoomLabel)
	}

	return nil
}

/* =======================================================
   Response DTO
   ======================================================= */

type ClassScheduleResponse struct {
	ClassScheduleID           uuid.UUID        `json:"class_schedule_id"`
	ClassSchedulesMasjidID    uuid.UUID        `json:"class_schedules_masjid_id"`
	ClassSchedulesSectionID   uuid.UUID        `json:"class_schedules_section_id"`
	ClassSchedulesSubjectID   *uuid.UUID       `json:"class_schedules_subject_id,omitempty"`
	ClassSchedulesSemesterID  *uuid.UUID       `json:"class_schedules_semester_id,omitempty"`
	ClassSchedulesTeacherID   *uuid.UUID       `json:"class_schedules_teacher_id,omitempty"`
	ClassSchedulesRoomID      *uuid.UUID       `json:"class_schedules_room_id,omitempty"`
	ClassSchedulesDayOfWeek   int              `json:"class_schedules_day_of_week"`
	ClassSchedulesStartTime   string           `json:"class_schedules_start_time"` // HH:mm:ss
	ClassSchedulesEndTime     string           `json:"class_schedules_end_time"`
	ClassSchedulesStartDate   string           `json:"class_schedules_start_date"` // YYYY-MM-DD
	ClassSchedulesEndDate     string           `json:"class_schedules_end_date"`
	ClassSchedulesStatus      m.SessionStatus  `json:"class_schedules_status"`
	ClassSchedulesIsActive    bool             `json:"class_schedules_is_active"`
	ClassSchedulesRoomLabel   *string          `json:"class_schedules_room_label,omitempty"`
	ClassSchedulesTimeRange   *string          `json:"class_schedules_time_range,omitempty"`
	ClassSchedulesCreatedAt   time.Time        `json:"class_schedules_created_at"`
	ClassSchedulesUpdatedAt   time.Time        `json:"class_schedules_updated_at"`
}

func NewClassScheduleResponse(src *m.ClassScheduleModel) ClassScheduleResponse {
	return ClassScheduleResponse{
		ClassScheduleID:          src.ClassScheduleID,
		ClassSchedulesMasjidID:   src.ClassSchedulesMasjidID,
		ClassSchedulesSectionID:  src.ClassSchedulesSectionID,
		ClassSchedulesSubjectID:  src.ClassSchedulesSubjectID,
		ClassSchedulesSemesterID: src.ClassSchedulesSemesterID,
		ClassSchedulesTeacherID:  src.ClassSchedulesTeacherID,
		ClassSchedulesRoomID:     src.ClassSchedulesRoomID,
		ClassSchedulesDayOfWeek:  src.ClassSchedulesDayOfWeek,
		ClassSchedulesStartTime:  src.ClassSchedulesStartTime.Format("15:04:05"),
		ClassSchedulesEndTime:    src.ClassSchedulesEndTime.Format("15:04:05"),
		ClassSchedulesStartDate:  src.ClassSchedulesStartDate.Format(layoutDate),
		ClassSchedulesEndDate:    src.ClassSchedulesEndDate.Format(layoutDate),
		ClassSchedulesStatus:     src.ClassSchedulesStatus,
		ClassSchedulesIsActive:   src.ClassSchedulesIsActive,
		ClassSchedulesRoomLabel:  src.ClassSchedulesRoomLabel,
		ClassSchedulesTimeRange:  src.ClassSchedulesTimeRange,
		ClassSchedulesCreatedAt:  src.ClassSchedulesCreatedAt,
		ClassSchedulesUpdatedAt:  src.ClassSchedulesUpdatedAt,
	}
}

/* =======================================================
   Convenience helpers untuk handler
   ======================================================= */

// ValidateCreate — panggil sebelum ApplyToModel
func (r *CreateClassScheduleRequest) Validate(v *validator.Validate) error {
	if err := v.Struct(r); err != nil {
		return err
	}
	// Validasi cross-field (time & date order) dilakukan di ApplyToModel agar reuse.
	return nil
}

// ValidateUpdate — panggil sebelum ApplyToModel
func (r *UpdateClassScheduleRequest) Validate(v *validator.Validate) error {
	return v.Struct(r)
}

// ValidatePatch — panggil sebelum ApplyPatch
func (r *PatchClassScheduleRequest) Validate(v *validator.Validate) error {
	return v.Struct(r)
}
