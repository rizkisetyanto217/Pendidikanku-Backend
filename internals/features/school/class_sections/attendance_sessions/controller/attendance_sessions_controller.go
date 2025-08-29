package controller

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"

	attendanceDTO "masjidku_backend/internals/features/school/class_sections/attendance_sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/class_sections/attendance_sessions/model"
	secModel "masjidku_backend/internals/features/school/class_sections/main/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionController struct {
	DB *gorm.DB
}

func NewClassAttendanceSessionController(db *gorm.DB) *ClassAttendanceSessionController {
	return &ClassAttendanceSessionController{DB: db}
}


// ========================================================= */
/* =========================================================
   GET /admin/class-attendance-sessions/section/:section_id?date_from=&date_to=&limit=&offset=
========================================================= */
func (ctrl *ClassAttendanceSessionController) ListBySection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	userID, _ := helper.GetUserIDFromToken(c)
	isAdmin := func() bool {
		if mid, err := helper.GetMasjidIDFromToken(c); err == nil && mid == masjidID { return true }
		return false
	}()
	isTeacher := func() bool {
		if mid, err := helper.GetTeacherMasjidIDFromToken(c); err == nil && mid == masjidID { return true }
		return false
	}()

	secID, err := uuid.Parse(strings.TrimSpace(c.Params("section_id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid") }

	var sec secModel.ClassSectionModel
	if err := ctrl.DB.
		Select("class_sections_id, class_sections_masjid_id").
		First(&sec, "class_sections_id = ? AND class_sections_deleted_at IS NULL", secID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
	}
	if sec.ClassSectionsMasjidID == nil || *sec.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
	}

	// Guard siswa/ortu
	if !isAdmin && !isTeacher {
		if userID == uuid.Nil {
			return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
		}
		var cnt int64
		if err := ctrl.DB.Table("user_class_sections AS ucs").
			Joins("JOIN user_classes uc ON uc.user_classes_id = ucs.user_class_sections_user_class_id").
			Where(`
				ucs.user_class_sections_section_id = ?
				AND ucs.user_class_sections_masjid_id = ?
				AND uc.user_classes_user_id = ?
				AND ucs.user_class_sections_unassigned_at IS NULL
				AND uc.user_classes_status = 'active'
			`, secID, masjidID, userID).
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek keanggotaan section")
		}

		if cnt == 0 {
			return fiber.NewError(fiber.StatusForbidden, "Anda tidak terdaftar pada section ini")
		}
	}

	// --- Filter & pagination ---
	parseDate := func(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }

	// Base query
	qBase := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where(`
			class_attendance_sessions_masjid_id = ?
			AND class_attendance_sessions_section_id = ?
		`, masjidID, secID)

	// Filter tanggal
	switch {
	case df != "" && dt != "":
		from, e1 := parseDate(df); if e1 != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
		to,   e2 := parseDate(dt); if e2 != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("class_attendance_sessions_date BETWEEN ? AND ?", from, to)
	case df != "":
		from, e := parseDate(df); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("class_attendance_sessions_date >= ?", from)
	case dt != "":
		to, e := parseDate(dt); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("class_attendance_sessions_date <= ?", to)
	}

	// total count
	var total int64
	if err := qBase.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// Ambil data
	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := qBase.
		Select(`
			class_attendance_sessions_id,
			class_attendance_sessions_section_id,
			class_attendance_sessions_masjid_id,
			class_attendance_sessions_class_subject_id,
			class_attendance_sessions_class_section_subject_teacher_id,
			class_attendance_sessions_date,
			class_attendance_sessions_title,
			class_attendance_sessions_general_info,
			class_attendance_sessions_note,
			class_attendance_sessions_created_at,
			class_attendance_sessions_updated_at,
			class_attendance_sessions_deleted_at
		`).
		Order("class_attendance_sessions_date DESC, class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	return helper.JsonOK(c, "Daftar sesi per section berhasil diambil", fiber.Map{
		"items": out,
		"meta": fiber.Map{
			"limit":     limit,
			"offset":    offset,
			"total":     total,
			"date_from": df,
			"date_to":   dt,
		},
	})
}

/* =========================================================
   LIST by TEACHER (SELF)
   GET /admin/class-attendance-sessions/teacher/me?section_id=&date_from=&date_to=&limit=&offset=
========================================================= */
func (ctrl *ClassAttendanceSessionController) ListMyTeachingSessions(c *fiber.Ctx) error {
	// Harus teacher (tenant dari klaim teacher)
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_ids tidak ditemukan di token")
	}
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
	}

	// Base query: gunakan JOIN ke csst (BUKAN teacher_user_id)
	qBase := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Joins(`
			JOIN class_section_subject_teachers csst
			  ON csst.class_section_subject_teachers_id = class_attendance_sessions_class_section_subject_teacher_id
		`).
		Where(`
			class_attendance_sessions_masjid_id = ?
			AND csst.class_section_subject_teachers_user_id = ?
		`, masjidID, userID).
		// Jika csst punya kolom masjid_id, aktifkan ini agar aman multi-tenant
		Where("csst.class_section_subject_teachers_masjid_id = ?", masjidID).
		// Hindari duplikasi karena JOIN
		Group("class_attendance_sessions_id")

	// ---------- Filter tanggal ----------
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))

	parseDate := func(s string) (time.Time, error) {
		t, err := time.Parse("2006-01-02", s)
		if err != nil { return time.Time{}, err }
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}

	if df == "" && dt == "" {
		// Default: hari ini s/d besok (inklusif)
		now := time.Now().In(time.Local)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		tomorrow := today.Add(24 * time.Hour)
		qBase = qBase.Where("class_attendance_sessions_date BETWEEN ? AND ?", today, tomorrow)
	} else {
		switch {
		case df != "" && dt != "":
			from, e1 := parseDate(df); if e1 != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
			to,   e2 := parseDate(dt); if e2 != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
			qBase = qBase.Where("class_attendance_sessions_date BETWEEN ? AND ?", from, to)
		case df != "":
			from, e := parseDate(df); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
			qBase = qBase.Where("class_attendance_sessions_date >= ?", from)
		case dt != "":
			to, e := parseDate(dt); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
			qBase = qBase.Where("class_attendance_sessions_date <= ?", to)
		}
	}

	// ---------- Filter opsional ----------
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		sid, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("class_attendance_sessions_section_id = ?", sid)
	}

	// ---------- Pagination ----------
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }

	// Total (wrap subquery agar COUNT tidak terduplikasi oleh JOIN/GROUP)
	var total int64
	if err := ctrl.DB.Table("(?) AS sub", qBase.Select("class_attendance_sessions_id")).
		Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// Ambil data
	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := qBase.
		Select(`
			class_attendance_sessions_id,
			class_attendance_sessions_section_id,
			class_attendance_sessions_masjid_id,
			class_attendance_sessions_class_subject_id,
			class_attendance_sessions_class_section_subject_teacher_id,
			class_attendance_sessions_date,
			class_attendance_sessions_title,
			class_attendance_sessions_general_info,
			class_attendance_sessions_note,
			class_attendance_sessions_created_at,
			class_attendance_sessions_updated_at
		`).
		Order("class_attendance_sessions_date DESC, class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	return helper.JsonOK(c, "Daftar sesi mengajar (by token) berhasil diambil", fiber.Map{
		"items": resp,
		"meta": fiber.Map{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

func (ctrl *ClassAttendanceSessionController) ListByMasjid(c *fiber.Ctx) error {
	reqID := uuid.New().String()

	// ===== Guard & tenant resolve =====
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helper.GetTeacherMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil
	isTeacher := teacherMasjidID != uuid.Nil

	log.Printf("[CAS][%s] START ListByMasjid isAdmin=%v isTeacher=%v adminMasjidID=%s teacherMasjidID=%s",
		reqID, isAdmin, isTeacher, adminMasjidID, teacherMasjidID)

	if !isAdmin && !isTeacher {
		log.Printf("[CAS][%s] UNAUTHORIZED (bukan admin/teacher)", reqID)
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// target masjid
	targetMasjidID := uuid.Nil
	if isTeacher {
		targetMasjidID = teacherMasjidID
	}
	if isAdmin {
		targetMasjidID = adminMasjidID
		if s := strings.TrimSpace(c.Query("masjid_id")); s != "" {
			qid, err := uuid.Parse(s)
			if err != nil {
				log.Printf("[CAS][%s] BAD masjid_id query=%q err=%v", reqID, s, err)
				return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak valid")
			}
			if qid != adminMasjidID {
				log.Printf("[CAS][%s] FORBIDDEN access other masjid qid=%s adminMasjidID=%s", reqID, qid, adminMasjidID)
				return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses masjid lain")
			}
			targetMasjidID = qid
		}
	}
	log.Printf("[CAS][%s] targetMasjidID=%s", reqID, targetMasjidID)

	// ===== Filter tanggal =====
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))
	if df == "" && dt == "" {
		now := time.Now().In(time.Local)
		df = now.Format("2006-01-02")
	}
	parseDate := func(s string) (time.Time, error) {
		t, err := time.Parse("2006-01-02", s)
		if err != nil { return time.Time{}, err }
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}
	log.Printf("[CAS][%s] date_from=%q date_to=%q", reqID, df, dt)

	// ===== Base query (pakai alias) =====
	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Where("cas.class_attendance_sessions_masjid_id = ?", targetMasjidID)

	switch {
	case df != "" && dt != "":
		from, e1 := parseDate(df); if e1 != nil { log.Printf("[CAS][%s] BAD date_from=%q err=%v", reqID, df, e1); return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
		to,   e2 := parseDate(dt); if e2 != nil { log.Printf("[CAS][%s] BAD date_to=%q err=%v", reqID, dt, e2); return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("cas.class_attendance_sessions_date BETWEEN ? AND ?", from, to)
	case df != "":
		from, e := parseDate(df); if e != nil { log.Printf("[CAS][%s] BAD date_from=%q err=%v", reqID, df, e); return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ?", from)
	case dt != "":
		to, e := parseDate(dt); if e != nil { log.Printf("[CAS][%s] BAD date_to=%q err=%v", reqID, dt, e); return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("cas.class_attendance_sessions_date <= ?", to)
	}

	// ===== Filter opsional =====
	sectionIDQ := strings.TrimSpace(c.Query("section_id"))
	if sectionIDQ != "" {
		if id, err := uuid.Parse(sectionIDQ); err == nil {
			qBase = qBase.Where("cas.class_attendance_sessions_section_id = ?", id)
		} else {
			log.Printf("[CAS][%s] BAD section_id=%q err=%v", reqID, sectionIDQ, err)
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
	}
	classSubjectIDQ := strings.TrimSpace(c.Query("class_subject_id"))
	if classSubjectIDQ != "" {
		if id, err := uuid.Parse(classSubjectIDQ); err == nil {
			qBase = qBase.Where("cas.class_attendance_sessions_class_subject_id = ?", id)
		} else {
			log.Printf("[CAS][%s] BAD class_subject_id=%q err=%v", reqID, classSubjectIDQ, err)
			return fiber.NewError(fiber.StatusBadRequest, "class_subject_id tidak valid")
		}
	}
	csstIDQ := strings.TrimSpace(c.Query("class_section_subject_teacher_id"))
	if csstIDQ != "" {
		if id, err := uuid.Parse(csstIDQ); err == nil {
			qBase = qBase.Where("cas.class_attendance_sessions_class_section_subject_teacher_id = ?", id)
		} else {
			log.Printf("[CAS][%s] BAD csst_id=%q err=%v", reqID, csstIDQ, err)
			return fiber.NewError(fiber.StatusBadRequest, "class_section_subject_teacher_id tidak valid")
		}
	}

	// ===== JOIN ke CSST, USERS, dan CLASS SECTIONS =====
	qBase = qBase.
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_id = cas.class_attendance_sessions_class_section_subject_teacher_id
		`).
		Joins(`
			LEFT JOIN users AS u
			  ON u.id = csst.class_section_subject_teachers_teacher_user_id
		`).
		Joins(`
			LEFT JOIN class_sections AS cs
			  ON cs.class_sections_id = cas.class_attendance_sessions_section_id
		`)

	// Filter tenant CSST (pakai kurung agar OR tidak bocor)
	qBase = qBase.Where(`
		(csst.class_section_subject_teachers_id IS NULL
		 OR csst.class_section_subject_teachers_masjid_id = ?)
	`, targetMasjidID)

	// filter teacher_user_id -> langsung ke CSST (bukan ke u.id)
	teacherUserIDQ := strings.TrimSpace(c.Query("teacher_user_id"))
	if teacherUserIDQ != "" {
		if id, err := uuid.Parse(teacherUserIDQ); err == nil {
			qBase = qBase.Where("csst.class_section_subject_teachers_teacher_user_id = ?", id)
		} else {
			log.Printf("[CAS][%s] BAD teacher_user_id=%q err=%v", reqID, teacherUserIDQ, err)
			return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid")
		}
	}
	log.Printf("[CAS][%s] filters: section_id=%q class_subject_id=%q csst_id=%q teacher_user_id=%q",
		reqID, sectionIDQ, classSubjectIDQ, csstIDQ, teacherUserIDQ)

	// ===== Sort & Paging =====
	sort := strings.ToLower(strings.TrimSpace(c.Query("sort", "asc")))
	orderClause := "cas.class_attendance_sessions_date ASC, cas.class_attendance_sessions_created_at DESC"
	if sort == "desc" {
		orderClause = "cas.class_attendance_sessions_date DESC, cas.class_attendance_sessions_created_at DESC"
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }
	log.Printf("[CAS][%s] paging: sort=%s order=%s limit=%d offset=%d", reqID, sort, orderClause, limit, offset)

	// ===== Total (DISTINCT) =====
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_sessions_id").
		Count(&total).Error; err != nil {
		log.Printf("[CAS][%s] ERROR Count: %v", reqID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}
	log.Printf("[CAS][%s] total=%d", reqID, total)

	// ===== Fetch =====
	type row struct {
		// CAS
		ID        uuid.UUID  `gorm:"column:class_attendance_sessions_id"`
		SectionID uuid.UUID  `gorm:"column:class_attendance_sessions_section_id"`
		MasjidID  uuid.UUID  `gorm:"column:class_attendance_sessions_masjid_id"`
		Date      time.Time  `gorm:"column:class_attendance_sessions_date"`
		Title     *string    `gorm:"column:class_attendance_sessions_title"`
		General   string     `gorm:"column:class_attendance_sessions_general_info"`
		Note      *string    `gorm:"column:class_attendance_sessions_note"`
		SubjectID *uuid.UUID `gorm:"column:subject_id"`
		CSSTID    *uuid.UUID `gorm:"column:class_attendance_sessions_class_section_subject_teacher_id"`
		CreatedAt time.Time  `gorm:"column:class_attendance_sessions_created_at"`
		UpdatedAt *time.Time `gorm:"column:class_attendance_sessions_updated_at"`
		DeletedAt *time.Time `gorm:"column:class_attendance_sessions_deleted_at"`

		// USERS (teacher)
		TeacherUserID *uuid.UUID `gorm:"column:teacher_user_id"`
		TeacherName   *string    `gorm:"column:teacher_name"`
		TeacherEmail  *string    `gorm:"column:teacher_email"`

		// CLASS SECTIONS (cs)
		SectionSlug     *string `gorm:"column:section_slug"`
		SectionName     *string `gorm:"column:section_name"`
		SectionCode     *string `gorm:"column:section_code"`
		SectionCapacity *int    `gorm:"column:section_capacity"`
		SectionSchedule []byte  `gorm:"column:section_schedule"` // JSONB
	}

	var rows []row
	if err := qBase.Select(`
			-- CAS
			cas.class_attendance_sessions_id,
			cas.class_attendance_sessions_section_id,
			cas.class_attendance_sessions_masjid_id,
			cas.class_attendance_sessions_date,
			cas.class_attendance_sessions_title,
			cas.class_attendance_sessions_general_info,
			cas.class_attendance_sessions_note,
			cas.class_attendance_sessions_class_subject_id AS subject_id,
			cas.class_attendance_sessions_class_section_subject_teacher_id,
			cas.class_attendance_sessions_created_at,
			cas.class_attendance_sessions_updated_at,
			cas.class_attendance_sessions_deleted_at,

			-- USERS
			u.id        AS teacher_user_id,
			u.user_name AS teacher_name,
			u.email     AS teacher_email,

			-- CLASS SECTIONS
			cs.class_sections_slug     AS section_slug,
			cs.class_sections_name     AS section_name,
			cs.class_sections_code     AS section_code,
			cs.class_sections_capacity AS section_capacity,
			cs.class_sections_schedule AS section_schedule
		`).
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		log.Printf("[CAS][%s] ERROR Scan: %v", reqID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	log.Printf("[CAS][%s] rows_len=%d", reqID, len(rows))

	for i := 0; i < len(rows) && i < 5; i++ {
		r := rows[i]
		log.Printf("[CAS][%s] row[%d] id=%s date=%s csst=%s teacher_user_id=%v section=%q",
			reqID, i, r.ID, r.Date.Format("2006-01-02"),
			nilIfNilUUID(r.CSSTID), nilIfNilUUID(r.TeacherUserID),
			ptrStr(r.SectionName),
		)
	}

	// ===== Map ke DTO =====
	items := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp := attendanceDTO.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:                           r.ID,
			ClassAttendanceSessionSectionId:                    r.SectionID,
			ClassAttendanceSessionMasjidId:                     r.MasjidID,
			ClassAttendanceSessionDate:                         r.Date,
			ClassAttendanceSessionTitle:                        r.Title,
			ClassAttendanceSessionGeneralInfo:                  r.General,
			ClassAttendanceSessionNote:                         r.Note,
			ClassAttendanceSessionClassSubjectId:               r.SubjectID,
			ClassAttendanceSessionClassSectionSubjectTeacherId: r.CSSTID,
			ClassAttendanceSessionCreatedAt:                    r.CreatedAt,
			ClassAttendanceSessionUpdatedAt:                    r.UpdatedAt,
			ClassAttendanceSessionDeletedAt:                    r.DeletedAt,
		}
		// teacher info
		resp.ClassAttendanceSessionTeacherUserId = r.TeacherUserID
		resp.ClassAttendanceSessionTeacherName   = r.TeacherName
		// resp.ClassAttendanceSessionTeacherEmail = r.TeacherEmail

		// section info (PASTIKAN field-field ini ada di DTO; kalau belum, tambahkan)
		resp.ClassSectionSlug     = r.SectionSlug
		resp.ClassSectionName     = r.SectionName
		resp.ClassSectionCode     = r.SectionCode
		resp.ClassSectionCapacity = r.SectionCapacity
		resp.ClassSectionSchedule = r.SectionSchedule // tipe []byte kompatibel dengan datatypes.JSON

		items = append(items, resp)
	}

	log.Printf("[CAS][%s] DONE → return items=%d total=%d meta{limit=%d offset=%d sort=%s df=%q dt=%q}",
		reqID, len(items), total, limit, offset, sort, df, dt)

	return helper.JsonList(c, items, fiber.Map{
		"limit":     limit,
		"offset":    offset,
		"total":     int(total),
		"sort":      sort,
		"date_from": df,
		"date_to":   dt,
	})
}


// --- helper kecil untuk logging aman ---
func nilIfNilUUID(p *uuid.UUID) any {
	if p == nil {
		return nil
	}
	return *p
}
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (ctrl *ClassAttendanceSessionController) CreateClassAttendanceSession(c *fiber.Ctx) error {
	// Tenant & Role Guard (admin ATAU teacher)
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helper.GetTeacherMasjidIDFromToken(c)

	var masjidID uuid.UUID
	switch {
	case adminMasjidID != uuid.Nil:
		masjidID = adminMasjidID
	case teacherMasjidID != uuid.Nil:
		masjidID = teacherMasjidID
	default:
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// Parse payload
	var req attendanceDTO.CreateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant & normalisasi tanggal (DATE @ local)
	req.ClassAttendanceSessionMasjidId = masjidID
	{
		d := req.ClassAttendanceSessionDate.In(time.Local)
		req.ClassAttendanceSessionDate = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
	}

	// Trim teks ringan
	if req.ClassAttendanceSessionTitle != nil {
		t := strings.TrimSpace(*req.ClassAttendanceSessionTitle)
		req.ClassAttendanceSessionTitle = &t
	}
	req.ClassAttendanceSessionGeneralInfo = strings.TrimSpace(req.ClassAttendanceSessionGeneralInfo)
	if req.ClassAttendanceSessionNote != nil {
		n := strings.TrimSpace(*req.ClassAttendanceSessionNote)
		req.ClassAttendanceSessionNote = &n
	}

	// Validasi payload
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Transaksi kecil
	// Transaksi kecil
	err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Validasi section milik masjid
		var sec secModel.ClassSectionModel
		if err := tx.
			Select("class_sections_id, class_sections_masjid_id, class_sections_is_active").
			First(&sec, "class_sections_id = ? AND class_sections_deleted_at IS NULL", req.ClassAttendanceSessionSectionId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil section")
		}
		if sec.ClassSectionsMasjidID == nil || *sec.ClassSectionsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
		}

		// 2) Cek duplikat aktif (aturan unik: (masjid, section, date, class_subject_id NULL/NOT NULL))
		var dupeCount int64
		dupeQ := tx.Model(&attendanceModel.ClassAttendanceSessionModel{}).
			Where(`
				class_attendance_sessions_masjid_id = ?
				AND class_attendance_sessions_section_id = ?
				AND class_attendance_sessions_date = ?
			`,
				req.ClassAttendanceSessionMasjidId,
				req.ClassAttendanceSessionSectionId,
				req.ClassAttendanceSessionDate,
			)

		if req.ClassAttendanceSessionClassSubjectId != nil {
			dupeQ = dupeQ.Where("class_attendance_sessions_class_subject_id = ?", *req.ClassAttendanceSessionClassSubjectId)
		} else {
			dupeQ = dupeQ.Where("class_attendance_sessions_class_subject_id IS NULL")
		}

		if err := dupeQ.Count(&dupeCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if dupeCount > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// 3) Validasi CSST (jika diisi)
		if req.ClassAttendanceSessionClassSectionSubjectTeacherId != nil {
			var csstCnt int64
			csstQ := tx.Table("class_section_subject_teachers").
				Where("class_section_subject_teachers_id = ?", *req.ClassAttendanceSessionClassSectionSubjectTeacherId).
				Where("class_section_subject_teachers_masjid_id = ?", masjidID)

			// (opsional) pastikan CSST sesuai section
			csstQ = csstQ.Where("class_section_subject_teachers_section_id = ?", req.ClassAttendanceSessionSectionId)

			// (opsional) jika Anda menyimpan subject di CSST, validasi konsistensi subject
			if req.ClassAttendanceSessionClassSubjectId != nil {
				csstQ = csstQ.Where(`
					(class_section_subject_teachers_class_subject_id IS NULL OR class_section_subject_teachers_class_subject_id = ?)
				`, *req.ClassAttendanceSessionClassSubjectId)
			}

			if err := csstQ.Count(&csstCnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi penugasan guru (CSST)")
			}
			if csstCnt == 0 {
				return fiber.NewError(fiber.StatusBadRequest, "Penugasan guru (CSST) tidak valid untuk masjid/section/subject ini")
			}
		}

		// 4) Simpan
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
			}
			if strings.Contains(low, "mismatch") || strings.Contains(low, "invalid") {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kehadiran")
		}

		c.Locals("created_model", m)
		return nil
	})

	if err != nil {
		return err
	}

	// Response
	m := c.Locals("created_model").(attendanceModel.ClassAttendanceSessionModel)
	c.Set("Location", fmt.Sprintf("/admin/class-attendance-sessions/%s", m.ClassAttendanceSessionId.String()))
	return helper.JsonCreated(c, "Sesi kehadiran berhasil dibuat", attendanceDTO.FromClassAttendanceSessionModel(m))
}


// PUT /admin/class-attendance-sessions/:id
func (ctrl *ClassAttendanceSessionController) UpdateClassAttendanceSession(c *fiber.Ctx) error {
	// ===== Role & Tenant (admin ATAU teacher) =====
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helper.GetTeacherMasjidIDFromToken(c)

	var masjidID uuid.UUID
	switch {
	case adminMasjidID != uuid.Nil:
		masjidID = adminMasjidID
	case teacherMasjidID != uuid.Nil:
		masjidID = teacherMasjidID
	default:
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// ===== Path param =====
	sessionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// ===== Parse payload =====
	var req attendanceDTO.UpdateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Paksa tenant (tidak boleh pindah masjid)
	req.ClassAttendanceSessionMasjidId = &masjidID

	// Trim ringan
	if req.ClassAttendanceSessionTitle != nil {
		t := strings.TrimSpace(*req.ClassAttendanceSessionTitle)
		req.ClassAttendanceSessionTitle = &t
	}
	if req.ClassAttendanceSessionGeneralInfo != nil {
		g := strings.TrimSpace(*req.ClassAttendanceSessionGeneralInfo)
		req.ClassAttendanceSessionGeneralInfo = &g
	}
	if req.ClassAttendanceSessionNote != nil {
		n := strings.TrimSpace(*req.ClassAttendanceSessionNote)
		req.ClassAttendanceSessionNote = &n
	}

	// Normalisasi tanggal → DATE (local)
	if req.ClassAttendanceSessionDate != nil {
		d := req.ClassAttendanceSessionDate.In(time.Local)
		dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
		req.ClassAttendanceSessionDate = &dd
	}

	// Validasi DTO
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ===== Transaksi agar atomic =====
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Ambil existing + guard tenant
		var existing attendanceModel.ClassAttendanceSessionModel
		if err := tx.First(&existing, "class_attendance_sessions_id = ?", sessionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if existing.ClassAttendanceSessionMasjidId != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah sesi milik masjid lain")
		}

		// 2) Validasi section baru (bila diganti)
		targetSectionID := existing.ClassAttendanceSessionSectionId
		if req.ClassAttendanceSessionSectionId != nil {
			targetSectionID = *req.ClassAttendanceSessionSectionId
			var sec secModel.ClassSectionModel
			if err := tx.
				Select("class_sections_id, class_sections_masjid_id, class_sections_is_active").
				First(&sec, "class_sections_id = ? AND class_sections_deleted_at IS NULL", targetSectionID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil section")
			}
			if sec.ClassSectionsMasjidID == nil || *sec.ClassSectionsMasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
			}
		}

		// 3) (hapus) Validasi guru by user_id — TIDAK DIPERLUKAN lagi

		// 4) Cek unik (aturan baru)
		targetDate := existing.ClassAttendanceSessionDate
		if req.ClassAttendanceSessionDate != nil {
			targetDate = *req.ClassAttendanceSessionDate
		}
		targetClassSubjectID := existing.ClassAttendanceSessionClassSubjectId
		if req.ClassAttendanceSessionClassSubjectId != nil {
			targetClassSubjectID = req.ClassAttendanceSessionClassSubjectId
		}

		dupe := tx.Model(&attendanceModel.ClassAttendanceSessionModel{}).
			Where(`
				class_attendance_sessions_masjid_id = ?
				AND class_attendance_sessions_section_id = ?
				AND class_attendance_sessions_date = ?
				AND class_attendance_sessions_id <> ?
				AND class_attendance_sessions_deleted_at IS NULL
			`, masjidID, targetSectionID, targetDate, existing.ClassAttendanceSessionId)

		if targetClassSubjectID != nil {
			dupe = dupe.Where("class_attendance_sessions_class_subject_id = ?", *targetClassSubjectID)
		} else {
			dupe = dupe.Where("class_attendance_sessions_class_subject_id IS NULL")
		}

		var cnt int64
		if err := dupe.Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// 5) Validasi CSST (jika diubah)
		if req.ClassAttendanceSessionClassSectionSubjectTeacherId != nil {
			var csstCnt int64
			csstQ := tx.Table("class_section_subject_teachers").
				Where("class_section_subject_teachers_id = ?", *req.ClassAttendanceSessionClassSectionSubjectTeacherId).
				Where("class_section_subject_teachers_masjid_id = ?", masjidID).
				Where("class_section_subject_teachers_section_id = ?", targetSectionID)

			if targetClassSubjectID != nil {
				csstQ = csstQ.Where(`
					(class_section_subject_teachers_class_subject_id IS NULL OR class_section_subject_teachers_class_subject_id = ?)
				`, *targetClassSubjectID)
			}

			if err := csstQ.Count(&csstCnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi penugasan guru (CSST)")
			}
			if csstCnt == 0 {
				return fiber.NewError(fiber.StatusBadRequest, "Penugasan guru (CSST) tidak valid untuk masjid/section/subject ini")
			}
		}

		// 6) Terapkan perubahan (patch terarah)
		patch := map[string]interface{}{}
		if req.ClassAttendanceSessionSectionId != nil {
			patch["class_attendance_sessions_section_id"] = *req.ClassAttendanceSessionSectionId
			existing.ClassAttendanceSessionSectionId = *req.ClassAttendanceSessionSectionId
		}
		patch["class_attendance_sessions_masjid_id"] = masjidID
		existing.ClassAttendanceSessionMasjidId = masjidID

		if req.ClassAttendanceSessionDate != nil {
			patch["class_attendance_sessions_date"] = *req.ClassAttendanceSessionDate
			existing.ClassAttendanceSessionDate = *req.ClassAttendanceSessionDate
		}
		if req.ClassAttendanceSessionTitle != nil {
			patch["class_attendance_sessions_title"] = req.ClassAttendanceSessionTitle
			existing.ClassAttendanceSessionTitle = req.ClassAttendanceSessionTitle
		}
		if req.ClassAttendanceSessionGeneralInfo != nil {
			patch["class_attendance_sessions_general_info"] = *req.ClassAttendanceSessionGeneralInfo
			existing.ClassAttendanceSessionGeneralInfo = *req.ClassAttendanceSessionGeneralInfo
		}
		if req.ClassAttendanceSessionNote != nil {
			patch["class_attendance_sessions_note"] = req.ClassAttendanceSessionNote
			existing.ClassAttendanceSessionNote = req.ClassAttendanceSessionNote
		}
		// (hapus) class_attendance_sessions_teacher_user_id

		if req.ClassAttendanceSessionClassSubjectId != nil {
			patch["class_attendance_sessions_class_subject_id"] = req.ClassAttendanceSessionClassSubjectId
			existing.ClassAttendanceSessionClassSubjectId = req.ClassAttendanceSessionClassSubjectId
		}
		if req.ClassAttendanceSessionClassSectionSubjectTeacherId != nil {
			patch["class_attendance_sessions_class_section_subject_teacher_id"] = req.ClassAttendanceSessionClassSectionSubjectTeacherId
			existing.ClassAttendanceSessionClassSectionSubjectTeacherId = req.ClassAttendanceSessionClassSectionSubjectTeacherId
		}

		now := time.Now().In(time.Local)
		patch["class_attendance_sessions_updated_at"] = now
		existing.ClassAttendanceSessionUpdatedAt = &now

		if err := tx.Model(&attendanceModel.ClassAttendanceSessionModel{}).
			Where("class_attendance_sessions_id = ?", existing.ClassAttendanceSessionId).
			Select([]string{
				"class_attendance_sessions_section_id",
				"class_attendance_sessions_masjid_id",
				"class_attendance_sessions_date",
				"class_attendance_sessions_title",
				"class_attendance_sessions_general_info",
				"class_attendance_sessions_note",
				// "class_attendance_sessions_teacher_user_id", // ← DIHAPUS
				"class_attendance_sessions_class_subject_id",
				"class_attendance_sessions_class_section_subject_teacher_id",
				"class_attendance_sessions_updated_at",
			}).
			Updates(patch).Error; err != nil {

			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
			}
			if strings.Contains(msg, "mismatch") || strings.Contains(msg, "invalid") {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi kehadiran")
		}

		c.Locals("updated_model", existing)
		return nil
	}); err != nil {
		return err
	}


	updated := c.Locals("updated_model").(attendanceModel.ClassAttendanceSessionModel)
	return helper.JsonUpdated(c, "Sesi kehadiran berhasil diperbarui", attendanceDTO.FromClassAttendanceSessionModel(updated))
}

// DELETE /admin/class-attendance-sessions/:id?force=true
func (ctrl *ClassAttendanceSessionController) DeleteClassAttendanceSession(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID

	sessionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		var existing attendanceModel.ClassAttendanceSessionModel
		if err := tx.
			First(&existing, "class_attendance_sessions_id = ?", sessionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if existing.ClassAttendanceSessionMasjidId != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus sesi milik masjid lain")
		}

		var delErr error
		if force {
			// Hard delete: bypass soft-delete
			delErr = tx.Unscoped().
				Delete(&attendanceModel.ClassAttendanceSessionModel{}, "class_attendance_sessions_id = ?", sessionID).Error
		} else {
			// Soft delete (default)
			delErr = tx.
				Delete(&attendanceModel.ClassAttendanceSessionModel{}, "class_attendance_sessions_id = ?", sessionID).Error
		}
		if delErr != nil {
			msg := strings.ToLower(delErr.Error())
			if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
				return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus sesi kehadiran")
		}

		c.Locals("deleted_model", existing)
		return nil
	}); err != nil {
		return err
	}

	deleted := c.Locals("deleted_model").(attendanceModel.ClassAttendanceSessionModel)
	return helper.JsonDeleted(c, "Sesi kehadiran berhasil dihapus", attendanceDTO.FromClassAttendanceSessionModel(deleted))
}
