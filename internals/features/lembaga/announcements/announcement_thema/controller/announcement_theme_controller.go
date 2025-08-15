// internals/features/lembaga/announcements/controller/announcement_theme_controller.go
package controller

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	annDTO "masjidku_backend/internals/features/lembaga/announcements/announcement_thema/dto"
	annModel "masjidku_backend/internals/features/lembaga/announcements/announcement_thema/model"
	helper "masjidku_backend/internals/helpers"
)

type AnnouncementThemeController struct {
	DB *gorm.DB
}

func NewAnnouncementThemeController(db *gorm.DB) *AnnouncementThemeController {
	return &AnnouncementThemeController{DB: db}
}

var validateAnnouncementTheme = validator.New()

/* ================= Helpers ================= */

func (h *AnnouncementThemeController) findThemeWithTenantGuard(id, masjidID uuid.UUID) (*annModel.AnnouncementThemeModel, error) {
	var m annModel.AnnouncementThemeModel
	if err := h.DB.
		Where("announcement_themes_id = ? AND announcement_themes_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Tema tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil tema")
	}
	return &m, nil
}

func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint")
}

/* ================= Handlers ================= */

// POST /admin/announcement-themes
func (h *AnnouncementThemeController) Create(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req annDTO.CreateAnnouncementThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// --- Generate/normalize slug sebelum validasi ---
	nameTrim := strings.TrimSpace(req.AnnouncementThemesName)
	slugTrim := strings.TrimSpace(req.AnnouncementThemesSlug)

	if slugTrim != "" {
		req.AnnouncementThemesSlug = helper.GenerateSlug(slugTrim)
	} else if nameTrim != "" {
		req.AnnouncementThemesSlug = helper.GenerateSlug(nameTrim)
	}
	// NB: jika name & slug dua-duanya kosong, biarkan validator yang menolak
	// ------------------------------------------------

	if err := validateAnnouncementTheme.Struct(req); err != nil {
		return helper.ValidationError(c, err)
	}

	m := req.ToModel(masjidID)

	if err := h.DB.Create(m).Error; err != nil {
		if isUniqueErr(err) {
			return helper.Error(c, fiber.StatusConflict, "Nama atau slug tema sudah dipakai")
		}
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal membuat tema")
	}

	return helper.Success(c, "Tema berhasil dibuat", annDTO.NewAnnouncementThemeResponse(m))
}

// GET /admin/announcement-themes/:id
func (h *AnnouncementThemeController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findThemeWithTenantGuard(id, masjidID)
	if err != nil {
		return err
	}
	return helper.JsonOK(c, "OK", annDTO.NewAnnouncementThemeResponse(m))
}

// GET /admin/announcement-themes
func (h *AnnouncementThemeController) List(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q annDTO.ListAnnouncementThemeQuery
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := h.DB.Model(&annModel.AnnouncementThemeModel{}).
		Where("announcement_themes_masjid_id = ?", masjidID)

	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		name := "%" + strings.TrimSpace(*q.Name) + "%"
		tx = tx.Where("announcement_themes_name ILIKE ?", name)
	}
	if q.Slug != nil && strings.TrimSpace(*q.Slug) != "" {
		tx = tx.Where("announcement_themes_slug = ?", strings.TrimSpace(*q.Slug))
	}
	if q.IsActive != nil {
		tx = tx.Where("announcement_themes_is_active = ?", *q.IsActive)
	}

	sort := "created_at_desc"
	if q.Sort != nil {
		sort = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sort {
	case "created_at_asc":
		tx = tx.Order("announcement_themes_created_at ASC")
	case "name_asc":
		tx = tx.Order("announcement_themes_name ASC")
	case "name_desc":
		tx = tx.Order("announcement_themes_name DESC")
	default:
		tx = tx.Order("announcement_themes_created_at DESC")
	}

	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	var rows []annModel.AnnouncementThemeModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data tema")
	}

	resp := make([]*annDTO.AnnouncementThemeResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, annDTO.NewAnnouncementThemeResponse(&rows[i]))
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"total": len(resp),
		"items": resp,
	})
}

// PUT /admin/announcement-themes/:id
// PUT /admin/announcement-themes/:id
func (h *AnnouncementThemeController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	existing, err := h.findThemeWithTenantGuard(id, masjidID)
	if err != nil {
		return err
	}

	var req annDTO.UpdateAnnouncementThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// --- Normalize slug jika dikirim ---
	if req.AnnouncementThemesSlug != nil {
		s := strings.TrimSpace(*req.AnnouncementThemesSlug)
		s = helper.GenerateSlug(s)
		if s == "" {
			return helper.Error(c, fiber.StatusBadRequest, "Slug tidak valid")
		}
		req.AnnouncementThemesSlug = &s
	}
	// -----------------------------------

	if err := validateAnnouncementTheme.Struct(req); err != nil {
		return helper.ValidationError(c, err)
	}

	req.ApplyToModel(existing)

	if err := h.DB.Save(existing).Error; err != nil {
		if isUniqueErr(err) {
			return helper.Error(c, fiber.StatusConflict, "Nama atau slug tema sudah dipakai")
		}
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal memperbarui tema")
	}

	return helper.JsonUpdated(c, "Tema diperbarui", annDTO.NewAnnouncementThemeResponse(existing))
}


// DELETE /admin/announcement-themes/:id  (soft delete)
func (h *AnnouncementThemeController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	res := h.DB.
		Where("announcement_themes_id = ? AND announcement_themes_masjid_id = ?", id, masjidID).
		Delete(&annModel.AnnouncementThemeModel{})

	if res.Error != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal menghapus tema")
	}
	if res.RowsAffected == 0 {
		return helper.Error(c, fiber.StatusNotFound, "Tema tidak ditemukan")
	}

	return helper.JsonDeleted(c, "Tema dihapus", fiber.Map{
		"announcement_themes_id": id,
	})
}
