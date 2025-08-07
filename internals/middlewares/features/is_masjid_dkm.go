package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func IsMasjidAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔍 [MIDDLEWARE] IsMasjidAdminSimple active")
		log.Println("    Path  :", c.Path())
		log.Println("    Method:", c.Method())

		// ✅ Owner bypass
		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
			log.Println("[MIDDLEWARE] Bypass: user is owner")
			c.Locals("role", role) // <== penting
			return c.Next()
		}

		// ✅ Ambil masjid_id dari token
		adminMasjids, ok := c.Locals("masjid_admin_ids").([]string)
		if !ok || len(adminMasjids) == 0 {
			log.Println("[MIDDLEWARE] Token tidak punya masjid_admin_ids")
			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak valid atau tidak memiliki akses masjid")
		}

		// ✅ Inject masjid_id
		masjidID := adminMasjids[0]
		c.Locals("masjid_id", masjidID)

		// ✅ Inject role dari token
		if role, ok := c.Locals("userRole").(string); ok {
			c.Locals("role", role)
		}

		log.Println("[MIDDLEWARE] Akses DIIJINKAN, masjid_id:", masjidID)
		return c.Next()
	}
}

// package middleware

// import (
// 	"log"
// 	"strings"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// func isValidUUID(val string) bool {
// 	_, err := uuid.Parse(val)
// 	return err == nil
// }


// var MasjidIDResolvers = map[string]func(*fiber.Ctx, *gorm.DB) string{
// 	"POST /api/a/lectures": func(c *fiber.Ctx, db *gorm.DB) string {
// 		// Coba ambil dari form-data (karena kamu pakai multipart/form-data di Postman)
// 		if id := c.FormValue("lecture_masjid_id"); isValidUUID(id) {
// 			log.Println("[DEBUG] masjid_id dari form-data: lecture_masjid_id")
// 			return id
// 		}

// 		// Optional: fallback jika pakai JSON
// 		var body map[string]interface{}
// 		if err := c.BodyParser(&body); err == nil {
// 			if id, ok := body["lecture_masjid_id"].(string); ok && isValidUUID(id) {
// 				log.Println("[DEBUG] masjid_id dari JSON body: lecture_masjid_id")
// 				return id
// 			}
// 		}

// 		// Fallback dari query param
// 		if id := c.Query("masjid_id"); isValidUUID(id) {
// 			log.Println("[DEBUG] masjid_id dari query param (fallback)")
// 			return id
// 		}

// 		log.Println("[WARN] masjid_id tetap tidak ditemukan dalam resolver POST /api/a/lectures")
// 		return ""
// 	},

// 	"POST /api/a/lecture-sessions": func(c *fiber.Ctx, db *gorm.DB) string {
// 		if id := c.FormValue("lecture_session_masjid_id"); isValidUUID(id) {
// 			log.Println("[DEBUG] masjid_id dari form-data: lecture_session_masjid_id")
// 			return id
// 		}
// 		var body map[string]interface{}
// 		if err := c.BodyParser(&body); err == nil {
// 			if id, ok := body["lecture_session_masjid_id"].(string); ok && isValidUUID(id) {
// 				log.Println("[DEBUG] masjid_id dari body: lecture_session_masjid_id")
// 				return id
// 			}
// 		}
// 		if id := c.Query("masjid_id"); isValidUUID(id) {
// 			log.Println("[DEBUG] masjid_id dari query param")
// 			return id
// 		}
// 		return ""
// 	},

// 	"GET /api/a/lectures/by-masjid": func(c *fiber.Ctx, db *gorm.DB) string {
// 		if ids, ok := c.Locals("masjid_admin_ids").([]string); ok && len(ids) > 0 && isValidUUID(ids[0]) {
// 			log.Println("[DEBUG] masjid_id dari token masjid_admin_ids")
// 			return ids[0]
// 		}
// 		return ""
// 	},

// 	"GET /api/a/lecture-sessions/by-masjid": func(c *fiber.Ctx, db *gorm.DB) string {
// 		if ids, ok := c.Locals("masjid_admin_ids").([]string); ok && len(ids) > 0 {
// 			log.Println("[DEBUG] masjid_id dari Locals: masjid_admin_ids[0]")
// 			return ids[0]
// 		}
// 		return ""
// 	},

// 	"GET /api/a/lecture-sessions/by-lecture-sessions/": func(c *fiber.Ctx, db *gorm.DB) string {
// 		lectureID := c.Params("lecture_id")
// 		if isValidUUID(lectureID) {
// 			var masjidID string
// 			err := db.Raw(`SELECT lecture_masjid_id FROM lectures WHERE lecture_id = ?`, lectureID).Scan(&masjidID).Error
// 			if err == nil && isValidUUID(masjidID) {
// 				log.Println("[DEBUG] masjid_id dari DB: lectures.lecture_masjid_id (by lecture_id)")
// 				return masjidID
// 			}
// 		}
// 		log.Println("[WARN] Tidak bisa resolve masjid_id dari lecture_id")
// 		return ""
// 	},

// 	"POST /api/a/posts": func(c *fiber.Ctx, db *gorm.DB) string {
// 		var body map[string]interface{}
// 		if err := c.BodyParser(&body); err == nil {
// 			if id, ok := body["post_masjid_id"].(string); ok && isValidUUID(id) {
// 				log.Println("[DEBUG] masjid_id dari body: post_masjid_id")
// 				return id
// 			}
// 		}
// 		return ""
// 	},

// 	"GET /api/a/advices/by-lecture": func(c *fiber.Ctx, db *gorm.DB) string {
// 		parts := strings.Split(c.Path(), "/")
// 		if len(parts) >= 6 {
// 			lectureID := parts[5]
// 			if isValidUUID(lectureID) {
// 				var masjidID string
// 				err := db.Raw(`SELECT lecture_masjid_id FROM lectures WHERE lecture_id = ?`, lectureID).Scan(&masjidID).Error
// 				if err == nil && isValidUUID(masjidID) {
// 					log.Println("[DEBUG] masjid_id dari DB: lectures.lecture_masjid_id (by lecture_id)")
// 					return masjidID
// 				}
// 			}
// 		}
// 		log.Println("[WARN] Tidak bisa resolve masjid_id dari lecture_id (resolver advices)")
// 		return ""
// 	},
// }

// func getMasjidIDFromRequest(c *fiber.Ctx, db *gorm.DB) string {
// 	key := c.Method() + " " + c.Path()
// 	log.Println("[DEBUG] Key resolver:", key)

// 	// Exact match: METHOD + PATH
// 	if resolver, ok := MasjidIDResolvers[key]; ok {
// 		if id := resolver(c, db); id != "" {
// 			return id
// 		}
// 	}

// 	// Prefix match (untuk path dinamis seperti /by-lecture/:id)
// 	for prefix, resolver := range MasjidIDResolvers {
// 		if strings.HasPrefix(key, prefix) {
// 			if id := resolver(c, db); id != "" {
// 				return id
// 			}
// 		}
// 	}

// 	// Fallback dari lecture_session_id (misal: /api/a/lecture-sessions/:id)
// 	if strings.HasPrefix(c.Path(), "/api/a/lecture-sessions/") {
// 		sessionID := c.Params("id")
// 		if isValidUUID(sessionID) {
// 			var masjidID string
// 			err := db.Raw(`SELECT lecture_session_masjid_id FROM lecture_sessions WHERE lecture_session_id = ?`, sessionID).Scan(&masjidID).Error
// 			if err == nil && isValidUUID(masjidID) {
// 				log.Println("[DEBUG] masjid_id dari DB fallback: lecture_sessions")
// 				return masjidID
// 			}
// 		}
// 	}

// 	// Fallback dari lecture_id (misal: /api/a/lectures/:id)
// 	if strings.HasPrefix(c.Path(), "/api/a/lectures/") {
// 		lectureID := c.Params("id")
// 		if isValidUUID(lectureID) {
// 			var masjidID string
// 			err := db.Raw(`SELECT lecture_masjid_id FROM lectures WHERE lecture_id = ?`, lectureID).Scan(&masjidID).Error
// 			if err == nil && isValidUUID(masjidID) {
// 				log.Println("[DEBUG] masjid_id dari DB fallback: lectures")
// 				return masjidID
// 			}
// 		}
// 	}

// 	log.Println("[WARN] Tidak ada resolver masjid_id untuk path:", c.Path())
// 	return ""
// }


// // ✅ Middleware utama
// func IsMasjidAdmin(db *gorm.DB) fiber.Handler {
// 	return func(c *fiber.Ctx) error {
// 		log.Println("🔍 DEBUG PARAMS:")
// 		log.Println("    Path : ", c.Path())
// 		log.Println("    Query: ", c.Context().QueryArgs().String())
// 		log.Println("    Body : ", string(c.Body()))

// 		// Bypass untuk owner
// 		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
// 			log.Println("[MIDDLEWARE] Bypass IsMasjidAdmin: user is owner")
// 			return c.Next()
// 		}

// 		masjidID := getMasjidIDFromRequest(c, db)
// 		if masjidID == "" {
// 			log.Println("[ERROR] masjid_id tidak ditemukan")
// 			return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak ditemukan")
// 		}

// 		adminMasjids, ok := c.Locals("masjid_admin_ids").([]string)
// 		if !ok {
// 			log.Println("[MIDDLEWARE] masjid_admin_ids tidak tersedia di token")
// 			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak mengandung data masjid_admin_ids")
// 		}

// 		for _, id := range adminMasjids {
// 			if id == masjidID {
// 				log.Println("[MIDDLEWARE] Akses DIIJINKAN ke masjid_id:", masjidID)
// 				c.Locals("masjid_id", masjidID)
// 				return c.Next()
// 			}
// 		}

// 		log.Println("[MIDDLEWARE] Akses DITOLAK ke masjid_id:", masjidID)
// 		return fiber.NewError(fiber.StatusForbidden, "Kamu bukan admin masjid ini")
// 	}
// }
