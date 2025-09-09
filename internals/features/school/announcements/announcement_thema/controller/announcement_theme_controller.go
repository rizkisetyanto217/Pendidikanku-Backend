// internals/features/lembaga/announcements/controller/announcement_theme_controller.go
package controller

import (
	"errors"
	"strconv"
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

// GET /admin/announcement-themes/:id
func (h *AnnouncementThemeController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findThemeWithTenantGuard(id, masjidID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Tema tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil tema")
	}
	return helper.JsonOK(c, "OK", annDTO.NewAnnouncementThemeResponse(m))
}

// GET /admin/announcement-themes
func (h *AnnouncementThemeController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var q annDTO.ListAnnouncementThemeQuery
	// default pagination
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := h.DB.Model(&annModel.AnnouncementThemeModel{}).
		Where("announcement_themes_masjid_id = ?", masjidID).
		Where("announcement_themes_deleted_at IS NULL") // hapus jika tidak pakai soft delete

	// ===== Filters
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

	// ===== Count total (sebelum limit/offset)
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data tema")
	}

	// ===== Sorting whitelist
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

	// ===== Pagination guard
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	// ===== Fetch
	var rows []annModel.AnnouncementThemeModel
	if err := tx.Limit(q.Limit).Offset(q.Offset).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data tema")
	}

	// ===== Map DTO
	resp := make([]*annDTO.AnnouncementThemeResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, annDTO.NewAnnouncementThemeResponse(&rows[i]))
	}

	// ===== Return konsisten dengan JsonList
	return helper.JsonList(c, resp, annDTO.Pagination{
		Limit:  q.Limit,
		Offset: q.Offset,
		Total:  int(total),
	})
}

// GET /admin/announcement-themes/search?q=tema&limit=10&active_only=true
func (h *AnnouncementThemeController) SearchByName(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// ambil query "q" (fallback ke "name")
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		q = strings.TrimSpace(c.Query("name"))
	}
	if q == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter q/name wajib diisi")
	}

	// limit (default 10, maks 50)
	limit := 10
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	// active_only (default true)
	activeOnly := true
	if v := strings.TrimSpace(c.Query("active_only")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			activeOnly = b
		}
	}

	tx := h.DB.Model(&annModel.AnnouncementThemeModel{}).
		Where("announcement_themes_masjid_id = ?", masjidID).
		Where("announcement_themes_deleted_at IS NULL")

	if activeOnly {
		tx = tx.Where("announcement_themes_is_active = TRUE")
	}

	like := "%" + q + "%"

	tx = tx.Where("announcement_themes_name ILIKE ?", like).
		Order("announcement_themes_name ASC").
		Limit(limit)

	var rows []annModel.AnnouncementThemeModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mencari tema")
	}

	resp := make([]*annDTO.AnnouncementThemeResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, annDTO.NewAnnouncementThemeResponse(&rows[i]))
	}
	return helper.JsonOK(c, "OK", resp)
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
