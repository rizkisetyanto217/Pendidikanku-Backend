package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"

	"github.com/gofiber/fiber/v2"
)

// 🟢 GET /api/u/events/by-masjid/:slug
func (ctrl *EventController) GetEventsByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Slug tidak boleh kosong",
		})
	}

	// Cari masjid_id dari slug
	type masjidRow struct {
		MasjidID string `gorm:"column:masjid_id"`
	}
	var m masjidRow
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Take(&m).Error; err != nil {
		log.Printf("[ERROR] Masjid dengan slug '%s' tidak ditemukan: %v", slug, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Masjid tidak ditemukan",
			"error":   err.Error(),
		})
	}

	// Ambil events berdasarkan masjid_id
	var events []model.EventModel
	if err := ctrl.DB.
		Where("event_masjid_id = ?", m.MasjidID).
		Order("event_created_at DESC").
		Find(&events).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil event by masjid slug '%s': %v", slug, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil event",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event berhasil diambil",
		"data":    dto.ToEventResponseList(events),
	})
}
