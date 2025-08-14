// internals/features/lembaga/class_sections/attendance_sessions/main/controller/class_attendance_session_controller.go
package controller

import (
	"errors"
	attendanceDTO "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/dto"
	attendanceModel "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/model"
	"strconv"
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
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	q := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where("class_attendance_sessions_masjid_id = ?", masjidID)

	if s := c.Query("section_id"); s != "" {
		if sid, err := uuid.Parse(s); err == nil {
			q = q.Where("class_attendance_sessions_section_id = ?", sid)
		}
	}
	if t := c.Query("teacher_id"); t != "" {
		if tid, err := uuid.Parse(t); err == nil {
			q = q.Where("class_attendance_sessions_teacher_user_id = ?", tid)
		}
	}
	if df := c.Query("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			q = q.Where("class_attendance_sessions_date >= ?", t)
		}
	}
	if dt := c.Query("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			q = q.Where("class_attendance_sessions_date <= ?", t)
		}
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }

	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := q.Order("class_attendance_sessions_date DESC, class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	// ✅ pakai helper.JsonOK
	return helper.JsonOK(c, "Daftar sesi kehadiran berhasil diambil", resp)
}
