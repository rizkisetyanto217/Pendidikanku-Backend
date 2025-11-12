// file: internals/features/school/sessions/schedules/service/generate_sessions.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	roomModel "schoolku_backend/internals/features/school/academics/rooms/model"
	sessModel "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
	schedModel "schoolku_backend/internals/features/school/classes/class_schedules/model"

	// Paket snapshot lama sebagai fallback
	snapshotCSST "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/snapshot"
)

/* =========================
   Utils (string & slug)
========================= */

func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

func stringsTrimLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func slugifySimple(s string) string {
	s = stringsTrimLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
			continue
		}
		if unicode.IsSpace(r) || r == '_' || r == '-' || r == '/' {
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
			continue
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "room"
	}
	return out
}

func ptr[T any](v T) *T { return &v }

/* =========================
   Generator + Options
========================= */

type Generator struct{ DB *gorm.DB }

type GenerateOptions struct {
	TZName                  string
	DefaultCSSTID           *uuid.UUID
	DefaultRoomID           *uuid.UUID
	DefaultTeacherID        *uuid.UUID
	DefaultAttendanceStatus string
	BatchSize               int
}

/* =========================
   Helper selector dinamis
========================= */

// nyusun COALESCE dinamis dari kandidat kolom yang eksis
func sel(cols map[string]struct{}, tableAlias string, candidates ...string) string {
	parts := []string{}
	for _, c := range candidates {
		if _, ok := cols[strings.ToLower(c)]; ok {
			parts = append(parts, fmt.Sprintf("%s.%s", tableAlias, c))
		}
	}
	if len(parts) == 0 {
		return "NULL"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("COALESCE(%s)", strings.Join(parts, ","))
}

/* =========================
   CSST rich loader
========================= */

func (g *Generator) getCSSTRich(ctx context.Context, expectSchool uuid.UUID, csstID uuid.UUID) (*csstRich, error) {
	// Deteksi kolom per tabel
	csstCols, _ := g.tableColumns(ctx, "class_section_subject_teachers")
	subjCols, _ := g.tableColumns(ctx, "subjects")
	secCols, _ := g.tableColumns(ctx, "class_sections")
	teaCols, _ := g.tableColumns(ctx, "school_teachers")
	roomCols, _ := g.tableColumns(ctx, "class_rooms")
	bookCols, _ := g.tableColumns(ctx, "class_subject_books")

	// kunci
	idCol := firstExisting(csstCols, "class_section_subject_teacher_id", "id")
	schoolCol := firstExisting(csstCols, "class_section_subject_teacher_school_id", "school_id")
	if idCol == "" || schoolCol == "" {
		return nil, fmt.Errorf("CSST: kolom id/school_id tidak ditemukan")
	}

	deletedCol := firstExisting(csstCols, "class_section_subject_teacher_deleted_at", "deleted_at")
	whereDeleted := ""
	if deletedCol != "" {
		whereDeleted = fmt.Sprintf(" AND csst.%s IS NULL", deletedCol)
	}

	// === SLUG & NAME dasar ===
	slugExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_slug", "class_section_subject_teacher_code", "slug", "code")
	nameExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_name", "name", "title", "label")

	// === SUBJECT === (COALESCE ke *_snapshot bila JOIN kosong)
	subjectIDExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_subject_id", "subject_id",
		"class_section_subject_teacher_subject_id_snapshot")
	subjectCodeExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(subjCols, "subj", "subject_code", "code"),
		sel(csstCols, "csst", "class_section_subject_teacher_subject_code_snapshot"))
	subjectNameExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(subjCols, "subj", "subject_name", "name", "title"),
		sel(csstCols, "csst", "class_section_subject_teacher_subject_name_snapshot"))
	subjectSlugExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_subject_slug_snapshot")

	// === SECTION ===
	sectionIDExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_class_section_id", "class_section_id", "section_id")
	sectionNameExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(secCols, "sec", "class_section_name", "name", "title"),
		sel(csstCols, "csst", "class_section_subject_teacher_class_section_name_snapshot"))

	// === TEACHER ===
	teacherIDExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_school_teacher_id", "school_teacher_id", "teacher_id")
	teacherCodeExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(teaCols, "tea", "school_teacher_code", "teacher_code", "code"),
		sel(csstCols, "csst", "class_section_subject_teacher_school_teacher_code_snapshot"))
	teacherNameExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(teaCols, "tea", "school_teacher_name", "teacher_name", "name", "full_name"),
		sel(csstCols, "csst", "class_section_subject_teacher_school_teacher_name_snapshot"))
	teacherSnapExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_school_teacher_snapshot")

	// === ROOM ===
	roomIDExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_class_room_id", "class_room_id", "room_id")
	roomCodeExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(roomCols, "room", "class_room_code", "room_code", "code"),
		sel(csstCols, "csst", "class_section_subject_teacher_class_room_code_snapshot"))
	roomNameExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(roomCols, "room", "class_room_name", "room_name", "name", "title"),
		sel(csstCols, "csst", "class_section_subject_teacher_class_room_name_snapshot"))

	// === BOOK ===
	bookIDExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_class_subject_book_id", "class_subject_book_id", "book_id")
	bookCodeExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(bookCols, "book", "class_subject_book_code", "book_code", "code"),
		sel(csstCols, "csst", "class_section_subject_teacher_book_code_snapshot"))
	bookNameExpr := fmt.Sprintf("COALESCE(%s, %s)",
		sel(bookCols, "book", "class_subject_book_name", "book_name", "name", "title"),
		sel(csstCols, "csst", "class_section_subject_teacher_book_title_snapshot"))
	bookSnapExpr := sel(csstCols, "csst",
		"class_section_subject_teacher_class_subject_book_snapshot")

	// Join ON enable/disable
	q := fmt.Sprintf(`
SELECT
  csst.%s AS id,
  csst.%s AS school_id,
  COALESCE(%s, csst.%s::text) AS slug,
  %s       AS name,

  %s AS subject_id,
  %s AS subject_code,
  %s AS subject_name,
  %s AS subject_slug,

  %s AS section_id,
  %s AS section_name,

  %s AS teacher_id,
  %s AS teacher_code,
  %s AS teacher_name,

  %s AS room_id,
  %s AS room_code,
  %s AS room_name,

  %s AS class_subject_book_id,
  %s AS book_code,
  %s AS book_name,

  %s AS teacher_snapshot,
  %s AS book_snapshot
FROM class_section_subject_teachers csst
LEFT JOIN subjects subj
  ON %s AND subj.subject_id = %s
LEFT JOIN class_sections sec
  ON %s AND sec.class_section_id = %s
LEFT JOIN school_teachers tea
  ON %s AND tea.school_teacher_id = %s
LEFT JOIN class_rooms room
  ON %s AND room.class_room_id = %s
LEFT JOIN class_subject_books book
  ON %s AND book.class_subject_book_id = %s
WHERE csst.%s = ?%s
LIMIT 1`,
		// heads
		idCol, schoolCol,
		slugExpr, idCol,
		nameExpr,

		subjectIDExpr, subjectCodeExpr, subjectNameExpr, subjectSlugExpr,
		sectionIDExpr, sectionNameExpr,
		teacherIDExpr, teacherCodeExpr, teacherNameExpr,
		roomIDExpr, roomCodeExpr, roomNameExpr,
		bookIDExpr, bookCodeExpr, bookNameExpr,
		teacherSnapExpr, bookSnapExpr,

		func() string {
			if subjectIDExpr != "NULL" {
				return "TRUE"
			} else {
				return "FALSE"
			}
		}(), subjectIDExpr,
		func() string {
			if sectionIDExpr != "NULL" {
				return "TRUE"
			} else {
				return "FALSE"
			}
		}(), sectionIDExpr,
		func() string {
			if teacherIDExpr != "NULL" {
				return "TRUE"
			} else {
				return "FALSE"
			}
		}(), teacherIDExpr,
		func() string {
			if roomIDExpr != "NULL" {
				return "TRUE"
			} else {
				return "FALSE"
			}
		}(), roomIDExpr,
		func() string {
			if bookIDExpr != "NULL" {
				return "TRUE"
			} else {
				return "FALSE"
			}
		}(), bookIDExpr,
		idCol, whereDeleted,
	)

	var row csstRich
	if err := g.DB.WithContext(ctx).Raw(q, csstID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	if expectSchool != uuid.Nil && row.SchoolID != expectSchool {
		return nil, fmt.Errorf("tenant mismatch csst.school=%s != expect=%s", row.SchoolID, expectSchool)
	}
	if strings.TrimSpace(row.Slug) == "" {
		row.Slug = row.ID.String()
	}

	// Hydrate dari JSON snapshot bila ada kekosongan
	hydrateFromSnapshots(&row)

	return &row, nil
}

type csstRich struct {
	ID       uuid.UUID `gorm:"column:id"`
	SchoolID uuid.UUID `gorm:"column:school_id"`
	Slug     string    `gorm:"column:slug"`
	Name     *string   `gorm:"column:name"`

	// Subject
	SubjectID   *uuid.UUID `gorm:"column:subject_id"`
	SubjectCode *string    `gorm:"column:subject_code"`
	SubjectName *string    `gorm:"column:subject_name"`
	SubjectSlug *string    `gorm:"column:subject_slug"`

	// Section
	SectionID   *uuid.UUID `gorm:"column:section_id"`
	SectionName *string    `gorm:"column:section_name"`

	// Teacher
	TeacherID       *uuid.UUID `gorm:"column:teacher_id"`
	TeacherCode     *string    `gorm:"column:teacher_code"`
	TeacherName     *string    `gorm:"column:teacher_name"`
	TeacherSnapshot *string    `gorm:"column:teacher_snapshot"` // raw JSON

	// Room
	RoomID   *uuid.UUID `gorm:"column:room_id"`
	RoomCode *string    `gorm:"column:room_code"`
	RoomName *string    `gorm:"column:room_name"`

	// Book
	BookID       *uuid.UUID `gorm:"column:class_subject_book_id"`
	BookCode     *string    `gorm:"column:book_code"`
	BookName     *string    `gorm:"column:book_name"`
	BookSnapshot *string    `gorm:"column:book_snapshot"` // raw JSON
}

// Lengkapi field dari JSON snapshot bila join/kolom snapshot string kosong
func hydrateFromSnapshots(row *csstRich) {
	if row == nil {
		return
	}

	// Teacher JSON: {"id":"...","name":"...","title_prefix":"Ustadz","title_suffix":"Lc",...}
	if (row.TeacherName == nil || strings.TrimSpace(*row.TeacherName) == "") &&
		row.TeacherSnapshot != nil && *row.TeacherSnapshot != "" && *row.TeacherSnapshot != "null" {
		var m map[string]any
		if json.Unmarshal([]byte(*row.TeacherSnapshot), &m) == nil {
			if v, ok := m["name"].(string); ok && strings.TrimSpace(v) != "" {
				pre, _ := m["title_prefix"].(string)
				suf, _ := m["title_suffix"].(string)
				full := strings.TrimSpace(strings.TrimSpace(pre+" ") + v)
				if strings.TrimSpace(suf) != "" {
					full = strings.TrimSpace(full + " " + suf)
				}
				row.TeacherName = ptr(full)
			}
			if row.TeacherID == nil {
				if v, ok := m["id"].(string); ok {
					if u, er := uuid.Parse(v); er == nil {
						row.TeacherID = &u
					}
				}
			}
			if row.TeacherCode == nil {
				if v, ok := m["code"].(string); ok && strings.TrimSpace(v) != "" {
					row.TeacherCode = &v
				}
			}
		}
	}

	// Book JSON: {"book": {...}, "subject": {"id","code","name","slug"}}
	if row.BookSnapshot != nil && *row.BookSnapshot != "" && *row.BookSnapshot != "null" {
		var m map[string]any
		if json.Unmarshal([]byte(*row.BookSnapshot), &m) == nil {
			if subj, ok := m["subject"].(map[string]any); ok {
				if row.SubjectID == nil {
					if v, ok := subj["id"].(string); ok {
						if u, er := uuid.Parse(v); er == nil {
							row.SubjectID = &u
						}
					}
				}
				if row.SubjectCode == nil {
					if v, ok := subj["code"].(string); ok && strings.TrimSpace(v) != "" {
						row.SubjectCode = &v
					}
				}
				if row.SubjectName == nil {
					if v, ok := subj["name"].(string); ok && strings.TrimSpace(v) != "" {
						row.SubjectName = &v
					}
				}
				if row.SubjectSlug == nil {
					if v, ok := subj["slug"].(string); ok && strings.TrimSpace(v) != "" {
						row.SubjectSlug = &v
					}
				}
			}
			if book, ok := m["book"].(map[string]any); ok {
				if row.BookName == nil {
					if v, ok := book["title"].(string); ok && strings.TrimSpace(v) != "" {
						row.BookName = &v
					}
				}
				if row.BookCode == nil {
					if v, ok := book["code"].(string); ok && strings.TrimSpace(v) != "" {
						row.BookCode = &v
					}
				}
				if row.BookID == nil {
					if v, ok := book["id"].(string); ok {
						if u, er := uuid.Parse(v); er == nil {
							row.BookID = &u
						}
					}
				}
			}
		}
	}
}

/* =========================
   Snapshot builder
========================= */

func putStr(m datatypes.JSONMap, key string, v *string) {
	if v != nil && strings.TrimSpace(*v) != "" {
		m[key] = strings.TrimSpace(*v)
	}
}
func putUUID(m datatypes.JSONMap, key string, v *uuid.UUID) {
	if v != nil && *v != uuid.Nil {
		m[key] = v.String()
	}
}

func (g *Generator) buildCSSTSnapshotJSON(
	ctx context.Context,
	expectSchoolID uuid.UUID,
	csstID uuid.UUID,
) (datatypes.JSONMap, *uuid.UUID, *string, error) {
	// a) RICH loader
	if rich, err := g.getCSSTRich(ctx, expectSchoolID, csstID); err == nil && rich != nil {
		out := datatypes.JSONMap{
			"csst_id":   rich.ID.String(),
			"school_id": rich.SchoolID.String(),
			"slug":      rich.Slug,
		}
		if rich.Name != nil && strings.TrimSpace(*rich.Name) != "" {
			out["name"] = strings.TrimSpace(*rich.Name)
		}

		// Subject
		putUUID(out, "subject_id", rich.SubjectID)
		putStr(out, "subject_code", rich.SubjectCode)
		putStr(out, "subject_name", rich.SubjectName)
		putStr(out, "subject_slug", rich.SubjectSlug) // NEW

		// Section
		putUUID(out, "section_id", rich.SectionID)
		putStr(out, "section_name", rich.SectionName)

		// Teacher
		putUUID(out, "teacher_id", rich.TeacherID)
		putStr(out, "teacher_code", rich.TeacherCode)
		putStr(out, "teacher_name", rich.TeacherName)

		// Room
		putUUID(out, "room_id", rich.RoomID)
		putStr(out, "room_code", rich.RoomCode)
		putStr(out, "room_name", rich.RoomName)

		// Book
		putUUID(out, "class_subject_book_id", rich.BookID)
		putStr(out, "class_subject_book_code", rich.BookCode)
		putStr(out, "class_subject_book_name", rich.BookName)

		return out, rich.TeacherID, rich.Name, nil
	}

	// b) Fallback paket snapshot lama
	if s, tid, name, err := func() (datatypes.JSONMap, *uuid.UUID, *string, error) {
		tx := g.DB.WithContext(ctx)
		cs, er := snapshotCSST.ValidateAndSnapshotCSST(tx, expectSchoolID, csstID)
		if er != nil {
			return nil, nil, nil, er
		}
		j := snapshotCSST.ToJSON(cs)
		var m map[string]any
		if er := json.Unmarshal(j, &m); er != nil {
			return nil, nil, nil, er
		}
		return datatypes.JSONMap(m), cs.TeacherID, cs.Name, nil
	}(); err == nil && s != nil {
		return s, tid, name, nil
	}

	// c) Minimal
	min := datatypes.JSONMap{
		"csst_id":   csstID.String(),
		"school_id": expectSchoolID.String(),
	}
	return min, nil, nil, nil
}

/* =========================
   Rule rows & section lite
========================= */

type ruleRow struct {
	ID              uuid.UUID     `gorm:"column:class_schedule_rule_id"`
	SchoolID        uuid.UUID     `gorm:"column:class_schedule_rule_school_id"`
	ScheduleID      uuid.UUID     `gorm:"column:class_schedule_rule_schedule_id"`
	DayOfWeek       int           `gorm:"column:class_schedule_rule_day_of_week"`
	StartStr        string        `gorm:"column:start_str"`
	EndStr          string        `gorm:"column:end_str"`
	IntervalWeeks   int           `gorm:"column:class_schedule_rule_interval_weeks"`
	StartOffset     int           `gorm:"column:class_schedule_rule_start_offset_weeks"`
	WeekParity      string        `gorm:"column:class_schedule_rule_week_parity"`
	WeeksOfMonth    pq.Int64Array `gorm:"column:class_schedule_rule_weeks_of_month"`
	LastWeekOfMonth bool          `gorm:"column:class_schedule_rule_last_week_of_month"`
	CSSTID          *uuid.UUID    `gorm:"column:class_schedule_rule_csst_id"`
}

type sectionLite struct {
	ID       uuid.UUID `gorm:"column:class_section_id"`
	SchoolID uuid.UUID `gorm:"column:class_section_school_id"`
	Name     *string   `gorm:"column:class_section_name"`
}

/* =========================
   Public API
========================= */

func (g *Generator) GenerateSessionsForSchedule(ctx context.Context, scheduleID string) (int, error) {
	return g.GenerateSessionsForScheduleWithOpts(ctx, scheduleID, nil)
}

func (g *Generator) GenerateSessionsForScheduleWithOpts(ctx context.Context, scheduleID string, opts *GenerateOptions) (created int, err error) {
	// Defaults
	if opts == nil {
		opts = &GenerateOptions{}
	}
	if opts.TZName == "" {
		opts.TZName = "Asia/Jakarta"
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 500
	}
	attendanceDefault := sessModel.AttendanceStatusOpen
	if s := stringsTrimLower(opts.DefaultAttendanceStatus); s != "" {
		attendanceDefault = sessModel.AttendanceStatus(s)
	}
	loc, err := time.LoadLocation(opts.TZName)
	if err != nil {
		loc = time.FixedZone("Asia/Jakarta", 7*3600)
	}

	// 1) Ambil schedule
	var sch schedModel.ClassScheduleModel
	if err = g.DB.WithContext(ctx).
		Where("class_schedule_id = ?", scheduleID).
		Take(&sch).Error; err != nil {
		return 0, err
	}
	startLocal := startOfDayInLoc(sch.ClassScheduleStartDate, loc)
	endLocal := startOfDayInLoc(sch.ClassScheduleEndDate, loc)
	if endLocal.Before(startLocal) {
		return 0, nil
	}

	// 2) Ambil rules (+ CSST per-rule bila ada)
	cRules, _ := g.tableColumns(ctx, "class_schedule_rules")
	csstSelect := "NULL::uuid"
	if _, ok := cRules["class_schedule_rule_csst_id"]; ok {
		csstSelect = "class_schedule_rule_csst_id"
	}
	var rr []ruleRow
	qRules := fmt.Sprintf(`
SELECT
  class_schedule_rule_id,
  class_schedule_rule_school_id,
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

	// 2.5) Preload default CSST snapshot (opsional)
	var (
		defCSSTSnap       datatypes.JSONMap
		teacherIDFromCSST *uuid.UUID
		defCSSTName       *string
	)
	if opts.DefaultCSSTID != nil {
		if s, tid, name, er := g.buildCSSTSnapshotJSON(ctx, sch.ClassScheduleSchoolID, *opts.DefaultCSSTID); er == nil {
			defCSSTSnap = s
			teacherIDFromCSST = tid
			defCSSTName = name
		} else {
			// tetap siapkan minimal supaya tidak null
			defCSSTSnap = datatypes.JSONMap{
				"csst_id":   opts.DefaultCSSTID.String(),
				"school_id": sch.ClassScheduleSchoolID.String(),
			}
		}
	}

	// Caches
	csstSnapCache := map[uuid.UUID]datatypes.JSONMap{}
	csstTeacherIDCache := map[uuid.UUID]*uuid.UUID{}
	csstNameCache := map[uuid.UUID]*string{}
	meetingCountByCSST := map[uuid.UUID]int{}

	// 3) Expand occurrences
	rows := make([]sessModel.ClassAttendanceSessionModel, 0, 1024)

	attachSnapshots := func(row *sessModel.ClassAttendanceSessionModel, ruleCSST *uuid.UUID) {
		// --- CSST (per-rule > default) ---
		var effCSST *uuid.UUID
		var effCSSTSnap datatypes.JSONMap
		var effTeacherFromCSST *uuid.UUID
		var baseName *string

		if ruleCSST != nil {
			effCSST = ruleCSST
			if s, ok := csstSnapCache[*ruleCSST]; ok {
				effCSSTSnap = s
				effTeacherFromCSST = csstTeacherIDCache[*ruleCSST]
				baseName = csstNameCache[*ruleCSST]
			} else if s, tid, name, er := g.buildCSSTSnapshotJSON(ctx, sch.ClassScheduleSchoolID, *ruleCSST); er == nil {
				csstSnapCache[*ruleCSST] = s
				csstTeacherIDCache[*ruleCSST] = tid
				csstNameCache[*ruleCSST] = name
				effCSSTSnap = s
				effTeacherFromCSST = tid
				baseName = name
			} else {
				// fallback minimal
				s := datatypes.JSONMap{
					"csst_id":   ruleCSST.String(),
					"school_id": sch.ClassScheduleSchoolID.String(),
				}
				csstSnapCache[*ruleCSST] = s
				effCSSTSnap = s
			}
		} else if opts.DefaultCSSTID != nil {
			effCSST = opts.DefaultCSSTID
			effCSSTSnap = defCSSTSnap
			effTeacherFromCSST = teacherIDFromCSST
			baseName = defCSSTName
			if effCSSTSnap == nil {
				effCSSTSnap = datatypes.JSONMap{
					"csst_id":   opts.DefaultCSSTID.String(),
					"school_id": sch.ClassScheduleSchoolID.String(),
				}
			}
		}

		// Simpan CSST + snapshot CSST (wajib ada minimal map)
		if effCSST != nil {
			row.ClassAttendanceSessionCSSTID = effCSST
			if effCSSTSnap == nil {
				effCSSTSnap = datatypes.JSONMap{
					"csst_id":   effCSST.String(),
					"school_id": sch.ClassScheduleSchoolID.String(),
				}
			}
			row.ClassAttendanceSessionCSSTSnapshot = effCSSTSnap
		}

		// TeacherID (tanpa snapshot)
		if opts.DefaultTeacherID != nil {
			row.ClassAttendanceSessionTeacherID = opts.DefaultTeacherID
		} else if effTeacherFromCSST != nil {
			row.ClassAttendanceSessionTeacherID = effTeacherFromCSST
		}

		// RoomID (tanpa snapshot)
		if opts.DefaultRoomID != nil {
			row.ClassAttendanceSessionClassRoomID = opts.DefaultRoomID
		} else if ruleCSST != nil {
			if rid, _, er := g.ResolveRoomFromCSSTOrSection(ctx, sch.ClassScheduleSchoolID, ruleCSST); er == nil && rid != nil {
				row.ClassAttendanceSessionClassRoomID = rid
			}
		}

		// Title otomatis "<nama CSST> pertemuan ke-N"
		if baseName != nil && strings.TrimSpace(*baseName) != "" && effCSST != nil {
			key := *effCSST
			meetingCountByCSST[key] = meetingCountByCSST[key] + 1
			n := meetingCountByCSST[key]
			title := fmt.Sprintf("%s pertemuan ke-%d", strings.TrimSpace(*baseName), n)
			row.ClassAttendanceSessionTitle = &title
		}
	}

	// Tanpa rule → satu sesi di start date
	if len(rr) == 0 {
		dateUTC := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, time.UTC)
		row := sessModel.ClassAttendanceSessionModel{
			ClassAttendanceSessionSchoolID:         sch.ClassScheduleSchoolID,
			ClassAttendanceSessionScheduleID:       ptrUUID(sch.ClassScheduleID),
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
		attachSnapshots(&row, nil)
		rows = append(rows, row)
	} else {
		// Dengan rules
		for d := startLocal; !d.After(endLocal); d = d.AddDate(0, 0, 1) {
			for _, r := range rr {
				if !dateMatchesRuleRow(d, startLocal, r) {
					continue
				}
				stTOD, err1 := parseTODString(r.StartStr)
				etTOD, err2 := parseTODString(r.EndStr)
				if err1 != nil || err2 != nil {
					continue
				}
				startAtLocal := combineLocalDateAndTOD(d, stTOD, loc)
				endAtLocal := combineLocalDateAndTOD(d, etTOD, loc)
				if endAtLocal.Before(startAtLocal) {
					endAtLocal = endAtLocal.Add(24 * time.Hour)
				}
				startAtUTC := toUTC(startAtLocal)
				endAtUTC := toUTC(endAtLocal)
				dateUTC := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
				rid := r.ID

				row := sessModel.ClassAttendanceSessionModel{
					ClassAttendanceSessionSchoolID:         sch.ClassScheduleSchoolID,
					ClassAttendanceSessionScheduleID:       ptrUUID(sch.ClassScheduleID),
					ClassAttendanceSessionRuleID:           &rid,
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
				attachSnapshots(&row, r.CSSTID)
				rows = append(rows, row)
			}
		}
	}

	if len(rows) == 0 {
		return 0, nil
	}

	// 4) Idempotent insert (batch)
	db := g.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true})
	tx := db.CreateInBatches(rows, opts.BatchSize)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return int(tx.RowsAffected), nil
}

/* =========================
   Room resolver & helpers
========================= */

func coalesceSchool(a, b uuid.UUID) uuid.UUID {
	if a != uuid.Nil {
		return a
	}
	return b
}

func (g *Generator) ResolveRoomFromCSSTOrSection(
	ctx context.Context,
	expectSchoolID uuid.UUID,
	csstID *uuid.UUID,
) (*uuid.UUID, datatypes.JSONMap, error) {
	if csstID == nil || *csstID == uuid.Nil {
		return nil, nil, nil
	}

	roomID, sectionID, csstSchoolID, sectionName, err := g.getRoomOrSectionFromCSST(ctx, *csstID)
	if err != nil {
		return nil, nil, err
	}
	if expectSchoolID != uuid.Nil && csstSchoolID != uuid.Nil && expectSchoolID != csstSchoolID {
		return nil, nil, fmt.Errorf("tenant mismatch: csst.school=%s != expect=%s", csstSchoolID, expectSchoolID)
	}

	// 1) Room pada CSST
	if roomID != nil && *roomID != uuid.Nil {
		return roomID, nil, nil
	}

	// 2) Room pada Section
	if sectionID != nil && *sectionID != uuid.Nil {
		rid, secSchoolID, _, er := g.getRoomFromSection(ctx, *sectionID)
		if er == nil {
			if rid != nil && *rid != uuid.Nil {
				return rid, nil, nil
			}
			// 3) Auto-provision
			useSchool := coalesceSchool(secSchoolID, csstSchoolID)
			rid, _, er2 := g.ensureSectionRoom(ctx, useSchool, *sectionID, sectionName)
			return rid, nil, er2
		}
		if csstSchoolID != uuid.Nil {
			rid, _, er3 := g.ensureSectionRoom(ctx, csstSchoolID, *sectionID, sectionName)
			return rid, nil, er3
		}
		return nil, nil, err
	}

	// 3) CSST tidak punya Section
	return nil, nil, nil
}

func (g *Generator) getRoomOrSectionFromCSST(ctx context.Context, csst uuid.UUID) (roomID *uuid.UUID, sectionID *uuid.UUID, schoolID uuid.UUID, sectionName *string, err error) {
	cols, err := g.tableColumns(ctx, "class_section_subject_teachers")
	if err != nil {
		return nil, nil, uuid.Nil, nil, err
	}
	idCol := firstExisting(cols, "class_section_subject_teacher_id", "id")
	schoolCol := firstExisting(cols, "class_section_subject_teacher_school_id", "school_id")
	roomCol := firstExisting(cols, "class_section_subject_teacher_room_id", "class_room_id", "room_id")
	sectionCol := firstExisting(cols, "class_section_subject_teacher_section_id", "class_section_id", "section_id")
	deletedCol := firstExisting(cols, "class_section_subject_teacher_deleted_at", "deleted_at")

	if idCol == "" || schoolCol == "" {
		return nil, nil, uuid.Nil, nil, fmt.Errorf("getRoomOrSectionFromCSST: kolom minimal (id/school_id) tidak ditemukan di CSST")
	}

	roomExpr := "NULL::uuid"
	if roomCol != "" {
		roomExpr = fmt.Sprintf("csst.%s", roomCol)
	}
	secExpr := "NULL::uuid"
	if sectionCol != "" {
		secExpr = fmt.Sprintf("csst.%s", sectionCol)
	}
	whereDeleted := ""
	if deletedCol != "" {
		whereDeleted = fmt.Sprintf(" AND csst.%s IS NULL", deletedCol)
	}

	q := fmt.Sprintf(`
SELECT
  csst.%s AS csst_id,
  csst.%s AS school_id,
  %s      AS room_id,
  %s      AS section_id
FROM class_section_subject_teachers csst
WHERE csst.%s = ?
%s
LIMIT 1`, idCol, schoolCol, roomExpr, secExpr, idCol, whereDeleted)

	var row struct {
		CSST     uuid.UUID  `gorm:"column:csst_id"`
		SchoolID uuid.UUID  `gorm:"column:school_id"`
		RoomID   *uuid.UUID `gorm:"column:room_id"`
		Section  *uuid.UUID `gorm:"column:section_id"`
	}
	if er := g.DB.WithContext(ctx).Raw(q, csst).Scan(&row).Error; er != nil {
		return nil, nil, uuid.Nil, nil, er
	}

	var secName *string
	if row.Section != nil && *row.Section != uuid.Nil {
		if s, er := g.getSectionNameAndSchool(ctx, *row.Section); er == nil {
			secName = s.Name
			if row.SchoolID == uuid.Nil {
				row.SchoolID = s.SchoolID
			}
		}
	}

	return row.RoomID, row.Section, row.SchoolID, secName, nil
}

func (g *Generator) getRoomFromSection(ctx context.Context, sectionID uuid.UUID) (roomID *uuid.UUID, schoolID uuid.UUID, name *string, err error) {
	cols, err := g.tableColumns(ctx, "class_sections")
	if err != nil {
		return nil, uuid.Nil, nil, err
	}
	idCol := firstExisting(cols, "class_section_id", "id")
	schoolCol := firstExisting(cols, "class_section_school_id", "school_id")
	nameCol := firstExisting(cols, "class_section_name", "name")
	roomCol := firstExisting(cols, "class_section_room_id", "class_room_id", "room_id")
	deletedCol := firstExisting(cols, "class_section_deleted_at", "deleted_at")

	if idCol == "" || schoolCol == "" {
		return nil, uuid.Nil, nil, fmt.Errorf("getRoomFromSection: kolom minimal (id/school_id) tidak ditemukan")
	}

	roomExpr := "NULL::uuid"
	if roomCol != "" {
		roomExpr = fmt.Sprintf("s.%s", roomCol)
	}
	nameExpr := "NULL::text"
	if nameCol != "" {
		nameExpr = fmt.Sprintf("s.%s", nameCol)
	}
	whereDeleted := ""
	if deletedCol != "" {
		whereDeleted = fmt.Sprintf(" AND s.%s IS NULL", deletedCol)
	}

	q := fmt.Sprintf(`
SELECT
  s.%s AS class_section_id,
  s.%s AS class_section_school_id,
  %s   AS class_section_name,
  %s   AS class_room_id
FROM class_sections s
WHERE s.%s = ?
%s
LIMIT 1`, idCol, schoolCol, nameExpr, roomExpr, idCol, whereDeleted)

	var row struct {
		ID       uuid.UUID  `gorm:"column:class_section_id"`
		SchoolID uuid.UUID  `gorm:"column:class_section_school_id"`
		Name     *string    `gorm:"column:class_section_name"`
		RoomID   *uuid.UUID `gorm:"column:class_room_id"`
	}
	if er := g.DB.WithContext(ctx).Raw(q, sectionID).Scan(&row).Error; er != nil {
		return nil, uuid.Nil, nil, er
	}
	return row.RoomID, row.SchoolID, row.Name, nil
}

func (g *Generator) getSectionNameAndSchool(ctx context.Context, sectionID uuid.UUID) (*sectionLite, error) {
	cols, err := g.tableColumns(ctx, "class_sections")
	if err != nil {
		return nil, err
	}
	idCol := firstExisting(cols, "class_section_id", "id")
	schoolCol := firstExisting(cols, "class_section_school_id", "school_id")
	nameCol := firstExisting(cols, "class_section_name", "name")
	if idCol == "" || schoolCol == "" {
		return nil, fmt.Errorf("getSectionNameAndSchool: kolom minimal tidak ditemukan")
	}
	nameExpr := "NULL::text"
	if nameCol != "" {
		nameExpr = fmt.Sprintf("s.%s", nameCol)
	}
	q := fmt.Sprintf(`
SELECT
  s.%s AS class_section_id,
  s.%s AS class_section_school_id,
  %s   AS class_section_name
FROM class_sections s
WHERE s.%s = ?
LIMIT 1`, idCol, schoolCol, nameExpr, idCol)

	var row sectionLite
	if er := g.DB.WithContext(ctx).Raw(q, sectionID).Scan(&row).Error; er != nil {
		return nil, er
	}
	return &row, nil
}

// Buat Room default untuk Section (bila belum ada sama sekali).
// Snapshot ruang tidak lagi digunakan → selalu return snapshot = nil.
func (g *Generator) ensureSectionRoom(
	ctx context.Context,
	schoolID uuid.UUID,
	sectionID uuid.UUID,
	sectionName *string,
) (*uuid.UUID, datatypes.JSONMap, error) {
	if schoolID == uuid.Nil || sectionID == uuid.Nil {
		return nil, nil, fmt.Errorf("ensureSectionRoom: schoolID/sectionID kosong")
	}

	baseName := "Ruang"
	if sectionName != nil && strings.TrimSpace(*sectionName) != "" {
		baseName = fmt.Sprintf("Ruang %s", strings.TrimSpace(*sectionName))
	}
	slug := fmt.Sprintf("section-%s", strings.ReplaceAll(sectionID.String(), "-", "")) // unik per section

	cols, err := g.tableColumns(ctx, "class_rooms")
	if err != nil {
		return nil, nil, err
	}
	hasSlug := firstExisting(cols, "class_room_slug", "slug") != ""

	var existing roomModel.ClassRoomModel
	if hasSlug {
		if er := g.DB.WithContext(ctx).
			Where("class_room_school_id = ? AND class_room_slug = ? AND class_room_deleted_at IS NULL", schoolID, slug).
			Limit(1).
			Take(&existing).Error; er == nil && existing.ClassRoomID != uuid.Nil {
			id := existing.ClassRoomID
			return &id, nil, nil
		}
	}

	cr := roomModel.ClassRoomModel{
		ClassRoomSchoolID:  schoolID,
		ClassRoomName:      baseName,
		ClassRoomIsVirtual: false,
		ClassRoomIsActive:  true,
	}
	if hasSlug {
		s := slug
		cr.ClassRoomSlug = &s
	} else {
		code := "SEC-" + strings.ToUpper(slugifySimple(baseName))
		cr.ClassRoomCode = &code
	}

	tx := g.DB.WithContext(ctx)
	if hasSlug {
		if er := tx.Clauses(clause.OnConflict{Columns: []clause.Column{
			{Name: "class_room_school_id"},
			{Name: "class_room_slug"},
		}, DoNothing: true}).Create(&cr).Error; er != nil {
			return nil, nil, er
		}
		if cr.ClassRoomID == uuid.Nil {
			if er := tx.
				Where("class_room_school_id = ? AND class_room_slug = ? AND class_room_deleted_at IS NULL", schoolID, slug).
				Take(&cr).Error; er != nil {
				return nil, nil, er
			}
		}
	} else {
		if er := tx.Create(&cr).Error; er != nil {
			return nil, nil, er
		}
	}

	id := cr.ClassRoomID
	return &id, nil, nil
}

/* =========================
   Helpers (schema detection)
========================= */

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

/* =========================
   Helpers (waktu & rule)
========================= */

func parseTODString(s string) (time.Time, error) {
	if t, err := time.Parse("15:04:05", s); err == nil {
		return time.Date(2000, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
	}
	if t, err := time.Parse("15:04", s); err == nil {
		return time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC), nil
	}
	return time.Time{}, fmt.Errorf("invalid time-of-day format: %q", s)
}

func startOfDayInLoc(t time.Time, loc *time.Location) time.Time {
	tt := t.In(loc)
	return time.Date(tt.Year(), tt.Month(), tt.Day(), 0, 0, 0, 0, loc)
}

func combineLocalDateAndTOD(dLocal, tod time.Time, loc *time.Location) time.Time {
	return time.Date(dLocal.Year(), dLocal.Month(), dLocal.Day(), tod.Hour(), tod.Minute(), tod.Second(), 0, loc)
}

func toUTC(t time.Time) time.Time { return t.In(time.UTC) }

func isoWeekday(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

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

func weeksBetween(base, target time.Time) int {
	ad := startOfDayInLoc(base, base.Location())
	bd := startOfDayInLoc(target, target.Location())
	if bd.Before(ad) {
		return -int(ad.Sub(bd).Hours() / 24 / 7)
	}
	return int(bd.Sub(ad).Hours() / 24 / 7)
}

func dateMatchesRuleRow(dLocal, baseStartLocal time.Time, r ruleRow) bool {
	if isoWeekday(dLocal) != r.DayOfWeek {
		return false
	}
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
	if r.LastWeekOfMonth && !isLastWeekOfMonth(dLocal) {
		return false
	}
	return true
}
