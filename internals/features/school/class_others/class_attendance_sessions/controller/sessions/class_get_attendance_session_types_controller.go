package controller

import (
	"strings"

	dto "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/dto"
	model "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
	helper "madinahsalam_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/*
======================================================

	List
	GET /api/.../class-attendance-session-types
	Query:
	  - q          : optional, search by name / slug
	  - name       : optional, filter spesifik by name
	  - is_active  : optional, true/false
	  - mode       : optional, "full" (default) | "compact"
	  - page       : default 1
	  - per_page   : default 20, max 100

======================================================
*/
func (ctl *ClassAttendanceSessionTypeController) List(c *fiber.Ctx) error {
	// kalau ada helper lain yang butuh DB di Locals
	c.Locals("DB", ctl.DB)

	schoolID, err := getSchoolIDFromCtx(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// mode response: full (default) | compact
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	if mode != "" && mode != "full" && mode != "compact" {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid mode, must be 'full' or 'compact'")
	}

	isActive, err := parseBoolQuery(c, "is_active")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid is_active query param")
	}

	// paging standard
	paging := helper.ResolvePaging(c, 20, 100)

	q := strings.TrimSpace(c.Query("q"))
	name := strings.TrimSpace(c.Query("name")) // 🔍 filter khusus by name

	dbq := ctl.DB.
		WithContext(c.Context()).
		Model(&model.ClassAttendanceSessionTypeModel{}).
		Where("class_attendance_session_type_school_id = ?", schoolID)

	if isActive != nil {
		dbq = dbq.Where("class_attendance_session_type_is_active = ?", *isActive)
	}

	// filter by id (optional)
	idStr := strings.TrimSpace(c.Query("id"))
	if idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid id param")
		}
		dbq = dbq.Where("class_attendance_session_type_id = ?", id)
	}

	// 🔍 full-text sederhana: q → name + slug
	if q != "" {
		pattern := "%" + strings.ToLower(q) + "%"
		dbq = dbq.Where(
			"(LOWER(class_attendance_session_type_name) LIKE ? OR LOWER(class_attendance_session_type_slug) LIKE ?)",
			pattern, pattern,
		)
	}

	// 🔍 filter spesifik by name: ?name=
	if name != "" {
		pattern := "%" + strings.ToLower(name) + "%"
		dbq = dbq.Where(
			"LOWER(class_attendance_session_type_name) LIKE ?",
			pattern,
		)
	}

	// total untuk pagination
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to count attendance session types")
	}

	var rows []*model.ClassAttendanceSessionTypeModel
	if err := dbq.
		Order("class_attendance_session_type_sort_order ASC, class_attendance_session_type_name ASC").
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&rows).Error; err != nil {

		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch attendance session types")
	}

	pagination := helper.BuildPaginationFromPage(total, paging.Page, paging.PerPage)

	// =========================
	//  Response by mode
	// =========================
	if mode == "compact" {
		// compact: nggak ada field waktu, jadi nggak perlu dbtime
		return helper.JsonList(
			c,
			"attendance session types list (compact)",
			dto.NewClassAttendanceSessionTypeCompactDTOs(rows),
			pagination,
		)
	}

	// default: full → pakai versi timezone-aware
	return helper.JsonList(
		c,
		"attendance session types list",
		dto.NewClassAttendanceSessionTypeDTOsWithSchoolTime(c, rows),
		pagination,
	)
}
