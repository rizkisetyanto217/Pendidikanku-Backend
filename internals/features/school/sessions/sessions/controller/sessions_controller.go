// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	attendanceDTO "masjidku_backend/internals/features/school/sessions/sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/sessions/sessions/model"

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

/* ========== small helpers ========== */

func parseYMDLocal(s string) (*time.Time, error) {
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

// ambil (section_id, class_subject_id) dari CSST
func (ctrl *ClassAttendanceSessionController) csstSectionAndSubject(csstID uuid.UUID) (sectionID uuid.UUID, subjectID uuid.UUID, err error) {
	var row struct {
		SectionID uuid.UUID `gorm:"column:section_id"`
		SubjectID uuid.UUID `gorm:"column:subject_id"`
	}
	err = ctrl.DB.Table("class_section_subject_teachers").
		Select(`
			class_section_subject_teachers_section_id     AS section_id,
			class_section_subject_teachers_class_subjects_id AS subject_id
		`).
		Where("class_section_subject_teachers_id = ? AND class_section_subject_teachers_deleted_at IS NULL", csstID).
		Take(&row).Error
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return row.SectionID, row.SubjectID, nil
}



/* =========================================================
   GET /admin/class-attendance-sessions/section/:section_id
   ?date_from=&date_to=&csst_id=&limit=&offset=&sort_by=&sort=
   (filter by CSST.section_id)
========================================================= */
// file: internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go

// ... (imports tetap) ...

func (ctrl *ClassAttendanceSessionController) ListBySection(c *fiber.Ctx) error {
	// ======= existing guard & scope (tetap, tidak diubah) =======
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	userID, _ := helperAuth.GetUserIDFromToken(c)
	isAdmin := func() bool {
		if mid, err := helperAuth.GetMasjidIDFromToken(c); err == nil && mid == masjidID {
			return true
		}
		return false
	}()
	isTeacher := func() bool {
		if mid, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && mid == masjidID {
			return true
		}
		return false
	}()

	secID, err := uuid.Parse(strings.TrimSpace(c.Params("section_id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
	}

	// validate section tenant
	var sec secModel.ClassSectionModel
	if err := ctrl.DB.
		Select("class_sections_id, class_sections_masjid_id").
		First(&sec, "class_sections_id = ? AND class_sections_deleted_at IS NULL", secID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
	}
	if sec.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
	}

	// guard siswa/ortu
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

	// ======= pagination & sort (tetap) =======
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

	// ======= date filters (inclusive) =======
	df, err := parseYMDLocal(c.Query("date_from"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYMDLocal(c.Query("date_to"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}

	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Joins(`
			JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_id = cas.class_attendance_sessions_csst_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`).
		Where(`
			cas.class_attendance_sessions_masjid_id = ?
			AND cas.class_attendance_sessions_deleted_at IS NULL
			AND csst.class_section_subject_teachers_section_id = ?
		`, masjidID, secID)

	if df != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ?", *df)
	}
	if dt != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date <= ?", *dt)
	}

	// Optional: csst_id
	if s := strings.TrimSpace(c.Query("csst_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "csst_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_csst_id = ?", id)
	}

	// ======= total =======
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_sessions_id").
		Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ======= data utama =======
	type row struct {
		ID        uuid.UUID  `gorm:"column:id"`
		MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
		Date      time.Time  `gorm:"column:date"`
		Title     *string    `gorm:"column:title"`
		General   string     `gorm:"column:general"`
		Note      *string    `gorm:"column:note"`
		TeacherID *uuid.UUID `gorm:"column:teacher_id"`
		RoomID    *uuid.UUID `gorm:"column:room_id"`
		CSSTID    uuid.UUID  `gorm:"column:csst_id"`
		SubjectID uuid.UUID  `gorm:"column:subject_id"`
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
			cas.class_attendance_sessions_csst_id    AS csst_id,
			csst.class_section_subject_teachers_class_subjects_id AS subject_id
		`).
		Order(orderExpr).
		Order("cas.class_attendance_sessions_date DESC, cas.class_attendance_sessions_id DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ======= mapping ke response =======
	out := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	sessionIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		sessionIDs = append(sessionIDs, r.ID)
		subj := r.SubjectID
		out = append(out, attendanceDTO.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:          r.ID,
			ClassAttendanceSessionMasjidId:    r.MasjidID,
			ClassAttendanceSessionCSSTId:      r.CSSTID,
			ClassAttendanceSessionClassRoomId: r.RoomID,
			ClassAttendanceSessionDate:        r.Date,
			ClassAttendanceSessionTitle:       r.Title,
			ClassAttendanceSessionGeneralInfo: r.General,
			ClassAttendanceSessionNote:        r.Note,
			ClassAttendanceSessionTeacherId:   r.TeacherID,

			// enrichment
			ClassSectionId: &secID,
			ClassSubjectId: &subj,
		})
	}

	// ======= OPSI: include URLs =======
	includeURLs := strings.EqualFold(strings.TrimSpace(c.Query("include_urls")), "1") ||
		strings.EqualFold(strings.TrimSpace(c.Query("include_urls")), "true")

	if includeURLs && len(sessionIDs) > 0 {
		// Optional filters untuk URL
		urlLabel := strings.TrimSpace(c.Query("url_label")) // exact match label (optional)
		urlLike := strings.TrimSpace(c.Query("url_like"))   // LIKE (substring) pada href (optional)
		
		type urlRow struct {
			ID           uuid.UUID  `gorm:"column:id"`
			MasjidID     uuid.UUID  `gorm:"column:mid"`
			SessionID    uuid.UUID  `gorm:"column:sid"`
			Label        *string    `gorm:"column:label"`
			Href         string     `gorm:"column:href"`
			TrashURL     *string    `gorm:"column:trash_url"`
			DeleteUntil  *time.Time `gorm:"column:delete_until"`
			CreatedAt    time.Time  `gorm:"column:created_at"`
		}
		var urlRows []urlRow

		uq := ctrl.DB.Table("class_attendance_session_url AS u").
			Where("u.class_attendance_session_url_deleted_at IS NULL").
			Where("u.class_attendance_session_url_masjid_id = ?", masjidID).
			Where("u.class_attendance_session_url_session_id IN ?", sessionIDs)

		if urlLabel != "" {
			uq = uq.Where("u.class_attendance_session_url_label = ?", urlLabel)
		}
		if urlLike != "" {
			uq = uq.Where("LOWER(u.class_attendance_session_url_href) LIKE ?", "%"+strings.ToLower(urlLike)+"%")
		}

		if err := uq.
			Select(`
				u.class_attendance_session_url_id                  AS id,
				u.class_attendance_session_url_masjid_id           AS mid,
				u.class_attendance_session_url_session_id          AS sid,
				u.class_attendance_session_url_label               AS label,
				u.class_attendance_session_url_href                AS href,
				u.class_attendance_session_url_trash_url           AS trash_url,
				u.class_attendance_session_url_delete_pending_until AS delete_until,
				u.class_attendance_session_url_created_at          AS created_at
			`).
			Order("u.class_attendance_session_url_created_at DESC, u.class_attendance_session_url_id DESC").
			Find(&urlRows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil URLs")
		}

		// Grouping by session_id
		urlMap := make(map[uuid.UUID][]attendanceDTO.ClassAttendanceSessionURL, len(sessionIDs))
		for _, u := range urlRows {
			urlMap[u.SessionID] = append(urlMap[u.SessionID], attendanceDTO.ClassAttendanceSessionURL{
				ClassAttendanceSessionURLId:                 u.ID,
				ClassAttendanceSessionURLMasjidId:           u.MasjidID,
				ClassAttendanceSessionURLSessionId:          u.SessionID,
				ClassAttendanceSessionURLLabel:              u.Label,
				ClassAttendanceSessionURLHref:               u.Href,
				ClassAttendanceSessionURLTrashURL:           u.TrashURL,
				ClassAttendanceSessionURLDeletePendingUntil: u.DeleteUntil,
				ClassAttendanceSessionURLCreatedAt:          u.CreatedAt,
			})
		}

		// Sematkan ke out (respect urlLimitPerSession jika > 0)
		// Sematkan SEMUA URLs ke masing-masing sesi
		for i := range out {
			list := urlMap[out[i].ClassAttendanceSessionId]
			if len(list) > 0 { // biarkan kosong ter-omit karena `omitempty`
				out[i].ClassAttendanceSessionUrls = list
			}
		}

	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}


/* =========================================================
   LIST by TEACHER (SELF)
   GET /api/u/sessions/teacher/me?section_id=&csst_id=&date_from=&date_to=&limit=&offset=&q=
========================================================= */
func (ctrl *ClassAttendanceSessionController) ListMyTeachingSessions(c *fiber.Ctx) error {
	// Wajib token teacher
	masjidID, err := helperAuth.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_ids tidak ditemukan di token")
	}
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
	}

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

	// Rentang tanggal (inklusif; upper bound eksklusif +24h)
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
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_id = cas.class_attendance_sessions_csst_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
			 AND csst.class_section_subject_teachers_is_active
		`).
		Joins(`
			LEFT JOIN masjid_teachers AS mt2
			  ON mt2.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id
			 AND mt2.masjid_teacher_deleted_at IS NULL
		`).
		Where(`
			cas.class_attendance_sessions_masjid_id = ?
			AND cas.class_attendance_sessions_deleted_at IS NULL
			AND (
			     mt.masjid_teacher_user_id = ?
			  OR mt2.masjid_teacher_user_id = ?
			)
		`, masjidID, userID, userID)

	// Filter tanggal opsional
	if lo != nil && hi != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ? AND cas.class_attendance_sessions_date < ?", *lo, *hi)
	} else if lo != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ?", *lo)
	} else if hi != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date < ?", *hi)
	}

	// Opsional: section_id (via CSST)
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("csst.class_section_subject_teachers_section_id = ?", id)
	}

	// Opsional: csst_id
	if s := strings.TrimSpace(c.Query("csst_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "csst_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_csst_id = ?", id)
	}

	// Keyword
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		pat := "%" + q + "%"
		qBase = qBase.Where(`(cas.class_attendance_sessions_title ILIKE ? OR cas.class_attendance_sessions_general_info ILIKE ?)`, pat, pat)
	}

	// Total distinct
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_sessions_id").
		Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// Data
	type row struct {
		ID        uuid.UUID  `gorm:"column:id"`
		MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
		Date      time.Time  `gorm:"column:date"`
		Title     *string    `gorm:"column:title"`
		General   string     `gorm:"column:general"`
		Note      *string    `gorm:"column:note"`
		CSSTID    uuid.UUID  `gorm:"column:csst_id"`
		TeacherID *uuid.UUID `gorm:"column:teacher_id"`
		RoomID    *uuid.UUID `gorm:"column:room_id"`
		SectionID uuid.UUID  `gorm:"column:section_id"`
		SubjectID uuid.UUID  `gorm:"column:subject_id"`
		DeletedAt *time.Time `gorm:"column:deleted_at"`
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
			cas.class_attendance_sessions_csst_id    AS csst_id,
			cas.class_attendance_sessions_teacher_id AS teacher_id,
			cas.class_attendance_sessions_class_room_id AS room_id,
			cas.class_attendance_sessions_deleted_at AS deleted_at,
			csst.class_section_subject_teachers_section_id AS section_id,
			csst.class_section_subject_teachers_class_subjects_id AS subject_id
		`).
		Order(orderExpr).
		Order("cas.class_attendance_sessions_date DESC, cas.class_attendance_sessions_id DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		subj := r.SubjectID
		resp = append(resp, attendanceDTO.ClassAttendanceSessionResponse{
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

			// enrichment sesuai DTO
			ClassSectionId: &r.SectionID,
			ClassSubjectId: &subj,
		})
	}


	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, resp, meta)
}

/* =========================================================
   GET /admin/class-attendance-sessions/masjid
   ?date_from=&date_to=&section_id=&class_subject_id=&room_id=&teacher_id=&teacher_user_id=&csst_id=&limit=&offset=&sort_by=&order=
   (section/subject via CSST)
========================================================= */
func (ctrl *ClassAttendanceSessionController) ListByMasjid(c *fiber.Ctx) error {

	// Guard & tenant resolve
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil
	isTeacher := teacherMasjidID != uuid.Nil

	if !isAdmin && !isTeacher {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// target masjid
	targetMasjidID := teacherMasjidID
	if isAdmin {
		targetMasjidID = adminMasjidID
		if s := strings.TrimSpace(c.Query("masjid_id")); s != "" {
			qid, err := uuid.Parse(s)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak valid")
			}
			if qid != adminMasjidID {
				return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses masjid lain")
			}
			targetMasjidID = qid
		}
	}

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

	// Date filters (default today if both empty)
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

	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Where("cas.class_attendance_sessions_masjid_id = ?", targetMasjidID).
		Where("cas.class_attendance_sessions_deleted_at IS NULL").
		Joins(`
            LEFT JOIN masjid_teachers AS mt
              ON mt.masjid_teacher_id = cas.class_attendance_sessions_teacher_id
        `).
		Joins(`
            LEFT JOIN users AS u
              ON u.id = mt.masjid_teacher_user_id
        `).
		// join CSST untuk ambil section & subject
		Joins(`
            LEFT JOIN class_section_subject_teachers AS csst
              ON csst.class_section_subject_teachers_id = cas.class_attendance_sessions_csst_id
             AND csst.class_section_subject_teachers_deleted_at IS NULL
        `).
		// join CLASS SECTIONS via CSST.section_id
		Joins(`
            LEFT JOIN class_sections AS cs
              ON cs.class_sections_id = csst.class_section_subject_teachers_section_id
        `)

	// date range
	if df != nil && dt != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date BETWEEN ? AND ?", *df, *dt)
	} else if df != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ?", *df)
	} else if dt != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date <= ?", *dt)
	}

	// filters
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("csst.class_section_subject_teachers_section_id = ?", id)
	}
	if s := strings.TrimSpace(c.Query("class_subject_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "class_subject_id tidak valid")
		}
		qBase = qBase.Where("csst.class_section_subject_teachers_class_subjects_id = ?", id)
	}
	if s := strings.TrimSpace(c.Query("csst_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "csst_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_csst_id = ?", id)
	}
	if s := strings.TrimSpace(c.Query("room_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "room_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_class_room_id = ?", id)
	}
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_teacher_id = ?", id)
	}
	var teacherUserID *uuid.UUID
	if s := strings.TrimSpace(c.Query("teacher_user_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid")
		}
		teacherUserID = &id
		qBase = qBase.Where("mt.masjid_teacher_user_id = ?", *teacherUserID)
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		pat := "%" + q + "%"
		qBase = qBase.Where(`(cas.class_attendance_sessions_title ILIKE ? OR cas.class_attendance_sessions_general_info ILIKE ?)`, pat, pat)
	}

	// Tenant-guard untuk mt (jaga NULL)
	qBase = qBase.Where(`(mt.masjid_teacher_id IS NULL OR mt.masjid_teacher_masjid_id = ?)`, targetMasjidID)

	// total
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_sessions_id").
		Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// fetch
	type row struct {
		ID        uuid.UUID  `gorm:"column:id"`
		MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
		Date      time.Time  `gorm:"column:date"`
		Title     *string    `gorm:"column:title"`
		General   string     `gorm:"column:general"`
		Note      *string    `gorm:"column:note"`
		CSSTID    uuid.UUID  `gorm:"column:csst_id"`
		TeacherId *uuid.UUID `gorm:"column:teacher_id"`
		RoomId    *uuid.UUID `gorm:"column:room_id"`
		DeletedAt *time.Time `gorm:"column:deleted_at"`

		// USERS
		TeacherName  *string `gorm:"column:teacher_name"`
		TeacherEmail *string `gorm:"column:teacher_email"`

		// CLASS SECTIONS (via CSST)
		SectionID       uuid.UUID `gorm:"column:section_id"`
		SubjectID       uuid.UUID `gorm:"column:subject_id"`
		SectionSlug     *string   `gorm:"column:section_slug"`
		SectionName     *string   `gorm:"column:section_name"`
		SectionCode     *string   `gorm:"column:section_code"`
		SectionCapacity *int      `gorm:"column:section_capacity"`
		SectionSchedule []byte    `gorm:"column:section_schedule"` // JSONB
	}
	var rows []row
	if err := qBase.Select(`
            cas.class_attendance_sessions_id         AS id,
            cas.class_attendance_sessions_masjid_id  AS masjid_id,
            cas.class_attendance_sessions_date       AS date,
            cas.class_attendance_sessions_title      AS title,
            cas.class_attendance_sessions_general_info AS general,
            cas.class_attendance_sessions_note       AS note,
            cas.class_attendance_sessions_csst_id    AS csst_id,
            cas.class_attendance_sessions_teacher_id AS teacher_id,
            cas.class_attendance_sessions_class_room_id AS room_id,
            cas.class_attendance_sessions_deleted_at AS deleted_at,

            u.user_name AS teacher_name,
            u.email     AS teacher_email,

            csst.class_section_subject_teachers_section_id       AS section_id,
            csst.class_section_subject_teachers_class_subjects_id AS subject_id,
            cs.class_sections_slug     AS section_slug,
            cs.class_sections_name     AS section_name,
            cs.class_sections_code     AS section_code,
            cs.class_sections_capacity AS section_capacity,
            cs.class_sections_schedule AS section_schedule
        `).
		Order(orderExpr).
		Order("cas.class_attendance_sessions_date DESC, cas.class_attendance_sessions_id DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		subj := r.SubjectID
		resp := attendanceDTO.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:          r.ID,
			ClassAttendanceSessionMasjidId:    r.MasjidID,
			ClassAttendanceSessionDate:        r.Date,
			ClassAttendanceSessionTitle:       r.Title,
			ClassAttendanceSessionGeneralInfo: r.General,
			ClassAttendanceSessionNote:        r.Note,
			ClassAttendanceSessionCSSTId:      r.CSSTID,
			ClassAttendanceSessionClassRoomId: r.RoomId,
			ClassAttendanceSessionTeacherId:   r.TeacherId,

			// enrichment sesuai DTO
			ClassSectionId:       &r.SectionID,
			ClassSubjectId:       &subj,
			ClassSectionSlug:     r.SectionSlug,
			ClassSectionName:     r.SectionName,
			ClassSectionCode:     r.SectionCode,
			ClassSectionCapacity: r.SectionCapacity,
			ClassSectionSchedule: datatypes.JSON(r.SectionSchedule),

			ClassAttendanceSessionDeletedAt: r.DeletedAt,
		}
		resp.ClassAttendanceSessionTeacherName = r.TeacherName
		resp.ClassAttendanceSessionTeacherEmail = r.TeacherEmail
		items = append(items, resp)
	}


	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}

/* =========================================================
   POST /admin/class-attendance-sessions
   Body: CreateClassAttendanceSessionRequest
   (CSST wajib; section/subject diturunkan dari CSST)
========================================================= */
func (ctrl *ClassAttendanceSessionController) CreateClassAttendanceSession(c *fiber.Ctx) error {
	// Tenant & Role Guard (admin ATAU teacher)
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)
	userID, _ := helperAuth.GetUserIDFromToken(c)

	// Parse payload
	var req attendanceDTO.CreateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant & normalisasi tanggal
	req.ClassAttendanceSessionMasjidId = masjidID
	if req.ClassAttendanceSessionDate != nil {
		d := req.ClassAttendanceSessionDate.In(time.Local)
		dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
		req.ClassAttendanceSessionDate = &dd
	}

	// Trim
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

	// Transaksi
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Validasi CSST (wajib) & sinkron teacher opsional
		if req.ClassAttendanceSessionCSSTId == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "class_attendance_session_csst_id wajib diisi")
		}
		var a struct {
			MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
			SectionID uuid.UUID  `gorm:"column:section_id"`
			CSID      uuid.UUID  `gorm:"column:cs_id"`
			TeacherID *uuid.UUID `gorm:"column:teacher_id"`
			IsActive  bool       `gorm:"column:is_active"`
			DeletedAt *time.Time `gorm:"column:deleted_at"`
		}
		if err := tx.Table("class_section_subject_teachers").
			Select(`
				class_section_subject_teachers_masjid_id       AS masjid_id,
				class_section_subject_teachers_section_id      AS section_id,
				class_section_subject_teachers_class_subjects_id AS cs_id,
				class_section_subject_teachers_teacher_id      AS teacher_id,
				class_section_subject_teachers_is_active       AS is_active,
				class_section_subject_teachers_deleted_at      AS deleted_at
			`).
			Where("class_section_subject_teachers_id = ?", req.ClassAttendanceSessionCSSTId).
			Take(&a).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Assignment (CSST) tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data CSST")
		}
		if a.MasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "CSST bukan milik masjid Anda")
		}
		if a.DeletedAt != nil || !a.IsActive {
			return fiber.NewError(fiber.StatusBadRequest, "CSST tidak aktif / sudah dihapus")
		}

		// Jika teacher_id kosong → isi dari CSST
		if req.ClassAttendanceSessionTeacherId == nil {
			req.ClassAttendanceSessionTeacherId = a.TeacherID
		} else if a.TeacherID != nil && *req.ClassAttendanceSessionTeacherId != *a.TeacherID {
			// Jika diharuskan strict sama, aktifkan blok ini.
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak sesuai dengan CSST")
		}

		// 2) Validasi TEACHER (opsional)
		if req.ClassAttendanceSessionTeacherId != nil {
			var row struct {
				MasjidID uuid.UUID `gorm:"column:masjid_id"`
				UserID   uuid.UUID `gorm:"column:user_id"`
			}
			if err := tx.Table("masjid_teachers mt").
				Select("mt.masjid_teacher_masjid_id AS masjid_id, mt.masjid_teacher_user_id AS user_id").
				Where("mt.masjid_teacher_id = ? AND mt.masjid_teacher_deleted_at IS NULL", *req.ClassAttendanceSessionTeacherId).
				Take(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Guru (masjid_teacher) tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi guru")
			}
			if row.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik masjid Anda")
			}
			// Jika caller TEACHER → harus milik dirinya
			if teacherMasjidID != uuid.Nil && userID != uuid.Nil && row.UserID != userID {
				return fiber.NewError(fiber.StatusForbidden, "Guru pada payload bukan akun Anda")
			}
		}

		// 3) Validasi ROOM (opsional)
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

		// 4) Cek duplikasi aktif (masjid, csst, date)
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
				AND class_attendance_sessions_csst_id = ?
				AND class_attendance_sessions_date = ?
				AND class_attendance_sessions_deleted_at IS NULL
			`,
				req.ClassAttendanceSessionMasjidId,
				req.ClassAttendanceSessionCSSTId,
				effDate,
			).
			Count(&dupeCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if dupeCount > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// 5) Simpan
		m := req.ToModel() // model sesuai skema baru (tanpa section_id/subject_id)
		if err := tx.Create(&m).Error; err != nil {
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kehadiran")
		}

		c.Locals("created_model", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("created_model").(attendanceModel.ClassAttendanceSessionModel)
	// enrich section & subject untuk response
	resp := attendanceDTO.FromClassAttendanceSessionModel(m)

	c.Set("Location", fmt.Sprintf("/admin/class-attendance-sessions/%s", m.ClassAttendanceSessionId.String()))
	return helper.JsonCreated(c, "Sesi kehadiran berhasil dibuat", resp)
}

/* =========================================================
   PUT /admin/class-attendance-sessions/:id
   (tanpa section/subject di CAS; konsistensi lewat CSST)
========================================================= */
func (ctrl *ClassAttendanceSessionController) UpdateClassAttendanceSession(c *fiber.Ctx) error {
	// Role & Tenant
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)
	userID, _ := helperAuth.GetUserIDFromToken(c)

	var masjidID uuid.UUID
	isTeacher := false
	switch {
	case adminMasjidID != uuid.Nil:
		masjidID = adminMasjidID
	case teacherMasjidID != uuid.Nil:
		masjidID = teacherMasjidID
		isTeacher = true
	default:
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	sessionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req attendanceDTO.UpdateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// enforce tenant
	req.ClassAttendanceSessionMasjidId = &masjidID
	// trim
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
	// normalize date
	if req.ClassAttendanceSessionDate != nil {
		d := req.ClassAttendanceSessionDate.In(time.Local)
		dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
		req.ClassAttendanceSessionDate = &dd
	}
	// validate
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Tx
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// load existing
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

		// Validasi CSST (bila diubah)
		if req.ClassAttendanceSessionCSSTId != nil {
			var a struct {
				MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
				TeacherID *uuid.UUID `gorm:"column:teacher_id"`
				IsActive  bool       `gorm:"column:is_active"`
				DeletedAt *time.Time `gorm:"column:deleted_at"`
			}
			if err := tx.Table("class_section_subject_teachers").
				Select(`
					class_section_subject_teachers_masjid_id  AS masjid_id,
					class_section_subject_teachers_teacher_id  AS teacher_id,
					class_section_subject_teachers_is_active   AS is_active,
					class_section_subject_teachers_deleted_at  AS deleted_at
				`).
				Where("class_section_subject_teachers_id = ?", *req.ClassAttendanceSessionCSSTId).
				Take(&a).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Assignment (CSST) tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data CSST")
			}
			if a.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "CSST bukan milik masjid Anda")
			}
			if a.DeletedAt != nil || !a.IsActive {
				return fiber.NewError(fiber.StatusBadRequest, "CSST tidak aktif / sudah dihapus")
			}
			// sinkron teacher jika kosong
			if req.ClassAttendanceSessionTeacherId == nil {
				req.ClassAttendanceSessionTeacherId = a.TeacherID
			} else if a.TeacherID != nil && *req.ClassAttendanceSessionTeacherId != *a.TeacherID {
				return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak sesuai dengan CSST")
			}
		}

		// Validasi TEACHER (bila diubah)
		if req.ClassAttendanceSessionTeacherId != nil {
			var row struct {
				MasjidID uuid.UUID `gorm:"column:masjid_id"`
				UserID   uuid.UUID `gorm:"column:user_id"`
			}
			if err := tx.Table("masjid_teachers mt").
				Select("mt.masjid_teacher_masjid_id AS masjid_id, mt.masjid_teacher_user_id AS user_id").
				Where("mt.masjid_teacher_id = ? AND mt.masjid_teacher_deleted_at IS NULL", *req.ClassAttendanceSessionTeacherId).
				Take(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Guru (masjid_teacher) tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi guru")
			}
			if row.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik masjid Anda")
			}
			if isTeacher && userID != uuid.Nil && row.UserID != userID {
				return fiber.NewError(fiber.StatusForbidden, "Guru pada payload bukan akun Anda")
			}
		}

		// Validasi ROOM (bila diubah)
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

		// Cek unik (masjid, csst, date) excluding current
		targetDate := func() time.Time {
			if req.ClassAttendanceSessionDate != nil {
				return *req.ClassAttendanceSessionDate
			}
			if existing.ClassAttendanceSessionDate != nil {
				return *existing.ClassAttendanceSessionDate
			}
			now := time.Now().In(time.Local)
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		}()
		targetCSST := existing.ClassAttendanceSessionCSSTId
		if req.ClassAttendanceSessionCSSTId != nil {
			targetCSST = *req.ClassAttendanceSessionCSSTId
		}
		var cnt int64
		if err := tx.Table("class_attendance_sessions").
			Where(`
				class_attendance_sessions_masjid_id = ?
				AND class_attendance_sessions_csst_id = ?
				AND class_attendance_sessions_date = ?
				AND class_attendance_sessions_id <> ?
				AND class_attendance_sessions_deleted_at IS NULL
			`,
				masjidID, targetCSST, targetDate, existing.ClassAttendanceSessionId,
			).Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// Patch
		patch := map[string]interface{}{
			"class_attendance_sessions_masjid_id": masjidID,
		}
		if req.ClassAttendanceSessionDate != nil {
			patch["class_attendance_sessions_date"] = *req.ClassAttendanceSessionDate
			existing.ClassAttendanceSessionDate = req.ClassAttendanceSessionDate
		}
		if req.ClassAttendanceSessionTitle != nil {
			t := strings.TrimSpace(*req.ClassAttendanceSessionTitle)
			patch["class_attendance_sessions_title"] = t
			existing.ClassAttendanceSessionTitle = &t
		}
		if req.ClassAttendanceSessionGeneralInfo != nil {
			g := strings.TrimSpace(*req.ClassAttendanceSessionGeneralInfo)
			patch["class_attendance_sessions_general_info"] = g
			existing.ClassAttendanceSessionGeneralInfo = g
		}
		if req.ClassAttendanceSessionNote != nil {
			n := strings.TrimSpace(*req.ClassAttendanceSessionNote)
			patch["class_attendance_sessions_note"] = n
			existing.ClassAttendanceSessionNote = &n
		}
		if req.ClassAttendanceSessionCSSTId != nil {
			patch["class_attendance_sessions_csst_id"] = *req.ClassAttendanceSessionCSSTId
			existing.ClassAttendanceSessionCSSTId = *req.ClassAttendanceSessionCSSTId
		}
		if req.ClassAttendanceSessionTeacherId != nil {
			patch["class_attendance_sessions_teacher_id"] = *req.ClassAttendanceSessionTeacherId
			existing.ClassAttendanceSessionTeacherId = req.ClassAttendanceSessionTeacherId
		}
		if req.ClassAttendanceSessionClassRoomId != nil {
			patch["class_attendance_sessions_class_room_id"] = *req.ClassAttendanceSessionClassRoomId
			existing.ClassAttendanceSessionClassRoomId = req.ClassAttendanceSessionClassRoomId
		}

		if err := tx.Model(&attendanceModel.ClassAttendanceSessionModel{}).
			Where("class_attendance_sessions_id = ?", existing.ClassAttendanceSessionId).
			Select([]string{
				"class_attendance_sessions_masjid_id",
				"class_attendance_sessions_date",
				"class_attendance_sessions_title",
				"class_attendance_sessions_general_info",
				"class_attendance_sessions_note",
				"class_attendance_sessions_csst_id",
				"class_attendance_sessions_teacher_id",
				"class_attendance_sessions_class_room_id",
			}).
			Updates(patch).Error; err != nil {

			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi kehadiran")
		}

		c.Locals("updated_model", existing)
		return nil
	}); err != nil {
		return err
	}

	updated := c.Locals("updated_model").(attendanceModel.ClassAttendanceSessionModel)
	// enrich di response
	resp := attendanceDTO.FromClassAttendanceSessionModel(updated)
	return helper.JsonUpdated(c, "Sesi kehadiran berhasil diperbarui", resp)
}

/* =========================================================
   DELETE /admin/class-attendance-sessions/:id?force=true
========================================================= */
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
		if err := tx.First(&existing, "class_attendance_sessions_id = ?", sessionID).Error; err != nil {
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
	resp := attendanceDTO.FromClassAttendanceSessionModel(deleted)
	return helper.JsonDeleted(c, "Sesi kehadiran berhasil dihapus", resp)
}
