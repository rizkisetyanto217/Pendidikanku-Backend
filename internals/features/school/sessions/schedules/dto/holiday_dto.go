// internals/features/school/sessions/holidays/dto/holiday_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/sessions/schedules/model"
)

/* ===================== Helpers ===================== */

func parseYMD(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

/* ===================== REQUESTS ===================== */

type CreateHolidayRequest struct {
	HolidaySlug              *string `json:"holiday_slug"              validate:"omitempty,max=160"`
	HolidayStartDate         string  `json:"holiday_start_date"         validate:"required,datetime=2006-01-02"`
	HolidayEndDate           string  `json:"holiday_end_date"           validate:"required,datetime=2006-01-02"`
	HolidayTitle             string  `json:"holiday_title"              validate:"required,max=200"`
	HolidayReason            *string `json:"holiday_reason"             validate:"omitempty"`
	HolidayIsActive          *bool   `json:"holiday_is_active"          validate:"omitempty"`
	HolidayIsRecurringYearly *bool   `json:"holiday_is_recurring_yearly" validate:"omitempty"`
}

func (r CreateHolidayRequest) ToModel(masjidID uuid.UUID) model.HolidayModel {
	start, _ := parseYMD(r.HolidayStartDate)
	end, _ := parseYMD(r.HolidayEndDate)

	active := true
	if r.HolidayIsActive != nil {
		active = *r.HolidayIsActive
	}
	rec := false
	if r.HolidayIsRecurringYearly != nil {
		rec = *r.HolidayIsRecurringYearly
	}

	return model.HolidayModel{
		HolidayMasjidID:          masjidID,
		HolidaySlug:              trimPtr(r.HolidaySlug),
		HolidayStartDate:         start,
		HolidayEndDate:           end,
		HolidayTitle:             strings.TrimSpace(r.HolidayTitle),
		HolidayReason:            trimPtr(r.HolidayReason),
		HolidayIsActive:          active,
		HolidayIsRecurringYearly: rec,
	}
}

type UpdateHolidayRequest struct {
	HolidaySlug              *string `json:"holiday_slug"              validate:"omitempty,max=160"`
	HolidayStartDate         *string `json:"holiday_start_date"         validate:"omitempty,datetime=2006-01-02"`
	HolidayEndDate           *string `json:"holiday_end_date"           validate:"omitempty,datetime=2006-01-02"`
	HolidayTitle             *string `json:"holiday_title"              validate:"omitempty,max=200"`
	HolidayReason            *string `json:"holiday_reason"             validate:"omitempty"`
	HolidayIsActive          *bool   `json:"holiday_is_active"          validate:"omitempty"`
	HolidayIsRecurringYearly *bool   `json:"holiday_is_recurring_yearly" validate:"omitempty"`
}

func (r UpdateHolidayRequest) Apply(m *model.HolidayModel) {
	if r.HolidaySlug != nil {
		m.HolidaySlug = trimPtr(r.HolidaySlug)
	}
	if r.HolidayStartDate != nil {
		if t, ok := parseYMD(*r.HolidayStartDate); ok {
			m.HolidayStartDate = t
		}
	}
	if r.HolidayEndDate != nil {
		if t, ok := parseYMD(*r.HolidayEndDate); ok {
			m.HolidayEndDate = t
		}
	}
	if r.HolidayTitle != nil {
		m.HolidayTitle = strings.TrimSpace(*r.HolidayTitle)
	}
	if r.HolidayReason != nil {
		m.HolidayReason = trimPtr(r.HolidayReason)
	}
	if r.HolidayIsActive != nil {
		m.HolidayIsActive = *r.HolidayIsActive
	}
	if r.HolidayIsRecurringYearly != nil {
		m.HolidayIsRecurringYearly = *r.HolidayIsRecurringYearly
	}
}

/* ===================== LIST QUERY ===================== */

type ListHolidayQuery struct {
	Limit         *int    `query:"limit"         validate:"omitempty,min=1,max=200"`
	Offset        *int    `query:"offset"        validate:"omitempty,min=0"`
	IsActive      *bool   `query:"is_active"     validate:"omitempty"`
	RecurringOnly *bool   `query:"recurring_only" validate:"omitempty"`
	DateFrom      *string `query:"date_from"     validate:"omitempty,datetime=2006-01-02"`
	DateTo        *string `query:"date_to"       validate:"omitempty,datetime=2006-01-02"`
	Q             *string `query:"q"             validate:"omitempty,max=200"` // cari di slug/title/reason (opsional di controller)

	// sort: start_date_asc|start_date_desc|end_date_asc|end_date_desc|created_at_asc|created_at_desc|updated_at_asc|updated_at_desc|title_asc|title_desc
	Sort *string `query:"sort" validate:"omitempty,oneof=start_date_asc start_date_desc end_date_asc end_date_desc created_at_asc created_at_desc updated_at_asc updated_at_desc title_asc title_desc"`
}

/* ===================== RESPONSES ===================== */

type HolidayResponse struct {
	HolidayID       uuid.UUID `json:"holiday_id"`
	HolidayMasjidID uuid.UUID `json:"holiday_masjid_id"`

	HolidaySlug      *string   `json:"holiday_slug,omitempty"`
	HolidayStartDate time.Time `json:"holiday_start_date"`
	HolidayEndDate   time.Time `json:"holiday_end_date"`

	HolidayTitle             string  `json:"holiday_title"`
	HolidayReason            *string `json:"holiday_reason,omitempty"`
	HolidayIsActive          bool    `json:"holiday_is_active"`
	HolidayIsRecurringYearly bool    `json:"holiday_is_recurring_yearly"`

	HolidayCreatedAt time.Time  `json:"holiday_created_at"`
	HolidayUpdatedAt *time.Time `json:"holiday_updated_at,omitempty"`
	HolidayDeletedAt *time.Time `json:"holiday_deleted_at,omitempty"`
}

type PaginationHoliday struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type HolidayListResponse struct {
	Items      []HolidayResponse `json:"items"`
	Pagination Pagination        `json:"pagination"`
}

/* ===================== MAPPERS ===================== */

func FromModelHoliday(m model.HolidayModel) HolidayResponse {
	var deletedAt *time.Time
	if m.HolidayDeletedAt.Valid {
		d := m.HolidayDeletedAt.Time
		deletedAt = &d
	}

	return HolidayResponse{
		HolidayID:                m.HolidayID,
		HolidayMasjidID:          m.HolidayMasjidID,
		HolidaySlug:              m.HolidaySlug,
		HolidayStartDate:         m.HolidayStartDate,
		HolidayEndDate:           m.HolidayEndDate,
		HolidayTitle:             m.HolidayTitle,
		HolidayReason:            m.HolidayReason,
		HolidayIsActive:          m.HolidayIsActive,
		HolidayIsRecurringYearly: m.HolidayIsRecurringYearly,

		HolidayCreatedAt: m.HolidayCreatedAt,
		HolidayUpdatedAt: timePtrOrNil(m.HolidayUpdatedAt),
		HolidayDeletedAt: deletedAt,
	}
}

func FromModelsHoliday(list []model.HolidayModel) []HolidayResponse {
	out := make([]HolidayResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelHoliday(list[i]))
	}
	return out
}
