package controller

import (
	"errors"
	attendanceDTO "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/dto"
	attendanceModel "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/model"
	helper "masjidku_backend/internals/helpers"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GET /admin/class-attendance-sessions/:id
func (ctrl *ClassAttendanceSessionController) GetClassAttendanceSession(c *fiber.Ctx) error {
	// Tenant (admin/teacher)
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	// Ambil data
	var m attendanceModel.ClassAttendanceSessionModel
	if err := ctrl.DB.
		Select(`
			class_attendance_session_id,
			class_attendance_session_section_id,
			class_attendance_session_masjid_id,
			class_attendance_session_date,
			class_attendance_session_title,
			class_attendance_session_general_info,
			class_attendance_session_note,
			class_attendance_session_teacher_user_id,
			class_attendance_session_subject_id,
			class_attendance_session_class_section_subject_teacher_id,
			class_attendance_session_created_at,
			class_attendance_session_updated_at,
			class_attendance_session_deleted_at
		`).
		First(&m, "class_attendance_session_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Tenant guard
	if m.ClassAttendanceSessionMasjidId != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Akses ditolak")
	}

	// Role guard
	userID, _ := helper.GetUserIDFromToken(c)
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helper.GetTeacherMasjidIDFromToken(c)

	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	isTeacher := teacherMasjidID != uuid.Nil && teacherMasjidID == masjidID

	if !isAdmin {
		if isTeacher {
			// guru hanya boleh lihat sesi yang dia ajar
			if userID == uuid.Nil || (m.ClassAttendanceSessionTeacherUserId != nil && *m.ClassAttendanceSessionTeacherUserId != userID) {
				return fiber.NewError(fiber.StatusForbidden, "Bukan pengajar sesi ini")
			}
		} else {
			// siswa/ortu: hanya jika terdaftar aktif di section
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


// GET /admin/class-attendance-sessions?section_id=&teacher_id=&date_from=&date_to=&limit=&offset=
func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	// ===== Tenant (admin/teacher) =====
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	// ===== Role =====
	userID, _ := helper.GetUserIDFromToken(c)
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helper.GetTeacherMasjidIDFromToken(c)

	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	isTeacher := teacherMasjidID != uuid.Nil && teacherMasjidID == masjidID

	// ===== Base query =====
	qBase := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where("class_attendance_session_masjid_id = ?", masjidID)

	// ===== Filter tanggal (opsional) =====
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))

	parseDate := func(s string) (time.Time, error) {
		t, e := time.Parse("2006-01-02", s)
		if e != nil { return time.Time{}, e }
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}
	switch {
	case df != "" && dt != "":
		from, e1 := parseDate(df); if e1 != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
		to,   e2 := parseDate(dt); if e2 != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("class_attendance_session_date BETWEEN ? AND ?", from, to)
	case df != "":
		from, e := parseDate(df); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("class_attendance_session_date >= ?", from)
	case dt != "":
		to, e := parseDate(dt); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
		qBase = qBase.Where("class_attendance_session_date <= ?", to)
	}

	// ===== Param opsional =====
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		if sid, err := uuid.Parse(s); err == nil {
			qBase = qBase.Where("class_attendance_session_section_id = ?", sid)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
	}
	if t := strings.TrimSpace(c.Query("teacher_id")); t != "" {
		if tid, err := uuid.Parse(t); err == nil {
			qBase = qBase.Where("class_attendance_session_teacher_user_id = ?", tid)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
	}

	// ===== Scope berdasarkan role =====
	if !isAdmin {
		if isTeacher {
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			qBase = qBase.Where("class_attendance_session_teacher_user_id = ?", userID)
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
			qBase = qBase.Where("class_attendance_session_section_id IN (?)", sub)
		}
	}

	// ===== Pagination =====
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }

	// ===== Total =====
	var total int64
	if err := qBase.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Query data =====
	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := qBase.
		Select(`
			class_attendance_session_id,
			class_attendance_session_section_id,
			class_attendance_session_masjid_id,
			class_attendance_session_date,
			class_attendance_session_title,
			class_attendance_session_general_info,
			class_attendance_session_note,
			class_attendance_session_teacher_user_id,
			class_attendance_session_subject_id,
			class_attendance_session_class_section_subject_teacher_id,
			class_attendance_session_created_at,
			class_attendance_session_updated_at,
			class_attendance_session_deleted_at
		`).
		Order("class_attendance_session_date DESC, class_attendance_session_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	// ===== Return konsisten: JsonList (pakai fiber.Map untuk pagination/meta)
	return helper.JsonList(c, items, fiber.Map{
		"limit":    limit,
		"offset":   offset,
		"total":    int(total),
		"date_from": df,
		"date_to":   dt,
	})
}

