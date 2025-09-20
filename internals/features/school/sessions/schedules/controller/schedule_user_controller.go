package controller

import (
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"net/http"
	"strings"
	"time"

	d "masjidku_backend/internals/features/school/sessions/schedules/dto"
	m "masjidku_backend/internals/features/school/sessions/schedules/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================
   Query: List
   ========================= */

func (ctl *ClassScheduleController) List(c *fiber.Ctx) error {
	// supaya resolver slug → ID bisa akses DB
	c.Locals("DB", ctl.DB)

	var q d.ListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// ===== Tentukan masjid_id aktif =====
	// Prefer masjid context eksplisit (wajib DKM/Admin di masjid tsb),
	// jika tidak ada context → fallback ke token (teacher-aware).
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		q.MasjidID = id.String()
	} else if act, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && act != uuid.Nil {
		q.MasjidID = act.String()
	}

	// Whitelist sorting
	sortCol := map[string]string{
		"start_time": "class_schedules_start_time",
		"end_time":   "class_schedules_end_time",
		"created_at": "class_schedules_created_at",
		"updated_at": "class_schedules_updated_at",
	}
	sortBy := "class_schedules_start_time"
	if s := strings.TrimSpace(q.SortBy); s != "" {
		if col, ok := sortCol[s]; ok {
			sortBy = col
		}
	}
	order := "ASC"
	if strings.EqualFold(q.Order, "desc") {
		order = "DESC"
	}

	// Pagination clamp
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	// ===== Build base query with filters =====
	tx := ctl.DB.Model(&m.ClassScheduleModel{}).
		Where("class_schedules_deleted_at IS NULL")

	// by masjid
	if s := strings.TrimSpace(q.MasjidID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "masjid_id invalid")
		}
		tx = tx.Where("class_schedules_masjid_id = ?", s)
	}

	// by ids (FIX: gunakan class_schedule_id)
	if s := strings.TrimSpace(q.ClassScheduleID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "class_schedule_id invalid")
		}
		tx = tx.Where("class_schedule_id = ?", s)
	}


	// by foreign keys

	if s := strings.TrimSpace(q.CSSTID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "csst_id invalid")
		}
		tx = tx.Where("class_schedules_csst_id = ?", s)
	}

	// by status
	if s := strings.TrimSpace(q.Status); s != "" {
		switch m.SessionStatus(s) {
		case m.SessionScheduled, m.SessionOngoing, m.SessionCompleted, m.SessionCanceled:
			tx = tx.Where("class_schedules_status = ?", s)
		default:
			return helper.JsonError(c, http.StatusBadRequest, "status invalid")
		}
	}

	// by active
	if q.Active != nil {
		tx = tx.Where("class_schedules_is_active = ?", *q.Active)
	}

	// by day-of-week
	if q.DayOfWeek != nil {
		if *q.DayOfWeek < 1 || *q.DayOfWeek > 7 {
			return helper.JsonError(c, http.StatusBadRequest, "dow must be 1..7")
		}
		tx = tx.Where("class_schedules_day_of_week = ?", *q.DayOfWeek)
	}

	// by on_date
	if s := strings.TrimSpace(q.OnDate); s != "" {
		dt, err := time.Parse("2006-01-02", s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "on_date invalid (YYYY-MM-DD)")
		}
		dow := int(dt.Weekday()) // Sunday(0)..Saturday(6)
		if dow == 0 {
			dow = 7 // ISO 1..7
		}
		tx = tx.
			Where("?::date BETWEEN class_schedules_start_date AND class_schedules_end_date", dt).
			Where("class_schedules_day_of_week = ?", dow)
	}

	// by time windows
	if s := strings.TrimSpace(q.StartAfter); s != "" {
		tm, err := parseTimeOfDayParam(s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "start_after invalid (HH:mm/HH:mm:ss)")
		}
		tx = tx.Where("class_schedules_start_time >= ?", tm)
	}
	if s := strings.TrimSpace(q.EndBefore); s != "" {
		tm, err := parseTimeOfDayParam(s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "end_before invalid (HH:mm/HH:mm:ss)")
		}
		tx = tx.Where("class_schedules_end_time <= ?", tm)
	}

	// ===== Count total =====
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return writePGError(c, err)
	}

	// ===== Fetch page =====
	var rows []m.ClassScheduleModel
	if err := tx.
		Order(sortBy + " " + order).
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return writePGError(c, err)
	}

	// Map ke response
	out := make([]d.ClassScheduleResponse, 0, len(rows))
	for i := range rows {
		out = append(out, d.NewClassScheduleResponse(&rows[i]))
	}

	// Meta
	nextOffset := q.Offset + q.Limit
	hasMore := nextOffset < int(total)

	meta := fiber.Map{
		"limit":    q.Limit,
		"offset":   q.Offset,
		"count":    len(out),
		"total":    total,
		"has_more": hasMore,
		"next_offset": func() *int {
			if hasMore {
				return &nextOffset
			}
			return nil
		}(),
		"sort_by": q.SortBy,
		"order":   strings.ToLower(order),
	}

	return helper.JsonList(c, out, meta)
}
