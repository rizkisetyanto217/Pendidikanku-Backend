package controller

import (
	d "masjidku_backend/internals/features/lembaga/masjids/dto"
	m "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	"math"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GET / (list + filter + pagination)
func (ctl *MasjidProfileController) List(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 1000 {
		limit = 20
	}
	offset := (page - 1) * limit

	dbq := ctl.DB.Model(&m.MasjidProfileModel{}).Where("masjid_profile_deleted_at IS NULL")

	// Full-text search (tsvector)
	if q != "" {
		dbq = dbq.Where("masjid_profile_search @@ plainto_tsquery('simple', ?)", q)
	}

	// Filters
	if acc := strings.TrimSpace(c.Query("accreditation")); acc != "" {
		dbq = dbq.Where("masjid_profile_school_accreditation = ?", acc)
	}
	if ib := strings.TrimSpace(c.Query("is_boarding")); ib != "" {
		switch strings.ToLower(ib) {
		case "true", "1", "yes", "y":
			dbq = dbq.Where("masjid_profile_school_is_boarding = TRUE")
		case "false", "0", "no", "n":
			dbq = dbq.Where("masjid_profile_school_is_boarding = FALSE")
		}
	}
	if fyMin := strings.TrimSpace(c.Query("founded_year_min")); fyMin != "" {
		if v, err := strconv.Atoi(fyMin); err == nil {
			dbq = dbq.Where("masjid_profile_founded_year >= ?", v)
		}
	}
	if fyMax := strings.TrimSpace(c.Query("founded_year_max")); fyMax != "" {
		if v, err := strconv.Atoi(fyMax); err == nil {
			dbq = dbq.Where("masjid_profile_founded_year <= ?", v)
		}
	}

	// Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// Data
	var rows []m.MasjidProfileModel
	if err := dbq.
		Order("masjid_profile_created_at DESC").
		Offset(offset).Limit(limit).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	items := make([]d.MasjidProfileResponse, 0, len(rows))
	for i := range rows {
		items = append(items, d.FromModelMasjidProfile(&rows[i]))
	}

	// Pakai JsonList: data & pagination dipisah
	return helper.JsonList(c, items, fiber.Map{
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": int(math.Ceil(float64(total) / float64(limit))),
	})
}
