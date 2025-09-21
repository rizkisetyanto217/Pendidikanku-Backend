// internals/features/lembaga/class_schedules/dto/class_schedule_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/sessions/schedules/model"
)

/* =========================================================
   Helpers
   ========================================================= */

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func parseDateYYYYMMDD(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func timePtrOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

/* =========================================================
   1) REQUESTS
   ========================================================= */

// Create: masjid_id dipaksa dari controller (parameter ToModel)
type CreateClassScheduleRequest struct {
	// optional slug
	ClassSchedulesSlug *string `json:"class_schedules_slug" validate:"omitempty,max=160"`

	// rentang wajib (YYYY-MM-DD)
	ClassSchedulesStartDate string `json:"class_schedules_start_date" validate:"required,datetime=2006-01-02"`
	ClassSchedulesEndDate   string `json:"class_schedules_end_date"   validate:"required,datetime=2006-01-02"`

	// status & aktif
	ClassSchedulesStatus   *string `json:"class_schedules_status"   validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	ClassSchedulesIsActive *bool   `json:"class_schedules_is_active" validate:"omitempty"`
}

func (r CreateClassScheduleRequest) ToModel(masjidID uuid.UUID) model.ClassScheduleModel {
	start, _ := parseDateYYYYMMDD(r.ClassSchedulesStartDate)
	end, _ := parseDateYYYYMMDD(r.ClassSchedulesEndDate)

	status := model.SessionScheduled
	if r.ClassSchedulesStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassSchedulesStatus)) {
		case "ongoing":
			status = model.SessionOngoing
		case "completed":
			status = model.SessionCompleted
		case "canceled":
			status = model.SessionCanceled
		default:
			status = model.SessionScheduled
		}
	}

	isActive := true
	if r.ClassSchedulesIsActive != nil {
		isActive = *r.ClassSchedulesIsActive
	}

	return model.ClassScheduleModel{
		// PK by DB
		ClassSchedulesMasjidID:  masjidID,
		ClassSchedulesSlug:      trimPtr(r.ClassSchedulesSlug),
		ClassSchedulesStartDate: start,
		ClassSchedulesEndDate:   end,
		ClassSchedulesStatus:    model.SessionStatusEnum(status),
		ClassSchedulesIsActive:  isActive,
	}
}

// Update (partial)
type UpdateClassScheduleRequest struct {
	ClassSchedulesSlug      *string `json:"class_schedules_slug"       validate:"omitempty,max=160"`
	ClassSchedulesStartDate *string `json:"class_schedules_start_date" validate:"omitempty,datetime=2006-01-02"`
	ClassSchedulesEndDate   *string `json:"class_schedules_end_date"   validate:"omitempty,datetime=2006-01-02"`
	ClassSchedulesStatus    *string `json:"class_schedules_status"     validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	ClassSchedulesIsActive  *bool   `json:"class_schedules_is_active"  validate:"omitempty"`
}

func (r UpdateClassScheduleRequest) Apply(m *model.ClassScheduleModel) {
	if r.ClassSchedulesSlug != nil {
		m.ClassSchedulesSlug = trimPtr(r.ClassSchedulesSlug)
	}
	if r.ClassSchedulesStartDate != nil {
		if t, ok := parseDateYYYYMMDD(*r.ClassSchedulesStartDate); ok {
			m.ClassSchedulesStartDate = t
		}
	}
	if r.ClassSchedulesEndDate != nil {
		if t, ok := parseDateYYYYMMDD(*r.ClassSchedulesEndDate); ok {
			m.ClassSchedulesEndDate = t
		}
	}
	if r.ClassSchedulesStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassSchedulesStatus)) {
		case "scheduled":
			m.ClassSchedulesStatus = model.SessionScheduled
		case "ongoing":
			m.ClassSchedulesStatus = model.SessionOngoing
		case "completed":
			m.ClassSchedulesStatus = model.SessionCompleted
		case "canceled":
			m.ClassSchedulesStatus = model.SessionCanceled
		}
	}
	if r.ClassSchedulesIsActive != nil {
		m.ClassSchedulesIsActive = *r.ClassSchedulesIsActive
	}
}

/* =========================================================
   2) LIST QUERY
   ========================================================= */

type ListClassScheduleQuery struct {
	Limit       *int    `query:"limit"        validate:"omitempty,min=1,max=200"`
	Offset      *int    `query:"offset"       validate:"omitempty,min=0"`
	Status      *string `query:"status"       validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	IsActive    *bool   `query:"is_active"    validate:"omitempty"`
	WithDeleted *bool   `query:"with_deleted" validate:"omitempty"`

	DateFrom *string `query:"date_from" validate:"omitempty,datetime=2006-01-02"`
	DateTo   *string `query:"date_to"   validate:"omitempty,datetime=2006-01-02"`

	// search ringan (slug)
	Q *string `query:"q" validate:"omitempty,max=100"`

	// sort: default created_at_desc
	// pilihan: start_date_asc|start_date_desc|end_date_asc|end_date_desc|created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
	Sort *string `query:"sort" validate:"omitempty,oneof=start_date_asc start_date_desc end_date_asc end_date_desc created_at_asc created_at_desc updated_at_asc updated_at_desc"`
}

/* =========================================================
   3) RESPONSES
   ========================================================= */

type ClassScheduleResponse struct {
	ClassScheduleID        uuid.UUID `json:"class_schedule_id"`
	ClassSchedulesMasjidID uuid.UUID `json:"class_schedules_masjid_id"`

	ClassSchedulesSlug      *string   `json:"class_schedules_slug,omitempty"`
	ClassSchedulesStartDate time.Time `json:"class_schedules_start_date"`
	ClassSchedulesEndDate   time.Time `json:"class_schedules_end_date"`
	ClassSchedulesStatus    string    `json:"class_schedules_status"`
	ClassSchedulesIsActive  bool      `json:"class_schedules_is_active"`

	ClassSchedulesCreatedAt time.Time  `json:"class_schedules_created_at"`
	ClassSchedulesUpdatedAt *time.Time `json:"class_schedules_updated_at,omitempty"`
	ClassSchedulesDeletedAt *time.Time `json:"class_schedules_deleted_at,omitempty"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type ClassScheduleListResponse struct {
	Items      []ClassScheduleResponse `json:"items"`
	Pagination Pagination              `json:"pagination"`
}

/* =========================================================
   4) MAPPERS
   ========================================================= */

func FromModel(m model.ClassScheduleModel) ClassScheduleResponse {
	var deletedAt *time.Time
	if m.ClassSchedulesDeletedAt.Valid {
		d := m.ClassSchedulesDeletedAt.Time
		deletedAt = &d
	}

	return ClassScheduleResponse{
		ClassScheduleID:        m.ClassScheduleID,
		ClassSchedulesMasjidID: m.ClassSchedulesMasjidID,

		ClassSchedulesSlug:      m.ClassSchedulesSlug,
		ClassSchedulesStartDate: m.ClassSchedulesStartDate,
		ClassSchedulesEndDate:   m.ClassSchedulesEndDate,
		ClassSchedulesStatus:    string(m.ClassSchedulesStatus),
		ClassSchedulesIsActive:  m.ClassSchedulesIsActive,

		ClassSchedulesCreatedAt: m.ClassSchedulesCreatedAt,
		ClassSchedulesUpdatedAt: timePtrOrNil(m.ClassSchedulesUpdatedAt),
		ClassSchedulesDeletedAt: deletedAt,
	}
}

func FromModels(list []model.ClassScheduleModel) []ClassScheduleResponse {
	out := make([]ClassScheduleResponse, 0, len(list))
	for _, m := range list {
		out = append(out, FromModel(m))
	}
	return out
}
