package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"masjidku_backend/internals/constants"
	helper "masjidku_backend/internals/helpers/auth"
)

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

/* ==========================
   Role priority (untuk auto-pick role terbaik)
========================== */

var rolePriority = map[string]int{
	constants.RoleOwner:     100,
	"admin":                 90, // jika ada
	constants.RoleDKM:       80,
	constants.RoleTeacher:   70,
	constants.RoleTreasurer: 60,
	constants.RoleAuthor:    50,
	constants.RoleStudent:   40,
	constants.RoleUser:      10,
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

/* ==========================
   Middleware 1 — UseMasjidScope
   (menetapkan active_masjid_id & active_role)
========================== */

/* util kecil tetap sama: getLocalsAsStrings, asString, trimLower, extractMasjidID, extractRole,
   getMasjidRoles, getMasjidIDsPref, rolePriority, bestRoleFor, respondNeedScope
   (biarkan seperti punyamu—dipakai sebagian) */

// --- helper: cek role ada di masjid tertentu (dari locals masjid_roles) ---
func roleInMasjid(c *fiber.Ctx, masjidID, role string) bool {
	mid := strings.TrimSpace(masjidID)
	r := trimLower(role)
	if mid == "" || r == "" {
		return false
	}
	for _, mr := range getMasjidRoles(c) {
		if strings.EqualFold(mr.MasjidID, mid) {
			for _, rr := range mr.Roles {
				if strings.EqualFold(rr, r) {
					return true
				}
			}
		}
	}
	return false
}

// Tambah util ini
func extractMasjidIDStrict(c *fiber.Ctx) string {
	// 1) Kalau middleware dipasang di group yang memang punya param
	for _, key := range []string{"masjid_id", "id", "mid"} {
		if v := strings.TrimSpace(c.Params(key)); v != "" {
			return v
		}
	}

	// 2) Fallback standar (query/header/body/form)
	if v := extractMasjidID(c); v != "" {
		return v
	}

	// 3) Parse path manual untuk beberapa pola umum
	path := strings.Trim(c.Path(), "/")
	parts := strings.Split(path, "/")
	n := len(parts)
	if n == 0 {
		return ""
	}

	// 3a) /api/(a|u)/:masjid_id/...
	if n >= 3 && strings.EqualFold(parts[0], "api") &&
		(strings.EqualFold(parts[1], "a") || strings.EqualFold(parts[1], "u")) {
		// kalau segmen ke-2 bukan "slug" atau "masjids", treat sebagai masjid_id langsung
		if !strings.EqualFold(parts[2], "slug") && !strings.EqualFold(parts[2], "masjids") {
			return parts[2]
		}
		// 3b) /api/(a|u)/masjids/:masjid_id/...
		if strings.EqualFold(parts[2], "masjids") && n >= 4 {
			return parts[3]
		}
		// 3c) /api/(a|u)/slug/:masjid_slug/...  (kalau kamu pakai slug di beberapa tempat)
		if strings.EqualFold(parts[2], "slug") && n >= 4 {
			return parts[3]
		}
	}

	// 3d) Pola lain: cari kata "masjids" lalu ambil segmen setelahnya
	for i := 0; i < n-1; i++ {
		if strings.EqualFold(parts[i], "masjids") {
			return parts[i+1]
		}
	}

	return ""
}

/* ==========================
   STRICT SCOPE — by PATH ONLY
========================== */

// UseMasjidScope (strict):
// - Ambil masjid_id dari PATH (atau eksplisit query/header/body).
// - Non-owner: wajib merupakan masjid yang ada di token.
// - Role: jika dikirim user, harus ada di masjid tersebut; jika tidak, pilih best role DI masjid itu.
// - Set locals: active_masjid_id, active_role (+ kompat: masjid_id, role)
func UseMasjidScope() fiber.Handler {
	return func(c *fiber.Ctx) error {

		// ==== BYPASS untuk endpoint global tanpa masjid_id ====
		p := strings.TrimRight(strings.ToLower(strings.TrimSpace(c.Path())), "/")

		// 1) join user-class-sections (sudah ada)
		if c.Method() == fiber.MethodPost && p == "/api/u/student-class-sections/join" {
			return c.Next()
		}

		// 2) 🔑 BYPASS join-teacher (dua pola):
		//    - POST /api/u/:masjid_id/join-teacher
		//    - POST /api/u/m/:masjid_slug/join-teacher
		if c.Method() == fiber.MethodPost && strings.HasPrefix(p, "/api/u/") && strings.HasSuffix(p, "/join-teacher") {
			// validasi pola dasar untuk kehati-hatian
			segs := strings.Split(p, "/") // e.g. ["", "api", "u", "<uuid>", "join-teacher"] atau ["", "api", "u", "m", "<slug>", "join-teacher"]
			if len(segs) == 5 && segs[0] == "" && segs[1] == "api" && segs[2] == "u" && segs[4] == "join-teacher" {
				// pola : /api/u/:masjid_id/join-teacher
				return c.Next()
			}
			if len(segs) == 6 && segs[0] == "" && segs[1] == "api" && segs[2] == "u" && segs[3] == "m" && segs[5] == "join-teacher" {
				// pola : /api/u/m/:masjid_slug/join-teacher
				return c.Next()
			}
			// jika tidak cocok, lanjut ke scope strict seperti biasa
		}

		log.Println("🎯 [MIDDLEWARE] UseMasjidScope (STRICT by path)")

		isOwner := helper.IsOwner(c)

		// ⬇️ pakai extractor baru
		reqMasjid := strings.TrimSpace(extractMasjidIDStrict(c))

		if reqMasjid == "" {
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib di path atau parameter")
		}
		if _, err := uuid.Parse(reqMasjid); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id pada path tidak valid, error")
		}

		reqRole := trimLower(extractRole(c))

		if isOwner {
			if reqRole == "" {
				reqRole = constants.RoleOwner
			}
			c.Locals("active_masjid_id", reqMasjid)
			c.Locals("active_role", reqRole)
			c.Locals("masjid_id", reqMasjid)
			c.Locals(helper.LocRole, reqRole)
			log.Println("    🔧 owner scope | masjid_id:", reqMasjid, "| role:", reqRole)
			return c.Next()
		}

		var rolesAtMasjid []string
		for _, mr := range getMasjidRoles(c) {
			if strings.EqualFold(mr.MasjidID, reqMasjid) {
				rolesAtMasjid = mr.Roles
				break
			}
		}
		if len(rolesAtMasjid) == 0 {
			return fiber.NewError(fiber.StatusForbidden, "Bukan anggota pada masjid yang diminta")
		}

		activeRole := reqRole
		if activeRole != "" {
			if !roleInMasjid(c, reqMasjid, activeRole) {
				return fiber.NewError(fiber.StatusForbidden, "Role tidak tersedia pada masjid tersebut")
			}
		} else {
			activeRole = bestRoleFor(rolesAtMasjid)
			if activeRole == "" {
				return fiber.NewError(fiber.StatusForbidden, "Tidak memiliki peran pada masjid tersebut")
			}
		}

		c.Locals("active_masjid_id", reqMasjid)
		c.Locals("active_role", activeRole)
		c.Locals("masjid_id", reqMasjid)
		c.Locals(helper.LocRole, activeRole)

		log.Println("    🔧 scope set | masjid_id:", reqMasjid, "| role:", activeRole)
		return c.Next()
	}
}

/*
	==========================
	  Guard: path ↔ scope harus cocok (defense in depth)

==========================
*/
func RequirePathScopeMatch() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !strings.HasPrefix(c.Path(), "/api/a/") {
			return c.Next()
		}
		pathID := strings.TrimSpace(extractMasjidIDStrict(c))
		if pathID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "masjid_id path tidak valid")
		}
		active := strings.TrimSpace(asString(c.Locals("active_masjid_id")))
		if active == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Scope masjid belum ditentukan")
		}
		if !strings.EqualFold(pathID, active) {
			return fiber.NewError(fiber.StatusForbidden, "Scope masjid tidak cocok dengan path")
		}
		return c.Next()
	}
}

/* ==========================
   STRICT ROLE CHECK
========================== */

// IsMasjidAdmin (strict):
// - Hanya izinkan owner/admin/dkm (teacher TIDAK otomatis lolos).
// - Pastikan role itu benar-benar ada di masjid PATH (sudah di-set di UseMasjidScope strict).
func IsMasjidAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔐 [MIDDLEWARE] IsMasjidAdmin (STRICT) | Path:", c.Path(), "| Method:", c.Method())

		mid := strings.TrimSpace(asString(c.Locals("active_masjid_id")))
		role := trimLower(asString(c.Locals("active_role")))

		if mid == "" || role == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Scope masjid/role belum ditentukan")
		}

		// owner bypass
		if helper.IsOwner(c) {
			return c.Next()
		}

		// only admin/dkm
		switch role {
		case "admin", constants.RoleDKM:
			// ok
		default:
			return fiber.NewError(fiber.StatusForbidden, "Role tidak berhak mengakses endpoint ini")
		}

		// hard verify role benar-benar ada pada masjid mid
		if !roleInMasjid(c, mid, role) {
			return fiber.NewError(fiber.StatusForbidden, "Role tidak terdaftar pada masjid ini")
		}

		// Kompat locals lama
		c.Locals("masjid_id", mid)
		c.Locals(helper.LocRole, role)

		log.Println("    ✅ akses diijinkan | role:", role, "| masjid_id:", mid)
		return c.Next()
	}
}

// Opsional: jika kamu butuh endpoint yang mengizinkan teacher juga
func IsMasjidStaff() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔐 [MIDDLEWARE] IsMasjidStaff (STRICT) | Path:", c.Path(), "| Method:", c.Method())

		mid := strings.TrimSpace(asString(c.Locals("active_masjid_id")))
		role := trimLower(asString(c.Locals("active_role")))
		if mid == "" || role == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Scope masjid/role belum ditentukan")
		}
		if helper.IsOwner(c) {
			return c.Next()
		}
		switch role {
		case "admin", constants.RoleDKM, constants.RoleTeacher:
			// ok
		default:
			return fiber.NewError(fiber.StatusForbidden, "Role tidak berhak mengakses endpoint ini")
		}
		if !roleInMasjid(c, mid, role) {
			return fiber.NewError(fiber.StatusForbidden, "Role tidak terdaftar pada masjid ini")
		}
		c.Locals("masjid_id", mid)
		c.Locals(helper.LocRole, role)
		log.Println("    ✅ akses diijinkan (staff) | role:", role, "| masjid_id:", mid)
		return c.Next()
	}
}

/* ==========================
   Owner-only tetap sama
========================== */

func IsOwnerGlobal() fiber.Handler {
	return func(c *fiber.Ctx) error {
		rc, ok := c.Locals("roles_claim").(helper.RolesClaim)
		if !ok {
			return fiber.NewError(http.StatusUnauthorized, "Roles claim tidak ditemukan")
		}
		for _, r := range rc.RolesGlobal {
			if strings.EqualFold(r, "owner") {
				return c.Next()
			}
		}
		return fiber.NewError(http.StatusForbidden, "Akses khusus owner")
	}
}
