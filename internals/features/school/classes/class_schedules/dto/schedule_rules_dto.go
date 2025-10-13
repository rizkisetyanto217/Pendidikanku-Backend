package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	model "masjidku_backend/internals/features/school/classes/class_schedules/model"
)

/* =========================================================
   Helpers
   ========================================================= */



func parseTimeOfDay(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse("15:04", s); err == nil {
		return t, true
	}
	if t, err := time.Parse("15:04:05", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func weeksToPQ(in []int) pq.Int64Array {
	if len(in) == 0 {
		return nil
	}
	out := make(pq.Int64Array, len(in))
	for i, v := range in {
		out[i] = int64(v)
	}
	return out
}

func formatTOD(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04:05")
}

/* =========================================================
   1) REQUESTS
   ========================================================= */

type CreateClassScheduleRuleRequest struct {
	// wajib
	ClassScheduleRuleScheduleID uuid.UUID `json:"class_schedule_rule_schedule_id" validate:"required,uuid"`
	ClassScheduleRuleDayOfWeek  int       `json:"class_schedule_rule_day_of_week"  validate:"required,min=1,max=7"`
	ClassScheduleRuleStartTime  string    `json:"class_schedule_rule_start_time"   validate:"required"` // "HH:mm" / "HH:mm:ss"
	ClassScheduleRuleEndTime    string    `json:"class_schedule_rule_end_time"     validate:"required"` // "HH:mm" / "HH:mm:ss"

	// CSST (WAJIB, tenant-safe)
	ClassScheduleRuleCSSTID       uuid.UUID `json:"class_schedule_rule_csst_id"        validate:"required,uuid"`
	ClassScheduleRuleCSSTMasjidID uuid.UUID `json:"class_schedule_rule_csst_masjid_id" validate:"required,uuid"`

	// opsional (defaults: 1, 0, "all", nil, false)
	ClassScheduleRuleIntervalWeeks    *int    `json:"class_schedule_rule_interval_weeks"     validate:"omitempty,min=1"`
	ClassScheduleRuleStartOffsetWeeks *int    `json:"class_schedule_rule_start_offset_weeks" validate:"omitempty,min=0"`
	ClassScheduleRuleWeekParity       *string `json:"class_schedule_rule_week_parity"        validate:"omitempty,oneof=all odd even"`
	ClassScheduleRuleWeeksOfMonth     []int   `json:"class_schedule_rule_weeks_of_month"     validate:"omitempty,dive,min=1,max=5"`
	ClassScheduleRuleLastWeekOfMonth  *bool   `json:"class_schedule_rule_last_week_of_month" validate:"omitempty"`
}

// masjidID dipaksa dari controller
func (r CreateClassScheduleRuleRequest) ToModel(masjidID uuid.UUID) (model.ClassScheduleRuleModel, error) {
	st, ok := parseTimeOfDay(r.ClassScheduleRuleStartTime)
	if !ok {
		return model.ClassScheduleRuleModel{}, ErrInvalidStartTime
	}
	et, ok := parseTimeOfDay(r.ClassScheduleRuleEndTime)
	if !ok {
		return model.ClassScheduleRuleModel{}, ErrInvalidEndTime
	}

	interval := 1
	if r.ClassScheduleRuleIntervalWeeks != nil {
		interval = *r.ClassScheduleRuleIntervalWeeks
	}
	offset := 0
	if r.ClassScheduleRuleStartOffsetWeeks != nil {
		offset = *r.ClassScheduleRuleStartOffsetWeeks
	}
	parity := model.WeekParityAll
	if r.ClassScheduleRuleWeekParity != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassScheduleRuleWeekParity)) {
		case "odd":
			parity = model.WeekParityOdd
		case "even":
			parity = model.WeekParityEven
		default:
			parity = model.WeekParityAll
		}
	}
	lastWeek := false
	if r.ClassScheduleRuleLastWeekOfMonth != nil {
		lastWeek = *r.ClassScheduleRuleLastWeekOfMonth
	}

	return model.ClassScheduleRuleModel{
		ClassScheduleRuleMasjidID:   masjidID,
		ClassScheduleRuleScheduleID: r.ClassScheduleRuleScheduleID,

		ClassScheduleRuleDayOfWeek: r.ClassScheduleRuleDayOfWeek,
		ClassScheduleRuleStartTime: st,
		ClassScheduleRuleEndTime:   et,

		ClassScheduleRuleIntervalWeeks:    interval,
		ClassScheduleRuleStartOffsetWeeks: offset,
		ClassScheduleRuleWeekParity:       parity,
		ClassScheduleRuleWeeksOfMonth:     weeksToPQ(r.ClassScheduleRuleWeeksOfMonth),
		ClassScheduleRuleLastWeekOfMonth:  lastWeek,

		// CSST wajib
		ClassScheduleRuleCSSTID:       r.ClassScheduleRuleCSSTID,
		ClassScheduleRuleCSSTMasjidID: r.ClassScheduleRuleCSSTMasjidID,
	}, nil
}

type UpdateClassScheduleRuleRequest struct {
	ClassScheduleRuleDayOfWeek        *int    `json:"class_schedule_rule_day_of_week"        validate:"omitempty,min=1,max=7"`
	ClassScheduleRuleStartTime        *string `json:"class_schedule_rule_start_time"         validate:"omitempty"`
	ClassScheduleRuleEndTime          *string `json:"class_schedule_rule_end_time"           validate:"omitempty"`
	ClassScheduleRuleIntervalWeeks    *int    `json:"class_schedule_rule_interval_weeks"     validate:"omitempty,min=1"`
	ClassScheduleRuleStartOffsetWeeks *int    `json:"class_schedule_rule_start_offset_weeks" validate:"omitempty,min=0"`
	ClassScheduleRuleWeekParity       *string `json:"class_schedule_rule_week_parity"        validate:"omitempty,oneof=all odd even"`
	// pointer agar bisa bedakan "tidak diubah" (nil) vs "set ke []" (empty slice & non-nil)
	ClassScheduleRuleWeeksOfMonth    *[]int `json:"class_schedule_rule_weeks_of_month"     validate:"omitempty,dive,min=1,max=5"`
	ClassScheduleRuleLastWeekOfMonth *bool  `json:"class_schedule_rule_last_week_of_month" validate:"omitempty"`

	// ganti CSST (opsional; butuh guard tenant di service/DB)
	ClassScheduleRuleCSSTID       *uuid.UUID `json:"class_schedule_rule_csst_id"        validate:"omitempty,uuid"`
	ClassScheduleRuleCSSTMasjidID *uuid.UUID `json:"class_schedule_rule_csst_masjid_id" validate:"omitempty,uuid"`
}

func (r UpdateClassScheduleRuleRequest) Apply(m *model.ClassScheduleRuleModel) error {
	if r.ClassScheduleRuleDayOfWeek != nil {
		m.ClassScheduleRuleDayOfWeek = *r.ClassScheduleRuleDayOfWeek
	}
	if r.ClassScheduleRuleStartTime != nil {
		if t, ok := parseTimeOfDay(*r.ClassScheduleRuleStartTime); ok {
			m.ClassScheduleRuleStartTime = t
		} else {
			return ErrInvalidStartTime
		}
	}
	if r.ClassScheduleRuleEndTime != nil {
		if t, ok := parseTimeOfDay(*r.ClassScheduleRuleEndTime); ok {
			m.ClassScheduleRuleEndTime = t
		} else {
			return ErrInvalidEndTime
		}
	}
	if r.ClassScheduleRuleIntervalWeeks != nil {
		m.ClassScheduleRuleIntervalWeeks = *r.ClassScheduleRuleIntervalWeeks
	}
	if r.ClassScheduleRuleStartOffsetWeeks != nil {
		m.ClassScheduleRuleStartOffsetWeeks = *r.ClassScheduleRuleStartOffsetWeeks
	}
	if r.ClassScheduleRuleWeekParity != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassScheduleRuleWeekParity)) {
		case "odd":
			m.ClassScheduleRuleWeekParity = model.WeekParityOdd
		case "even":
			m.ClassScheduleRuleWeekParity = model.WeekParityEven
		case "all":
			m.ClassScheduleRuleWeekParity = model.WeekParityAll
		}
	}
	if r.ClassScheduleRuleWeeksOfMonth != nil {
		m.ClassScheduleRuleWeeksOfMonth = weeksToPQ(*r.ClassScheduleRuleWeeksOfMonth)
	}
	if r.ClassScheduleRuleLastWeekOfMonth != nil {
		m.ClassScheduleRuleLastWeekOfMonth = *r.ClassScheduleRuleLastWeekOfMonth
	}
	if r.ClassScheduleRuleCSSTID != nil {
		m.ClassScheduleRuleCSSTID = *r.ClassScheduleRuleCSSTID
	}
	if r.ClassScheduleRuleCSSTMasjidID != nil {
		m.ClassScheduleRuleCSSTMasjidID = *r.ClassScheduleRuleCSSTMasjidID
	}
	return nil
}

/* =========================================================
   2) LIST QUERY
   ========================================================= */

type ListClassScheduleRuleQuery struct {
	Limit      *int       `query:"limit"       validate:"omitempty,min=1,max=200"`
	Offset     *int       `query:"offset"      validate:"omitempty,min=0"`
	ScheduleID *uuid.UUID `query:"schedule_id" validate:"omitempty,uuid"`
	DayOfWeek  *int       `query:"dow"         validate:"omitempty,min=1,max=7"`
	WeekParity *string    `query:"parity"      validate:"omitempty,oneof=all odd even"`

	// Filter by generated columns (tanpa join)
	TeacherID      *uuid.UUID `query:"teacher_id"       validate:"omitempty,uuid"`
	SectionID      *uuid.UUID `query:"section_id"       validate:"omitempty,uuid"`
	ClassSubjectID *uuid.UUID `query:"class_subject_id" validate:"omitempty,uuid"`
	RoomID         *uuid.UUID `query:"room_id"          validate:"omitempty,uuid"`

	// sort_by: day_of_week|start_time|end_time|created_at|updated_at
	// order: asc|desc
	SortBy *string `query:"sort_by" validate:"omitempty,oneof=day_of_week start_time end_time created_at updated_at"`
	Order  *string `query:"order"   validate:"omitempty,oneof=asc desc"`
}

/* =========================================================
   3) RESPONSES
   ========================================================= */

type ClassScheduleRuleResponse struct {
	ClassScheduleRuleID uuid.UUID `json:"class_schedule_rule_id"`

	ClassScheduleRuleMasjidID   uuid.UUID `json:"class_schedule_rule_masjid_id"`
	ClassScheduleRuleScheduleID uuid.UUID `json:"class_schedule_rule_schedule_id"`

	ClassScheduleRuleDayOfWeek int    `json:"class_schedule_rule_day_of_week"`
	ClassScheduleRuleStartTime string `json:"class_schedule_rule_start_time"` // "HH:mm:ss"
	ClassScheduleRuleEndTime   string `json:"class_schedule_rule_end_time"`   // "HH:mm:ss"

	ClassScheduleRuleIntervalWeeks    int    `json:"class_schedule_rule_interval_weeks"`
	ClassScheduleRuleStartOffsetWeeks int    `json:"class_schedule_rule_start_offset_weeks"`
	ClassScheduleRuleWeekParity       string `json:"class_schedule_rule_week_parity"`

	ClassScheduleRuleWeeksOfMonth    []int64 `json:"class_schedule_rule_weeks_of_month,omitempty"`
	ClassScheduleRuleLastWeekOfMonth bool    `json:"class_schedule_rule_last_week_of_month"`

	// CSST (idempotent dengan DB)
	ClassScheduleRuleCSSTID       uuid.UUID `json:"class_schedule_rule_csst_id"`
	ClassScheduleRuleCSSTMasjidID uuid.UUID `json:"class_schedule_rule_csst_masjid_id"`

	// Snapshot minimal (raw) + generated columns (flatten) biar UI tanpa join
	ClassScheduleRuleCSSTSnapshot       map[string]any `json:"class_schedule_rule_csst_snapshot,omitempty"`
	ClassScheduleRuleCSSTTeacherID      *uuid.UUID     `json:"class_schedule_rule_csst_teacher_id,omitempty"`
	ClassScheduleRuleCSSTSectionID      *uuid.UUID     `json:"class_schedule_rule_csst_section_id,omitempty"`
	ClassScheduleRuleCSSTClassSubjectID *uuid.UUID     `json:"class_schedule_rule_csst_class_subject_id,omitempty"`
	ClassScheduleRuleCSSTRoomID         *uuid.UUID     `json:"class_schedule_rule_csst_room_id,omitempty"`

	ClassScheduleRuleCreatedAt time.Time `json:"class_schedule_rule_created_at"`
	ClassScheduleRuleUpdatedAt time.Time `json:"class_schedule_rule_updated_at"`
}

/* =========================================================
   4) MAPPERS
   ========================================================= */

func FromRuleModel(m model.ClassScheduleRuleModel) ClassScheduleRuleResponse {
	var weeks []int64
	if len(m.ClassScheduleRuleWeeksOfMonth) > 0 {
		weeks = make([]int64, len(m.ClassScheduleRuleWeeksOfMonth))
		copy(weeks, m.ClassScheduleRuleWeeksOfMonth)
	}

	var snap map[string]any
	if m.ClassScheduleRuleCSSTSnapshot != nil {
		// datatypes.JSONMap underlying type is map[string]any, safe to convert
		snap = map[string]any(m.ClassScheduleRuleCSSTSnapshot)
	}

	return ClassScheduleRuleResponse{
		ClassScheduleRuleID: m.ClassScheduleRuleID,

		ClassScheduleRuleMasjidID:   m.ClassScheduleRuleMasjidID,
		ClassScheduleRuleScheduleID: m.ClassScheduleRuleScheduleID,

		ClassScheduleRuleDayOfWeek: m.ClassScheduleRuleDayOfWeek,
		ClassScheduleRuleStartTime: formatTOD(m.ClassScheduleRuleStartTime),
		ClassScheduleRuleEndTime:   formatTOD(m.ClassScheduleRuleEndTime),

		ClassScheduleRuleIntervalWeeks:    m.ClassScheduleRuleIntervalWeeks,
		ClassScheduleRuleStartOffsetWeeks: m.ClassScheduleRuleStartOffsetWeeks,
		ClassScheduleRuleWeekParity:       string(m.ClassScheduleRuleWeekParity),

		ClassScheduleRuleWeeksOfMonth:    weeks,
		ClassScheduleRuleLastWeekOfMonth: m.ClassScheduleRuleLastWeekOfMonth,

		ClassScheduleRuleCSSTID:       m.ClassScheduleRuleCSSTID,
		ClassScheduleRuleCSSTMasjidID: m.ClassScheduleRuleCSSTMasjidID,

		ClassScheduleRuleCSSTSnapshot:       snap,
		ClassScheduleRuleCSSTTeacherID:      m.ClassScheduleRuleCSSTTeacherID,
		ClassScheduleRuleCSSTSectionID:      m.ClassScheduleRuleCSSTSectionID,
		ClassScheduleRuleCSSTClassSubjectID: m.ClassScheduleRuleCSSTClassSubjectID,
		ClassScheduleRuleCSSTRoomID:         m.ClassScheduleRuleCSSTRoomID,

		ClassScheduleRuleCreatedAt: m.ClassScheduleRuleCreatedAt,
		ClassScheduleRuleUpdatedAt: m.ClassScheduleRuleUpdatedAt,
	}
}

func FromRuleModels(list []model.ClassScheduleRuleModel) []ClassScheduleRuleResponse {
	out := make([]ClassScheduleRuleResponse, 0, len(list))
	for i := range list {
		out = append(out, FromRuleModel(list[i]))
	}
	return out
}

/* =========================================================
   5) Errors (ringan)
   ========================================================= */

var (
	ErrInvalidStartTime = fmtErr("invalid start_time (use HH:mm or HH:mm:ss)")
	ErrInvalidEndTime   = fmtErr("invalid end_time (use HH:mm or HH:mm:ss)")
)

type fmtErr string

func (e fmtErr) Error() string { return string(e) }
