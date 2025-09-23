// internals/features/lembaga/announcements/controller/announcement_theme_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	annDTO "masjidku_backend/internals/features/school/others/post/dto"
	annModel "masjidku_backend/internals/features/school/others/post/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type AnnouncementThemeController struct {
	DB *gorm.DB
}

func NewAnnouncementThemeController(db *gorm.DB) *AnnouncementThemeController {
	return &AnnouncementThemeController{DB: db}
}

var validateAnnouncementTheme = validator.New()

/* ================= Helpers ================= */

// Hanya mengembalikan error Go biasa; mapping status → JSON dilakukan di handler.
func (h *AnnouncementThemeController) findThemeWithTenantGuard(id, masjidID uuid.UUID) (*annModel.AnnouncementThemeModel, error) {
	var m annModel.AnnouncementThemeModel
	err := h.DB.
		Where("announcement_themes_id = ? AND announcement_themes_masjid_id = ?", id, masjidID).
		First(&m).Error
	if err != nil {
		return nil, err
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

// resolveMasjidIDDKM: ambil masjid dari path/header/cookie/query/host/token,
// lalu pastikan user adalah DKM/Admin masjid tsb.
// Mengembalikan masjidID atau JSON error yang sudah terformat.
func (h *AnnouncementThemeController) resolveMasjidIDDKM(c *fiber.Ctx) (uuid.UUID, error) {
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return uuid.Nil, helper.JsonError(c, fe.Code, fe.Message)
		}
		return uuid.Nil, helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak valid")
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return uuid.Nil, helper.JsonError(c, fe.Code, fe.Message)
		}
		return uuid.Nil, helper.JsonError(c, fiber.StatusForbidden, "Akses masjid ditolak")
	}
	return masjidID, nil
}

/* ================= Handlers ================= */

// POST /admin/announcement-themes
func (h *AnnouncementThemeController) Create(c *fiber.Ctx) error {
	masjidID, jerr := h.resolveMasjidIDDKM(c)
	if jerr != nil {
		return jerr // sudah JSON
	}

	var req annDTO.CreateAnnouncementThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalize + rapikan slug pakai helpers baru
	nameTrim := strings.TrimSpace(req.AnnouncementThemesName)
	slugTrim := strings.TrimSpace(req.AnnouncementThemesSlug)
	if slugTrim != "" {
		req.AnnouncementThemesSlug = helper.Slugify(slugTrim, 120)
	} else if nameTrim != "" {
		req.AnnouncementThemesSlug = helper.SuggestSlugFromName(nameTrim) // default maxLen=100
	}

	if err := validateAnnouncementTheme.Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	m := req.ToModel(masjidID)

	if err := h.DB.Create(m).Error; err != nil {
		if isUniqueErr(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama atau slug tema sudah dipakai")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat tema")
	}

	return helper.JsonCreated(c, "Tema berhasil dibuat", annDTO.NewAnnouncementThemeResponse(m))
}

// GET /admin/announcement-themes
// (handler list-mu tetap, tidak diubah di sini)

// PUT /admin/announcement-themes/:id
func (h *AnnouncementThemeController) Update(c *fiber.Ctx) error {
	masjidID, jerr := h.resolveMasjidIDDKM(c)
	if jerr != nil {
		return jerr
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	existing, err := h.findThemeWithTenantGuard(id, masjidID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Tema tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil tema")
	}

	var req annDTO.UpdateAnnouncementThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalize slug jika dikirim (pakai helpers baru)
	if req.AnnouncementThemesSlug != nil {
		s := helper.Slugify(strings.TrimSpace(*req.AnnouncementThemesSlug), 120)
		if s == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak valid")
		}
		req.AnnouncementThemesSlug = &s
	}

	if err := validateAnnouncementTheme.Struct(req); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}

	req.ApplyToModel(existing)

	if err := h.DB.Save(existing).Error; err != nil {
		if isUniqueErr(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama atau slug tema sudah dipakai")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui tema")
	}

	return helper.JsonUpdated(c, "Tema diperbarui", annDTO.NewAnnouncementThemeResponse(existing))
}

// DELETE /admin/announcement-themes/:id  (soft delete)
func (h *AnnouncementThemeController) Delete(c *fiber.Ctx) error {
	masjidID, jerr := h.resolveMasjidIDDKM(c)
	if jerr != nil {
		return jerr
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	res := h.DB.
		Where("announcement_themes_id = ? AND announcement_themes_masjid_id = ?", id, masjidID).
		Delete(&annModel.AnnouncementThemeModel{})

	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus tema")
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Tema tidak ditemukan")
	}

	return helper.JsonDeleted(c, "Tema dihapus", fiber.Map{
		"announcement_themes_id": id,
	})
}
