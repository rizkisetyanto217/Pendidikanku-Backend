// file: internals/features/lembaga/school_yayasans/schools/controller/list.go
package controller

import (
	d "schoolku_backend/internals/features/lembaga/school_yayasans/schools/dto"
	m "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"
	helper "schoolku_backend/internals/helpers"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GET / (list + filter + pagination)
func (ctl *SchoolProfileController) List(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))

	// 🔹 Resolve paging dari query (?page, ?per_page / ?limit)
	pgReq := helper.ResolvePaging(c, 20, 1000)

	dbq := ctl.DB.Model(&m.SchoolProfileModel{}).
		Where("school_profile_deleted_at IS NULL")

	// Full-text search (tsvector)
	if q != "" {
		dbq = dbq.Where("school_profile_search @@ plainto_tsquery('simple', ?)", q)
	}

	// Filters
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

	// Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// Data
	var rows []m.SchoolProfileModel
	if err := dbq.
		Order("school_profile_created_at DESC").
		Offset(pgReq.Offset).
		Limit(pgReq.Limit).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	items := make([]d.SchoolProfileResponse, 0, len(rows))
	for i := range rows {
		items = append(items, d.FromModelSchoolProfile(&rows[i]))
	}

	// 🔹 Build pagination (JsonList akan auto-isi count & per_page_options)
	pg := helper.BuildPaginationFromPage(total, pgReq.Page, pgReq.PerPage)
	return helper.JsonList(c, "ok", items, pg)
}
