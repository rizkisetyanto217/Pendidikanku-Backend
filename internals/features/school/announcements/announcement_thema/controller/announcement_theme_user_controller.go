package controller

import (
	"errors"
	annDTO "masjidku_backend/internals/features/school/announcements/announcement_thema/dto"
	annModel "masjidku_backend/internals/features/school/announcements/announcement_thema/model"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strconv"
	"strings"

	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GET /admin/announcement-themes
// Opsional:
//   ?announcement_theme_id=<uuid>  (atau ?id=<uuid> / /admin/announcement-themes/:id)
//   ?name=..., ?slug=...
//   ?is_active=true|false  (alias: ?active_only=true|false)
//   Search: ?q=... (akan ILIKE ke name; tetap ikut pagination)
//   Pagination: ?page=1&per_page=25 (atau limit), sort_by=created_at|updated_at|name|slug, order=asc|desc
func (h *AnnouncementThemeController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// parsing ringan dari DTO existing (name/slug/is_active)
	var q annDTO.ListAnnouncementThemeQuery
	_ = c.QueryParser(&q)

	// tambahan untuk search dan alias active_only
	searchQ := strings.TrimSpace(c.Query("q"))
	activeOnlyStr := strings.TrimSpace(c.Query("active_only"))
	if q.IsActive == nil && activeOnlyStr != "" {
		if b, e := strconv.ParseBool(activeOnlyStr); e == nil {
			q.IsActive = &b
		}
	}

	// pagination + sorting (pakai helper)
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// whitelist kolom sort
	allowed := map[string]string{
		"created_at": "announcement_themes_created_at",
		"updated_at": "announcement_themes_updated_at",
		"name":       "announcement_themes_name",
		"slug":       "announcement_themes_slug",
	}
	orderCol := allowed["created_at"]
	if col, ok := allowed[strings.ToLower(p.SortBy)]; ok {
		orderCol = col
	}
	orderDir := "DESC"
	if strings.ToLower(p.SortOrder) == "asc" {
		orderDir = "ASC"
	}

	// base scope
	tx := h.DB.Model(&annModel.AnnouncementThemeModel{}).
		Where("announcement_themes_masjid_id = ?", masjidID).
		Where("announcement_themes_deleted_at IS NULL")

	// filter by id (query/path) — PRIORITAS: jika ada id, langsung ambil detail
	idStr := strings.TrimSpace(c.Query("announcement_theme_id"))
	if idStr == "" {
		idStr = strings.TrimSpace(c.Query("id"))
	}
	if idStr == "" {
		idStr = strings.TrimSpace(c.Params("id"))
	}
	if idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "announcement_theme_id tidak valid")
		}

		// auto-detect nama kolom id
		idCol := "announcement_theme_id"
		if !hasColumn(h.DB, "announcement_themes", idCol) && hasColumn(h.DB, "announcement_themes", "announcement_themes_id") {
			idCol = "announcement_themes_id"
		}

		var one annModel.AnnouncementThemeModel
		if err := tx.Where(idCol+" = ?", id).Take(&one).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "Tema tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data tema")
		}
		return helper.JsonOK(c, "OK", annDTO.NewAnnouncementThemeResponse(&one))
	}

	// filters lain
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		tx = tx.Where("announcement_themes_name ILIKE ?", "%"+strings.TrimSpace(*q.Name)+"%")
	}
	if q.Slug != nil && strings.TrimSpace(*q.Slug) != "" {
		tx = tx.Where("announcement_themes_slug = ?", strings.TrimSpace(*q.Slug))
	}
	if q.IsActive != nil {
		tx = tx.Where("announcement_themes_is_active = ?", *q.IsActive)
	}
	// search sederhana (ILIKE name)
	if searchQ != "" {
		tx = tx.Where("announcement_themes_name ILIKE ?", "%"+strings.ToLower(searchQ)+"%")
	}

	// hitung total (sebelum limit/offset)
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data tema")
	}

	// order & paging
	tx = tx.Order(orderCol + " " + orderDir)
	if !p.All {
		tx = tx.Limit(p.Limit()).Offset(p.Offset())
	}

	// fetch
	var rows []annModel.AnnouncementThemeModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data tema")
	}

	// map DTO
	resp := make([]*annDTO.AnnouncementThemeResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, annDTO.NewAnnouncementThemeResponse(&rows[i]))
	}

	// meta
	return helper.JsonList(c, resp, helper.BuildMeta(total, p))
}
