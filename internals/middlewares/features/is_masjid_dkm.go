// file: internals/middlewares/features/is_masjid_admin.go
package middleware

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"

	"masjidku_backend/internals/constants"
	helper "masjidku_backend/internals/helpers"
)

// --- helper ekstra ---
func extractMasjidID(c *fiber.Ctx) string {
	// 1) param
	if v := strings.TrimSpace(c.Params("masjid_id")); v != "" {
		return v
	}
	// 2) query
	if v := strings.TrimSpace(c.Query("masjid_id")); v != "" {
		return v
	}
	// 3) header
	if v := strings.TrimSpace(c.Get("X-Masjid-ID")); v != "" {
		return v
	}
	// 4) body JSON (best-effort; abaikan error)
	var body map[string]any
	if b := c.Body(); len(b) > 0 {
		_ = json.Unmarshal(b, &body)
		if raw, ok := body["masjid_id"]; ok {
			if s, ok := raw.(string); ok {
				if v := strings.TrimSpace(s); v != "" {
					return v
				}
			}
		}
	}
	// 5) application/x-www-form-urlencoded
	if v := strings.TrimSpace(c.FormValue("masjid_id")); v != "" {
		return v
	}
	// 6) multipart/form-data
	if f, err := c.MultipartForm(); err == nil && f != nil {
		if vals, ok := f.Value["masjid_id"]; ok && len(vals) > 0 {
			if v := strings.TrimSpace(vals[0]); v != "" {
				return v
			}
		}
	}

	return ""
}

func IsMasjidAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔍 [MIDDLEWARE] IsMasjidAdmin active")
		log.Println("    Path  :", c.Path())
		log.Println("    Method:", c.Method())

		role := strings.ToLower(strings.TrimSpace(helper.GetRole(c)))
		log.Println("    role(locals.role):", role, "  (locals.userRole):", c.Locals("userRole"))

		// Scope yang diminta klien (opsional)
		reqMasjid := extractMasjidID(c)

		// =========================
		// 1) OWNER: selalu lolos
		// =========================
		if role == constants.RoleOwner {
			log.Println("[MIDDLEWARE] Bypass: user is OWNER")
			c.Locals(helper.LocRole, constants.RoleOwner)
			if reqMasjid != "" {
				c.Locals("masjid_id", reqMasjid)
				log.Println("[MIDDLEWARE] OWNER scope masjid_id:", reqMasjid)
			} else {
				log.Println("[MIDDLEWARE] OWNER tanpa scope masjid_id (gunakan X-Masjid-ID / ?masjid_id / body.masjid_id)")
			}
			return c.Next()
		}

		// =========================
		// 2) Admin/DKM ATAU Teacher
		// =========================
		adminMasjids   := getLocalsAsStrings(c, helper.LocMasjidAdminIDs)   // untuk admin/dkm
		teacherMasjids := getLocalsAsStrings(c, helper.LocMasjidTeacherIDs) // untuk teacher

		if len(adminMasjids) == 0 && len(teacherMasjids) == 0 {
			log.Println("[MIDDLEWARE] Token tidak punya masjid_admin_ids atau masjid_teacher_ids")
			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak valid atau tidak memiliki akses masjid (admin/teacher)")
		}

		// pilih masjid aktif
		chosen := ""
		if reqMasjid != "" {
			// harus match di salah satu daftar
			if contains(adminMasjids, reqMasjid) || contains(teacherMasjids, reqMasjid) {
				chosen = reqMasjid
			} else {
				return fiber.NewError(fiber.StatusForbidden, "Bukan admin/teacher pada masjid yang diminta")
			}
		} else {
			// default prioritas: admin dulu, baru teacher
			if len(adminMasjids) > 0 {
				chosen = adminMasjids[0]
			} else {
				chosen = teacherMasjids[0]
			}
		}

		c.Locals("masjid_id", chosen)
		c.Locals(helper.LocRole, role)
		log.Println("[MIDDLEWARE] Akses DIIJINKAN, role:", role, "masjid_id:", chosen)
		return c.Next()
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if strings.TrimSpace(x) == strings.TrimSpace(want) {
			return true
		}
	}
	return false
}



// helper kecil untuk baca slice string dari Locals apapun tipenya
func getLocalsAsStrings(c *fiber.Ctx, key string) []string {
	v := c.Locals(key)
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, it := range t {
			if s, ok := it.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case string:
		if s := strings.TrimSpace(t); s != "" {
			return []string{s}
		}
	}
	return nil
}
