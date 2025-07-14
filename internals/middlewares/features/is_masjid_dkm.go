package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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


func getMasjidIDFromRequest(c *fiber.Ctx) string {
	// 1. Query param
	if id := c.Query("masjid_id"); isValidUUID(id) {
		log.Println("[DEBUG] masjid_id dari query:", id)
		return id
	}

	// 2. Body (object)
	var bodyObj map[string]interface{}
	if err := c.BodyParser(&bodyObj); err == nil {
		if id := extractMasjidIDFromMap(bodyObj); id != "" {
			log.Println("[DEBUG] masjid_id dari body object:", id)
			return id
		}
	}

	// 3. Body (array of object)
	var bodyArr []map[string]interface{}
	if err := c.BodyParser(&bodyArr); err == nil && len(bodyArr) > 0 {
		if id := extractMasjidIDFromMap(bodyArr[0]); id != "" {
			log.Println("[DEBUG] masjid_id dari body array:", id)
			return id
		}
	}

	// 4. (Opsional) Path param, misal: /masjid/:masjid_id/...
	if id := c.Params("masjid_id"); isValidUUID(id) {
		log.Println("[DEBUG] masjid_id dari path:", id)
		return id
	}

	// 5. Gagal ditemukan
	return ""
}



// ✨ Middleware utama
func IsMasjidAdmin(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔍 DEBUG PARAMS:")
		log.Println("    Path : ", c.AllParams())
		log.Println("    Query: ", c.Context().QueryArgs().String())
		log.Println("    Body : ", string(c.Body()))

		// Bypass jika role owner
		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
			log.Println("[MIDDLEWARE] Bypass IsMasjidAdmin: user is owner")
			return c.Next()
		}

		// Ambil masjid_id dari berbagai sumber
		masjidID := getMasjidIDFromRequest(c)
		if masjidID == "" {
			log.Println("[ERROR] masjid_id tidak ditemukan")
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak ditemukan di body, query, path, atau DB lookup")
		}

		// Ambil daftar masjid_id yang dimiliki admin
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
