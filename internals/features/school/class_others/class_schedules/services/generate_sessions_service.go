// file: internals/features/school/class_others/class_schedules/services/generate_sessions.go
package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	roomModel "madinahsalam_backend/internals/features/school/academics/rooms/model"
	sessModel "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
	schedModel "madinahsalam_backend/internals/features/school/class_others/class_schedules/model"
)

/* =========================
   Utils (string & slug)
========================= */

func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

func stringsTrimLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

const maxScheduleDays = 180 * 2 // misal maksimal 2 tahun

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
	DefaultSessionTypeID    *uuid.UUID // type default utk semua sesi generate
	DefaultAttendanceStatus string
	BatchSize               int
}

/*
	=========================
	  CSST loader (SLIM)
	  - hanya untuk dapat TeacherID + baseName + slug
	  - tanpa JSON snapshot
	=========================
*/

type csstRich struct {
	ID          uuid.UUID  `gorm:"column:id"`
	SchoolID    uuid.UUID  `gorm:"column:school_id"`
	Slug        string     `gorm:"column:slug"`
	Name        *string    `gorm:"column:name"`
	SubjectID   *uuid.UUID `gorm:"column:subject_id"`
	SubjectName *string    `gorm:"column:subject_name"`
	TeacherID   *uuid.UUID `gorm:"column:teacher_id"`
	RoomID      *uuid.UUID `gorm:"column:room_id"`
}

func (g *Generator) getCSSTRich(
	ctx context.Context,
	expectSchool uuid.UUID,
	csstID uuid.UUID,
) (*csstRich, error) {
	// SQL statis berdasarkan schema fix; tanpa JSON cache
	q := `
SELECT
  csst.class_section_subject_teacher_id                    AS id,
  csst.class_section_subject_teacher_school_id             AS school_id,
  COALESCE(
    csst.class_section_subject_teacher_slug,
    csst.class_section_subject_teacher_id::text
  )                                                        AS slug,
  COALESCE(
    csst.class_section_subject_teacher_class_section_name_cache,
    csst.class_section_subject_teacher_subject_name_cache,
    sec.class_section_name
  )                                                        AS name,

  -- Subject
  csst.class_section_subject_teacher_subject_id   AS subject_id,
  COALESCE(subj.subject_name,
           csst.class_section_subject_teacher_subject_name_cache)
    AS subject_name,

  -- Teacher
  csst.class_section_subject_teacher_school_teacher_id     AS teacher_id,

  -- Room (kalau ada di CSST)
  csst.class_section_subject_teacher_class_room_id         AS room_id

FROM class_section_subject_teachers csst
LEFT JOIN subjects subj
  ON subj.subject_id = csst.class_section_subject_teacher_subject_id
LEFT JOIN class_sections sec
  ON sec.class_section_id = csst.class_section_subject_teacher_class_section_id
WHERE csst.class_section_subject_teacher_id = ?
  AND csst.class_section_subject_teacher_deleted_at IS NULL
`
	args := []any{csstID}
	if expectSchool != uuid.Nil {
		q += "  AND csst.class_section_subject_teacher_school_id = ?\n"
		args = append(args, expectSchool)
	}
	q += "LIMIT 1"

	var row csstRich
	if err := g.DB.WithContext(ctx).Raw(q, args...).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	if strings.TrimSpace(row.Slug) == "" {
		row.Slug = row.ID.String()
	}

	return &row, nil
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

/* =========================
   Rule row (tanpa JSON)
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
	WeekParity      *string       `gorm:"column:class_schedule_rule_week_parity"`
	WeeksOfMonth    pq.Int64Array `gorm:"column:class_schedule_rule_weeks_of_month"`
	LastWeekOfMonth bool          `gorm:"column:class_schedule_rule_last_week_of_month"`
	CSSTID          *uuid.UUID    `gorm:"column:class_schedule_rule_csst_id"`
}

/*
	=========================
	  Session Type helper
	  (auto-generate default type per tenant)
	=========================
*/

type sessionTypeRow struct {
	ID       uuid.UUID `gorm:"column:class_attendance_session_type_id"`
	SchoolID uuid.UUID `gorm:"column:class_attendance_session_type_school_id"`
	Slug     string    `gorm:"column:class_attendance_session_type_slug"`
	Name     string    `gorm:"column:class_attendance_session_type_name"`
	// meta visual
	Description *string `gorm:"column:class_attendance_session_type_description"`
	Color       *string `gorm:"column:class_attendance_session_type_color"`
	Icon        *string `gorm:"column:class_attendance_session_type_icon"`

	// konfigurasi attendance (sesuai ALTER TABLE terbaru)
	AllowStudentSelfAttendance bool           `gorm:"column:class_attendance_session_type_allow_student_self_attendance"`
	AllowTeacherMarkAttendance bool           `gorm:"column:class_attendance_session_type_allow_teacher_mark_attendance"`
	RequireTeacherAttendance   bool           `gorm:"column:class_attendance_session_type_require_teacher_attendance"`
	RequireAttendanceReason    pq.StringArray `gorm:"column:class_attendance_session_type_require_attendance_reason"`

	// ✅ kolom baru window mode + offset
	AttendanceWindowMode         string `gorm:"column:class_attendance_session_type_attendance_window_mode"`
	AttendanceOpenOffsetMinutes  *int   `gorm:"column:class_attendance_session_type_attendance_open_offset_minutes"`
	AttendanceCloseOffsetMinutes *int   `gorm:"column:class_attendance_session_type_attendance_close_offset_minutes"`
}

// ensureDefaultSessionType:
// - cari type default (slug fix) per tenant
// - kalau belum ada, buat
// - return row utk dipakai di semua sesi hasil generate
func (g *Generator) ensureDefaultSessionType(
	ctx context.Context,
	schoolID uuid.UUID,
) (*sessionTypeRow, error) {
	if schoolID == uuid.Nil {
		return nil, fmt.Errorf("ensureDefaultSessionType: schoolID kosong")
	}

	const slug = "kbm-regular"
	const name = "Pertemuan KBM"

	var row sessionTypeRow

	// 1) Cek existing (alive)
	if err := g.DB.WithContext(ctx).
		Table("class_attendance_session_types").
		Where(`
			class_attendance_session_type_school_id = ?
			AND lower(class_attendance_session_type_slug) = lower(?)
			AND class_attendance_session_type_deleted_at IS NULL
		`, schoolID, slug).
		Limit(1).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID != uuid.Nil {
		return &row, nil
	}

	// 2) Belum ada → buat baru
	now := time.Now()

	desc := "Sesi kehadiran hasil generate otomatis dari jadwal"
	color := "#2563eb"       // biru (opsional)
	icon := "CalendarCheck2" // nama icon di FE (opsional)
	isActive := true
	sortOrder := 10

	insert := map[string]any{
		"class_attendance_session_type_school_id":                     schoolID,
		"class_attendance_session_type_slug":                          slug,
		"class_attendance_session_type_name":                          name,
		"class_attendance_session_type_description":                   desc,
		"class_attendance_session_type_color":                         color,
		"class_attendance_session_type_icon":                          icon,
		"class_attendance_session_type_is_active":                     isActive,
		"class_attendance_session_type_sort_order":                    sortOrder,
		"class_attendance_session_type_created_at":                    now,
		"class_attendance_session_type_updated_at":                    now,
		"class_attendance_session_type_allow_student_self_attendance": true,
		"class_attendance_session_type_allow_teacher_mark_attendance": true,
		"class_attendance_session_type_require_teacher_attendance":    true,
		"class_attendance_session_type_require_attendance_reason":     pq.StringArray{"unmarked"},

		// ✅ default attendance window
		"class_attendance_session_type_attendance_window_mode": "same_day",
	}

	if err := g.DB.WithContext(ctx).
		Table("class_attendance_session_types").
		Create(&insert).Error; err != nil {
		// Kemungkinan race condition unique index → coba select ulang
		if !strings.Contains(strings.ToLower(err.Error()), "uq_castype_school_slug_alive") {
			return nil, err
		}
	}

	// 3) Ambil lagi setelah insert
	row = sessionTypeRow{}
	if err := g.DB.WithContext(ctx).
		Table("class_attendance_session_types").
		Where(`
			class_attendance_session_type_school_id = ?
			AND lower(class_attendance_session_type_slug) = lower(?)
			AND class_attendance_session_type_deleted_at IS NULL
		`, schoolID, slug).
		Limit(1).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, fmt.Errorf("ensureDefaultSessionType: gagal mengambil row setelah insert")
	}

	return &row, nil
}

// Ambil type by ID, dengan guard tenant
func (g *Generator) getSessionTypeByID(
	ctx context.Context,
	schoolID uuid.UUID,
	typeID uuid.UUID,
) (*sessionTypeRow, error) {
	if schoolID == uuid.Nil || typeID == uuid.Nil {
		return nil, fmt.Errorf("getSessionTypeByID: schoolID/typeID kosong")
	}

	var row sessionTypeRow
	if err := g.DB.WithContext(ctx).
		Table("class_attendance_session_types").
		Where(`
			class_attendance_session_type_school_id = ?
			AND class_attendance_session_type_id = ?
			AND class_attendance_session_type_deleted_at IS NULL
		`, schoolID, typeID).
		Limit(1).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

/* =========================
   Public API
========================= */

func (g *Generator) GenerateSessionsForSchedule(ctx context.Context, scheduleID string) (int, error) {
	return g.GenerateSessionsForScheduleWithOpts(ctx, scheduleID, nil)
}

func (g *Generator) GenerateSessionsForScheduleWithOpts(
	ctx context.Context,
	scheduleID string,
	opts *GenerateOptions,
) (created int, err error) {
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

	// Guard: end < start → tidak generate
	if endLocal.Before(startLocal) {
		return 0, fmt.Errorf("invalid date range: start_date (%s) after end_date (%s)",
			sch.ClassScheduleStartDate.Format("2006-01-02"),
			sch.ClassScheduleEndDate.Format("2006-01-02"),
		)
	}

	// Guard: batasi maksimum range hari
	daysSpan := int(endLocal.Sub(startLocal).Hours()/24) + 1
	if daysSpan <= 0 {
		return 0, nil
	}
	if daysSpan > maxScheduleDays {
		return 0, fmt.Errorf(
			"date range too long for schedule %s: %d days (max %d)",
			sch.ClassScheduleID, daysSpan, maxScheduleDays,
		)
	}

	// 1.5) Siapkan default Session Type untuk semua sesi hasil generate
	var defTypeID *uuid.UUID

	var defTypeRow *sessionTypeRow

	// Prioritas: DefaultSessionTypeID dari caller (schedule payload)
	if opts.DefaultSessionTypeID != nil && *opts.DefaultSessionTypeID != uuid.Nil {
		st, er := g.getSessionTypeByID(ctx, sch.ClassScheduleSchoolID, *opts.DefaultSessionTypeID)
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				log.Printf(
					"[Generator.GenerateSessionsForSchedule] invalid session_type_id=%s for schedule=%s school=%s (not found / different tenant)",
					opts.DefaultSessionTypeID.String(),
					scheduleID,
					sch.ClassScheduleSchoolID,
				)
				return 0, fmt.Errorf("session_type_id tidak ditemukan atau bukan milik sekolah ini")
			}

			log.Printf(
				"[Generator.GenerateSessionsForSchedule] error loading session_type_id=%s for schedule=%s school=%s: %v",
				opts.DefaultSessionTypeID.String(),
				scheduleID,
				sch.ClassScheduleSchoolID,
				er,
			)
			return 0, fmt.Errorf("gagal mengambil session_type_id: %w", er)
		}

		if st != nil {
			defTypeID = &st.ID
			defTypeRow = st
		}
	}

	// Kalau belum ada dari payload → fallback ke default auto ("kbm-regular")
	if defTypeID == nil {
		if st, er := g.ensureDefaultSessionType(ctx, sch.ClassScheduleSchoolID); er == nil && st != nil {
			defTypeID = &st.ID
			defTypeRow = st
		} else if er != nil {
			log.Printf(
				"[Generator.GenerateSessionsForSchedule] ensureDefaultSessionType failed for schedule=%s school=%s: %v",
				scheduleID,
				sch.ClassScheduleSchoolID,
				er,
			)
		}
	}

	// Kalau belum ada dari payload → fallback ke default auto ("kbm-regular")
	if defTypeID == nil {
		if st, er := g.ensureDefaultSessionType(ctx, sch.ClassScheduleSchoolID); er == nil && st != nil {
			defTypeID = &st.ID
		} else if er != nil {
			log.Printf(
				"[Generator.GenerateSessionsForSchedule] ensureDefaultSessionType failed for schedule=%s school=%s: %v",
				scheduleID,
				sch.ClassScheduleSchoolID,
				er,
			)
		}
	}

	if defTypeID == nil {
		log.Printf(
			"[Generator.GenerateSessionsForSchedule] no session type resolved for schedule=%s school=%s; sessions will have nil type",
			scheduleID,
			sch.ClassScheduleSchoolID,
		)
	}

	// 2) Ambil rules (tanpa JSON) — TANPA information_schema
	var rr []ruleRow
	qRules := `
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
  class_schedule_rule_csst_id           AS class_schedule_rule_csst_id
FROM class_schedule_rules
WHERE class_schedule_rule_schedule_id = ?
  AND class_schedule_rule_deleted_at IS NULL
ORDER BY class_schedule_rule_day_of_week, class_schedule_rule_start_time`
	if err = g.DB.WithContext(ctx).Raw(qRules, sch.ClassScheduleID).Scan(&rr).Error; err != nil {
		return 0, err
	}

	// 2.5) Preload default CSST basic info (opsional)
	var (
		teacherIDFromCSST *uuid.UUID
		defCSSTName       *string
	)
	if opts.DefaultCSSTID != nil && *opts.DefaultCSSTID != uuid.Nil {
		if rich, er := g.getCSSTRich(ctx, sch.ClassScheduleSchoolID, *opts.DefaultCSSTID); er == nil && rich != nil {
			teacherIDFromCSST = rich.TeacherID
			baseName := rich.SubjectName
			if baseName == nil || strings.TrimSpace(*baseName) == "" {
				baseName = rich.Name
			}
			defCSSTName = baseName
		}
	}

	// Caches (per scheduler run)
	csstTeacherIDCache := map[uuid.UUID]*uuid.UUID{}
	csstNameCache := map[uuid.UUID]*string{}
	csstSlugCache := map[uuid.UUID]string{}
	meetingCountByCSST := map[uuid.UUID]int{}
	roomFromCSSTCache := map[uuid.UUID]*uuid.UUID{}

	// ===============================
	// Preload offset meeting number
	// ===============================
	existingMeetingOffset := map[uuid.UUID]int{}

	// Kumpulkan semua CSST yang akan dipakai (dari rules + default)
	csstIDsSet := map[uuid.UUID]struct{}{}
	if opts.DefaultCSSTID != nil && *opts.DefaultCSSTID != uuid.Nil {
		csstIDsSet[*opts.DefaultCSSTID] = struct{}{}
	}
	for _, r := range rr {
		if r.CSSTID != nil && *r.CSSTID != uuid.Nil {
			csstIDsSet[*r.CSSTID] = struct{}{}
		}
	}

	if len(csstIDsSet) > 0 {
		ids := make([]uuid.UUID, 0, len(csstIDsSet))
		for id := range csstIDsSet {
			ids = append(ids, id)
		}

		type meetingRow struct {
			CSSTID uuid.UUID `gorm:"column:csst_id"`
			MaxNo  *int      `gorm:"column:max_no"`
		}
		var mm []meetingRow

		// Ambil max meeting_number per CSST yang sudah pernah ada
		if err := g.DB.WithContext(ctx).
			Raw(`
				SELECT
				  class_attendance_session_csst_id AS csst_id,
				  MAX(class_attendance_session_meeting_number) AS max_no
				FROM class_attendance_sessions
				WHERE class_attendance_session_school_id = ?
				  AND class_attendance_session_csst_id = ANY(?)
				  AND class_attendance_session_deleted_at IS NULL
				GROUP BY class_attendance_session_csst_id
			`, sch.ClassScheduleSchoolID, pq.Array(ids)).
			Scan(&mm).Error; err != nil {
			// Kalau gagal, jangan matiin generator, cukup log aja
			log.Printf("[Generator] failed preload meeting offsets: %v", err)
		} else {
			for _, m := range mm {
				if m.MaxNo != nil {
					existingMeetingOffset[m.CSSTID] = *m.MaxNo
				}
			}
		}
	}

	// 3) Expand occurrences
	rows := make([]sessModel.ClassAttendanceSessionModel, 0, 1024)

	attachCaches := func(row *sessModel.ClassAttendanceSessionModel, ruleCSST *uuid.UUID) {
		// --- CSST (per-rule > default) ---
		var effCSST *uuid.UUID
		var effTeacherFromCSST *uuid.UUID
		var baseName *string

		if ruleCSST != nil && *ruleCSST != uuid.Nil {
			effCSST = ruleCSST

			if t, ok := csstTeacherIDCache[*ruleCSST]; ok {
				effTeacherFromCSST = t
			}
			if n, ok := csstNameCache[*ruleCSST]; ok {
				baseName = n
			}

			// kalau belum di-cache, load sekali dari DB
			if effTeacherFromCSST == nil || baseName == nil {
				if rich, er := g.getCSSTRich(ctx, sch.ClassScheduleSchoolID, *ruleCSST); er == nil && rich != nil {
					csstTeacherIDCache[*ruleCSST] = rich.TeacherID

					bn := rich.SubjectName
					if bn == nil || strings.TrimSpace(*bn) == "" {
						bn = rich.Name
					}
					csstNameCache[*ruleCSST] = bn
					if strings.TrimSpace(rich.Slug) != "" {
						csstSlugCache[*ruleCSST] = strings.TrimSpace(rich.Slug)
					}

					effTeacherFromCSST = rich.TeacherID
					baseName = bn
				}
			}
		} else if opts.DefaultCSSTID != nil && *opts.DefaultCSSTID != uuid.Nil {
			effCSST = opts.DefaultCSSTID
			effTeacherFromCSST = teacherIDFromCSST
			baseName = defCSSTName
		}

		// Simpan CSST ID (tanpa snapshot JSON)
		if effCSST != nil {
			row.ClassAttendanceSessionCSSTID = effCSST
		}

		// TeacherID
		if opts.DefaultTeacherID != nil {
			row.ClassAttendanceSessionTeacherID = opts.DefaultTeacherID
		} else if effTeacherFromCSST != nil {
			row.ClassAttendanceSessionTeacherID = effTeacherFromCSST
		}

		// RoomID — resolve dari CSST / Section bila DefaultRoom kosong
		if opts.DefaultRoomID != nil {
			row.ClassAttendanceSessionClassRoomID = opts.DefaultRoomID
		} else if ruleCSST != nil && *ruleCSST != uuid.Nil {
			// Cache hasil resolve room per CSST
			if cached, ok := roomFromCSSTCache[*ruleCSST]; ok {
				if cached != nil && *cached != uuid.Nil {
					row.ClassAttendanceSessionClassRoomID = cached
				}
			} else {
				if rid, er := g.ResolveRoomFromCSSTOrSection(ctx, sch.ClassScheduleSchoolID, ruleCSST); er == nil && rid != nil {
					roomFromCSSTCache[*ruleCSST] = rid
					row.ClassAttendanceSessionClassRoomID = rid
				} else {
					// tandai sudah pernah dicoba tapi tidak ada
					var zero uuid.UUID
					roomFromCSSTCache[*ruleCSST] = &zero
				}
			}
		}

		// Title otomatis + meeting number + slug per pertemuan
		if baseName != nil && strings.TrimSpace(*baseName) != "" && effCSST != nil {
			key := *effCSST

			// offset existing (max meeting number sebelumnya)
			offset := existingMeetingOffset[key] // default 0 kalau nggak ada

			// counter run ini
			meetingCountByCSST[key] = meetingCountByCSST[key] + 1

			// nomor final = existing max + urutan baru
			n := offset + meetingCountByCSST[key]

			// 🔢 simpan nomor pertemuan ke kolom khusus
			row.ClassAttendanceSessionMeetingNumber = ptr(n)

			// 🏷️ title: hanya nama CSST (tanpa "pertemuan ke-N")
			title := strings.TrimSpace(*baseName)
			row.ClassAttendanceSessionTitle = &title

			// 🔗 slug: "<slug_csst>-pertemuan-N" (kalau slug session masih kosong)
			if row.ClassAttendanceSessionSlug == nil || strings.TrimSpace(ptrStr(row.ClassAttendanceSessionSlug)) == "" {
				var baseSlug string
				if s, ok := csstSlugCache[key]; ok {
					baseSlug = strings.TrimSpace(s)
				}
				if baseSlug == "" {
					// fallback: pakai ID csst
					baseSlug = key.String()
				}
				slug := fmt.Sprintf("%s-pertemuan-%d", baseSlug, n)
				row.ClassAttendanceSessionSlug = &slug
			}
		}

		// TYPE default + SNAPSHOT
		if defTypeID != nil {
			row.ClassAttendanceSessionTypeID = defTypeID
		}

		// ✅ SNAPSHOT TYPE (JSONB) — kalau defTypeRow tersedia
		if defTypeRow != nil {
			// convert pq.StringArray -> []string biar bersih
			requireStates := []string{}
			if len(defTypeRow.RequireAttendanceReason) > 0 {
				requireStates = append(requireStates, defTypeRow.RequireAttendanceReason...)
			}

			snap := datatypes.JSONMap{
				"id":          defTypeRow.ID,
				"slug":        defTypeRow.Slug,
				"name":        defTypeRow.Name,
				"description": defTypeRow.Description,
				"color":       defTypeRow.Color,
				"icon":        defTypeRow.Icon,

				"allow_student_self_attendance": defTypeRow.AllowStudentSelfAttendance,
				"allow_teacher_mark_attendance": defTypeRow.AllowTeacherMarkAttendance,
				"require_teacher_attendance":    defTypeRow.RequireTeacherAttendance,
				"require_attendance_reason":     requireStates,

				"attendance_window_mode":          defTypeRow.AttendanceWindowMode,
				"attendance_open_offset_minutes":  defTypeRow.AttendanceOpenOffsetMinutes,
				"attendance_close_offset_minutes": defTypeRow.AttendanceCloseOffsetMinutes,
			}

			row.ClassAttendanceSessionTypeSnapshot = snap
		}

	}

	// Tanpa rule → satu sesi di start date
	if len(rr) == 0 {
		dateUTC := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, time.UTC)
		row := sessModel.ClassAttendanceSessionModel{
			ClassAttendanceSessionSchoolID:   sch.ClassScheduleSchoolID,
			ClassAttendanceSessionScheduleID: ptrUUID(sch.ClassScheduleID),
			ClassAttendanceSessionRuleID:     nil,
			// Snapshot2 (rule/CSST/type) dibiarkan NULL
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
		attachCaches(&row, nil)
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
					ClassAttendanceSessionSchoolID:   sch.ClassScheduleSchoolID,
					ClassAttendanceSessionScheduleID: ptrUUID(sch.ClassScheduleID),
					ClassAttendanceSessionRuleID:     &rid,
					// Snapshot rule / CSST / type dibiarkan NULL
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
				rowCSST := r.CSSTID
				attachCaches(&row, rowCSST)
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
) (*uuid.UUID, error) {
	if csstID == nil || *csstID == uuid.Nil {
		return nil, nil
	}

	roomID, sectionID, csstSchoolID, sectionName, err := g.getRoomOrSectionFromCSST(ctx, *csstID)
	if err != nil {
		return nil, err
	}
	if expectSchoolID != uuid.Nil && csstSchoolID != uuid.Nil && expectSchoolID != csstSchoolID {
		return nil, fmt.Errorf("tenant mismatch: csst.school=%s != expect=%s", csstSchoolID, expectSchoolID)
	}

	// 1) Room pada CSST
	if roomID != nil && *roomID != uuid.Nil {
		return roomID, nil
	}

	// 2) Room pada Section
	if sectionID != nil && *sectionID != uuid.Nil {
		rid, secSchoolID, _, er := g.getRoomFromSection(ctx, *sectionID)
		if er == nil {
			if rid != nil && *rid != uuid.Nil {
				return rid, nil
			}
			// 3) Auto-provision
			useSchool := coalesceSchool(secSchoolID, csstSchoolID)
			rid, er2 := g.ensureSectionRoom(ctx, useSchool, *sectionID, sectionName)
			return rid, er2
		}
		if csstSchoolID != uuid.Nil {
			rid, er3 := g.ensureSectionRoom(ctx, csstSchoolID, *sectionID, sectionName)
			return rid, er3
		}
		return nil, err
	}

	// 3) CSST tidak punya Section
	return nil, nil
}

// STATIC, no information_schema: pakai kolom fix
func (g *Generator) getRoomOrSectionFromCSST(
	ctx context.Context,
	csstID uuid.UUID,
) (roomID *uuid.UUID, sectionID *uuid.UUID, schoolID uuid.UUID, sectionName *string, err error) {
	var row struct {
		SchoolID    uuid.UUID  `gorm:"column:school_id"`
		RoomID      *uuid.UUID `gorm:"column:room_id"`
		SectionID   *uuid.UUID `gorm:"column:section_id"`
		SectionName *string    `gorm:"column:section_name"`
	}

	q := `
SELECT
  csst.class_section_subject_teacher_school_id   AS school_id,
  csst.class_section_subject_teacher_class_room_id    AS room_id,
  csst.class_section_subject_teacher_class_section_id AS section_id,
  sec.class_section_name                         AS section_name
FROM class_section_subject_teachers csst
LEFT JOIN class_sections sec
  ON sec.class_section_id = csst.class_section_subject_teacher_class_section_id
WHERE csst.class_section_subject_teacher_id = ?
  AND csst.class_section_subject_teacher_deleted_at IS NULL
LIMIT 1`
	if er := g.DB.WithContext(ctx).Raw(q, csstID).Scan(&row).Error; er != nil {
		return nil, nil, uuid.Nil, nil, er
	}

	return row.RoomID, row.SectionID, row.SchoolID, row.SectionName, nil
}

// STATIC, no information_schema
func (g *Generator) getRoomFromSection(
	ctx context.Context,
	sectionID uuid.UUID,
) (roomID *uuid.UUID, schoolID uuid.UUID, name *string, err error) {
	var row struct {
		SchoolID uuid.UUID  `gorm:"column:school_id"`
		Name     *string    `gorm:"column:section_name"`
		RoomID   *uuid.UUID `gorm:"column:room_id"`
	}

	q := `
SELECT
  s.class_section_school_id AS school_id,
  s.class_section_name      AS section_name,
  s.class_section_class_room_id   AS room_id
FROM class_sections s
WHERE s.class_section_id = ?
  AND s.class_section_deleted_at IS NULL
LIMIT 1`
	if er := g.DB.WithContext(ctx).Raw(q, sectionID).Scan(&row).Error; er != nil {
		return nil, uuid.Nil, nil, er
	}
	return row.RoomID, row.SchoolID, row.Name, nil
}

// Buat Room default untuk Section (bila belum ada sama sekali).
func (g *Generator) ensureSectionRoom(
	ctx context.Context,
	schoolID uuid.UUID,
	sectionID uuid.UUID,
	sectionName *string,
) (*uuid.UUID, error) {
	if schoolID == uuid.Nil || sectionID == uuid.Nil {
		return nil, fmt.Errorf("ensureSectionRoom: schoolID/sectionID kosong")
	}

	baseName := "Ruang"
	if sectionName != nil && strings.TrimSpace(*sectionName) != "" {
		baseName = fmt.Sprintf("Ruang %s", strings.TrimSpace(*sectionName))
	}
	slug := fmt.Sprintf("section-%s", strings.ReplaceAll(sectionID.String(), "-", "")) // unik per section

	// 1) Cek dulu kalau sudah pernah dibuat
	var existing roomModel.ClassRoomModel
	if err := g.DB.WithContext(ctx).
		Where("class_room_school_id = ? AND class_room_slug = ? AND class_room_deleted_at IS NULL",
			schoolID, slug).
		Limit(1).
		Take(&existing).Error; err == nil && existing.ClassRoomID != uuid.Nil {
		id := existing.ClassRoomID
		return &id, nil
	}

	// 2) Belum ada → buat baru
	cr := roomModel.ClassRoomModel{
		ClassRoomSchoolID:  schoolID,
		ClassRoomName:      baseName,
		ClassRoomIsVirtual: false,
		ClassRoomIsActive:  true,
	}
	s := slug
	cr.ClassRoomSlug = &s

	tx := g.DB.WithContext(ctx)
	if err := tx.Create(&cr).Error; err != nil {
		return nil, err
	}

	id := cr.ClassRoomID
	return &id, nil
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

	// --- PARITY (pointer-safe) ---
	parity := ""
	if r.WeekParity != nil {
		parity = strings.ToLower(strings.TrimSpace(*r.WeekParity))
	}
	switch parity {
	case "odd":
		if ((wkAdj/interval)+1)%2 != 1 {
			return false
		}
	case "even":
		if ((wkAdj/interval)+1)%2 != 0 {
			return false
		}
	}

	// Weeks-of-month filter
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

	// Last-week-of-month
	if r.LastWeekOfMonth && !isLastWeekOfMonth(dLocal) {
		return false
	}
	return true
}
