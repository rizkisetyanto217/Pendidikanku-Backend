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

	annDTO "masjidku_backend/internals/features/school/other/announcement/dto"
	annModel "masjidku_backend/internals/features/school/other/announcement/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type AnnouncementController struct{ DB *gorm.DB }

func NewAnnouncementController(db *gorm.DB) *AnnouncementController { return &AnnouncementController{DB: db} }

var validateAnnouncement = validator.New()



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
