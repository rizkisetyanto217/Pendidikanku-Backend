// file: internals/features/school/sessions/schedules/service/generate_sessions.go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	schedModel "masjidku_backend/internals/features/school/academics/schedules/model"
	sessModel "masjidku_backend/internals/features/school/classes/class_sessions/model"
)

type Generator struct{ DB *gorm.DB }

// ============================
// Options untuk generator
// ============================
type GenerateOptions struct {
	// Timezone lokal untuk interpretasi date/rule time (default: "Asia/Jakarta")
	TZName string

	// Default assignment untuk semua occurrence hasil generate (opsional)
	DefaultCSSTID    *uuid.UUID
	DefaultRoomID    *uuid.UUID
	DefaultTeacherID *uuid.UUID

	// Attendance status default (default: "open")
	DefaultAttendanceStatus string

	// Ukuran batch insert
	BatchSize int
}

// ============================
// Row ringan untuk load rules
// ============================
type ruleRow struct {
	ID              uuid.UUID     `gorm:"column:class_schedule_rules_id"`
	MasjidID        uuid.UUID     `gorm:"column:class_schedule_rule_masjid_id"`
	ScheduleID      uuid.UUID     `gorm:"column:class_schedule_rule_schedule_id"`
	DayOfWeek       int           `gorm:"column:class_schedule_rule_day_of_week"` // 1=Mon..7=Sun (ISO)
	StartStr        string        `gorm:"column:start_str"`                       // TIME::text
	EndStr          string        `gorm:"column:end_str"`                         // TIME::text
	IntervalWeeks   int           `gorm:"column:class_schedule_rule_interval_weeks"`
	StartOffset     int           `gorm:"column:class_schedule_rule_start_offset_weeks"`
	WeekParity      string        `gorm:"column:class_schedule_rule_week_parity"` // all|odd|even
	WeeksOfMonth    pq.Int64Array `gorm:"column:class_schedule_rule_weeks_of_month"`
	LastWeekOfMonth bool          `gorm:"column:class_schedule_rule_last_week_of_month"`
}

// ============================
// Public API
// ============================

// Backward-compat wrapper (tetap ada biar call lama tidak putus).
func (g *Generator) GenerateSessionsForSchedule(ctx context.Context, scheduleID string) (int, error) {
	return g.GenerateSessionsForScheduleWithOpts(ctx, scheduleID, nil)
}

// Versi lengkap dengan options.
func (g *Generator) GenerateSessionsForScheduleWithOpts(ctx context.Context, scheduleID string, opts *GenerateOptions) (created int, err error) {
	// ---------- Options defaults ----------
	if opts == nil {
		opts = &GenerateOptions{}
	}
	if opts.TZName == "" {
		opts.TZName = "Asia/Jakarta"
	}
	if opts.DefaultAttendanceStatus == "" {
		opts.DefaultAttendanceStatus = "open"
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 500
	}

	loc, err := time.LoadLocation(opts.TZName)
	if err != nil {
		// fallback aman
		loc = time.FixedZone("Asia/Jakarta", 7*3600)
	}

	// ---------- 1) Ambil schedule ----------
	var sch schedModel.ClassScheduleModel
	if err = g.DB.WithContext(ctx).
		Where("class_schedule_id = ?", scheduleID).
		Take(&sch).Error; err != nil {
		return 0, err
	}

	// Normalisasi start/end DATE ke lokal
	startLocal := startOfDayInLoc(sch.ClassSchedulesStartDate, loc)
	endLocal := startOfDayInLoc(sch.ClassSchedulesEndDate, loc)
	if endLocal.Before(startLocal) {
		// Jika range salah, ya sudah tidak ada yang digenerate
		return 0, nil
	}

	// ---------- 2) Ambil rules ----------
	var rr []ruleRow
	const qRules = `
SELECT
  class_schedule_rules_id,
  class_schedule_rule_masjid_id,
  class_schedule_rule_schedule_id,
  class_schedule_rule_day_of_week,
  class_schedule_rule_start_time::text AS start_str,
  class_schedule_rule_end_time::text   AS end_str,
  class_schedule_rule_interval_weeks,
  class_schedule_rule_start_offset_weeks,
  class_schedule_rule_week_parity,
  class_schedule_rule_weeks_of_month,
  class_schedule_rule_last_week_of_month
FROM class_schedule_rules
WHERE class_schedule_rule_schedule_id = ?
  AND class_schedule_rule_deleted_at IS NULL
ORDER BY class_schedule_rule_day_of_week, class_schedule_rule_start_time
`
	if err = g.DB.WithContext(ctx).Raw(qRules, sch.ClassScheduleID).Scan(&rr).Error; err != nil {
		return 0, err
	}

	// ---------- 3) Expand occurrences ----------
	rows := make([]sessModel.ClassAttendanceSessionModel, 0, 1024)

	if len(rr) == 0 {
		// Tanpa rule → buat 1 sesi pada start date (date lokal), waktu kosong
		rows = append(rows, sessModel.ClassAttendanceSessionModel{
			ClassAttendanceSessionsMasjidID:         sch.ClassSchedulesMasjidID,
			ClassAttendanceSessionsScheduleID:       sch.ClassScheduleID,
			ClassAttendanceSessionsRuleID:           nil,
			ClassAttendanceSessionsDate:             startLocal, // DATE lokal
			ClassAttendanceSessionsStartsAt:         nil,
			ClassAttendanceSessionsEndsAt:           nil,
			ClassAttendanceSessionsStatus:           sessModel.SessionScheduled,
			ClassAttendanceSessionsAttendanceStatus: opts.DefaultAttendanceStatus,
			ClassAttendanceSessionsLocked:           false,
			ClassAttendanceSessionsIsOverride:       false,
			ClassAttendanceSessionsIsCanceled:       false,
			ClassAttendanceSessionsGeneralInfo:      "",
		})
	} else {
		// Loop per hari di rentang, gunakan tanggal LOKAL
		for d := startLocal; !d.After(endLocal); d = d.AddDate(0, 0, 1) {
			for _, r := range rr {
				if !dateMatchesRuleRow(d, startLocal, r) {
					continue
				}
				// Parse start/end time-of-day
				stTOD, err1 := parseTODString(r.StartStr)
				etTOD, err2 := parseTODString(r.EndStr)
				if err1 != nil || err2 != nil {
					// Jika ada rule time invalid, skip saja tanggal ini untuk rule tsb
					continue
				}

				// Gabungkan d (LOCAL) + TOD (jam-menit-detik), lalu simpan ke DB sebagai UTC (TIMESTAMPTZ)
				startAtLocal := combineLocalDateAndTOD(d, stTOD, loc)
				endAtLocal := combineLocalDateAndTOD(d, etTOD, loc)

				// Overnight guard: kalau end < start, anggap lebihi tengah malam → tambah 1 hari
				if endAtLocal.Before(startAtLocal) {
					endAtLocal = endAtLocal.Add(24 * time.Hour)
				}

				startAtUTC := toUTC(startAtLocal)
				endAtUTC := toUTC(endAtLocal)

				dateUTC := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)

				row := sessModel.ClassAttendanceSessionModel{
					ClassAttendanceSessionsMasjidID:         sch.ClassSchedulesMasjidID,
					ClassAttendanceSessionsScheduleID:       sch.ClassScheduleID,
					ClassAttendanceSessionsRuleID:           &r.ID,
					ClassAttendanceSessionsDate:             dateUTC,     // simpan DATE sebagai midnight UTC
					ClassAttendanceSessionsStartsAt:         &startAtUTC, // UTC
					ClassAttendanceSessionsEndsAt:           &endAtUTC,   // UTC
					ClassAttendanceSessionsStatus:           sessModel.SessionScheduled,
					ClassAttendanceSessionsAttendanceStatus: opts.DefaultAttendanceStatus,
					ClassAttendanceSessionsLocked:           false,
					ClassAttendanceSessionsIsOverride:       false,
					ClassAttendanceSessionsIsCanceled:       false,
					ClassAttendanceSessionsGeneralInfo:      "",
				}

				// Propagasi default assignment jika disediakan
				if opts.DefaultCSSTID != nil {
					row.ClassAttendanceSessionsCSSTID = opts.DefaultCSSTID
				}
				if opts.DefaultRoomID != nil {
					row.ClassAttendanceSessionsClassRoomID = opts.DefaultRoomID
				}
				if opts.DefaultTeacherID != nil {
					row.ClassAttendanceSessionsTeacherID = opts.DefaultTeacherID
				}

				rows = append(rows, row)
			}
		}
	}

	if len(rows) == 0 {
		return 0, nil
	}

	// ---------- 4) Idempotent insert (batch) ----------
	tx := g.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true})
	if err := tx.CreateInBatches(&rows, opts.BatchSize).Error; err != nil {
		return 0, err
	}
	return len(rows), nil
}

// ============================
// Helpers
// ============================

// parse "HH:mm[:ss]" ke time.Time (tanggal dummy) dalam UTC basis
func parseTODString(s string) (time.Time, error) {
	if t, err := time.Parse("15:04:05", s); err == nil {
		return time.Date(2000, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
	}
	if t, err := time.Parse("15:04", s); err == nil {
		return time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("invalid time-of-day format: %q", s)
}

// Start-of-day pada lokasi tertentu
func startOfDayInLoc(t time.Time, loc *time.Location) time.Time {
	tt := t.In(loc)
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc)
}

// Gabungkan tanggal lokal + TOD (jam-menit-detik) ke time lokal
func combineLocalDateAndTOD(dLocal, tod time.Time, loc *time.Location) time.Time {
	return time.Date(dLocal.Year(), dLocal.Month(), dLocal.Day(), tod.Hour(), tod.Minute(), tod.Second(), 0, loc)
}

// Konversi ke UTC
func toUTC(t time.Time) time.Time {
	return t.In(time.UTC)
}

// ISO weekday: Senin=1..Minggu=7
func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

// Week-of-month (ISO): pekan dihitung mulai Senin
func weekOfMonthISO(t time.Time) int {
	// cari Senin pertama yang <= tanggal 1 bulan itu
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	firstWeekStart := first
	for isoWeekday(firstWeekStart) != 1 {
		firstWeekStart = firstWeekStart.AddDate(0, 0, -1)
	}
	days := int(startOfDayInLoc(t, t.Location()).Sub(firstWeekStart).Hours() / 24)
	return (days / 7) + 1
}

func isLastWeekOfMonth(t time.Time) bool {
	return t.AddDate(0, 0, 7).Month() != t.Month()
}

// Selisih pekan antardate (dibulatkan ala kalender, berbasis date lokal)
func weeksBetween(base, target time.Time) int {
	ad := startOfDayInLoc(base, base.Location())
	bd := startOfDayInLoc(target, target.Location())
	if bd.Before(ad) {
		return -int(ad.Sub(bd).Hours() / 24 / 7)
	}
	return int(bd.Sub(ad).Hours() / 24 / 7)
}

// Matcher rule → tanggal lokal
func dateMatchesRuleRow(dLocal, baseStartLocal time.Time, r ruleRow) bool {
	// Day-of-week
	if isoWeekday(dLocal) != r.DayOfWeek {
		return false
	}

	// Interval & offset
	wk := weeksBetween(baseStartLocal, dLocal)
	wkAdj := wk - r.StartOffset
	if wkAdj < 0 {
		return false
	}
	interval := r.IntervalWeeks
	if interval <= 0 {
		interval = 1
	}
	if wkAdj%interval != 0 {
		return false
	}

	// Parity
	switch r.WeekParity {
	case "odd":
		// 1st occurrence (wkAdj=0) dianggap #1 (ganjil)
		if ((wkAdj/interval)+1)%2 != 1 {
			return false
		}
	case "even":
		if ((wkAdj/interval)+1)%2 != 0 {
			return false
		}
	}

	// Weeks of month (opsional)
	if len(r.WeeksOfMonth) > 0 {
		wm := weekOfMonthISO(dLocal)
		ok := false
		for _, w := range r.WeeksOfMonth {
			if int(w) == wm {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	// Last week of month (opsional)
	if r.LastWeekOfMonth && !isLastWeekOfMonth(dLocal) {
		return false
	}

	return true
}
