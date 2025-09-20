// file: internals/features/school/others/announcements/controller/announcement_list_controller.go
package controller

import (
	"strings"
	"time"

	annDTO "masjidku_backend/internals/features/school/others/announcements/dto"
	annModel "masjidku_backend/internals/features/school/others/announcements/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================
   Helpers
========================================= */

// Resolve masjid scope utk LIST:
// - Jika ada context (path/header/cookie/query/host): validasi akses (DKM atau minimal member), return 1 ID
// - Jika tidak ada: fallback ke semua masjid_ids dari token (multi-tenant)
func resolveMasjidIDsForList(c *fiber.Ctx, db *gorm.DB) ([]uuid.UUID, error) {
	// Inject DB agar helper slug→id bisa jalan
	c.Locals("DB", db)

	// 1) Coba context eksplisit
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		var masjidID uuid.UUID
		if mc.ID != uuid.Nil {
			masjidID = mc.ID
		} else {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil || id == uuid.Nil {
				return nil, fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			masjidID = id
		}
		// Minimal member
		if !helperAuth.UserHasMasjid(c, masjidID) {
			return nil, helperAuth.ErrMasjidContextForbidden
		}
		return []uuid.UUID{masjidID}, nil
	}

	// 2) Fallback: semua masjid_ids dari token
	ids, e := helperAuth.GetMasjidIDsFromToken(c)
	if e != nil {
		return nil, helper.JsonError(c, fiber.StatusUnauthorized, e.Error())
	}
	if len(ids) == 0 {
		return nil, fiber.NewError(fiber.StatusForbidden, "Tidak ada akses masjid")
	}
	return ids, nil
}

/* =========================================
   LIST
========================================= */

// GET /admin/announcements
func (h *AnnouncementController) List(c *fiber.Ctx) error {
	// 1) Tenant scope (pakai helper context)
	masjidIDs, err := resolveMasjidIDsForList(c, h.DB)
	if err != nil {
		// err bisa dari helper; bungkus ke JSON standar
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 2) Pagination & default sort
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// 3) Parse DTO query (+ include langsung dari DTO) // <<< changed
	var q annDTO.ListAnnouncementQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	includeTheme := q.WantTheme() // <<< changed
	includeURLs := q.WantURLs()   // <<< changed

	// 4) Base query (tenant-safe)
	tx := h.DB.Model(&annModel.AnnouncementModel{}).
		Where("announcement_masjid_id IN ?", masjidIDs)

	// 4a) Filter by IDs
	if raw := strings.TrimSpace(c.Query("id")); raw != "" {
		if ids, e := parseUUIDsCSV(raw); e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id berisi UUID tidak valid")
		} else if len(ids) > 0 {
			tx = tx.Where("announcement_id IN ?", ids)
		}
	} else if raw := strings.TrimSpace(c.Query("announcement_id")); raw != "" {
		if ids, e := parseUUIDsCSV(raw); e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "announcement_id berisi UUID tidak valid")
		} else if len(ids) > 0 {
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

	// 4c) Section vs Global
	includeGlobal := true
	if q.IncludeGlobal != nil {
		includeGlobal = *q.IncludeGlobal
	}
	onlyGlobal := q.OnlyGlobal != nil && *q.OnlyGlobal

	if rawSecs := strings.TrimSpace(c.Query("section_ids")); rawSecs != "" {
		sectionIDs, secErr := parseUUIDsCSV(rawSecs)
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
		}
	} else if q.SectionID != nil {
		if *q.SectionID == uuid.Nil {
			tx = tx.Where("announcement_class_section_id IS NULL")
		} else if includeGlobal {
			tx = tx.Where("(announcement_class_section_id = ? OR announcement_class_section_id IS NULL)", *q.SectionID)
		} else {
			tx = tx.Where("announcement_class_section_id = ?", *q.SectionID)
		}
	}

	// 4d) Filter attachment (pakai tabel URLs, bukan kolom lama)
	if q.HasAttachment != nil {
		if *q.HasAttachment {
			tx = tx.Where(`
				EXISTS (
					SELECT 1 FROM announcement_urls au
					WHERE au.announcement_url_announcement_id = announcements.announcement_id
					  AND au.announcement_url_masjid_id = announcements.announcement_masjid_id
					  AND au.announcement_url_deleted_at IS NULL
					  AND (
						au.announcement_url_href IS NOT NULL AND au.announcement_url_href <> ''
					  	OR au.announcement_url_object_key IS NOT NULL AND au.announcement_url_object_key <> ''
					  )
				)`)
		} else {
			tx = tx.Where(`
				NOT EXISTS (
					SELECT 1 FROM announcement_urls au
					WHERE au.announcement_url_announcement_id = announcements.announcement_id
					  AND au.announcement_url_masjid_id = announcements.announcement_masjid_id
					  AND au.announcement_url_deleted_at IS NULL
					  AND (
						au.announcement_url_href IS NOT NULL AND au.announcement_url_href <> ''
					  	OR au.announcement_url_object_key IS NOT NULL AND au.announcement_url_object_key <> ''
					  )
				)`)
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

	// 8) Batch-load Themes (include=theme)
	if includeTheme {
		themeIDs := make([]uuid.UUID, 0, len(rows))
		seen := make(map[uuid.UUID]struct{}, len(rows))
		for i := range rows {
			if rows[i].AnnouncementThemeID == nil || *rows[i].AnnouncementThemeID == uuid.Nil {
				continue
			}
			id := *rows[i].AnnouncementThemeID
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				themeIDs = append(themeIDs, id)
			}
		}
		if len(themeIDs) > 0 {
			var themes []annModel.AnnouncementThemeModel
			if err := h.DB.
				Select("announcement_themes_id, announcement_themes_masjid_id, announcement_themes_name, announcement_themes_color").
				Where("announcement_themes_deleted_at IS NULL").
				Where("announcement_themes_id IN ?", themeIDs).
				Find(&themes).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat tema")
			}
			tmap := make(map[uuid.UUID]annModel.AnnouncementThemeModel, len(themes))
			for i := range themes {
				tmap[themes[i].AnnouncementThemesID] = themes[i]
			}
			for i := range rows {
				if rows[i].AnnouncementThemeID != nil {
					if th, ok := tmap[*rows[i].AnnouncementThemeID]; ok {
						rows[i].Theme = th
					}
				}
			}
		}
	}

	// 9) Siapkan DTO list
	out := make([]*annDTO.AnnouncementResponse, 0, len(rows))
	for i := range rows {
		out = append(out, annDTO.NewAnnouncementResponse(&rows[i]))
	}

	// 9a) Batch-load URLs jika diminta (pakai AttachURLs dari DTO) // <<< changed
	if includeURLs && len(rows) > 0 {
		annIDs := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			annIDs = append(annIDs, rows[i].AnnouncementID)
		}

		// Ambil rows sebagai model agar kompatibel dengan AttachURLs
		var urlRows []annModel.AnnouncementURLModel
		if err := h.DB.
			// pilih kolom yang diperlukan AttachURLs (href, label, kind, order, is_primary, ann_id, id)
			Select(`announcement_url_id,
			        announcement_url_label,
			        announcement_url_announcement_id,
			        announcement_url_href,
			        announcement_url_object_key,
			        announcement_url_kind,
			        announcement_url_order,
			        announcement_url_is_primary,
			        announcement_url_created_at`).
			Where("announcement_url_deleted_at IS NULL").
			Where("announcement_url_announcement_id IN ?", annIDs).
			Order("announcement_url_order ASC, announcement_url_created_at ASC").
			Find(&urlRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat lampiran")
		}

		// Group by announcement_id
		urlMap := make(map[uuid.UUID][]annModel.AnnouncementURLModel, len(rows))
		for i := range urlRows {
			aid := urlRows[i].AnnouncementURLAnnouncementId
			urlMap[aid] = append(urlMap[aid], urlRows[i])
		}

		// Lampirkan ke DTO
		for i := range out {
			if group, ok := urlMap[out[i].AnnouncementID]; ok {
				out[i].AttachURLs(group)
			}
		}
	}

	// 10) Response
	return helper.JsonList(c, out, helper.BuildMeta(total, p))
}
