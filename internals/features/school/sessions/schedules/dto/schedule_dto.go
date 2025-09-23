package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/sessions/schedules/model"

	sessModel "masjidku_backend/internals/features/school/sessions/sessions/model"
)

/* =========================================================
   Helpers
   ========================================================= */

// Pastikan DATE yang disimpan konsisten (midnight UTC) agar tak bergeser
func toUTCDateFromLocal(local time.Time) time.Time {
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

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

// --- Helpers TZ-aware ---
func loadLocOrDefault(tz string) *time.Location {
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		// fallback WIB
		return time.FixedZone("Asia/Jakarta", 7*3600)
	}
	return loc
}

// join tanggal lokal + "HH:mm[:ss]" → kembalikan (UTC, LOCAL)
func joinDateClockInTZ(d time.Time, hhmmss, tz string) (time.Time, time.Time, error) {
	loc := loadLocOrDefault(tz)
	// parse jam
	offset, err := parseClockHHMMSS(hhmmss)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	// d diinterpretasi di TZ lokal
	dayLocal := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
	local := dayLocal.Add(offset)
	utc := local.UTC()
	return utc, local, nil
}

// normalisasi tanggal (pukul 00:00) di TZ tertentu
func startOfDayInTZ(d time.Time, tz string) time.Time {
	loc := loadLocOrDefault(tz)
	dd := d.In(loc)
	return time.Date(dd.Year(), dd.Month(), dd.Day(), 0, 0, 0, 0, loc)
}

/* =================== helper baru =================== */

func parseClockHHMMSS(s string) (time.Duration, error) {
	// terima "HH:mm" atau "HH:mm:ss"
	ss := strings.TrimSpace(s)
	if ss == "" {
		return 0, fmt.Errorf("jam kosong")
	}
	layouts := []string{"15:04:05", "15:04"}
	for _, lo := range layouts {
		if t, err := time.Parse(lo, ss); err == nil {
			return time.Duration(t.Hour())*time.Hour +
				time.Duration(t.Minute())*time.Minute +
				time.Duration(t.Second())*time.Second, nil
		}
	}
	return 0, fmt.Errorf("format jam tidak valid: %s", s)
}

/* =============== payload sessions (lite) =============== */

type CreateClassAttendanceSessionLite struct {
	// tanggal dan jam
	Date      string `json:"date"       validate:"required,datetime=2006-01-02"` // YYYY-MM-DD
	StartTime string `json:"start_time" validate:"required"`                     // "HH:mm" / "HH:mm:ss"
	EndTime   string `json:"end_time"   validate:"required"`

	// opsional metadata (pakai yang ada di model kamu)
	ClassRoomID *uuid.UUID `json:"class_room_id,omitempty" validate:"omitempty"`
	CSSTID      *uuid.UUID `json:"csst_id,omitempty"       validate:"omitempty"` // ClassSectionSubjectTeacher id
	Status      *string    `json:"status,omitempty"        validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	Notes       *string    `json:"notes,omitempty"         validate:"omitempty,max=500"`
}

// Mapper: sessions → models (butuh masjid & schedule id)
func (r CreateClassScheduleRequest) SessionsToModels(
	masjidID, scheduleID uuid.UUID,
	schedStart, schedEnd time.Time,
) ([]sessModel.ClassAttendanceSessionModel, error) {
	if len(r.Sessions) == 0 {
		return nil, nil
	}
	out := make([]sessModel.ClassAttendanceSessionModel, 0, len(r.Sessions))

	// TODO: ambil TZ dari profil masjid kalau ada
	const tzName = "Asia/Jakarta"

	for i, s := range r.Sessions {
		d, ok := parseDateYYYYMMDD(s.Date)
		if !ok {
			return nil, fmt.Errorf("sessions[%d]: tanggal invalid (harap YYYY-MM-DD)", i)
		}
		if d.Before(schedStart) || d.After(schedEnd) {
			return nil, fmt.Errorf("sessions[%d]: tanggal di luar rentang schedule", i)
		}

		// --- Waktu lokal & UTC ---
		startUTC, startLocal, err := joinDateClockInTZ(d, s.StartTime, tzName)
		if err != nil {
			return nil, fmt.Errorf("sessions[%d]: start_time invalid: %v", i, err)
		}
		endUTC, endLocal, err := joinDateClockInTZ(d, s.EndTime, tzName)
		if err != nil {
			return nil, fmt.Errorf("sessions[%d]: end_time invalid: %v", i, err)
		}
		// overnight guard
		if endLocal.Before(startLocal) {
			endLocal = endLocal.Add(24 * time.Hour)
			endUTC = endLocal.UTC()
		}
		if !endLocal.After(startLocal) {
			return nil, fmt.Errorf("sessions[%d]: end_time harus > start_time", i)
		}

		st := sessModel.SessionScheduled
		if s.Status != nil {
			switch strings.ToLower(strings.TrimSpace(*s.Status)) {
			case "ongoing":
				st = sessModel.SessionOngoing
			case "completed":
				st = sessModel.SessionCompleted
			case "canceled":
				st = sessModel.SessionCanceled
			default:
				st = sessModel.SessionScheduled
			}
		}

		// --- PENTING: set DATE lokal dari startLocal ---
		dateLocal := startOfDayInTZ(startLocal, tzName)
		dateUTC := toUTCDateFromLocal(dateLocal)

		m := sessModel.ClassAttendanceSessionModel{
			ClassAttendanceSessionsMasjidID:   masjidID,
			ClassAttendanceSessionsScheduleID: scheduleID,

			ClassAttendanceSessionsDate:     dateUTC,   // <— DATE disimpan midnight UTC
			ClassAttendanceSessionsStartsAt: &startUTC, // UTC
			ClassAttendanceSessionsEndsAt:   &endUTC,   // UTC

			ClassAttendanceSessionsStatus:           st,
			ClassAttendanceSessionsAttendanceStatus: "open", // <— default agar worker happy
			ClassAttendanceSessionsNote:             trimPtr(s.Notes),

			// opsional FK
			ClassAttendanceSessionsClassRoomID: s.ClassRoomID,
			ClassAttendanceSessionsCSSTID:      s.CSSTID,
		}
		out = append(out, m)
	}
	return out, nil
}

/* =========================================================
   1) REQUESTS
   ========================================================= */

// Create: masjid_id dipaksa dari controller (parameter ToModel)
// Sekarang mendukung pengiriman RULES langsung.
type CreateClassScheduleRequest struct {
	// optional slug
	ClassSchedulesSlug *string `json:"class_schedules_slug" validate:"omitempty,max=160"`

	// rentang wajib (YYYY-MM-DD)
	ClassSchedulesStartDate string `json:"class_schedules_start_date" validate:"required,datetime=2006-01-02"`
	ClassSchedulesEndDate   string `json:"class_schedules_end_date"   validate:"required,datetime=2006-01-02"`

	// status & aktif
	ClassSchedulesStatus   *string `json:"class_schedules_status"   validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	ClassSchedulesIsActive *bool   `json:"class_schedules_is_active" validate:"omitempty"`

	// RULES opsional — lite (tanpa schedule_id; akan diisi server)
	Rules []CreateClassScheduleRuleLite `json:"rules" validate:"omitempty,dive"`

	Sessions []CreateClassAttendanceSessionLite `json:"sessions" validate:"omitempty,dive"`
}

// Lite rule untuk keperluan embed di CreateClassScheduleRequest
type CreateClassScheduleRuleLite struct {
	DayOfWeek        int     `json:"day_of_week"         validate:"required,min=1,max=7"`
	StartTime        string  `json:"start_time"          validate:"required"` // "HH:mm" / "HH:mm:ss"
	EndTime          string  `json:"end_time"            validate:"required"` // "HH:mm" / "HH:mm:ss"
	IntervalWeeks    *int    `json:"interval_weeks"      validate:"omitempty,min=1"`
	StartOffsetWeeks *int    `json:"start_offset_weeks"  validate:"omitempty,min=0"`
	WeekParity       *string `json:"week_parity"         validate:"omitempty,oneof=all odd even"`
	WeeksOfMonth     []int   `json:"weeks_of_month"      validate:"omitempty,dive,min=1,max=5"`
	LastWeekOfMonth  *bool   `json:"last_week_of_month"  validate:"omitempty"`
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
		ClassSchedulesMasjidID:  masjidID,
		ClassSchedulesSlug:      trimPtr(r.ClassSchedulesSlug),
		ClassSchedulesStartDate: start,
		ClassSchedulesEndDate:   end,
		ClassSchedulesStatus:    model.SessionStatusEnum(status),
		ClassSchedulesIsActive:  isActive,
	}
}

// Konversi Rules lite di body menjadi models lengkap (butuh masjidID & scheduleID)
func (r CreateClassScheduleRequest) RulesToModels(masjidID, scheduleID uuid.UUID) ([]model.ClassScheduleRuleModel, error) {
	if len(r.Rules) == 0 {
		return nil, nil
	}
	out := make([]model.ClassScheduleRuleModel, 0, len(r.Rules))
	for _, it := range r.Rules {
		cr := CreateClassScheduleRuleRequest{
			ClassScheduleRuleScheduleID:       scheduleID,
			ClassScheduleRuleDayOfWeek:        it.DayOfWeek,
			ClassScheduleRuleStartTime:        it.StartTime,
			ClassScheduleRuleEndTime:          it.EndTime,
			ClassScheduleRuleIntervalWeeks:    it.IntervalWeeks,
			ClassScheduleRuleStartOffsetWeeks: it.StartOffsetWeeks,
			ClassScheduleRuleWeekParity:       it.WeekParity,
			ClassScheduleRuleWeeksOfMonth:     it.WeeksOfMonth,
			ClassScheduleRuleLastWeekOfMonth:  it.LastWeekOfMonth,
		}
		m, err := cr.ToModel(masjidID)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
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
	// pilihan:
	//   start_date_asc|start_date_desc|end_date_asc|end_date_desc|
	//   created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
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
