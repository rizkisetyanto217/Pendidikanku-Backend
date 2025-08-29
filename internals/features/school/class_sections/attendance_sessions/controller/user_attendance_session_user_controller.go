package controller

import (
	"masjidku_backend/internals/features/school/class_sections/attendance_sessions/dto"
	"masjidku_backend/internals/features/school/class_sections/attendance_sessions/model"
	helper "masjidku_backend/internals/helpers"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GET /teacher/class-attendance-sessions?session_id=...&user_class_id=...&status=present&date_from=YYYY-MM-DD&date_to=YYYY-MM-DD&limit=50&offset=0
func (ctrl *TeacherClassAttendanceSessionController) ListAttendanceSessions(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	q := ctrl.DB.Model(&model.UserClassAttendanceSessionModel{}).
		Where("user_class_attendance_sessions_masjid_id = ?", masjidID)

	// filter by session_id
	if s := strings.TrimSpace(c.Query("session_id")); s != "" {
		sid, err := uuid.Parse(s)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "session_id tidak valid")
		}
		q = q.Where("user_class_attendance_sessions_session_id = ?", sid)
	}

	// filter by user_class_id
	if uc := strings.TrimSpace(c.Query("user_class_id")); uc != "" {
		uid, err := uuid.Parse(uc)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "user_class_id tidak valid")
		}
		q = q.Where("user_class_attendance_sessions_user_class_id = ?", uid)
	}

	// filter by status (present|sick|leave|absent) - biarkan bebas jika enum sudah diproteksi DB
	if st := strings.TrimSpace(c.Query("status")); st != "" {
		q = q.Where("user_class_attendance_sessions_attendance_status = ?", st)
	}

	// filter tanggal via JOIN ke class_attendance_sessions (punya kolom date)
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))
	if df != "" || dt != "" {
		q = q.Joins(`
			JOIN class_attendance_sessions s
			  ON s.class_attendance_sessions_id = user_class_attendance_sessions_session_id
		`).Where("s.class_attendance_sessions_masjid_id = ?", masjidID)

		if df != "" {
			t, err := time.Parse("2006-01-02", df)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
			}
			q = q.Where("s.class_attendance_sessions_date >= ?", t)
		}
		if dt != "" {
			t, err := time.Parse("2006-01-02", dt)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
			}
			q = q.Where("s.class_attendance_sessions_date <= ?", t)
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

	// total sebelum limit/offset
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// fetch
	var rows []model.UserClassAttendanceSessionModel
	if err := q.
		Order("user_class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]dto.UserClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, dto.FromUserClassAttendanceSessionModel(r))
	}

	// konsisten: JsonList
	return helper.JsonList(c, items, fiber.Map{
		"limit":     limit,
		"offset":    offset,
		"total":     int(total),
		"date_from": df,
		"date_to":   dt,
	})
}
