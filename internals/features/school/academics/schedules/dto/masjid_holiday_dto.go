// file: internals/features/school/holidays/dto/school_holiday_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/school/academics/schedules/model"
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
type CreateMasjidHolidayRequest struct {
	MasjidHolidaySlug *string `json:"school_holiday_slug" validate:"omitempty,max=160"`

	// Dates in "YYYY-MM-DD"
	MasjidHolidayStartDate string `json:"school_holiday_start_date" validate:"required,datetime=2006-01-02"`
	MasjidHolidayEndDate   string `json:"school_holiday_end_date"   validate:"required,datetime=2006-01-02"`

	MasjidHolidayTitle  string  `json:"school_holiday_title"  validate:"required,max=200"`
	MasjidHolidayReason *string `json:"school_holiday_reason" validate:"omitempty,max=10000"`

	MasjidHolidayIsActive          *bool `json:"school_holiday_is_active"`           // default true (db)
	MasjidHolidayIsRecurringYearly *bool `json:"school_holiday_is_recurring_yearly"` // default false (db)
}

func (r *CreateMasjidHolidayRequest) ToModel(masjidID uuid.UUID) (*m.MasjidHoliday, error) {
	start, ok := parseDateYYYYMMDD(r.MasjidHolidayStartDate)
	if !ok {
		return nil, errors.New("invalid school_holiday_start_date (expected YYYY-MM-DD)")
	}
	end, ok := parseDateYYYYMMDD(r.MasjidHolidayEndDate)
	if !ok {
		return nil, errors.New("invalid school_holiday_end_date (expected YYYY-MM-DD)")
	}
	if end.Before(start) {
		return nil, errors.New("school_holiday_end_date must be >= school_holiday_start_date")
	}

	h := &m.MasjidHoliday{
		MasjidHolidayMasjidID: masjidID,

		MasjidHolidaySlug: trimPtr(r.MasjidHolidaySlug),

		MasjidHolidayStartDate: start,
		MasjidHolidayEndDate:   end,

		MasjidHolidayTitle:  strings.TrimSpace(r.MasjidHolidayTitle),
		MasjidHolidayReason: trimPtr(r.MasjidHolidayReason),
	}

	if r.MasjidHolidayIsActive != nil {
		h.MasjidHolidayIsActive = *r.MasjidHolidayIsActive
	} else {
		h.MasjidHolidayIsActive = true
	}
	if r.MasjidHolidayIsRecurringYearly != nil {
		h.MasjidHolidayIsRecurringYearly = *r.MasjidHolidayIsRecurringYearly
	} else {
		h.MasjidHolidayIsRecurringYearly = false
	}

	return h, nil
}

// Patch (partial update)
// Catatan:
//   - Untuk kolom nullable (slug, reason, deleted_at) gunakan PatchNullable
//     sehingga bisa membedakan set null vs kosong vs tidak diubah.
type PatchMasjidHolidayRequest struct {
	MasjidHolidaySlug PatchNullable[string] `json:"school_holiday_slug"`

	// Dates in "YYYY-MM-DD" (bila hadir → wajib valid)
	MasjidHolidayStartDate Patch[string] `json:"school_holiday_start_date"`
	MasjidHolidayEndDate   Patch[string] `json:"school_holiday_end_date"`

	MasjidHolidayTitle             Patch[string]         `json:"school_holiday_title"`
	MasjidHolidayReason            PatchNullable[string] `json:"school_holiday_reason"`
	MasjidHolidayIsActive          Patch[bool]           `json:"school_holiday_is_active"`
	MasjidHolidayIsRecurringYearly Patch[bool]           `json:"school_holiday_is_recurring_yearly"`
}

// Apply changes to model (in-memory). Validasi ringan disertakan.
func (p *PatchMasjidHolidayRequest) Apply(h *m.MasjidHoliday) error {
	// slug (nullable)
	if p.MasjidHolidaySlug.IsSet() {
		if p.MasjidHolidaySlug.Valid {
			h.MasjidHolidaySlug = trimPtr(&p.MasjidHolidaySlug.Value)
		} else {
			// explicit null
			h.MasjidHolidaySlug = nil
		}
	}

	// title (non-null)
	if p.MasjidHolidayTitle.IsSet() {
		title := strings.TrimSpace(p.MasjidHolidayTitle.Value)
		if title == "" {
			return errors.New("school_holiday_title cannot be empty when set")
		}
		if len(title) > 200 {
			return errors.New("school_holiday_title max length is 200")
		}
		h.MasjidHolidayTitle = title
	}

	// reason (nullable)
	if p.MasjidHolidayReason.IsSet() {
		if p.MasjidHolidayReason.Valid {
			h.MasjidHolidayReason = trimPtr(&p.MasjidHolidayReason.Value)
		} else {
			h.MasjidHolidayReason = nil
		}
	}

	// dates
	var (
		newStart = h.MasjidHolidayStartDate
		newEnd   = h.MasjidHolidayEndDate
	)

	if p.MasjidHolidayStartDate.IsSet() {
		t, ok := parseDateYYYYMMDD(p.MasjidHolidayStartDate.Value)
		if !ok {
			return errors.New("invalid school_holiday_start_date (expected YYYY-MM-DD)")
		}
		newStart = t
	}
	if p.MasjidHolidayEndDate.IsSet() {
		t, ok := parseDateYYYYMMDD(p.MasjidHolidayEndDate.Value)
		if !ok {
			return errors.New("invalid school_holiday_end_date (expected YYYY-MM-DD)")
		}
		newEnd = t
	}
	if p.MasjidHolidayStartDate.IsSet() || p.MasjidHolidayEndDate.IsSet() {
		if newEnd.Before(newStart) {
			return errors.New("school_holiday_end_date must be >= school_holiday_start_date")
		}
		h.MasjidHolidayStartDate = newStart
		h.MasjidHolidayEndDate = newEnd
	}

	// booleans
	if p.MasjidHolidayIsActive.IsSet() {
		h.MasjidHolidayIsActive = p.MasjidHolidayIsActive.Value
	}
	if p.MasjidHolidayIsRecurringYearly.IsSet() {
		h.MasjidHolidayIsRecurringYearly = p.MasjidHolidayIsRecurringYearly.Value
	}

	return nil
}

/* =========================================================
   2) QUERY (list/filter)
   ========================================================= */

type ListMasjidHolidaysQuery struct {
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

func (q *ListMasjidHolidaysQuery) Normalize() {
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

type MasjidHolidayResponse struct {
	MasjidHolidayID uuid.UUID `json:"school_holiday_id"`

	MasjidHolidayMasjidID uuid.UUID `json:"school_holiday_masjid_id"`

	MasjidHolidaySlug *string `json:"school_holiday_slug,omitempty"`

	MasjidHolidayStartDate string `json:"school_holiday_start_date"` // YYYY-MM-DD
	MasjidHolidayEndDate   string `json:"school_holiday_end_date"`   // YYYY-MM-DD

	MasjidHolidayTitle  string  `json:"school_holiday_title"`
	MasjidHolidayReason *string `json:"school_holiday_reason,omitempty"`

	MasjidHolidayIsActive          bool `json:"school_holiday_is_active"`
	MasjidHolidayIsRecurringYearly bool `json:"school_holiday_is_recurring_yearly"`

	MasjidHolidayCreatedAt time.Time  `json:"school_holiday_created_at"`
	MasjidHolidayUpdatedAt time.Time  `json:"school_holiday_updated_at"`
	MasjidHolidayDeletedAt *time.Time `json:"school_holiday_deleted_at,omitempty"`
}

func dateYMD(t time.Time) string { return t.Format("2006-01-02") }

func FromModelMasjidHoliday(h *m.MasjidHoliday) *MasjidHolidayResponse {
	if h == nil {
		return nil
	}
	return &MasjidHolidayResponse{
		MasjidHolidayID:                h.MasjidHolidayID,
		MasjidHolidayMasjidID:          h.MasjidHolidayMasjidID,
		MasjidHolidaySlug:              h.MasjidHolidaySlug,
		MasjidHolidayStartDate:         dateYMD(h.MasjidHolidayStartDate),
		MasjidHolidayEndDate:           dateYMD(h.MasjidHolidayEndDate),
		MasjidHolidayTitle:             h.MasjidHolidayTitle,
		MasjidHolidayReason:            h.MasjidHolidayReason,
		MasjidHolidayIsActive:          h.MasjidHolidayIsActive,
		MasjidHolidayIsRecurringYearly: h.MasjidHolidayIsRecurringYearly,
		MasjidHolidayCreatedAt:         h.MasjidHolidayCreatedAt,
		MasjidHolidayUpdatedAt:         h.MasjidHolidayUpdatedAt,
		MasjidHolidayDeletedAt:         h.MasjidHolidayDeletedAt,
	}
}

type MasjidHolidayListResponse struct {
	Data       []*MasjidHolidayResponse `json:"data"`
	Pagination struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	} `json:"pagination"`
}

/* =========================================================
   4) Small convenience for logging/errs
   ========================================================= */

func (r *CreateMasjidHolidayRequest) String() string {
	return fmt.Sprintf("CreateMasjidHolidayRequest{slug=%v, start=%s, end=%s, title=%q, is_active=%v, yearly=%v}",
		r.MasjidHolidaySlug, r.MasjidHolidayStartDate, r.MasjidHolidayEndDate, r.MasjidHolidayTitle,
		boolOrNil(r.MasjidHolidayIsActive), boolOrNil(r.MasjidHolidayIsRecurringYearly))
}

func boolOrNil(p *bool) interface{} {
	if p == nil {
		return nil
	}
	return *p
}
