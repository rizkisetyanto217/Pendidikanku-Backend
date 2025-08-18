package controller

import (
	"masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/dto"
	"masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/model"
	helper "masjidku_backend/internals/helpers"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* ===============
====== LIST ===================== */
// GET /teacher/class-attendance-sessions?session_id=...&user_class_id=...&status=present&date_from=YYYY-MM-DD&date_to=YYYY-MM-DD&limit=50&offset=0
func (ctrl *TeacherClassAttendanceSessionController) ListAttendanceSessions(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	q := ctrl.DB.Model(&model.UserClassAttendanceSessionModel{}).
		Where("user_class_attendance_sessions_masjid_id = ?", masjidID)

	// filter by session_id
	if s := c.Query("session_id"); s != "" {
		if sid, err := uuid.Parse(s); err == nil {
			q = q.Where("user_class_attendance_sessions_session_id = ?", sid)
		}
	}

	// filter by user_class_id
	if uc := c.Query("user_class_id"); uc != "" {
		if u, err := uuid.Parse(uc); err == nil {
			q = q.Where("user_class_attendance_sessions_user_class_id = ?", u)
		}
	}

	// filter by status (TEXT: present|sick|leave|absent)
	if st := c.Query("status"); st != "" {
		q = q.Where("user_class_attendance_sessions_attendance_status = ?", st)
	}

	// filter tanggal via JOIN ke class_attendance_sessions (punya kolom date)
	df := c.Query("date_from")
	dt := c.Query("date_to")
	if df != "" || dt != "" {
		q = q.Joins(`JOIN class_attendance_sessions s
                      ON s.class_attendance_sessions_id = user_class_attendance_sessions_session_id`).
			Where("s.class_attendance_sessions_masjid_id = ?", masjidID)

		if df != "" {
			if t, err := time.Parse("2006-01-02", df); err == nil {
				q = q.Where("s.class_attendance_sessions_date >= ?", t)
			}
		}
		if dt != "" {
			if t, err := time.Parse("2006-01-02", dt); err == nil {
				q = q.Where("s.class_attendance_sessions_date <= ?", t)
			}
		}
	}

	// pagination
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var rows []model.UserClassAttendanceSessionModel
	if err := q.
		Order("user_class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]dto.UserClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, dto.FromUserClassAttendanceSessionModel(r))
	}

	return helper.JsonOK(c, "Daftar kehadiran ditemukan", resp)
}
