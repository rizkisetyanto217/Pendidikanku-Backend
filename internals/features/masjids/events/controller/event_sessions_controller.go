package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/events/dto"
	"masjidku_backend/internals/features/masjids/events/model"

	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventSessionController struct {
	DB *gorm.DB
}

func NewEventSessionController(db *gorm.DB) *EventSessionController {
	return &EventSessionController{DB: db}
}

// 🟢 POST /api/a/event-sessions
// 🟢 POST /api/a/event-sessions
func (ctrl *EventSessionController) CreateEventSession(c *fiber.Ctx) error {
	// Ambil user_id dari token (middleware harus sudah set ini di Locals)
	userIDRaw := c.Locals("user_id")
	userIDStr, ok := userIDRaw.(string)
	if !ok || userIDStr == "" {
		log.Println("[ERROR] Gagal mendapatkan user_id dari token")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "User tidak terautentikasi",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Printf("[ERROR] Gagal parsing user_id: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "User ID tidak valid",
		})
	}

	var req dto.EventSessionRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Body parser gagal: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Permintaan tidak valid",
			"error":   err.Error(),
		})
	}

	session := req.ToModel()
	session.EventSessionCreatedBy = &userID // ✅ Set created_by dari token

	if err := ctrl.DB.Create(session).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan event session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menyimpan event session",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Event session berhasil dibuat",
		"data":    dto.ToEventSessionResponse(session),
	})
}

// 🟢 GET /api/u/event-sessions/by-event/:event_id
func (ctrl *EventSessionController) GetEventSessionsByEvent(c *fiber.Ctx) error {
	eventID := c.Params("event_id")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Event ID tidak boleh kosong",
		})
	}

	var sessions []model.EventSessionModel
	if err := ctrl.DB.Where("event_session_event_id = ?", eventID).
		Order("event_session_start_time ASC").
		Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil event sessions: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil event sessions",
			"error":   err.Error(),
		})
	}

	now := time.Now()
	var result []dto.EventSessionResponse
	for _, s := range sessions {
		status := "upcoming"
		if now.After(s.EventSessionStartTime) && now.Before(s.EventSessionEndTime) {
			status = "ongoing"
		} else if now.After(s.EventSessionEndTime) {
			status = "completed"
		}

		resp := dto.ToEventSessionResponse(&s)
		resp.EventSessionStatus = status
		result = append(result, *resp)
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil event sessions",
		"data":    result,
	})
}

// 🟢 GET /api/u/event-sessions/all
func (ctrl *EventSessionController) GetAllEventSessions(c *fiber.Ctx) error {
	var sessions []model.EventSessionModel
	if err := ctrl.DB.Order("event_session_start_time DESC").Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil semua event session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data event sessions",
			"error":   err.Error(),
		})
	}

	now := time.Now()
	var result []dto.EventSessionResponse
	for _, s := range sessions {
		status := "upcoming"
		if now.After(s.EventSessionStartTime) && now.Before(s.EventSessionEndTime) {
			status = "ongoing"
		} else if now.After(s.EventSessionEndTime) {
			status = "completed"
		}

		resp := dto.ToEventSessionResponse(&s)
		resp.EventSessionStatus = status
		result = append(result, *resp)
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil semua event session",
		"data":    result,
	})
}

func (ctrl *EventSessionController) GetUpcomingEventSessions(c *fiber.Ctx) error {
	var sessions []model.EventSessionModel

	// CATATAN PENTING UNTUK ROUTING DI FILE UTAMA (misal: main.go atau routes.go):
	//
	// Untuk menangani permintaan ke: /public/event-sessions/upcoming/masjid-id (TANPA ID)
	// Anda HARUS menambahkan rute ini LEBIH DULU:
	// ```go
	// session.Get("/upcoming/masjid-id", func(c *fiber.Ctx) error {
	//    // Ini akan terpicu jika URL adalah /public/event-sessions/upcoming/masjid-id
	//    log.Printf("[WARNING] Request received for /upcoming/masjid-id without a specific ID. Returning Bad Request.")
	//    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	//        "message": "ID masjid diperlukan di URL path untuk endpoint ini.",
	//        "error":   "Missing masjid_id in URL path",
	//    })
	// })
	// ```
	//
	// DAN KEMUDIAN, rute Anda yang sudah ada untuk menangani permintaan DENGAN ID:
	// ```go
	// session.Get("/upcoming/masjid-id/:masjid_id", ctrl.GetUpcomingEventSessions)
	// ```
	//
	// Perhatikan urutan definisi rute ini sangat krusial di Fiber.

	// 1. Ambil masjid_id dari PATH parameter
	// PERHATIKAN: Nama parameter di c.Params() HARUS SAMA dengan di definisi rute
	masjidIDStr := c.Params("masjid_id") // Nama parameter harus cocok dengan definisi rute (:masjid_id)

	// Inisialisasi builder kueri GORM
	// Filter awal untuk sesi yang akan datang dan bersifat publik
	query := ctrl.DB.
		Where("event_session_start_time > ? AND event_session_is_public = ?", time.Now(), true)

	// 2. Jika masjid_id ada (dari path), tambahkan filter ke kueri
	if masjidIDStr != "" {
		masjidID, err := uuid.Parse(masjidIDStr)
		if err != nil {
			log.Printf("[ERROR] Invalid masjid_id format from path: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Format ID masjid tidak valid",
				"error":   "Invalid UUID format for masjid_id in path",
			})
		}
		// Tambahkan filter berdasarkan event_session_masjid_id
		query = query.Where("event_session_masjid_id = ?", masjidID)
	} else {
		// Log ini akan muncul jika GetUpcomingEventSessions dipanggil melalui rute tanpa parameter ID
		// dan Flutter tidak mengirimkan ID di path.
		// Jika ini terus muncul dan Anda ingin masjid_id selalu ada, pastikan Flutter mengirimkannya.
		log.Printf("[INFO] GetUpcomingEventSessions dipanggil tanpa masjid_id di path. Mengambil semua event publik.")
	}

	// Ambil query parameter 'order' (jika ada)
	// Ini tetap query parameter dari URL seperti ?order=terbaru
	order := c.Query("order")
	if order != "" {
		// Contoh logika untuk order, jika diperlukan
		// if order == "terbaru" {
		// 	query = query.Order("event_session_created_at DESC")
		// }
		// Untuk saat ini, kita biarkan Order("event_session_start_time ASC") sebagai default
	}

	// 3. Lanjutkan dengan order dan Find
	if err := query.
		Order("event_session_start_time ASC"). // Tetap urutkan berdasarkan waktu mulai
		Find(&sessions).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil sesi event upcoming: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil sesi event upcoming",
			"error":   err.Error(),
		})
	}

	// Mengembalikan respons JSON dengan data sesi event
	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil sesi event yang akan datang",
		"data":    dto.ToEventSessionResponseList(sessions), // Asumsi ini mengonversi ke DTO yang sesuai
	})
}
