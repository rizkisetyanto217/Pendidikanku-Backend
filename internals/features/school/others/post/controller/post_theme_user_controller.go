// controller/announcement_theme_list.go
package controller

import (
	"net/http"
	"strings"

	dto "masjidku_backend/internals/features/school/others/post/dto"
	pmodel "masjidku_backend/internals/features/school/others/post/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================================================
   LIST (PUBLIC)
   GET /post-themes
   Query: masjid context via resolver (slug/id), kind, parent_id, is_active, q, sort_by, order, page/per_page
========================================================= */

func (ctl *PostThemeController) List(c *fiber.Ctx) error {
	// biar resolver slug→id bisa akses DB
	c.Locals("DB", ctl.DB)

	// Resolve masjid (tanpa auth ketat — publik)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, http.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	} else {
		return helperAuth.ErrMasjidContextMissing
	}

	// Filters
	var q dto.ListPostThemesQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Query tidak valid")
	}
	q.Normalize()

	// Pagination + Sorting via helper
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	orderMap := map[string]string{
		"name":       "post_theme_name",
		"slug":       "post_theme_slug",
		"created_at": "post_theme_created_at",
		"updated_at": "post_theme_updated_at",
	}
	orderClause, _ := p.SafeOrderClause(orderMap, "created_at")
	// GORM .Order() tidak butuh "ORDER BY "
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	tx := ctl.DB.WithContext(c.Context()).
		Model(&pmodel.PostThemeModel{}).
		Where("post_theme_masjid_id = ? AND post_theme_deleted_at IS NULL", masjidID)

	// apply filters
	if q.Kind != nil && strings.TrimSpace(*q.Kind) != "" {
		tx = tx.Where("post_theme_kind = ?", strings.ToLower(strings.TrimSpace(*q.Kind)))
	}
	if q.ParentID != nil {
		tx = tx.Where("post_theme_parent_id = ?", *q.ParentID)
	}
	if q.IsActive != nil {
		tx = tx.Where("post_theme_is_active = ?", *q.IsActive)
	}
	if q.SearchName != nil {
		kw := "%" + strings.ToLower(*q.SearchName) + "%"
		tx = tx.Where("LOWER(post_theme_name) LIKE ?", kw)
	}

	// count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return writeDBErr(c, err)
	}

	// data
	var rows []pmodel.PostThemeModel
	if err := tx.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return writeDBErr(c, err)
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, dto.FromModels(rows), meta)
}
