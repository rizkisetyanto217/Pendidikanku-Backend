package middleware

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ✅ Resolver per path
var MasjidIDResolvers = map[string]func(*fiber.Ctx, *gorm.DB) string{
	"/api/a/lectures": resolveMasjidIDFromBody("lecture_masjid_id"),
	"/api/a/lecture-sessions": resolveMasjidIDFromBody("lecture_session_masjid_id"),
	"/api/a/posts": resolveMasjidIDFromBody("post_masjid_id"),
	"/api/a/lectures/by-masjid": resolveMasjidIDFromLocals("masjid_admin_ids"),
	"/api/a/lecture-sessions/by-masjid": resolveMasjidIDFromLocals("masjid_admin_ids"),
	"/api/a/advices/by-lecture": resolveMasjidIDFromLectureParam(5),
	"/api/a/lecture-sessions/by-lecture-sessions/": resolveMasjidIDFromLectureParamPath("lecture_id"),
	"/api/a/masjid-teachers": resolveMasjidIDWithFallback(
	resolveMasjidIDFromBody("masjid_teachers_masjid_id"),
	resolveMasjidIDFromLocals("masjid_admin_ids"),),
	"/api/a/masjid-teachers/by-masjid": resolveMasjidIDFromLocals("masjid_admin_ids"),
}

// ✅ Resolver generator
func resolveMasjidIDFromBody(field string) func(*fiber.Ctx, *gorm.DB) string {
	return func(c *fiber.Ctx, db *gorm.DB) string {
		var body map[string]interface{}
		if err := c.BodyParser(&body); err == nil {
			if id, ok := body[field].(string); ok && isValidUUID(id) {
				log.Println("[DEBUG] masjid_id dari body:", field)
				return id
			}
		}
		if id := c.Query("masjid_id"); isValidUUID(id) {
			log.Println("[DEBUG] masjid_id dari query param")
			return id
		}
		return ""
	}
}

func resolveMasjidIDFromLocals(field string) func(*fiber.Ctx, *gorm.DB) string {
	return func(c *fiber.Ctx, db *gorm.DB) string {
		if ids, ok := c.Locals(field).([]string); ok && len(ids) > 0 && isValidUUID(ids[0]) {
			log.Println("[DEBUG] masjid_id dari locals:", field)
			return ids[0]
		}
		return ""
	}
}

func resolveMasjidIDFromLectureParam(paramIndex int) func(*fiber.Ctx, *gorm.DB) string {
	return func(c *fiber.Ctx, db *gorm.DB) string {
		parts := strings.Split(c.Path(), "/")
		if len(parts) > paramIndex {
			lectureID := parts[paramIndex]
			if isValidUUID(lectureID) {
				var masjidID string
				err := db.Raw(`SELECT lecture_masjid_id FROM lectures WHERE lecture_id = ?`, lectureID).Scan(&masjidID).Error
				if err == nil && isValidUUID(masjidID) {
					log.Println("[DEBUG] masjid_id dari DB: lectures.lecture_masjid_id (by lecture_id)")
					return masjidID
				}
			}
		}
		log.Println("[WARN] Tidak bisa resolve masjid_id dari lecture_id (resolver)")
		return ""
	}
}

func resolveMasjidIDWithFallback(primary, fallback func(*fiber.Ctx, *gorm.DB) string) func(*fiber.Ctx, *gorm.DB) string {
	return func(c *fiber.Ctx, db *gorm.DB) string {
		if id := primary(c, db); id != "" {
			return id
		}
		return fallback(c, db)
	}
}

func resolveMasjidIDFromLectureParamPath(paramName string) func(*fiber.Ctx, *gorm.DB) string {
	return func(c *fiber.Ctx, db *gorm.DB) string {
		lectureID := c.Params(paramName)
		if isValidUUID(lectureID) {
			var masjidID string
			err := db.Raw(`SELECT lecture_masjid_id FROM lectures WHERE lecture_id = ?`, lectureID).Scan(&masjidID).Error
			if err == nil && isValidUUID(masjidID) {
				log.Println("[DEBUG] masjid_id dari DB: lectures.lecture_masjid_id (by lecture_id)")
				return masjidID
			}
		}
		log.Println("[WARN] Tidak bisa resolve masjid_id dari lecture_id (resolver)")
		return ""
	}
}
