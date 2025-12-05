// file: internals/features/school/schedules/dto/class_schedule_dto.go
package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	sessModel "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
	model "madinahsalam_backend/internals/features/school/class_others/class_schedules/model"
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
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{"2006-01-02", "2006-1-2"}
	for _, lo := range layouts {
		if t, err := time.Parse(lo, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
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
		if t, err := time.Parse(lo, s); err == nil {
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

	// opsional metadata (pakai yang ada di model session)
	ClassRoomID *uuid.UUID `json:"class_room_id,omitempty" validate:"omitempty"`

	// CSST efektif untuk sesi (WAJIB di DB); kalau school_id tidak diisi, fallback ke schoolID schedule
	CSSTID       *uuid.UUID `json:"csst_id,omitempty"         validate:"omitempty,uuid"`
	CSSTSchoolID *uuid.UUID `json:"csst_school_id,omitempty"  validate:"omitempty,uuid"`

	// opsional metadata
	TeacherID *uuid.UUID `json:"teacher_id,omitempty" validate:"omitempty,uuid"`
	SubjectID *uuid.UUID `json:"subject_id,omitempty" validate:"omitempty,uuid"`
	SectionID *uuid.UUID `json:"section_id,omitempty" validate:"omitempty,uuid"`

	// opsional type (manual session)
	SessionTypeID *uuid.UUID `json:"session_type_id,omitempty" validate:"omitempty,uuid"`

	Status *string `json:"status,omitempty" validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	Notes  *string `json:"notes,omitempty"  validate:"omitempty,max=500"`
}

// Mapper: sessions → models (butuh school & schedule id)
func (r CreateClassScheduleRequest) SessionsToModels(
	schoolID, scheduleID uuid.UUID,
	schedStart, schedEnd time.Time,
) ([]sessModel.ClassAttendanceSessionModel, error) {
	if len(r.Sessions) == 0 {
		return nil, nil
	}
	out := make([]sessModel.ClassAttendanceSessionModel, 0, len(r.Sessions))

	// TODO: ambil TZ dari profil school kalau ada
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
		// Overnight guard
		if endLocal.Before(startLocal) {
			endLocal = endLocal.Add(24 * time.Hour)
			endUTC = endLocal.In(time.UTC)
		}
		if !endLocal.After(startLocal) {
			return nil, fmt.Errorf("sessions[%d]: end_time harus > start_time", i)
		}

		// Status session
		st := sessModel.SessionStatusScheduled
		if s.Status != nil {
			switch strings.ToLower(strings.TrimSpace(*s.Status)) {
			case "ongoing":
				st = sessModel.SessionStatusOngoing
			case "completed":
				st = sessModel.SessionStatusCompleted
			case "canceled":
				st = sessModel.SessionStatusCanceled
			}
		}

		// ===== CSST efektif (WAJIB) =====
		if s.CSSTID == nil {
			return nil, fmt.Errorf("sessions[%d]: csst_id wajib", i)
		}
		// Komposit FK: (csst_id, session_school_id). Jika csst_school_id diisi dan beda → tolak.
		// Jika tidak diisi, dianggap sama dengan schoolID dari path (tidak perlu disimpan eksplisit).
		if s.CSSTSchoolID != nil && *s.CSSTSchoolID != schoolID {
			return nil, fmt.Errorf("sessions[%d]: csst_school_id != path school_id", i)
		}

		// --- Set DATE lokal dari startLocal ---
		dateLocal := startOfDayInTZ(startLocal, tzName)
		dateUTC := toUTCDateFromLocal(dateLocal)

		m := sessModel.ClassAttendanceSessionModel{
			ClassAttendanceSessionSchoolID:   schoolID,
			ClassAttendanceSessionScheduleID: &scheduleID,

			ClassAttendanceSessionDate:     dateUTC,   // DATE (midnight UTC)
			ClassAttendanceSessionStartsAt: &startUTC, // UTC
			ClassAttendanceSessionEndsAt:   &endUTC,   // UTC

			ClassAttendanceSessionStatus:           st,
			ClassAttendanceSessionAttendanceStatus: sessModel.AttendanceStatusOpen,
			ClassAttendanceSessionNote:             trimPtr(s.Notes),

			// optional overrides
			ClassAttendanceSessionClassRoomID: s.ClassRoomID,
			ClassAttendanceSessionTeacherID:   s.TeacherID,

			// CSST pointer saja (komposit FK pakai school_id schedule)
			ClassAttendanceSessionCSSTID: s.CSSTID,
		}

		// opsional: session type (manual)
		if s.SessionTypeID != nil {
			m.ClassAttendanceSessionTypeID = s.SessionTypeID
			// snapshot biarkan kosong; bisa di-enrich di service lain kalau perlu
		}

		out = append(out, m)
	}
	return out, nil
}

/* =========================================================
   1) REQUESTS (singular)
========================================================= */

// Create: school_id dipaksa dari controller (parameter ToModel).
// Mendukung pengiriman RULES & SESSIONS langsung.
type CreateClassScheduleRequest struct {
	// optional slug
	ClassScheduleSlug *string `json:"class_schedule_slug" validate:"omitempty,max=160"`

	// rentang wajib (YYYY-MM-DD)
	ClassScheduleStartDate string `json:"class_schedule_start_date" validate:"required,datetime=2006-01-02"`
	ClassScheduleEndDate   string `json:"class_schedule_end_date"   validate:"required,datetime=2006-01-02"`

	// status & aktif
	ClassScheduleStatus   *string `json:"class_schedule_status"   validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	ClassScheduleIsActive *bool   `json:"class_schedule_is_active" validate:"omitempty"`

	GenerateSessions *bool `json:"generate_sessions,omitempty"` // default: true

	// Defaults untuk semua sesi hasil generate (opsional)
	DefaultCSSTID    *uuid.UUID `json:"default_csst_id"    validate:"omitempty,uuid"`
	DefaultRoomID    *uuid.UUID `json:"default_room_id"    validate:"omitempty,uuid"`
	DefaultTeacherID *uuid.UUID `json:"default_teacher_id" validate:"omitempty,uuid"`

	// default session type untuk semua sesi hasil generate
	SessionTypeID *uuid.UUID `json:"session_type_id,omitempty" validate:"omitempty,uuid"`

	// RULES opsional — lite (tanpa schedule_id; akan diisi server)
	Rules []CreateClassScheduleRuleLite `json:"rules" validate:"omitempty,dive"`

	// SESSIONS opsional — lite
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

	// CSST wajib untuk rule
	CSSTID       uuid.UUID  `json:"csst_id"                  validate:"required,uuid"`
	CSSTSchoolID *uuid.UUID `json:"csst_school_id,omitempty" validate:"omitempty,uuid"` // hanya untuk validasi; tidak dipetakan lagi
}

func (r CreateClassScheduleRequest) ToModel(schoolID uuid.UUID) (model.ClassScheduleModel, error) {
	start, ok := parseDateYYYYMMDD(r.ClassScheduleStartDate)
	if !ok {
		return model.ClassScheduleModel{}, fmt.Errorf("class_schedule_start_date format invalid (gunakan YYYY-MM-DD)")
	}

	end, ok := parseDateYYYYMMDD(r.ClassScheduleEndDate)
	if !ok {
		return model.ClassScheduleModel{}, fmt.Errorf("class_schedule_end_date format invalid (gunakan YYYY-MM-DD)")
	}

	status := model.SessionStatusScheduled
	if r.ClassScheduleStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassScheduleStatus)) {
		case "ongoing":
			status = model.SessionStatusOngoing
		case "completed":
			status = model.SessionStatusCompleted
		case "canceled":
			status = model.SessionStatusCanceled
		default:
			status = model.SessionStatusScheduled
		}
	}

	isActive := true
	if r.ClassScheduleIsActive != nil {
		isActive = *r.ClassScheduleIsActive
	}

	return model.ClassScheduleModel{
		ClassScheduleSchoolID:  schoolID,
		ClassScheduleSlug:      trimPtr(r.ClassScheduleSlug),
		ClassScheduleStartDate: start,
		ClassScheduleEndDate:   end,
		ClassScheduleStatus:    status,
		ClassScheduleIsActive:  isActive,
	}, nil
}

// Konversi Rules lite di body menjadi models lengkap (butuh schoolID & scheduleID)
func (r CreateClassScheduleRequest) RulesToModels(schoolID, scheduleID uuid.UUID) ([]model.ClassScheduleRuleModel, error) {
	if len(r.Rules) == 0 {
		return nil, nil
	}
	out := make([]model.ClassScheduleRuleModel, 0, len(r.Rules))

	for idx, it := range r.Rules {
		// validasi ringan
		if strings.TrimSpace(it.StartTime) == "" || strings.TrimSpace(it.EndTime) == "" {
			return nil, fmt.Errorf("rules[%d]: start_time/end_time wajib", idx)
		}

		// Jika klien mengirim csst_school_id dan beda dengan schoolID path → tolak (guard aplikasi saja)
		if it.CSSTSchoolID != nil && *it.CSSTSchoolID != schoolID {
			return nil, fmt.Errorf("rules[%d]: csst_school_id != path school_id", idx)
		}

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

			// CSST wajib
			ClassScheduleRuleCSSTID: it.CSSTID,
		}

		m, err := cr.ToModel(schoolID)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}

// Update (partial) — singular
type UpdateClassScheduleRequest struct {
	ClassScheduleSlug      *string `json:"class_schedule_slug"       validate:"omitempty,max=160"`
	ClassScheduleStartDate *string `json:"class_schedule_start_date" validate:"omitempty,datetime=2006-01-02"`
	ClassScheduleEndDate   *string `json:"class_schedule_end_date"   validate:"omitempty,datetime=2006-01-02"`
	ClassScheduleStatus    *string `json:"class_schedule_status"     validate:"omitempty,oneof=scheduled ongoing completed canceled"`
	ClassScheduleIsActive  *bool   `json:"class_schedule_is_active"  validate:"omitempty"`
}

func (r UpdateClassScheduleRequest) Apply(m *model.ClassScheduleModel) {
	if r.ClassScheduleSlug != nil {
		m.ClassScheduleSlug = trimPtr(r.ClassScheduleSlug)
	}
	if r.ClassScheduleStartDate != nil {
		if t, ok := parseDateYYYYMMDD(*r.ClassScheduleStartDate); ok {
			m.ClassScheduleStartDate = t
		}
	}
	if r.ClassScheduleEndDate != nil {
		if t, ok := parseDateYYYYMMDD(*r.ClassScheduleEndDate); ok {
			m.ClassScheduleEndDate = t
		}
	}
	if r.ClassScheduleStatus != nil {
		switch strings.ToLower(strings.TrimSpace(*r.ClassScheduleStatus)) {
		case "scheduled":
			m.ClassScheduleStatus = model.SessionStatusScheduled
		case "ongoing":
			m.ClassScheduleStatus = model.SessionStatusOngoing
		case "completed":
			m.ClassScheduleStatus = model.SessionStatusCompleted
		case "canceled":
			m.ClassScheduleStatus = model.SessionStatusCanceled
		}
	}
	if r.ClassScheduleIsActive != nil {
		m.ClassScheduleIsActive = *r.ClassScheduleIsActive
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
   3) RESPONSES (singular)
========================================================= */

type ClassScheduleResponse struct {
	ClassScheduleID       uuid.UUID `json:"class_schedule_id"`
	ClassScheduleSchoolID uuid.UUID `json:"class_schedule_school_id"`

	ClassScheduleSlug      *string   `json:"class_schedule_slug,omitempty"`
	ClassScheduleStartDate time.Time `json:"class_schedule_start_date"`
	ClassScheduleEndDate   time.Time `json:"class_schedule_end_date"`
	ClassScheduleStatus    string    `json:"class_schedule_status"`
	ClassScheduleIsActive  bool      `json:"class_schedule_is_active"`

	ClassScheduleCreatedAt time.Time  `json:"class_schedule_created_at"`
	ClassScheduleUpdatedAt *time.Time `json:"class_schedule_updated_at,omitempty"`
	ClassScheduleDeletedAt *time.Time `json:"class_schedule_deleted_at,omitempty"`
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
	if m.ClassScheduleDeletedAt.Valid {
		d := m.ClassScheduleDeletedAt.Time
		deletedAt = &d
	}

	return ClassScheduleResponse{
		ClassScheduleID:       m.ClassScheduleID,
		ClassScheduleSchoolID: m.ClassScheduleSchoolID,

		ClassScheduleSlug:      m.ClassScheduleSlug,
		ClassScheduleStartDate: m.ClassScheduleStartDate,
		ClassScheduleEndDate:   m.ClassScheduleEndDate,
		ClassScheduleStatus:    string(m.ClassScheduleStatus),
		ClassScheduleIsActive:  m.ClassScheduleIsActive,

		ClassScheduleCreatedAt: m.ClassScheduleCreatedAt,
		ClassScheduleUpdatedAt: timePtrOrNil(m.ClassScheduleUpdatedAt),
		ClassScheduleDeletedAt: deletedAt,
	}
}

func FromModels(list []model.ClassScheduleModel) []ClassScheduleResponse {
	out := make([]ClassScheduleResponse, 0, len(list))
	for _, m := range list {
		out = append(out, FromModel(m))
	}
	return out
}
