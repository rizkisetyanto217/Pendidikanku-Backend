// file: internals/features/school/others/announcements/controller/announcement_list_controller.go
package controller

import (
	"strings"

	dto "masjidku_backend/internals/features/school/others/post/dto"
	model "masjidku_backend/internals/features/school/others/post/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /posts — list
// Query (opsional):
// kind, class_section_id, theme_id, is_active, is_published, date_from, date_to, q, limit, offset, sort_by, sort_dir
// id, slug (exact, bisa untuk single fetch via list)
func (ctl *PostController) List(c *fiber.Ctx) error {
	// auth: minimal member
	mid, err := resolveMasjidForRead(c, ctl.DB)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// filters
	kindStr := strings.TrimSpace(c.Query("kind"))
	classSectionStr := strings.TrimSpace(c.Query("class_section_id"))
	themeStr := strings.TrimSpace(c.Query("theme_id"))
	qStr := strings.TrimSpace(c.Query("q"))
	isActive := parseBoolPtr(c.Query("is_active"))
	isPublished := parseBoolPtr(c.Query("is_published"))
	idStr := strings.TrimSpace(c.Query("id"))
	slugStr := strings.TrimSpace(c.Query("slug"))
	limit := atoiOr(20, c.Query("limit"))
	offset := atoiOr(0, c.Query("offset"))
	sortBy := strings.TrimSpace(c.Query("sort_by"))   // created_at|date|title|published_at
	sortDir := strings.TrimSpace(c.Query("sort_dir")) // asc|desc

	q := ctl.DB.WithContext(c.Context()).
		Model(&model.Post{}).
		Where("post_masjid_id = ? AND post_deleted_at IS NULL", mid)

	// exact filters
	if idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
		}
		q = q.Where("post_id = ?", id)
	}
	if slugStr != "" {
		q = q.Where("post_slug = ?", slugStr)
	}
	if kindStr != "" {
		kind := model.PostKind(strings.ToLower(kindStr))
		switch kind {
		case model.PostKindAnnouncement, model.PostKindMaterial, model.PostKindPost, model.PostKindOther:
			q = q.Where("post_kind = ?", kind)
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "kind tidak valid (announcement|material|post|other)")
		}
	}
	if classSectionStr != "" {
		id, err := uuid.Parse(classSectionStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_section_id tidak valid")
		}
		q = q.Where("post_class_section_id = ?", id)
	}
	if themeStr != "" {
		id, err := uuid.Parse(themeStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "theme_id tidak valid")
		}
		q = q.Where("post_theme_id = ?", id)
	}
	if isActive != nil {
		q = q.Where("post_is_active = ?", *isActive)
	}
	if isPublished != nil {
		q = q.Where("post_is_published = ?", *isPublished)
	}

	// date range (post_date)
	if df, err := parseYMDLocal(c.Query("date_from")); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	} else if df != nil {
		q = q.Where("post_date >= ?", *df)
	}
	if dt, err := parseYMDLocal(c.Query("date_to")); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	} else if dt != nil {
		q = q.Where("post_date <= ?", *dt)
	}

	// keyword
	if qStr != "" {
		pat := "%" + strings.ToLower(qStr) + "%"
		q = q.Where(`(LOWER(post_title) ILIKE ? OR LOWER(post_content) ILIKE ? OR LOWER(COALESCE(post_excerpt,'')) ILIKE ?)`,
			pat, pat, pat)
	}

	// total
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// data
	var rows []model.Post
	if err := applySort(q, sortBy, sortDir).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonList(c, dto.FromModelsPost(rows), fiber.Map{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
