// internals/features/lembaga/announcements/controller/announcement_theme_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	annDTO "masjidku_backend/internals/features/school/announcements/announcement_thema/dto"
	annModel "masjidku_backend/internals/features/school/announcements/announcement_thema/model"
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

/* ================= Handlers ================= */

// POST /admin/announcement-themes
func (h *AnnouncementThemeController) Create(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var req annDTO.CreateAnnouncementThemeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
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
// Opsional:
//   ?announcement_theme_id=<uuid>  (atau ?id=<uuid> / /admin/announcement-themes/:id)
//   ?name=..., ?slug=..., ?is_active=true|false
//   Pagination: ?page=1&per_page=25 (atau limit), sort_by=created_at|updated_at|name|slug, order=asc|desc
// import tambahan:
//   "errors"
//   "strconv"
//   "strings"
//   "github.com/google/uuid"
//   "gorm.io/gorm"

// helper kecil: cek kolom ada/tidak di tabel (untuk cari nama kolom id)
func hasColumn(db *gorm.DB, table, col string) bool {
	var n int64
	_ = db.Raw(`
		SELECT COUNT(*)
		FROM pg_attribute
		WHERE attrelid = to_regclass(?) AND attname = ? AND NOT attisdropped
	`, "public."+table, col).Scan(&n).Error
	return n > 0
}


// PUT /admin/announcement-themes/:id
func (h *AnnouncementThemeController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
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

	// --- Normalize slug jika dikirim ---
	if req.AnnouncementThemesSlug != nil {
		s := strings.TrimSpace(*req.AnnouncementThemesSlug)
		s = helper.GenerateSlug(s)
		if s == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak valid")
		}
		req.AnnouncementThemesSlug = &s
	}
	// -----------------------------------

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
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
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
