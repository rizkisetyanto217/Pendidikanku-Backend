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

	"schoolku_backend/internals/constants"
	helper "schoolku_backend/internals/helpers/auth"
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
   Ekstraksi school_id & role dari request
========================== */

func extractSchoolID(c *fiber.Ctx) string {
	// 1) param (/:school_id)
	if v := strings.TrimSpace(c.Params("school_id")); v != "" {
		return v
	}
	// 2) query (?school_id=)
	if v := strings.TrimSpace(c.Query("school_id")); v != "" {
		return v
	}
	// 3) header (X-School-ID)
	if v := strings.TrimSpace(c.Get("X-School-ID")); v != "" {
		return v
	}
	// 4) body json (best-effort; hanya kalau content-type json)
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	if strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
		var body map[string]any
		if b := c.Body(); len(b) > 0 {
			_ = json.Unmarshal(b, &body)
			if raw, ok := body["school_id"]; ok {
				if s, ok := raw.(string); ok {
					if v := strings.TrimSpace(s); v != "" {
						return v
					}
				}
			}
		}
	}
	// 5) form-urlencoded
	if v := strings.TrimSpace(c.FormValue("school_id")); v != "" {
		return v
	}
	// 6) multipart
	if f, err := c.MultipartForm(); err == nil && f != nil {
		if vals, ok := f.Value["school_id"]; ok && len(vals) > 0 {
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
   Representasi school_roles & pengambilan dari locals
========================== */

type SchoolRole struct {
	SchoolID string   `json:"school_id"`
	Roles    []string `json:"roles"`
}

// Ambil school_roles dari locals yang sudah diisi Auth middleware.
// - Idealnya, auth middleware men-set helper.LocSchoolRoles dengan []SchoolRole.
// - Fallback: bangun dari LocSchoolAdminIDs & LocSchoolTeacherIDs jika masih dipakai.
func getSchoolRoles(c *fiber.Ctx) []SchoolRole {
	// 1) format sudah []SchoolRole
	if v := c.Locals(helper.LocSchoolRoles); v != nil {
		if xs, ok := v.([]SchoolRole); ok {
			return xs
		}
		// 2) format generic []interface{}
		if arr, ok := v.([]interface{}); ok {
			out := make([]SchoolRole, 0, len(arr))
			for _, it := range arr {
				if m, ok := it.(map[string]interface{}); ok {
					mid, _ := m["school_id"].(string)
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
						out = append(out, SchoolRole{SchoolID: mid, Roles: roles})
					}
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	// 3) fallback lama
	adminSchools := getLocalsAsStrings(c, helper.LocSchoolAdminIDs)
	teacherSchools := getLocalsAsStrings(c, helper.LocSchoolTeacherIDs)
	m := map[string]map[string]bool{}
	for _, id := range adminSchools {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if m[id] == nil {
			m[id] = map[string]bool{}
		}
		m[id][constants.RoleDKM] = true // atau "admin" jika kamu punya role admin eksplisit
	}
	for _, id := range teacherSchools {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if m[id] == nil {
			m[id] = map[string]bool{}
		}
		m[id][constants.RoleTeacher] = true
	}
	out := make([]SchoolRole, 0, len(m))
	for k, bag := range m {
		var rs []string
		for r := range bag {
			rs = append(rs, r)
		}
		out = append(out, SchoolRole{SchoolID: k, Roles: rs})
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
	SchoolID string   `json:"school_id"`
	Roles    []string `json:"roles"`
}

/* ==========================
   Middleware helpers
========================== */

// --- helper: cek role ada di school tertentu (dari locals school_roles) ---
func roleInSchool(c *fiber.Ctx, schoolID, role string) bool {
	mid := strings.TrimSpace(schoolID)
	r := trimLower(role)
	if mid == "" || r == "" {
		return false
	}
	for _, mr := range getSchoolRoles(c) {
		if strings.EqualFold(mr.SchoolID, mid) {
			for _, rr := range mr.Roles {
				if strings.EqualFold(rr, r) {
					return true
				}
			}
		}
	}
	return false
}

// extractor STRICT: hanya balikin kalau benar-benar UUID school_id
func extractSchoolIDStrict(c *fiber.Ctx) string {
	// 1) param biasa
	for _, key := range []string{"school_id", "id", "mid"} {
		if v := strings.TrimSpace(c.Params(key)); v != "" {
			if _, err := uuid.Parse(v); err == nil {
				return v
			}
		}
	}

	// 2) Fallback standar (query/header/body/form) → tapi validasi UUID
	if v := extractSchoolID(c); v != "" {
		if _, err := uuid.Parse(v); err == nil {
			return v
		}
	}

	// 3) Parse path manual untuk beberapa pola umum
	path := strings.Trim(c.Path(), "/")
	parts := strings.Split(path, "/")
	n := len(parts)
	if n == 0 {
		return ""
	}

	// 3a) /api/(a|u)/:school_id/...
	if n >= 3 && strings.EqualFold(parts[0], "api") &&
		(strings.EqualFold(parts[1], "a") || strings.EqualFold(parts[1], "u")) {
		cand := strings.TrimSpace(parts[2])
		if _, err := uuid.Parse(cand); err == nil {
			return cand
		}

		// /api/(a|u)/schools/:school_id/...
		if strings.EqualFold(parts[2], "schools") && n >= 4 {
			cand = strings.TrimSpace(parts[3])
			if _, err := uuid.Parse(cand); err == nil {
				return cand
			}
		}
		// /api/(a|u)/slug/:school_slug/... → slug, sengaja di-skip
	}

	// 3d) Pola lain: cari kata "schools" lalu ambil segmen setelahnya (kalau UUID)
	for i := 0; i < n-1; i++ {
		if strings.EqualFold(parts[i], "schools") {
			cand := strings.TrimSpace(parts[i+1])
			if _, err := uuid.Parse(cand); err == nil {
				return cand
			}
		}
	}

	return ""
}

/* ==========================
   STRICT SCOPE — by PATH + token fallback
========================== */

// UseSchoolScope (strict-ish):
// - Coba ambil school_id dari PATH/param (UUID).
// - Kalau kosong, fallback ke GetActiveSchoolIDFromToken (1 sesi = 1 sekolah).
// - Non-owner: school harus ada di token (school_roles).
// - Role: jika dikirim user, harus ada di school tsb; kalau tidak, pilih best role di school tsb.
// - Set locals: active_school_id, active_role (+ kompat: school_id, role).
func UseSchoolScope() fiber.Handler {
	return func(c *fiber.Ctx) error {

		// ==== BYPASS untuk endpoint global tanpa school_id ====
		p := strings.TrimRight(strings.ToLower(strings.TrimSpace(c.Path())), "/")

		// 1) join user-class-sections (sudah ada)
		if c.Method() == fiber.MethodPost && p == "/api/u/student-class-sections/join" {
			return c.Next()
		}

		// 2) BYPASS semua route PUBLIC
		if strings.HasPrefix(p, "/api/public/") {
			return c.Next()
		}

		// 3) BYPASS join-teacher (dua pola):
		//    - POST /api/u/:school_id/join-teacher
		//    - POST /api/u/m/:school_slug/join-teacher
		if c.Method() == fiber.MethodPost && strings.HasPrefix(p, "/api/u/") && strings.HasSuffix(p, "/join-teacher") {
			segs := strings.Split(p, "/") // e.g. ["", "api", "u", "<uuid>", "join-teacher"] atau ["", "api", "u", "m", "<slug>", "join-teacher"]
			if len(segs) == 5 && segs[0] == "" && segs[1] == "api" && segs[2] == "u" && segs[4] == "join-teacher" {
				// pola : /api/u/:school_id/join-teacher
				return c.Next()
			}
			if len(segs) == 6 && segs[0] == "" && segs[1] == "api" && segs[2] == "u" && segs[3] == "m" && segs[5] == "join-teacher" {
				// pola : /api/u/m/:school_slug/join-teacher
				return c.Next()
			}
			// jika tidak cocok, lanjut ke scope strict seperti biasa
		}

		// 4) BYPASS registration-enroll (self-registration + payment student)
		//    Contoh path:
		//    - /api/u/:school_id/finance/payments/registration-enroll
		//    - /api/u/m/:school_slug/finance/payments/registration-enroll
		if c.Method() == fiber.MethodPost &&
			strings.HasPrefix(p, "/api/u/s") &&
			strings.HasSuffix(p, "/payments/registration-enroll") {
			return c.Next()
		}

		log.Println("🎯 [MIDDLEWARE] UseSchoolScope (STRICT by path + token fallback) | Path:", c.Path(), "| Method:", c.Method())

		isOwner := helper.IsOwner(c)

		// 1) Coba ambil dari PATH/PARAM (UUID saja)
		reqSchool := strings.TrimSpace(extractSchoolIDStrict(c))

		// 2) Fallback: kalau kosong, ambil dari token (active school)
		if reqSchool == "" {
			if id, err := helper.GetActiveSchoolIDFromToken(c); err == nil && id != uuid.Nil {
				reqSchool = id.String()
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "school_id wajib di path, parameter, atau token")
			}
		}

		// Validasi UUID
		if _, err := uuid.Parse(reqSchool); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "school_id tidak valid")
		}

		reqRole := trimLower(extractRole(c))

		// OWNER bypass
		if isOwner {
			if reqRole == "" {
				reqRole = constants.RoleOwner
			}
			c.Locals("active_school_id", reqSchool)
			c.Locals("active_role", reqRole)
			c.Locals("school_id", reqSchool)
			c.Locals(helper.LocRole, reqRole)
			log.Println("    🔧 owner scope | school_id:", reqSchool, "| role:", reqRole)
			return c.Next()
		}

		// Ambil roles yang dimiliki user di school ini
		var rolesAtSchool []string
		for _, mr := range getSchoolRoles(c) {
			if strings.EqualFold(mr.SchoolID, reqSchool) {
				rolesAtSchool = mr.Roles
				break
			}
		}

		// 🆕 NEW: kalau belum ada role, tapi school_id ada di token → anggap dia "user"
		if len(rolesAtSchool) == 0 {
			if ids, err := helper.GetSchoolIDsFromToken(c); err == nil {
				for _, id := range ids {
					if strings.EqualFold(id.String(), reqSchool) {
						rolesAtSchool = []string{constants.RoleUser}
						break
					}
				}
			}
		}

		if len(rolesAtSchool) == 0 {
			return fiber.NewError(fiber.StatusForbidden, "Bukan anggota pada school yang diminta")
		}

		activeRole := reqRole
		if activeRole != "" {
			if !roleInSchool(c, reqSchool, activeRole) {
				return fiber.NewError(fiber.StatusForbidden, "Role tidak tersedia pada school tersebut")
			}
		} else {
			activeRole = bestRoleFor(rolesAtSchool)
			if activeRole == "" {
				return fiber.NewError(fiber.StatusForbidden, "Tidak memiliki peran pada school tersebut")
			}
		}

		// Set locals scope
		c.Locals("active_school_id", reqSchool)
		c.Locals("active_role", activeRole)
		c.Locals("school_id", reqSchool)
		c.Locals(helper.LocRole, activeRole)

		log.Println("    🔧 scope set | school_id:", reqSchool, "| role:", activeRole)
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
		if !strings.HasPrefix(strings.ToLower(c.Path()), "/api/a/") {
			return c.Next()
		}

		pathID := strings.TrimSpace(extractSchoolIDStrict(c))

		// kalau path memang tidak mengandung UUID sebagai school_id → skip check
		if pathID == "" {
			return c.Next()
		}

		active := strings.TrimSpace(asString(c.Locals("active_school_id")))
		if active == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Scope school belum ditentukan")
		}
		if !strings.EqualFold(pathID, active) {
			return fiber.NewError(fiber.StatusForbidden, "Scope school tidak cocok dengan path")
		}
		return c.Next()
	}
}

/* ==========================
   STRICT ROLE CHECK
========================== */

// IsSchoolAdmin (strict):
// - Hanya izinkan owner/admin/dkm (teacher TIDAK otomatis lolos).
// - Pastikan role itu benar-benar ada di school PATH (sudah di-set di UseSchoolScope strict).
func IsSchoolAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔐 [MIDDLEWARE] IsSchoolAdmin (STRICT) | Path:", c.Path(), "| Method:", c.Method())

		mid := strings.TrimSpace(asString(c.Locals("active_school_id")))
		role := trimLower(asString(c.Locals("active_role")))

		if mid == "" || role == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Scope school/role belum ditentukan")
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

		// hard verify role benar-benar ada pada school mid
		if !roleInSchool(c, mid, role) {
			return fiber.NewError(fiber.StatusForbidden, "Role tidak terdaftar pada school ini")
		}

		// Kompat locals lama
		c.Locals("school_id", mid)
		c.Locals(helper.LocRole, role)

		log.Println("    ✅ akses diijinkan | role:", role, "| school_id:", mid)
		return c.Next()
	}
}

// IsSchoolStaff:
// - Izinkan admin/dkm/teacher (plus owner).
func IsSchoolStaff() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log.Println("🔐 [MIDDLEWARE] IsSchoolStaff (STRICT) | Path:", c.Path(), "| Method:", c.Method())

		mid := strings.TrimSpace(asString(c.Locals("active_school_id")))
		role := trimLower(asString(c.Locals("active_role")))
		if mid == "" || role == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Scope school/role belum ditentukan")
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
		if !roleInSchool(c, mid, role) {
			return fiber.NewError(fiber.StatusForbidden, "Role tidak terdaftar pada school ini")
		}
		c.Locals("school_id", mid)
		c.Locals(helper.LocRole, role)
		log.Println("    ✅ akses diijinkan (staff) | role:", role, "| school_id:", mid)
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
