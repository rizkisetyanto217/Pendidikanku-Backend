package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type EventController struct {
	DB *gorm.DB
}

func NewEventController(db *gorm.DB) *EventController {
	return &EventController{DB: db}
}

// 🟢 POST /api/a/events
func (ctrl *EventController) CreateEvent(c *fiber.Ctx) error {
	var req dto.EventRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Body parser gagal: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Permintaan tidak valid",
			"error":   err.Error(),
		})
	}

	newEvent := req.ToModel()
	if err := ctrl.DB.Create(newEvent).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan event: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menyimpan event",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event berhasil ditambahkan",
		"data":    dto.ToEventResponse(newEvent),
	})
}

// 🟢 GET /api/u/events/id/:id
func (ctrl *EventController) GetEventByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Event ID tidak boleh kosong",
		})
	}

	var ev model.EventModel
	if err := ctrl.DB.Where("event_id = ?", id).First(&ev).Error; err != nil {
		log.Printf("[ERROR] Event dengan ID '%s' tidak ditemukan: %v", id, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Event tidak ditemukan",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event berhasil ditemukan",
		"data":    dto.ToEventResponse(&ev),
	})
}



// 🟡 PATCH /api/a/events/:id
func (ctrl *EventController) UpdateEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Event ID tidak boleh kosong",
		})
	}

	// Ambil record lama
	var ev model.EventModel
	if err := ctrl.DB.Where("event_id = ?", id).First(&ev).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Event tidak ditemukan",
			"error":   err.Error(),
		})
	}

	// Parse body
	var req dto.EventUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Permintaan tidak valid",
			"error":   err.Error(),
		})
	}

	updates := map[string]interface{}{}

	// Jika judul diupdate → slug ikut diupdate
	if req.EventTitle != nil {
		updates["event_title"] = *req.EventTitle
		updates["event_slug"] = dto.GenerateSlug(*req.EventTitle)
	}
	if req.EventDescription != nil {
		updates["event_description"] = *req.EventDescription
	}
	if req.EventLocation != nil {
		updates["event_location"] = *req.EventLocation
	}
	if req.EventMasjidID != nil {
		// (Opsional) precheck FK biar error jadi 404/400, bukan 500
		var cnt int64
		if err := ctrl.DB.Table("masjids").
			Where("masjid_id = ?", *req.EventMasjidID).
			Count(&cnt).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal memeriksa masjid",
				"error":   err.Error(),
			})
		}
		if cnt == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Masjid tidak ditemukan",
			})
		}
		updates["event_masjid_id"] = *req.EventMasjidID
	}

	if len(updates) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Tidak ada field yang diupdate",
		})
	}

	// Lakukan update
	if err := ctrl.DB.Model(&ev).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal memperbarui event",
			"error":   err.Error(),
		})
	}

	// Reload untuk response terbaru
	if err := ctrl.DB.Where("event_id = ?", id).First(&ev).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal memuat data event terbaru",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event berhasil diperbarui",
		"data":    dto.ToEventResponse(&ev),
	})
}


// 🟢 POST /api/a/events/by-masjid
func (ctrl *EventController) GetEventsByMasjid(c *fiber.Ctx) error {
	type Request struct {
		MasjidID string `json:"masjid_id"`
	}
	var body Request
	if err := c.BodyParser(&body); err != nil || body.MasjidID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Masjid ID tidak valid"})
	}

	var events []model.EventModel
	if err := ctrl.DB.
		Where("event_masjid_id = ?", body.MasjidID).
		Order("event_created_at DESC").
		Find(&events).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil data event: %v", err)
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

// 🟢 GET /api/a/events/all atau /api/u/events/all
func (ctrl *EventController) GetAllEvents(c *fiber.Ctx) error {
	var events []model.EventModel

	if err := ctrl.DB.Order("event_created_at DESC").Find(&events).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil semua event: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data event",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil semua event",
		"data":    dto.ToEventResponseList(events),
	})
}

// 🟢 GET /api/u/events/:slug
func (ctrl *EventController) GetEventBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Slug tidak boleh kosong",
		})
	}

	var event model.EventModel
	if err := ctrl.DB.Where("event_slug = ?", slug).First(&event).Error; err != nil {
		log.Printf("[ERROR] Event dengan slug '%s' tidak ditemukan: %v", slug, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Event tidak ditemukan",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event berhasil ditemukan",
		"data":    dto.ToEventResponse(&event),
	})
}
