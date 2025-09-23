// file: internals/features/system/holidays/dto/national_holiday_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/academics/schedules/model"
)

/* =========================================================
   Helpers
   ========================================================= */

/* =========================================================
   1) REQUESTS
   ========================================================= */

// ---------- CREATE ----------
type NationalHolidayCreateRequest struct {
	NationalHolidaySlug *string `json:"national_holiday_slug" validate:"omitempty,max=160"`

	NationalHolidayStartDate string `json:"national_holiday_start_date" validate:"required,datetime=2006-01-02"`
	NationalHolidayEndDate   string `json:"national_holiday_end_date"   validate:"required,datetime=2006-01-02"`

	NationalHolidayTitle  string  `json:"national_holiday_title"  validate:"required,max=200"`
	NationalHolidayReason *string `json:"national_holiday_reason" validate:"omitempty"`

	NationalHolidayIsActive          *bool `json:"national_holiday_is_active"          validate:"omitempty"`
	NationalHolidayIsRecurringYearly *bool `json:"national_holiday_is_recurring_yearly" validate:"omitempty"`
}

var (
	ErrInvalidStartDate = errors.New("invalid start_date (use YYYY-MM-DD)")
	ErrInvalidEndDate   = errors.New("invalid end_date (use YYYY-MM-DD)")
	ErrEndBeforeStart   = errors.New("end_date must be >= start_date")
)

func (r NationalHolidayCreateRequest) ToModel() (model.NationalHolidayModel, error) {
	start, ok := parseDateYYYYMMDD(r.NationalHolidayStartDate)
	if !ok {
		return model.NationalHolidayModel{}, ErrInvalidStartDate
	}
	end, ok := parseDateYYYYMMDD(r.NationalHolidayEndDate)
	if !ok {
		return model.NationalHolidayModel{}, ErrInvalidEndDate
	}
	if end.Before(start) {
		return model.NationalHolidayModel{}, ErrEndBeforeStart
	}

	isActive := true
	if r.NationalHolidayIsActive != nil {
		isActive = *r.NationalHolidayIsActive
	}
	isRecurring := false
	if r.NationalHolidayIsRecurringYearly != nil {
		isRecurring = *r.NationalHolidayIsRecurringYearly
	}

	return model.NationalHolidayModel{
		NationalHolidaySlug:              trimPtr(r.NationalHolidaySlug),
		NationalHolidayStartDate:         start,
		NationalHolidayEndDate:           end,
		NationalHolidayTitle:             strings.TrimSpace(r.NationalHolidayTitle),
		NationalHolidayReason:            trimPtr(r.NationalHolidayReason),
		NationalHolidayIsActive:          isActive,
		NationalHolidayIsRecurringYearly: isRecurring,
	}, nil
}

// ---------- UPDATE / PATCH ----------
type NationalHolidayUpdateRequest struct {
	NationalHolidaySlug *string `json:"national_holiday_slug" validate:"omitempty,max=160"`

	NationalHolidayStartDate *string `json:"national_holiday_start_date" validate:"omitempty,datetime=2006-01-02"`
	NationalHolidayEndDate   *string `json:"national_holiday_end_date"   validate:"omitempty,datetime=2006-01-02"`

	NationalHolidayTitle  *string `json:"national_holiday_title"  validate:"omitempty,max=200"`
	NationalHolidayReason *string `json:"national_holiday_reason" validate:"omitempty"`

	NationalHolidayIsActive          *bool `json:"national_holiday_is_active"           validate:"omitempty"`
	NationalHolidayIsRecurringYearly *bool `json:"national_holiday_is_recurring_yearly" validate:"omitempty"`
}

// Apply ke model existing (controller: ambil existing → req.Apply(&existing) → Save)
func (r NationalHolidayUpdateRequest) Apply(m *model.NationalHolidayModel) error {
	if r.NationalHolidaySlug != nil {
		m.NationalHolidaySlug = trimPtr(r.NationalHolidaySlug)
	}
	// tanggal: validasi per kolom, lalu cek konsistensi rentang
	var newStart = m.NationalHolidayStartDate
	var newEnd = m.NationalHolidayEndDate

	if r.NationalHolidayStartDate != nil {
		if t, ok := parseDateYYYYMMDD(*r.NationalHolidayStartDate); ok {
			newStart = t
		} else {
			return ErrInvalidStartDate
		}
	}
	if r.NationalHolidayEndDate != nil {
		if t, ok := parseDateYYYYMMDD(*r.NationalHolidayEndDate); ok {
			newEnd = t
		} else {
			return ErrInvalidEndDate
		}
	}
	if newEnd.Before(newStart) {
		return ErrEndBeforeStart
	}
	m.NationalHolidayStartDate = newStart
	m.NationalHolidayEndDate = newEnd

	if r.NationalHolidayTitle != nil {
		title := strings.TrimSpace(*r.NationalHolidayTitle)
		if title != "" {
			m.NationalHolidayTitle = title
		} else {
			// Boleh kosongkan? Umumnya tidak—abaikan jika kosong supaya tidak melanggar NOT NULL
		}
	}
	if r.NationalHolidayReason != nil {
		m.NationalHolidayReason = trimPtr(r.NationalHolidayReason)
	}
	if r.NationalHolidayIsActive != nil {
		m.NationalHolidayIsActive = *r.NationalHolidayIsActive
	}
	if r.NationalHolidayIsRecurringYearly != nil {
		m.NationalHolidayIsRecurringYearly = *r.NationalHolidayIsRecurringYearly
	}
	return nil
}

/* =========================================================
   2) LIST QUERY (index)
   ========================================================= */

type NationalHolidayListQuery struct {
	Limit       *int    `query:"limit"        validate:"omitempty,min=1,max=200"`
	Offset      *int    `query:"offset"       validate:"omitempty,min=0"`
	Q           *string `query:"q"            validate:"omitempty,max=160"` // cari di slug/title ringan (di DB implementasikan sendiri)
	IsActive    *bool   `query:"is_active"    validate:"omitempty"`
	IsRecurring *bool   `query:"is_recurring" validate:"omitempty"`
	WithDeleted *bool   `query:"with_deleted" validate:"omitempty"`

	// Filter tanggal (opsional): overlap dengan rentang libur
	// Implementasi: (end >= date_from AND start <= date_to)
	DateFrom *string `query:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo   *string `query:"date_to"   validate:"omitempty,datetime=2006-01-02"`

	// Sort:
	//   start_date_asc|start_date_desc|end_date_asc|end_date_desc|
	//   created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
	Sort *string `query:"sort" validate:"omitempty,oneof=start_date_asc start_date_desc end_date_asc end_date_desc created_at_asc created_at_desc updated_at_asc updated_at_desc"`
}

/* =========================================================
   3) RESPONSES
   ========================================================= */

type NationalHolidayResponse struct {
	NationalHolidayID uuid.UUID `json:"national_holiday_id"`

	NationalHolidaySlug *string `json:"national_holiday_slug,omitempty"`

	NationalHolidayStartDate time.Time `json:"national_holiday_start_date"`
	NationalHolidayEndDate   time.Time `json:"national_holiday_end_date"`

	NationalHolidayTitle  string  `json:"national_holiday_title"`
	NationalHolidayReason *string `json:"national_holiday_reason,omitempty"`

	NationalHolidayIsActive          bool `json:"national_holiday_is_active"`
	NationalHolidayIsRecurringYearly bool `json:"national_holiday_is_recurring_yearly"`

	NationalHolidayCreatedAt time.Time  `json:"national_holiday_created_at"`
	NationalHolidayUpdatedAt *time.Time `json:"national_holiday_updated_at,omitempty"`
	NationalHolidayDeletedAt *time.Time `json:"national_holiday_deleted_at,omitempty"`
}

type NationalHolidayListResponse struct {
	Items      []NationalHolidayResponse `json:"items"`
	Pagination Pagination                `json:"pagination"`
}

/* =========================================================
   4) MAPPERS
   ========================================================= */

func FromModelNationHoliday(m model.NationalHolidayModel) NationalHolidayResponse {
	var deletedAt *time.Time
	if m.NationalHolidayDeletedAt.Valid {
		d := m.NationalHolidayDeletedAt.Time
		deletedAt = &d
	}
	return NationalHolidayResponse{
		NationalHolidayID:                m.NationalHolidayID,
		NationalHolidaySlug:              m.NationalHolidaySlug,
		NationalHolidayStartDate:         m.NationalHolidayStartDate,
		NationalHolidayEndDate:           m.NationalHolidayEndDate,
		NationalHolidayTitle:             m.NationalHolidayTitle,
		NationalHolidayReason:            m.NationalHolidayReason,
		NationalHolidayIsActive:          m.NationalHolidayIsActive,
		NationalHolidayIsRecurringYearly: m.NationalHolidayIsRecurringYearly,
		NationalHolidayCreatedAt:         m.NationalHolidayCreatedAt,
		NationalHolidayUpdatedAt:         timePtrOrNil(m.NationalHolidayUpdatedAt),
		NationalHolidayDeletedAt:         deletedAt,
	}
}

func FromModelNationHolidays(list []model.NationalHolidayModel) []NationalHolidayResponse {
	out := make([]NationalHolidayResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelNationHoliday(list[i]))
	}
	return out
}
