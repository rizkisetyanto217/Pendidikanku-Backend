package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func IsMasjidAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// ✅ Kalau role = owner, langsung bypass
		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
			log.Println("[MIDDLEWARE] Bypass IsMasjidAdmin: user is owner")
			return c.Next()
		}

		var masjidID string

		// ✅ 1. Cek dari query
		if id := c.Query("masjid_id"); id != "" {
			masjidID = id
		} else {
			// ✅ 2. Cek dari body dengan daftar field yang disetujui
			validMasjidIDFields := []string{
				"masjid_id",
				"lecture_masjid_id",
				"event_masjid_id",
				"notification_masjid_id",
				"post_masjid_id",
			}

			var body map[string]interface{}
			if err := c.BodyParser(&body); err == nil {
				for _, field := range validMasjidIDFields {
					if val, ok := body[field].(string); ok {
						if _, err := uuid.Parse(val); err == nil {
							masjidID = val
							break
						}
					}
				}
			}
		}

		if masjidID == "" {
			log.Println("[MIDDLEWARE] masjid_id tidak ditemukan di body atau query")
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak ditemukan di body atau query")
		}

		// ✅ 3. Ambil daftar masjid dari token
		adminMasjids, ok := c.Locals("masjid_admin_ids").([]string)
		if !ok {
			log.Println("[MIDDLEWARE] masjid_admin_ids tidak tersedia di token")
			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak mengandung data masjid_admin_ids")
		}

		// ✅ 4. Cocokkan masjid_id dari request dengan masjid yang dimiliki user
		for _, id := range adminMasjids {
			if id == masjidID {
				log.Println("[MIDDLEWARE] Akses DIIJINKAN ke masjid_id:", masjidID)
				return c.Next()
			}
		}

		log.Println("[MIDDLEWARE] Akses DITOLAK ke masjid_id:", masjidID)
		return fiber.NewError(fiber.StatusForbidden, "Kamu bukan admin masjid ini")
	}
}
