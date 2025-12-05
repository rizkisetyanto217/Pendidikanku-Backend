// file: internals/features/school/holidays/dto/school_holiday_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	m "madinahsalam_backend/internals/features/school/class_others/class_schedules/model"
)

/* =========================================================
   Helpers
   ========================================================= */

/* =========================================================
   Patch types (tri-state)
   - Patch[T]           : not-set | set(value)
   - PatchNullable[T]   : not-set | set(null) | set(value)
   ========================================================= */

type Patch[T any] struct {
	Set   bool
	Value T
}

func (p *Patch[T]) UnmarshalJSON(b []byte) error {
	// Any presence in JSON means Set=true (even if zero value)
	p.Set = true
	return json.Unmarshal(b, &p.Value)
}

func (p Patch[T]) IsSet() bool { return p.Set }

type PatchNullable[T any] struct {
	Set   bool // field key present?
	Valid bool // true => has Value, false => explicit null
	Value T
}

func (p *PatchNullable[T]) UnmarshalJSON(b []byte) error {
	p.Set = true
	if string(b) == "null" {
		p.Valid = false
		return nil
	}
	p.Valid = true
	return json.Unmarshal(b, &p.Value)
}

func (p PatchNullable[T]) IsSet() bool { return p.Set }

/* =========================================================
   1) REQUESTS
   ========================================================= */

// Create
type CreateSchoolHolidayRequest struct {
	SchoolHolidaySlug *string `json:"school_holiday_slug" validate:"omitempty,max=160"`

	// Dates in "YYYY-MM-DD"
	SchoolHolidayStartDate string `json:"school_holiday_start_date" validate:"required,datetime=2006-01-02"`
	SchoolHolidayEndDate   string `json:"school_holiday_end_date"   validate:"required,datetime=2006-01-02"`

	SchoolHolidayTitle  string  `json:"school_holiday_title"  validate:"required,max=200"`
	SchoolHolidayReason *string `json:"school_holiday_reason" validate:"omitempty,max=10000"`

	SchoolHolidayIsActive          *bool `json:"school_holiday_is_active"`           // default true (db)
	SchoolHolidayIsRecurringYearly *bool `json:"school_holiday_is_recurring_yearly"` // default false (db)
}

func (r *CreateSchoolHolidayRequest) ToModel(schoolID uuid.UUID) (*m.SchoolHoliday, error) {
	start, ok := parseDateYYYYMMDD(r.SchoolHolidayStartDate)
	if !ok {
		return nil, errors.New("invalid school_holiday_start_date (expected YYYY-MM-DD)")
	}
	end, ok := parseDateYYYYMMDD(r.SchoolHolidayEndDate)
	if !ok {
		return nil, errors.New("invalid school_holiday_end_date (expected YYYY-MM-DD)")
	}
	if end.Before(start) {
		return nil, errors.New("school_holiday_end_date must be >= school_holiday_start_date")
	}

	h := &m.SchoolHoliday{
		SchoolHolidaySchoolID: schoolID,

		SchoolHolidaySlug: trimPtr(r.SchoolHolidaySlug),

		SchoolHolidayStartDate: start,
		SchoolHolidayEndDate:   end,

		SchoolHolidayTitle:  strings.TrimSpace(r.SchoolHolidayTitle),
		SchoolHolidayReason: trimPtr(r.SchoolHolidayReason),
	}

	if r.SchoolHolidayIsActive != nil {
		h.SchoolHolidayIsActive = *r.SchoolHolidayIsActive
	} else {
		h.SchoolHolidayIsActive = true
	}
	if r.SchoolHolidayIsRecurringYearly != nil {
		h.SchoolHolidayIsRecurringYearly = *r.SchoolHolidayIsRecurringYearly
	} else {
		h.SchoolHolidayIsRecurringYearly = false
	}

	return h, nil
}

// Patch (partial update)
// Catatan:
//   - Untuk kolom nullable (slug, reason, deleted_at) gunakan PatchNullable
//     sehingga bisa membedakan set null vs kosong vs tidak diubah.
type PatchSchoolHolidayRequest struct {
	SchoolHolidaySlug PatchNullable[string] `json:"school_holiday_slug"`

	// Dates in "YYYY-MM-DD" (bila hadir → wajib valid)
	SchoolHolidayStartDate Patch[string] `json:"school_holiday_start_date"`
	SchoolHolidayEndDate   Patch[string] `json:"school_holiday_end_date"`

	SchoolHolidayTitle             Patch[string]         `json:"school_holiday_title"`
	SchoolHolidayReason            PatchNullable[string] `json:"school_holiday_reason"`
	SchoolHolidayIsActive          Patch[bool]           `json:"school_holiday_is_active"`
	SchoolHolidayIsRecurringYearly Patch[bool]           `json:"school_holiday_is_recurring_yearly"`
}

// Apply changes to model (in-memory). Validasi ringan disertakan.
func (p *PatchSchoolHolidayRequest) Apply(h *m.SchoolHoliday) error {
	// slug (nullable)
	if p.SchoolHolidaySlug.IsSet() {
		if p.SchoolHolidaySlug.Valid {
			h.SchoolHolidaySlug = trimPtr(&p.SchoolHolidaySlug.Value)
		} else {
			// explicit null
			h.SchoolHolidaySlug = nil
		}
	}

	// title (non-null)
	if p.SchoolHolidayTitle.IsSet() {
		title := strings.TrimSpace(p.SchoolHolidayTitle.Value)
		if title == "" {
			return errors.New("school_holiday_title cannot be empty when set")
		}
		if len(title) > 200 {
			return errors.New("school_holiday_title max length is 200")
		}
		h.SchoolHolidayTitle = title
	}

	// reason (nullable)
	if p.SchoolHolidayReason.IsSet() {
		if p.SchoolHolidayReason.Valid {
			h.SchoolHolidayReason = trimPtr(&p.SchoolHolidayReason.Value)
		} else {
			h.SchoolHolidayReason = nil
		}
	}

	// dates
	var (
		newStart = h.SchoolHolidayStartDate
		newEnd   = h.SchoolHolidayEndDate
	)

	if p.SchoolHolidayStartDate.IsSet() {
		t, ok := parseDateYYYYMMDD(p.SchoolHolidayStartDate.Value)
		if !ok {
			return errors.New("invalid school_holiday_start_date (expected YYYY-MM-DD)")
		}
		newStart = t
	}
	if p.SchoolHolidayEndDate.IsSet() {
		t, ok := parseDateYYYYMMDD(p.SchoolHolidayEndDate.Value)
		if !ok {
			return errors.New("invalid school_holiday_end_date (expected YYYY-MM-DD)")
		}
		newEnd = t
	}
	if p.SchoolHolidayStartDate.IsSet() || p.SchoolHolidayEndDate.IsSet() {
		if newEnd.Before(newStart) {
			return errors.New("school_holiday_end_date must be >= school_holiday_start_date")
		}
		h.SchoolHolidayStartDate = newStart
		h.SchoolHolidayEndDate = newEnd
	}

	// booleans
	if p.SchoolHolidayIsActive.IsSet() {
		h.SchoolHolidayIsActive = p.SchoolHolidayIsActive.Value
	}
	if p.SchoolHolidayIsRecurringYearly.IsSet() {
		h.SchoolHolidayIsRecurringYearly = p.SchoolHolidayIsRecurringYearly.Value
	}

	return nil
}

/* =========================================================
   2) QUERY (list/filter)
   ========================================================= */

type ListSchoolHolidaysQuery struct {
	// Filter tanggal (opsional) — format YYYY-MM-DD
	DateFrom *string `query:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo   *string `query:"date_to"   validate:"omitempty,datetime=2006-01-02"`

	// Only active?
	OnlyActive *bool `query:"only_active" validate:"omitempty"`

	// Search by slug/title (server side: ILIKE %q%)
	Q *string `query:"q" validate:"omitempty,max=200"`

	// Pagination
	Limit  int `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int `query:"offset" validate:"omitempty,min=0"`
}

func (q *ListSchoolHolidaysQuery) Normalize() {
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

type SchoolHolidayResponse struct {
	SchoolHolidayID uuid.UUID `json:"school_holiday_id"`

	SchoolHolidaySchoolID uuid.UUID `json:"school_holiday_school_id"`

	SchoolHolidaySlug *string `json:"school_holiday_slug,omitempty"`

	SchoolHolidayStartDate string `json:"school_holiday_start_date"` // YYYY-MM-DD
	SchoolHolidayEndDate   string `json:"school_holiday_end_date"`   // YYYY-MM-DD

	SchoolHolidayTitle  string  `json:"school_holiday_title"`
	SchoolHolidayReason *string `json:"school_holiday_reason,omitempty"`

	SchoolHolidayIsActive          bool `json:"school_holiday_is_active"`
	SchoolHolidayIsRecurringYearly bool `json:"school_holiday_is_recurring_yearly"`

	SchoolHolidayCreatedAt time.Time  `json:"school_holiday_created_at"`
	SchoolHolidayUpdatedAt time.Time  `json:"school_holiday_updated_at"`
	SchoolHolidayDeletedAt *time.Time `json:"school_holiday_deleted_at,omitempty"`
}

func dateYMD(t time.Time) string { return t.Format("2006-01-02") }

func FromModelSchoolHoliday(h *m.SchoolHoliday) *SchoolHolidayResponse {
	if h == nil {
		return nil
	}
	return &SchoolHolidayResponse{
		SchoolHolidayID:                h.SchoolHolidayID,
		SchoolHolidaySchoolID:          h.SchoolHolidaySchoolID,
		SchoolHolidaySlug:              h.SchoolHolidaySlug,
		SchoolHolidayStartDate:         dateYMD(h.SchoolHolidayStartDate),
		SchoolHolidayEndDate:           dateYMD(h.SchoolHolidayEndDate),
		SchoolHolidayTitle:             h.SchoolHolidayTitle,
		SchoolHolidayReason:            h.SchoolHolidayReason,
		SchoolHolidayIsActive:          h.SchoolHolidayIsActive,
		SchoolHolidayIsRecurringYearly: h.SchoolHolidayIsRecurringYearly,
		SchoolHolidayCreatedAt:         h.SchoolHolidayCreatedAt,
		SchoolHolidayUpdatedAt:         h.SchoolHolidayUpdatedAt,
		SchoolHolidayDeletedAt:         h.SchoolHolidayDeletedAt,
	}
}

type SchoolHolidayListResponse struct {
	Data       []*SchoolHolidayResponse `json:"data"`
	Pagination struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	} `json:"pagination"`
}

/* =========================================================
   4) Small convenience for logging/errs
   ========================================================= */

func (r *CreateSchoolHolidayRequest) String() string {
	return fmt.Sprintf("CreateSchoolHolidayRequest{slug=%v, start=%s, end=%s, title=%q, is_active=%v, yearly=%v}",
		r.SchoolHolidaySlug, r.SchoolHolidayStartDate, r.SchoolHolidayEndDate, r.SchoolHolidayTitle,
		boolOrNil(r.SchoolHolidayIsActive), boolOrNil(r.SchoolHolidayIsRecurringYearly))
}

func boolOrNil(p *bool) interface{} {
	if p == nil {
		return nil
	}
	return *p
}
