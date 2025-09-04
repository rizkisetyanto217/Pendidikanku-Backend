package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	attendanceDTO "masjidku_backend/internals/features/school/sessions_assesment/sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ClassAttendanceSessionController struct {
	DB *gorm.DB
}

func NewClassAttendanceSessionController(db *gorm.DB) *ClassAttendanceSessionController {
	return &ClassAttendanceSessionController{DB: db}
}

// =========================================================
/* =========================================================
   GET /admin/class-attendance-sessions/section/:section_id?date_from=&date_to=&limit=&offset=
========================================================= */
func (ctrl *ClassAttendanceSessionController) ListBySection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	userID, _ := helperAuth.GetUserIDFromToken(c)
	isAdmin := func() bool {
		if mid, err := helperAuth.GetMasjidIDFromToken(c); err == nil && mid == masjidID { return true }
		return false
	}()
	isTeacher := func() bool {
		if mid, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && mid == masjidID { return true }
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

	// ✅ cukup compare value
	if sec.ClassSectionsMasjidID != masjidID {
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

	// ===== Pagination & sorting
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
	p := helper.ParseWith(httpReq, "date", "desc", helper.AdminOpts)

	allowedSort := map[string]string{
		"date":       "class_attendance_sessions_date",
		"created_at": "class_attendance_sessions_created_at",
		"title":      "class_attendance_sessions_title",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "date")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// ===== Filter tanggal
	parseYmd := func(s string) (*time.Time, error) {
		s = strings.TrimSpace(s)
		if s == "" { return nil, nil }
		t, err := time.Parse("2006-01-02", s)
		if err != nil { return nil, err }
		tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		return &tt, nil
	}
	df, err := parseYmd(c.Query("date_from"))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
	dt, err := parseYmd(c.Query("date_to"))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }

	// ===== Base query
	qBase := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where(`
			class_attendance_sessions_masjid_id = ?
			AND class_attendance_sessions_section_id = ?
		`, masjidID, secID)

	if df != nil && dt != nil {
		qBase = qBase.Where("class_attendance_sessions_date BETWEEN ? AND ?", *df, *dt)
	} else if df != nil {
		qBase = qBase.Where("class_attendance_sessions_date >= ?", *df)
	} else if dt != nil {
		qBase = qBase.Where("class_attendance_sessions_date <= ?", *dt)
	}

	// total
	var total int64
	if err := qBase.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// data
	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := qBase.
		Select(`
			class_attendance_sessions_id,
			class_attendance_sessions_section_id,
			class_attendance_sessions_masjid_id,
			class_attendance_sessions_class_subject_id,
			class_attendance_sessions_teacher_id,
			class_attendance_sessions_date,
			class_attendance_sessions_title,
			class_attendance_sessions_general_info,
			class_attendance_sessions_note,
			class_attendance_sessions_created_at,
			class_attendance_sessions_updated_at,
			class_attendance_sessions_deleted_at
		`).
		Order(orderExpr).
		Order("class_attendance_sessions_created_at DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}



// =========================================================
// LIST by TEACHER (SELF)
// GET /admin/class-attendance-sessions/teacher/me?section_id=&date_from=&date_to=&limit=&offset=
// =========================================================
func (ctrl *ClassAttendanceSessionController) ListMyTeachingSessions(c *fiber.Ctx) error {
	// ===== Auth: harus teacher
	masjidID, err := helperAuth.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_ids tidak ditemukan di token")
	}
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
	}

	// ===== Pagination & sorting
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
	p := helper.ParseWith(httpReq, "date", "desc", helper.AdminOpts)

	allowedSort := map[string]string{
		"date":       "class_attendance_sessions_date",
		"created_at": "class_attendance_sessions_created_at",
		"title":      "class_attendance_sessions_title",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "date")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// ===== Helper tanggal
	parseYmd := func(s string) (*time.Time, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			return nil, err
		}
		t0 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		return &t0, nil
	}

	df, err := parseYmd(c.Query("date_from"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYmd(c.Query("date_to"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}
	// eksklusif-kan upper bound (to+1d) agar inklusif tanggal
	var lo, hi *time.Time
	if df == nil && dt == nil {
		now := time.Now().In(time.Local)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		tomorrow := today.Add(24 * time.Hour)
		lo, hi = &today, &tomorrow
	} else {
		lo = df
		if dt != nil {
			tomorrow := dt.Add(24 * time.Hour)
			hi = &tomorrow
		}
	}

	// ===== Base query
	// NOTE: pakai kolom singular di masjid_teachers (konsisten dgn masjid_teacher_id)
	qBase := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Joins(`
			JOIN masjid_teachers mt
			  ON mt.masjid_teacher_id = class_attendance_sessions_teacher_id
			 AND mt.masjid_teacher_deleted_at IS NULL
		`).
		Where(`
			class_attendance_sessions_masjid_id = ?
			AND mt.masjid_teacher_user_id = ?
			AND mt.masjid_teacher_masjid_id = ?
		`, masjidID, userID, masjidID)

	// Filter tanggal [lo, hi)
	if lo != nil && hi != nil {
		qBase = qBase.Where("class_attendance_sessions_date >= ? AND class_attendance_sessions_date < ?", *lo, *hi)
	} else if lo != nil {
		qBase = qBase.Where("class_attendance_sessions_date >= ?", *lo)
	} else if hi != nil {
		qBase = qBase.Where("class_attendance_sessions_date < ?", *hi)
	}

	// Filter opsional: section_id
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		sid, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("class_attendance_sessions_section_id = ?", sid)
	}

	// ===== Total (distinct untuk jaga-jaga dari duplikasi join)
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("class_attendance_sessions_id").
		Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Data
	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := qBase.
		Select(`
			class_attendance_sessions_id,
			class_attendance_sessions_section_id,
			class_attendance_sessions_masjid_id,
			class_attendance_sessions_class_subject_id,
			class_attendance_sessions_teacher_id,
			class_attendance_sessions_date,
			class_attendance_sessions_title,
			class_attendance_sessions_general_info,
			class_attendance_sessions_note,
			class_attendance_sessions_created_at,
			class_attendance_sessions_updated_at
		`).
		Order(orderExpr).
		Order("class_attendance_sessions_created_at DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, resp, meta)
}


func (ctrl *ClassAttendanceSessionController) ListByMasjid(c *fiber.Ctx) error {
	reqID := uuid.New().String()

	// ===== Guard & tenant resolve =====
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)
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

	// ===== Pagination & sorting (helper) =====
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

	// ===== Filter tanggal (default: today jika kosong)
	parseYmd := func(s string) (*time.Time, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return nil, err
		}
		tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		return &tt, nil
	}
	dfStr := strings.TrimSpace(c.Query("date_from"))
	dtStr := strings.TrimSpace(c.Query("date_to"))
	if dfStr == "" && dtStr == "" {
		now := time.Now().In(time.Local)
		dfStr = now.Format("2006-01-02")
	}
	df, err := parseYmd(dfStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYmd(dtStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}
	log.Printf("[CAS][%s] date_from=%q date_to=%q", reqID, dfStr, dtStr)

	// ===== Base query (pakai alias) =====
	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Where("cas.class_attendance_sessions_masjid_id = ?", targetMasjidID).
		Where("cas.class_attendance_sessions_deleted_at IS NULL") // alive only

	// range tanggal
	if df != nil && dt != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date BETWEEN ? AND ?", *df, *dt)
	} else if df != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ?", *df)
	} else if dt != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date <= ?", *dt)
	}

	// ===== Filter opsional =====
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			log.Printf("[CAS][%s] BAD section_id=%q err=%v", reqID, s, e)
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_section_id = ?", id)
	}
	if s := strings.TrimSpace(c.Query("class_subject_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			log.Printf("[CAS][%s] BAD class_subject_id=%q err=%v", reqID, s, e)
			return fiber.NewError(fiber.StatusBadRequest, "class_subject_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_class_subject_id = ?", id)
	}
	if s := strings.TrimSpace(c.Query("room_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			log.Printf("[CAS][%s] BAD room_id=%q err=%v", reqID, s, e)
			return fiber.NewError(fiber.StatusBadRequest, "room_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_class_room_id = ?", id)
	}
	// filter berdasarkan masjid_teacher_id langsung di CAS
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			log.Printf("[CAS][%s] BAD teacher_id=%q err=%v", reqID, s, e)
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_teacher_id = ?", id)
	}
	// legacy: filter berdasarkan users.id guru via masjid_teachers
	var teacherUserID *uuid.UUID
	if s := strings.TrimSpace(c.Query("teacher_user_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			log.Printf("[CAS][%s] BAD teacher_user_id=%q err=%v", reqID, s, e)
			return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid")
		}
		teacherUserID = &id
	}

	// keyword (opsional) → cari di title / general_info
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		pat := "%" + q + "%"
		qBase = qBase.Where(`(cas.class_attendance_sessions_title ILIKE ? OR cas.class_attendance_sessions_general_info ILIKE ?)`, pat, pat)
	}

	log.Printf("[CAS][%s] filters: section_id=%q class_subject_id=%q room_id=%q teacher_id=%q teacher_user_id=%v q=%q",
		reqID, c.Query("section_id"), c.Query("class_subject_id"), c.Query("room_id"), c.Query("teacher_id"), teacherUserID, c.Query("q"))

	// ===== JOINs: masjid_teachers (mt), users (u), class_sections (cs)
	qBase = qBase.
		Joins(`
            LEFT JOIN masjid_teachers AS mt
              ON mt.masjid_teacher_id = cas.class_attendance_sessions_teacher_id
        `).
		Joins(`
            LEFT JOIN users AS u
              ON u.id = mt.masjid_teachers_user_id
        `).
		Joins(`
            LEFT JOIN class_sections AS cs
              ON cs.class_sections_id = cas.class_attendance_sessions_section_id
        `)

	// Tenant-guard untuk mt (jaga kasus teacher_id NULL tetap lolos)
	qBase = qBase.Where(`(mt.masjid_teacher_id IS NULL OR mt.masjid_teacher_masjid_id = ?)`, targetMasjidID)
	if teacherUserID != nil {
		qBase = qBase.Where("mt.masjid_teachers_user_id = ?", *teacherUserID)
	}

	// ===== Total (DISTINCT by CAS id) =====
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
		TeacherId *uuid.UUID `gorm:"column:teacher_id"`
		RoomId    *uuid.UUID `gorm:"column:room_id"`
		DeletedAt *time.Time `gorm:"column:class_attendance_sessions_deleted_at"`

		// USERS (via mt)
		TeacherName  *string `gorm:"column:teacher_name"`
		TeacherEmail *string `gorm:"column:teacher_email"`

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
            cas.class_attendance_sessions_teacher_id       AS teacher_id,
            cas.class_attendance_sessions_class_room_id    AS room_id,
            cas.class_attendance_sessions_deleted_at,

            -- USERS (via masjid_teachers)
            u.user_name AS teacher_name,
            u.email     AS teacher_email,

            -- CLASS SECTIONS
            cs.class_sections_slug     AS section_slug,
            cs.class_sections_name     AS section_name,
            cs.class_sections_code     AS section_code,
            cs.class_sections_capacity AS section_capacity,
            cs.class_sections_schedule AS section_schedule
        `).
		Order(orderExpr).
		// fallback stabilizer (tanpa created_at)
		Order("cas.class_attendance_sessions_id DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		log.Printf("[CAS][%s] ERROR Scan: %v", reqID, err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	log.Printf("[CAS][%s] rows_len=%d", reqID, len(rows))

	// ===== Map ke DTO =====
	items := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp := attendanceDTO.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:             r.ID,
			ClassAttendanceSessionSectionId:      r.SectionID,
			ClassAttendanceSessionMasjidId:       r.MasjidID,
			ClassAttendanceSessionDate:           r.Date,
			ClassAttendanceSessionTitle:          r.Title,
			ClassAttendanceSessionGeneralInfo:    r.General,
			ClassAttendanceSessionNote:           r.Note,
			ClassAttendanceSessionClassSubjectId: r.SubjectID, // *uuid.UUID
			ClassAttendanceSessionClassRoomId:    r.RoomId,    // *uuid.UUID
			ClassAttendanceSessionTeacherId:      r.TeacherId, // *uuid.UUID

			// section info
			ClassSectionSlug:     r.SectionSlug,
			ClassSectionName:     r.SectionName,
			ClassSectionCode:     r.SectionCode,
			ClassSectionCapacity: r.SectionCapacity,
			ClassSectionSchedule: datatypes.JSON(r.SectionSchedule),

			// timestamps: hanya deleted_at (sesuai skema)
			ClassAttendanceSessionDeletedAt: r.DeletedAt,
		}

		// teacher cosmetic info (opsional dari join users)
		resp.ClassAttendanceSessionTeacherName = r.TeacherName
		resp.ClassAttendanceSessionTeacherEmail = r.TeacherEmail

		items = append(items, resp)
	}

	meta := helper.BuildMeta(total, p)
	log.Printf("[CAS][%s] DONE → return items=%d total=%d meta=%+v df=%q dt=%q",
		reqID, len(items), total, meta, dfStr, dtStr)

	return helper.JsonList(c, items, meta)
}


// tambahkan di import kalau belum:
// import (
//   "errors"
//   "fmt"
//   "log"
//   "net/http"
//   "net/url"
//   "strings"
//   "time"
//
//   "github.com/gofiber/fiber/v2"
//   "github.com/google/uuid"
//   "gorm.io/gorm"
// )

func (ctrl *ClassAttendanceSessionController) CreateClassAttendanceSession(c *fiber.Ctx) error {
	// ===== Tenant & Role Guard (admin ATAU teacher)
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		log.Printf("[CAS:create] unauthorized: no admin/teacher tenant in token: %v", err)
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}
	role := helperAuth.GetRole(c)

	// untuk validasi “teacher hanya boleh memakai dirinya sendiri”
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)
	userID, _ := helperAuth.GetUserIDFromToken(c)

	log.Printf("[CAS:create] start masjid_id=%s user_id=%s role=%s", masjidID, userID, role)

	// ===== Parse payload
	var req attendanceDTO.CreateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[CAS:create] body parse error: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// ===== Force tenant & normalisasi tanggal (jika diisi)
	req.ClassAttendanceSessionMasjidId = masjidID
	if req.ClassAttendanceSessionDate != nil {
		d := req.ClassAttendanceSessionDate.In(time.Local)
		dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
		req.ClassAttendanceSessionDate = &dd
	}

	// ===== Trim
	if req.ClassAttendanceSessionTitle != nil {
		t := strings.TrimSpace(*req.ClassAttendanceSessionTitle)
		req.ClassAttendanceSessionTitle = &t
	}
	req.ClassAttendanceSessionGeneralInfo = strings.TrimSpace(req.ClassAttendanceSessionGeneralInfo)
	if req.ClassAttendanceSessionNote != nil {
		n := strings.TrimSpace(*req.ClassAttendanceSessionNote)
		req.ClassAttendanceSessionNote = &n
	}

	// ===== Validasi payload (sesuai DTO)
	if err := validator.New().Struct(req); err != nil {
		log.Printf("[CAS:create] payload validation error: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	log.Printf(
		"[CAS:create] payload normalized: masjid_id=%s section_id=%s class_subject_id=%s teacher_id=%v room_id=%v date=%v",
		req.ClassAttendanceSessionMasjidId,
		req.ClassAttendanceSessionSectionId,
		req.ClassAttendanceSessionClassSubjectId,
		req.ClassAttendanceSessionTeacherId,
		req.ClassAttendanceSessionClassRoomId,
		func() string {
			if req.ClassAttendanceSessionDate == nil {
				return "<nil>"
			}
			return req.ClassAttendanceSessionDate.Format("2006-01-02")
		}(),
	)

	// ===== Transaksi
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Validasi SECTION milik masjid
		var sec struct {
			MasjidID *uuid.UUID `gorm:"column:class_sections_masjid_id"`
			ClassID  *uuid.UUID `gorm:"column:class_sections_class_id"`
		}
		if err := tx.Table("class_sections").
			Select("class_sections_masjid_id, class_sections_class_id").
			Where("class_sections_id = ? AND class_sections_deleted_at IS NULL", req.ClassAttendanceSessionSectionId).
			Take(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("[CAS:create] section not found: section_id=%s", req.ClassAttendanceSessionSectionId)
				return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
			}
			log.Printf("[CAS:create] section query error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil section")
		}
		if sec.MasjidID == nil || *sec.MasjidID != masjidID {
			log.Printf("[CAS:create] section tenant mismatch: section.msj=%v req.msj=%s", sec.MasjidID, masjidID)
			return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
		}

		// 2) Validasi CLASS_SUBJECT (milik masjid & class match)
		var cs struct {
			MasjidID uuid.UUID  `gorm:"column:class_subjects_masjid_id"`
			ClassID  *uuid.UUID `gorm:"column:class_subjects_class_id"`
		}
		if err := tx.Table("class_subjects").
			Select("class_subjects_masjid_id, class_subjects_class_id").
			Where("class_subjects_id = ? AND class_subjects_deleted_at IS NULL", req.ClassAttendanceSessionClassSubjectId).
			Take(&cs).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("[CAS:create] class_subject not found: cs_id=%s", req.ClassAttendanceSessionClassSubjectId)
				return fiber.NewError(fiber.StatusBadRequest, "Class subject tidak ditemukan")
			}
			log.Printf("[CAS:create] class_subject query error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil class subject")
		}
		if cs.MasjidID != masjidID {
			log.Printf("[CAS:create] class_subject tenant mismatch: cs.msj=%s req.msj=%s", cs.MasjidID, masjidID)
			return fiber.NewError(fiber.StatusForbidden, "Class subject bukan milik masjid Anda")
		}
		if sec.ClassID == nil || cs.ClassID == nil || *sec.ClassID != *cs.ClassID {
			log.Printf("[CAS:create] class_subject vs section.class mismatch: section.class=%v cs.class=%v", sec.ClassID, cs.ClassID)
			return fiber.NewError(fiber.StatusBadRequest, "Class subject tidak sesuai dengan class pada section")
		}

		// 3) Validasi TEACHER (opsional)
		if req.ClassAttendanceSessionTeacherId != nil {
			var row struct {
				MasjidID string `gorm:"column:masjid_id"`
				UserID   string `gorm:"column:user_id"`
			}
			if err := tx.
				Table("masjid_teachers mt").
				Select(`mt.masjid_teacher_masjid_id::text AS masjid_id, mt.masjid_teacher_user_id::text AS user_id`).
				Where("mt.masjid_teacher_id = ? AND mt.masjid_teacher_deleted_at IS NULL", *req.ClassAttendanceSessionTeacherId).
				Take(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					log.Printf("[CAS:create] teacher not found: mt_id=%s", *req.ClassAttendanceSessionTeacherId)
					return fiber.NewError(fiber.StatusBadRequest, "Guru (masjid_teacher) tidak ditemukan")
				}
				log.Printf("[CAS:create] teacher query error: %v", err)
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi guru")
			}

			mtMasjid, err := uuid.Parse(row.MasjidID)
			if err != nil {
				log.Printf("[CAS:create] teacher masjid_id parse error: %v (raw=%q)", err, row.MasjidID)
				return fiber.NewError(fiber.StatusInternalServerError, "Format masjid_id guru tidak valid")
			}
			if mtMasjid != masjidID {
				log.Printf("[CAS:create] teacher tenant mismatch: mt.msj=%s req.msj=%s", mtMasjid, masjidID)
				return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik masjid Anda")
			}

			// Jika caller adalah TEACHER (bukan admin), pastikan teacher_id milik user yang login
			if teacherMasjidID != uuid.Nil && userID != uuid.Nil {
				mtUser, err := uuid.Parse(row.UserID)
				if err != nil || mtUser != userID {
					log.Printf("[CAS:create] teacher-id not owned by caller: mt.user=%v caller.user=%v err=%v", row.UserID, userID, err)
					return fiber.NewError(fiber.StatusForbidden, "Guru pada payload bukan akun Anda")
				}
			}
			log.Printf("[CAS:create] teacher validated: mt_id=%s masjid_id=%s user_id=%s", *req.ClassAttendanceSessionTeacherId, mtMasjid, row.UserID)
		} else {
			log.Printf("[CAS:create] no teacher_id provided (optional)")
		}

		// 3b) Validasi ROOM (opsional)
		if req.ClassAttendanceSessionClassRoomId != nil {
			var room struct {
				MasjidID  uuid.UUID  `gorm:"column:class_rooms_masjid_id"`
				DeletedAt *time.Time `gorm:"column:deleted_at"`
			}
			if err := tx.Table("class_rooms").
				Select("class_rooms_masjid_id, deleted_at").
				Where("class_room_id = ?", *req.ClassAttendanceSessionClassRoomId).
				Take(&room).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					log.Printf("[CAS:create] room not found: room_id=%s", *req.ClassAttendanceSessionClassRoomId)
					return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
				}
				log.Printf("[CAS:create] room query error: %v", err)
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang")
			}
			if room.DeletedAt != nil {
				log.Printf("[CAS:create] room soft-deleted: room_id=%s", *req.ClassAttendanceSessionClassRoomId)
				return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas sudah dihapus")
			}
			if room.MasjidID != masjidID {
				log.Printf("[CAS:create] room tenant mismatch: room.msj=%s req.msj=%s", room.MasjidID, masjidID)
				return fiber.NewError(fiber.StatusForbidden, "Ruang bukan milik masjid Anda")
			}
			log.Printf("[CAS:create] room validated: room_id=%s masjid_id=%s", *req.ClassAttendanceSessionClassRoomId, room.MasjidID)
		}

		// 4) Cek duplikasi aktif (mengikuti unique index: masjid, section, class_subject, date, alive)
		//     Jika date nil → gunakan "efektif" hari ini (midnight local) hanya untuk pengecekan dini
		effDate := func() time.Time {
			if req.ClassAttendanceSessionDate != nil {
				return *req.ClassAttendanceSessionDate
			}
			now := time.Now().In(time.Local)
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		}()

		var dupeCount int64
		if err := tx.Table("class_attendance_sessions").
			Where(`
				class_attendance_sessions_masjid_id = ?
				AND class_attendance_sessions_section_id = ?
				AND class_attendance_sessions_class_subject_id = ?
				AND class_attendance_sessions_date = ?
				AND class_attendance_sessions_deleted_at IS NULL
			`,
				req.ClassAttendanceSessionMasjidId,
				req.ClassAttendanceSessionSectionId,
				req.ClassAttendanceSessionClassSubjectId,
				effDate,
			).
			Count(&dupeCount).Error; err != nil {
			log.Printf("[CAS:create] dupe check error: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if dupeCount > 0 {
			log.Printf("[CAS:create] duplicate found (masjid=%s, section=%s, cs=%s, date=%s)",
				req.ClassAttendanceSessionMasjidId,
				req.ClassAttendanceSessionSectionId,
				req.ClassAttendanceSessionClassSubjectId,
				effDate.Format("2006-01-02"),
			)
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// 5) Simpan
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			low := strings.ToLower(err.Error())
			log.Printf("[CAS:create] insert error: %v", err)
			if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
			}
			if strings.Contains(low, "mismatch") || strings.Contains(low, "invalid") {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kehadiran")
		}

		c.Locals("created_model", m)
		log.Printf("[CAS:create] created ok id=%s", m.ClassAttendanceSessionId)
		return nil
	}); err != nil {
		return err // sudah dilog di atas
	}

	// ===== Response
	m := c.Locals("created_model").(attendanceModel.ClassAttendanceSessionModel)
	c.Set("Location", fmt.Sprintf("/admin/class-attendance-sessions/%s", m.ClassAttendanceSessionId.String()))
	return helper.JsonCreated(c, "Sesi kehadiran berhasil dibuat", attendanceDTO.FromClassAttendanceSessionModel(m))
}


// PUT /admin/class-attendance-sessions/:id
// tambahkan di import kalau belum:
// import (
//   "errors"
//   "strings"
//   "time"
//   "github.com/gofiber/fiber/v2"
//   "github.com/google/uuid"
//   "gorm.io/gorm"
// )

// PUT /admin/class-attendance-sessions/:id
func (ctrl *ClassAttendanceSessionController) UpdateClassAttendanceSession(c *fiber.Ctx) error {
	// ===== Role & Tenant (admin ATAU teacher) =====
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)

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

		// 2) Validasi section (bila diganti) & ambil class_id section
		targetSectionID := existing.ClassAttendanceSessionSectionId
		var sectionClassID *uuid.UUID
		if req.ClassAttendanceSessionSectionId != nil {
			targetSectionID = *req.ClassAttendanceSessionSectionId
		}
		{
			var sec struct {
				MasjidID *uuid.UUID `gorm:"column:class_sections_masjid_id"`
				ClassID  *uuid.UUID `gorm:"column:class_sections_class_id"`
			}
			if err := tx.Table("class_sections").
				Select("class_sections_masjid_id, class_sections_class_id").
				Where("class_sections_id = ? AND class_sections_deleted_at IS NULL", targetSectionID).
				Take(&sec).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil section")
			}
			if sec.MasjidID == nil || *sec.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
			}
			sectionClassID = sec.ClassID
		}

		// 3) Validasi class_subject (bila diganti) & cocokkan class_id
		targetClassSubjectID := existing.ClassAttendanceSessionClassSubjectId
		if req.ClassAttendanceSessionClassSubjectId != nil {
			targetClassSubjectID = *req.ClassAttendanceSessionClassSubjectId
		}
		var csClassID *uuid.UUID
		{
			var cs struct {
				MasjidID uuid.UUID  `gorm:"column:class_subjects_masjid_id"`
				ClassID  *uuid.UUID `gorm:"column:class_subjects_class_id"`
			}
			if err := tx.Table("class_subjects").
				Select("class_subjects_masjid_id, class_subjects_class_id").
				Where("class_subjects_id = ? AND class_subjects_deleted_at IS NULL", targetClassSubjectID).
				Take(&cs).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Class subject tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil class subject")
			}
			if cs.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Class subject bukan milik masjid Anda")
			}
			csClassID = cs.ClassID
		}
		if sectionClassID == nil || csClassID == nil || *sectionClassID != *csClassID {
			return fiber.NewError(fiber.StatusBadRequest, "Class subject tidak sesuai dengan class pada section")
		}

		// 4) Validasi TEACHER (bila diubah) → harus milik masjid yang sama
		if req.ClassAttendanceSessionTeacherId != nil {
			var mtMasjid uuid.UUID
			if err := tx.Table("masjid_teachers").
				Select("masjid_teacher_masjid_id").
				Where("masjid_teacher_id = ? AND masjid_teacher_deleted_at IS NULL", *req.ClassAttendanceSessionTeacherId).
				Take(&mtMasjid).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Guru (masjid_teacher) tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi guru")
			}
			if mtMasjid != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik masjid Anda")
			}
		}

		// 4b) Validasi ROOM (bila diubah) → harus alive & se-tenant
		if req.ClassAttendanceSessionClassRoomId != nil {
			var room struct {
				MasjidID  uuid.UUID  `gorm:"column:class_rooms_masjid_id"`
				DeletedAt *time.Time `gorm:"column:deleted_at"`
			}
			if err := tx.Table("class_rooms").
				Select("class_rooms_masjid_id, deleted_at").
				Where("class_room_id = ?", *req.ClassAttendanceSessionClassRoomId).
				Take(&room).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang")
			}
			if room.DeletedAt != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas sudah dihapus")
			}
			if room.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Ruang bukan milik masjid Anda")
			}
		}

		// 5) Cek unik baru: (masjid, section, class_subject, date) excluding current
		var targetDate time.Time
		if req.ClassAttendanceSessionDate != nil {
			targetDate = *req.ClassAttendanceSessionDate
		} else if existing.ClassAttendanceSessionDate != nil {
			targetDate = *existing.ClassAttendanceSessionDate
		} else {
			// fallback defensif; idealnya existing tidak pernah nil
			now := time.Now().In(time.Local)
			targetDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		}

		var cnt int64
		if err := tx.Table("class_attendance_sessions").
			Where(`
				class_attendance_sessions_masjid_id = ?
				AND class_attendance_sessions_section_id = ?
				AND class_attendance_sessions_class_subject_id = ?
				AND class_attendance_sessions_date = ?
				AND class_attendance_sessions_id <> ?
				AND class_attendance_sessions_deleted_at IS NULL
			`,
				masjidID, targetSectionID, targetClassSubjectID, targetDate, existing.ClassAttendanceSessionId,
			).Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// 6) Terapkan perubahan
		patch := map[string]interface{}{
			"class_attendance_sessions_masjid_id": masjidID, // enforce tenant
		}
		if req.ClassAttendanceSessionSectionId != nil {
			patch["class_attendance_sessions_section_id"] = *req.ClassAttendanceSessionSectionId
			existing.ClassAttendanceSessionSectionId = *req.ClassAttendanceSessionSectionId
		}
		if req.ClassAttendanceSessionDate != nil {
			patch["class_attendance_sessions_date"] = *req.ClassAttendanceSessionDate
			existing.ClassAttendanceSessionDate = req.ClassAttendanceSessionDate
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
		if req.ClassAttendanceSessionClassSubjectId != nil {
			patch["class_attendance_sessions_class_subject_id"] = *req.ClassAttendanceSessionClassSubjectId
			existing.ClassAttendanceSessionClassSubjectId = *req.ClassAttendanceSessionClassSubjectId
		}
		if req.ClassAttendanceSessionTeacherId != nil {
			patch["class_attendance_sessions_teacher_id"] = req.ClassAttendanceSessionTeacherId
			existing.ClassAttendanceSessionTeacherId = req.ClassAttendanceSessionTeacherId
		}
		if req.ClassAttendanceSessionClassRoomId != nil {
			patch["class_attendance_sessions_class_room_id"] = req.ClassAttendanceSessionClassRoomId
			existing.ClassAttendanceSessionClassRoomId = req.ClassAttendanceSessionClassRoomId
		}

		if err := tx.Model(&attendanceModel.ClassAttendanceSessionModel{}).
			Where("class_attendance_sessions_id = ?", existing.ClassAttendanceSessionId).
			Select([]string{
				"class_attendance_sessions_section_id",
				"class_attendance_sessions_masjid_id",
				"class_attendance_sessions_date",
				"class_attendance_sessions_title",
				"class_attendance_sessions_general_info",
				"class_attendance_sessions_note",
				"class_attendance_sessions_class_subject_id",
				"class_attendance_sessions_teacher_id",
				"class_attendance_sessions_class_room_id",
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
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
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
			delErr = tx.Unscoped().
				Delete(&attendanceModel.ClassAttendanceSessionModel{}, "class_attendance_sessions_id = ?", sessionID).Error
		} else {
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
