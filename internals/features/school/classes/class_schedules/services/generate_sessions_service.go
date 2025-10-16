// file: internals/features/school/sessions/schedules/service/generate_sessions.go
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	sessModel "masjidku_backend/internals/features/school/classes/class_attendance_sessions/model"
	schedModel "masjidku_backend/internals/features/school/classes/class_schedules/model"
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
	ID              uuid.UUID     `gorm:"column:class_schedule_rule_id"`
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
	CSSTID          *uuid.UUID    `gorm:"column:class_schedule_rule_csst_id"` // CSST per-rule (opsional)
}

// ============================
// Row ringan untuk snapshot (v2)
// ============================
type csstLite struct {
	ID        uuid.UUID  `gorm:"column:class_section_subject_teacher_id"`
	MasjidID  uuid.UUID  `gorm:"column:class_section_subject_teacher_masjid_id"`
	Name      *string    `gorm:"column:class_section_subject_teacher_name"`
	TeacherID *uuid.UUID `gorm:"column:masjid_teacher_id"`
}

type teacherLite struct {
	ID       uuid.UUID `gorm:"column:masjid_teacher_id"`
	MasjidID uuid.UUID `gorm:"column:masjid_teacher_masjid_id"`
	Name     *string   `gorm:"column:teacher_name"` // COALESCE(snapshot, ut.name)
	Whatsapp *string   `gorm:"column:whatsapp_url"` // ut.user_teacher_whatsapp_url
	TitlePre *string   `gorm:"column:title_prefix"` // ut.user_teacher_title_prefix
	TitleSuf *string   `gorm:"column:title_suffix"` // ut.user_teacher_title_suffix
}

// ============================
// Public API
// ============================

// Backward-compat wrapper
func (g *Generator) GenerateSessionsForSchedule(ctx context.Context, scheduleID string) (int, error) {
	return g.GenerateSessionsForScheduleWithOpts(ctx, scheduleID, nil)
}

// Versi lengkap dengan options
func (g *Generator) GenerateSessionsForScheduleWithOpts(ctx context.Context, scheduleID string, opts *GenerateOptions) (created int, err error) {
	// ---------- Options defaults ----------
	if opts == nil {
		opts = &GenerateOptions{}
	}
	if opts.TZName == "" {
		opts.TZName = "Asia/Jakarta"
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 500
	}

	// AttendanceStatus default -> tipe enum pada model
	attendanceDefault := sessModel.AttendanceStatusOpen
	if s := stringsTrimLower(opts.DefaultAttendanceStatus); s != "" {
		attendanceDefault = sessModel.AttendanceStatus(s)
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

	// Normalisasi start/end DATE ke lokal (pakai field singular)
	startLocal := startOfDayInLoc(sch.ClassScheduleStartDate, loc)
	endLocal := startOfDayInLoc(sch.ClassScheduleEndDate, loc)
	if endLocal.Before(startLocal) {
		// Range salah → tidak ada yang digenerate
		return 0, nil
	}

	// ---------- 2) Ambil rules (termasuk CSST per-rule bila ada) ----------
	cRules, _ := g.tableColumns(ctx, "class_schedule_rules")
	csstSelect := "NULL::uuid"
	if _, ok := cRules["class_schedule_rule_csst_id"]; ok {
		csstSelect = "class_schedule_rule_csst_id"
	}

	var rr []ruleRow
	qRules := fmt.Sprintf(`
SELECT
  class_schedule_rule_id,
  class_schedule_rule_masjid_id,
  class_schedule_rule_schedule_id,
  class_schedule_rule_day_of_week,
  class_schedule_rule_start_time::text AS start_str,
  class_schedule_rule_end_time::text   AS end_str,
  class_schedule_rule_interval_weeks,
  class_schedule_rule_start_offset_weeks,
  class_schedule_rule_week_parity,
  class_schedule_rule_weeks_of_month,
  class_schedule_rule_last_week_of_month,
  %s AS class_schedule_rule_csst_id
FROM class_schedule_rules
WHERE class_schedule_rule_schedule_id = ?
  AND class_schedule_rule_deleted_at IS NULL
ORDER BY class_schedule_rule_day_of_week, class_schedule_rule_start_time`, csstSelect)

	if err = g.DB.WithContext(ctx).Raw(qRules, sch.ClassScheduleID).Scan(&rr).Error; err != nil {
		return 0, err
	}

	// ---------- 2.5) PRELOAD SNAPSHOTS default (sekali) ----------
	var (
		defCSSTSnap    datatypes.JSONMap
		defTeacherSnap datatypes.JSONMap
		defRoomSnap    datatypes.JSONMap

		teacherIDFromCSST *uuid.UUID
	)

	// CSST snapshot (jika default disediakan)
	if opts.DefaultCSSTID != nil {
		if s, tid, er := g.loadCSSTSnapshot(ctx, *opts.DefaultCSSTID); er == nil {
			defCSSTSnap = s
			teacherIDFromCSST = tid
		}
	}

	// Teacher snapshot: prioritas DefaultTeacherID; kalau kosong pakai teacher dari CSST
	if opts.DefaultTeacherID != nil {
		if s, er := g.loadTeacherSnapshot(ctx, *opts.DefaultTeacherID); er == nil {
			defTeacherSnap = s
		}
	} else if teacherIDFromCSST != nil {
		if s, er := g.loadTeacherSnapshot(ctx, *teacherIDFromCSST); er == nil {
			defTeacherSnap = s
		}
	}

	// Room snapshot
	if opts.DefaultRoomID != nil {
		if s, er := g.loadRoomSnapshot(ctx, *opts.DefaultRoomID); er == nil {
			defRoomSnap = s
		}
	}

	// ---------- Cache untuk snapshot per entitas ----------
	csstSnapCache := map[uuid.UUID]datatypes.JSONMap{}
	csstTeacherIDCache := map[uuid.UUID]*uuid.UUID{}
	teacherSnapCache := map[uuid.UUID]datatypes.JSONMap{}
	roomSnapCache := map[uuid.UUID]datatypes.JSONMap{}

	// ---------- 3) Expand occurrences ----------
	rows := make([]sessModel.ClassAttendanceSessionModel, 0, 1024)

	// Helper: pasang CSST/Teacher/Room + snapshot ke row
	attachSnapshots := func(row *sessModel.ClassAttendanceSessionModel, ruleCSST *uuid.UUID) {
		// Tentukan CSST efektif: per-rule > default
		var effCSST *uuid.UUID
		var effCSSTSnap datatypes.JSONMap
		var effTeacherFromCSST *uuid.UUID

		if ruleCSST != nil {
			effCSST = ruleCSST
			// snapshot dari cache atau load
			if s, ok := csstSnapCache[*ruleCSST]; ok {
				effCSSTSnap = s
				effTeacherFromCSST = csstTeacherIDCache[*ruleCSST]
			} else if s, tid, er := g.loadCSSTSnapshot(ctx, *ruleCSST); er == nil {
				csstSnapCache[*ruleCSST] = s
				csstTeacherIDCache[*ruleCSST] = tid
				effCSSTSnap = s
				effTeacherFromCSST = tid
			}
		} else if opts.DefaultCSSTID != nil {
			effCSST = opts.DefaultCSSTID
			effCSSTSnap = defCSSTSnap
			effTeacherFromCSST = teacherIDFromCSST
		}

		// Set CSST + snapshot
		if effCSST != nil {
			row.ClassAttendanceSessionCSSTID = effCSST
			if effCSSTSnap != nil {
				row.ClassAttendanceSessionCSSTSnapshot = effCSSTSnap
			}
		}

		// Tentukan TEACHER: default > dari CSST > none
		var effTeacher *uuid.UUID
		var effTeacherSnap datatypes.JSONMap

		if opts.DefaultTeacherID != nil {
			effTeacher = opts.DefaultTeacherID
			effTeacherSnap = defTeacherSnap
		} else if effTeacherFromCSST != nil {
			effTeacher = effTeacherFromCSST
			// ambil snapshot dari cache atau load
			if s, ok := teacherSnapCache[*effTeacherFromCSST]; ok {
				effTeacherSnap = s
			} else if s, er := g.loadTeacherSnapshot(ctx, *effTeacherFromCSST); er == nil {
				teacherSnapCache[*effTeacherFromCSST] = s
				effTeacherSnap = s
			}
		}
		if effTeacher != nil {
			row.ClassAttendanceSessionTeacherID = effTeacher
		}
		if effTeacherSnap != nil {
			row.ClassAttendanceSessionTeacherSnapshot = effTeacherSnap
		}

		// ROOM: hanya default (belum ada per-rule)
		if opts.DefaultRoomID != nil {
			row.ClassAttendanceSessionClassRoomID = opts.DefaultRoomID
			if defRoomSnap != nil {
				row.ClassAttendanceSessionRoomSnapshot = defRoomSnap
			} else {
				// jaga-jaga kalau belum diload
				if s, ok := roomSnapCache[*opts.DefaultRoomID]; ok {
					row.ClassAttendanceSessionRoomSnapshot = s
				} else if s, er := g.loadRoomSnapshot(ctx, *opts.DefaultRoomID); er == nil {
					roomSnapCache[*opts.DefaultRoomID] = s
					row.ClassAttendanceSessionRoomSnapshot = s
				}
			}
		}
	}

	if len(rr) == 0 {
		// Tanpa rule → buat 1 sesi pada start date, tanpa jam
		// DATE disimpan sebagai midnight UTC (konsisten)
		dateUTC := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, time.UTC)

		row := sessModel.ClassAttendanceSessionModel{
			ClassAttendanceSessionMasjidID:         sch.ClassScheduleMasjidID,
			ClassAttendanceSessionScheduleID:       sch.ClassScheduleID,
			ClassAttendanceSessionRuleID:           nil,
			ClassAttendanceSessionDate:             dateUTC,
			ClassAttendanceSessionStartsAt:         nil,
			ClassAttendanceSessionEndsAt:           nil,
			ClassAttendanceSessionStatus:           sessModel.SessionStatusScheduled,
			ClassAttendanceSessionAttendanceStatus: attendanceDefault,
			ClassAttendanceSessionLocked:           false,
			ClassAttendanceSessionIsOverride:       false,
			ClassAttendanceSessionIsCanceled:       false,
			ClassAttendanceSessionGeneralInfo:      "",
		}

		// Default assignment + SNAPSHOTS
		attachSnapshots(&row, nil)
		rows = append(rows, row)
	} else {
		// Dengan rules → loop setiap hari di rentang lokal
		for d := startLocal; !d.After(endLocal); d = d.AddDate(0, 0, 1) {
			for _, r := range rr {
				if !dateMatchesRuleRow(d, startLocal, r) {
					continue
				}
				// Parse start/end time-of-day
				stTOD, err1 := parseTODString(r.StartStr)
				etTOD, err2 := parseTODString(r.EndStr)
				if err1 != nil || err2 != nil {
					continue
				}

				// Gabungkan d (LOCAL) + TOD (jam-menit-detik), lalu konversi ke UTC
				startAtLocal := combineLocalDateAndTOD(d, stTOD, loc)
				endAtLocal := combineLocalDateAndTOD(d, etTOD, loc)

				// Overnight guard: kalau end < start → anggap lewat tengah malam
				if endAtLocal.Before(startAtLocal) {
					endAtLocal = endAtLocal.Add(24 * time.Hour)
				}

				startAtUTC := toUTC(startAtLocal)
				endAtUTC := toUTC(endAtLocal)

				// DATE disimpan sebagai midnight UTC
				dateUTC := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)

				// ⚠️ penting: simpan pointer ke salinan lokal, bukan &r.ID
				rid := r.ID

				row := sessModel.ClassAttendanceSessionModel{
					ClassAttendanceSessionMasjidID:         sch.ClassScheduleMasjidID,
					ClassAttendanceSessionScheduleID:       sch.ClassScheduleID,
					ClassAttendanceSessionRuleID:           &rid, // pointer unik per-row
					ClassAttendanceSessionDate:             dateUTC,
					ClassAttendanceSessionStartsAt:         &startAtUTC,
					ClassAttendanceSessionEndsAt:           &endAtUTC,
					ClassAttendanceSessionStatus:           sessModel.SessionStatusScheduled,
					ClassAttendanceSessionAttendanceStatus: attendanceDefault,
					ClassAttendanceSessionLocked:           false,
					ClassAttendanceSessionIsOverride:       false,
					ClassAttendanceSessionIsCanceled:       false,
					ClassAttendanceSessionGeneralInfo:      "",
				}

				// Per-rule CSST > Default CSST
				attachSnapshots(&row, r.CSSTID)

				rows = append(rows, row)
			}
		}
	}

	if len(rows) == 0 {
		return 0, nil
	}

	// ---------- 4) Idempotent insert (batch) ----------
	db := g.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true})
	tx := db.CreateInBatches(rows, opts.BatchSize) // TANPA &rows
	if tx.Error != nil {
		return 0, tx.Error
	}
	return int(tx.RowsAffected), nil
}

// ============================
// Loaders Snapshot (adaptif & aman)
// ============================

// v2: minimal CSST snapshot (id, masjid_id, name, teacher_id)
func (g *Generator) loadCSSTSnapshot(ctx context.Context, id uuid.UUID) (snap datatypes.JSONMap, teacherID *uuid.UUID, err error) {
	cols, err := g.tableColumns(ctx, "class_section_subject_teachers")
	if err != nil {
		return nil, nil, err
	}

	idCol := firstExisting(cols, "class_section_subject_teacher_id", "id")
	masjidCol := firstExisting(cols, "class_section_subject_teacher_masjid_id", "masjid_id")
	nameCol := firstExisting(cols, "class_section_subject_teacher_name", "name")
	teachCol := firstExisting(cols, "class_section_subject_teacher_teacher_id", "masjid_teacher_id", "teacher_id")
	deletedCol := firstExisting(cols, "class_section_subject_teacher_deleted_at", "deleted_at")

	if idCol == "" || masjidCol == "" || nameCol == "" {
		return nil, nil, fmt.Errorf("loadCSSTSnapshot: kolom minimal (id/masjid_id/name) tidak ditemukan di class_section_subject_teachers")
	}

	teachExpr := "NULL::uuid"
	if teachCol != "" {
		teachExpr = fmt.Sprintf("csst.%s", teachCol)
	}

	whereDeleted := ""
	if deletedCol != "" {
		whereDeleted = fmt.Sprintf(" AND csst.%s IS NULL", deletedCol)
	}

	q := fmt.Sprintf(`
SELECT
  csst.%s AS class_section_subject_teacher_id,
  csst.%s AS class_section_subject_teacher_masjid_id,
  csst.%s AS class_section_subject_teacher_name,
  %s      AS masjid_teacher_id
FROM class_section_subject_teachers csst
WHERE csst.%s = ?
%s
LIMIT 1`,
		idCol, masjidCol, nameCol, teachExpr, idCol, whereDeleted,
	)

	var row csstLite
	if err = g.DB.WithContext(ctx).Raw(q, id).Scan(&row).Error; err != nil {
		return nil, nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil, nil
	}
	if row.TeacherID != nil {
		teacherID = row.TeacherID
	}

	snap = datatypes.JSONMap{
		"id":          row.ID,
		"masjid_id":   row.MasjidID,
		"name":        row.Name,
		"teacher_id":  row.TeacherID,
		"captured_at": time.Now().UTC(),
		"source":      "generator_v2",
	}
	return snap, teacherID, nil
}

func (g *Generator) loadTeacherSnapshot(ctx context.Context, id uuid.UUID) (datatypes.JSONMap, error) {
	const q = `
SELECT
  mt.masjid_teacher_id                                AS masjid_teacher_id,
  mt.masjid_teacher_masjid_id                         AS masjid_teacher_masjid_id,
  COALESCE(mt.masjid_teacher_user_teacher_name_snapshot, ut.user_teacher_name) AS teacher_name,
  ut.user_teacher_whatsapp_url                        AS whatsapp_url,
  ut.user_teacher_title_prefix                        AS title_prefix,
  ut.user_teacher_title_suffix                        AS title_suffix
FROM masjid_teachers mt
LEFT JOIN user_teachers ut
  ON ut.user_teacher_id = mt.masjid_teacher_user_teacher_id
 AND ut.user_teacher_deleted_at IS NULL
WHERE mt.masjid_teacher_id = ?
  AND mt.masjid_teacher_deleted_at IS NULL
LIMIT 1`

	var row teacherLite
	if err := g.DB.WithContext(ctx).Raw(q, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}

	return datatypes.JSONMap{
		"id":           row.ID,
		"masjid_id":    row.MasjidID,
		"name":         row.Name,
		"whatsapp_url": row.Whatsapp,
		"title_prefix": row.TitlePre,
		"title_suffix": row.TitleSuf,
		"captured_at":  time.Now().UTC(),
		"source":       "generator_v2",
	}, nil
}

// v2: Room snapshot adaptif (slug/location bila ada)
func (g *Generator) loadRoomSnapshot(ctx context.Context, id uuid.UUID) (datatypes.JSONMap, error) {
	cols, err := g.tableColumns(ctx, "class_rooms")
	if err != nil {
		return nil, err
	}
	idCol := firstExisting(cols, "class_room_id", "id")
	masjidCol := firstExisting(cols, "class_room_masjid_id", "masjid_id")
	nameCol := firstExisting(cols, "class_room_name", "name")
	codeCol := firstExisting(cols, "class_room_code", "code")
	capCol := firstExisting(cols, "class_room_capacity", "capacity")
	slugCol := firstExisting(cols, "class_room_slug", "slug")
	locCol := firstExisting(cols, "class_room_location", "location")
	deletedCol := firstExisting(cols, "class_room_deleted_at", "deleted_at")

	if idCol == "" || masjidCol == "" || nameCol == "" {
		return nil, fmt.Errorf("loadRoomSnapshot: kolom minimal (id/masjid_id/name) tidak ditemukan")
	}

	codeExpr := "NULL"
	if codeCol != "" {
		codeExpr = fmt.Sprintf("r.%s", codeCol)
	}
	capExpr := "NULL"
	if capCol != "" {
		capExpr = fmt.Sprintf("r.%s", capCol)
	}
	slugExpr := "NULL"
	if slugCol != "" {
		slugExpr = fmt.Sprintf("r.%s", slugCol)
	}
	locExpr := "NULL"
	if locCol != "" {
		locExpr = fmt.Sprintf("r.%s", locCol)
	}

	whereDeleted := ""
	if deletedCol != "" {
		whereDeleted = fmt.Sprintf(" AND r.%s IS NULL", deletedCol)
	}

	q := fmt.Sprintf(`
SELECT
  r.%s AS class_room_id,
  r.%s AS class_room_masjid_id,
  r.%s AS class_room_name,
  %s   AS class_room_code,
  %s   AS class_room_capacity,
  %s   AS class_room_slug,
  %s   AS class_room_location
FROM class_rooms r
WHERE r.%s = ?
%s
LIMIT 1`,
		idCol, masjidCol, nameCol,
		codeExpr, capExpr, slugExpr, locExpr,
		idCol, whereDeleted,
	)

	var row struct {
		ID       uuid.UUID `gorm:"column:class_room_id"`
		MasjidID uuid.UUID `gorm:"column:class_room_masjid_id"`
		Name     *string   `gorm:"column:class_room_name"`
		Code     *string   `gorm:"column:class_room_code"`
		Capacity *int      `gorm:"column:class_room_capacity"`
		Slug     *string   `gorm:"column:class_room_slug"`
		Location *string   `gorm:"column:class_room_location"`
	}
	if err := g.DB.WithContext(ctx).Raw(q, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return datatypes.JSONMap{
		"id":          row.ID,
		"masjid_id":   row.MasjidID,
		"name":        row.Name,
		"code":        row.Code,
		"capacity":    row.Capacity,
		"slug":        row.Slug,     // baru, jika kolom ada
		"location":    row.Location, // baru, jika kolom ada
		"captured_at": time.Now().UTC(),
		"source":      "generator_v2",
	}, nil
}

// ============================
// Helpers (schema detection)
// ============================

func (g *Generator) tableColumns(ctx context.Context, table string) (map[string]struct{}, error) {
	type colRow struct {
		ColumnName string `gorm:"column:column_name"`
	}
	var rows []colRow

	q := `
SELECT column_name
FROM information_schema.columns
WHERE table_name = ?
  AND table_schema = ANY (current_schemas(true))`
	if err := g.DB.WithContext(ctx).Raw(q, table).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		out[strings.ToLower(strings.TrimSpace(r.ColumnName))] = struct{}{}
	}
	return out, nil
}

func firstExisting(cols map[string]struct{}, candidates ...string) string {
	for _, c := range candidates {
		if _, ok := cols[strings.ToLower(c)]; ok {
			return c
		}
	}
	return ""
}

// ============================
// Helpers (waktu & rule)
// ============================

func stringsTrimLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// parse "HH:mm[:ss]" ke time.Time (tanggal dummy) basis UTC
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
func toUTC(t time.Time) time.Time { return t.In(time.UTC) }

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
