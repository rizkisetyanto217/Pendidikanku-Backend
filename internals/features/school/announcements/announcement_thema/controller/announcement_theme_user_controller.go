// controller/announcement_theme_list.go
package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	anModel "masjidku_backend/internals/features/school/announcements/announcement/model"
	annDTO "masjidku_backend/internals/features/school/announcements/announcement_thema/dto"
	annModel "masjidku_backend/internals/features/school/announcements/announcement_thema/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===================== helpers (local) ===================== */

func parseInclude(raw string) map[string]bool {
	m := map[string]bool{}
	if raw == "" {
		return m
	}
	for _, part := range strings.Split(strings.ToLower(strings.TrimSpace(raw)), ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			m[p] = true
		}
	}
	return m
}

// NOTE: sesuaikan nama struct model announcement & url sesuai package kamu.
// Di sini diasumsikan:
//
// type AnnouncementModel struct { ... fields eksplisit announcement_* ... }
// func (a AnnouncementModel) GetID() uuid.UUID { return a.AnnouncementID }
// func (a AnnouncementModel) GetThemeID() *uuid.UUID { return pointer dari a.AnnouncementThemeID }
// func (a AnnouncementModel) GetThemeIDVal() uuid.UUID { return *a.AnnouncementThemeID } // hati2 nil, hanya pakai bila non-nil
// func (a AnnouncementModel) GetSectionID() *uuid.UUID { return a.AnnouncementClassSectionID }
// func (a AnnouncementModel) GetCreatedByTeacherID() *uuid.UUID { return a.AnnouncementCreatedByTeacherID }
//
// type AnnouncementURLModel struct { ... fields eksplisit announcement_url_* ... }
// func (u AnnouncementURLModel) GetAnnouncementID() uuid.UUID { return u.AnnouncementURLAnnouncementID }
//
// Jika belum ada getter seperti di komentar, kamu bisa akses langsung field eksplisitnya.

func (h *AnnouncementThemeController) fetchAnnouncementsForThemes(
	db *gorm.DB,
	masjidID uuid.UUID,
	themeIDs []uuid.UUID,
	activeOnly bool,
	sectionID *uuid.UUID,
	dateFromStr, dateToStr string,
	sortBy, order string,
	limitPerTheme int,
	withURLs bool,
) (map[uuid.UUID][]anModel.AnnouncementModel, map[uuid.UUID][]anModel.AnnouncementURLModel, error) {

	if len(themeIDs) == 0 {
		return map[uuid.UUID][]anModel.AnnouncementModel{}, nil, nil
	}

	q := db.Model(&anModel.AnnouncementModel{}).
		Where("announcement_masjid_id = ?", masjidID).
		Where("announcement_deleted_at IS NULL").
		Where("announcement_theme_id IN ?", themeIDs)

	if activeOnly {
		q = q.Where("announcement_is_active = TRUE")
	}
	if sectionID != nil {
		q = q.Where("announcement_class_section_id = ?", *sectionID)
	}
	// date range
	if dateFromStr != "" {
		if t, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			q = q.Where("announcement_date >= ?", t)
		}
	}
	if dateToStr != "" {
		if t, err := time.Parse("2006-01-02", dateToStr); err == nil {
			q = q.Where("announcement_date <= ?", t)
		}
	}

	orderCol := "announcement_date"
	if strings.ToLower(sortBy) == "created_at" {
		orderCol = "announcement_created_at"
	}
	orderDir := "DESC"
	if strings.ToLower(order) == "asc" {
		orderDir = "ASC"
	}
	q = q.Order(orderCol + " " + orderDir)

	var all []anModel.AnnouncementModel
	if err := q.Find(&all).Error; err != nil {
		return nil, nil, err
	}

	// Bucket by theme + apply limit per theme
	byTheme := make(map[uuid.UUID][]anModel.AnnouncementModel, len(themeIDs))
	counts := make(map[uuid.UUID]int, len(themeIDs))
	for i := range all {
		tidPtr := all[i].AnnouncementThemeID // *uuid.UUID langsung dari model
		if tidPtr == nil {
			continue // skip kalau announcement tidak punya theme_id
		}
		tid := *tidPtr
		if limitPerTheme > 0 && counts[tid] >= limitPerTheme {
			continue
		}
		byTheme[tid] = append(byTheme[tid], all[i])
		counts[tid]++
	}

	// URLs (opsional)
	var urlMap map[uuid.UUID][]anModel.AnnouncementURLModel
	if withURLs && len(all) > 0 {
		ids := make([]uuid.UUID, 0, len(all))
		for i := range all {
			ids = append(ids, all[i].AnnouncementID)
		}
		var urls []anModel.AnnouncementURLModel
		if err := db.Model(&anModel.AnnouncementURLModel{}).
			Where("announcement_url_masjid_id = ?", masjidID).
			Where("announcement_url_deleted_at IS NULL").
			Where("announcement_url_announcement_id IN ?", ids).
			Order("announcement_url_created_at DESC").
			Find(&urls).Error; err != nil {
			return nil, nil, err
		}
		urlMap = make(map[uuid.UUID][]anModel.AnnouncementURLModel, len(ids))
		for i := range urls {
			aID := urls[i].AnnouncementURLAnnouncementID
			urlMap[aID] = append(urlMap[aID], urls[i])
		}
	}

	return byTheme, urlMap, nil
}

func toAnnouncementEmbeds(
	src []anModel.AnnouncementModel,
	urlMap map[uuid.UUID][]anModel.AnnouncementURLModel,
	withURLs bool,
) []annDTO.AnnouncementResponseEmbed {
	if len(src) == 0 {
		return nil
	}
	out := make([]annDTO.AnnouncementResponseEmbed, 0, len(src))

	for i := range src {
		a := src[i]

		// convert gorm.DeletedAt -> *time.Time (DTO kamu pakai *time.Time)
		var deletedAtPtr *time.Time
		if a.AnnouncementDeletedAt.Valid {
			t := a.AnnouncementDeletedAt.Time
			deletedAtPtr = &t
		}

		item := annDTO.AnnouncementResponseEmbed{
			AnnouncementID:                 a.AnnouncementID,
			AnnouncementMasjidID:           a.AnnouncementMasjidID,
			AnnouncementTitle:              a.AnnouncementTitle,
			AnnouncementDate:               a.AnnouncementDate,
			AnnouncementContent:            a.AnnouncementContent,
			AnnouncementIsActive:           a.AnnouncementIsActive,
			AnnouncementCreatedByTeacherID: a.AnnouncementCreatedByTeacherID,
			AnnouncementCreatedAt:          a.AnnouncementCreatedAt,
			AnnouncementUpdatedAt:          a.AnnouncementUpdatedAt,
			AnnouncementDeletedAt:          deletedAtPtr,                // ✅ sudah *time.Time
			AnnouncementThemeID:            a.AnnouncementThemeID,       // *uuid.UUID
			AnnouncementClassSectionID:     a.AnnouncementClassSectionID, // *uuid.UUID
		}

		if withURLs && urlMap != nil {
			if urls, ok := urlMap[a.AnnouncementID]; ok && len(urls) > 0 {
				item.AnnouncementURLs = make([]annDTO.AnnouncementURLResponseEmbed, 0, len(urls))
				for j := range urls {
					u := urls[j]
					item.AnnouncementURLs = append(item.AnnouncementURLs, annDTO.AnnouncementURLResponseEmbed{
						AnnouncementURLID:                 u.AnnouncementURLID,
						AnnouncementURLLabel:              u.AnnouncementURLLabel,
						AnnouncementURLHref:               u.AnnouncementURLHref,
						AnnouncementURLTrashURL:           u.AnnouncementURLTrashURL,
						AnnouncementURLDeletePendingUntil: u.AnnouncementURLDeletePendingUntil,
						AnnouncementURLCreatedAt:          u.AnnouncementURLCreatedAt,
						AnnouncementURLUpdatedAt:          u.AnnouncementURLUpdatedAt,
						AnnouncementURLDeletedAt:          u.AnnouncementURLDeletedAt, // kemungkinan sudah *time.Time
					})
				}
			}
		}

		out = append(out, item)
	}
	return out
}

/* ===================== LIST (updated) ===================== */

// GET /admin/announcement-themes
// Opsional:
//   ?announcement_theme_id=<uuid>  (atau ?id=<uuid> / /admin/announcement-themes/:id)
//   ?name=..., ?slug=...
//   ?is_active=true|false  (alias: ?active_only=true|false)
//   Search: ?q=... (akan ILIKE ke name; tetap ikut pagination)
//   Pagination: ?page=1&per_page=25 (atau limit), sort_by=created_at|updated_at|name|slug, order=asc|desc
//   Include (opsional):
//     include=announcements[,announcements.urls]
//   Filter utk announcements (hanya saat include announcements):
//     ann_active_only=true|false (default true)
//     ann_section_id=<uuid>
//     ann_date_from=YYYY-MM-DD
//     ann_date_to=YYYY-MM-DD
//     ann_limit_per_theme=<n> (default 3)
//     ann_sort_by=date|created_at (default date)
//     ann_order=asc|desc (default desc)
func (h *AnnouncementThemeController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// include flags
	inc := parseInclude(c.Query("include"))
	incAnn := inc["announcements"] || inc["announcement"] || inc["ann"]
	incAnnURLs := inc["announcements.urls"] || inc["ann.urls"]

	// parse ann_* filters (only used if incAnn)
	annActiveOnly := true
	if v := strings.TrimSpace(c.Query("ann_active_only")); v != "" {
		if b, e := strconv.ParseBool(v); e == nil {
			annActiveOnly = b
		}
	}
	var annSectionID *uuid.UUID
	if s := strings.TrimSpace(c.Query("ann_section_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "ann_section_id tidak valid")
		}
		annSectionID = &id
	}
	annDateFrom := strings.TrimSpace(c.Query("ann_date_from"))
	annDateTo := strings.TrimSpace(c.Query("ann_date_to"))
	annLimitPerTheme := 3
	if v := strings.TrimSpace(c.Query("ann_limit_per_theme")); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			annLimitPerTheme = n
		}
	}
	annSortBy := strings.ToLower(strings.TrimSpace(c.Query("ann_sort_by")))
	if annSortBy != "created_at" && annSortBy != "date" {
		annSortBy = "date"
	}
	annOrder := strings.ToLower(strings.TrimSpace(c.Query("ann_order")))
	if annOrder != "asc" && annOrder != "desc" {
		annOrder = "desc"
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

		resp := annDTO.NewAnnouncementThemeResponse(&one)

		// include announcements (detail mode)
		if incAnn {
			annsByTheme, urlMap, err := h.fetchAnnouncementsForThemes(
				h.DB, masjidID, []uuid.UUID{one.AnnouncementThemesID},
				annActiveOnly, annSectionID, annDateFrom, annDateTo,
				annSortBy, annOrder, annLimitPerTheme, incAnnURLs,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}
			resp.Announcements = toAnnouncementEmbeds(annsByTheme[one.AnnouncementThemesID], urlMap, incAnnURLs)
		}

		return helper.JsonOK(c, "OK", resp)
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

	// fetch themes
	var rows []annModel.AnnouncementThemeModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data tema")
	}

	// map DTO
	resp := make([]*annDTO.AnnouncementThemeResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, annDTO.NewAnnouncementThemeResponse(&rows[i]))
	}

	// include announcements (list mode)
	if incAnn && len(rows) > 0 {
		ids := make([]uuid.UUID, 0, len(rows))
		indexByID := make(map[uuid.UUID]int, len(rows))
		for i := range rows {
			tid := rows[i].AnnouncementThemesID
			ids = append(ids, tid)
			indexByID[tid] = i
		}

		annsByTheme, urlMap, err := h.fetchAnnouncementsForThemes(
			h.DB, masjidID, ids,
			annActiveOnly, annSectionID, annDateFrom, annDateTo,
			annSortBy, annOrder, annLimitPerTheme, incAnnURLs,
		)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		for themeID, list := range annsByTheme {
			if pos, ok := indexByID[themeID]; ok && pos >= 0 && pos < len(resp) {
				resp[pos].Announcements = toAnnouncementEmbeds(list, urlMap, incAnnURLs)
			}
		}
	}

	// meta
	return helper.JsonList(c, resp, helper.BuildMeta(total, p))
}
