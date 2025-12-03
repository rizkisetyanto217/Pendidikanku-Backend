package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	sessiondto "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/dto"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   Utils
========================= */

func parseYmd(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, err
	}
	tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return &tt, nil
}

func parseUUIDPtr(s string, field string) (*uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, field+" tidak valid")
	}
	return &id, nil
}

func parseUUIDList(raw string) ([]uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "id tidak valid")
		}
		out = append(out, id)
	}
	return out, nil
}

func jsonToMap(j datatypes.JSON) map[string]any {
	if len(j) == 0 {
		return nil
	}
	var m map[string]any
	_ = json.Unmarshal(j, &m)
	return m
}

func queryBoolFlag(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	return raw == "1" || raw == "true" || raw == "yes"
}

func queryBoolFlagInverse(raw string) bool {
	// "0"/"false"/"no" → false
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "0" || raw == "false" || raw == "no" {
		return false
	}
	if raw == "" {
		return true
	}
	return true
}

/* =================================================================
   LIST /admin|/u/class-attendance-sessions
   + support mode:
     - nearest=1 → 3 hari ke depan (by starts_at)
     - mode kalender:
       * default: kemarin–hari ini–besok
       * kalau focus_date di luar range itu → sebulan penuh dari focus_date
     - filter:
       * teacher_id=true  → pakai teacher dari token
       * student_id=true  → pakai student dari token
     - student_timeline=1 → mode khusus timeline murid
     - teacher_timeline=1 → mode khusus timeline guru
================================================================= */

func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// ===== School context: ambil dari token dulu =====
	schoolID, err := resolveSchoolID(c)
	if err != nil {
		return err
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, http.StatusForbidden, "Scope school tidak ditemukan")
	}

	// ===== Mode: timeline (student / teacher) =====
	isStudentTimeline := queryBoolFlag(c.Query("student_timeline"))
	isTeacherTimeline := queryBoolFlag(c.Query("teacher_timeline"))

	// include_session: kalau false → Session dikosongkan
	includeSession := true
	if isStudentTimeline || isTeacherTimeline {
		if !queryBoolFlagInverse(c.Query("include_session")) {
			// pakai flag terbalik: 0/false/no → false
			includeSession = false
		}
	}

	// ===== Guard: role by mode =====
	if isStudentTimeline {
		if err := helperAuth.EnsureStudentSchool(c, schoolID); err != nil {
			return err
		}
	} else if isTeacherTimeline {
		if err := helperAuth.EnsureTeacherSchool(c, schoolID); err != nil {
			return err
		}
	} else {
		if err := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); err != nil {
			return err
		}
	}

	// ----------------------------------------------------------------
	// MODE KHUSUS: STUDENT / TEACHER TIMELINE
	// ----------------------------------------------------------------
	if isStudentTimeline {
		return ctrl.listStudentTimeline(c, schoolID, includeSession)
	}
	if isTeacherTimeline {
		return ctrl.listTeacherTimeline(c, schoolID, includeSession)
	}

	// ----------------------------------------------------------------
	// MODE DEFAULT: LIST SESSIONS (PERILAKU LAMA)
	// ----------------------------------------------------------------
	return ctrl.listSessionsDefault(c, schoolID)
}

/* =========================
   Helpers: school & flags
========================= */

func resolveSchoolID(c *fiber.Ctx) (uuid.UUID, error) {
	var schoolID uuid.UUID

	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else if id, err := helperAuth.GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
			if mc.ID != uuid.Nil {
				schoolID = mc.ID
			} else if strings.TrimSpace(mc.Slug) != "" {
				if sid, e2 := helperAuth.GetSchoolIDBySlug(c, mc.Slug); e2 == nil {
					schoolID = sid
				}
			}
		}
	}

	return schoolID, nil
}

/* =========================================================
   MODE: STUDENT TIMELINE
========================================================= */

func (ctrl *ClassAttendanceSessionController) listStudentTimeline(
	c *fiber.Ctx,
	schoolID uuid.UUID,
	includeSession bool,
) error {
	const defaultUnknownState = "unknown"

	// 1) Ambil school_student_id dari TOKEN (required)
	studentID, err := helperAuth.GetSchoolStudentIDForSchool(c, schoolID)
	if err != nil {
		return err
	}
	if studentID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusForbidden, "school_student_id tidak ditemukan di token")
	}

	// 2) Filter tanggal (pakai pola: date_from/date_to/range/month)
	df, dt, err := resolveTimelineDateRange(c)
	if err != nil {
		return err
	}

	paging := helper.ResolvePaging(c, 20, 200)

	// 3) Filter participant (state/type/is_passed)
	state := strings.ToLower(strings.TrimSpace(c.Query("participant_state")))

	typeIDPtr, err := parseUUIDPtr(c.Query("participant_type_id"), "participant_type_id")
	if err != nil {
		return err
	}

	// 3b) Filter by CSST (optional)
	csstIDPtr, err := parseUUIDPtr(strings.TrimSpace(c.Query("csst_id")), "csst_id")
	if err != nil {
		return err
	}

	var isPassedPtr *bool
	if s := strings.TrimSpace(c.Query("participant_is_passed")); s != "" {
		if b, e := strconv.ParseBool(s); e == nil {
			isPassedPtr = &b
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "participant_is_passed tidak valid (true/false)")
		}
	}

	// 4) Row hasil join session + participant (1 siswa, bisa null)
	type row struct {
		// Session
		SessionID        uuid.UUID  `gorm:"column:class_attendance_session_id"`
		SessionSchoolID  uuid.UUID  `gorm:"column:class_attendance_session_school_id"`
		Date             time.Time  `gorm:"column:class_attendance_session_date"`
		StartsAt         *time.Time `gorm:"column:class_attendance_session_starts_at"`
		EndsAt           *time.Time `gorm:"column:class_attendance_session_ends_at"`
		Title            *string    `gorm:"column:class_attendance_session_title"`
		DisplayTitle     *string    `gorm:"column:class_attendance_session_display_title"`
		Gen              *string    `gorm:"column:class_attendance_session_general_info"`
		Status           string     `gorm:"column:class_attendance_session_status"`
		AttendanceStatus string     `gorm:"column:class_attendance_session_attendance_status"`

		// Caches untuk map ke CompactResponse
		SubjectNameCache *string    `gorm:"column:class_attendance_session_subject_name_cache"`
		SubjectCodeCache *string    `gorm:"column:class_attendance_session_subject_code_cache"`
		SectionNameCache *string    `gorm:"column:class_attendance_session_section_name_cache"`
		RoomNameCache    *string    `gorm:"column:class_attendance_session_room_name_cache"`
		TeacherNameCache *string    `gorm:"column:class_attendance_session_teacher_name_cache"`
		TeacherIDCache   *uuid.UUID `gorm:"column:class_attendance_session_teacher_id_cache"`
		SectionIDCache   *uuid.UUID `gorm:"column:class_attendance_session_section_id_cache"`
		SubjectIDCache   *uuid.UUID `gorm:"column:class_attendance_session_subject_id_cache"`
		CSSTIDCache      *uuid.UUID `gorm:"column:class_attendance_session_csst_id_cache"`

		TypeID   *uuid.UUID     `gorm:"column:class_attendance_session_type_id"`
		TypeSnap datatypes.JSON `gorm:"column:class_attendance_session_type_snapshot"`
		CSSTSnap datatypes.JSON `gorm:"column:class_attendance_session_csst_snapshot"`

		// Participant (bisa NULL)
		ParticipantID    *uuid.UUID `gorm:"column:class_attendance_session_participant_id"`
		ParticipantState *string    `gorm:"column:class_attendance_session_participant_state"`
	}

	db := ctrl.DB

	// 5) Base query: mulai dari SESSIONS, left join participant 1 murid
	q := db.Table("class_attendance_sessions AS cas").
		Where("cas.class_attendance_session_school_id = ?", schoolID).
		Where("cas.class_attendance_session_deleted_at IS NULL").
		Joins(`
        JOIN student_class_section_subject_teachers scst
          ON scst.student_class_section_subject_teacher_school_id = cas.class_attendance_session_school_id
         AND scst.student_class_section_subject_teacher_csst_id = cas.class_attendance_session_csst_id
         AND scst.student_class_section_subject_teacher_student_id = ?
         AND scst.student_class_section_subject_teacher_is_active = TRUE
         AND scst.student_class_section_subject_teacher_deleted_at IS NULL
    `, studentID)

	// Filter tanggal di level session
	if df != nil && dt != nil {
		q = q.Where("cas.class_attendance_session_date BETWEEN ? AND ?", *df, *dt)
	} else if df != nil {
		q = q.Where("cas.class_attendance_session_date >= ?", *df)
	} else if dt != nil {
		q = q.Where("cas.class_attendance_session_date <= ?", *dt)
	}

	// 🔹 Filter CSST (opsional)
	if csstIDPtr != nil {
		q = q.Where("cas.class_attendance_session_csst_id = ?", *csstIDPtr)
	}

	// LEFT JOIN participants khusus murid ini
	q = q.Joins(`
		LEFT JOIN class_attendance_session_participants AS p
		       ON p.class_attendance_session_participant_session_id = cas.class_attendance_session_id
		      AND p.class_attendance_session_participant_school_id = cas.class_attendance_session_school_id
		      AND p.class_attendance_session_participant_school_student_id = ?
		      AND p.class_attendance_session_participant_kind = 'student'
		      AND p.class_attendance_session_participant_deleted_at IS NULL
	`, studentID)

	// Filter participant di level join (akan otomatis menghilangkan sesi yang tidak punya participant jika filter dipakai)
	if state != "" {
		// enum, cukup bandingkan langsung tanpa LOWER()
		q = q.Where("p.class_attendance_session_participant_state = ?", state)
	}

	if typeIDPtr != nil {
		q = q.Where("p.class_attendance_session_participant_type_id = ?", *typeIDPtr)
	}
	if isPassedPtr != nil {
		q = q.Where("p.class_attendance_session_participant_is_passed = ?", *isPassedPtr)
	}

	// 6) Hitung total (distinct per session)
	var total int64
	if err := q.Session(&gorm.Session{}).
		Select("cas.class_attendance_session_id").
		Distinct().
		Count(&total).Error; err != nil {

		pg := helper.BuildPaginationFromOffset(0, paging.Offset, paging.Limit)
		empty := []sessiondto.StudentSessionAttendanceItem{}
		return helper.JsonList(c, "data tidak ditemukan", empty, pg)
	}

	// 7) Ambil page data
	var rows []row
	if err := q.
		Select(`
			cas.class_attendance_session_id,
			cas.class_attendance_session_school_id,
			cas.class_attendance_session_date,
			cas.class_attendance_session_starts_at,
			cas.class_attendance_session_ends_at,
			cas.class_attendance_session_title,
			cas.class_attendance_session_display_title,
			cas.class_attendance_session_general_info,
			cas.class_attendance_session_status,
			cas.class_attendance_session_attendance_status,
			cas.class_attendance_session_subject_name_cache,
			cas.class_attendance_session_subject_code_cache,
			cas.class_attendance_session_section_name_cache,
			cas.class_attendance_session_room_name_cache,
			cas.class_attendance_session_teacher_name_cache,
			cas.class_attendance_session_teacher_id_cache,
			cas.class_attendance_session_section_id_cache,
			cas.class_attendance_session_subject_id_cache,
			cas.class_attendance_session_csst_id_cache,
			cas.class_attendance_session_type_id,
			cas.class_attendance_session_type_snapshot,
			cas.class_attendance_session_csst_snapshot,
			p.class_attendance_session_participant_id,
			p.class_attendance_session_participant_state
		`).
		Order(`
			cas.class_attendance_session_date ASC,
			cas.class_attendance_session_starts_at ASC NULLS LAST,
			cas.class_attendance_session_id ASC
		`).
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&rows).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil timeline kehadiran siswa")
	}

	// 8) Map ke DTO StudentSessionAttendanceItem
	items := make([]sessiondto.StudentSessionAttendanceItem, 0, len(rows))

	for _, r := range rows {
		gen := ""
		if r.Gen != nil {
			gen = *r.Gen
		}

		item := sessiondto.StudentSessionAttendanceItem{}

		if includeSession {
			sess := sessiondto.ClassAttendanceSessionCompactResponse{
				ClassAttendanceSessionId:       r.SessionID,
				ClassAttendanceSessionSchoolId: r.SessionSchoolID,

				ClassAttendanceSessionDate:     r.Date,
				ClassAttendanceSessionStartsAt: r.StartsAt,
				ClassAttendanceSessionEndsAt:   r.EndsAt,

				ClassAttendanceSessionTitle:        r.Title,
				ClassAttendanceSessionDisplayTitle: r.DisplayTitle,
				ClassAttendanceSessionGeneralInfo:  gen,

				ClassAttendanceSessionStatus:           r.Status,
				ClassAttendanceSessionAttendanceStatus: r.AttendanceStatus,

				ClassAttendanceSessionSubjectNameCache: r.SubjectNameCache,
				ClassAttendanceSessionSubjectCodeCache: r.SubjectCodeCache,
				ClassAttendanceSessionSectionNameCache: r.SectionNameCache,
				ClassAttendanceSessionRoomNameCache:    r.RoomNameCache,
				ClassAttendanceSessionTeacherNameCache: r.TeacherNameCache,
				ClassAttendanceSessionTeacherIdCache:   r.TeacherIDCache,
				ClassAttendanceSessionSectionIdCache:   r.SectionIDCache,
				ClassAttendanceSessionSubjectIdCache:   r.SubjectIDCache,
				ClassAttendanceSessionCSSTIdCache:      r.CSSTIDCache,

				ClassAttendanceSessionCSSTSnapshot: jsonToMap(r.CSSTSnap),

				ClassAttendanceSessionTypeId:       r.TypeID,
				ClassAttendanceSessionTypeSnapshot: jsonToMap(r.TypeSnap),
			}
			item.Session = sess
		}

		// Participant: kalau tidak ada row, state = "unknown"
		if r.ParticipantID != nil {
			item.Participant.ID = *r.ParticipantID
		}

		if r.ParticipantState != nil && strings.TrimSpace(*r.ParticipantState) != "" {
			item.Participant.State = *r.ParticipantState
		} else {
			item.Participant.State = defaultUnknownState
		}

		items = append(items, item)
	}

	pg := helper.BuildPaginationFromOffset(total, paging.Offset, paging.Limit)
	return helper.JsonList(c, "ok", items, pg)
}

/* =========================================================
   MODE: TEACHER TIMELINE
   (mirip student_timeline tapi anchor di guru)
========================================================= */

func (ctrl *ClassAttendanceSessionController) listTeacherTimeline(
	c *fiber.Ctx,
	schoolID uuid.UUID,
	includeSession bool,
) error {
	const defaultUnknownState = "unknown"

	// 1) Ambil school_teacher_id dari TOKEN (required)
	teacherID, err := helperAuth.GetSchoolTeacherIDForSchool(c, schoolID)
	if err != nil {
		return err
	}
	if teacherID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusForbidden, "school_teacher_id tidak ditemukan di token")
	}

	// 2) Filter tanggal (pakai pola: date_from/date_to/range/month)
	df, dt, err := resolveTimelineDateRange(c)
	if err != nil {
		return err
	}

	paging := helper.ResolvePaging(c, 20, 200)

	// 3) Filter participant (state/type/is_passed)
	state := strings.ToLower(strings.TrimSpace(c.Query("participant_state")))

	typeIDPtr, err := parseUUIDPtr(c.Query("participant_type_id"), "participant_type_id")
	if err != nil {
		return err
	}

	var isPassedPtr *bool
	if s := strings.TrimSpace(c.Query("participant_is_passed")); s != "" {
		if b, e := strconv.ParseBool(s); e == nil {
			isPassedPtr = &b
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "participant_is_passed tidak valid (true/false)")
		}
	}

	// 3b) Filter by CSST (optional)
	csstIDPtr, err := parseUUIDPtr(strings.TrimSpace(c.Query("csst_id")), "csst_id")
	if err != nil {
		return err
	}

	// 4) Row hasil join session + participant (1 guru, bisa null)
	type row struct {
		// Session
		SessionID        uuid.UUID  `gorm:"column:class_attendance_session_id"`
		SessionSchoolID  uuid.UUID  `gorm:"column:class_attendance_session_school_id"`
		Date             time.Time  `gorm:"column:class_attendance_session_date"`
		StartsAt         *time.Time `gorm:"column:class_attendance_session_starts_at"`
		EndsAt           *time.Time `gorm:"column:class_attendance_session_ends_at"`
		Title            *string    `gorm:"column:class_attendance_session_title"`
		DisplayTitle     *string    `gorm:"column:class_attendance_session_display_title"`
		Gen              *string    `gorm:"column:class_attendance_session_general_info"`
		Status           string     `gorm:"column:class_attendance_session_status"`
		AttendanceStatus string     `gorm:"column:class_attendance_session_attendance_status"`

		// Caches
		SubjectNameCache *string    `gorm:"column:class_attendance_session_subject_name_cache"`
		SubjectCodeCache *string    `gorm:"column:class_attendance_session_subject_code_cache"`
		SectionNameCache *string    `gorm:"column:class_attendance_session_section_name_cache"`
		RoomNameCache    *string    `gorm:"column:class_attendance_session_room_name_cache"`
		TeacherNameCache *string    `gorm:"column:class_attendance_session_teacher_name_cache"`
		TeacherIDCache   *uuid.UUID `gorm:"column:class_attendance_session_teacher_id_cache"`
		SectionIDCache   *uuid.UUID `gorm:"column:class_attendance_session_section_id_cache"`
		SubjectIDCache   *uuid.UUID `gorm:"column:class_attendance_session_subject_id_cache"`
		CSSTIDCache      *uuid.UUID `gorm:"column:class_attendance_session_csst_id_cache"`

		TypeID   *uuid.UUID     `gorm:"column:class_attendance_session_type_id"`
		TypeSnap datatypes.JSON `gorm:"column:class_attendance_session_type_snapshot"`
		CSSTSnap datatypes.JSON `gorm:"column:class_attendance_session_csst_snapshot"`

		// Participant guru (bisa NULL)
		ParticipantID          *uuid.UUID `gorm:"column:class_attendance_session_participant_id"`
		ParticipantState       *string    `gorm:"column:class_attendance_session_participant_state"`
		ParticipantTeacherRole *string    `gorm:"column:class_attendance_session_participant_teacher_role"`
	}

	db := ctrl.DB

	// 5) Base query: sessions yang memang dia ajar
	q := db.Table("class_attendance_sessions AS cas").
		Where("cas.class_attendance_session_school_id = ?", schoolID).
		Where("cas.class_attendance_session_deleted_at IS NULL").
		Where("cas.class_attendance_session_teacher_id = ?", teacherID)

	// Filter tanggal di level session
	if df != nil && dt != nil {
		q = q.Where("cas.class_attendance_session_date BETWEEN ? AND ?", *df, *dt)
	} else if df != nil {
		q = q.Where("cas.class_attendance_session_date >= ?", *df)
	} else if dt != nil {
		q = q.Where("cas.class_attendance_session_date <= ?", *dt)
	}

	// 🔹 Filter CSST (opsional)
	if csstIDPtr != nil {
		q = q.Where("cas.class_attendance_session_csst_id = ?", *csstIDPtr)
	}

	// LEFT JOIN participants khusus guru ini
	q = q.Joins(`
		LEFT JOIN class_attendance_session_participants AS p
		       ON p.class_attendance_session_participant_session_id = cas.class_attendance_session_id
		      AND p.class_attendance_session_participant_school_id = cas.class_attendance_session_school_id
		      AND p.class_attendance_session_participant_school_teacher_id = ?
		      AND p.class_attendance_session_participant_kind = 'teacher'
		      AND p.class_attendance_session_participant_deleted_at IS NULL
	`, teacherID)

	// Filter participant di level join
	if state != "" {
		// enum attendance_state_enum
		q = q.Where("p.class_attendance_session_participant_state = ?", state)
	}

	if typeIDPtr != nil {
		q = q.Where("p.class_attendance_session_participant_type_id = ?", *typeIDPtr)
	}
	if isPassedPtr != nil {
		q = q.Where("p.class_attendance_session_participant_is_passed = ?", *isPassedPtr)
	}

	// 6) Hitung total (distinct per session)
	var total int64
	if err := q.Session(&gorm.Session{}).
		Select("cas.class_attendance_session_id").
		Distinct().
		Count(&total).Error; err != nil {

		pg := helper.BuildPaginationFromOffset(0, paging.Offset, paging.Limit)
		empty := []sessiondto.TeacherSessionAttendanceItem{}
		return helper.JsonList(c, "data tidak ditemukan", empty, pg)
	}

	// 7) Ambil page data
	var rows []row
	if err := q.
		Select(`
			cas.class_attendance_session_id,
			cas.class_attendance_session_school_id,
			cas.class_attendance_session_date,
			cas.class_attendance_session_starts_at,
			cas.class_attendance_session_ends_at,
			cas.class_attendance_session_title,
			cas.class_attendance_session_display_title,
			cas.class_attendance_session_general_info,
			cas.class_attendance_session_status,
			cas.class_attendance_session_attendance_status,
			cas.class_attendance_session_subject_name_cache,
			cas.class_attendance_session_subject_code_cache,
			cas.class_attendance_session_section_name_cache,
			cas.class_attendance_session_room_name_cache,
			cas.class_attendance_session_teacher_name_cache,
			cas.class_attendance_session_teacher_id_cache,
			cas.class_attendance_session_section_id_cache,
			cas.class_attendance_session_subject_id_cache,
			cas.class_attendance_session_csst_id_cache,
			cas.class_attendance_session_type_id,
			cas.class_attendance_session_type_snapshot,
			cas.class_attendance_session_csst_snapshot,
			p.class_attendance_session_participant_id,
			p.class_attendance_session_participant_state,
			p.class_attendance_session_participant_teacher_role
		`).
		Order(`
			cas.class_attendance_session_date ASC,
			cas.class_attendance_session_starts_at ASC NULLS LAST,
			cas.class_attendance_session_id ASC
		`).
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&rows).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil timeline kehadiran guru")
	}

	// 8) Map ke DTO TeacherSessionAttendanceItem
	items := make([]sessiondto.TeacherSessionAttendanceItem, 0, len(rows))

	for _, r := range rows {
		gen := ""
		if r.Gen != nil {
			gen = *r.Gen
		}

		item := sessiondto.TeacherSessionAttendanceItem{}

		if includeSession {
			sess := sessiondto.ClassAttendanceSessionCompactResponse{
				ClassAttendanceSessionId:       r.SessionID,
				ClassAttendanceSessionSchoolId: r.SessionSchoolID,

				ClassAttendanceSessionDate:     r.Date,
				ClassAttendanceSessionStartsAt: r.StartsAt,
				ClassAttendanceSessionEndsAt:   r.EndsAt,

				ClassAttendanceSessionTitle:        r.Title,
				ClassAttendanceSessionDisplayTitle: r.DisplayTitle,
				ClassAttendanceSessionGeneralInfo:  gen,

				ClassAttendanceSessionStatus:           r.Status,
				ClassAttendanceSessionAttendanceStatus: r.AttendanceStatus,

				ClassAttendanceSessionSubjectNameCache: r.SubjectNameCache,
				ClassAttendanceSessionSubjectCodeCache: r.SubjectCodeCache,
				ClassAttendanceSessionSectionNameCache: r.SectionNameCache,
				ClassAttendanceSessionRoomNameCache:    r.RoomNameCache,
				ClassAttendanceSessionTeacherNameCache: r.TeacherNameCache,
				ClassAttendanceSessionTeacherIdCache:   r.TeacherIDCache,
				ClassAttendanceSessionSectionIdCache:   r.SectionIDCache,
				ClassAttendanceSessionSubjectIdCache:   r.SubjectIDCache,
				ClassAttendanceSessionCSSTIdCache:      r.CSSTIDCache,

				ClassAttendanceSessionCSSTSnapshot: jsonToMap(r.CSSTSnap),

				ClassAttendanceSessionTypeId:       r.TypeID,
				ClassAttendanceSessionTypeSnapshot: jsonToMap(r.TypeSnap),
			}
			item.Session = sess
		}

		// Participant: kalau tidak ada row, state = "unknown"
		if r.ParticipantID != nil {
			item.Participant.ID = *r.ParticipantID
		}
		if r.ParticipantState != nil && strings.TrimSpace(*r.ParticipantState) != "" {
			item.Participant.State = *r.ParticipantState
		} else {
			item.Participant.State = defaultUnknownState
		}
		if r.ParticipantTeacherRole != nil {
			item.Participant.TeacherRole = r.ParticipantTeacherRole
		}

		items = append(items, item)
	}

	pg := helper.BuildPaginationFromOffset(total, paging.Offset, paging.Limit)
	return helper.JsonList(c, "ok", items, pg)
}

/* =========================================================
   TIMELINE DATE RANGE (dipakai student & teacher)
========================================================= */

func resolveTimelineDateRange(c *fiber.Ctx) (*time.Time, *time.Time, error) {
	var df, dt *time.Time
	var err error

	monthRaw := strings.TrimSpace(c.Query("month"))
	rangeRaw := strings.ToLower(strings.TrimSpace(c.Query("range")))

	// date_from/date_to (optional)
	if s := strings.TrimSpace(c.Query("date_from")); s != "" {
		df, err = parseYmd(s)
		if err != nil {
			return nil, nil, helper.JsonError(c, fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
		}
	}
	if s := strings.TrimSpace(c.Query("date_to")); s != "" {
		dt, err = parseYmd(s)
		if err != nil {
			return nil, nil, helper.JsonError(c, fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
		}
	}

	// range: today / week / sepekan / next7
	isTodayRange :=
		rangeRaw == "today" ||
			rangeRaw == "hari_ini" ||
			rangeRaw == "today_only"

	isWeekRange :=
		rangeRaw == "week" ||
			rangeRaw == "next7" ||
			rangeRaw == "sepekan"

	if isTodayRange || isWeekRange {
		now := time.Now().In(time.Local)
		todayLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

		start := todayLocal
		var end time.Time
		if isTodayRange {
			end = todayLocal
		} else {
			end = todayLocal.AddDate(0, 0, 7)
		}

		df = &start
		dt = &end
		monthRaw = ""
	}

	// month=YYYY-MM → override ke full 1 bulan
	if monthRaw != "" && !isTodayRange && !isWeekRange {
		mt, err2 := time.ParseInLocation("2006-01", monthRaw, time.Local)
		if err2 != nil {
			return nil, nil, helper.JsonError(c, fiber.StatusBadRequest, "month tidak valid (YYYY-MM)")
		}
		firstOfMonth := time.Date(mt.Year(), mt.Month(), 1, 0, 0, 0, 0, time.Local)
		lastOfMonth := time.Date(mt.Year(), mt.Month()+1, 0, 0, 0, 0, 0, time.Local)

		df = &firstOfMonth
		dt = &lastOfMonth
	}

	return df, dt, nil
}

/* =========================================================
   MODE: DEFAULT (ADMIN/TEACHER VIEW)
========================================================= */

func (ctrl *ClassAttendanceSessionController) listSessionsDefault(
	c *fiber.Ctx,
	schoolID uuid.UUID,
) error {
	// Roles (dipakai untuk scope participants & filter self)
	userID, _ := helperAuth.GetUserIDFromToken(c)
	adminSchoolID, _ := helperAuth.GetSchoolIDFromToken(c)
	teacherSchoolID, _ := helperAuth.GetSchoolIDFromTokenPreferTeacher(c)
	teacherIDFromToken, _ := helperAuth.GetTeacherIDFromToken(c)

	isAdmin := (adminSchoolID != uuid.Nil && adminSchoolID == schoolID) ||
		helperAuth.HasRoleInSchool(c, schoolID, "admin") ||
		helperAuth.HasRoleInSchool(c, schoolID, "dkm") ||
		helperAuth.IsDKMInSchool(c, schoolID)

	isTeacher := (teacherSchoolID != uuid.Nil && teacherSchoolID == schoolID) ||
		helperAuth.HasRoleInSchool(c, schoolID, "teacher") ||
		helperAuth.IsTeacherInSchool(c, schoolID)

	// Includes
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeAll := includeStr == "all"
	includeSet := map[string]bool{}
	for _, part := range strings.Split(includeStr, ",") {
		if p := strings.TrimSpace(part); p != "" {
			includeSet[p] = true
		}
	}
	wantParticipants :=
		includeAll ||
			includeSet["participants"] ||
			includeSet["participant"] ||
			includeSet["session_participants"] ||
			includeSet["session_participant"]

	// Mode view (full / compact)
	modeRaw := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	wantCompact := modeRaw == "compact"

	// Pagination
	p := helper.ResolvePaging(c, 20, 200)

	// Sorting whitelist
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	order := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	col := "cas.class_attendance_session_date" // default
	switch sortBy {
	case "title":
		col = "cas.class_attendance_session_title"
	case "date", "":
		col = "cas.class_attendance_session_date"
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// MODE NEAREST (3 hari ke depan, urut paling dekat sekarang)
	nearestRaw := strings.ToLower(strings.TrimSpace(c.Query("nearest")))
	isNearest := nearestRaw == "1" || nearestRaw == "true" || nearestRaw == "yes"

	// Filters dasar (date_from, date_to, focus_date, month, range)
	var df, dt *time.Time
	var focusDate *time.Time

	monthRaw := strings.TrimSpace(c.Query("month"))
	rangeRaw := strings.ToLower(strings.TrimSpace(c.Query("range")))
	var err error

	if !isNearest {
		df, err = parseYmd(c.Query("date_from"))
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
		}
		dt, err = parseYmd(c.Query("date_to"))
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
		}

		focusDate, err = parseYmd(c.Query("focus_date"))
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "focus_date tidak valid (YYYY-MM-DD)")
		}
	}

	// Special range: today / week
	isTodayRange :=
		rangeRaw == "today" ||
			rangeRaw == "hari_ini" ||
			rangeRaw == "today_only"

	isWeekRange :=
		rangeRaw == "week" ||
			rangeRaw == "next7" ||
			rangeRaw == "sepekan"

	if isTodayRange || isWeekRange {
		now := time.Now().In(time.Local)
		todayLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

		start := todayLocal
		var end time.Time
		if isTodayRange {
			end = todayLocal
		} else {
			end = todayLocal.AddDate(0, 0, 7)
		}

		df = &start
		dt = &end

		focusDate = nil
		monthRaw = ""
	}

	// Kalau FE kirim month=YYYY-MM → pakai full 1 bulan
	if monthRaw != "" && !isTodayRange && !isWeekRange {
		mt, err2 := time.ParseInLocation("2006-01", monthRaw, time.Local)
		if err2 != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "month tidak valid (YYYY-MM)")
		}
		firstOfMonth := time.Date(mt.Year(), mt.Month(), 1, 0, 0, 0, 0, time.Local)
		lastOfMonth := time.Date(mt.Year(), mt.Month()+1, 0, 0, 0, 0, 0, time.Local)

		df = &firstOfMonth
		dt = &lastOfMonth
	}

	// Flag filter self by token (teacher/student)
	teacherFlag := strings.ToLower(strings.TrimSpace(c.Query("teacher_id")))
	studentFlag := strings.ToLower(strings.TrimSpace(c.Query("student_id")))

	wantTeacherSelf :=
		teacherFlag == "1" ||
			teacherFlag == "true" ||
			teacherFlag == "yes" ||
			teacherFlag == "me"

	wantStudentSelf :=
		studentFlag == "1" ||
			studentFlag == "true" ||
			studentFlag == "yes" ||
			studentFlag == "me"

	if wantTeacherSelf && wantStudentSelf {
		return helper.JsonError(c, fiber.StatusBadRequest, "Tidak boleh sekaligus filter teacher_id dan student_id")
	}

	if strings.TrimSpace(c.Query("section_id")) != "" || strings.TrimSpace(c.Query("class_subject_id")) != "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Filter section_id / class_subject_id belum didukung skema jadwal saat ini")
	}

	// schedule (opsional) & dukung null
	scheduleRaw := strings.TrimSpace(c.Query("schedule_id"))
	scheduleIDPtr, err := parseUUIDPtr(scheduleRaw, "schedule_id")
	if err != nil {
		return err
	}
	wantScheduleNull := strings.EqualFold(scheduleRaw, "null") || strings.EqualFold(scheduleRaw, "nil")

	// keyword
	var like *string
	if kw := strings.TrimSpace(c.Query("q")); kw != "" {
		pat := "%" + kw + "%"
		like = &pat
	}

	// filter by id(s)
	var sessionIDs []uuid.UUID
	if ids, err := parseUUIDList(c.Query("id")); err != nil {
		return err
	} else if len(ids) > 0 {
		sessionIDs = ids
	} else if ids, err := parseUUIDList(c.Query("session_id")); err != nil {
		return err
	} else if len(ids) > 0 {
		sessionIDs = ids
	} else if ids, err := parseUUIDList(c.Query("cas_id")); err != nil {
		return err
	} else if len(ids) > 0 {
		sessionIDs = ids
	}

	// Base query
	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Where("cas.class_attendance_session_school_id = ?", schoolID).
		Where("cas.class_attendance_session_deleted_at IS NULL")

	// Filter self by token (teacher/student)
	if wantTeacherSelf {
		if userID == uuid.Nil {
			return helper.JsonError(c, http.StatusUnauthorized, "User tidak terautentik")
		}
		if !isTeacher {
			return helper.JsonError(c, http.StatusForbidden, "Filter teacher_id hanya untuk guru di sekolah ini")
		}
		if teacherIDFromToken == uuid.Nil {
			return helper.JsonError(c, http.StatusForbidden, "Token tidak memiliki teacher_id")
		}

		qBase = qBase.Where("cas.class_attendance_session_teacher_id = ?", teacherIDFromToken)
	}

	if wantStudentSelf {
		if userID == uuid.Nil {
			return helper.JsonError(c, http.StatusUnauthorized, "User tidak terautentik")
		}

		qBase = qBase.
			Joins(`
				JOIN class_attendance_session_participants sp
				  ON sp.class_attendance_session_participant_session_id = cas.class_attendance_session_id
				 AND sp.class_attendance_session_participant_deleted_at IS NULL
				 AND sp.class_attendance_session_participant_school_id = cas.class_attendance_session_school_id
			`).
			Joins(`
				JOIN school_students ms
				  ON ms.school_student_id = sp.class_attendance_session_participant_school_student_id
				 AND ms.school_student_deleted_at IS NULL
				 AND ms.school_student_school_id = cas.class_attendance_session_school_id
			`).
			Where("ms.school_student_user_id = ?", userID)
	}

	// Filter waktu
	if isNearest {
		now := time.Now()
		threeDaysLater := now.AddDate(0, 0, 3)

		qBase = qBase.Where(`
			cas.class_attendance_session_starts_at IS NOT NULL
			AND cas.class_attendance_session_starts_at >= ?
			AND cas.class_attendance_session_starts_at <= ?
		`, now, threeDaysLater)

		orderExpr = "cas.class_attendance_session_starts_at ASC, cas.class_attendance_session_date ASC, cas.class_attendance_session_id ASC"
	} else {
		if df == nil && dt == nil {
			now := time.Now()
			todayLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			yesterday := todayLocal.AddDate(0, 0, -1)
			tomorrow := todayLocal.AddDate(0, 0, 1)

			if focusDate != nil {
				fdLocal := time.Date(focusDate.Year(), focusDate.Month(), focusDate.Day(), 0, 0, 0, 0, time.Local)

				if fdLocal.Before(yesterday) || fdLocal.After(tomorrow) {
					firstOfMonth := time.Date(fdLocal.Year(), fdLocal.Month(), 1, 0, 0, 0, 0, time.Local)
					lastOfMonth := time.Date(fdLocal.Year(), fdLocal.Month()+1, 0, 0, 0, 0, 0, time.Local)

					df = &firstOfMonth
					dt = &lastOfMonth
				} else {
					df = &yesterday
					dt = &tomorrow
				}
			} else {
				df = &yesterday
				dt = &tomorrow
			}
		}

		if df != nil && dt != nil {
			qBase = qBase.Where("cas.class_attendance_session_date BETWEEN ? AND ?", *df, *dt)
		} else if df != nil {
			qBase = qBase.Where("cas.class_attendance_session_date >= ?", *df)
		} else if dt != nil {
			qBase = qBase.Where("cas.class_attendance_session_date <= ?", *dt)
		}
	}

	if scheduleIDPtr != nil {
		qBase = qBase.Where("cas.class_attendance_session_schedule_id = ?", *scheduleIDPtr)
	} else if wantScheduleNull {
		qBase = qBase.Where(`cas.class_attendance_session_schedule_id IS NULL OR cas.class_attendance_session_schedule_id = '00000000-0000-0000-0000-000000000000'`)
	}

	if len(sessionIDs) > 0 {
		qBase = qBase.Where("cas.class_attendance_session_id IN ?", sessionIDs)
	}
	if like != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_session_title ILIKE ?
			 OR cas.class_attendance_session_general_info ILIKE ?
             OR cas.class_attendance_session_display_title ILIKE ?)`, *like, *like, *like)
	}

	// Types untuk hasil query utama & participants
	type row struct {
		ID       uuid.UUID `gorm:"column:class_attendance_session_id"`
		SchoolID uuid.UUID `gorm:"column:class_attendance_session_school_id"`

		ScheduleID *uuid.UUID `gorm:"column:class_attendance_session_schedule_id"`
		RuleID     *uuid.UUID `gorm:"column:class_attendance_session_rule_id"`

		TypeID   *uuid.UUID     `gorm:"column:class_attendance_session_type_id"`
		TypeSnap datatypes.JSON `gorm:"column:class_attendance_session_type_snapshot"`

		Slug *string `gorm:"column:class_attendance_session_slug"`

		Date     time.Time  `gorm:"column:class_attendance_session_date"`
		StartsAt *time.Time `gorm:"column:class_attendance_session_starts_at"`
		EndsAt   *time.Time `gorm:"column:class_attendance_session_ends_at"`

		Status           string `gorm:"column:class_attendance_session_status"`
		AttendanceStatus string `gorm:"column:class_attendance_session_attendance_status"`
		Locked           bool   `gorm:"column:class_attendance_session_locked"`
		IsOverride       bool   `gorm:"column:class_attendance_session_is_override"`
		IsCanceled       bool   `gorm:"column:class_attendance_session_is_canceled"`

		OriginalStartAt *time.Time `gorm:"column:class_attendance_session_original_start_at"`
		OriginalEndAt   *time.Time `gorm:"column:class_attendance_session_original_end_at"`
		Kind            *string    `gorm:"column:class_attendance_session_kind"`
		OverrideReason  *string    `gorm:"column:class_attendance_session_override_reason"`
		OverrideEventID *uuid.UUID `gorm:"column:class_attendance_session_override_event_id"`

		TeacherID *uuid.UUID `gorm:"column:class_attendance_session_teacher_id"`
		RoomID    *uuid.UUID `gorm:"column:class_attendance_session_class_room_id"`

		CSSTID *uuid.UUID `gorm:"column:class_attendance_session_csst_id"`

		Title        *string `gorm:"column:class_attendance_session_title"`
		DisplayTitle *string `gorm:"column:class_attendance_session_display_title"`
		Gen          *string `gorm:"column:class_attendance_session_general_info"`
		Note         *string `gorm:"column:class_attendance_session_note"`

		PresentCount *int `gorm:"column:class_attendance_session_present_count"`
		AbsentCount  *int `gorm:"column:class_attendance_session_absent_count"`
		LateCount    *int `gorm:"column:class_attendance_session_late_count"`
		ExcusedCount *int `gorm:"column:class_attendance_session_excused_count"`
		SickCount    *int `gorm:"column:class_attendance_session_sick_count"`
		LeaveCount   *int `gorm:"column:class_attendance_session_leave_count"`

		CSSTSnap datatypes.JSON `gorm:"column:class_attendance_session_csst_snapshot"`

		// CACHE kolom turunan dari snapshot
		CSSTIDCache      *uuid.UUID `gorm:"column:class_attendance_session_csst_id_cache"`
		SubjectIDCache   *uuid.UUID `gorm:"column:class_attendance_session_subject_id_cache"`
		SectionIDCache   *uuid.UUID `gorm:"column:class_attendance_session_section_id_cache"`
		TeacherIDCache   *uuid.UUID `gorm:"column:class_attendance_session_teacher_id_cache"`
		RoomIDCache      *uuid.UUID `gorm:"column:class_attendance_session_room_id_cache"`
		SubjectCodeCache *string    `gorm:"column:class_attendance_session_subject_code_cache"`
		SubjectNameCache *string    `gorm:"column:class_attendance_session_subject_name_cache"`
		SectionNameCache *string    `gorm:"column:class_attendance_session_section_name_cache"`
		TeacherNameCache *string    `gorm:"column:class_attendance_session_teacher_name_cache"`
		RoomNameCache    *string    `gorm:"column:class_attendance_session_room_name_cache"`

		RuleSnapshot        datatypes.JSON `gorm:"column:class_attendance_session_rule_snapshot"`
		RuleDayOfWeekCache  *int           `gorm:"column:class_attendance_session_rule_day_of_week_cache"`
		RuleStartTimeCache  *string        `gorm:"column:class_attendance_session_rule_start_time_cache"`
		RuleEndTimeCache    *string        `gorm:"column:class_attendance_session_rule_end_time_cache"`
		RuleWeekParityCache *string        `gorm:"column:class_attendance_session_rule_week_parity_cache"`

		CreatedAt time.Time  `gorm:"column:class_attendance_session_created_at"`
		UpdatedAt time.Time  `gorm:"column:class_attendance_session_updated_at"`
		DeletedAt *time.Time `gorm:"column:class_attendance_session_deleted_at"`
	}

	// Total
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_session_id").
		Count(&total).Error; err != nil {

		pg := helper.BuildPaginationFromOffset(0, p.Offset, p.Limit)

		if wantParticipants {
			type SessionWithParticipants struct {
				sessiondto.ClassAttendanceSessionResponse
				Participants []SessionParticipantLite `json:"participants,omitempty"`
			}
			empty := []SessionWithParticipants{}
			return helper.JsonList(c, "data tidak ditemukan", empty, pg)
		}

		if wantCompact {
			empty := []sessiondto.ClassAttendanceSessionCompactResponse{}
			return helper.JsonList(c, "data tidak ditemukan", empty, pg)
		}

		empty := []sessiondto.ClassAttendanceSessionResponse{}
		return helper.JsonList(c, "data tidak ditemukan", empty, pg)
	}

	// Page data
	var rows []row

	qSelect := qBase.
		Select(`
			cas.class_attendance_session_id,
			cas.class_attendance_session_school_id,
			cas.class_attendance_session_schedule_id,
			cas.class_attendance_session_rule_id,
			cas.class_attendance_session_type_id,
			cas.class_attendance_session_type_snapshot,
			cas.class_attendance_session_slug,
			cas.class_attendance_session_date,
			cas.class_attendance_session_starts_at,
			cas.class_attendance_session_ends_at,
			cas.class_attendance_session_status,
			cas.class_attendance_session_attendance_status,
			cas.class_attendance_session_locked,
			cas.class_attendance_session_is_override,
			cas.class_attendance_session_is_canceled,
			cas.class_attendance_session_original_start_at,
			cas.class_attendance_session_original_end_at,
			cas.class_attendance_session_kind,
			cas.class_attendance_session_override_reason,
			cas.class_attendance_session_override_event_id,
			cas.class_attendance_session_teacher_id,
			cas.class_attendance_session_class_room_id,
			cas.class_attendance_session_csst_id,
			cas.class_attendance_session_title,
			cas.class_attendance_session_display_title,
			cas.class_attendance_session_general_info,
			cas.class_attendance_session_note,
			cas.class_attendance_session_present_count,
			cas.class_attendance_session_absent_count,
			cas.class_attendance_session_late_count,
			cas.class_attendance_session_excused_count,
			cas.class_attendance_session_sick_count,
			cas.class_attendance_session_leave_count,
			cas.class_attendance_session_csst_snapshot,
			cas.class_attendance_session_csst_id_cache,
			cas.class_attendance_session_subject_id_cache,
			cas.class_attendance_session_section_id_cache,
			cas.class_attendance_session_teacher_id_cache,
			cas.class_attendance_session_room_id_cache,
			cas.class_attendance_session_subject_code_cache,
			cas.class_attendance_session_subject_name_cache,
			cas.class_attendance_session_section_name_cache,
			cas.class_attendance_session_teacher_name_cache,
			cas.class_attendance_session_room_name_cache,
			cas.class_attendance_session_rule_snapshot,
			cas.class_attendance_session_rule_day_of_week_cache,
			cas.class_attendance_session_rule_start_time_cache,
			cas.class_attendance_session_rule_end_time_cache,
			cas.class_attendance_session_rule_week_parity_cache,
			cas.class_attendance_session_created_at,
			cas.class_attendance_session_updated_at,
			cas.class_attendance_session_deleted_at
		`).
		Order(orderExpr)

	if !isNearest {
		qSelect = qSelect.Order("cas.class_attendance_session_date DESC, cas.class_attendance_session_id DESC")
	}

	if err := qSelect.
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	pageIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		pageIDs = append(pageIDs, r.ID)
	}

	// Prefetch Participants (opsional)
	partMap := map[uuid.UUID][]SessionParticipantLite{}

	if wantParticipants && len(rows) > 0 {
		if err := ctrl.prefetchParticipants(c, schoolID, userID, isAdmin, isTeacher, pageIDs, partMap); err != nil {
			return err
		}
	}

	// Compose response
	buildBase := func(r row) sessiondto.ClassAttendanceSessionResponse {
		gen := ""
		if r.Gen != nil {
			gen = *r.Gen
		}

		return sessiondto.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:         r.ID,
			ClassAttendanceSessionSchoolId:   r.SchoolID,
			ClassAttendanceSessionScheduleId: r.ScheduleID,

			ClassAttendanceSessionSlug:         r.Slug,
			ClassAttendanceSessionTitle:        r.Title,
			ClassAttendanceSessionDisplayTitle: r.DisplayTitle,

			ClassAttendanceSessionGeneralInfo: gen,
			ClassAttendanceSessionNote:        r.Note,

			ClassAttendanceSessionPresentCount: r.PresentCount,
			ClassAttendanceSessionAbsentCount:  r.AbsentCount,
			ClassAttendanceSessionLateCount:    r.LateCount,
			ClassAttendanceSessionExcusedCount: r.ExcusedCount,
			ClassAttendanceSessionSickCount:    r.SickCount,
			ClassAttendanceSessionLeaveCount:   r.LeaveCount,

			ClassAttendanceSessionDate:     r.Date,
			ClassAttendanceSessionStartsAt: r.StartsAt,
			ClassAttendanceSessionEndsAt:   r.EndsAt,

			ClassAttendanceSessionStatus:           r.Status,
			ClassAttendanceSessionAttendanceStatus: r.AttendanceStatus,
			ClassAttendanceSessionLocked:           r.Locked,

			ClassAttendanceSessionIsOverride:      r.IsOverride,
			ClassAttendanceSessionIsCanceled:      r.IsCanceled,
			ClassAttendanceSessionOriginalStartAt: r.OriginalStartAt,
			ClassAttendanceSessionOriginalEndAt:   r.OriginalEndAt,
			ClassAttendanceSessionKind:            r.Kind,
			ClassAttendanceSessionOverrideReason:  r.OverrideReason,
			ClassAttendanceSessionOverrideEventId: r.OverrideEventID,

			ClassAttendanceSessionTeacherId:   r.TeacherID,
			ClassAttendanceSessionClassRoomId: r.RoomID,
			ClassAttendanceSessionCSSTId:      r.CSSTID,

			ClassAttendanceSessionTypeId:       r.TypeID,
			ClassAttendanceSessionTypeSnapshot: jsonToMap(r.TypeSnap),

			ClassAttendanceSessionCSSTSnapshot: jsonToMap(r.CSSTSnap),
			ClassAttendanceSessionRuleSnapshot: jsonToMap(r.RuleSnapshot),

			ClassAttendanceSessionCSSTIdCache:      r.CSSTIDCache,
			ClassAttendanceSessionSubjectIdCache:   r.SubjectIDCache,
			ClassAttendanceSessionSectionIdCache:   r.SectionIDCache,
			ClassAttendanceSessionTeacherIdCache:   r.TeacherIDCache,
			ClassAttendanceSessionRoomIdCache:      r.RoomIDCache,
			ClassAttendanceSessionSubjectCodeCache: r.SubjectCodeCache,
			ClassAttendanceSessionSubjectNameCache: r.SubjectNameCache,
			ClassAttendanceSessionSectionNameCache: r.SectionNameCache,
			ClassAttendanceSessionTeacherNameCache: r.TeacherNameCache,
			ClassAttendanceSessionRoomNameCache:    r.RoomNameCache,

			ClassAttendanceSessionRuleDayOfWeekCache:  r.RuleDayOfWeekCache,
			ClassAttendanceSessionRuleStartTimeCache:  r.RuleStartTimeCache,
			ClassAttendanceSessionRuleEndTimeCache:    r.RuleEndTimeCache,
			ClassAttendanceSessionRuleWeekParityCache: r.RuleWeekParityCache,

			ClassAttendanceSessionCreatedAt: r.CreatedAt,
			ClassAttendanceSessionUpdatedAt: r.UpdatedAt,
			ClassAttendanceSessionDeletedAt: r.DeletedAt,
		}
	}

	buildCompact := func(r row) sessiondto.ClassAttendanceSessionCompactResponse {
		gen := ""
		if r.Gen != nil {
			gen = *r.Gen
		}

		return sessiondto.ClassAttendanceSessionCompactResponse{
			ClassAttendanceSessionId:       r.ID,
			ClassAttendanceSessionSchoolId: r.SchoolID,

			ClassAttendanceSessionDate:     r.Date,
			ClassAttendanceSessionStartsAt: r.StartsAt,
			ClassAttendanceSessionEndsAt:   r.EndsAt,

			ClassAttendanceSessionTitle:        r.Title,
			ClassAttendanceSessionDisplayTitle: r.DisplayTitle,
			ClassAttendanceSessionGeneralInfo:  gen,

			ClassAttendanceSessionStatus:           r.Status,
			ClassAttendanceSessionAttendanceStatus: r.AttendanceStatus,

			ClassAttendanceSessionSubjectNameCache: r.SubjectNameCache,
			ClassAttendanceSessionSubjectCodeCache: r.SubjectCodeCache,
			ClassAttendanceSessionSectionNameCache: r.SectionNameCache,
			ClassAttendanceSessionRoomNameCache:    r.RoomNameCache,
			ClassAttendanceSessionTeacherNameCache: r.TeacherNameCache,
			ClassAttendanceSessionTeacherIdCache:   r.TeacherIDCache,
			ClassAttendanceSessionSectionIdCache:   r.SectionIDCache,
			ClassAttendanceSessionSubjectIdCache:   r.SubjectIDCache,
			ClassAttendanceSessionCSSTIdCache:      r.CSSTIDCache,

			ClassAttendanceSessionCSSTSnapshot: jsonToMap(r.CSSTSnap),

			ClassAttendanceSessionTypeId:       r.TypeID,
			ClassAttendanceSessionTypeSnapshot: jsonToMap(r.TypeSnap),
		}
	}

	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	if wantParticipants {
		type SessionWithParticipants struct {
			sessiondto.ClassAttendanceSessionResponse
			Participants []SessionParticipantLite `json:"participants,omitempty"`
		}
		out := make([]SessionWithParticipants, 0, len(rows))
		for _, r := range rows {
			out = append(out, SessionWithParticipants{
				ClassAttendanceSessionResponse: buildBase(r),
				Participants:                   partMap[r.ID],
			})
		}
		return helper.JsonList(c, "ok", out, pg)
	}

	if wantCompact {
		items := make([]sessiondto.ClassAttendanceSessionCompactResponse, 0, len(rows))
		for _, r := range rows {
			items = append(items, buildCompact(r))
		}
		return helper.JsonList(c, "ok", items, pg)
	}

	items := make([]sessiondto.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, buildBase(r))
	}
	return helper.JsonList(c, "ok", items, pg)
}

/*
	=========================================================
	  PREFETCH PARTICIPANTS

=========================================================
*/
func (ctrl *ClassAttendanceSessionController) prefetchParticipants(
	c *fiber.Ctx,
	schoolID uuid.UUID,
	userID uuid.UUID,
	isAdmin, isTeacher bool,
	pageIDs []uuid.UUID,
	partMap map[uuid.UUID][]SessionParticipantLite,
) error {
	state := strings.ToLower(strings.TrimSpace(c.Query("participant_state")))
	kind := strings.ToLower(strings.TrimSpace(c.Query("participant_kind")))

	typeIDPtr, err := parseUUIDPtr(c.Query("participant_type_id"), "participant_type_id")
	if err != nil {
		return err
	}

	var studentIDs []uuid.UUID
	if ids, err := parseUUIDList(c.Query("participant_student_id")); err != nil {
		return err
	} else if len(ids) > 0 {
		studentIDs = ids
	} else if ids, err := parseUUIDList(c.Query("school_student_id")); err != nil {
		return err
	} else if len(ids) > 0 {
		studentIDs = ids
	}

	var like *string
	if q := strings.TrimSpace(c.Query("participant_q")); q != "" {
		pat := "%" + q + "%"
		like = &pat
	}

	var isPassedPtr *bool
	if s := strings.TrimSpace(c.Query("participant_is_passed")); s != "" {
		if b, e := strconv.ParseBool(s); e == nil {
			isPassedPtr = &b
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "participant_is_passed tidak valid (true/false)")
		}
	}

	paQ := ctrl.DB.Table("class_attendance_session_participants AS p").
		Where("p.class_attendance_session_participant_deleted_at IS NULL").
		Where("p.class_attendance_session_participant_school_id = ?", schoolID).
		Where("p.class_attendance_session_participant_session_id IN ?", pageIDs)

	if state != "" {
		// enum attendance_state_enum → langsung bandingkan
		paQ = paQ.Where("p.class_attendance_session_participant_state = ?", state)
	}
	if kind != "" {
		// enum participant_kind_enum → langsung bandingkan
		paQ = paQ.Where("p.class_attendance_session_participant_kind = ?", kind)
	}
	if typeIDPtr != nil {
		paQ = paQ.Where("p.class_attendance_session_participant_type_id = ?", *typeIDPtr)
	}
	if len(studentIDs) > 0 {
		paQ = paQ.Where("p.class_attendance_session_participant_school_student_id IN ?", studentIDs)
	}
	if like != nil {
		paQ = paQ.Where(`
			(p.class_attendance_session_participant_desc ILIKE ?
			 OR p.class_attendance_session_participant_user_note ILIKE ?
			 OR p.class_attendance_session_participant_teacher_note ILIKE ?)`,
			*like, *like, *like)
	}
	if isPassedPtr != nil {
		paQ = paQ.Where("p.class_attendance_session_participant_is_passed = ?", *isPassedPtr)
	}

	// Scope kalau bukan admin/teacher → hanya peserta yg terkait user tsb
	if !isAdmin && !isTeacher {
		if userID == uuid.Nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentik")
		}
		paQ = paQ.Joins(`
			JOIN school_students ms
			  ON ms.school_student_id = p.class_attendance_session_participant_school_student_id
			 AND ms.school_student_deleted_at IS NULL
			 AND ms.school_student_user_id = ?
			 AND ms.school_student_school_id = ?
		`, userID, schoolID)
	}

	type paRow struct {
		ID                uuid.UUID  `gorm:"column:class_attendance_session_participant_id"`
		SessionID         uuid.UUID  `gorm:"column:class_attendance_session_participant_session_id"`
		SchoolStudentID   *uuid.UUID `gorm:"column:class_attendance_session_participant_school_student_id"`
		Kind              string     `gorm:"column:class_attendance_session_participant_kind"`
		State             string     `gorm:"column:class_attendance_session_participant_state"`
		TypeID            *uuid.UUID `gorm:"column:class_attendance_session_participant_type_id"`
		Desc              *string    `gorm:"column:class_attendance_session_participant_desc"`
		Score             *float64   `gorm:"column:class_attendance_session_participant_score"`
		IsPassed          *bool      `gorm:"column:class_attendance_session_participant_is_passed"`
		UserNote          *string    `gorm:"column:class_attendance_session_participant_user_note"`
		TeacherNote       *string    `gorm:"column:class_attendance_session_participant_teacher_note"`
		CheckinAt         *time.Time `gorm:"column:class_attendance_session_participant_checkin_at"`
		CheckoutAt        *time.Time `gorm:"column:class_attendance_session_participant_checkout_at"`
		LateSeconds       *int       `gorm:"column:class_attendance_session_participant_late_seconds"`
		MarkedAt          *time.Time `gorm:"column:class_attendance_session_participant_marked_at"`
		MarkedByTeacherID *uuid.UUID `gorm:"column:class_attendance_session_participant_marked_by_teacher_id"`
		Method            *string    `gorm:"column:class_attendance_session_participant_method"`
		TeacherRole       *string    `gorm:"column:class_attendance_session_participant_teacher_role"`
		Lat               *float64   `gorm:"column:class_attendance_session_participant_lat"`
		Lng               *float64   `gorm:"column:class_attendance_session_participant_lng"`
		DistanceM         *int       `gorm:"column:class_attendance_session_participant_distance_m"`
		CreatedAt         time.Time  `gorm:"column:class_attendance_session_participant_created_at"`
		UpdatedAt         time.Time  `gorm:"column:class_attendance_session_participant_updated_at"`
	}

	var paRows []paRow
	if err := paQ.
		Select(`
			p.class_attendance_session_participant_id,
			p.class_attendance_session_participant_session_id,
			p.class_attendance_session_participant_school_student_id,
			p.class_attendance_session_participant_kind,
			p.class_attendance_session_participant_state,
			p.class_attendance_session_participant_type_id,
			p.class_attendance_session_participant_desc,
			p.class_attendance_session_participant_score,
			p.class_attendance_session_participant_is_passed,
			p.class_attendance_session_participant_user_note,
			p.class_attendance_session_participant_teacher_note,
			p.class_attendance_session_participant_checkin_at,
			p.class_attendance_session_participant_checkout_at,
			p.class_attendance_session_participant_late_seconds,
			p.class_attendance_session_participant_marked_at,
			p.class_attendance_session_participant_marked_by_teacher_id,
			p.class_attendance_session_participant_method,
			p.class_attendance_session_participant_teacher_role,
			p.class_attendance_session_participant_lat,
			p.class_attendance_session_participant_lng,
			p.class_attendance_session_participant_distance_m,
			p.class_attendance_session_participant_created_at,
			p.class_attendance_session_participant_updated_at
		`).
		Order(`
			p.class_attendance_session_participant_session_id ASC,
			p.class_attendance_session_participant_created_at ASC,
			p.class_attendance_session_participant_id ASC
		`).
		Find(&paRows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data peserta absensi")
	}

	for _, r := range paRows {
		partMap[r.SessionID] = append(partMap[r.SessionID], SessionParticipantLite{
			ParticipantID:     r.ID,
			SessionID:         r.SessionID,
			SchoolStudentID:   r.SchoolStudentID,
			Kind:              r.Kind,
			State:             r.State,
			TypeID:            r.TypeID,
			Desc:              r.Desc,
			Score:             r.Score,
			IsPassed:          r.IsPassed,
			UserNote:          r.UserNote,
			TeacherNote:       r.TeacherNote,
			CheckinAt:         r.CheckinAt,
			CheckoutAt:        r.CheckoutAt,
			LateSeconds:       r.LateSeconds,
			CreatedAt:         r.CreatedAt,
			UpdatedAt:         r.UpdatedAt,
			MarkedAt:          r.MarkedAt,
			MarkedByTeacherID: r.MarkedByTeacherID,
			Method:            r.Method,
			TeacherRole:       r.TeacherRole,
			Lat:               r.Lat,
			Lng:               r.Lng,
			DistanceM:         r.DistanceM,
		})
	}

	// 🔹 Tambahan: kalau ada filter participant_kind (teacher/student)
	// tapi tidak ada satupun row peserta untuk session tsb,
	// tetap buat placeholder dengan state = "unknown"
	if kind != "" {
		for _, sid := range pageIDs {
			if parts, ok := partMap[sid]; !ok || len(parts) == 0 {
				partMap[sid] = []SessionParticipantLite{
					{
						ParticipantID: uuid.Nil,
						SessionID:     sid,
						Kind:          kind,
						State:         "unknown",
					},
				}
			}
		}
	}

	return nil
}

// SessionParticipantLite dipakai di response & prefetchParticipants
type SessionParticipantLite struct {
	ParticipantID     uuid.UUID  `json:"participant_id"`
	SessionID         uuid.UUID  `json:"participant_session_id"`
	SchoolStudentID   *uuid.UUID `json:"participant_school_student_id,omitempty"`
	Kind              string     `json:"participant_kind"`
	State             string     `json:"participant_state"`
	TypeID            *uuid.UUID `json:"participant_type_id,omitempty"`
	Desc              *string    `json:"participant_desc,omitempty"`
	Score             *float64   `json:"participant_score,omitempty"`
	IsPassed          *bool      `json:"participant_is_passed,omitempty"`
	UserNote          *string    `json:"participant_user_note,omitempty"`
	TeacherNote       *string    `json:"participant_teacher_note,omitempty"`
	CheckinAt         *time.Time `json:"participant_checkin_at,omitempty"`
	CheckoutAt        *time.Time `json:"participant_checkout_at,omitempty"`
	LateSeconds       *int       `json:"participant_late_seconds,omitempty"`
	CreatedAt         time.Time  `json:"participant_created_at"`
	UpdatedAt         time.Time  `json:"participant_updated_at"`
	MarkedAt          *time.Time `json:"participant_marked_at,omitempty"`
	MarkedByTeacherID *uuid.UUID `json:"participant_marked_by_teacher_id,omitempty"`
	Method            *string    `json:"participant_method,omitempty"`
	TeacherRole       *string    `json:"participant_teacher_role,omitempty"`
	Lat               *float64   `json:"participant_lat,omitempty"`
	Lng               *float64   `json:"participant_lng,omitempty"`
	DistanceM         *int       `json:"participant_distance_m,omitempty"`
}
