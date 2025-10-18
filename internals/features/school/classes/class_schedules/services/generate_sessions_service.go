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

	roomModel "masjidku_backend/internals/features/school/academics/rooms/model"
	sessModel "masjidku_backend/internals/features/school/classes/class_attendance_sessions/model"
	schedModel "masjidku_backend/internals/features/school/classes/class_schedules/model"

	snapshotTeacher "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/snapshot"
	snapshotClassRoom "masjidku_backend/internals/features/school/academics/rooms/snapshot"
	snapshotCSST "masjidku_backend/internals/features/school/classes/class_section_subject_teachers/snapshot"
)

/* =========================
   Utils (string & slug)
========================= */

// tambahkan di file yang sama (di dekat helpers lain)
func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

func stringsTrimLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// --- util sederhana untuk slug ---
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
		// ignore other chars
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "room"
	}
	return out
}

/* =========================
   Generator + Options
========================= */

type Generator struct{ DB *gorm.DB }

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

/* =========================
   Row ringan (rules & section)
========================= */

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

type sectionLite struct {
	ID       uuid.UUID `gorm:"column:class_section_id"`
	MasjidID uuid.UUID `gorm:"column:class_section_masjid_id"`
	Name     *string   `gorm:"column:class_section_name"`
}

/* =========================
   Snapshot builders
========================= */

// Room snapshot
func (g *Generator) buildRoomSnapshotJSON(
	ctx context.Context,
	expectMasjidID uuid.UUID,
	roomID uuid.UUID,
) (datatypes.JSONMap, error) {
	tx := g.DB.WithContext(ctx)
	rs, err := snapshotClassRoom.ValidateAndSnapshotRoom(tx, expectMasjidID, roomID)
	if err != nil {
		return nil, err
	}
	j := snapshotClassRoom.ToJSON(rs)
	var m map[string]any
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return datatypes.JSONMap(m), nil
}

// CSST snapshot (return JSONMap + teacherID + name)
func (g *Generator) buildCSSTSnapshotJSON(
	ctx context.Context,
	expectMasjidID uuid.UUID,
	csstID uuid.UUID,
) (datatypes.JSONMap, *uuid.UUID, *string, error) {
	tx := g.DB.WithContext(ctx)
	cs, err := snapshotCSST.ValidateAndSnapshotCSST(tx, expectMasjidID, csstID)
	if err != nil {
		return nil, nil, nil, err
	}
	j := snapshotCSST.ToJSON(cs)
	var m map[string]any
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, nil, nil, err
	}
	return datatypes.JSONMap(m), cs.TeacherID, cs.Name, nil
}

// Teacher snapshot
func (g *Generator) buildTeacherSnapshotJSON(
	ctx context.Context,
	expectMasjidID uuid.UUID,
	teacherID uuid.UUID,
) (datatypes.JSONMap, error) {
	tx := g.DB.WithContext(ctx)
	ts, err := snapshotTeacher.ValidateAndSnapshotTeacher(tx, expectMasjidID, teacherID)
	if err != nil {
		return nil, err
	}
	j := snapshotTeacher.ToJSON(ts)
	var m map[string]any
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return datatypes.JSONMap(m), nil
}

/* =========================
   Public API
========================= */

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
		defCSSTSnap       datatypes.JSONMap
		defTeacherSnap    datatypes.JSONMap
		defRoomSnap       datatypes.JSONMap
		teacherIDFromCSST *uuid.UUID
		defCSSTName       *string
	)

	if opts.DefaultCSSTID != nil {
		if s, tid, name, er := g.buildCSSTSnapshotJSON(ctx, sch.ClassScheduleMasjidID, *opts.DefaultCSSTID); er == nil {
			defCSSTSnap = s
			teacherIDFromCSST = tid
			defCSSTName = name
		}
	}
	if opts.DefaultTeacherID != nil {
		if s, er := g.buildTeacherSnapshotJSON(ctx, sch.ClassScheduleMasjidID, *opts.DefaultTeacherID); er == nil {
			defTeacherSnap = s
		}
	} else if teacherIDFromCSST != nil {
		if s, er := g.buildTeacherSnapshotJSON(ctx, sch.ClassScheduleMasjidID, *teacherIDFromCSST); er == nil {
			defTeacherSnap = s
		}
	}
	if opts.DefaultRoomID != nil {
		if s, er := g.buildRoomSnapshotJSON(ctx, sch.ClassScheduleMasjidID, *opts.DefaultRoomID); er == nil {
			defRoomSnap = s
		}
	}

	// ---------- Cache untuk snapshot per entitas ----------
	csstSnapCache := map[uuid.UUID]datatypes.JSONMap{}
	csstTeacherIDCache := map[uuid.UUID]*uuid.UUID{}
	teacherSnapCache := map[uuid.UUID]datatypes.JSONMap{}
	roomSnapCache := map[uuid.UUID]datatypes.JSONMap{}

	// ---------- Cache resolusi room & nama CSST ----------
	roomIDByCSST := map[uuid.UUID]*uuid.UUID{}
	csstNameCache := map[uuid.UUID]*string{}
	meetingCountByCSST := map[uuid.UUID]int{}

	// ---------- 3) Expand occurrences ----------
	rows := make([]sessModel.ClassAttendanceSessionModel, 0, 1024)

	// Helper: pasang CSST/Teacher/Room + snapshot ke row
	attachSnapshots := func(row *sessModel.ClassAttendanceSessionModel, ruleCSST *uuid.UUID) {
		// --- CSST (per-rule > default) ---
		var effCSST *uuid.UUID
		var effCSSTSnap datatypes.JSONMap
		var effTeacherFromCSST *uuid.UUID

		if ruleCSST != nil {
			effCSST = ruleCSST
			if s, ok := csstSnapCache[*ruleCSST]; ok {
				effCSSTSnap = s
				effTeacherFromCSST = csstTeacherIDCache[*ruleCSST]
			} else if s, tid, name, er := g.buildCSSTSnapshotJSON(ctx, sch.ClassScheduleMasjidID, *ruleCSST); er == nil {
				csstSnapCache[*ruleCSST] = s
				csstTeacherIDCache[*ruleCSST] = tid
				csstNameCache[*ruleCSST] = name
				effCSSTSnap = s
				effTeacherFromCSST = tid
			}
		} else if opts.DefaultCSSTID != nil {
			effCSST = opts.DefaultCSSTID
			effCSSTSnap = defCSSTSnap
			effTeacherFromCSST = teacherIDFromCSST
		}
		if effCSST != nil {
			row.ClassAttendanceSessionCSSTID = effCSST
			if effCSSTSnap != nil {
				row.ClassAttendanceSessionCSSTSnapshot = effCSSTSnap
			}
		}

		// --- TEACHER ---
		var effTeacher *uuid.UUID
		var effTeacherSnap datatypes.JSONMap

		if opts.DefaultTeacherID != nil {
			effTeacher = opts.DefaultTeacherID
			effTeacherSnap = defTeacherSnap
		} else if effTeacherFromCSST != nil {
			effTeacher = effTeacherFromCSST
			if s, ok := teacherSnapCache[*effTeacherFromCSST]; ok {
				effTeacherSnap = s
			} else if s, er := g.buildTeacherSnapshotJSON(ctx, sch.ClassScheduleMasjidID, *effTeacherFromCSST); er == nil {
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

		// --- Title otomatis "<nama CSST> pertemuan ke-N" ---
		var baseName *string
		if effCSST != nil {
			if ruleCSST != nil {
				baseName = csstNameCache[*ruleCSST]
			} else {
				baseName = defCSSTName
			}
		}
		if baseName != nil && strings.TrimSpace(*baseName) != "" {
			key := *effCSST
			meetingCountByCSST[key] = meetingCountByCSST[key] + 1
			n := meetingCountByCSST[key]
			title := fmt.Sprintf("%s pertemuan ke-%d", strings.TrimSpace(*baseName), n)
			row.ClassAttendanceSessionTitle = &title
		}

		// --- ROOM: Default → CSST/Section (auto-provision jika perlu) ---
		if opts.DefaultRoomID != nil {
			row.ClassAttendanceSessionClassRoomID = opts.DefaultRoomID
			if defRoomSnap != nil {
				row.ClassAttendanceSessionRoomSnapshot = defRoomSnap
			} else {
				if s, ok := roomSnapCache[*opts.DefaultRoomID]; ok {
					row.ClassAttendanceSessionRoomSnapshot = s
				} else if s, er := g.buildRoomSnapshotJSON(ctx, sch.ClassScheduleMasjidID, *opts.DefaultRoomID); er == nil {
					roomSnapCache[*opts.DefaultRoomID] = s
					row.ClassAttendanceSessionRoomSnapshot = s
				}
			}
		} else {
			var resolvedRoomID *uuid.UUID
			var resolvedSnap datatypes.JSONMap
			var er error

			if ruleCSST != nil {
				if rid, ok := roomIDByCSST[*ruleCSST]; ok {
					resolvedRoomID = rid
					if rid != nil {
						if s, ok2 := roomSnapCache[*rid]; ok2 {
							resolvedSnap = s
						} else if s, er2 := g.buildRoomSnapshotJSON(ctx, sch.ClassScheduleMasjidID, *rid); er2 == nil {
							roomSnapCache[*rid] = s
							resolvedSnap = s
						}
					}
				} else {
					resolvedRoomID, resolvedSnap, er = g.ResolveRoomFromCSSTOrSection(ctx, sch.ClassScheduleMasjidID, ruleCSST)
					roomIDByCSST[*ruleCSST] = resolvedRoomID
					if resolvedRoomID != nil && resolvedSnap != nil {
						roomSnapCache[*resolvedRoomID] = resolvedSnap
					}
				}
			}

			if er == nil && resolvedRoomID != nil {
				row.ClassAttendanceSessionClassRoomID = resolvedRoomID
			}
			if resolvedSnap != nil {
				row.ClassAttendanceSessionRoomSnapshot = resolvedSnap
			}
		}
	}

	// Tanpa rule → satu sesi di start date
	if len(rr) == 0 {
		dateUTC := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, time.UTC)
		row := sessModel.ClassAttendanceSessionModel{
			ClassAttendanceSessionMasjidID:         sch.ClassScheduleMasjidID,
			ClassAttendanceSessionScheduleID:       ptrUUID(sch.ClassScheduleID), // <— was: sch.ClassScheduleID
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
				// Parse start/end TOD
				stTOD, err1 := parseTODString(r.StartStr)
				etTOD, err2 := parseTODString(r.EndStr)
				if err1 != nil || err2 != nil {
					continue
				}

				startAtLocal := combineLocalDateAndTOD(d, stTOD, loc)
				endAtLocal := combineLocalDateAndTOD(d, etTOD, loc)

				// Overnight guard
				if endAtLocal.Before(startAtLocal) {
					endAtLocal = endAtLocal.Add(24 * time.Hour)
				}

				startAtUTC := toUTC(startAtLocal)
				endAtUTC := toUTC(endAtLocal)
				dateUTC := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
				rid := r.ID
				row := sessModel.ClassAttendanceSessionModel{
					ClassAttendanceSessionMasjidID:         sch.ClassScheduleMasjidID,
					ClassAttendanceSessionScheduleID:       ptrUUID(sch.ClassScheduleID), // <— was: sch.ClassScheduleID
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

	// ---------- 4) Idempotent insert (batch) ----------
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

// coalesceMasjid mengembalikan a jika tidak Nil, selain itu b.
func coalesceMasjid(a, b uuid.UUID) uuid.UUID {
	if a != uuid.Nil {
		return a
	}
	return b
}

// ResolveRoomFromCSSTOrSection mengembalikan (roomID, roomSnapshot) berdasar:
// 1) Room langsung pada CSST
// 2) Room pada Section dari CSST
// 3) Auto-provision room untuk Section (jika belum ada)
// NOTE: expectMasjidID dipakai untuk validasi tenant saat membangun snapshot room.
func (g *Generator) ResolveRoomFromCSSTOrSection(
	ctx context.Context,
	expectMasjidID uuid.UUID,
	csstID *uuid.UUID,
) (*uuid.UUID, datatypes.JSONMap, error) {
	// Guard input
	if csstID == nil || *csstID == uuid.Nil {
		return nil, nil, nil
	}

	// Ambil info dasar dari CSST
	roomID, sectionID, csstMasjidID, sectionName, err := g.getRoomOrSectionFromCSST(ctx, *csstID)
	if err != nil {
		return nil, nil, err
	}

	// Validasi tenant (opsional)
	if expectMasjidID != uuid.Nil && csstMasjidID != uuid.Nil && expectMasjidID != csstMasjidID {
		return nil, nil, fmt.Errorf("tenant mismatch: csst.masjid=%s != expect=%s", csstMasjidID, expectMasjidID)
	}

	// 1) Room pada CSST → snapshot & return
	if roomID != nil && *roomID != uuid.Nil {
		snap, er := g.buildRoomSnapshotJSON(ctx, coalesceMasjid(expectMasjidID, csstMasjidID), *roomID)
		return roomID, snap, er
	}

	// 2) Fallback: Room pada Section
	if sectionID != nil && *sectionID != uuid.Nil {
		rid, secMasjidID, secName, er := g.getRoomFromSection(ctx, *sectionID)
		if er == nil {
			if rid != nil && *rid != uuid.Nil {
				snap, er2 := g.buildRoomSnapshotJSON(ctx, coalesceMasjid(expectMasjidID, secMasjidID), *rid)
				return rid, snap, er2
			}
			// 3) Auto-provision jika belum ada room
			useMasjid := coalesceMasjid(secMasjidID, csstMasjidID)
			return g.ensureSectionRoom(ctx, useMasjid, *sectionID, secName)
		}

		// Gagal baca section, masih bisa autoprov dengan masjid dari CSST
		if csstMasjidID != uuid.Nil {
			return g.ensureSectionRoom(ctx, csstMasjidID, *sectionID, sectionName)
		}
		return nil, nil, err
	}

	// 3) CSST tidak punya Section → tidak bisa resolve
	return nil, nil, nil
}

// Ambil (roomID, sectionID, masjidID, sectionName) dari CSST dengan deteksi kolom fleksibel
func (g *Generator) getRoomOrSectionFromCSST(ctx context.Context, csst uuid.UUID) (roomID *uuid.UUID, sectionID *uuid.UUID, masjidID uuid.UUID, sectionName *string, err error) {
	cols, err := g.tableColumns(ctx, "class_section_subject_teachers")
	if err != nil {
		return nil, nil, uuid.Nil, nil, err
	}

	idCol := firstExisting(cols, "class_section_subject_teacher_id", "id")
	masjidCol := firstExisting(cols, "class_section_subject_teacher_masjid_id", "masjid_id")

	roomCol := firstExisting(cols,
		"class_section_subject_teacher_room_id",
		"class_room_id",
		"room_id",
	)
	sectionCol := firstExisting(cols,
		"class_section_subject_teacher_section_id",
		"class_section_id",
		"section_id",
	)
	deletedCol := firstExisting(cols, "class_section_subject_teacher_deleted_at", "deleted_at")

	if idCol == "" || masjidCol == "" {
		return nil, nil, uuid.Nil, nil, fmt.Errorf("getRoomOrSectionFromCSST: kolom minimal (id/masjid_id) tidak ditemukan di CSST")
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
  csst.%s AS masjid_id,
  %s      AS room_id,
  %s      AS section_id
FROM class_section_subject_teachers csst
WHERE csst.%s = ?
%s
LIMIT 1`, idCol, masjidCol, roomExpr, secExpr, idCol, whereDeleted)

	var row struct {
		CSST     uuid.UUID  `gorm:"column:csst_id"`
		MasjidID uuid.UUID  `gorm:"column:masjid_id"`
		RoomID   *uuid.UUID `gorm:"column:room_id"`
		Section  *uuid.UUID `gorm:"column:section_id"`
	}
	if er := g.DB.WithContext(ctx).Raw(q, csst).Scan(&row).Error; er != nil {
		return nil, nil, uuid.Nil, nil, er
	}

	// optionally tarik nama section (untuk auto-provision)
	var secName *string
	if row.Section != nil && *row.Section != uuid.Nil {
		if s, er := g.getSectionNameAndMasjid(ctx, *row.Section); er == nil {
			secName = s.Name
			// kalau masjidID kosong dari CSST, isi dari section
			if row.MasjidID == uuid.Nil {
				row.MasjidID = s.MasjidID
			}
		}
	}

	return row.RoomID, row.Section, row.MasjidID, secName, nil
}

// Kembalikan (roomID, sectionMasjidID, sectionName) dari Section
func (g *Generator) getRoomFromSection(ctx context.Context, sectionID uuid.UUID) (roomID *uuid.UUID, masjidID uuid.UUID, name *string, err error) {
	cols, err := g.tableColumns(ctx, "class_sections")
	if err != nil {
		return nil, uuid.Nil, nil, err
	}
	idCol := firstExisting(cols, "class_section_id", "id")
	masjidCol := firstExisting(cols, "class_section_masjid_id", "masjid_id")
	nameCol := firstExisting(cols, "class_section_name", "name")
	roomCol := firstExisting(cols,
		"class_section_room_id",
		"class_room_id",
		"room_id",
	)
	deletedCol := firstExisting(cols, "class_section_deleted_at", "deleted_at")

	if idCol == "" || masjidCol == "" {
		return nil, uuid.Nil, nil, fmt.Errorf("getRoomFromSection: kolom minimal (id/masjid_id) tidak ditemukan")
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
  s.%s AS class_section_masjid_id,
  %s   AS class_section_name,
  %s   AS class_room_id
FROM class_sections s
WHERE s.%s = ?
%s
LIMIT 1`, idCol, masjidCol, nameExpr, roomExpr, idCol, whereDeleted)

	var row struct {
		ID       uuid.UUID  `gorm:"column:class_section_id"`
		MasjidID uuid.UUID  `gorm:"column:class_section_masjid_id"`
		Name     *string    `gorm:"column:class_section_name"`
		RoomID   *uuid.UUID `gorm:"column:class_room_id"`
	}
	if er := g.DB.WithContext(ctx).Raw(q, sectionID).Scan(&row).Error; er != nil {
		return nil, uuid.Nil, nil, er
	}
	return row.RoomID, row.MasjidID, row.Name, nil
}

func (g *Generator) getSectionNameAndMasjid(ctx context.Context, sectionID uuid.UUID) (*sectionLite, error) {
	cols, err := g.tableColumns(ctx, "class_sections")
	if err != nil {
		return nil, err
	}
	idCol := firstExisting(cols, "class_section_id", "id")
	masjidCol := firstExisting(cols, "class_section_masjid_id", "masjid_id")
	nameCol := firstExisting(cols, "class_section_name", "name")
	if idCol == "" || masjidCol == "" {
		return nil, fmt.Errorf("getSectionNameAndMasjid: kolom minimal tidak ditemukan")
	}
	nameExpr := "NULL::text"
	if nameCol != "" {
		nameExpr = fmt.Sprintf("s.%s", nameCol)
	}
	q := fmt.Sprintf(`
SELECT
  s.%s AS class_section_id,
  s.%s AS class_section_masjid_id,
  %s   AS class_section_name
FROM class_sections s
WHERE s.%s = ?
LIMIT 1`, idCol, masjidCol, nameExpr, idCol)

	var row sectionLite
	if er := g.DB.WithContext(ctx).Raw(q, sectionID).Scan(&row).Error; er != nil {
		return nil, er
	}
	return &row, nil
}

// Buat Room default untuk Section (bila belum ada sama sekali). Return (roomID, snapshot RoomSnapshot)
func (g *Generator) ensureSectionRoom(
	ctx context.Context,
	masjidID uuid.UUID,
	sectionID uuid.UUID,
	sectionName *string,
) (*uuid.UUID, datatypes.JSONMap, error) {
	if masjidID == uuid.Nil || sectionID == uuid.Nil {
		return nil, nil, fmt.Errorf("ensureSectionRoom: masjidID/sectionID kosong")
	}

	// Nama & slug default
	baseName := "Ruang"
	if sectionName != nil && strings.TrimSpace(*sectionName) != "" {
		baseName = fmt.Sprintf("Ruang %s", strings.TrimSpace(*sectionName))
	}
	slug := fmt.Sprintf("section-%s", strings.ReplaceAll(sectionID.String(), "-", "")) // unik per section

	// Cek apakah class_rooms punya kolom slug
	cols, err := g.tableColumns(ctx, "class_rooms")
	if err != nil {
		return nil, nil, err
	}
	hasSlug := firstExisting(cols, "class_room_slug", "slug") != ""

	// Cari Room dengan slug tersebut (jika slug tersedia)
	var existing roomModel.ClassRoomModel
	if hasSlug {
		if er := g.DB.WithContext(ctx).
			Where("class_room_masjid_id = ? AND class_room_slug = ? AND class_room_deleted_at IS NULL", masjidID, slug).
			Limit(1).
			Take(&existing).Error; er == nil && existing.ClassRoomID != uuid.Nil {
			id := existing.ClassRoomID
			snapJSON, er2 := g.buildRoomSnapshotJSON(ctx, masjidID, id)
			return &id, snapJSON, er2
		}
	}

	// Buat room baru
	cr := roomModel.ClassRoomModel{
		ClassRoomMasjidID:  masjidID,
		ClassRoomName:      baseName,
		ClassRoomIsVirtual: false,
		ClassRoomIsActive:  true,
	}
	if hasSlug {
		s := slug // stabil & deterministic
		cr.ClassRoomSlug = &s
	} else {
		code := "SEC-" + strings.ToUpper(slugifySimple(baseName))
		cr.ClassRoomCode = &code
	}

	// Idempotent: bila (masjid_id, slug) unik → DoNothing lalu fetch
	tx := g.DB.WithContext(ctx)
	if hasSlug {
		if er := tx.Clauses(clause.OnConflict{Columns: []clause.Column{
			{Name: "class_room_masjid_id"},
			{Name: "class_room_slug"},
		}, DoNothing: true}).Create(&cr).Error; er != nil {
			return nil, nil, er
		}
		// jika tidak inserted (sudah ada), ambil kembali
		if cr.ClassRoomID == uuid.Nil {
			if er := tx.
				Where("class_room_masjid_id = ? AND class_room_slug = ? AND class_room_deleted_at IS NULL", masjidID, slug).
				Take(&cr).Error; er != nil {
				return nil, nil, er
			}
		}
	} else {
		// tanpa slug → create biasa
		if er := tx.Create(&cr).Error; er != nil {
			return nil, nil, er
		}
	}

	id := cr.ClassRoomID
	snapJSON, er := g.buildRoomSnapshotJSON(ctx, masjidID, id)
	return &id, snapJSON, er
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
