// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	attendanceDTO "masjidku_backend/internals/features/school/sessions_assesment/sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"
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

func scopeSection(id *uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if id == nil {
			return db
		}
		return db.Where("class_attendance_sessions_section_id = ?", *id)
	}
}

func scopeClassSubject(id *uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if id == nil {
			return db
		}
		return db.Where("class_attendance_sessions_class_subject_id = ?", *id)
	}
}

// (baru) filter by CSST
func scopeCSST(id *uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if id == nil {
			return db
		}
		return db.Where("class_attendance_sessions_csst_id = ?", *id)
	}
}

// Filter berdasarkan masjid_teacher_id yang tersimpan di CAS.teacher_id
func scopeTeacherId(teacherId *uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if teacherId == nil {
			return db
		}
		return db.Where("class_attendance_sessions_teacher_id = ?", *teacherId)
	}
}

func parseYmd(s string) (*time.Time, error) {
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

// isTeacherOfSession: user (guru) adalah pengajar sesi bila CAS.teacher_id
// merujuk ke masjid_teachers milik user tsb pada masjid yang sama.
func (ctrl *ClassAttendanceSessionController) isTeacherOfSession(session attendanceModel.ClassAttendanceSessionModel, userID, masjidID uuid.UUID) (bool, error) {
	if session.ClassAttendanceSessionTeacherId == nil {
		return false, nil
	}
	var cnt int64
	q := ctrl.DB.Table("masjid_teachers AS mt").
		Where("mt.masjid_teacher_id = ?", *session.ClassAttendanceSessionTeacherId).
		Where("mt.masjid_teacher_user_id = ?", userID).
		Where("mt.masjid_teacher_masjid_id = ?", masjidID)

	if err := q.Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

/* =========================================================
   GET /admin/class-attendance-sessions/:id
========================================================= */

func (ctrl *ClassAttendanceSessionController) GetClassAttendanceSession(c *fiber.Ctx) error {
	// Tenant (admin/teacher)
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
		Scopes(scopeMasjid(masjidID)).
		First(&m, "class_attendance_sessions_id = ?", id).Error; err != nil {
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

	if !isAdmin {
		if isTeacher {
			// Guru hanya boleh lihat sesi yang dia ajar (berdasarkan CAS.teacher_id)
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
			// Siswa/ortu: hanya jika terdaftar aktif di section
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
				`, m.ClassAttendanceSessionSectionId, masjidID, userID).
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal verifikasi akses")
			}
			if cnt == 0 {
				return fiber.NewError(fiber.StatusForbidden, "Tidak berhak mengakses sesi ini")
			}
		}
	}

	return helper.JsonOK(c, "Sesi kehadiran ditemukan", attendanceDTO.FromClassAttendanceSessionModel(m))
}

/* ==========================================================================================
   GET /admin/class-attendance-sessions?section_id=&teacher_id=&teacher_user_id=&class_subject_id=&csst_id=&date_from=&date_to=&limit=&offset=
   - teacher_id        → masjid_teacher_id (langsung ke CAS.teacher_id)
   - teacher_user_id   → users.id (legacy), lewat JOIN ke masjid_teachers
   - csst_id           → class_section_subject_teachers_id (baru, opsional)
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

	// ===== Pagination & sorting (via helper) =====
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}

	// pakai preset AdminOpts (endpoint /admin/...)
	p := helper.ParseWith(httpReq, "date", "desc", helper.AdminOpts)

	// whitelist kolom sort (tanpa created_at karena tidak ada di skema)
	allowedSort := map[string]string{
		"date":  "class_attendance_sessions_date",
		"title": "class_attendance_sessions_title",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "date")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// ===== Query params filter lain =====
	dfStr := strings.TrimSpace(c.Query("date_from"))
	dtStr := strings.TrimSpace(c.Query("date_to"))

	df, err := parseYmd(dfStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYmd(dtStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}

	var sectionIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		if sid, e := uuid.Parse(s); e == nil {
			sectionIDPtr = &sid
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
	}

	// teacher_id = masjid_teacher_id (baru)
	var teacherIdPtr *uuid.UUID
	if t := strings.TrimSpace(c.Query("teacher_id")); t != "" {
		if tid, e := uuid.Parse(t); e == nil {
			teacherIdPtr = &tid
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
	}

	// legacy: teacher_user_id = users.id → join ke masjid_teachers
	var teacherUserIDPtr *uuid.UUID
	if tu := strings.TrimSpace(c.Query("teacher_user_id")); tu != "" {
		if tuid, e := uuid.Parse(tu); e == nil {
			teacherUserIDPtr = &tuid
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid")
		}
	}

	// optional: class_subject_id
	var classSubjectIDPtr *uuid.UUID
	if cs := strings.TrimSpace(c.Query("class_subject_id")); cs != "" {
		if csid, e := uuid.Parse(cs); e == nil {
			classSubjectIDPtr = &csid
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "class_subject_id tidak valid")
		}
	}

	// (baru) optional: csst_id
	var csstIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.Query("csst_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			csstIDPtr = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "csst_id tidak valid")
		}
	}

	// ===== Base query + scopes =====
	qBase := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Scopes(
			scopeMasjid(masjidID),
			scopeDateBetween(df, dt),
			scopeSection(sectionIDPtr),
			scopeClassSubject(classSubjectIDPtr),
			scopeCSST(csstIDPtr),
			scopeTeacherId(teacherIdPtr), // langsung ke CAS.teacher_id
		)

	// legacy filter by teacher_user_id (pakai alias berbeda agar tidak bentrok)
	if teacherUserIDPtr != nil {
		qBase = qBase.Joins(`
			JOIN masjid_teachers mt_q
			  ON mt_q.masjid_teacher_id = class_attendance_sessions_teacher_id
		`).Where("mt_q.masjid_teacher_user_id = ?", *teacherUserIDPtr)
	}

	// ===== Scope berdasarkan role =====
	if !isAdmin {
		if isTeacher {
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			// Batasi ke sesi yang diajar oleh guru (user) ini via join ke masjid_teachers
			qBase = qBase.
				Joins(`
					JOIN masjid_teachers mt
					  ON mt.masjid_teacher_id = class_attendance_sessions_teacher_id
				`).
				Where("mt.masjid_teacher_user_id = ?", userID).
				Where("mt.masjid_teacher_masjid_id = ?", masjidID)
		} else {
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			sub := ctrl.DB.Table("user_class_sections AS ucs").
				Joins("JOIN user_classes uc ON uc.user_classes_id = ucs.user_class_sections_user_class_id").
				Where(`
					ucs.user_class_sections_masjid_id = ?
					AND uc.user_classes_user_id = ?
					AND ucs.user_class_sections_unassigned_at IS NULL
					AND uc.user_classes_status = 'active'
				`, masjidID, userID).
				Select("ucs.user_class_sections_section_id")
			qBase = qBase.Where("class_attendance_sessions_section_id IN (?)", sub)
		}
	}

	// ===== Total =====
	var total int64
	if err := qBase.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Data (order+limit+offset dari helper) =====
	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := qBase.
		Order(orderExpr).
		// secondary order untuk stabilitas (tanpa created_at di skema)
		Order("class_attendance_sessions_id DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for i := range rows {
		items = append(items, attendanceDTO.FromClassAttendanceSessionModel(rows[i]))
	}

	// ===== Meta dari helper =====
	meta := helper.BuildMeta(total, p)

	return helper.JsonList(c, items, meta)
}
