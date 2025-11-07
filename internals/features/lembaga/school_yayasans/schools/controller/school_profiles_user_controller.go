// file: internals/features/lembaga/school_yayasans/schools/controller/list.go
package controller

import (
	d "schoolku_backend/internals/features/lembaga/school_yayasans/schools/dto"
	m "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"
	helper "schoolku_backend/internals/helpers"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET / (list + filter + pagination)
func (ctl *SchoolProfileController) List(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))

	// 🔹 paging (?page, ?per_page / ?limit)
	pgReq := helper.ResolvePaging(c, 20, 1000)

	// === filter by primary id ===
	id := strings.TrimSpace(c.Query("id"))        // UUID
	idsParam := strings.TrimSpace(c.Query("ids")) // comma-separated UUID

	// === filter by SCHOOL (canonical) + alias MASJID (deprecated) ===
	// Canonical
	schoolID := strings.TrimSpace(c.Query("school_id"))
	schoolIDsParam := strings.TrimSpace(c.Query("school_ids"))

	const (
		colID     = "school_profile_id"        // PK profile
		colSchool = "school_profile_school_id" // FK ke schools (canonical)
	)

	dbq := ctl.DB.Model(&m.SchoolProfileModel{}).
		Where("school_profile_deleted_at IS NULL")

	// ---- filter by profile id (single) ----
	if id != "" {
		if _, err := uuid.Parse(id); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter id tidak valid (harus UUID)")
		}
		dbq = dbq.Where(colID+" = ?", id)
	}

	// ---- filter by profile ids (multi) ----
	if idsParam != "" {
		raw := strings.Split(idsParam, ",")
		ids := make([]string, 0, len(raw))
		for _, s := range raw {
			v := strings.TrimSpace(s)
			if v == "" {
				continue
			}
			if _, err := uuid.Parse(v); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ids mengandung UUID tidak valid")
			}
			ids = append(ids, v)
		}
		if len(ids) > 0 {
			dbq = dbq.Where(colID+" IN ?", ids)
		}
	}

	// ---- filter by SCHOOL (single) ----
	if schoolID != "" {
		if _, err := uuid.Parse(schoolID); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter school_id tidak valid (harus UUID)")
		}
		dbq = dbq.Where(colSchool+" = ?", schoolID)
	}

	// ---- filter by SCHOOL (multi) ----
	if schoolIDsParam != "" {
		raw := strings.Split(schoolIDsParam, ",")
		sIDs := make([]string, 0, len(raw))
		for _, s := range raw {
			v := strings.TrimSpace(s)
			if v == "" {
				continue
			}
			if _, err := uuid.Parse(v); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Parameter school_ids mengandung UUID tidak valid")
			}
			sIDs = append(sIDs, v)
		}
		if len(sIDs) > 0 {
			dbq = dbq.Where(colSchool+" IN ?", sIDs)
		}
	}

	// ---- Full-text search (tsvector) ----
	if q != "" {
		dbq = dbq.Where("school_profile_search @@ plainto_tsquery('simple', ?)", q)
	}

	// ---- Filters tambahan ----
	if acc := strings.TrimSpace(c.Query("accreditation")); acc != "" {
		dbq = dbq.Where("school_profile_school_accreditation = ?", acc)
	}
	if ib := strings.TrimSpace(c.Query("is_boarding")); ib != "" {
		switch strings.ToLower(ib) {
		case "true", "1", "yes", "y":
			dbq = dbq.Where("school_profile_school_is_boarding = TRUE")
		case "false", "0", "no", "n":
			dbq = dbq.Where("school_profile_school_is_boarding = FALSE")
		}
	}
	if fyMin := strings.TrimSpace(c.Query("founded_year_min")); fyMin != "" {
		if v, err := strconv.Atoi(fyMin); err == nil {
			dbq = dbq.Where("school_profile_founded_year >= ?", v)
		}
	}
	if fyMax := strings.TrimSpace(c.Query("founded_year_max")); fyMax != "" {
		if v, err := strconv.Atoi(fyMax); err == nil {
			dbq = dbq.Where("school_profile_founded_year <= ?", v)
		}
	}

	// ---- Count ----
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// ---- Data ----
	var rows []m.SchoolProfileModel
	if err := dbq.
		Order("school_profile_created_at DESC").
		Offset(pgReq.Offset).
		Limit(pgReq.Limit).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// ---- DTO ----
	items := make([]d.SchoolProfileResponse, 0, len(rows))
	for i := range rows {
		items = append(items, d.FromModelSchoolProfile(&rows[i]))
	}

	// ---- Pagination ----
	pg := helper.BuildPaginationFromPage(total, pgReq.Page, pgReq.PerPage)
	return helper.JsonList(c, "ok", items, pg)
}
