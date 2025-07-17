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


// ✅ Middleware utama
func IsMasjidAdmin(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔍 DEBUG PARAMS:", c.AllParams())
		log.Println("    Body:", string(c.Body()))

		if role, ok := c.Locals("userRole").(string); ok && role == "owner" {
			log.Println("[MIDDLEWARE] Bypass: user is owner")
			return c.Next()
		}

		masjidID := getMasjidIDFromRequest(c, db)
		if masjidID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak ditemukan")
		}

		adminMasjids, ok := c.Locals("masjid_admin_ids").([]string)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak mengandung data masjid_admin_ids")
		}

		for _, id := range adminMasjids {
			if id == masjidID {
				c.Locals("masjid_id", masjidID)
				return c.Next()
			}
		}
		return fiber.NewError(fiber.StatusForbidden, "Kamu bukan admin masjid ini")
	}
}

// ✅ Ambil masjid_id pakai resolver per route
func getMasjidIDFromRequest(c *fiber.Ctx, db *gorm.DB) string {
	path := c.Path()
	if resolver, ok := MasjidIDResolvers[path]; ok {
		return resolver(c, db)
	}
	for prefix, resolver := range MasjidIDResolvers {
		if strings.HasPrefix(path, prefix) {
			return resolver(c, db)
		}
	}
	// fallback by lecture_session_id or lecture_id
	if strings.HasPrefix(path, "/api/a/lecture-sessions/") {
		sessionID := c.Params("id")
		if isValidUUID(sessionID) {
			var masjidID string
			err := db.Raw(`SELECT lecture_session_masjid_id FROM lecture_sessions WHERE lecture_session_id = ?`, sessionID).Scan(&masjidID).Error
			if err == nil && isValidUUID(masjidID) {
				return masjidID
			}
		}
	}
	if strings.HasPrefix(path, "/api/a/lectures/") {
		lectureID := c.Params("id")
		if isValidUUID(lectureID) {
			var masjidID string
			err := db.Raw(`SELECT lecture_masjid_id FROM lectures WHERE lecture_id = ?`, lectureID).Scan(&masjidID).Error
			if err == nil && isValidUUID(masjidID) {
				return masjidID
			}
		}
	}
	return ""
}