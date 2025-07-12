package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func IsMasjidAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
			log.Println("[MIDDLEWARE] Bypass IsMasjidAdmin: user is owner")
			return c.Next()
		}

		var masjidID string

		// ✅ 1. Cek dari query
		if id := c.Query("masjid_id"); id != "" {
			masjidID = id
		} else {
			// ✅ 2. Cek dari body
			validMasjidIDFields := []string{
				"masjid_id",
				"lecture_masjid_id",
				"event_masjid_id",
				"notification_masjid_id",
				"post_masjid_id",
				"masjid_profile_masjid_id",
			}

			// ✅ 2a. Coba parse sebagai object
			var bodyObj map[string]interface{}
			if err := c.BodyParser(&bodyObj); err == nil {
				for _, field := range validMasjidIDFields {
					if val, ok := bodyObj[field].(string); ok {
						if _, err := uuid.Parse(val); err == nil {
							masjidID = val
							break
						}
					}
				}
			}

			// ✅ 2b. Coba parse sebagai array (fallback)
			if masjidID == "" {
				var bodyArr []map[string]interface{}
				if err := c.BodyParser(&bodyArr); err == nil && len(bodyArr) > 0 {
					for _, field := range validMasjidIDFields {
						if val, ok := bodyArr[0][field].(string); ok {
							if _, err := uuid.Parse(val); err == nil {
								masjidID = val
								break
							}
						}
					}
				}
			}
		}

		if masjidID == "" {
			log.Println("[MIDDLEWARE] masjid_id tidak ditemukan di body atau query")
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak ditemukan di body atau query")
		}

		adminMasjids, ok := c.Locals("masjid_admin_ids").([]string)
		if !ok {
			log.Println("[MIDDLEWARE] masjid_admin_ids tidak tersedia di token")
			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak mengandung data masjid_admin_ids")
		}

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
