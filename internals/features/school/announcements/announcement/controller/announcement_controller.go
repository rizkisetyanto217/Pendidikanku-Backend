// internals/features/lembaga/announcements/announcement/controller/announcement_controller.go
package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	annDTO "masjidku_backend/internals/features/school/announcements/announcement/dto"
	annModel "masjidku_backend/internals/features/school/announcements/announcement/model"
	annThemeModel "masjidku_backend/internals/features/school/announcements/announcement_thema/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type AnnouncementController struct{ DB *gorm.DB }

func NewAnnouncementController(db *gorm.DB) *AnnouncementController { return &AnnouncementController{DB: db} }

var validateAnnouncement = validator.New()


// ===================== LIST =====================
// GET /admin/announcements
// ===================== LIST =====================
// GET /admin/announcements
func (h *AnnouncementController) List(c *fiber.Ctx) error {
	// ---------------------------
	// 1) Tenant scope (masjid)
	// ---------------------------
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	if len(masjidIDs) == 0 {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak ada akses masjid")
	}

	// ---------------------------
	// 2) Pagination & default sort
	// ---------------------------
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// ---------------------------
	// 3) Parse DTO query
	// ---------------------------
	var q annDTO.ListAnnouncementQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ---------------------------
	// 4) Base query (tenant-safe)
	// ---------------------------
	tx := h.DB.Model(&annModel.AnnouncementModel{}).
		Where("announcement_masjid_id IN ?", masjidIDs)

	// ---------------------------
	// 4a) Filter by Announcement ID (single/multi)
	//     ?id=uuid[,uuid...] atau ?announcement_id=uuid[,uuid...]
	// ---------------------------
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

	// ---------------------------
	// 4b) Filter Theme
	// ---------------------------
	if q.ThemeID != nil {
		if *q.ThemeID == uuid.Nil {
			tx = tx.Where("announcement_theme_id IS NULL")
		} else {
			tx = tx.Where("announcement_theme_id = ?", *q.ThemeID)
		}
	}

	// ---------------------------
	// 4c) Filter Section vs Global
	// ---------------------------
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

	// ---------------------------
	// 4d) Filter attachment
	// ---------------------------
	if q.HasAttachment != nil {
		if *q.HasAttachment {
			tx = tx.Where("announcement_attachment_url IS NOT NULL AND announcement_attachment_url <> ''")
		} else {
			tx = tx.Where("(announcement_attachment_url IS NULL OR announcement_attachment_url = '')")
		}
	}

	// ---------------------------
	// 4e) Filter is_active
	// ---------------------------
	if q.IsActive != nil {
		tx = tx.Where("announcement_is_active = ?", *q.IsActive)
	}

	// ---------------------------
	// 4f) Filter date range (YYYY-MM-DD)
	// ---------------------------
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

	// ---------------------------
	// 5) Total
	// ---------------------------
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ---------------------------
	// 6) Sorting (map-based)
	// ---------------------------
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

	// ---------------------------
	// 7) Fetch rows
	// ---------------------------
	var rows []annModel.AnnouncementModel
	if err := tx.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ---------------------------
	// 8) Batch-load Themes
	// ---------------------------
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
					rows[i].Theme = th
				}
			}
		}
	}

	// ---------------------------
	// 8b) Batch-load URLs
	// ---------------------------
	annIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		annIDs = append(annIDs, rows[i].AnnouncementID)
	}
	urlMap := make(map[uuid.UUID][]*annDTO.AnnouncementURLLite, len(rows))
	if len(annIDs) > 0 {
		var urlRows []annModel.AnnouncementURLModel
		if err := h.DB.
			Where("announcement_url_deleted_at IS NULL").
			Where("announcement_url_announcement_id IN ?", annIDs).
			Find(&urlRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat URL")
		}
		for i := range urlRows {
			u := urlRows[i]
			urlMap[u.AnnouncementURLAnnouncementID] = append(urlMap[u.AnnouncementURLAnnouncementID],
				&annDTO.AnnouncementURLLite{
					ID:    u.AnnouncementURLID,
					Label: u.AnnouncementURLLabel,
					Href:  u.AnnouncementURLHref,
				},
			)
		}
	}

	// ---------------------------
	// 9) Map ke DTO + inject Urls
	// ---------------------------
	items := make([]*annDTO.AnnouncementResponse, 0, len(rows))
	for i := range rows {
		if resp := annDTO.NewAnnouncementResponse(&rows[i]); resp != nil {
			if urls := urlMap[rows[i].AnnouncementID]; len(urls) > 0 {
				resp.Urls = urls
			}
			items = append(items, resp)
		}
	}

	// ---------------------------
	// 10) Response standar
	// ---------------------------
	return helper.JsonList(c, items, helper.BuildMeta(total, p))
}

// ===================== Utils =====================

func parseUUIDsCSV(s string) ([]uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			return nil, fmt.Errorf("invalid uuid: %q", p)
		}
		out = append(out, id)
	}
	return out, nil
}

// ===================== CREATE =====================
// POST /admin/announcements
func (h *AnnouncementController) Create(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// role detection
	isAdmin := func() bool {
		if id, err := helperAuth.GetMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	isTeacher := func() bool {
		if id, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	if !isAdmin && !isTeacher {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak diizinkan")
	}

	// Request DTO
	var req annDTO.CreateAnnouncementRequest
	ct := c.Get("Content-Type")

	// Parse body
	if strings.HasPrefix(ct, "multipart/form-data") {
		req.AnnouncementTitle = strings.TrimSpace(c.FormValue("announcement_title"))
		req.AnnouncementDate = strings.TrimSpace(c.FormValue("announcement_date"))
		req.AnnouncementContent = strings.TrimSpace(c.FormValue("announcement_content"))

		if v := strings.TrimSpace(c.FormValue("announcement_theme_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.AnnouncementThemeID = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("announcement_class_section_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.AnnouncementClassSectionID = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("announcement_is_active")); v != "" {
			b := strings.EqualFold(v, "true") || v == "1"
			req.AnnouncementIsActive = &b
		}
	} else {
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// Validasi DTO
	if err := validateAnnouncement.Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	// Aturan role (Admin prioritas)
	if isAdmin {
		req.AnnouncementClassSectionID = nil // global
	} else if isTeacher {
		if req.AnnouncementClassSectionID == nil || *req.AnnouncementClassSectionID == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Teacher wajib memilih section")
		}
		if err := h.ensureSectionBelongsToMasjid(*req.AnnouncementClassSectionID, masjidID); err != nil {
			return err
		}
	}

	// Validasi tema
	if req.AnnouncementThemeID != nil {
		if err := h.ensureThemeBelongsToMasjid(*req.AnnouncementThemeID, masjidID); err != nil {
			return err
		}
	}

	// Bangun model dari DTO
	m := req.ToModel(masjidID)

	// Set who created:
	if isTeacher {
		mtID, err := helperAuth.GetMasjidTeacherIDForMasjid(c, masjidID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusForbidden, "Akun Anda tidak terdaftar sebagai guru di masjid ini")
		}
		m.AnnouncementCreatedByTeacherID = &mtID
	} else {
		m.AnnouncementCreatedByTeacherID = nil
	}

	if err := h.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat pengumuman")
	}
	return helper.JsonCreated(c, "Pengumuman berhasil dibuat", annDTO.NewAnnouncementResponse(m))
}

// Pastikan section milik masjid ini
func (h *AnnouncementController) ensureSectionBelongsToMasjid(sectionID, masjidID uuid.UUID) error {
	var cnt int64
	if err := h.DB.
		Table("class_sections").
		Joins("JOIN classes ON classes.class_id = class_sections.class_sections_class_id").
		Where("class_sections.class_sections_id = ? AND classes.class_masjid_id = ?", sectionID, masjidID).
		Count(&cnt).Error; err != nil {
		return helper.JsonError(nil, fiber.StatusInternalServerError, "Gagal validasi section")
	}
	if cnt == 0 {
		return helper.JsonError(nil, fiber.StatusBadRequest, "Section bukan milik masjid Anda")
	}
	return nil
}

// Pastikan theme milik masjid & belum soft-deleted
func (h *AnnouncementController) ensureThemeBelongsToMasjid(themeID, masjidID uuid.UUID) error {
	var cnt int64
	if err := h.DB.
		Table("announcement_themes").
		Where("announcement_themes_id = ? AND announcement_themes_masjid_id = ? AND announcement_themes_deleted_at IS NULL",
			themeID, masjidID).
		Count(&cnt).Error; err != nil {
		return helper.JsonError(nil, fiber.StatusInternalServerError, "Gagal validasi tema")
	}
	if cnt == 0 {
		return helper.JsonError(nil, fiber.StatusBadRequest, "Tema tidak ditemukan atau bukan milik masjid Anda")
	}
	return nil
}

// --- tenant guard fetch
func (h *AnnouncementController) findWithTenantGuard(id, masjidID uuid.UUID) (*annModel.AnnouncementModel, error) {
	var m annModel.AnnouncementModel
	if err := h.DB.Where("announcement_id = ? AND announcement_masjid_id = ?", id, masjidID).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, helper.JsonError(nil, fiber.StatusNotFound, "Pengumuman tidak ditemukan")
		}
		return nil, helper.JsonError(nil, fiber.StatusInternalServerError, "Gagal mengambil pengumuman")
	}
	return &m, nil
}

// ===================== UPDATE =====================
// PUT /admin/announcements/:id
func (h *AnnouncementController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// role detection
	isAdmin := func() bool {
		if id, err := helperAuth.GetMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	isTeacher := func() bool {
		if id, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	if !isAdmin && !isTeacher {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak diizinkan")
	}

	// fetch existing (tenant-safe)
	existing, err := h.findWithTenantGuard(annID, masjidID)
	if err != nil {
		return err
	}

	var req annDTO.UpdateAnnouncementRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// parse payload (multipart / json)
	if strings.HasPrefix(ct, "multipart/form-data") {
		if v := strings.TrimSpace(c.FormValue("announcement_title")); v != "" {
			req.AnnouncementTitle = &v
		}
		if v := strings.TrimSpace(c.FormValue("announcement_date")); v != "" {
			req.AnnouncementDate = &v
		}
		if v := strings.TrimSpace(c.FormValue("announcement_content")); v != "" {
			req.AnnouncementContent = &v
		}
		if v := strings.TrimSpace(c.FormValue("announcement_is_active")); v != "" {
			b := strings.EqualFold(v, "true") || v == "1"
			req.AnnouncementIsActive = &b
		}
		if v := strings.TrimSpace(c.FormValue("announcement_theme_id")); v != "" {
			if id, e := uuid.Parse(v); e == nil {
				req.AnnouncementThemeID = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("announcement_class_section_id")); v != "" {
			if id, e := uuid.Parse(v); e == nil {
				req.AnnouncementClassSectionID = &id
			}
		}
	} else {
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// Normalisasi: uuid.Nil => NULL
	if req.AnnouncementThemeID != nil && *req.AnnouncementThemeID == uuid.Nil {
		req.AnnouncementThemeID = nil
	}
	if req.AnnouncementClassSectionID != nil && *req.AnnouncementClassSectionID == uuid.Nil {
		req.AnnouncementClassSectionID = nil
	}

	// Validasi DTO
	if err := validateAnnouncement.Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	// Aturan per-role
	if isAdmin && !isTeacher {
		req.AnnouncementClassSectionID = nil // Admin → GLOBAL
	}
	if isTeacher && req.AnnouncementClassSectionID != nil {
		if err := h.ensureSectionBelongsToMasjid(*req.AnnouncementClassSectionID, masjidID); err != nil {
			return err
		}
	}

	// Validasi theme bila diubah
	if req.AnnouncementThemeID != nil {
		if err := h.ensureThemeBelongsToMasjid(*req.AnnouncementThemeID, masjidID); err != nil {
			return err
		}
	}

	// Build updates map agar nilai falsy (false / "") juga ter-update
	updates := map[string]interface{}{}
	if req.AnnouncementTitle != nil {
		updates["announcement_title"] = strings.TrimSpace(*req.AnnouncementTitle)
	}
	if req.AnnouncementDate != nil {
		if dt := strings.TrimSpace(*req.AnnouncementDate); dt != "" {
			if parsed, e := time.Parse("2006-01-02", dt); e == nil {
				updates["announcement_date"] = parsed
			} else {
				return helper.JsonError(c, fiber.StatusBadRequest, "announcement_date tidak valid (YYYY-MM-DD)")
			}
		}
	}
	if req.AnnouncementContent != nil {
		updates["announcement_content"] = strings.TrimSpace(*req.AnnouncementContent)
	}
	if req.AnnouncementIsActive != nil {
		updates["announcement_is_active"] = *req.AnnouncementIsActive
	}
	if req.AnnouncementThemeID != nil {
		updates["announcement_theme_id"] = req.AnnouncementThemeID // nil → set NULL
	}
	if req.AnnouncementClassSectionID != nil {
		updates["announcement_class_section_id"] = req.AnnouncementClassSectionID // nil → set NULL
	}

	// Tidak ada perubahan
	if len(updates) == 0 {
		return helper.JsonUpdated(c, "Tidak ada perubahan", annDTO.NewAnnouncementResponse(existing))
	}

	// Enforce rule setelah perubahan diaplikasikan:
	if isTeacher && !isAdmin {
		finalSection := existing.AnnouncementClassSectionID
		if v, ok := updates["announcement_class_section_id"]; ok {
			if v == nil {
				finalSection = nil
			} else if ptr, ok2 := v.(*uuid.UUID); ok2 {
				finalSection = ptr
			}
		}
		if finalSection == nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Teacher wajib memilih section (tidak boleh global)")
		}
		// Hanya pembuat yang boleh update
		if existing.AnnouncementCreatedByTeacherID != nil && *existing.AnnouncementCreatedByTeacherID != userID {
			return helper.JsonError(c, fiber.StatusForbidden, "Hanya pembuat yang boleh mengubah pengumuman ini")
		}
	}

	// Jalankan update
	if err := h.DB.Model(&annModel.AnnouncementModel{}).
		Where("announcement_id = ? AND announcement_masjid_id = ?", existing.AnnouncementID, masjidID).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui pengumuman")
	}

	// Reload agar updated_at terbaru ikut
	var after annModel.AnnouncementModel
	if err := h.DB.
		Where("announcement_id = ? AND announcement_masjid_id = ?", existing.AnnouncementID, masjidID).
		First(&after).Error; err == nil {
		return helper.JsonUpdated(c, "Pengumuman diperbarui", annDTO.NewAnnouncementResponse(&after))
	}

	// Fallback jika reload gagal
	return helper.JsonUpdated(c, "Pengumuman diperbarui", annDTO.NewAnnouncementResponse(existing))
}

// ===================== DELETE =====================
// DELETE /admin/announcements/:id (soft/hard via ?force=true)
func (h *AnnouncementController) Delete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// role detection
	isAdmin := func() bool {
		if id, err := helperAuth.GetMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	isTeacher := func() bool {
		if id, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID {
			return true
		}
		return false
	}()
	if !isAdmin && !isTeacher {
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak diizinkan")
	}

	existing, err := h.findWithTenantGuard(annID, masjidID)
	if err != nil {
		return err
	}

	// Rule delete
	if isTeacher && !isAdmin {
		if existing.AnnouncementClassSectionID == nil {
			return helper.JsonError(c, fiber.StatusForbidden, "Teacher tidak boleh menghapus pengumuman global")
		}
		if existing.AnnouncementCreatedByTeacherID != nil && *existing.AnnouncementCreatedByTeacherID != userID {
			return helper.JsonError(c, fiber.StatusForbidden, "Hanya pembuat yang boleh menghapus pengumuman ini")
		}
	}

	// Opsi hard delete
	force := strings.EqualFold(c.Query("force"), "true")

	db := h.DB
	if force {
		db = db.Unscoped()
	}

	if err := db.
		Where("announcement_id = ? AND announcement_masjid_id = ?", existing.AnnouncementID, masjidID).
		Delete(&annModel.AnnouncementModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus pengumuman")
	}

	msg := "Pengumuman dihapus"
	if force {
		msg = "Pengumuman dihapus permanen"
	}
	return helper.JsonDeleted(c, msg, fiber.Map{
		"announcement_id": existing.AnnouncementID,
	})
}
