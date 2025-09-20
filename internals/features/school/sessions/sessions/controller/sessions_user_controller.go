// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	sessiondto "masjidku_backend/internals/features/school/sessions/sessions/dto"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
   Scopes & small helpers
========================= */

func scopeMasjid(masjidID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("class_attendance_sessions_masjid_id = ?", masjidID)
	}
}

func scopeDateBetween(df, dt *time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// inclusive [df, dt]
		if df != nil && dt != nil {
			return db.Where("class_attendance_sessions_date BETWEEN ? AND ?", *df, *dt)
		}
		if df != nil {
			return db.Where("class_attendance_sessions_date >= ?", *df)
		}
		if dt != nil {
			return db.Where("class_attendance_sessions_date <= ?", *dt)
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
		return db.Where("class_attendance_sessions_schedule_id = ?", *id)
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

// GET /admin/class-attendance-sessions
//
//	?id=&session_id=&cas_id=&teacher_id=&teacher_user_id=&section_id=&class_subject_id=&schedule_id=
//	&date_from=&date_to=&limit=&offset=&q=&sort_by=&sort=
//	&include=ua,ua_urls,urls|session_urls|casu
// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go

func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	// ===== Masjid context (pakai helpers) =====
	c.Locals("DB", ctrl.DB) // resolver slug→id butuh DB

	var masjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		// Jika context eksplisit (path/header/query/host) ⇒ wajib DKM/Admin di masjid tsb
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		masjidID = id
	} else {
		// Fallback ke token (teacher-aware)
		if id, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			masjidID = id
		} else {
			return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak ditemukan")
		}
	}

	// ===== Role (dipakai hanya untuk UA scope) =====
	userID, _ := helperAuth.GetUserIDFromToken(c)
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	isTeacher := teacherMasjidID != uuid.Nil && teacherMasjidID == masjidID

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

	// ===== Pagination & sorting =====
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
	p := helper.ParseWith(httpReq, "date", "desc", helper.AdminOpts)

	allowedSort := map[string]string{
		"date":  "cas.class_attendance_sessions_date",
		"title": "cas.class_attendance_sessions_title",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "date")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// ===== Filters dasar =====
	df, err := parseYmd(c.Query("date_from"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYmd(c.Query("date_to"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}

	var teacherIdPtr, teacherUserIDPtr, sectionIDPtr, classSubjectIDPtr, scheduleIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			teacherIdPtr = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("teacher_user_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			teacherUserIDPtr = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			sectionIDPtr = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("class_subject_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			classSubjectIDPtr = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "class_subject_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("schedule_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			scheduleIDPtr = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "schedule_id tidak valid")
		}
	}

	keyword := strings.TrimSpace(c.Query("q"))
	var like *string
	if keyword != "" {
		pat := "%" + keyword + "%"
		like = &pat
	}

	// ===== Filter by session ID(s) =====
	parseUUIDList := func(raw string) ([]uuid.UUID, error) {
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

	// ===== Base query (aliases) =====
	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Scopes(scopeMasjid(masjidID), scopeDateBetween(df, dt), scopeSchedule(scheduleIDPtr)).
		Where("cas.class_attendance_sessions_deleted_at IS NULL").
		// JOIN ke class_schedules untuk membawa CSST, lalu turunkan ke section/subject
		Joins(`
			LEFT JOIN class_schedules AS sch
			  ON sch.class_schedule_id = cas.class_attendance_sessions_schedule_id
			 AND sch.class_schedules_masjid_id = cas.class_attendance_sessions_masjid_id
			 AND sch.class_schedules_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_id = sch.class_schedules_csst_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`)
	if len(sessionIDs) > 0 {
		qBase = qBase.Where("cas.class_attendance_sessions_id IN ?", sessionIDs)
	}
	if sectionIDPtr != nil {
		qBase = qBase.Where("csst.class_section_subject_teachers_section_id = ?", *sectionIDPtr)
	}
	if classSubjectIDPtr != nil {
		qBase = qBase.Where("csst.class_section_subject_teachers_class_subjects_id = ?", *classSubjectIDPtr)
	}
	if teacherIdPtr != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_sessions_teacher_id = ?
			 OR csst.class_section_subject_teachers_teacher_id = ?)`,
			*teacherIdPtr, *teacherIdPtr,
		)
	}
	if teacherUserIDPtr != nil {
		qBase = qBase.
			Joins(`LEFT JOIN masjid_teachers mt_cas  ON mt_cas.masjid_teacher_id  = cas.class_attendance_sessions_teacher_id`).
			Joins(`LEFT JOIN masjid_teachers mt_csst ON mt_csst.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id`).
			Where(`
				(mt_cas.masjid_teacher_user_id = ? AND mt_cas.masjid_teacher_masjid_id = ?)
			 OR (mt_csst.masjid_teacher_user_id = ? AND mt_csst.masjid_teacher_masjid_id = ?)`,
				*teacherUserIDPtr, masjidID, *teacherUserIDPtr, masjidID,
			)
	}
	if like != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_sessions_title ILIKE ?
			 OR cas.class_attendance_sessions_general_info ILIKE ?)`, *like, *like)
	}

	// ===== Total (distinct id) =====
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_sessions_id").
		Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Data page (sessions) =====
	type row struct {
		ID         uuid.UUID  `gorm:"column:class_attendance_sessions_id"`
		MasjidID   uuid.UUID  `gorm:"column:class_attendance_sessions_masjid_id"`
		ScheduleID uuid.UUID  `gorm:"column:class_attendance_sessions_schedule_id"`
		RoomID     *uuid.UUID `gorm:"column:class_attendance_sessions_class_room_id"`
		Date       time.Time  `gorm:"column:class_attendance_sessions_date"`
		Title      *string    `gorm:"column:class_attendance_sessions_title"`
		General    string     `gorm:"column:class_attendance_sessions_general_info"`
		Note       *string    `gorm:"column:class_attendance_sessions_note"`
		TeacherID  *uuid.UUID `gorm:"column:class_attendance_sessions_teacher_id"`
		DeletedAt  *time.Time `gorm:"column:class_attendance_sessions_deleted_at"`
		SectionID  *uuid.UUID `gorm:"column:section_id"`
		SubjectID  *uuid.UUID `gorm:"column:subject_id"`
	}
	var rows []row
	if err := qBase.
		Select(`
			cas.class_attendance_sessions_id,
			cas.class_attendance_sessions_masjid_id,
			cas.class_attendance_sessions_schedule_id,
			cas.class_attendance_sessions_class_room_id,
			cas.class_attendance_sessions_date,
			cas.class_attendance_sessions_title,
			cas.class_attendance_sessions_general_info,
			cas.class_attendance_sessions_note,
			cas.class_attendance_sessions_teacher_id,
			cas.class_attendance_sessions_deleted_at,
			csst.class_section_subject_teachers_section_id        AS section_id,
			csst.class_section_subject_teachers_class_subjects_id AS subject_id
		`).
		Order(orderExpr).
		Order("cas.class_attendance_sessions_date DESC, cas.class_attendance_sessions_id DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ===== Kumpulkan session IDs halaman ini =====
	pageIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		pageIDs = append(pageIDs, r.ID)
	}

	// ===== (Opsional) Prefetch USER_ATTENDANCE untuk page sessions =====
	type UserAttendanceLite struct {
		UserAttendanceID uuid.UUID  `json:"user_attendance_id"`
		SessionID        uuid.UUID  `json:"user_attendance_session_id"`
		MasjidStudentID  uuid.UUID  `json:"user_attendance_masjid_student_id"`
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
		// --- Filters UA ---
		uaStatus := strings.ToLower(strings.TrimSpace(c.Query("ua_status")))

		var uaTypeIDPtr *uuid.UUID
		if s := strings.TrimSpace(c.Query("ua_type_id")); s != "" {
			if id, e := uuid.Parse(s); e == nil {
				uaTypeIDPtr = &id
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "ua_type_id tidak valid")
			}
		}

		// reuse local parseUUIDList
		var uaStudentIDs []uuid.UUID
		if ids, err := parseUUIDList(c.Query("ua_student_id")); err != nil {
			return err
		} else if len(ids) > 0 {
			uaStudentIDs = ids
		} else if ids, err := parseUUIDList(c.Query("masjid_student_id")); err != nil {
			return err
		} else if len(ids) > 0 {
			uaStudentIDs = ids
		}

		uaQuery := strings.TrimSpace(c.Query("ua_q"))
		var uaLike *string
		if uaQuery != "" {
			pat := "%" + uaQuery + "%"
			uaLike = &pat
		}

		var uaIsPassedPtr *bool
		if s := strings.TrimSpace(c.Query("ua_is_passed")); s != "" {
			if b, e := strconv.ParseBool(s); e == nil {
				uaIsPassedPtr = &b
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "ua_is_passed tidak valid (true/false)")
			}
		}

		// --- Build UA query ---
		uaQ := ctrl.DB.Table("user_attendance AS ua").
			Where("ua.user_attendance_deleted_at IS NULL").
			Where("ua.user_attendance_masjid_id = ?", masjidID).
			Where("ua.user_attendance_session_id IN ?", pageIDs)

		if uaStatus != "" {
			uaQ = uaQ.Where("LOWER(ua.user_attendance_status) = ?", uaStatus)
		}
		if uaTypeIDPtr != nil {
			uaQ = uaQ.Where("ua.user_attendance_type_id = ?", *uaTypeIDPtr)
		}
		if len(uaStudentIDs) > 0 {
			uaQ = uaQ.Where("ua.user_attendance_masjid_student_id IN ?", uaStudentIDs)
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

		// 🔐 Role-scope utk Student/Ortu: hanya data milik dirinya
		if !isAdmin && !isTeacher {
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			uaQ = uaQ.Joins(`
				JOIN masjid_students ms ON ms.masjid_student_id = ua.user_attendance_masjid_student_id
				 AND ms.masjid_student_deleted_at IS NULL
				 AND ms.masjid_student_user_id = ?
				 AND ms.masjid_student_masjid_id = ?
			`, userID, masjidID)
		}

		// --- Ambil UA rows ---
		type uaRow struct {
			ID          uuid.UUID  `gorm:"column:user_attendance_id"`
			SessionID   uuid.UUID  `gorm:"column:user_attendance_session_id"`
			StudentID   uuid.UUID  `gorm:"column:user_attendance_masjid_student_id"`
			Status      string     `gorm:"column:user_attendance_status"`
			TypeID      *uuid.UUID `gorm:"column:user_attendance_type_id"`
			Desc        *string    `gorm:"column:user_attendance_desc"`
			Score       *float64   `gorm:"column:user_attendance_score"`
			IsPassed    *bool      `gorm:"column:user_attendance_is_passed"`
			UserNote    *string    `gorm:"column:user_attendance_user_note"`
			TeacherNote *string    `gorm:"column:user_attendance_teacher_note"`
			CreatedAt   time.Time  `gorm:"column:user_attendance_created_at"`
			UpdatedAt   time.Time  `gorm:"column:user_attendance_updated_at"`
		}
		var uaRows []uaRow
		if err := uaQ.
			Select(`
				ua.user_attendance_id,
				ua.user_attendance_session_id,
				ua.user_attendance_masjid_student_id,
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
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil user_attendance")
		}

		for _, r := range uaRows {
			uaMap[r.SessionID] = append(uaMap[r.SessionID], UserAttendanceLite{
				UserAttendanceID: r.ID, SessionID: r.SessionID, MasjidStudentID: r.StudentID,
				Status: r.Status, TypeID: r.TypeID, Desc: r.Desc, Score: r.Score,
				IsPassed: r.IsPassed, UserNote: r.UserNote, TeacherNote: r.TeacherNote,
				CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
			})
		}
	}

	// ===== Compose output =====
	buildBase := func(r row) sessiondto.ClassAttendanceSessionResponse {
		return sessiondto.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:          r.ID,
			ClassAttendanceSessionMasjidId:    r.MasjidID,
			ClassAttendanceSessionScheduleId:  r.ScheduleID,
			ClassAttendanceSessionDate:        r.Date,
			ClassAttendanceSessionTitle:       r.Title,
			ClassAttendanceSessionGeneralInfo: r.General,
			ClassAttendanceSessionNote:        r.Note,
			ClassAttendanceSessionTeacherId:   r.TeacherID,
			ClassAttendanceSessionDeletedAt:   r.DeletedAt,
			ClassSectionId:                    r.SectionID,
			ClassSubjectId:                    r.SubjectID,
		}
	}

	meta := helper.BuildMeta(total, p)

	switch {
	case wantUA:
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
		return helper.JsonList(c, out, meta)

	default:
		items := make([]sessiondto.ClassAttendanceSessionResponse, 0, len(rows))
		for _, r := range rows {
			items = append(items, buildBase(r))
		}
		return helper.JsonList(c, items, meta)
	}
}

// LIST by TEACHER (SELF)
// GET /api/u/sessions/teacher/me?section_id=&schedule_id=&date_from=&date_to=&limit=&offset=&q=
func (ctrl *ClassAttendanceSessionController) ListMyTeachingSessions(c *fiber.Ctx) error {
	// Hanya guru (atau admin/DKM) yang boleh akses endpoint ini
	if !helperAuth.IsTeacher(c) && !helperAuth.IsDKM(c) && !helperAuth.IsOwner(c) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya guru (atau admin) yang diizinkan")
	}

	// 🎯 Resolusi context masjid
	mc, er := helperAuth.ResolveMasjidContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	var masjidID uuid.UUID
	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		masjidID = id

	default: // Teacher ⇒ wajib member pada masjid context
		if mc.ID != uuid.Nil {
			masjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			masjidID = id
		} else if id, er := helperAuth.GetActiveMasjidID(c); er == nil && id != uuid.Nil {
			masjidID = id
		}
		if masjidID == uuid.Nil || !helperAuth.UserHasMasjid(c, masjidID) {
			return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak valid untuk Teacher")
		}
	}

	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
	}

	// ==== lanjutkan kode asli (pagination, query, mapping) ====
	// Pagination & sorting
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
	p := helper.ParseWith(httpReq, "date", "desc", helper.AdminOpts)

	allowedSort := map[string]string{
		"date":  "cas.class_attendance_sessions_date",
		"title": "cas.class_attendance_sessions_title",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "date")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// Rentang tanggal
	df, err := parseYMDLocal(c.Query("date_from"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYMDLocal(c.Query("date_to"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}
	if df != nil && dt != nil && dt.Before(*df) {
		return fiber.NewError(fiber.StatusBadRequest, "date_to harus >= date_from")
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
			LEFT JOIN masjid_teachers AS mt
			  ON mt.masjid_teacher_id = cas.class_attendance_sessions_teacher_id
			 AND mt.masjid_teacher_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_schedules AS cs
			  ON cs.class_schedule_id = cas.class_attendance_sessions_schedule_id
			 AND cs.class_schedules_deleted_at IS NULL
			 AND cs.class_schedules_is_active
		`).
		Joins(`
			LEFT JOIN masjid_teachers AS mt2
			  ON mt2.masjid_teacher_id = cs.class_schedules_teacher_id
			 AND mt2.masjid_teacher_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_id = cs.class_schedules_csst_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN masjid_teachers AS mt3
			  ON mt3.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id
			 AND mt3.masjid_teacher_deleted_at IS NULL
		`).
		Where(`
			cas.class_attendance_sessions_masjid_id = ?
			AND cas.class_attendance_sessions_deleted_at IS NULL
			AND (
			     mt.masjid_teacher_user_id = ?
			  OR mt2.masjid_teacher_user_id = ?
			  OR mt3.masjid_teacher_user_id = ?
			)
		`, masjidID, userID, userID, userID)

	// Filter tanggal opsional
	if lo != nil && hi != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ? AND cas.class_attendance_sessions_date < ?", *lo, *hi)
	} else if lo != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ?", *lo)
	} else if hi != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date < ?", *hi)
	}

	// Opsional: section_id
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("COALESCE(cs.class_schedules_section_id, csst.class_section_subject_teachers_section_id) = ?", id)
	}

	// Opsional: schedule_id
	if s := strings.TrimSpace(c.Query("schedule_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "schedule_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_schedule_id = ?", id)
	}

	// Keyword
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		pat := "%" + q + "%"
		qBase = qBase.Where(`(cas.class_attendance_sessions_title ILIKE ? OR cas.class_attendance_sessions_general_info ILIKE ?)`, pat, pat)
	}

	// Total distinct
	var total int64
	if err := qBase.Session(&gorm.Session{}).Distinct("cas.class_attendance_sessions_id").Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// Data
	type row struct {
		ID         uuid.UUID  `gorm:"column:id"`
		MasjidID   uuid.UUID  `gorm:"column:masjid_id"`
		Date       time.Time  `gorm:"column:date"`
		Title      *string    `gorm:"column:title"`
		General    string     `gorm:"column:general"`
		Note       *string    `gorm:"column:note"`
		TeacherID  *uuid.UUID `gorm:"column:teacher_id"`
		RoomID     *uuid.UUID `gorm:"column:room_id"`
		ScheduleID uuid.UUID  `gorm:"column:schedule_id"`
		SectionID  *uuid.UUID `gorm:"column:section_id"`
		SubjectID  *uuid.UUID `gorm:"column:subject_id"`
		DeletedAt  *time.Time `gorm:"column:deleted_at"`
	}
	var rows []row
	if err := qBase.
		Select(`
			cas.class_attendance_sessions_id         AS id,
			cas.class_attendance_sessions_masjid_id  AS masjid_id,
			cas.class_attendance_sessions_date       AS date,
			cas.class_attendance_sessions_title      AS title,
			cas.class_attendance_sessions_general_info AS general,
			cas.class_attendance_sessions_note       AS note,
			cas.class_attendance_sessions_teacher_id AS teacher_id,
			cas.class_attendance_sessions_class_room_id AS room_id,
			cas.class_attendance_sessions_schedule_id   AS schedule_id,
			cas.class_attendance_sessions_deleted_at AS deleted_at,
			COALESCE(cs.class_schedules_section_id, csst.class_section_subject_teachers_section_id) AS section_id,
			COALESCE(cs.class_schedules_class_subject_id, csst.class_section_subject_teachers_class_subjects_id) AS subject_id
		`).
		Order(orderExpr).
		Order("cas.class_attendance_sessions_date DESC, cas.class_attendance_sessions_id DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]sessiondto.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, sessiondto.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:          r.ID,
			ClassAttendanceSessionMasjidId:    r.MasjidID,
			ClassAttendanceSessionScheduleId:  r.ScheduleID,
			ClassAttendanceSessionDate:        r.Date,
			ClassAttendanceSessionTitle:       r.Title,
			ClassAttendanceSessionGeneralInfo: r.General,
			ClassAttendanceSessionNote:        r.Note,
			ClassAttendanceSessionTeacherId:   r.TeacherID,
			ClassAttendanceSessionDeletedAt:   r.DeletedAt,
			ClassSectionId:                    r.SectionID,
			ClassSubjectId:                    r.SubjectID,
		})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, resp, meta)
}
