package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func IsMasjidAdminOrTeacher() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔍 DEBUG PARAMS:")
		log.Println("    Path : ", c.AllParams())
		log.Println("    Query: ", c.Context().QueryArgs().String())
		log.Println("    Body : ", string(c.Body()))

		// Bypass jika owner
		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
			log.Println("[MIDDLEWARE] Bypass IsMasjidAdminOrTeacher: user is owner")
			return c.Next()
		}

		var masjidID string

		// 1️⃣ Dari path
		if id := c.Params("id"); isValidUUID(id) {
			masjidID = id
			log.Println("[DEBUG] masjid_id dari path param:", masjidID)
		}

		// 2️⃣ Dari query
		if masjidID == "" {
			if id := c.Query("masjid_id"); isValidUUID(id) {
				masjidID = id
				log.Println("[DEBUG] masjid_id dari query param:", masjidID)
			}
		}

		// 3️⃣ Dari body object
		if masjidID == "" {
			var body map[string]interface{}
			if err := c.BodyParser(&body); err == nil {
				masjidID = extractMasjidIDFromMap(body)
				log.Println("[DEBUG] masjid_id dari body map:", masjidID)
			}
		}

		// 4️⃣ Dari array of body
		if masjidID == "" {
			var bodyArr []map[string]interface{}
			if err := c.BodyParser(&bodyArr); err == nil && len(bodyArr) > 0 {
				masjidID = extractMasjidIDFromMap(bodyArr[0])
				log.Println("[DEBUG] masjid_id dari body array:", masjidID)
			}
		}

		if masjidID == "" {
			log.Println("[ERROR] masjid_id tetap kosong setelah semua pengecekan")
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak ditemukan")
		}

		// 5️⃣ Cek role dan akses masjid_id
		if role, ok := c.Locals("userRole").(string); ok {
			switch role {
			case "dkm":
				if ids, ok := c.Locals("masjid_admin_ids").([]string); ok {
					for _, id := range ids {
						if id == masjidID {
							log.Println("[MIDDLEWARE] DKM punya akses ke masjid_id:", masjidID)
							return c.Next()
						}
					}
				}
			case "teacher":
				if ids, ok := c.Locals("teacher_masjid_ids").([]string); ok {
					for _, id := range ids {
						if id == masjidID {
							log.Println("[MIDDLEWARE] Guru punya akses ke masjid_id:", masjidID)
							return c.Next()
						}
					}
				}
			}
		}

		log.Println("[MIDDLEWARE] Akses DITOLAK ke masjid_id:", masjidID)
		return fiber.NewError(fiber.StatusForbidden, "Kamu tidak punya akses ke masjid ini")
	}
}
