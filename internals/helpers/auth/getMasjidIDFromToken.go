// package: internals/helpers/helper.go
package helper

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ============================================
   Locals Keys (middleware should set these)
   ============================================ */

const (
	LocRole   = "role"    // optional, legacy single role
	LocUserID = "user_id" // string | uuid

	// Generic lists (legacy / optional)
	LocMasjidIDs        = "masjid_ids"         // []string | []uuid.UUID
	LocMasjidAdminIDs   = "masjid_admin_ids"   // []string | []uuid.UUID
	LocMasjidDKMIDs     = "masjid_dkm_ids"     // []string | []uuid.UUID
	LocMasjidTeacherIDs = "masjid_teacher_ids" // []string | []uuid.UUID
	LocMasjidStudentIDs = "masjid_student_ids" // []string | []uuid.UUID

	// New structured claims (from your token)
	LocRolesGlobal    = "roles_global"     // []string
	LocMasjidRoles    = "masjid_roles"     // []MasjidRolesEntry | []map[string]any
	LocIsOwner        = "is_owner"         // bool | "true"/"false"
	LocActiveMasjidID = "active_masjid_id" // string UUID
	LocTeacherRecords = "teacher_records"  // []TeacherRecordEntry | []map[string]any
	LocStudentRecords = "student_records"  // []StudentRecordEntry | []map[string]any (BARU)
)

/* ============================================
   Types for structured claims
   ============================================ */

type MasjidRolesEntry struct {
	MasjidID uuid.UUID `json:"masjid_id"`
	Roles    []string  `json:"roles"`
}

type RolesClaim struct {
	RolesGlobal []string           `json:"roles_global"`
	MasjidRoles []MasjidRolesEntry `json:"masjid_roles"`
}

type TeacherRecordEntry struct {
	MasjidTeacherID uuid.UUID `json:"masjid_teacher_id"`
	MasjidID        uuid.UUID `json:"masjid_id"`
}

type StudentRecordEntry struct {
	MasjidStudentID uuid.UUID `json:"masjid_student_id"`
	MasjidID        uuid.UUID `json:"masjid_id"`
}

/* ============================================
   Tiny shared helpers
   ============================================ */

func quickHasTable(db *gorm.DB, table string) bool {
	if db == nil || table == "" {
		return false
	}
	var ok bool
	_ = db.Raw(`SELECT to_regclass((SELECT current_schema()) || '.' || ?) IS NOT NULL`, table).Scan(&ok).Error
	return ok
}

func quickHasFunction(db *gorm.DB, name string) bool {
	if db == nil || name == "" {
		return false
	}
	var ok bool
	_ = db.Raw(`SELECT EXISTS(SELECT 1 FROM pg_proc WHERE proname = ?)`, name).Scan(&ok).Error
	return ok
}

func normalizeLocalsToStrings(v any) []string {
	out := make([]string, 0)
	switch t := v.(type) {
	case []string:
		for _, s := range t {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	case []interface{}:
		for _, it := range t {
			switch vv := it.(type) {
			case string:
				if s := strings.TrimSpace(vv); s != "" {
					out = append(out, s)
				}
			case uuid.UUID:
				if vv != uuid.Nil {
					out = append(out, vv.String())
				}
			}
		}
	case string:
		if s := strings.TrimSpace(t); s != "" {
			out = append(out, s)
		}
	case uuid.UUID:
		if t != uuid.Nil {
			out = append(out, t.String())
		}
	case []uuid.UUID:
		for _, id := range t {
			if id != uuid.Nil {
				out = append(out, id.String())
			}
		}
	}
	return out
}

func parseFirstUUIDFromLocals(c *fiber.Ctx, key string) (uuid.UUID, error) {
	v := c.Locals(key)
	if v == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" tidak ditemukan di token")
	}
	items := normalizeLocalsToStrings(v)
	if len(items) == 0 {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token")
	}
	id, err := uuid.Parse(items[0])
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Format "+key+" tidak valid di token")
	}
	return id, nil
}

func parseUUIDSliceFromLocals(c *fiber.Ctx, key string) ([]uuid.UUID, error) {
	v := c.Locals(key)
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, key+" tidak ditemukan di token")
	}
	raw := normalizeLocalsToStrings(v)
	if len(raw) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token")
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, key+" berisi UUID tidak valid")
		}
		out = append(out, id)
	}
	return out, nil
}

/* ============================================
   JWT fallback utilities (no signature verify)
   ============================================ */

func readJWTClaims(c *fiber.Ctx) map[string]any {
	auth := strings.TrimSpace(c.Get("Authorization"))
	if auth == "" {
		return nil
	}
	parts := strings.SplitN(auth, " ", 2)
	token := parts[len(parts)-1]
	seg := strings.Split(token, ".")
	if len(seg) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(seg[1])
	if err != nil {
		return nil
	}
	var m map[string]any
	_ = json.Unmarshal(payload, &m)
	return m
}

func claimString(claims map[string]any, key string) string {
	if claims == nil {
		return ""
	}
	if v, ok := claims[key]; ok {
		if s, ok2 := v.(string); ok2 {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func claimAny(claims map[string]any, key string) any {
	if claims == nil {
		return nil
	}
	if v, ok := claims[key]; ok {
		return v
	}
	return nil
}

/* ============================================
   roles_global & masjid_roles
   ============================================ */

func GetRolesGlobal(c *fiber.Ctx) []string {
	v := c.Locals(LocRolesGlobal)
	if v == nil {
		if arr := claimAny(readJWTClaims(c), "roles_global"); arr != nil {
			v = arr
			c.Locals(LocRolesGlobal, arr)
		}
	}
	out := make([]string, 0)
	switch t := v.(type) {
	case []string:
		for _, r := range t {
			r = strings.ToLower(strings.TrimSpace(r))
			if r != "" {
				out = append(out, r)
			}
		}
	case []interface{}:
		for _, it := range t {
			if s, ok := it.(string); ok {
				s = strings.ToLower(strings.TrimSpace(s))
				if s != "" { out = append(out, s) }
			}
		}
	}
	return out
}

func HasGlobalRole(c *fiber.Ctx, role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" { return false }
	for _, r := range GetRolesGlobal(c) {
		if r == role { return true }
	}
	return false
}

func GetRole(c *fiber.Ctx) string {
	if v := c.Locals(LocRole); v != nil {
		if s, ok := v.(string); ok { return strings.ToLower(strings.TrimSpace(s)) }
	}
	return ""
}

// --- masjid_roles parsing (deduped) ---

func parseMasjidRoles(c *fiber.Ctx) ([]MasjidRolesEntry, error) {
	v := c.Locals(LocMasjidRoles)
	if v == nil {
		if any := claimAny(readJWTClaims(c), "masjid_roles"); any != nil {
			v = any
			c.Locals(LocMasjidRoles, any)
		}
	}
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" tidak ditemukan di token")
	}

	drain := func(m map[string]any) (MasjidRolesEntry, bool) {
		var e MasjidRolesEntry
		if s, ok := m["masjid_id"].(string); ok {
			if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil { e.MasjidID = id }
		}
		if rr, ok := m["roles"].([]interface{}); ok {
			for _, it := range rr {
				if rs, ok := it.(string); ok {
					rs = strings.ToLower(strings.TrimSpace(rs))
					if rs != "" { e.Roles = append(e.Roles, rs) }
				}
			}
		}
		return e, e.MasjidID != uuid.Nil && len(e.Roles) > 0
	}

	switch t := v.(type) {
	case []MasjidRolesEntry:
		out := make([]MasjidRolesEntry, 0, len(t))
		for _, mr := range t { if mr.MasjidID != uuid.Nil && len(mr.Roles) > 0 { out = append(out, mr) } }
		if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" kosong") }
		return out, nil
	case []map[string]any:
		out := make([]MasjidRolesEntry, 0, len(t))
		for _, m := range t { if e, ok := drain(m); ok { out = append(out, e) } }
		if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" kosong/invalid") }
		return out, nil
	case []interface{}:
		out := make([]MasjidRolesEntry, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok { if e, ok := drain(m); ok { out = append(out, e) } }
		}
		if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" kosong/invalid") }
		return out, nil
	}
	return nil, fiber.NewError(fiber.StatusBadRequest, LocMasjidRoles+" format tidak didukung")
}

func getMasjidIDsFromMasjidRoles(c *fiber.Ctx) ([]uuid.UUID, error) {
	entries, err := parseMasjidRoles(c)
	if err != nil { return nil, err }
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		if _, ok := seen[e.MasjidID]; !ok { seen[e.MasjidID] = struct{}{}; out = append(out, e.MasjidID) }
	}
	if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_id tidak ada pada masjid_roles") }
	return out, nil
}

func HasRoleInMasjid(c *fiber.Ctx, masjidID uuid.UUID, role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" || masjidID == uuid.Nil { return false }
	entries, err := parseMasjidRoles(c)
	if err != nil { return false }
	for _, e := range entries {
		if e.MasjidID == masjidID {
			for _, r := range e.Roles { if r == role { return true } }
		}
	}
	return false
}

/* ============================================
   active_masjid_id & role flags
   ============================================ */

func GetActiveMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals(LocActiveMasjidID)
	if v == nil {
		if s := claimString(readJWTClaims(c), "active_masjid_id"); s != "" {
			c.Locals(LocActiveMasjidID, s)
			v = s
		}
	}
	if v == nil { return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, LocActiveMasjidID+" tidak ditemukan di token") }
	s := ""
	switch t := v.(type) {
	case string:
		s = t
	case uuid.UUID:
		if t != uuid.Nil { return t, nil }
	}
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil || id == uuid.Nil { return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, LocActiveMasjidID+" tidak valid") }
	return id, nil
}

func GetActiveMasjidID(c *fiber.Ctx) (uuid.UUID, error) { return GetActiveMasjidIDFromToken(c) }

func IsOwner(c *fiber.Ctx) bool {
	if v := c.Locals(LocIsOwner); v != nil {
		if b, ok := v.(bool); ok && b { return true }
		if s, ok := v.(string); ok && strings.EqualFold(s, "true") { return true }
	}
	if HasGlobalRole(c, "owner") { return true }
	return strings.ToLower(GetRole(c)) == "owner"
}

func roleExistsInAnyMasjid(c *fiber.Ctx, role string) bool {
	ids, err := getMasjidIDsFromMasjidRoles(c)
	if err == nil {
		for _, id := range ids { if HasRoleInMasjid(c, id, role) { return true } }
	}
	return false
}

func IsDKM(c *fiber.Ctx) bool {
	if strings.ToLower(GetRole(c)) == "dkm" { return true }
	if roleExistsInAnyMasjid(c, "dkm") { return true }
	return HasUUIDClaim(c, LocMasjidDKMIDs) || HasUUIDClaim(c, LocMasjidAdminIDs)
}

func IsTeacher(c *fiber.Ctx) bool {
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 { return true }
	if strings.EqualFold(GetRole(c), "teacher") || HasGlobalRole(c, "teacher") { return true }
	return roleExistsInAnyMasjid(c, "teacher")
}

func IsStudent(c *fiber.Ctx) bool {
	if recs, err := parseStudentRecordsFromLocals(c); err == nil && len(recs) > 0 { return true }
	if strings.EqualFold(GetRole(c), "student") || HasGlobalRole(c, "student") { return true }
	if roleExistsInAnyMasjid(c, "student") { return true }
	return HasUUIDClaim(c, LocMasjidStudentIDs)
}

/* ============================================
   teacher_records / student_records (deduped parsing)
   ============================================ */

type recordPair struct { MasjidID, SecondID uuid.UUID }

// drain map[string]any to recordPair using keys
func drainPair(m map[string]any, masjidKey, secondKey string) (recordPair, bool) {
	var rp recordPair
	if s, ok := m[masjidKey].(string); ok {
		if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil { rp.MasjidID = id }
	}
	if s, ok := m[secondKey].(string); ok {
		if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil { rp.SecondID = id }
	}
	return rp, rp.MasjidID != uuid.Nil && rp.SecondID != uuid.Nil
}

func parsePairsFromLocals(c *fiber.Ctx, localsKey, masjidKey, secondKey string) ([]recordPair, error) {
	v := c.Locals(localsKey)
	if v == nil {
		if any := claimAny(readJWTClaims(c), localsKey); any != nil { v = any; c.Locals(localsKey, any) }
	}
	if v == nil { return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" tidak ditemukan di token") }
	switch t := v.(type) {
	case []map[string]any:
		out := make([]recordPair, 0, len(t))
		for _, m := range t { if rp, ok := drainPair(m, masjidKey, secondKey); ok { out = append(out, rp) } }
		if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong/invalid") }
		return out, nil
	case []interface{}:
		out := make([]recordPair, 0, len(t))
		for _, it := range t { if m, ok := it.(map[string]any); ok { if rp, ok := drainPair(m, masjidKey, secondKey); ok { out = append(out, rp) } } }
		if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong/invalid") }
		return out, nil
	}
	// allow concrete slices too (fast-path)
	switch t := v.(type) {
	case []TeacherRecordEntry:
		if localsKey == LocTeacherRecords {
			out := make([]recordPair, 0, len(t))
			for _, r := range t { if r.MasjidID != uuid.Nil && r.MasjidTeacherID != uuid.Nil { out = append(out, recordPair{r.MasjidID, r.MasjidTeacherID}) } }
			if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong") }
			return out, nil
		}
	case []StudentRecordEntry:
		if localsKey == LocStudentRecords {
			out := make([]recordPair, 0, len(t))
			for _, r := range t { if r.MasjidID != uuid.Nil && r.MasjidStudentID != uuid.Nil { out = append(out, recordPair{r.MasjidID, r.MasjidStudentID}) } }
			if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong") }
			return out, nil
		}
	}
	return nil, fiber.NewError(fiber.StatusBadRequest, localsKey+" format tidak didukung")
}

// ---- TEACHER ----
func parseTeacherRecordsFromLocals(c *fiber.Ctx) ([]TeacherRecordEntry, error) {
	pairs, err := parsePairsFromLocals(c, LocTeacherRecords, "masjid_id", "masjid_teacher_id")
	if err != nil { return nil, err }
	out := make([]TeacherRecordEntry, 0, len(pairs))
	for _, p := range pairs { out = append(out, TeacherRecordEntry{MasjidTeacherID: p.SecondID, MasjidID: p.MasjidID}) }
	return out, nil
}

func GetTeacherRecordsFromToken(c *fiber.Ctx) ([]TeacherRecordEntry, error) { return parseTeacherRecordsFromLocals(c) }

func GetMasjidTeacherIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil { return nil, err }
	out := make([]uuid.UUID, 0, len(recs))
	seen := map[uuid.UUID]struct{}{}
	for _, r := range recs {
		if r.MasjidTeacherID == uuid.Nil { continue }
		if _, ok := seen[r.MasjidTeacherID]; !ok { seen[r.MasjidTeacherID] = struct{}{}; out = append(out, r.MasjidTeacherID) }
	}
	if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_id tidak ditemukan di token") }
	return out, nil
}

func GetMasjidTeacherIDForMasjid(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) {
	if masjidID == uuid.Nil { return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib") }
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil { return uuid.Nil, err }
	for _, r := range recs { if r.MasjidID == masjidID && r.MasjidTeacherID != uuid.Nil { return r.MasjidTeacherID, nil } }
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_id untuk masjid tersebut tidak ada di token")
}

func GetPrimaryMasjidTeacherID(c *fiber.Ctx) (uuid.UUID, error) {
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil { return uuid.Nil, err }
	if act, err2 := GetActiveMasjidIDFromToken(c); err2 == nil && act != uuid.Nil {
		if mt, e := GetMasjidTeacherIDForMasjid(c, act); e == nil { return mt, nil }
	}
	if len(recs) > 0 && recs[0].MasjidTeacherID != uuid.Nil { return recs[0].MasjidTeacherID, nil }
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_id tidak tersedia")
}

// ---- STUDENT ----
func parseStudentRecordsFromLocals(c *fiber.Ctx) ([]StudentRecordEntry, error) {
	pairs, err := parsePairsFromLocals(c, LocStudentRecords, "masjid_id", "masjid_student_id")
	if err != nil { return nil, err }
	out := make([]StudentRecordEntry, 0, len(pairs))
	for _, p := range pairs { out = append(out, StudentRecordEntry{MasjidStudentID: p.SecondID, MasjidID: p.MasjidID}) }
	return out, nil
}

func GetStudentRecordsFromToken(c *fiber.Ctx) ([]StudentRecordEntry, error) { return parseStudentRecordsFromLocals(c) }

func GetMasjidStudentIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil { return nil, err }
	out := make([]uuid.UUID, 0, len(recs))
	seen := map[uuid.UUID]struct{}{}
	for _, r := range recs {
		if r.MasjidStudentID == uuid.Nil { continue }
		if _, ok := seen[r.MasjidStudentID]; !ok { seen[r.MasjidStudentID] = struct{}{}; out = append(out, r.MasjidStudentID) }
	}
	if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_student_id tidak ditemukan di token") }
	return out, nil
}

func GetMasjidStudentIDForMasjid(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) {
	if masjidID == uuid.Nil { return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib") }
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil { return uuid.Nil, err }
	for _, r := range recs { if r.MasjidID == masjidID && r.MasjidStudentID != uuid.Nil { return r.MasjidStudentID, nil } }
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_student_id untuk masjid tersebut tidak ada di token")
}

func GetPrimaryMasjidStudentID(c *fiber.Ctx) (uuid.UUID, error) {
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil { return uuid.Nil, err }
	if act, err2 := GetActiveMasjidIDFromToken(c); err2 == nil && act != uuid.Nil {
		if sid, e := GetMasjidStudentIDForMasjid(c, act); e == nil { return sid, nil }
	}
	if len(recs) > 0 && recs[0].MasjidStudentID != uuid.Nil { return recs[0].MasjidStudentID, nil }
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_student_id tidak tersedia")
}

// Helper alias for controller
func GetStudentIDFromToken(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) { return GetMasjidStudentIDForMasjid(c, masjidID) }

/* ============================================
   Single-tenant getters (compat + new token)
   ============================================ */

func GetUserIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocUserID); err == nil && id != uuid.Nil { return id, nil }
	if v := c.Locals("sub"); v != nil { if s, ok := v.(string); ok { if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil { c.Locals(LocUserID, id.String()); return id, nil } } }
	if v := c.Locals("id"); v != nil { if s, ok := v.(string); ok { if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil { c.Locals(LocUserID, id.String()); return id, nil } } }
	claims := readJWTClaims(c)
	for _, k := range []string{"id", "sub", "user_id"} {
		if s := claimString(claims, k); s != "" {
			if id, err := uuid.Parse(s); err == nil { c.Locals(LocUserID, id.String()); return id, nil }
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "user_id tidak ditemukan di token")
}

// Admin/DKM only (legacy behavior)
func GetMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocMasjidDKMIDs); err == nil { return id, nil }
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

// Prefer teacher scope
func GetMasjidIDFromTokenPreferTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		if act, err2 := GetActiveMasjidIDFromToken(c); err2 == nil && act != uuid.Nil {
			for _, r := range recs { if r.MasjidID == act { return act, nil } }
		}
		return recs[0].MasjidID, nil
	}
	if id, err := GetMasjidIDFromToken(c); err == nil { return id, nil }
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil && len(ids) > 0 { return ids[0], nil }
	if ids, err := getMasjidIDsFromMasjidRoles(c); err == nil && len(ids) > 0 { return ids[0], nil }
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

// For legacy code that expects TeacherMasjidID == masjid_id where they teach
func GetTeacherMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		if act, err2 := GetActiveMasjidIDFromToken(c); err2 == nil && act != uuid.Nil {
			for _, r := range recs { if r.MasjidID == act { return act, nil } }
		}
		return recs[0].MasjidID, nil
	}
	return parseFirstUUIDFromLocals(c, LocMasjidTeacherIDs)
}

/* ============================================
   Multi-tenant getter
   ============================================ */

func GetMasjidIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil && len(ids) > 0 { return ids, nil }
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		seen := map[uuid.UUID]struct{}{}
		out := make([]uuid.UUID, 0, len(recs))
		for _, r := range recs { if _, ok := seen[r.MasjidID]; !ok { seen[r.MasjidID] = struct{}{}; out = append(out, r.MasjidID) } }
		if len(out) > 0 { return out, nil }
	}
	if recs, err := parseStudentRecordsFromLocals(c); err == nil && len(recs) > 0 {
		seen := map[uuid.UUID]struct{}{}
		out := make([]uuid.UUID, 0, len(recs))
		for _, r := range recs { if _, ok := seen[r.MasjidID]; !ok { seen[r.MasjidID] = struct{}{}; out = append(out, r.MasjidID) } }
		if len(out) > 0 { return out, nil }
	}
	if ids, err := getMasjidIDsFromMasjidRoles(c); err == nil && len(ids) > 0 { return ids, nil }

	groups := []string{LocMasjidTeacherIDs, LocMasjidDKMIDs, LocMasjidAdminIDs, LocMasjidStudentIDs}
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, 4)
	var anyFound bool
	for _, key := range groups {
		v := c.Locals(key); if v == nil { continue }
		anyFound = true
		for _, s := range normalizeLocalsToStrings(v) {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil { return nil, fiber.NewError(fiber.StatusBadRequest, key+" berisi UUID tidak valid") }
			if id == uuid.Nil { continue }
			if _, ok := seen[id]; !ok { seen[id] = struct{}{}; out = append(out, id) }
		}
	}
	if !anyFound || len(out) == 0 {
		if id, err := GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil { return []uuid.UUID{id}, nil }
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	return out, nil
}

/* ============================================
   DB-role helpers (unchanged logic)
   ============================================ */

func EnsureGlobalRole(tx *gorm.DB, userID uuid.UUID, roleName string, assignedBy *uuid.UUID) error {
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") { return nil }
	if quickHasFunction(tx, "fn_grant_role") {
		var idStr string
		if assignedBy != nil {
			return tx.Raw(`SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, ?::uuid)::text`, userID, strings.ToLower(roleName), *assignedBy).Scan(&idStr).Error
		}
		return tx.Raw(`SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, NULL::uuid)::text`, userID, strings.ToLower(roleName)).Scan(&idStr).Error
	}

	var roleID string
	if err := tx.Raw(`SELECT role_id::text FROM roles WHERE LOWER(role_name)=LOWER(?) LIMIT 1`, roleName).Scan(&roleID).Error; err != nil { return err }
	if roleID == "" {
		if err := tx.Raw(`INSERT INTO roles(role_name) VALUES (LOWER(?)) RETURNING role_id::text`, roleName).Scan(&roleID).Error; err != nil { return err }
	}

	var exists bool
	if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id=?::uuid AND role_id=?::uuid AND masjid_id IS NULL AND deleted_at IS NULL)`, userID, roleID).Scan(&exists).Error; err != nil { return err }
	if exists { return nil }

	if assignedBy != nil {
		return tx.Exec(`INSERT INTO user_roles(user_id, role_id, masjid_id, assigned_at, assigned_by) VALUES (?::uuid, ?::uuid, NULL, now(), ?::uuid)`, userID, roleID, *assignedBy).Error
	}
	return tx.Exec(`INSERT INTO user_roles(user_id, role_id, masjid_id, assigned_at) VALUES (?::uuid, ?::uuid, NULL, now())`, userID, roleID).Error
}

func GrantScopedRoleDKM(tx *gorm.DB, userID, masjidID uuid.UUID) error {
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") { return nil }
	if quickHasFunction(tx, "fn_grant_role") {
		var idStr string
		return tx.Raw(`SELECT fn_grant_role(?::uuid, 'dkm'::text, ?::uuid, ?::uuid)::text`, userID, masjidID, userID).Scan(&idStr).Error
	}

	var roleID string
	if err := tx.Raw(`SELECT role_id::text FROM roles WHERE LOWER(role_name)='dkm' LIMIT 1`).Scan(&roleID).Error; err != nil { return err }
	if roleID == "" {
		if err := tx.Raw(`INSERT INTO roles(role_name) VALUES ('dkm') RETURNING role_id::text`).Scan(&roleID).Error; err != nil { return err }
	}
	var exists bool
	if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id=?::uuid AND role_id=?::uuid AND masjid_id=?::uuid AND deleted_at IS NULL)`, userID, roleID, masjidID).Scan(&exists).Error; err != nil { return err }
	if exists { return nil }
	return tx.Exec(`INSERT INTO user_roles(user_id, role_id, masjid_id, assigned_at, assigned_by) VALUES (?::uuid, ?::uuid, ?::uuid, now(), ?::uuid)`, userID, roleID, masjidID, userID).Error
}

/* ============================================
   Misc
   ============================================ */

func HasUUIDClaim(c *fiber.Ctx, key string) bool {
	ids, err := parseUUIDSliceFromLocals(c, key)
	return err == nil && len(ids) > 0
}

func GetActiveMasjidIDIfSingle(rc RolesClaim) *string {
	if len(rc.MasjidRoles) == 1 && rc.MasjidRoles[0].MasjidID != uuid.Nil {
		id := rc.MasjidRoles[0].MasjidID.String()
		return &id
	}
	return nil
}
