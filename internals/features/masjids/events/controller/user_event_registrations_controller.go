package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserEventRegistrationController struct {
	DB *gorm.DB
}

func NewUserEventRegistrationController(db *gorm.DB) *UserEventRegistrationController {
	return &UserEventRegistrationController{DB: db}
}

// 🟢 POST /api/a/user-event-registrations
func (ctrl *UserEventRegistrationController) CreateRegistration(c *fiber.Ctx) error {
	var req dto.UserEventRegistrationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Permintaan tidak valid", "error": err.Error()})
	}

	registration := req.ToModel()
	if err := ctrl.DB.Create(registration).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan registrasi event: %v", err)
		return c.Status(500).JSON(fiber.Map{"message": "Gagal menyimpan registrasi", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Registrasi berhasil",
		"data":    dto.ToUserEventRegistrationResponse(registration),
	})
}

// 🟢 POST /api/a/user-event-registrations/by-event
func (ctrl *UserEventRegistrationController) GetRegistrantsByEvent(c *fiber.Ctx) error {
	var payload struct {
		EventID string `json:"event_id"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Permintaan tidak valid", "error": err.Error()})
	}

	var registrations []model.UserEventRegistrationModel
	if err := ctrl.DB.Where("user_event_registration_event_id = ?", payload.EventID).Find(&registrations).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal mengambil data", "error": err.Error()})
	}

	var responses []dto.UserEventRegistrationResponse
	for _, r := range registrations {
		responses = append(responses, *dto.ToUserEventRegistrationResponse(&r))
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil data registrasi",
		"data":    responses,
	})
}
