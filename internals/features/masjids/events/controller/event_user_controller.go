package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"
	helper "masjidku_backend/internals/helpers"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// 🟢 GET /api/u/events/by-masjid/:slug?page=&limit=
func (ctrl *EventController) GetEventsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
        return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak boleh kosong")
	}

	// Ambil masjid_id dari slug
	var m struct{ MasjidID string `gorm:"column:masjid_id"` }
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Take(&m).Error; err != nil || m.MasjidID == "" {
		log.Printf("[ERROR] Masjid slug '%s' tidak ditemukan: %v", slug, err)
        return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 { page = 1 }
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 { limit = 10 }
	if limit > 100 { limit = 100 }
	offset := (page - 1) * limit

	// Hitung total
	var total int64
	if err := ctrl.DB.Model(&model.EventModel{}).
		Where("event_masjid_id = ?", m.MasjidID).
		Count(&total).Error; err != nil {
		log.Printf("[ERROR] Count events by slug '%s': %v", slug, err)
        return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung event")
	}

	// Ambil data page
	var events []model.EventModel
	if err := ctrl.DB.
		Where("event_masjid_id = ?", m.MasjidID).
		Order("event_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&events).Error; err != nil {
		log.Printf("[ERROR] Gagal ambil events by slug '%s': %v", slug, err)
        return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil event")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"masjid_slug": slug,
	}

	return helper.JsonList(c, dto.ToEventResponseList(events), pagination)
}
