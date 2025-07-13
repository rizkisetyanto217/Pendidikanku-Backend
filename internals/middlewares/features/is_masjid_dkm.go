package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func IsMasjidAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔍 DEBUG PARAMS:")
		log.Println("    Path : ", c.AllParams())
		log.Println("    Query: ", c.Context().QueryArgs().String())
		log.Println("    Body : ", string(c.Body()))

		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
			log.Println("[MIDDLEWARE] Bypass IsMasjidAdmin: user is owner")
			return c.Next()
		}

		var masjidID string

		if id := c.Params("id"); isValidUUID(id) {
			masjidID = id
			log.Println("[DEBUG] masjid_id dari path param:", masjidID)
		}

		if masjidID == "" {
			if id := c.Query("masjid_id"); isValidUUID(id) {
				masjidID = id
				log.Println("[DEBUG] masjid_id dari query param:", masjidID)
			}
		}

		if masjidID == "" {
			var bodyObj map[string]interface{}
			if err := c.BodyParser(&bodyObj); err == nil {
				log.Println("[DEBUG] bodyObj parsed:", bodyObj)
				masjidID = extractMasjidIDFromMap(bodyObj)
				log.Println("[DEBUG] masjid_id dari body map:", masjidID)
			}
		}

		if masjidID == "" {
			var bodyArr []map[string]interface{}
			if err := c.BodyParser(&bodyArr); err == nil && len(bodyArr) > 0 {
				log.Println("[DEBUG] bodyArr parsed:", bodyArr)
				masjidID = extractMasjidIDFromMap(bodyArr[0])
				log.Println("[DEBUG] masjid_id dari body array:", masjidID)
			}
		}

		if masjidID == "" {
			log.Println("[ERROR] masjid_id tetap kosong setelah semua pengecekan")
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak ditemukan di body, query, atau path")
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


// 🔧 Helper: Cek UUID valid
func isValidUUID(val string) bool {
	_, err := uuid.Parse(val)
	return err == nil
}

// 🔧 Helper: Ekstrak masjid_id dari berbagai nama field
func extractMasjidIDFromMap(m map[string]interface{}) string {
	fields := []string{
		"masjid_id",
		"lecture_masjid_id",
		"event_masjid_id",
		"notification_masjid_id",
		"post_masjid_id",
		"masjid_profile_masjid_id",
	}
	for _, field := range fields {
		if val, ok := m[field].(string); ok && isValidUUID(val) {
			return val
		}
	}
	return ""
}
