// file: internals/features/school/class_daily/dto/class_daily_dto.go
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


/* =======================================================
   Request DTOs
   ======================================================= */

type CreateClassDailyRequest struct {
	// Required
	ClassDailyMasjidID  string `json:"class_daily_masjid_id"  validate:"required,uuid4"`
	ClassDailySectionID string `json:"class_daily_section_id" validate:"required,uuid4"`
	ClassDailyDate      string `json:"class_daily_date"       validate:"required"` // "YYYY-MM-DD"
	ClassDailyStartTime string `json:"class_daily_start_time" validate:"required"` // "HH:mm" / "HH:mm:ss"
	ClassDailyEndTime   string `json:"class_daily_end_time"   validate:"required"`

	// Optional link/snapshot/metadata
	ClassDailyScheduleID      *string           `json:"class_daily_schedule_id,omitempty"      validate:"omitempty,uuid4"`
	ClassDailyAttendanceID    *string           `json:"class_daily_attendance_id,omitempty"    validate:"omitempty,uuid4"`
	ClassDailySubjectID       *string           `json:"class_daily_subject_id,omitempty"       validate:"omitempty,uuid4"`
	ClassDailyAcademicTermsID *string           `json:"class_daily_academic_terms_id,omitempty" validate:"omitempty,uuid4"`
	ClassDailyTeacherID       *string           `json:"class_daily_teacher_id,omitempty"       validate:"omitempty,uuid4"`
	ClassDailyRoomID          *string           `json:"class_daily_room_id,omitempty"          validate:"omitempty,uuid4"`
	ClassDailyStatus          *m.SessionStatus  `json:"class_daily_status,omitempty"           validate:"omitempty,oneof=scheduled ongoing finished canceled"`
	ClassDailyIsActive        *bool             `json:"class_daily_is_active,omitempty"`
	ClassDailyRoomLabel       *string           `json:"class_daily_room_label,omitempty"`
}

type UpdateClassDailyRequest struct {
	// Required — update penuh (PUT-like)
	ClassDailyMasjidID  string `json:"class_daily_masjid_id"  validate:"required,uuid4"`
	ClassDailySectionID string `json:"class_daily_section_id" validate:"required,uuid4"`
	ClassDailyDate      string `json:"class_daily_date"       validate:"required"`
	ClassDailyStartTime string `json:"class_daily_start_time" validate:"required"`
	ClassDailyEndTime   string `json:"class_daily_end_time"   validate:"required"`
	ClassDailyStatus    string `json:"class_daily_status"     validate:"required,oneof=scheduled ongoing finished canceled"`
	ClassDailyIsActive  bool   `json:"class_daily_is_active"`

	// Optional link/snapshot/metadata
	ClassDailyScheduleID      *string `json:"class_daily_schedule_id,omitempty"      validate:"omitempty,uuid4"`
	ClassDailyAttendanceID    *string `json:"class_daily_attendance_id,omitempty"    validate:"omitempty,uuid4"`
	ClassDailySubjectID       *string `json:"class_daily_subject_id,omitempty"       validate:"omitempty,uuid4"`
	ClassDailyAcademicTermsID *string `json:"class_daily_academic_terms_id,omitempty" validate:"omitempty,uuid4"`
	ClassDailyTeacherID       *string `json:"class_daily_teacher_id,omitempty"       validate:"omitempty,uuid4"`
	ClassDailyRoomID          *string `json:"class_daily_room_id,omitempty"          validate:"omitempty,uuid4"`
	ClassDailyRoomLabel       *string `json:"class_daily_room_label,omitempty"`
}

type PatchClassDailyRequest struct {
	// Semua optional — hanya field non-nil yang di-apply
	ClassDailyMasjidID        *string          `json:"class_daily_masjid_id,omitempty"        validate:"omitempty,uuid4"`
	ClassDailySectionID       *string          `json:"class_daily_section_id,omitempty"       validate:"omitempty,uuid4"`
	ClassDailyDate            *string          `json:"class_daily_date,omitempty"`
	ClassDailyStartTime       *string          `json:"class_daily_start_time,omitempty"`
	ClassDailyEndTime         *string          `json:"class_daily_end_time,omitempty"`
	ClassDailyScheduleID      *string          `json:"class_daily_schedule_id,omitempty"      validate:"omitempty,uuid4"`
	ClassDailyAttendanceID    *string          `json:"class_daily_attendance_id,omitempty"    validate:"omitempty,uuid4"`
	ClassDailySubjectID       *string          `json:"class_daily_subject_id,omitempty"       validate:"omitempty,uuid4"`
	ClassDailyAcademicTermsID *string          `json:"class_daily_academic_terms_id,omitempty" validate:"omitempty,uuid4"`
	ClassDailyTeacherID       *string          `json:"class_daily_teacher_id,omitempty"       validate:"omitempty,uuid4"`
	ClassDailyRoomID          *string          `json:"class_daily_room_id,omitempty"          validate:"omitempty,uuid4"`
	ClassDailyStatus          *m.SessionStatus `json:"class_daily_status,omitempty"           validate:"omitempty,oneof=scheduled ongoing finished canceled"`
	ClassDailyIsActive        *bool            `json:"class_daily_is_active,omitempty"`
	ClassDailyRoomLabel       *string          `json:"class_daily_room_label,omitempty"`
}

/* =======================================================
   Optional: Validator registrar
   (aman di-skip; Validate() handle v == nil)
   ======================================================= */

func RegisterClassDailyValidators(v *validator.Validate) {
	// Enum status sudah ditangani dengan tag oneof
}

/* =======================================================
   Convert & Apply (Create / Update)
   ======================================================= */

func (r *CreateClassDailyRequest) ApplyToModel(dst *m.ClassDailyModel) error {
	masjidID, _ := uuid.Parse(r.ClassDailyMasjidID)
	sectionID, _ := uuid.Parse(r.ClassDailySectionID)

	date, err := parseDate(r.ClassDailyDate)
	if err != nil {
		return err
	}

	startTime, err := parseTime(r.ClassDailyStartTime)
	if err != nil {
		return err
	}
	endTime, err := parseTime(r.ClassDailyEndTime)
	if err != nil {
		return err
	}
	if !endTime.After(startTime) {
		return errors.New("class_daily_end_time must be greater than start_time")
	}

	scheduleID, err := uuidPtrFromString(r.ClassDailyScheduleID)
	if err != nil {
		return err
	}
	attendanceID, err := uuidPtrFromString(r.ClassDailyAttendanceID)
	if err != nil {
		return err
	}
	subjectID, err := uuidPtrFromString(r.ClassDailySubjectID)
	if err != nil {
		return err
	}
	termID, err := uuidPtrFromString(r.ClassDailyAcademicTermsID)
	if err != nil {
		return err
	}
	teacherID, err := uuidPtrFromString(r.ClassDailyTeacherID)
	if err != nil {
		return err
	}
	roomID, err := uuidPtrFromString(r.ClassDailyRoomID)
	if err != nil {
		return err
	}

	dst.ClassDailyMasjidID = masjidID
	dst.ClassDailySectionID = sectionID
	dst.ClassDailyDate = date
	dst.ClassDailyStartTime = startTime
	dst.ClassDailyEndTime = endTime

	if r.ClassDailyStatus != nil {
		dst.ClassDailyStatus = *r.ClassDailyStatus
	} else {
		dst.ClassDailyStatus = m.SessionScheduled
	}
	if r.ClassDailyIsActive != nil {
		dst.ClassDailyIsActive = *r.ClassDailyIsActive
	} else {
		dst.ClassDailyIsActive = true
	}

	dst.ClassDailyScheduleID = scheduleID
	dst.ClassDailyAttendanceID = attendanceID
	dst.ClassDailySubjectID = subjectID
	dst.ClassDailyAcademicTermsID = termID
	dst.ClassDailyTeacherID = teacherID
	dst.ClassDailyRoomID = roomID
	dst.ClassDailyRoomLabel = strPtrOrNil(r.ClassDailyRoomLabel)

	return nil
}

func (r *UpdateClassDailyRequest) ApplyToModel(dst *m.ClassDailyModel) error {
	masjidID, _ := uuid.Parse(r.ClassDailyMasjidID)
	sectionID, _ := uuid.Parse(r.ClassDailySectionID)

	date, err := parseDate(r.ClassDailyDate)
	if err != nil {
		return err
	}

	startTime, err := parseTime(r.ClassDailyStartTime)
	if err != nil {
		return err
	}
	endTime, err := parseTime(r.ClassDailyEndTime)
	if err != nil {
		return err
	}
	if !endTime.After(startTime) {
		return errors.New("class_daily_end_time must be greater than start_time")
	}

	scheduleID, err := uuidPtrFromString(r.ClassDailyScheduleID)
	if err != nil {
		return err
	}
	attendanceID, err := uuidPtrFromString(r.ClassDailyAttendanceID)
	if err != nil {
		return err
	}
	subjectID, err := uuidPtrFromString(r.ClassDailySubjectID)
	if err != nil {
		return err
	}
	termID, err := uuidPtrFromString(r.ClassDailyAcademicTermsID)
	if err != nil {
		return err
	}
	teacherID, err := uuidPtrFromString(r.ClassDailyTeacherID)
	if err != nil {
		return err
	}
	roomID, err := uuidPtrFromString(r.ClassDailyRoomID)
	if err != nil {
		return err
	}

	dst.ClassDailyMasjidID = masjidID
	dst.ClassDailySectionID = sectionID
	dst.ClassDailyDate = date
	dst.ClassDailyStartTime = startTime
	dst.ClassDailyEndTime = endTime

	// status & active
	dst.ClassDailyStatus = m.SessionStatus(r.ClassDailyStatus)
	dst.ClassDailyIsActive = r.ClassDailyIsActive

	dst.ClassDailyScheduleID = scheduleID
	dst.ClassDailyAttendanceID = attendanceID
	dst.ClassDailySubjectID = subjectID
	dst.ClassDailyAcademicTermsID = termID
	dst.ClassDailyTeacherID = teacherID
	dst.ClassDailyRoomID = roomID
	dst.ClassDailyRoomLabel = strPtrOrNil(r.ClassDailyRoomLabel)

	return nil
}

/* =======================================================
   PATCH — apply only non-nil fields
   ======================================================= */

func (p *PatchClassDailyRequest) ApplyPatch(dst *m.ClassDailyModel) error {
	// IDs
	if p.ClassDailyMasjidID != nil {
		id, err := uuid.Parse(strings.TrimSpace(*p.ClassDailyMasjidID))
		if err != nil {
			return fmt.Errorf("class_daily_masjid_id: %w", err)
		}
		dst.ClassDailyMasjidID = id
	}
	if p.ClassDailySectionID != nil {
		id, err := uuid.Parse(strings.TrimSpace(*p.ClassDailySectionID))
		if err != nil {
			return fmt.Errorf("class_daily_section_id: %w", err)
		}
		dst.ClassDailySectionID = id
	}

	// Link IDs
	if p.ClassDailyScheduleID != nil {
		idp, err := uuidPtrFromString(p.ClassDailyScheduleID)
		if err != nil {
			return fmt.Errorf("class_daily_schedule_id: %w", err)
		}
		dst.ClassDailyScheduleID = idp
	}
	if p.ClassDailyAttendanceID != nil {
		idp, err := uuidPtrFromString(p.ClassDailyAttendanceID)
		if err != nil {
			return fmt.Errorf("class_daily_attendance_id: %w", err)
		}
		dst.ClassDailyAttendanceID = idp
	}
	if p.ClassDailySubjectID != nil {
		idp, err := uuidPtrFromString(p.ClassDailySubjectID)
		if err != nil {
			return fmt.Errorf("class_daily_subject_id: %w", err)
		}
		dst.ClassDailySubjectID = idp
	}
	if p.ClassDailyAcademicTermsID != nil {
		idp, err := uuidPtrFromString(p.ClassDailyAcademicTermsID)
		if err != nil {
			return fmt.Errorf("class_daily_academic_terms_id: %w", err)
		}
		dst.ClassDailyAcademicTermsID = idp
	}
	if p.ClassDailyTeacherID != nil {
		idp, err := uuidPtrFromString(p.ClassDailyTeacherID)
		if err != nil {
			return fmt.Errorf("class_daily_teacher_id: %w", err)
		}
		dst.ClassDailyTeacherID = idp
	}
	if p.ClassDailyRoomID != nil {
		idp, err := uuidPtrFromString(p.ClassDailyRoomID)
		if err != nil {
			return fmt.Errorf("class_daily_room_id: %w", err)
		}
		dst.ClassDailyRoomID = idp
	}

	// Date
	if p.ClassDailyDate != nil {
		d, err := parseDate(*p.ClassDailyDate)
		if err != nil {
			return fmt.Errorf("class_daily_date: %w", err)
		}
		dst.ClassDailyDate = d
	}

	// Time
	if p.ClassDailyStartTime != nil {
		t, err := parseTime(*p.ClassDailyStartTime)
		if err != nil {
			return fmt.Errorf("class_daily_start_time: %w", err)
		}
		dst.ClassDailyStartTime = t
	}
	if p.ClassDailyEndTime != nil {
		t, err := parseTime(*p.ClassDailyEndTime)
		if err != nil {
			return fmt.Errorf("class_daily_end_time: %w", err)
		}
		dst.ClassDailyEndTime = t
	}
	// Ensure time validity if any changed
	if p.ClassDailyStartTime != nil || p.ClassDailyEndTime != nil {
		if !dst.ClassDailyEndTime.After(dst.ClassDailyStartTime) {
			return errors.New("class_daily_end_time must be greater than start_time")
		}
	}

	// Status & metadata
	if p.ClassDailyStatus != nil {
		switch *p.ClassDailyStatus {
		case m.SessionScheduled, m.SessionOngoing, m.SessionFinished, m.SessionCanceled:
			dst.ClassDailyStatus = *p.ClassDailyStatus
		default:
			return errors.New("invalid class_daily_status")
		}
	}
	if p.ClassDailyIsActive != nil {
		dst.ClassDailyIsActive = *p.ClassDailyIsActive
	}
	if p.ClassDailyRoomLabel != nil {
		dst.ClassDailyRoomLabel = strPtrOrNil(p.ClassDailyRoomLabel)
	}

	return nil
}

/* =======================================================
   Response DTO
   ======================================================= */

type ClassDailyResponse struct {
	ClassDailyID             uuid.UUID       `json:"class_daily_id"`
	ClassDailyMasjidID       uuid.UUID       `json:"class_daily_masjid_id"`
	ClassDailySectionID      uuid.UUID       `json:"class_daily_section_id"`
	ClassDailyScheduleID     *uuid.UUID      `json:"class_daily_schedule_id,omitempty"`
	ClassDailyAttendanceID   *uuid.UUID      `json:"class_daily_attendance_id,omitempty"`
	ClassDailySubjectID      *uuid.UUID      `json:"class_daily_subject_id,omitempty"`
	ClassDailyAcademicTermsID *uuid.UUID     `json:"class_daily_academic_terms_id,omitempty"`
	ClassDailyTeacherID      *uuid.UUID      `json:"class_daily_teacher_id,omitempty"`
	ClassDailyRoomID         *uuid.UUID      `json:"class_daily_room_id,omitempty"`

	ClassDailyDate      string          `json:"class_daily_date"`       // YYYY-MM-DD
	ClassDailyStartTime string          `json:"class_daily_start_time"` // HH:mm:ss
	ClassDailyEndTime   string          `json:"class_daily_end_time"`

	ClassDailyStatus    m.SessionStatus `json:"class_daily_status"`
	ClassDailyIsActive  bool            `json:"class_daily_is_active"`
	ClassDailyRoomLabel *string         `json:"class_daily_room_label,omitempty"`

	ClassDailyDayOfWeek int      `json:"class_daily_day_of_week"`
	ClassDailyTimeRange *string  `json:"class_daily_time_range,omitempty"`

	ClassDailyCreatedAt time.Time `json:"class_daily_created_at"`
	ClassDailyUpdatedAt time.Time `json:"class_daily_updated_at"`
}

func NewClassDailyResponse(src *m.ClassDailyModel) ClassDailyResponse {
	return ClassDailyResponse{
		ClassDailyID:              src.ClassDailyID,
		ClassDailyMasjidID:        src.ClassDailyMasjidID,
		ClassDailySectionID:       src.ClassDailySectionID,
		ClassDailyScheduleID:      src.ClassDailyScheduleID,
		ClassDailyAttendanceID:    src.ClassDailyAttendanceID,
		ClassDailySubjectID:       src.ClassDailySubjectID,
		ClassDailyAcademicTermsID: src.ClassDailyAcademicTermsID,
		ClassDailyTeacherID:       src.ClassDailyTeacherID,
		ClassDailyRoomID:          src.ClassDailyRoomID,

		ClassDailyDate:      src.ClassDailyDate.Format(layoutDate),
		ClassDailyStartTime: src.ClassDailyStartTime.Format("15:04:05"),
		ClassDailyEndTime:   src.ClassDailyEndTime.Format("15:04:05"),

		ClassDailyStatus:    src.ClassDailyStatus,
		ClassDailyIsActive:  src.ClassDailyIsActive,
		ClassDailyRoomLabel: src.ClassDailyRoomLabel,

		ClassDailyDayOfWeek: src.ClassDailyDayOfWeek,
		ClassDailyTimeRange: src.ClassDailyTimeRange,

		ClassDailyCreatedAt: src.ClassDailyCreatedAt,
		ClassDailyUpdatedAt: src.ClassDailyUpdatedAt,
	}
}

/* =======================================================
   Convenience helpers untuk handler
   ======================================================= */

// Nil-safe: kalau v == nil, lewati validasi tag
func (r *CreateClassDailyRequest) Validate(v *validator.Validate) error {
	if v == nil {
		return nil
	}
	return v.Struct(r)
}

func (r *UpdateClassDailyRequest) Validate(v *validator.Validate) error {
	if v == nil {
		return nil
	}
	return v.Struct(r)
}

func (r *PatchClassDailyRequest) Validate(v *validator.Validate) error {
	if v == nil {
		return nil
	}
	return v.Struct(r)
}
