package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func IsMasjidAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔍 [MIDDLEWARE] IsMasjidAdmin active")
		log.Println("    Path  :", c.Path())
		log.Println("    Method:", c.Method())

		role, _ := c.Locals("userRole").(string)
		reqMasjid := c.Params("masjid_id")
		if reqMasjid == "" {
			// optional override via header untuk tools internal
			if h := c.Get("X-Masjid-ID"); h != "" {
				reqMasjid = h
			}
		}

		// =========================
		// 1) Bypass untuk OWNER
		// =========================
		if role == "owner" {
			log.Println("[MIDDLEWARE] Bypass: user is OWNER")
			c.Locals("role", "owner")
			if reqMasjid != "" {
				c.Locals("masjid_id", reqMasjid) // owner bebas memilih scope
				log.Println("[MIDDLEWARE] OWNER scope masjid_id:", reqMasjid)
			}
			return c.Next()
		}

		// Ambil daftar masjid yang dikelola dari token (untuk admin dkm / admin biasa)
		adminMasjids, ok := c.Locals("masjid_admin_ids").([]string)
		if !ok || len(adminMasjids) == 0 {
			log.Println("[MIDDLEWARE] Token tidak punya masjid_admin_ids")
			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak valid atau tidak memiliki akses masjid")
		}

		// =========================
		// 2) Admin DKM & admin biasa
		//    - Kalau ada :masjid_id → harus termasuk list adminMasjids
		//    - Kalau tidak ada → pakai default adminMasjids[0]
		// =========================
		chosen := adminMasjids[0]
		if reqMasjid != "" {
			okMatch := false
			for _, id := range adminMasjids {
				if id == reqMasjid {
					okMatch = true
					chosen = reqMasjid
					break
				}
			}
			if !okMatch {
				return fiber.NewError(fiber.StatusForbidden, "Bukan admin pada masjid yang diminta")
			}
		}

		c.Locals("masjid_id", chosen)
		c.Locals("role", role)

		log.Println("[MIDDLEWARE] Akses DIIJINKAN, role:", role, "masjid_id:", chosen)
		return c.Next()
	}
}
