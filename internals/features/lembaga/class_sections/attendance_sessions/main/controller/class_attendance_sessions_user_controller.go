// internals/features/lembaga/class_sections/attendance_sessions/main/controller/class_attendance_session_controller.go
package controller

import (
	"errors"
	attendanceDTO "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/dto"
	attendanceModel "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/model"
	"strconv"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GET /admin/class-attendance-sessions/:id
func (ctrl *ClassAttendanceSessionController) GetClassAttendanceSession(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	id, err := uuid.Parse(c.Params("id"))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	var m attendanceModel.ClassAttendanceSessionModel
	if err := ctrl.DB.First(&m, "class_attendance_sessions_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.MasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Akses ditolak")
	}

	// ✅ pakai helper.JsonOK
	return helper.JsonOK(c, "Sesi kehadiran ditemukan", attendanceDTO.FromClassAttendanceSessionModel(m))
}



// GET /admin/class-attendance-sessions?section_id=&teacher_id=&date_from=&date_to=&limit=&offset=
func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	// Tenant
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	// User & role
	userID, _ := helper.GetUserIDFromToken(c)

	isAdmin := func() bool {
		if mid, err := helper.GetMasjidIDFromToken(c); err == nil && mid == masjidID { return true }
		return false
	}()
	isTeacher := func() bool {
		if mid, err := helper.GetTeacherMasjidIDFromToken(c); err == nil && mid == masjidID { return true }
		return false
	}()

	q := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where("class_attendance_sessions_masjid_id = ?", masjidID)

	// ====== FILTER TANGGAL (opsional; default: TIDAK difilter) ======
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))

	parseDate := func(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }

	if df != "" || dt != "" {
		// gunakan half-open range agar satu hari penuh ikut
		if df != "" {
			if t, err := parseDate(df); err == nil {
				start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
				q = q.Where("class_attendance_sessions_date >= ?", start)
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
			}
		}
		if dt != "" {
			if t, err := parseDate(dt); err == nil {
				endExclusive := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Add(24 * time.Hour)
				q = q.Where("class_attendance_sessions_date < ?", endExclusive)
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
			}
		}
	}
	// jika df & dt kosong → tidak ada kondisi tanggal sama sekali (tampilkan semua)

	// ====== PARAM OPSIONAL ======
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		if sid, err := uuid.Parse(s); err == nil {
			q = q.Where("class_attendance_sessions_section_id = ?", sid)
		}
	}
	if t := strings.TrimSpace(c.Query("teacher_id")); t != "" {
		if tid, err := uuid.Parse(t); err == nil {
			q = q.Where("class_attendance_sessions_teacher_user_id = ?", tid)
		}
	}

	// ====== SCOPE BERDASARKAN ROLE ======
	if !isAdmin {
		if isTeacher {
			// Guru hanya lihat sesi yang dia ajar
			if userID == uuid.Nil {
				return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
			}
			q = q.Where("class_attendance_sessions_teacher_user_id = ?", userID)
		} else {
			// Siswa/Ortu: hanya section tempat user sedang ter-assign (aktif)
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

			q = q.Where("class_attendance_sessions_section_id IN (?)", sub)
		}
	}
	// Catatan: admin tetap bisa melihat semua, plus filter manual via query param.

	// ====== PAGINATION ======
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }

	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := q.
		Order("class_attendance_sessions_date DESC, class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	return helper.JsonOK(c, "Daftar sesi kehadiran berhasil diambil", fiber.Map{
		"limit":  limit,
		"offset": offset,
		"count":  len(resp),
		"items":  resp,
	})
}
