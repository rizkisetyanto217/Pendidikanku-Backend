package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	sessiondto "schoolku_backend/internals/features/school/classes/class_attendance_sessions/dto"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   Scopes & small helpers
========================= */

func scopeSchool(schoolID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("class_attendance_session_school_id = ?", schoolID)
	}
}

func scopeDateBetween(df, dt *time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// inclusive [df, dt]
		if df != nil && dt != nil {
			return db.Where("class_attendance_session_date BETWEEN ? AND ?", *df, *dt)
		}
		if df != nil {
			return db.Where("class_attendance_session_date >= ?", *df)
		}
		if dt != nil {
			return db.Where("class_attendance_session_date <= ?", *dt)
		}
		return db
	}
}

// filter by SCHEDULE (kolom di CAS)
func scopeSchedule(id *uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if id == nil {
			return db
		}
		return db.Where("class_attendance_session_schedule_id = ?", *id)
	}
}

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

/* =================================================================
   LIST /admin/class-attendance-sessions — schedule opsional & null
================================================================= */

// GET /admin/class-attendance-sessions
//
//	?id=&session_id=&cas_id=&teacher_id=&teacher_user_id=&schedule_id=|null
//	&date_from=&date_to=&limit=&offset=&q=&sort_by=&sort=
//	&include=ua,ua_urls,urls|session_urls|casu
//
// file: internals/features/school/classes/class_attendance_sessions/controller/list_controller.go
func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	// ===== School context =====
	c.Locals("DB", ctrl.DB)
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			// tetap balikin via helper agar seragam
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, http.StatusForbidden, er.Error())
		}
		schoolID = id
	} else if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		return helper.JsonError(c, http.StatusForbidden, "Scope school tidak ditemukan")
	}

	// ===== Roles (buat UA scope di bawah) =====
	userID, _ := helperAuth.GetUserIDFromToken(c)
	adminSchoolID, _ := helperAuth.GetSchoolIDFromToken(c)
	teacherSchoolID, _ := helperAuth.GetSchoolIDFromTokenPreferTeacher(c)

	isAdmin := (adminSchoolID != uuid.Nil && adminSchoolID == schoolID) ||
		helperAuth.HasRoleInSchool(c, schoolID, "admin") ||
		helperAuth.HasRoleInSchool(c, schoolID, "dkm") ||
		helperAuth.IsDKMInSchool(c, schoolID)

	isTeacher := (teacherSchoolID != uuid.Nil && teacherSchoolID == schoolID) ||
		helperAuth.HasRoleInSchool(c, schoolID, "teacher") ||
		helperAuth.IsTeacherInSchool(c, schoolID)

	// ===== Includes =====
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeAll := includeStr == "all"
	includeSet := map[string]bool{}
	for _, part := range strings.Split(includeStr, ",") {
		if p := strings.TrimSpace(part); p != "" {
			includeSet[p] = true
		}
	}
	wantUA := includeAll || includeSet["user_attendance"] || includeSet["user_attendances"] || includeSet["attendance"] || includeSet["ua"]

	// ===== Pagination (jsonresponse) =====
	// default per_page = 20, max = 200
	p := helper.ResolvePaging(c, 20, 200)

	// ===== Sorting whitelist (gantikan SafeOrderClause) =====
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
	default:
		// sort_by tak dikenal → tetap default
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// ===== Filters dasar =====
	df, err := parseYmd(c.Query("date_from"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYmd(c.Query("date_to"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}

	teacherIdPtr, err := parseUUIDPtr(c.Query("teacher_id"), "teacher_id")
	if err != nil {
		return err
	}
	teacherUserIDPtr, err := parseUUIDPtr(c.Query("teacher_user_id"), "teacher_user_id")
	if err != nil {
		return err
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

	// ===== Base query =====
	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Where("cas.class_attendance_session_school_id = ?", schoolID).
		Where("cas.class_attendance_session_deleted_at IS NULL")

	if df != nil && dt != nil {
		qBase = qBase.Where("cas.class_attendance_session_date BETWEEN ? AND ?", *df, *dt)
	} else if df != nil {
		qBase = qBase.Where("cas.class_attendance_session_date >= ?", *df)
	} else if dt != nil {
		qBase = qBase.Where("cas.class_attendance_session_date <= ?", *dt)
	}

	if scheduleIDPtr != nil {
		qBase = qBase.Where("cas.class_attendance_session_schedule_id = ?", *scheduleIDPtr)
	} else if wantScheduleNull {
		qBase = qBase.Where(`cas.class_attendance_session_schedule_id IS NULL OR cas.class_attendance_session_schedule_id = '00000000-0000-0000-0000-000000000000'`)
	}

	if len(sessionIDs) > 0 {
		qBase = qBase.Where("cas.class_attendance_session_id IN ?", sessionIDs)
	}
	if teacherIdPtr != nil {
		qBase = qBase.Where("cas.class_attendance_session_teacher_id = ?", *teacherIdPtr)
	}
	if teacherUserIDPtr != nil {
		qBase = qBase.
			Joins(`LEFT JOIN school_teachers mt_cas
                     ON mt_cas.school_teacher_id = cas.class_attendance_session_teacher_id
                    AND mt_cas.school_teacher_deleted_at IS NULL
                    AND mt_cas.school_teacher_school_id = cas.class_attendance_session_school_id`).
			Where(`mt_cas.school_teacher_user_id = ?`, *teacherUserIDPtr)
	}
	if like != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_session_title ILIKE ?
			 OR cas.class_attendance_session_general_info ILIKE ?
             OR cas.class_attendance_session_display_title ILIKE ?)`, *like, *like, *like)
	}

	// ===== Total =====
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_session_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Page data =====
	type row struct {
		ID, SchoolID                                   uuid.UUID
		ScheduleID, RoomID, TeacherID                  *uuid.UUID
		Date                                           time.Time
		Title, Disp, Gen, Note                         *string // Gen di DB string → baca ke *string aman
		DeletedAt                                      *time.Time
		CSSTSnap, TeacherSnap, RoomSnap                datatypes.JSON
		CSSTIDSnap, SubjectIDSnap, SectionIDSnap       *uuid.UUID
		TeacherIDSnap, RoomIDSnap                      *uuid.UUID
		SubjectCodeSnap, SubjectNameSnap               *string
		SectionNameSnap, TeacherNameSnap, RoomNameSnap *string
	}
	var rows []row
	if err := qBase.
		Select(`
			cas.class_attendance_session_id,
			cas.class_attendance_session_school_id,
			cas.class_attendance_session_schedule_id,
			cas.class_attendance_session_class_room_id,

			cas.class_attendance_session_date,
			cas.class_attendance_session_title,
			cas.class_attendance_session_display_title,
			cas.class_attendance_session_general_info,
			cas.class_attendance_session_note,
			cas.class_attendance_session_teacher_id,
			cas.class_attendance_session_deleted_at,

			cas.class_attendance_session_csst_snapshot,
			cas.class_attendance_session_teacher_snapshot,
			cas.class_attendance_session_room_snapshot,

			cas.class_attendance_session_csst_id_snap,
			cas.class_attendance_session_subject_id_snap,
			cas.class_attendance_session_section_id_snap,
			cas.class_attendance_session_teacher_id_snap,
			cas.class_attendance_session_room_id_snap,
			cas.class_attendance_session_subject_code_snap,
			cas.class_attendance_session_subject_name_snap,
			cas.class_attendance_session_section_name_snap,
			cas.class_attendance_session_teacher_name_snap,
			cas.class_attendance_session_room_name_snap
		`).
		Order(orderExpr).
		// tie-breaker stabil
		Order("cas.class_attendance_session_date DESC, cas.class_attendance_session_id DESC").
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	pageIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		pageIDs = append(pageIDs, r.ID)
	}

	// ===== Prefetch UA (opsional) =====
	type UserAttendanceLite struct {
		UserAttendanceID uuid.UUID  `json:"user_attendance_id"`
		SessionID        uuid.UUID  `json:"user_attendance_session_id"`
		SchoolStudentID  uuid.UUID  `json:"user_attendance_school_student_id"`
		Status           string     `json:"user_attendance_status"`
		TypeID           *uuid.UUID `json:"user_attendance_type_id,omitempty"`
		Desc             *string    `json:"user_attendance_desc,omitempty"`
		Score            *float64   `json:"user_attendance_score,omitempty"`
		IsPassed         *bool      `json:"user_attendance_is_passed,omitempty"`
		UserNote         *string    `json:"user_attendance_user_note,omitempty"`
		TeacherNote      *string    `json:"user_attendance_teacher_note,omitempty"`
		CreatedAt        time.Time  `json:"user_attendance_created_at"`
		UpdatedAt        time.Time  `json:"user_attendance_updated_at"`
	}
	uaMap := map[uuid.UUID][]UserAttendanceLite{}

	if wantUA && len(rows) > 0 {
		uaStatus := strings.ToLower(strings.TrimSpace(c.Query("ua_status")))
		uaTypeIDPtr, err := parseUUIDPtr(c.Query("ua_type_id"), "ua_type_id")
		if err != nil {
			return err
		}
		var uaStudentIDs []uuid.UUID
		if ids, err := parseUUIDList(c.Query("ua_student_id")); err != nil {
			return err
		} else if len(ids) > 0 {
			uaStudentIDs = ids
		} else if ids, err := parseUUIDList(c.Query("school_student_id")); err != nil {
			return err
		} else if len(ids) > 0 {
			uaStudentIDs = ids
		}
		var uaLike *string
		if q := strings.TrimSpace(c.Query("ua_q")); q != "" {
			pat := "%" + q + "%"
			uaLike = &pat
		}
		var uaIsPassedPtr *bool
		if s := strings.TrimSpace(c.Query("ua_is_passed")); s != "" {
			if b, e := strconv.ParseBool(s); e == nil {
				uaIsPassedPtr = &b
			} else {
				return helper.JsonError(c, fiber.StatusBadRequest, "ua_is_passed tidak valid (true/false)")
			}
		}

		uaQ := ctrl.DB.Table("user_attendance AS ua").
			Where("ua.user_attendance_deleted_at IS NULL").
			Where("ua.user_attendance_school_id = ?", schoolID).
			Where("ua.user_attendance_session_id IN ?", pageIDs)

		if uaStatus != "" {
			uaQ = uaQ.Where("LOWER(ua.user_attendance_status) = ?", uaStatus)
		}
		if uaTypeIDPtr != nil {
			uaQ = uaQ.Where("ua.user_attendance_type_id = ?", *uaTypeIDPtr)
		}
		if len(uaStudentIDs) > 0 {
			uaQ = uaQ.Where("ua.user_attendance_school_student_id IN ?", uaStudentIDs)
		}
		if uaLike != nil {
			uaQ = uaQ.Where(`
				(ua.user_attendance_desc ILIKE ?
				 OR ua.user_attendance_user_note ILIKE ?
				 OR ua.user_attendance_teacher_note ILIKE ?)`, *uaLike, *uaLike, *uaLike)
		}
		if uaIsPassedPtr != nil {
			uaQ = uaQ.Where("ua.user_attendance_is_passed = ?", *uaIsPassedPtr)
		}

		// Role-scope Student/Parent
		if !isAdmin && !isTeacher {
			if userID == uuid.Nil {
				return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentik")
			}
			uaQ = uaQ.Joins(`
				JOIN school_students ms ON ms.school_student_id = ua.user_attendance_school_student_id
				 AND ms.school_student_deleted_at IS NULL
				 AND ms.school_student_user_id = ?
				 AND ms.school_student_school_id = ?
			`, userID, schoolID)
		}

		type uaRow struct {
			ID, SessionID, StudentID uuid.UUID
			Status                   string
			TypeID                   *uuid.UUID
			Desc, UserNote           *string
			TeacherNote              *string
			Score                    *float64
			IsPassed                 *bool
			CreatedAt, UpdatedAt     time.Time
		}
		var uaRows []uaRow
		if err := uaQ.
			Select(`
				ua.user_attendance_id,
				ua.user_attendance_session_id,
				ua.user_attendance_school_student_id,
				ua.user_attendance_status,
				ua.user_attendance_type_id,
				ua.user_attendance_desc,
				ua.user_attendance_score,
				ua.user_attendance_is_passed,
				ua.user_attendance_user_note,
				ua.user_attendance_teacher_note,
				ua.user_attendance_created_at,
				ua.user_attendance_updated_at
			`).
			Order("ua.user_attendance_session_id ASC, ua.user_attendance_created_at ASC, ua.user_attendance_id ASC").
			Find(&uaRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil user_attendance")
		}
		for _, r := range uaRows {
			uaMap[r.SessionID] = append(uaMap[r.SessionID], UserAttendanceLite{
				UserAttendanceID: r.ID, SessionID: r.SessionID, SchoolStudentID: r.StudentID,
				Status: r.Status, TypeID: r.TypeID, Desc: r.Desc, Score: r.Score,
				IsPassed: r.IsPassed, UserNote: r.UserNote, TeacherNote: r.TeacherNote,
				CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
			})
		}
	}

	// ===== Compose response =====
	buildBase := func(r row) sessiondto.ClassAttendanceSessionResponse {
		// r.Gen bertipe *string di struct lokal (aman untuk nil)
		gen := ""
		if r.Gen != nil {
			gen = *r.Gen
		}
		return sessiondto.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:              r.ID,
			ClassAttendanceSessionSchoolId:        r.SchoolID,
			ClassAttendanceSessionScheduleId:      r.ScheduleID,
			ClassAttendanceSessionDate:            r.Date,
			ClassAttendanceSessionTitle:           r.Title,
			ClassAttendanceSessionDisplayTitle:    r.Disp,
			ClassAttendanceSessionGeneralInfo:     gen,
			ClassAttendanceSessionNote:            r.Note,
			ClassAttendanceSessionTeacherId:       r.TeacherID,
			ClassAttendanceSessionClassRoomId:     r.RoomID,
			ClassAttendanceSessionCSSTSnapshot:    jsonToMap(r.CSSTSnap),
			ClassAttendanceSessionTeacherSnapshot: jsonToMap(r.TeacherSnap),
			ClassAttendanceSessionRoomSnapshot:    jsonToMap(r.RoomSnap),
			ClassAttendanceSessionCSSTIdSnap:      r.CSSTIDSnap,
			ClassAttendanceSessionSubjectIdSnap:   r.SubjectIDSnap,
			ClassAttendanceSessionSectionIdSnap:   r.SectionIDSnap,
			ClassAttendanceSessionTeacherIdSnap:   r.TeacherIDSnap,
			ClassAttendanceSessionRoomIdSnap:      r.RoomIDSnap,
			ClassAttendanceSessionSubjectCodeSnap: r.SubjectCodeSnap,
			ClassAttendanceSessionSubjectNameSnap: r.SubjectNameSnap,
			ClassAttendanceSessionSectionNameSnap: r.SectionNameSnap,
			ClassAttendanceSessionTeacherNameSnap: r.TeacherNameSnap,
			ClassAttendanceSessionRoomNameSnap:    r.RoomNameSnap,
			ClassAttendanceSessionDeletedAt:       r.DeletedAt,
		}
	}

	// ===== Meta (jsonresponse) =====
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	if wantUA {
		type SessionWithUA struct {
			sessiondto.ClassAttendanceSessionResponse
			UserAttendance []UserAttendanceLite `json:"user_attendance,omitempty"`
		}
		out := make([]SessionWithUA, 0, len(rows))
		for _, r := range rows {
			out = append(out, SessionWithUA{
				ClassAttendanceSessionResponse: buildBase(r),
				UserAttendance:                 uaMap[r.ID],
			})
		}
		return helper.JsonList(c, "ok", out, pg)
	}

	items := make([]sessiondto.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, buildBase(r))
	}
	return helper.JsonList(c, "ok", items, pg)
}

/* ==========================================================
   LIST by TEACHER (SELF) — schedule opsional & pointer-safe
========================================================== */

// GET /api/u/sessions/teacher/me?section_id=&schedule_id=&date_from=&date_to=&limit=&offset=&q=
func (ctrl *ClassAttendanceSessionController) ListMyTeachingSessions(c *fiber.Ctx) error {
	if !helperAuth.IsTeacher(c) && !helperAuth.IsDKM(c) && !helperAuth.IsOwner(c) {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Hanya guru (atau admin) yang diizinkan")
	}

	// ===== School context (seragam helper) =====
	mc, er := helperAuth.ResolveSchoolContext(c)
	if er != nil {
		if fe, ok := er.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
	}
	var schoolID uuid.UUID
	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
		schoolID = id
	default:
		if mc.ID != uuid.Nil {
			schoolID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, e2 := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if e2 != nil {
				return helper.JsonError(c, http.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		} else if id, e3 := helperAuth.GetActiveSchoolID(c); e3 == nil && id != uuid.Nil {
			schoolID = id
		}
		if schoolID == uuid.Nil || !helperAuth.UserHasSchool(c, schoolID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak valid untuk Teacher")
		}
	}

	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || userID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentik")
	}

	// ===== Pagination (jsonresponse) =====
	p := helper.ResolvePaging(c, 20, 200)

	// ===== Sorting whitelist =====
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	order := strings.ToLower(strings.TrimSpace(c.Query("order")))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	col := "cas.class_attendance_session_date"
	switch sortBy {
	case "title":
		col = "cas.class_attendance_session_title"
	case "date", "":
		col = "cas.class_attendance_session_date"
	default:
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// Rentang tanggal
	df, err := parseYmd(c.Query("date_from"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYmd(c.Query("date_to"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}
	if df != nil && dt != nil && dt.Before(*df) {
		return helper.JsonError(c, fiber.StatusBadRequest, "date_to harus >= date_from")
	}
	var lo, hi *time.Time
	if df != nil {
		lo = df
	}
	if dt != nil {
		h := dt.Add(24 * time.Hour)
		hi = &h
	}

	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Joins(`
			LEFT JOIN school_teachers AS mt_override
			  ON mt_override.school_teacher_id = cas.class_attendance_session_teacher_id
			 AND mt_override.school_teacher_deleted_at IS NULL
			 AND mt_override.school_teacher_school_id = cas.class_attendance_session_school_id
		`).
		Joins(`
			LEFT JOIN school_teachers AS mt_snap
			  ON mt_snap.school_teacher_id = cas.class_attendance_session_teacher_id_snap
			 AND mt_snap.school_teacher_deleted_at IS NULL
			 AND mt_snap.school_teacher_school_id = cas.class_attendance_session_school_id
		`).
		Where(`
			cas.class_attendance_session_school_id = ?
			AND cas.class_attendance_session_deleted_at IS NULL
			AND (
			     mt_override.school_teacher_user_id = ?
			  OR mt_snap.school_teacher_user_id = ?
			)
		`, schoolID, userID, userID)

	if lo != nil && hi != nil {
		qBase = qBase.Where("cas.class_attendance_session_date >= ? AND cas.class_attendance_session_date < ?", *lo, *hi)
	} else if lo != nil {
		qBase = qBase.Where("cas.class_attendance_session_date >= ?", *lo)
	} else if hi != nil {
		qBase = qBase.Where("cas.class_attendance_session_date < ?", *hi)
	}

	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_session_section_id_snap = ?", id)
	}
	if s := strings.TrimSpace(c.Query("schedule_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "schedule_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_session_schedule_id = ?", id)
	}

	if q := strings.TrimSpace(c.Query("q")); q != "" {
		pat := "%" + q + "%"
		qBase = qBase.Where(`(cas.class_attendance_session_title ILIKE ? OR cas.class_attendance_session_general_info ILIKE ? OR cas.class_attendance_session_display_title ILIKE ?)`, pat, pat, pat)
	}

	// Total
	var total int64
	if err := qBase.Session(&gorm.Session{}).Distinct("cas.class_attendance_session_id").Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// Data
	type row struct {
		ID, SchoolID                  uuid.UUID
		Date                          time.Time
		Title, Display, General, Note *string
		TeacherID, RoomID, ScheduleID *uuid.UUID
		SectionIDSnap, SubjectIDSnap  *uuid.UUID
		DeletedAt                     *time.Time
	}
	var rows []row
	if err := qBase.
		Select(`
			cas.class_attendance_session_id         AS id,
			cas.class_attendance_session_school_id  AS school_id,
			cas.class_attendance_session_date       AS date,
			cas.class_attendance_session_title      AS title,
			cas.class_attendance_session_display_title AS display,
			cas.class_attendance_session_general_info AS general,
			cas.class_attendance_session_note       AS note,
			cas.class_attendance_session_teacher_id AS teacher_id,
			cas.class_attendance_session_class_room_id AS room_id,
			cas.class_attendance_session_schedule_id   AS schedule_id,
			cas.class_attendance_session_deleted_at AS deleted_at,
			cas.class_attendance_session_section_id_snap AS section_id_snap,
			cas.class_attendance_session_subject_id_snap AS subject_id_snap
		`).
		Order(orderExpr).
		Order("cas.class_attendance_session_date DESC, cas.class_attendance_session_id DESC").
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]sessiondto.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		gen := ""
		if r.General != nil {
			gen = *r.General
		}
		resp = append(resp, sessiondto.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:            r.ID,
			ClassAttendanceSessionSchoolId:      r.SchoolID,
			ClassAttendanceSessionScheduleId:    r.ScheduleID,
			ClassAttendanceSessionDate:          r.Date,
			ClassAttendanceSessionTitle:         r.Title,
			ClassAttendanceSessionDisplayTitle:  r.Display,
			ClassAttendanceSessionGeneralInfo:   gen,
			ClassAttendanceSessionNote:          r.Note,
			ClassAttendanceSessionTeacherId:     r.TeacherID,
			ClassAttendanceSessionClassRoomId:   r.RoomID,
			ClassAttendanceSessionDeletedAt:     r.DeletedAt,
			ClassAttendanceSessionSectionIdSnap: r.SectionIDSnap,
			ClassAttendanceSessionSubjectIdSnap: r.SubjectIDSnap,
		})
	}

	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)
	return helper.JsonList(c, "ok", resp, pg)
}
