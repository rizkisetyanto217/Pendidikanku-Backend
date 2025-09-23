package controller

import (
	"net/http"
	"strings"
	"time"

	d "masjidku_backend/internals/features/school/academics/schedules/dto"
	m "masjidku_backend/internals/features/school/academics/schedules/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// =========================
// List (with filters/sort/pagination)
// =========================
func (ctl *ClassScheduleController) List(c *fiber.Ctx) error {
	// untuk resolver slug→ID lain bila diperlukan
	c.Locals("DB", ctl.DB)

	var q d.ListClassScheduleQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// ===== Tetapkan masjid_id aktif =====
	// Prefer masjid context eksplisit (wajib DKM/Admin di masjid tsb).
	// Jika tak ada context → fallback ke token (teacher-aware).
	var masjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		masjidID = id
	} else if act, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && act != uuid.Nil {
		masjidID = act
	} else {
		return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak ditemukan")
	}

	// ===== Pagination clamp =====
	limit := 50
	if q.Limit != nil {
		if *q.Limit <= 0 {
			limit = 50
		} else if *q.Limit > 200 {
			limit = 200
		} else {
			limit = *q.Limit
		}
	}
	offset := 0
	if q.Offset != nil && *q.Offset > 0 {
		offset = *q.Offset
	}

	// ===== Sorting (default: created_at_desc) =====
	orderExpr := "class_schedules_created_at DESC"
	if q.Sort != nil {
		switch strings.ToLower(strings.TrimSpace(*q.Sort)) {
		case "start_date_asc":
			orderExpr = "class_schedules_start_date ASC, class_schedules_end_date ASC, class_schedules_created_at DESC"
		case "start_date_desc":
			orderExpr = "class_schedules_start_date DESC, class_schedules_end_date DESC, class_schedules_created_at DESC"
		case "end_date_asc":
			orderExpr = "class_schedules_end_date ASC, class_schedules_start_date ASC, class_schedules_created_at DESC"
		case "end_date_desc":
			orderExpr = "class_schedules_end_date DESC, class_schedules_start_date DESC, class_schedules_created_at DESC"
		case "created_at_asc":
			orderExpr = "class_schedules_created_at ASC"
		case "created_at_desc":
			orderExpr = "class_schedules_created_at DESC"
		case "updated_at_asc":
			orderExpr = "class_schedules_updated_at ASC NULLS LAST"
		case "updated_at_desc":
			orderExpr = "class_schedules_updated_at DESC NULLS LAST"
		}
	}

	// ===== Build base query =====
	tx := ctl.DB.Model(&m.ClassScheduleModel{})

	// WithDeleted? default: hanya yang alive.
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_schedules_deleted_at IS NULL")
	}

	// Tenant
	tx = tx.Where("class_schedules_masjid_id = ?", masjidID)

	// Status
	if q.Status != nil {
		s := strings.ToLower(strings.TrimSpace(*q.Status))
		switch s {
		case "scheduled", "ongoing", "completed", "canceled":
			tx = tx.Where("class_schedules_status = ?", s)
		default:
			return helper.JsonError(c, http.StatusBadRequest, "status invalid")
		}
	}

	// IsActive
	if q.IsActive != nil {
		tx = tx.Where("class_schedules_is_active = ?", *q.IsActive)
	}

	// Rentang tanggal (filter overlap dengan [from,to])
	// - jika from ada: ambil yang end_date >= from
	// - jika to ada:   ambil yang start_date <= to
	if q.DateFrom != nil && strings.TrimSpace(*q.DateFrom) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*q.DateFrom)); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "date_from invalid (YYYY-MM-DD)")
		}
		tx = tx.Where("class_schedules_end_date >= ?::date", strings.TrimSpace(*q.DateFrom))
	}
	if q.DateTo != nil && strings.TrimSpace(*q.DateTo) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*q.DateTo)); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "date_to invalid (YYYY-MM-DD)")
		}
		tx = tx.Where("class_schedules_start_date <= ?::date", strings.TrimSpace(*q.DateTo))
	}

	// Q: cari di slug (case-insensitive)
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		term := strings.ToLower(strings.TrimSpace(*q.Q))
		tx = tx.Where("class_schedules_slug IS NOT NULL AND lower(class_schedules_slug) LIKE ?", "%"+term+"%")
	}

	// ===== Count =====
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// ===== Fetch =====
	var rows []m.ClassScheduleModel
	if err := tx.
		Order(orderExpr).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// Map ke response DTO
	out := make([]d.ClassScheduleResponse, 0, len(rows))
	for i := range rows {
		out = append(out, d.FromModel(rows[i]))
	}

	// Meta sederhana
	meta := fiber.Map{
		"limit":  limit,
		"offset": offset,
		"total":  total,
	}

	return helper.JsonList(c, out, meta)
}
