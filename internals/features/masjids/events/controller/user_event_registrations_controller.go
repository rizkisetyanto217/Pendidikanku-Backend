package controller

import (
	"log"
	"strconv"
	"strings"

	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"
	helper "masjidku_backend/internals/helpers"

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
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	reg := req.ToModel()
	if err := ctrl.DB.Create(reg).Error; err != nil {
		// Tabrak UNIQUE(user_event_registration_event_session_id, user_event_registration_user_id)
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") ||
			strings.Contains(strings.ToLower(err.Error()), "unique constraint") {
			return helper.JsonError(c, fiber.StatusConflict, "Pengguna sudah terdaftar pada sesi ini")
		}
		log.Printf("[ERROR] create registration: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan registrasi")
	}

	return helper.JsonCreated(c, "Registrasi berhasil", dto.ToUserEventRegistrationResponse(reg))
}

// 🟢 POST /api/a/user-event-registrations/by-event
// Body optional: { "event_session_id": "...", "event_id": "...", "status": "registered", "masjid_id": "..." }
// Query: ?page=1&limit=10 (pagination)
func (ctrl *UserEventRegistrationController) GetRegistrantsByEvent(c *fiber.Ctx) error {
	// Ambil filter dari body (opsional)
	var payload struct {
		EventSessionID string `json:"event_session_id"`
		EventID        string `json:"event_id"`
		Status         string `json:"status"`
		MasjidID       string `json:"masjid_id"`
	}
	_ = c.BodyParser(&payload) // abaikan error; semua opsional

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Base query
	q := ctrl.DB.Table("user_event_registrations")
	// Jika filter by event_id → join ke event_sessions
	if payload.EventID != "" {
		q = q.Joins("JOIN event_sessions es ON es.event_session_id = user_event_registrations.user_event_registration_event_session_id").
			Where("es.event_session_event_id = ?", payload.EventID)
	}
	// Jika filter by event_session_id (lebih spesifik)
	if payload.EventSessionID != "" {
		q = q.Where("user_event_registration_event_session_id = ?", payload.EventSessionID)
	}
	// Filter status (opsional)
	if payload.Status != "" {
		q = q.Where("user_event_registration_status = ?", payload.Status)
	}
	// Filter masjid (opsional)
	if payload.MasjidID != "" {
		q = q.Where("user_event_registration_masjid_id = ?", payload.MasjidID)
	}

	// Count total
	var total int64
	if err := q.Count(&total).Error; err != nil {
		log.Printf("[ERROR] count registrations: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// Ambil data page
	var registrations []model.UserEventRegistrationModel
	if err := q.Order("user_event_registration_registered_at DESC").
		Limit(limit).Offset(offset).
		Find(&registrations).Error; err != nil {
		log.Printf("[ERROR] get registrations: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Map response DTO
	responses := make([]dto.UserEventRegistrationResponse, 0, len(registrations))
	for i := range registrations {
		responses = append(responses, *dto.ToUserEventRegistrationResponse(&registrations[i]))
	}

	// Pagination meta
	pagination := fiber.Map{
		"page":         page,
		"limit":        limit,
		"total":        total,
		"total_pages":  int((total + int64(limit) - 1) / int64(limit)),
		"has_next":     int64(page*limit) < total,
		"has_prev":     page > 1,
		"event_id":     payload.EventID,
		"event_session_id": payload.EventSessionID,
		"masjid_id":    payload.MasjidID,
		"status":       payload.Status,
	}

	return helper.JsonList(c, responses, pagination)
}
