// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"errors"
	"net/http"
	"net/url"
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

/* =========================================================
   GET /admin/class-attendance-sessions/:id
========================================================= */
/* =========================================================
   GET /admin/class-attendance-sessions/:id
   (enrich section_id & class_subject_id dari CSST)
========================================================= */
func (ctrl *ClassAttendanceSessionController) GetClassAttendanceSession(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m attendanceModel.ClassAttendanceSessionModel
	if err := ctrl.DB.
		Where("class_attendance_sessions_masjid_id = ?", masjidID).
		First(&m, "class_attendance_sessions_id = ? AND class_attendance_sessions_deleted_at IS NULL", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Role guard
	userID, _ := helperAuth.GetUserIDFromToken(c)
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)

	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	isTeacher := teacherMasjidID != uuid.Nil && teacherMasjidID == masjidID

	// Ambil section & subject dari CSST untuk guard siswa/ortu & enrichment
	secID, subID, err := ctrl.csstSectionAndSubject(m.ClassAttendanceSessionCSSTId)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data CSST")
	}

	if !isAdmin {
		if isTeacher {
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			ok, e := ctrl.isTeacherOfSession(m, userID, masjidID)
			if e != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal verifikasi pengajar sesi")
			}
			if !ok {
				return fiber.NewError(fiber.StatusForbidden, "Bukan pengajar sesi ini")
			}
		} else {
			// siswa/ortu: harus aktif di section dari CSST
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			var cnt int64
			q := ctrl.DB.Table("user_class_sections AS ucs").
				Joins("JOIN user_classes uc ON uc.user_classes_id = ucs.user_class_sections_user_class_id").
				Joins("JOIN masjid_students ms ON ms.masjid_student_id = uc.user_classes_masjid_student_id AND ms.masjid_student_deleted_at IS NULL").
				Where(`
					ucs.user_class_sections_masjid_id = ?
					AND ucs.user_class_sections_section_id = ?
					AND ucs.user_class_sections_unassigned_at IS NULL
					AND uc.user_classes_status = 'active'
					AND ms.masjid_student_user_id = ?
					AND ms.masjid_student_masjid_id = ?
				`, masjidID, secID, userID, masjidID).
				Count(&cnt)
			if q.Error != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal verifikasi akses")
			}
			if cnt == 0 {
				return fiber.NewError(fiber.StatusForbidden, "Tidak berhak mengakses sesi ini")
			}
		}
	}

	// Build response + enrichment (section & subject dari CSST)
	// ... setelah cek akses & ambil secID, subID
	// Build response + enrichment (section & subject dari CSST)
	resp := attendanceDTO.FromClassAttendanceSessionModel(m)
	resp.ClassSectionId = &secID
	resp.ClassSubjectId = &subID

	return helper.JsonOK(c, "Sesi kehadiran ditemukan", resp)

}

/* ==========================================================================================
   GET /admin/class-attendance-sessions
     ?teacher_id=&teacher_user_id=&section_id=&class_subject_id=&csst_id=&date_from=&date_to=&limit=&offset=&q=&sort_by=&sort=
   - teacher_id      → masjid_teacher_id (CAS.teacher_id atau CSST.teacher_id)
   - teacher_user_id → users.id (JOIN ke masjid_teachers)
   - section_id      → filter via CSST.section_id
   - class_subject_id→ filter via CSST.class_subjects_id
   - csst_id         → langsung ke CAS.csst_id
========================================================================================== */
/* ==========================================================================================
   GET /admin/class-attendance-sessions
     ?teacher_id=&teacher_user_id=&section_id=&class_subject_id=&csst_id=&date_from=&date_to=&limit=&offset=&q=&sort_by=&sort=
   - teacher_id      → masjid_teacher_id (CAS.teacher_id atau CSST.teacher_id)
   - teacher_user_id → users.id (JOIN ke masjid_teachers)
   - section_id      → filter via CSST.section_id
   - class_subject_id→ filter via CSST.class_subjects_id
   - csst_id         → langsung ke CAS.csst_id
========================================================================================== */
func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	// Tenant (admin/teacher)
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// Role
	userID, _ := helperAuth.GetUserIDFromToken(c)
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)

	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	isTeacher := teacherMasjidID != uuid.Nil && teacherMasjidID == masjidID

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

	// ===== Filters =====
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
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
		teacherIdPtr = &id
	}

	var teacherUserIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("teacher_user_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid")
		}
		teacherUserIDPtr = &id
	}

	var sectionIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		sectionIDPtr = &id
	}

	var classSubjectIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("class_subject_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "class_subject_id tidak valid")
		}
		classSubjectIDPtr = &id
	}

	var csstIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("csst_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "csst_id tidak valid")
		}
		csstIDPtr = &id
	}

	keyword := strings.TrimSpace(c.Query("q"))
	var like *string
	if keyword != "" {
		pat := "%" + keyword + "%"
		like = &pat
	}

	// ===== Base with aliases =====
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

	// Filter by CSST-derived fields
	if sectionIDPtr != nil {
		qBase = qBase.Where("csst.class_section_subject_teachers_section_id = ?", *sectionIDPtr)
	}
	if classSubjectIDPtr != nil {
		qBase = qBase.Where("csst.class_section_subject_teachers_class_subjects_id = ?", *classSubjectIDPtr)
	}

	// teacher_id filter harus cover CAS.teacher_id ATAU CSST.teacher_id (hanya bila diberikan)
	if teacherIdPtr != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_sessions_teacher_id = ?
			 OR csst.class_section_subject_teachers_teacher_id = ?)
		`, *teacherIdPtr, *teacherIdPtr)
	}

	// teacher_user_id: map ke users.id via masjid_teachers
	if teacherUserIDPtr != nil {
		qBase = qBase.
			Joins(`
				LEFT JOIN masjid_teachers mt_cas
				  ON mt_cas.masjid_teacher_id = cas.class_attendance_sessions_teacher_id
			`).
			Joins(`
				LEFT JOIN masjid_teachers mt_csst
				  ON mt_csst.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id
			`).
			Where(`
				(mt_cas.masjid_teacher_user_id = ? AND mt_cas.masjid_teacher_masjid_id = ?)
			 OR (mt_csst.masjid_teacher_user_id = ? AND mt_csst.masjid_teacher_masjid_id = ?)
			`, *teacherUserIDPtr, masjidID, *teacherUserIDPtr, masjidID)
	}

	// keyword
	if like != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_sessions_title ILIKE ?
			 OR cas.class_attendance_sessions_general_info ILIKE ?)
		`, *like, *like)
	}

	// ===== Scope berdasarkan role =====
	if !isAdmin {
		if isTeacher {
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			// Guru hanya sesi yang diajar oleh dirinya (CAS.teacher_id ATAU CSST.teacher_id)
			qBase = qBase.
				Joins(`
					LEFT JOIN masjid_teachers mt1
					  ON mt1.masjid_teacher_id = cas.class_attendance_sessions_teacher_id
				`).
				Joins(`
					LEFT JOIN masjid_teachers mt2
					  ON mt2.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id
				`).
				Where(`
					(mt1.masjid_teacher_user_id = ? AND mt1.masjid_teacher_masjid_id = ?)
				 OR (mt2.masjid_teacher_user_id = ? AND mt2.masjid_teacher_masjid_id = ?)
				`, userID, masjidID, userID, masjidID)
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
				)
			`, userID, masjidID)
		}
	}

	// ===== Total (distinct id) =====
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_sessions_id").
		Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Data =====
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

		// Enrichment dari CSST (LEFT JOIN → pakai pointer supaya aman jika NULL)
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
			csst.class_section_subject_teachers_section_id      AS section_id,
			csst.class_section_subject_teachers_class_subjects_id AS subject_id
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
		resp := attendanceDTO.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:          r.ID,
			ClassAttendanceSessionMasjidId:    r.MasjidID,
			ClassAttendanceSessionCSSTId:      r.CSSTID,   // CSST = wajib → non-pointer
			ClassAttendanceSessionClassRoomId: r.RoomID,
			ClassAttendanceSessionDate:        r.Date,
			ClassAttendanceSessionTitle:       r.Title,
			ClassAttendanceSessionGeneralInfo: r.General,
			ClassAttendanceSessionNote:        r.Note,
			ClassAttendanceSessionTeacherId:   r.TeacherID,
			ClassAttendanceSessionDeletedAt:   r.DeletedAt,

			// enrichment (opsional)
			ClassSectionId: r.SectionID,
			ClassSubjectId: r.SubjectID,
		}
		items = append(items, resp)
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}
