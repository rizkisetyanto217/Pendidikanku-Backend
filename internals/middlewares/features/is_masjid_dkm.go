package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"masjidku_backend/internals/constants"
	helper "masjidku_backend/internals/helpers/auth"
)

/* ==========================
   Small helpers
========================== */

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if strings.TrimSpace(x) == strings.TrimSpace(want) {
			return true
		}
	}
	return false
}

// baca slice string dari Locals apapun tipenya
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

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return ""
	}
}

func trimLower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

/* ==========================
   Ekstraksi masjid_id & role dari request
========================== */

func extractMasjidID(c *fiber.Ctx) string {
	// 1) param (/:masjid_id)
	if v := strings.TrimSpace(c.Params("masjid_id")); v != "" {
		return v
	}
	// 2) query (?masjid_id=)
	if v := strings.TrimSpace(c.Query("masjid_id")); v != "" {
		return v
	}
	// 3) header (X-Masjid-ID)
	if v := strings.TrimSpace(c.Get("X-Masjid-ID")); v != "" {
		return v
	}
	// 4) body json (best-effort; hanya kalau content-type json)
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	if strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
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
	}
	// 5) form-urlencoded
	if v := strings.TrimSpace(c.FormValue("masjid_id")); v != "" {
		return v
	}
	// 6) multipart
	if f, err := c.MultipartForm(); err == nil && f != nil {
		if vals, ok := f.Value["masjid_id"]; ok && len(vals) > 0 {
			if v := strings.TrimSpace(vals[0]); v != "" {
				return v
			}
		}
	}
	return ""
}

func extractRole(c *fiber.Ctx) string {
	// query
	if v := trimLower(c.Query("role")); v != "" {
		return v
	}
	if v := trimLower(c.Query("active_role")); v != "" {
		return v
	}
	// header
	if v := trimLower(c.Get("X-Role")); v != "" {
		return v
	}
	if v := trimLower(c.Get("X-Active-Role")); v != "" {
		return v
	}
	// body json
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	if strings.HasPrefix(ct, fiber.MIMEApplicationJSON) && len(c.Body()) > 0 {
		var body map[string]any
		_ = json.Unmarshal(c.Body(), &body)
		for _, k := range []string{"role", "active_role"} {
			if raw, ok := body[k]; ok {
				if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
					return strings.ToLower(strings.TrimSpace(s))
				}
			}
		}
	}
	// form
	if v := trimLower(c.FormValue("role")); v != "" {
		return v
	}
	if v := trimLower(c.FormValue("active_role")); v != "" {
		return v
	}
	return ""
}

/* ==========================
   Representasi masjid_roles & pengambilan dari locals
========================== */

type MasjidRole struct {
	MasjidID string   `json:"masjid_id"`
	Roles    []string `json:"roles"`
}

// Ambil masjid_roles dari locals yang sudah diisi Auth middleware.
// - Idealnya, auth middleware men-set helper.LocMasjidRoles dengan []MasjidRole.
// - Fallback: bangun dari LocMasjidAdminIDs & LocMasjidTeacherIDs jika masih dipakai.
func getMasjidRoles(c *fiber.Ctx) []MasjidRole {
	// 1) format sudah []MasjidRole
	if v := c.Locals(helper.LocMasjidRoles); v != nil {
		if xs, ok := v.([]MasjidRole); ok {
			return xs
		}
		// 2) format generic []interface{}
		if arr, ok := v.([]interface{}); ok {
			out := make([]MasjidRole, 0, len(arr))
			for _, it := range arr {
				if m, ok := it.(map[string]interface{}); ok {
					mid, _ := m["masjid_id"].(string)
					var roles []string
					switch rr := m["roles"].(type) {
					case []interface{}:
						for _, r := range rr {
							if s, ok := r.(string); ok && strings.TrimSpace(s) != "" {
								roles = append(roles, trimLower(s))
							}
						}
					case []string:
						for _, s := range rr {
							if strings.TrimSpace(s) != "" {
								roles = append(roles, trimLower(s))
							}
						}
					}
					if strings.TrimSpace(mid) != "" {
						out = append(out, MasjidRole{MasjidID: mid, Roles: roles})
					}
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	// 3) fallback lama
	adminMasjids := getLocalsAsStrings(c, helper.LocMasjidAdminIDs)
	teacherMasjids := getLocalsAsStrings(c, helper.LocMasjidTeacherIDs)
	m := map[string]map[string]bool{}
	for _, id := range adminMasjids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if m[id] == nil {
			m[id] = map[string]bool{}
		}
		m[id][constants.RoleDKM] = true // atau "admin" jika kamu punya role admin eksplisit
	}
	for _, id := range teacherMasjids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if m[id] == nil {
			m[id] = map[string]bool{}
		}
		m[id][constants.RoleTeacher] = true
	}
	out := make([]MasjidRole, 0, len(m))
	for k, bag := range m {
		var rs []string
		for r := range bag {
			rs = append(rs, r)
		}
		out = append(out, MasjidRole{MasjidID: k, Roles: rs})
	}
	return out
}

func getMasjidIDsPref(c *fiber.Ctx) []string {
	if xs := getLocalsAsStrings(c, helper.LocMasjidIDs); len(xs) > 0 {
		return xs
	}
	return nil
}

/* ==========================
   Role priority (untuk auto-pick role terbaik)
========================== */

var rolePriority = map[string]int{
	constants.RoleOwner:     100,
	"admin":                  90,   // jika ada
	constants.RoleDKM:        80,
	constants.RoleTeacher:    70,
	constants.RoleTreasurer:  60,
	constants.RoleAuthor:     50,
	constants.RoleStudent:    40,
	constants.RoleUser:       10,
}

func bestRoleFor(roles []string) string {
	if len(roles) == 0 {
		return ""
	}
	cands := make([]string, 0, len(roles))
	for _, r := range roles {
		r = trimLower(r)
		if r != "" {
			cands = append(cands, r)
		}
	}
	if len(cands) == 0 {
		return ""
	}
	sort.Slice(cands, func(i, j int) bool { return rolePriority[cands[i]] > rolePriority[cands[j]] })
	return cands[0]
}

/* ==========================
   Response helper (untuk memaksa frontend pilih scope)
========================== */

type ScopeChoice struct {
	MasjidID string   `json:"masjid_id"`
	Roles    []string `json:"roles"`
}

func respondNeedScope(c *fiber.Ctx, choices []ScopeChoice) error {
	// 428: Precondition Required — minta client kirim X-Masjid-ID & X-Role (atau query/body)
	payload := fiber.Map{
		"code":    428,
		"status":  "need_scope",
		"message": "Beberapa masjid/role tersedia. Tentukan masjid_id & role yang akan dipakai.",
		"data": fiber.Map{
			"choices":       choices,                    // untuk dropdown frontend
			"how_to_select": "Kirim ?masjid_id=...&role=... atau header X-Masjid-ID & X-Role, atau di body JSON.",
		},
	}
	return c.Status(428).JSON(payload)
}

/* ==========================
   Middleware 1 — UseMasjidScope
   (menetapkan active_masjid_id & active_role)
========================== */

func UseMasjidScope() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🎯 [MIDDLEWARE] UseMasjidScope")

		isOwner := helper.IsOwner(c)
		userMasjidRoles := getMasjidRoles(c)

		// Owner boleh tanpa scope (kecuali endpoint tertentu), tapi tetap izinkan set jika ada.
		if !isOwner && len(userMasjidRoles) == 0 {
			return fiber.NewError(fiber.StatusUnauthorized, "Token tidak memiliki akses masjid")
		}

		// 0) Jika hanya ada satu masjid & satu role → auto-select
		if !isOwner && len(userMasjidRoles) == 1 && len(userMasjidRoles[0].Roles) == 1 {
			mid := strings.TrimSpace(userMasjidRoles[0].MasjidID)
			role := trimLower(userMasjidRoles[0].Roles[0])
			c.Locals("active_masjid_id", mid)
			c.Locals("active_role", role)
			log.Println("    🔧 auto-select scope | masjid_id:", mid, "| role:", role)
			return c.Next()
		}

		// 1) Ambil scope eksplisit dari request
		reqMasjid := strings.TrimSpace(extractMasjidID(c))
		reqRole := trimLower(extractRole(c))

		if reqMasjid != "" {
			if isOwner {
				// Owner bebas menentukan role; default owner jika kosong
				if reqRole == "" {
					reqRole = constants.RoleOwner
				}
				c.Locals("active_masjid_id", reqMasjid)
				c.Locals("active_role", reqRole)
				return c.Next()
			}
			// Non-owner → validasi masjid & role ada di daftar
			for _, mr := range userMasjidRoles {
				if mr.MasjidID == reqMasjid {
					if reqRole != "" {
						ok := false
						for _, r := range mr.Roles {
							if strings.EqualFold(r, reqRole) {
								ok = true
								break
							}
						}
						if !ok {
							return fiber.NewError(fiber.StatusForbidden, "Role tidak tersedia pada masjid tersebut")
						}
						c.Locals("active_masjid_id", reqMasjid)
						c.Locals("active_role", reqRole)
						return c.Next()
					}
					// role tidak diminta → pilih yang terbaik di masjid itu
					c.Locals("active_masjid_id", reqMasjid)
					c.Locals("active_role", bestRoleFor(mr.Roles))
					return c.Next()
				}
			}
			return fiber.NewError(fiber.StatusForbidden, "Bukan anggota pada masjid yang diminta")
		}

		// 2) Jika sudah ada active_masjid_id sebelumnya & valid → pakai itu
		if v := strings.TrimSpace(asString(c.Locals("active_masjid_id"))); v != "" {
			if isOwner {
				if r := trimLower(asString(c.Locals("active_role"))); r == "" {
					c.Locals("active_role", constants.RoleOwner)
				}
				return c.Next()
			}
			for _, mr := range userMasjidRoles {
				if mr.MasjidID == v {
					r := trimLower(asString(c.Locals("active_role")))
					if r == "" {
						r = bestRoleFor(mr.Roles)
					}
					c.Locals("active_masjid_id", v)
					c.Locals("active_role", r)
					return c.Next()
				}
			}
		}

		// 3) Pakai preferensi masjid_ids[0] dari token (legacy) jika ada & valid
		if !isOwner {
			if prefs := getMasjidIDsPref(c); len(prefs) > 0 {
				for _, mid := range prefs {
					mid = strings.TrimSpace(mid)
					if mid == "" {
						continue
					}
					for _, mr := range userMasjidRoles {
						if mr.MasjidID == mid {
							c.Locals("active_masjid_id", mid)
							c.Locals("active_role", bestRoleFor(mr.Roles))
							return c.Next()
						}
					}
				}
			}
		}

		// 4) Jika masih tak jelas:
		//    - Owner → boleh lanjut tanpa scope (tapi banyak endpoint akan butuh masjid_id)
		//    - Non-owner & ada banyak opsi → minta frontend memilih (balikkan daftar choices)
		if isOwner {
			c.Locals("active_role", constants.RoleOwner)
			return c.Next()
		}

		if len(userMasjidRoles) == 1 {
			// satu masjid tapi multi role → minta user tentukan role (kalau perlu)
			mr := userMasjidRoles[0]
			if len(mr.Roles) == 1 {
				// (sebenarnya sudah di-handle auto-select di atas, tapi jaga-jaga)
				c.Locals("active_masjid_id", mr.MasjidID)
				c.Locals("active_role", trimLower(mr.Roles[0]))
				return c.Next()
			}
			return respondNeedScope(c, []ScopeChoice{{MasjidID: mr.MasjidID, Roles: mr.Roles}})
		}

		choices := make([]ScopeChoice, len(userMasjidRoles))
		for i, mr := range userMasjidRoles {
			choices[i] = ScopeChoice(mr)
		}

		return respondNeedScope(c, choices)
	}
}

/* ==========================
   Middleware 2 — IsMasjidAdmin
   (izin akses berdasar scope aktif)
========================== */

func IsMasjidAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔐 [MIDDLEWARE] IsMasjidAdmin | Path:", c.Path(), "| Method:", c.Method())

		mid := strings.TrimSpace(asString(c.Locals("active_masjid_id")))
		role := trimLower(asString(c.Locals("active_role")))

		// Pastikan UseMasjidScope sudah menentukan scope
		if mid == "" || role == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Scope masjid/role belum ditentukan")
		}

		switch role {
		case constants.RoleOwner, "admin", constants.RoleDKM, constants.RoleTeacher:
			// allowed
		default:
			return fiber.NewError(fiber.StatusForbidden, "Role tidak berhak mengakses endpoint ini")
		}

		// Kompat: set locals lama yang mungkin masih dipakai downstream
		c.Locals("masjid_id", mid)
		c.Locals(helper.LocRole, role)

		log.Println("    ✅ akses diijinkan | role:", role, "| masjid_id:", mid)
		return c.Next()
	}
}
