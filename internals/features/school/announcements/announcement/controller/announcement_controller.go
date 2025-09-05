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
	annThemeModel "masjidku_backend/internals/features/school/announcements/announcement_thema/model" // import model tema agar tidak bentrok dengan model announcementModel
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	// annURLModel "masjidku_backend/internals/features/school/announcements/announcement_urls/model"
)

type AnnouncementController struct{ DB *gorm.DB }

func NewAnnouncementController(db *gorm.DB) *AnnouncementController { return &AnnouncementController{DB: db} }

var validateAnnouncement = validator.New()


// ===================== GET BY ID =====================
// GET /admin/announcements/:id
func (h *AnnouncementController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findWithTenantGuard(annID, masjidID)
	if err != nil {
		return err
	}
	return helper.Success(c, "OK", annDTO.NewAnnouncementResponse(m))
}




// ===================== LIST =====================
// GET /admin/announcements
// ===================== LIST =====================
// GET /admin/announcements
func (h *AnnouncementController) List(c *fiber.Ctx) error {
	// 1) scope masjid
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}
	if len(masjidIDs) == 0 {
		return fiber.NewError(fiber.StatusForbidden, "Tidak ada akses masjid")
	}

	// 2) pagination via helper.ParseFiber
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// 3) parse query
	var q annDTO.ListAnnouncementQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	// 4) base query
	tx := h.DB.Model(&annModel.AnnouncementModel{}).
		Where("announcement_masjid_id IN ?", masjidIDs)

	// ---- filter Theme
	if q.ThemeID != nil {
		if *q.ThemeID == uuid.Nil {
			tx = tx.Where("announcement_theme_id IS NULL")
		} else {
			tx = tx.Where("announcement_theme_id = ?", *q.ThemeID)
		}
	}

	// ---- filter Section vs Global
	includeGlobal := true
	if q.IncludeGlobal != nil {
		includeGlobal = *q.IncludeGlobal
	}
	onlyGlobal := q.OnlyGlobal != nil && *q.OnlyGlobal

	sectionIDsCSV := strings.TrimSpace(c.Query("section_ids"))
	sectionIDs, parseErr := parseUUIDsCSV(sectionIDsCSV)
	if parseErr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "section_ids berisi UUID tidak valid")
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

	// ---- filter Attachment
	if q.HasAttachment != nil {
		if *q.HasAttachment {
			tx = tx.Where("announcement_attachment_url IS NOT NULL AND announcement_attachment_url <> ''")
		} else {
			tx = tx.Where("(announcement_attachment_url IS NULL OR announcement_attachment_url = '')")
		}
	}

	// ---- filter is_active
	if q.IsActive != nil {
		tx = tx.Where("announcement_is_active = ?", *q.IsActive)
	}

	// ---- filter date range
	parseDate := func(s string) (time.Time, error) {
		return time.Parse("2006-01-02", strings.TrimSpace(s))
	}
	if q.DateFrom != nil && strings.TrimSpace(*q.DateFrom) != "" {
		t, err := parseDate(*q.DateFrom)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
		}
		tx = tx.Where("announcement_date >= ?", t)
	}
	if q.DateTo != nil && strings.TrimSpace(*q.DateTo) != "" {
		t, err := parseDate(*q.DateTo)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
		}
		tx = tx.Where("announcement_date <= ?", t)
	}

	// 5) total
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// 6) sorting
	orderExpr := "announcement_date DESC"
	if q.Sort != nil {
		switch strings.ToLower(strings.TrimSpace(*q.Sort)) {
		case "date_asc":
			orderExpr = "announcement_date ASC"
		case "created_at_desc":
			orderExpr = "announcement_created_at DESC"
		case "created_at_asc":
			orderExpr = "announcement_created_at ASC"
		case "title_asc":
			orderExpr = "announcement_title ASC"
		case "title_desc":
			orderExpr = "announcement_title DESC"
		}
	}

	// 7) fetch rows
	var rows []annModel.AnnouncementModel
	if err := tx.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 8) batch-load Theme
	themeIDs := make([]uuid.UUID, 0, len(rows))
	seenTheme := map[uuid.UUID]struct{}{}
	for i := range rows {
		if rows[i].AnnouncementThemeID != nil {
			id := *rows[i].AnnouncementThemeID
			if id == uuid.Nil { continue }
			if _, ok := seenTheme[id]; !ok {
				seenTheme[id] = struct{}{}
				themeIDs = append(themeIDs, id)
			}
		}
	}
	if len(themeIDs) > 0 {
		var themes []annThemeModel.AnnouncementThemeModel
		if err := h.DB.
			Select("announcement_themes_id, announcement_themes_masjid_id, announcement_themes_name, announcement_themes_color").
			Where("announcement_themes_deleted_at IS NULL").
			Where("announcement_themes_id IN ?", themeIDs).
			Find(&themes).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memuat tema")
		}
		tmap := make(map[uuid.UUID]*annThemeModel.AnnouncementThemeModel, len(themes))
		for i := range themes {
			t := themes[i]
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

	// 8b) batch-load URLs
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
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memuat URL")
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

	// 9) map ke response DTO + inject Urls
	items := make([]*annDTO.AnnouncementResponse, 0, len(rows))
	for i := range rows {
		resp := annDTO.NewAnnouncementResponse(&rows[i])
		if resp != nil {
			if urls := urlMap[rows[i].AnnouncementID]; len(urls) > 0 {
				resp.Urls = urls
			}
			items = append(items, resp)
		}
	}

	// 10) response standar
	return helper.JsonList(c, items, helper.BuildMeta(total, p))
}


// ===================== Utils =====================

// aman: kembalikan []uuid.UUID, skip kosong
func parseUUIDsCSV(s string) ([]uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" { // SKIP item kosong
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


// internals/features/lembaga/announcements/announcement/controller/announcement_controller.go

// POST /admin/announcements
// POST /admin/announcements
// internals/features/lembaga/announcements/announcement/controller/announcement_controller.go

func (h *AnnouncementController) Create(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// role detection (tetap versi kamu biar minimal diff)
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
		return fiber.NewError(fiber.StatusForbidden, "Tidak diizinkan")
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
			return helper.Error(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// Validasi DTO
	if err := validateAnnouncement.Struct(req); err != nil {
		return helper.ValidationError(c, err)
	}

	// Aturan role (Admin menang prioritas)
	if isAdmin {
		// Admin/DKM → pengumuman global (tanpa section)
		req.AnnouncementClassSectionID = nil
	} else if isTeacher {
		// Teacher wajib pilih section yang milik masjid
		if req.AnnouncementClassSectionID == nil || *req.AnnouncementClassSectionID == uuid.Nil {
			return helper.Error(c, fiber.StatusBadRequest, "Teacher wajib memilih section")
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

	// Bangun model dari DTO (tanpa set teacher_id di sini)
	m := req.ToModel(masjidID)

	// Set who created:
	// 1) kalau teacher → isi FK ke masjid_teachers
	if isTeacher {
		mtID, err := helperAuth.GetMasjidTeacherIDForMasjid(c, masjidID)
		if err != nil {
			return helper.Error(c, fiber.StatusForbidden, "Akun Anda tidak terdaftar sebagai guru di masjid ini")
		}
		m.AnnouncementCreatedByTeacherID = &mtID
	} else {
		// Admin/DKM → harus NULL agar lolos FK
		m.AnnouncementCreatedByTeacherID = nil
	}

	// 2) (opsional) kalau model punya kolom created_by_user_id, isi userID
	// m.AnnouncementCreatedByUserID = &userID

	if err := h.DB.Create(m).Error; err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal membuat pengumuman")
	}
	return helper.Success(c, "Pengumuman berhasil dibuat", annDTO.NewAnnouncementResponse(m))
}

// Pastikan section milik masjid ini
func (h *AnnouncementController) ensureSectionBelongsToMasjid(sectionID, masjidID uuid.UUID) error {
	var cnt int64
	if err := h.DB.
		Table("class_sections").
		Joins("JOIN classes ON classes.class_id = class_sections.class_sections_class_id").
		Where("class_sections.class_sections_id = ? AND classes.class_masjid_id = ?", sectionID, masjidID).
		Count(&cnt).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
	}
	if cnt == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Section bukan milik masjid Anda")
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
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi tema")
	}
	if cnt == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Tema tidak ditemukan atau bukan milik masjid Anda")
	}
	return nil
}


// --- tenant guard fetch
func (h *AnnouncementController) findWithTenantGuard(id, masjidID uuid.UUID) (*annModel.AnnouncementModel, error) {
	var m annModel.AnnouncementModel
	if err := h.DB.Where("announcement_id = ? AND announcement_masjid_id = ?", id, masjidID).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Pengumuman tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil pengumuman")
	}
	return &m, nil
}



// ===================== UPDATE =====================
// PUT /admin/announcements/:id
// ===================== UPDATE =====================
// PUT /admin/announcements/:id
func (h *AnnouncementController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil { return err }

	annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	// role detection
	isAdmin := func() bool {
		if id, err := helperAuth.GetMasjidIDFromToken(c); err == nil && id == masjidID { return true }
		return false
	}()
	isTeacher := func() bool {
		if id, err := helperAuth.GetTeacherMasjidIDFromToken(c); err == nil && id == masjidID { return true }
		return false
	}()
	if !isAdmin && !isTeacher {
		return fiber.NewError(fiber.StatusForbidden, "Tidak diizinkan")
	}

	// fetch existing (tenant-safe)
	existing, err := h.findWithTenantGuard(annID, masjidID)
	if err != nil { return err }

	var req annDTO.UpdateAnnouncementRequest
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// parse payload (multipart / json)
	if strings.HasPrefix(ct, "multipart/form-data") {
		// Parsing form fields
		if v := strings.TrimSpace(c.FormValue("announcement_title")); v != "" { req.AnnouncementTitle = &v }
		if v := strings.TrimSpace(c.FormValue("announcement_date")); v != "" { req.AnnouncementDate = &v }
		if v := strings.TrimSpace(c.FormValue("announcement_content")); v != "" { req.AnnouncementContent = &v }
		if v := strings.TrimSpace(c.FormValue("announcement_is_active")); v != "" {
			b := strings.EqualFold(v, "true") || v == "1"
			req.AnnouncementIsActive = &b
		}
		if v := strings.TrimSpace(c.FormValue("announcement_theme_id")); v != "" {
			if id, e := uuid.Parse(v); e == nil { req.AnnouncementThemeID = &id }
		}
		if v := strings.TrimSpace(c.FormValue("announcement_class_section_id")); v != "" {
			if id, e := uuid.Parse(v); e == nil { req.AnnouncementClassSectionID = &id }
		}
	} else {
		// JSON Parsing
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
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
		return helper.ValidationError(c, err)
	}

	// Aturan per-role
	if isAdmin && !isTeacher {
		// Admin: paksa GLOBAL
		req.AnnouncementClassSectionID = nil
	}
	if isTeacher && req.AnnouncementClassSectionID != nil {
		// Teacher set section → pastikan milik masjid
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
				return fiber.NewError(fiber.StatusBadRequest, "announcement_date tidak valid (YYYY-MM-DD)")
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
		// nil -> set NULL
		updates["announcement_theme_id"] = req.AnnouncementThemeID
	}
	if req.AnnouncementClassSectionID != nil {
		// nil -> set NULL
		updates["announcement_class_section_id"] = req.AnnouncementClassSectionID
	}

	// Kalau tidak ada perubahan, kembalikan apa adanya
	if len(updates) == 0 {
		return helper.JsonUpdated(c, "Tidak ada perubahan", annDTO.NewAnnouncementResponse(existing))
	}

	// Enforce rule setelah perubahan diaplikasikan:
	// - Jika teacher dan hasil akhirnya global → tolak.
	// Enforce rule setelah perubahan diaplikasikan:
	// - Jika teacher dan hasil akhirnya global → tolak.
	if isTeacher && !isAdmin {
		// Prediksi hasil final (pakai updates atau existing)
		finalSection := existing.AnnouncementClassSectionID
		if v, ok := updates["announcement_class_section_id"]; ok {
			if v == nil {
				finalSection = nil
			} else if ptr, ok2 := v.(*uuid.UUID); ok2 {
				finalSection = ptr
			}
		}
		if finalSection == nil {
			return fiber.NewError(fiber.StatusBadRequest, "Teacher wajib memilih section (tidak boleh global)")
		}
		// Hanya pembuat yang boleh update
		if existing.AnnouncementCreatedByTeacherID != nil && *existing.AnnouncementCreatedByTeacherID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Hanya pembuat yang boleh mengubah pengumuman ini")
		}
	}


	// Jalankan update (trigger DB akan menyentuh updated_at)
	if err := h.DB.Model(&annModel.AnnouncementModel{}).
		Where("announcement_id = ? AND announcement_masjid_id = ?", existing.AnnouncementID, masjidID).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui pengumuman")
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
// DELETE /admin/announcements/:id (soft delete)
// DELETE /admin/announcements/:id
func (h *AnnouncementController) Delete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}
	annID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
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
		return fiber.NewError(fiber.StatusForbidden, "Tidak diizinkan")
	}

	existing, err := h.findWithTenantGuard(annID, masjidID)
	if err != nil {
		return err
	}

	// Rule delete:
	// - Admin: boleh hapus apapun di masjidnya.
	// - Teacher: TIDAK boleh hapus global (section NULL) dan hanya boleh hapus yang dia buat.
	if isTeacher && !isAdmin {
		if existing.AnnouncementClassSectionID == nil {
			return fiber.NewError(fiber.StatusForbidden, "Teacher tidak boleh menghapus pengumuman global")
		}
		// Periksa jika AnnouncementCreatedByTeacherID bukan nil dan membandingkannya dengan userID
		if existing.AnnouncementCreatedByTeacherID != nil && *existing.AnnouncementCreatedByTeacherID != userID {
			return fiber.NewError(fiber.StatusForbidden, "Hanya pembuat yang boleh menghapus pengumuman ini")
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
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus pengumuman")
	}

	msg := "Pengumuman dihapus"
	if force {
		msg = "Pengumuman dihapus permanen"
	}
	return helper.JsonDeleted(c, msg, fiber.Map{
		"announcement_id": existing.AnnouncementID,
	})
}

