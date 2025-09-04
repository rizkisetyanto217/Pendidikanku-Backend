// file: internals/features/school/class_daily/dto/class_daily_dto.go
package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/schedule_daily_rooms/schedule_daily/model"
)


/* =======================================================
   Request DTOs
   ======================================================= */

type CreateClassDailyRequest struct {
	// Required
	ClassDailyMasjidID  string `json:"class_daily_masjid_id"  validate:"required,uuid4"`
	ClassDailySectionID string `json:"class_daily_section_id" validate:"required,uuid4"`
	ClassDailyDate      string `json:"class_daily_date"       validate:"required"` // "YYYY-MM-DD"

	// Optional
	ClassDailyIsActive *bool `json:"class_daily_is_active,omitempty"`
}

type UpdateClassDailyRequest struct {
	// PUT-like (semua wajib)
	ClassDailyMasjidID  string `json:"class_daily_masjid_id"  validate:"required,uuid4"`
	ClassDailySectionID string `json:"class_daily_section_id" validate:"required,uuid4"`
	ClassDailyDate      string `json:"class_daily_date"       validate:"required"`
	ClassDailyIsActive  bool   `json:"class_daily_is_active"`
}

type PatchClassDailyRequest struct {
	// Semua optional — hanya field non-nil yang di-apply
	ClassDailyMasjidID  *string `json:"class_daily_masjid_id,omitempty"  validate:"omitempty,uuid4"`
	ClassDailySectionID *string `json:"class_daily_section_id,omitempty" validate:"omitempty,uuid4"`
	ClassDailyDate      *string `json:"class_daily_date,omitempty"`
	ClassDailyIsActive  *bool   `json:"class_daily_is_active,omitempty"`
}

/* =======================================================
   Optional: Validator registrar
   ======================================================= */

func RegisterClassDailyValidators(v *validator.Validate) {
	// pakai tag bawaan (uuid4, required); tidak perlu custom
}

/* =======================================================
   Convert & Apply (Create / Update / Patch)
   ======================================================= */

func (r *CreateClassDailyRequest) ApplyToModel(dst *m.ClassDailyModel) error {
	masjidID, err := uuid.Parse(strings.TrimSpace(r.ClassDailyMasjidID))
	if err != nil {
		return fmt.Errorf("class_daily_masjid_id: %w", err)
	}
	sectionID, err := uuid.Parse(strings.TrimSpace(r.ClassDailySectionID))
	if err != nil {
		return fmt.Errorf("class_daily_section_id: %w", err)
	}
	date, err := parseDate(r.ClassDailyDate)
	if err != nil {
		return fmt.Errorf("class_daily_date: %w", err)
	}

	dst.ClassDailyMasjidID = masjidID
	dst.ClassDailySectionID = sectionID
	dst.ClassDailyDate = date

	if r.ClassDailyIsActive != nil {
		dst.ClassDailyIsActive = *r.ClassDailyIsActive
	} else {
		dst.ClassDailyIsActive = true
	}
	return nil
}

func (r *UpdateClassDailyRequest) ApplyToModel(dst *m.ClassDailyModel) error {
	masjidID, err := uuid.Parse(strings.TrimSpace(r.ClassDailyMasjidID))
	if err != nil {
		return fmt.Errorf("class_daily_masjid_id: %w", err)
	}
	sectionID, err := uuid.Parse(strings.TrimSpace(r.ClassDailySectionID))
	if err != nil {
		return fmt.Errorf("class_daily_section_id: %w", err)
	}
	date, err := parseDate(r.ClassDailyDate)
	if err != nil {
		return fmt.Errorf("class_daily_date: %w", err)
	}

	dst.ClassDailyMasjidID = masjidID
	dst.ClassDailySectionID = sectionID
	dst.ClassDailyDate = date
	dst.ClassDailyIsActive = r.ClassDailyIsActive
	return nil
}

func (p *PatchClassDailyRequest) ApplyPatch(dst *m.ClassDailyModel) error {
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
	if p.ClassDailyDate != nil {
		d, err := parseDate(*p.ClassDailyDate)
		if err != nil {
			return fmt.Errorf("class_daily_date: %w", err)
		}
		dst.ClassDailyDate = d
	}
	if p.ClassDailyIsActive != nil {
		dst.ClassDailyIsActive = *p.ClassDailyIsActive
	}
	return nil
}

/* =======================================================
   Response DTO
   ======================================================= */

type ClassDailyResponse struct {
	ClassDailyID        uuid.UUID `json:"class_daily_id"`
	ClassDailyMasjidID  uuid.UUID `json:"class_daily_masjid_id"`
	ClassDailySectionID uuid.UUID `json:"class_daily_section_id"`

	ClassDailyDate     string `json:"class_daily_date"` // YYYY-MM-DD
	ClassDailyIsActive bool   `json:"class_daily_is_active"`

	ClassDailyDayOfWeek     int       `json:"class_daily_day_of_week"`
	ClassDailyCreatedAt     time.Time `json:"class_daily_created_at"`
	ClassDailyUpdatedAt     time.Time `json:"class_daily_updated_at"`
	// (opsional) tambahkan DeletedAt jika ingin diekspos:
	// ClassDailyDeletedAt *time.Time `json:"class_daily_deleted_at,omitempty"`
}

func NewClassDailyResponse(src *m.ClassDailyModel) ClassDailyResponse {
	return ClassDailyResponse{
		ClassDailyID:        src.ClassDailyID,
		ClassDailyMasjidID:  src.ClassDailyMasjidID,
		ClassDailySectionID: src.ClassDailySectionID,

		ClassDailyDate:     src.ClassDailyDate.Format(layoutDate),
		ClassDailyIsActive: src.ClassDailyIsActive,

		ClassDailyDayOfWeek: src.ClassDailyDayOfWeek,
		ClassDailyCreatedAt: src.ClassDailyCreatedAt,
		ClassDailyUpdatedAt: src.ClassDailyUpdatedAt,
	}
}

/* =======================================================
   Convenience helpers untuk handler
   ======================================================= */

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
