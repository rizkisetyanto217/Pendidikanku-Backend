package middleware

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ✅ Validasi UUID
func isValidUUID(val string) bool {
	_, err := uuid.Parse(val)
	return err == nil
}

// ✅ Resolver Map (modular, scalable)
var MasjidIDResolvers = map[string]func(*fiber.Ctx) string{
	"/api/a/lectures": func(c *fiber.Ctx) string {
	var body map[string]interface{}
	if err := c.BodyParser(&body); err == nil {
		if id, ok := body["lecture_masjid_id"].(string); ok && isValidUUID(id) {
			log.Println("[DEBUG] masjid_id dari body: lecture_masjid_id")
			return id
		}
	}
	if id := c.Query("masjid_id"); isValidUUID(id) {
		log.Println("[DEBUG] masjid_id dari query param (fallback)")
		return id
	}
	return ""
	},
	"/api/a/lecture-sessions": func(c *fiber.Ctx) string {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err == nil {
			if id, ok := body["lecture_session_masjid_id"].(string); ok && isValidUUID(id) {
				log.Println("[DEBUG] masjid_id dari body: lecture_session_masjid_id")
				return id
			}
		}
		if id := c.Query("masjid_id"); isValidUUID(id) {
			log.Println("[DEBUG] masjid_id dari query param")
			return id
		}
		return ""
	},

	"/api/a/lectures/by-masjid": func(c *fiber.Ctx) string {
	if ids, ok := c.Locals("masjid_admin_ids").([]string); ok && len(ids) > 0 && isValidUUID(ids[0]) {
		log.Println("[DEBUG] masjid_id dari token masjid_admin_ids")
		return ids[0]
		}
		return ""
	},


	"/api/a/posts": func(c *fiber.Ctx) string {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err == nil {
			if id, ok := body["post_masjid_id"].(string); ok && isValidUUID(id) {
				log.Println("[DEBUG] masjid_id dari body: post_masjid_id")
				return id
			}
		}
		return ""
	},

	// 🔧 Tambahkan endpoint lain di sini sesuai kebutuhan
}

// ✅ Ambil masjid_id pakai resolver per route, fallback ke DB jika perlu
func getMasjidIDFromRequest(c *fiber.Ctx, db *gorm.DB) string {
	path := c.Route().Path

	// 🔍 Resolver spesifik
	if resolver, ok := MasjidIDResolvers[path]; ok {
		if id := resolver(c); id != "" {
			return id
		}
	}

	// 🔍 Resolver prefix
	for prefix, resolver := range MasjidIDResolvers {
		if strings.HasPrefix(path, prefix) {
			if id := resolver(c); id != "" {
				return id
			}
		}
	}

	// 🔍 Fallback DB: dari lecture_session_id
	if strings.HasPrefix(path, "/api/a/lecture-sessions/") {
		sessionID := c.Params("id")
		if isValidUUID(sessionID) {
			var masjidID string
			err := db.Raw(`SELECT lecture_session_masjid_id FROM lecture_sessions WHERE lecture_session_id = ?`, sessionID).Scan(&masjidID).Error
			if err == nil && isValidUUID(masjidID) {
				log.Println("[DEBUG] masjid_id dari DB fallback: lecture_sessions")
				return masjidID
			}
		}
	}

	// 🔍 Fallback DB: dari lecture_id
	if strings.HasPrefix(path, "/api/a/lectures/") {
		lectureID := c.Params("id")
		if isValidUUID(lectureID) {
			var masjidID string
			err := db.Raw(`SELECT lecture_masjid_id FROM lectures WHERE lecture_id = ?`, lectureID).Scan(&masjidID).Error
			if err == nil && isValidUUID(masjidID) {
				log.Println("[DEBUG] masjid_id dari DB fallback: lectures")
				return masjidID
			}
		}
	}

	log.Println("[WARN] Tidak ada resolver masjid_id untuk path:", path)
	return ""
}

// ✅ Middleware utama
func IsMasjidAdmin(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔍 DEBUG PARAMS:")
		log.Println("    Path : ", c.AllParams())
		log.Println("    Query: ", c.Context().QueryArgs().String())
		log.Println("    Body : ", string(c.Body()))

		// 🚫 Bypass untuk owner
		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
			log.Println("[MIDDLEWARE] Bypass IsMasjidAdmin: user is owner")
			return c.Next()
		}

		masjidID := getMasjidIDFromRequest(c, db)
		if masjidID == "" {
			log.Println("[ERROR] masjid_id tidak ditemukan")
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak ditemukan")
		}

		adminMasjids, ok := c.Locals("masjid_admin_ids").([]string)
		if !ok {
			log.Println("[MIDDLEWARE] masjid_admin_ids tidak tersedia di token")
			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak mengandung data masjid_admin_ids")
		}

		for _, id := range adminMasjids {
			if id == masjidID {
				log.Println("[MIDDLEWARE] Akses DIIJINKAN ke masjid_id:", masjidID)
				c.Locals("masjid_id", masjidID) // ✅ Simpan ke context
				return c.Next()
			}
		}


		log.Println("[MIDDLEWARE] Akses DITOLAK ke masjid_id:", masjidID)
		return fiber.NewError(fiber.StatusForbidden, "Kamu bukan admin masjid ini")
	}
}
