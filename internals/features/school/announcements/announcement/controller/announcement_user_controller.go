package controller

import (
	annModel "masjidku_backend/internals/features/school/announcements/announcement/model"
	annThemeModel "masjidku_backend/internals/features/school/announcements/announcement_thema/model"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	annDTO "masjidku_backend/internals/features/school/announcements/announcement/dto"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

// helper kecil untuk include: "urls", "theme"
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

// ===================== LIST =====================
// GET /admin/announcements
func (h *AnnouncementController) List(c *fiber.Ctx) error {
	// 1) Tenant scope
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	if len(masjidIDs) == 0 {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak ada akses masjid")
	}

	// 1a) Include flags (opsional)
	inc := parseInclude(c.Query("include"))
	includeURLs := inc["urls"] || inc["announcement_urls"] || inc["links"]
	includeTheme := inc["theme"] || inc["themes"] || inc["announcement_theme"]

	// 2) Pagination & default sort
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// 3) Parse DTO query
	var q annDTO.ListAnnouncementQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// 4) Base query (tenant-safe)
	tx := h.DB.Model(&annModel.AnnouncementModel{}).
		Where("announcement_masjid_id IN ?", masjidIDs)

	// 4a) Filter by IDs
	if raw := strings.TrimSpace(c.Query("id")); raw != "" {
		ids, e := parseUUIDsCSV(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id berisi UUID tidak valid")
		}
		if len(ids) > 0 {
			tx = tx.Where("announcement_id IN ?", ids)
		}
	} else if raw := strings.TrimSpace(c.Query("announcement_id")); raw != "" {
		ids, e := parseUUIDsCSV(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "announcement_id berisi UUID tidak valid")
		}
		if len(ids) > 0 {
			tx = tx.Where("announcement_id IN ?", ids)
		}
	}

	// 4b) Filter Theme
	if q.ThemeID != nil {
		if *q.ThemeID == uuid.Nil {
			tx = tx.Where("announcement_theme_id IS NULL")
		} else {
			tx = tx.Where("announcement_theme_id = ?", *q.ThemeID)
		}
	}

	// 4c) Filter Section vs Global
	includeGlobal := true
	if q.IncludeGlobal != nil {
		includeGlobal = *q.IncludeGlobal
	}
	onlyGlobal := q.OnlyGlobal != nil && *q.OnlyGlobal

	sectionIDs, secErr := parseUUIDsCSV(strings.TrimSpace(c.Query("section_ids")))
	if secErr != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "section_ids berisi UUID tidak valid")
	}
	switch {
	case onlyGlobal:
		tx = tx.Where("announcement_class_section_id IS NULL")
	case len(sectionIDs) > 0:
		if includeGlobal {
			tx = tx.Where("(announcement_class_section_id IN ? OR announcement_class_section_id IS NULL)", sectionIDs)
		} else {
			tx = tx.Where("announcement_class_section_id IN ?", sectionIDs)
		}
	case q.SectionID != nil:
		if *q.SectionID == uuid.Nil {
			tx = tx.Where("announcement_class_section_id IS NULL")
		} else if includeGlobal {
			tx = tx.Where("(announcement_class_section_id = ? OR announcement_class_section_id IS NULL)", *q.SectionID)
		} else {
			tx = tx.Where("announcement_class_section_id = ?", *q.SectionID)
		}
	}

	// 4d) Filter attachment
	if q.HasAttachment != nil {
		if *q.HasAttachment {
			tx = tx.Where("announcement_attachment_url IS NOT NULL AND announcement_attachment_url <> ''")
		} else {
			tx = tx.Where("(announcement_attachment_url IS NULL OR announcement_attachment_url = '')")
		}
	}

	// 4e) Filter is_active
	if q.IsActive != nil {
		tx = tx.Where("announcement_is_active = ?", *q.IsActive)
	}

	// 4f) Date range
	parseDate := func(s string) (time.Time, error) {
		return time.Parse("2006-01-02", strings.TrimSpace(s))
	}
	if q.DateFrom != nil && strings.TrimSpace(*q.DateFrom) != "" {
		t, e := parseDate(*q.DateFrom)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
		}
		tx = tx.Where("announcement_date >= ?", t)
	}
	if q.DateTo != nil && strings.TrimSpace(*q.DateTo) != "" {
		t, e := parseDate(*q.DateTo)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
		}
		tx = tx.Where("announcement_date <= ?", t)
	}

	// 5) Total
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// 6) Sorting
	sortKey := "date_desc"
	if q.Sort != nil {
		sortKey = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	sortMap := map[string]string{
		"date_desc":       "announcement_date DESC",
		"date_asc":        "announcement_date ASC",
		"created_at_desc": "announcement_created_at DESC",
		"created_at_asc":  "announcement_created_at ASC",
		"title_asc":       "announcement_title ASC",
		"title_desc":      "announcement_title DESC",
	}
	orderExpr, ok := sortMap[sortKey]
	if !ok {
		orderExpr = sortMap["date_desc"]
	}

	// 7) Fetch rows
	var rows []annModel.AnnouncementModel
	if err := tx.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 8) Batch-load Themes (HANYA jika include=theme)
	if includeTheme {
		themeIDs := make([]uuid.UUID, 0, len(rows))
		seenTheme := make(map[uuid.UUID]struct{}, len(rows))
		for i := range rows {
			if rows[i].AnnouncementThemeID == nil {
				continue
			}
			id := *rows[i].AnnouncementThemeID
			if id == uuid.Nil {
				continue
			}
			if _, ok := seenTheme[id]; !ok {
				seenTheme[id] = struct{}{}
				themeIDs = append(themeIDs, id)
			}
		}
		if len(themeIDs) > 0 {
			var themes []annThemeModel.AnnouncementThemeModel
			if err := h.DB.
				Select("announcement_themes_id, announcement_themes_masjid_id, announcement_themes_name, announcement_themes_color").
				Where("announcement_themes_deleted_at IS NULL").
				Where("announcement_themes_id IN ?", themeIDs).
				Find(&themes).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat tema")
			}
			tmap := make(map[uuid.UUID]*annThemeModel.AnnouncementThemeModel, len(themes))
			for i := range themes {
				t := themes[i] // copy
				tmap[t.AnnouncementThemesID] = &t
			}
			for i := range rows {
				if rows[i].AnnouncementThemeID != nil {
					if th := tmap[*rows[i].AnnouncementThemeID]; th != nil {
						rows[i].Theme = th // hanya inject jika diminta
					}
				}
			}
		}
	}


	// 8b) Batch-load URLs (HANYA jika include=urls)
	urlMap := make(map[uuid.UUID][]*annDTO.AnnouncementURLLite, len(rows))
	if includeURLs {
		annIDs := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			annIDs = append(annIDs, rows[i].AnnouncementID)
		}
		if len(annIDs) > 0 {
			var urlRows []annModel.AnnouncementURLModel
			if err := h.DB.
				// Select opsional, biar jelas kolom yang dipakai
				Select("announcement_url_id, announcement_url_announcement_id, announcement_url_label, announcement_url_href").
				Where("announcement_url_deleted_at IS NULL").
				Where("announcement_url_announcement_id IN ?", annIDs).
				Order("announcement_url_created_at DESC").
				Find(&urlRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat URL")
			}
			for i := range urlRows {
				u := urlRows[i]
				urlMap[u.AnnouncementURLAnnouncementID] = append(
					urlMap[u.AnnouncementURLAnnouncementID],
					&annDTO.AnnouncementURLLite{
						ID:             u.AnnouncementURLID,
						AnnouncementID: u.AnnouncementURLAnnouncementID, // ⬅️ set field baru
						Label:          u.AnnouncementURLLabel,
						Href:           u.AnnouncementURLHref,
					},
				)
			}
		}
	}


	// 9) Map ke DTO + inject Urls (conditional)
	items := make([]*annDTO.AnnouncementResponse, 0, len(rows))
	for i := range rows {
		if resp := annDTO.NewAnnouncementResponse(&rows[i]); resp != nil {
			if includeURLs {
				if urls := urlMap[rows[i].AnnouncementID]; len(urls) > 0 {
					resp.Urls = urls
				}
			}
			items = append(items, resp)
		}
	}

	// 10) Response
	return helper.JsonList(c, items, helper.BuildMeta(total, p))
}


