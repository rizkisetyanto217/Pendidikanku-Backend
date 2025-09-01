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

	attendanceDTO "masjidku_backend/internals/features/school/attendance_assesment/attendance_sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/attendance_assesment/attendance_sessions/model"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"

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
        // select tanpa CSST; sertakan teacher_id agar ikut di-serialize
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
        // secondary order untuk stabilitas
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

    // ===== Base query:
    // join ke masjid_teachers agar memastikan CAS.teacher_id milik user & masjid yang sama
    qBase := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
        Joins(`
            JOIN masjid_teachers mt
              ON mt.masjid_teacher_id = class_attendance_sessions_teacher_id
        `).
        Where(`
            class_attendance_sessions_masjid_id = ?
            AND mt.masjid_teachers_user_id = ?
            AND mt.masjid_teachers_masjid_id = ?
        `, masjidID, userID, masjidID)

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

    if df == nil && dt == nil {
        // default: hari ini (00:00) sampai besok (00:00) → mencakup hari ini
        now := time.Now().In(time.Local)
        today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
        tomorrow := today.Add(24 * time.Hour)
        qBase = qBase.Where("class_attendance_sessions_date >= ? AND class_attendance_sessions_date < ?", today, tomorrow)
    } else {
        if df != nil && dt != nil {
            qBase = qBase.Where("class_attendance_sessions_date BETWEEN ? AND ?", *df, *dt)
        } else if df != nil {
            qBase = qBase.Where("class_attendance_sessions_date >= ?", *df)
        } else if dt != nil {
            qBase = qBase.Where("class_attendance_sessions_date <= ?", *dt)
        }
    }

    // Filter opsional: section_id
    if s := strings.TrimSpace(c.Query("section_id")); s != "" {
        sid, e := uuid.Parse(s)
        if e != nil {
            return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
        }
        qBase = qBase.Where("class_attendance_sessions_section_id = ?", sid)
    }

    // ===== Total
    // Aman dihitung langsung (join 1:1), atau pakai subquery kalau mau benar2 pasti
    var total int64
    if err := qBase.Session(&gorm.Session{}).Count(&total).Error; err != nil {
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

    // ===== Pagination & sorting (helper) =====
    rawQ := string(c.Request().URI().QueryString())
    httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
    p := helper.ParseWith(httpReq, "date", "desc", helper.AdminOpts)

    allowedSort := map[string]string{
        "date":       "cas.class_attendance_sessions_date",
        "created_at": "cas.class_attendance_sessions_created_at",
        "title":      "cas.class_attendance_sessions_title",
    }
    orderClause, err := p.SafeOrderClause(allowedSort, "date")
    if err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
    }
    orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

    // ===== Filter tanggal (default: today jika kosong)
    parseYmd := func(s string) (*time.Time, error) {
        s = strings.TrimSpace(s)
        if s == "" { return nil, nil }
        t, err := time.Parse("2006-01-02", s)
        if err != nil { return nil, err }
        tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
        return &tt, nil
    }
    dfStr := strings.TrimSpace(c.Query("date_from"))
    dtStr := strings.TrimSpace(c.Query("date_to"))
    if dfStr == "" && dtStr == "" {
        now := time.Now().In(time.Local)
        dfStr = now.Format("2006-01-02")
    }
    df, err := parseYmd(dfStr); if err != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
    dt, err := parseYmd(dtStr); if err != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
    log.Printf("[CAS][%s] date_from=%q date_to=%q", reqID, dfStr, dtStr)

    // ===== Base query (pakai alias) =====
    db := ctrl.DB
    qBase := db.Table("class_attendance_sessions AS cas").
        Where("cas.class_attendance_sessions_masjid_id = ?", targetMasjidID)

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
        id, e := uuid.Parse(s); if e != nil {
            log.Printf("[CAS][%s] BAD section_id=%q err=%v", reqID, s, e)
            return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
        }
        qBase = qBase.Where("cas.class_attendance_sessions_section_id = ?", id)
    }
    if s := strings.TrimSpace(c.Query("class_subject_id")); s != "" {
        id, e := uuid.Parse(s); if e != nil {
            log.Printf("[CAS][%s] BAD class_subject_id=%q err=%v", reqID, s, e)
            return fiber.NewError(fiber.StatusBadRequest, "class_subject_id tidak valid")
        }
        qBase = qBase.Where("cas.class_attendance_sessions_class_subject_id = ?", id)
    }
    // filter berdasarkan masjid_teacher_id langsung di CAS
    if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
        id, e := uuid.Parse(s); if e != nil {
            log.Printf("[CAS][%s] BAD teacher_id=%q err=%v", reqID, s, e)
            return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
        }
        qBase = qBase.Where("cas.class_attendance_sessions_teacher_id = ?", id)
    }
    // legacy: filter berdasarkan users.id guru via masjid_teachers
    var teacherUserID *uuid.UUID
    if s := strings.TrimSpace(c.Query("teacher_user_id")); s != "" {
        id, e := uuid.Parse(s); if e != nil {
            log.Printf("[CAS][%s] BAD teacher_user_id=%q err=%v", reqID, s, e)
            return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid")
        }
        teacherUserID = &id
    }
    log.Printf("[CAS][%s] filters: section_id=%q class_subject_id=%q teacher_id=%q teacher_user_id=%v",
        reqID, c.Query("section_id"), c.Query("class_subject_id"), c.Query("teacher_id"), teacherUserID)

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
    qBase = qBase.Where(`(mt.masjid_teacher_id IS NULL OR mt.masjid_teachers_masjid_id = ?)`, targetMasjidID)
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
        CreatedAt time.Time  `gorm:"column:class_attendance_sessions_created_at"`
        UpdatedAt *time.Time `gorm:"column:class_attendance_sessions_updated_at"`
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
            cas.class_attendance_sessions_created_at,
            cas.class_attendance_sessions_updated_at,
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
        Order("cas.class_attendance_sessions_created_at DESC").
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
            ClassAttendanceSessionClassSubjectId: r.SubjectID,   // *uuid.UUID (DTO pointer)

            ClassAttendanceSessionTeacherId:      r.TeacherId,   // *uuid.UUID

            ClassAttendanceSessionCreatedAt: r.CreatedAt,
            ClassAttendanceSessionUpdatedAt: r.UpdatedAt,
            ClassAttendanceSessionDeletedAt: r.DeletedAt,

            // section info
            ClassSectionSlug:     r.SectionSlug,
            ClassSectionName:     r.SectionName,
            ClassSectionCode:     r.SectionCode,
            ClassSectionCapacity: r.SectionCapacity,
            ClassSectionSchedule: datatypes.JSON(r.SectionSchedule), // cast []byte -> JSON
        }

        // teacher cosmetic info (opsional dari join users)
        resp.ClassAttendanceSessionTeacherName  = r.TeacherName
        resp.ClassAttendanceSessionTeacherEmail = r.TeacherEmail

        items = append(items, resp)
    }

    meta := helper.BuildMeta(total, p)
    log.Printf("[CAS][%s] DONE → return items=%d total=%d meta=%+v df=%q dt=%q",
        reqID, len(items), total, meta, dfStr, dtStr)

    return helper.JsonList(c, items, meta)
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

	// Force tenant & normalisasi tanggal (kolom DATE disimpan midnight lokal)
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

	// Validasi payload (DTO: class_subject_id sudah required)
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Transaksi kecil
	err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Validasi SECTION milik masjid
		var sec struct {
			ID       uuid.UUID  `gorm:"column:class_sections_id"`
			MasjidID *uuid.UUID `gorm:"column:class_sections_masjid_id"`
			ClassID  *uuid.UUID `gorm:"column:class_sections_class_id"`
		}
		if err := tx.Table("class_sections").
			Select("class_sections_id, class_sections_masjid_id, class_sections_class_id").
			Where("class_sections_id = ? AND class_sections_deleted_at IS NULL", req.ClassAttendanceSessionSectionId).
			Take(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil section")
		}
		if sec.MasjidID == nil || *sec.MasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
		}

		// 2) Validasi CLASS_SUBJECT (milik masjid & class_id cocok dengan section)
		var cs struct {
			ID       uuid.UUID  `gorm:"column:class_subjects_id"`
			MasjidID uuid.UUID  `gorm:"column:class_subjects_masjid_id"`
			ClassID  *uuid.UUID `gorm:"column:class_subjects_class_id"`
		}
		if err := tx.Table("class_subjects").
			Select("class_subjects_id, class_subjects_masjid_id, class_subjects_class_id").
			Where("class_subjects_id = ? AND class_subjects_deleted_at IS NULL", req.ClassAttendanceSessionClassSubjectId).
			Take(&cs).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Class subject tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil class subject")
		}
		if cs.MasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Class subject bukan milik masjid Anda")
		}
		if sec.ClassID == nil || cs.ClassID == nil || *sec.ClassID != *cs.ClassID {
			return fiber.NewError(fiber.StatusBadRequest, "Class subject tidak sesuai dengan class pada section")
		}

		// 3) Validasi TEACHER (opsional) → harus milik masjid yang sama
		if req.ClassAttendanceSessionTeacherId != nil {
			var mtMasjid uuid.UUID
			if err := tx.Table("masjid_teachers").
				Select("masjid_teachers_masjid_id").
				Where("masjid_teacher_id = ?", *req.ClassAttendanceSessionTeacherId).
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

		// 4) Cek duplikasi aktif: (masjid, section, class_subject, date)
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
				req.ClassAttendanceSessionDate,
			).Count(&dupeCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if dupeCount > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// 5) Simpan
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
				Select("masjid_teachers_masjid_id").
				Where("masjid_teacher_id = ?", *req.ClassAttendanceSessionTeacherId).
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

		// 5) Cek unik baru: (masjid, section, class_subject, date) excluding current
		targetDate := existing.ClassAttendanceSessionDate
		if req.ClassAttendanceSessionDate != nil {
			targetDate = *req.ClassAttendanceSessionDate
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
		if req.ClassAttendanceSessionClassSubjectId != nil {
			patch["class_attendance_sessions_class_subject_id"] = *req.ClassAttendanceSessionClassSubjectId
			existing.ClassAttendanceSessionClassSubjectId = *req.ClassAttendanceSessionClassSubjectId
		}
		if req.ClassAttendanceSessionTeacherId != nil {
			patch["class_attendance_sessions_teacher_id"] = req.ClassAttendanceSessionTeacherId
			existing.ClassAttendanceSessionTeacherId = req.ClassAttendanceSessionTeacherId
		}

		now := time.Now().In(time.Local)
		patch["class_attendance_sessions_updated_at"] = now
		existing.ClassAttendanceSessionMasjidId = masjidID
		existing.ClassAttendanceSessionUpdatedAt = now

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
