// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	attendanceDTO "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/dto"
	attendanceModel "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/model"
	helper "masjidku_backend/internals/helpers"

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

// scopeTeacherUser membatasi ke sesi yang diajar oleh user (guru) tertentu
// dengan JOIN ke class_section_subject_teachers (csst).
func scopeTeacherUser(userID *uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if userID == nil {
			return db
		}
		return db.
			Joins(`
				JOIN class_section_subject_teachers csst
				  ON csst.class_section_subject_teachers_id = class_attendance_sessions_class_section_subject_teacher_id
			`).
			Where("csst.class_section_subject_teachers_user_id = ?", *userID).
			// Hindari duplikasi jika ada JOIN tambahan lain di masa depan
			Group("class_attendance_sessions_id")
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
	// normalisasi ke midnight local (kolom tipe DATE)
	tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return &tt, nil
}

// isTeacherOfSession mengecek apakah user (guru) adalah pengajar untuk sesi tertentu
// via tabel class_section_subject_teachers.
func (ctrl *ClassAttendanceSessionController) isTeacherOfSession(session attendanceModel.ClassAttendanceSessionModel, userID, masjidID uuid.UUID) (bool, error) {
	if session.ClassAttendanceSessionClassSectionSubjectTeacherId == nil {
		return false, nil
	}
	var cnt int64
	q := ctrl.DB.Table("class_section_subject_teachers AS csst").
		Where("csst.class_section_subject_teachers_id = ?", *session.ClassAttendanceSessionClassSectionSubjectTeacherId).
		Where("csst.class_section_subject_teachers_user_id = ?", userID)

	// Jika Anda menyimpan masjid_id di csst, aktifkan baris berikut:
	q = q.Where("csst.class_section_subject_teachers_masjid_id = ?", masjidID)

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
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
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
	userID, _ := helper.GetUserIDFromToken(c)
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helper.GetTeacherMasjidIDFromToken(c)

	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	isTeacher := teacherMasjidID != uuid.Nil && teacherMasjidID == masjidID

	if !isAdmin {
		if isTeacher {
			// Guru hanya boleh lihat sesi yang dia ajar
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
					AND uc.user_classes_ended_at IS NULL
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
   GET /admin/class-attendance-sessions?section_id=&teacher_id=&date_from=&date_to=&limit=&offset=
   - teacher_id sekarang diartikan sebagai USER ID dari guru.
   - Filtering dilakukan via JOIN ke class_section_subject_teachers.
========================================================================================== */

func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	// Tenant (admin/teacher)
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// Role
	userID, _ := helper.GetUserIDFromToken(c)
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helper.GetTeacherMasjidIDFromToken(c)

	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	isTeacher := teacherMasjidID != uuid.Nil && teacherMasjidID == masjidID

	// Query params
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

	// teacher_id = user_id guru
	var teacherUserIDPtr *uuid.UUID
	if t := strings.TrimSpace(c.Query("teacher_id")); t != "" {
		if tid, e := uuid.Parse(t); e == nil {
			teacherUserIDPtr = &tid
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
	}

	// Base query + scopes
	qBase := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Scopes(
			scopeMasjid(masjidID),
			scopeDateBetween(df, dt),
			scopeSection(sectionIDPtr),
			scopeTeacherUser(teacherUserIDPtr), // << gunakan JOIN ke csst jika filter teacher diberikan
		)

	// Scope berdasarkan role
	if !isAdmin {
		if isTeacher {
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			// Batasi ke sesi yang diajar oleh guru (user) ini
			qBase = qBase.
				Joins(`
					JOIN class_section_subject_teachers csst
					  ON csst.class_section_subject_teachers_id = class_attendance_sessions_class_section_subject_teacher_id
				`).
				Where("csst.class_section_subject_teachers_user_id = ?", userID).
				// Jika csst punya kolom masjid_id:
				Where("csst.class_section_subject_teachers_masjid_id = ?", masjidID).
				Group("class_attendance_sessions_id")
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
					AND uc.user_classes_ended_at IS NULL
				`, masjidID, userID).
				Select("ucs.user_class_sections_section_id")
			qBase = qBase.Where("class_attendance_sessions_section_id IN (?)", sub)
		}
	}

	// Pagination
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Total
	var total int64
	if err := qBase.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// Data
	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := qBase.
		Order("class_attendance_sessions_date DESC, class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for i := range rows {
		items = append(items, attendanceDTO.FromClassAttendanceSessionModel(rows[i]))
	}

	return helper.JsonList(c, items, fiber.Map{
		"limit":     limit,
		"offset":    offset,
		"total":     int(total),
		"date_from": dfStr,
		"date_to":   dtStr,
	})
}
