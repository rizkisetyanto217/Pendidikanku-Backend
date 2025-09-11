// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	attendanceDTO "masjidku_backend/internals/features/school/sessions/sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/sessions/sessions/model"
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

// filter by CSST (kolom di CAS)
func scopeCSST(id *uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if id == nil {
			return db
		}
		return db.Where("class_attendance_sessions_csst_id = ?", *id)
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

// isTeacherOfSession: user (guru) dianggap pengajar sesi bila:
// - CAS.teacher_id milik user tsb (masjid_teachers.user_id = userID, masjid sama), ATAU
// - CSST.teacher_id milik user tsb (CSST is_active & not deleted, masjid sama).
// cek apakah user (guru) adalah pengajar sesi (CAS.teacher_id atau CSST.teacher_id)
func (ctrl *ClassAttendanceSessionController) isTeacherOfSession(session attendanceModel.ClassAttendanceSessionModel, userID, masjidID uuid.UUID) (bool, error) {
	// via CAS.teacher_id
	if session.ClassAttendanceSessionTeacherId != nil {
		var cnt int64
		if err := ctrl.DB.Table("masjid_teachers AS mt").
			Where("mt.masjid_teacher_id = ?", *session.ClassAttendanceSessionTeacherId).
			Where("mt.masjid_teacher_user_id = ?", userID).
			Where("mt.masjid_teacher_masjid_id = ?", masjidID).
			Count(&cnt).Error; err != nil {
			return false, err
		}
		if cnt > 0 {
			return true, nil
		}
	}
	// via CSST.teacher_id
	if session.ClassAttendanceSessionCSSTId != uuid.Nil {
		var cnt int64
		if err := ctrl.DB.Table("class_section_subject_teachers AS csst").
			Joins("JOIN masjid_teachers mt ON mt.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id").
			Where(`
				csst.class_section_subject_teachers_id = ?
				AND csst.class_section_subject_teachers_deleted_at IS NULL
				AND csst.class_section_subject_teachers_is_active = TRUE
				AND csst.class_section_subject_teachers_masjid_id = ?
				AND mt.masjid_teacher_user_id = ?
				AND mt.masjid_teacher_masjid_id = ?
			`, session.ClassAttendanceSessionCSSTId, masjidID, userID, masjidID).
			Count(&cnt).Error; err != nil {
			return false, err
		}
		return cnt > 0, nil
	}
	return false, nil
}


/* ==========================================================================================
   GET /admin/class-attendance-sessions
     ?id=&session_id=&cas_id=&teacher_id=&teacher_user_id=&section_id=&class_subject_id=&csst_id=
     &date_from=&date_to=&limit=&offset=&q=&sort_by=&sort=
   - id / session_id / cas_id → filter ke CAS.id (boleh comma-separated)
   - teacher_id      → masjid_teacher_id (CAS.teacher_id atau CSST.teacher_id)
   - teacher_user_id → users.id (JOIN ke masjid_teachers)
   - section_id      → filter via CSST.section_id
   - class_subject_id→ filter via CSST.class_subjects_id
   - csst_id         → langsung ke CAS.csst_id
========================================================================================== */
/* ==========================================================================================
   GET /admin/class-attendance-sessions
     ?id=&session_id=&cas_id=&teacher_id=&teacher_user_id=&section_id=&class_subject_id=&csst_id=
     &date_from=&date_to=&limit=&offset=&q=&sort_by=&sort=
     &include=... (user_attendance|user_attendances|attendance|ua)

   -- Filter untuk user_attendance (aktif bila include user_attendance)
     &ua_status=&ua_type_id=&ua_student_id=&masjid_student_id=&ua_q=&ua_is_passed=

   - id / session_id / cas_id → filter ke CAS.id (boleh comma-separated)
   - teacher_id      → masjid_teacher_id (CAS.teacher_id atau CSST.teacher_id)
   - teacher_user_id → users.id (JOIN ke masjid_teachers)
   - section_id      → filter via CSST.section_id
   - class_subject_id→ filter via CSST.class_subjects_id
   - csst_id         → langsung ke CAS.csst_id
========================================================================================== */

func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	// ===== Tenant (admin/teacher) =====
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// ===== Role =====
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
	wantUA := includeAll ||
		includeSet["user_attendance"] ||
		includeSet["user_attendances"] ||
		includeSet["attendance"] ||
		includeSet["ua"]

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

	// ===== Filters dasar (tanggal, foreign keys) =====
	df, err := parseYmd(c.Query("date_from"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYmd(c.Query("date_to"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}

	var teacherIdPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid") }
		teacherIdPtr = &id
	}

	var teacherUserIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("teacher_user_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid") }
		teacherUserIDPtr = &id
	}

	var sectionIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid") }
		sectionIDPtr = &id
	}

	var classSubjectIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("class_subject_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "class_subject_id tidak valid") }
		classSubjectIDPtr = &id
	}

	var csstIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("csst_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "csst_id tidak valid") }
		csstIDPtr = &id
	}

	// ===== Filter keyword (sessions) =====
	keyword := strings.TrimSpace(c.Query("q"))
	var like *string
	if keyword != "" {
		pat := "%" + keyword + "%"
		like = &pat
	}

	// ===== Filter by session ID(s): id / session_id / cas_id =====
	parseUUIDList := func(raw string) ([]uuid.UUID, error) {
		raw = strings.TrimSpace(raw)
		if raw == "" { return nil, nil }
		parts := strings.Split(raw, ",")
		out := make([]uuid.UUID, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" { continue }
			id, err := uuid.Parse(part)
			if err != nil { return nil, fiber.NewError(fiber.StatusBadRequest, "id tidak valid") }
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
		Scopes(
			scopeMasjid(masjidID),
			scopeDateBetween(df, dt),
			scopeCSST(csstIDPtr),
		).
		Where("cas.class_attendance_sessions_deleted_at IS NULL").
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_id = cas.class_attendance_sessions_csst_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`)

	// By-ID filter
	if len(sessionIDs) > 0 {
		qBase = qBase.Where("cas.class_attendance_sessions_id IN ?", sessionIDs)
	}

	// Filter via CSST
	if sectionIDPtr != nil {
		qBase = qBase.Where("csst.class_section_subject_teachers_section_id = ?", *sectionIDPtr)
	}
	if classSubjectIDPtr != nil {
		qBase = qBase.Where("csst.class_section_subject_teachers_class_subjects_id = ?", *classSubjectIDPtr)
	}

	// teacher_id → match CAS.teacher_id ATAU CSST.teacher_id
	if teacherIdPtr != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_sessions_teacher_id = ?
			 OR csst.class_section_subject_teachers_teacher_id = ?)`,
			*teacherIdPtr, *teacherIdPtr,
		)
	}

	// teacher_user_id → map ke users.id via masjid_teachers
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

	// Keyword (ILIKE) untuk sessions
	if like != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_sessions_title ILIKE ?
			 OR cas.class_attendance_sessions_general_info ILIKE ?)`, *like, *like)
	}

	// ===== Scope by role =====
	if !isAdmin {
		if isTeacher {
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			// Guru: hanya sesi yang diajar oleh dirinya (CAS.teacher_id / CSST.teacher_id)
			qBase = qBase.
				Joins(`LEFT JOIN masjid_teachers mt1 ON mt1.masjid_teacher_id = cas.class_attendance_sessions_teacher_id`).
				Joins(`LEFT JOIN masjid_teachers mt2 ON mt2.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id`).
				Where(`
					(mt1.masjid_teacher_user_id = ? AND mt1.masjid_teacher_masjid_id = ?)
				 OR (mt2.masjid_teacher_user_id = ? AND mt2.masjid_teacher_masjid_id = ?)`,
					userID, masjidID, userID, masjidID,
				)
		} else {
			// Siswa/Ortu: hanya sesi di section yang dia ikuti (aktif)
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			qBase = qBase.Where(`
				EXISTS (
				  SELECT 1
				  FROM user_class_sections ucs
				  JOIN user_classes uc
				    ON uc.user_classes_id = ucs.user_class_sections_user_class_id
				  JOIN masjid_students ms
				    ON ms.masjid_student_id = uc.user_classes_masjid_student_id
				   AND ms.masjid_student_deleted_at IS NULL
				  WHERE ucs.user_class_sections_masjid_id = cas.class_attendance_sessions_masjid_id
				    AND ucs.user_class_sections_section_id = csst.class_section_subject_teachers_section_id
				    AND ucs.user_class_sections_unassigned_at IS NULL
				    AND uc.user_classes_status = 'active'
				    AND ms.masjid_student_user_id = ?
				    AND ms.masjid_student_masjid_id = ?
				)`,
				userID, masjidID,
			)
		}
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
		ID        uuid.UUID  `gorm:"column:class_attendance_sessions_id"`
		MasjidID  uuid.UUID  `gorm:"column:class_attendance_sessions_masjid_id"`
		CSSTID    uuid.UUID  `gorm:"column:class_attendance_sessions_csst_id"`
		RoomID    *uuid.UUID `gorm:"column:class_attendance_sessions_class_room_id"`
		Date      time.Time  `gorm:"column:class_attendance_sessions_date"`
		Title     *string    `gorm:"column:class_attendance_sessions_title"`
		General   string     `gorm:"column:class_attendance_sessions_general_info"`
		Note      *string    `gorm:"column:class_attendance_sessions_note"`
		TeacherID *uuid.UUID `gorm:"column:class_attendance_sessions_teacher_id"`
		DeletedAt *time.Time `gorm:"column:class_attendance_sessions_deleted_at"`
		SectionID *uuid.UUID `gorm:"column:section_id"`
		SubjectID *uuid.UUID `gorm:"column:subject_id"`
	}

	var rows []row
	if err := qBase.
		Select(`
			cas.class_attendance_sessions_id,
			cas.class_attendance_sessions_masjid_id,
			cas.class_attendance_sessions_csst_id,
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

	// ===== (Opsional) Prefetch USER_ATTENDANCE untuk page sessions =====
	type UserAttendanceLite struct {
		UserAttendanceID        uuid.UUID  `json:"user_attendance_id"`
		SessionID               uuid.UUID  `json:"user_attendance_session_id"`
		MasjidStudentID         uuid.UUID  `json:"user_attendance_masjid_student_id"`
		Status                  string     `json:"user_attendance_status"`
		TypeID                  *uuid.UUID `json:"user_attendance_type_id,omitempty"`
		Desc                    *string    `json:"user_attendance_desc,omitempty"`
		Score                   *float64   `json:"user_attendance_score,omitempty"`
		IsPassed                *bool      `json:"user_attendance_is_passed,omitempty"`
		UserNote                *string    `json:"user_attendance_user_note,omitempty"`
		TeacherNote             *string    `json:"user_attendance_teacher_note,omitempty"`
		CreatedAt               time.Time  `json:"user_attendance_created_at"`
		UpdatedAt               time.Time  `json:"user_attendance_updated_at"`
	}

	uaMap := map[uuid.UUID][]UserAttendanceLite{}
	if wantUA && len(rows) > 0 {
		// Kumpulkan page session IDs
		pageIDs := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows { pageIDs = append(pageIDs, r.ID) }

		// Parse filter UA
		uaStatus := strings.ToLower(strings.TrimSpace(c.Query("ua_status")))
		var uaTypeIDPtr *uuid.UUID
		if s := strings.TrimSpace(c.Query("ua_type_id")); s != "" {
			if id, e := uuid.Parse(s); e == nil { uaTypeIDPtr = &id } else {
				return fiber.NewError(fiber.StatusBadRequest, "ua_type_id tidak valid")
			}
		}
		// student filter: ua_student_id / masjid_student_id
		var uaStudentIDs []uuid.UUID
		if ids, err := parseUUIDList(c.Query("ua_student_id")); err != nil { return err
		} else if len(ids) > 0 { uaStudentIDs = ids
		} else if ids, err := parseUUIDList(c.Query("masjid_student_id")); err != nil { return err
		} else if len(ids) > 0 { uaStudentIDs = ids }

		uaQuery := strings.TrimSpace(c.Query("ua_q"))
		var uaLike *string
		if uaQuery != "" {
			pat := "%" + uaQuery + "%"
			uaLike = &pat
		}
		// is_passed
		var uaIsPassedPtr *bool
		if s := strings.TrimSpace(c.Query("ua_is_passed")); s != "" {
			b, e := strconv.ParseBool(s)
			if e != nil { return fiber.NewError(fiber.StatusBadRequest, "ua_is_passed tidak valid (true/false)") }
			uaIsPassedPtr = &b
		}

		// Build UA query
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

		// Role-scope untuk Student/Ortu: hanya UA milik dirinya
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

		// Ambil UA rows
		type uaRow struct {
			ID            uuid.UUID  `gorm:"column:user_attendance_id"`
			SessionID     uuid.UUID  `gorm:"column:user_attendance_session_id"`
			StudentID     uuid.UUID  `gorm:"column:user_attendance_masjid_student_id"`
			Status        string     `gorm:"column:user_attendance_status"`
			TypeID        *uuid.UUID `gorm:"column:user_attendance_type_id"`
			Desc          *string    `gorm:"column:user_attendance_desc"`
			Score         *float64   `gorm:"column:user_attendance_score"`
			IsPassed      *bool      `gorm:"column:user_attendance_is_passed"`
			UserNote      *string    `gorm:"column:user_attendance_user_note"`
			TeacherNote   *string    `gorm:"column:user_attendance_teacher_note"`
			CreatedAt     time.Time  `gorm:"column:user_attendance_created_at"`
			UpdatedAt     time.Time  `gorm:"column:user_attendance_updated_at"`
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
			// urutkan stabil: by session, kemudian created_at
			Order("ua.user_attendance_session_id ASC, ua.user_attendance_created_at ASC, ua.user_attendance_id ASC").
			Find(&uaRows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil user_attendance")
		}

		for _, r := range uaRows {
			uaMap[r.SessionID] = append(uaMap[r.SessionID], UserAttendanceLite{
				UserAttendanceID: r.ID,
				SessionID:        r.SessionID,
				MasjidStudentID:  r.StudentID,
				Status:           r.Status,
				TypeID:           r.TypeID,
				Desc:             r.Desc,
				Score:            r.Score,
				IsPassed:         r.IsPassed,
				UserNote:         r.UserNote,
				TeacherNote:      r.TeacherNote,
				CreatedAt:        r.CreatedAt,
				UpdatedAt:        r.UpdatedAt,
			})
		}
	}

	// ===== Compose output =====
	// Base DTO yang sudah ada
	buildBase := func(r row) attendanceDTO.ClassAttendanceSessionResponse {
		return attendanceDTO.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:          r.ID,
			ClassAttendanceSessionMasjidId:    r.MasjidID,
			ClassAttendanceSessionCSSTId:      r.CSSTID,
			ClassAttendanceSessionClassRoomId: r.RoomID,
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

	// Jika include UA, tambahkan field baru; kalau tidak, kirim base saja
	if wantUA {
		type SessionWithUA struct {
			attendanceDTO.ClassAttendanceSessionResponse
			UserAttendance []UserAttendanceLite `json:"user_attendance,omitempty"`
		}
		out := make([]SessionWithUA, 0, len(rows))
		for _, r := range rows {
			out = append(out, SessionWithUA{
				ClassAttendanceSessionResponse: buildBase(r),
				UserAttendance:                 uaMap[r.ID],
			})
		}
		meta := helper.BuildMeta(total, p)
		return helper.JsonList(c, out, meta)
	}

	items := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, buildBase(r))
	}
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}
