// file: internals/helpers/helper.go
package helper

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	helper "masjidku_backend/internals/helpers"

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

	// New structured claims (from verified middleware → set to locals)
	LocRolesGlobal    = "roles_global"     // []string
	LocMasjidRoles    = "masjid_roles"     // []MasjidRolesEntry | []map[string]any
	LocIsOwner        = "is_owner"         // bool | "true"/"false"
	LocActiveMasjidID = "active_masjid_id" // string UUID
	LocTeacherRecords = "teacher_records"  // []TeacherRecordEntry | []map[string]any
	LocStudentRecords = "student_records"  // []StudentRecordEntry | []map[string]any
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
	if db == nil || table == "" { return false }
	var ok bool
	_ = db.Raw(`SELECT to_regclass((SELECT current_schema()) || '.' || ?) IS NOT NULL`, table).Scan(&ok).Error
	return ok
}

func quickHasFunction(db *gorm.DB, name string) bool {
	if db == nil || name == "" { return false }
	var ok bool
	_ = db.Raw(`SELECT EXISTS(SELECT 1 FROM pg_proc WHERE proname = ?)`, name).Scan(&ok).Error
	return ok
}

func normalizeLocalsToStrings(v any) []string {
	out := make([]string, 0)
	switch t := v.(type) {
	case []string:
		for _, s := range t {
			if s = strings.TrimSpace(s); s != "" { out = append(out, s) }
		}
	case []interface{}:
		for _, it := range t {
			switch vv := it.(type) {
			case string:
				if s := strings.TrimSpace(vv); s != "" { out = append(out, s) }
			case uuid.UUID:
				if vv != uuid.Nil { out = append(out, vv.String()) }
			}
		}
	case string:
		if s := strings.TrimSpace(t); s != "" { out = append(out, s) }
	case uuid.UUID:
		if t != uuid.Nil { out = append(out, t.String()) }
	case []uuid.UUID:
		for _, id := range t { if id != uuid.Nil { out = append(out, id.String()) } }
	}
	return out
}

func parseFirstUUIDFromLocals(c *fiber.Ctx, key string) (uuid.UUID, error) {
	v := c.Locals(key)
	if v == nil { return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" tidak ditemukan di token") }
	items := normalizeLocalsToStrings(v)
	if len(items) == 0 { return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token") }
	id, err := uuid.Parse(items[0])
	if err != nil { return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Format "+key+" tidak valid di token") }
	return id, nil
}

func parseUUIDSliceFromLocals(c *fiber.Ctx, key string) ([]uuid.UUID, error) {
	v := c.Locals(key)
	if v == nil { return nil, fiber.NewError(fiber.StatusUnauthorized, key+" tidak ditemukan di token") }
	raw := normalizeLocalsToStrings(v)
	if len(raw) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token") }
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil { return nil, fiber.NewError(fiber.StatusBadRequest, key+" berisi UUID tidak valid") }
		out = append(out, id)
	}
	return out, nil
}

/* ============================================
   JWT fallback utilities (NO signature verify).
   Pakai HANYA untuk data non-kritis (mis. user id fallback).
   ============================================ */

func readJWTClaims(c *fiber.Ctx) map[string]any {
	auth := strings.TrimSpace(c.Get("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		token := parts[len(parts)-1]
		seg := strings.Split(token, ".")
		if len(seg) >= 2 {
			if payload, err := base64.RawURLEncoding.DecodeString(seg[1]); err == nil {
				var m map[string]any
				_ = json.Unmarshal(payload, &m)
				if len(m) > 0 { return m }
			}
		}
	}
	return nil
}

func claimString(claims map[string]any, key string) string {
	if claims == nil { return "" }
	if v, ok := claims[key]; ok {
		if s, ok2 := v.(string); ok2 { return strings.TrimSpace(s) }
	}
	return ""
}

func claimAny(claims map[string]any, key string) any {
	if claims == nil { return nil }
	if v, ok := claims[key]; ok { return v }
	return nil
}

/* ============================================
   roles_global & masjid_roles (STRICT: locals only)
   ============================================ */

func GetRolesGlobal(c *fiber.Ctx) []string {
	v := c.Locals(LocRolesGlobal) // ← hanya locals terverifikasi
	out := make([]string, 0)
	switch t := v.(type) {
	case []string:
		for _, r := range t {
			r = strings.ToLower(strings.TrimSpace(r))
			if r != "" { out = append(out, r) }
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

// --- masjid_roles parsing (STRICT: NO claims-first) ---

func parseMasjidRoles(c *fiber.Ctx) ([]MasjidRolesEntry, error) {
	v := c.Locals(LocMasjidRoles) // ← hanya locals dari middleware
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
		for _, mr := range t {
			if mr.MasjidID != uuid.Nil && len(mr.Roles) > 0 {
				out = append(out, mr)
			}
		}
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
			if m, ok := it.(map[string]any); ok {
				if e, ok := drain(m); ok { out = append(out, e) }
			}
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
		if _, ok := seen[e.MasjidID]; !ok {
			seen[e.MasjidID] = struct{}{}
			out = append(out, e.MasjidID)
		}
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_id tidak ada pada masjid_roles")
	}
	return out, nil
}

func HasRoleInMasjid(c *fiber.Ctx, masjidID uuid.UUID, role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" || masjidID == uuid.Nil { return false }
	entries, err := parseMasjidRoles(c)
	if err != nil { return false }
	for _, e := range entries {
		if e.MasjidID == masjidID {
			for _, r := range e.Roles {
				if r == role { return true }
			}
		}
	}
	return false
}

/* ============================================
   active_masjid_id & role flags (STRICT: locals)
   ============================================ */

func GetActiveMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals(LocActiveMasjidID)
	if v == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, LocActiveMasjidID+" tidak ditemukan di token")
	}
	switch t := v.(type) {
	case string:
		id, err := uuid.Parse(strings.TrimSpace(t))
		if err != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, LocActiveMasjidID+" tidak valid")
		}
		return id, nil
	case uuid.UUID:
		if t != uuid.Nil { return t, nil }
	}
	return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, LocActiveMasjidID+" tidak valid")
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
		for _, id := range ids {
			if HasRoleInMasjid(c, id, role) { return true }
		}
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
   teacher_records / student_records (STRICT: locals)
   ============================================ */

type recordPair struct{ MasjidID, SecondID uuid.UUID }

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
	// STRICT: hanya locals, tidak mengambil dari JWT mentah
	v := c.Locals(localsKey)
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" tidak ditemukan di token")
	}
	switch t := v.(type) {
	case []map[string]any:
		out := make([]recordPair, 0, len(t))
		for _, m := range t { if rp, ok := drainPair(m, masjidKey, secondKey); ok { out = append(out, rp) } }
		if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong/invalid") }
		return out, nil
	case []interface{}:
		out := make([]recordPair, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok {
				if rp, ok := drainPair(m, masjidKey, secondKey); ok { out = append(out, rp) }
			}
		}
		if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong/invalid") }
		return out, nil
	case []TeacherRecordEntry:
		if localsKey == LocTeacherRecords {
			out := make([]recordPair, 0, len(t))
			for _, r := range t {
				if r.MasjidID != uuid.Nil && r.MasjidTeacherID != uuid.Nil {
					out = append(out, recordPair{r.MasjidID, r.MasjidTeacherID})
				}
			}
			if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong") }
			return out, nil
		}
	case []StudentRecordEntry:
		if localsKey == LocStudentRecords {
			out := make([]recordPair, 0, len(t))
			for _, r := range t {
				if r.MasjidID != uuid.Nil && r.MasjidStudentID != uuid.Nil {
					out = append(out, recordPair{r.MasjidID, r.MasjidStudentID})
				}
			}
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
		if _, ok := seen[r.MasjidTeacherID]; !ok {
			seen[r.MasjidTeacherID] = struct{}{}
			out = append(out, r.MasjidTeacherID)
		}
	}
	if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_id tidak ditemukan di token") }
	return out, nil
}

func GetMasjidTeacherIDForMasjid(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) {
	if masjidID == uuid.Nil { return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib") }
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil { return uuid.Nil, err }
	for _, r := range recs {
		if r.MasjidID == masjidID && r.MasjidTeacherID != uuid.Nil {
			return r.MasjidTeacherID, nil
		}
	}
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
		if _, ok := seen[r.MasjidStudentID]; !ok {
			seen[r.MasjidStudentID] = struct{}{}
			out = append(out, r.MasjidStudentID)
		}
	}
	if len(out) == 0 { return nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_student_id tidak ditemukan di token") }
	return out, nil
}

func GetMasjidStudentIDForMasjid(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) {
	if masjidID == uuid.Nil { return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib") }
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil { return uuid.Nil, err }
	for _, r := range recs {
		if r.MasjidID == masjidID && r.MasjidStudentID != uuid.Nil {
			return r.MasjidStudentID, nil
		}
	}
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
func GetStudentIDFromToken(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) {
	return GetMasjidStudentIDForMasjid(c, masjidID)
}

/* ============================================
   Single-tenant getters (compat + new token)
   ============================================ */

func GetUserIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocUserID); err == nil && id != uuid.Nil { return id, nil }
	if v := c.Locals("sub"); v != nil {
		if s, ok := v.(string); ok {
			if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
				c.Locals(LocUserID, id.String())
				return id, nil
			}
		}
	}
	if v := c.Locals("id"); v != nil {
		if s, ok := v.(string); ok {
			if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
				c.Locals(LocUserID, id.String())
				return id, nil
			}
		}
	}
	// Fallback terakhir: dari JWT mentah (non-kritis untuk auth scoping)
	claims := readJWTClaims(c)
	for _, k := range []string{"id", "sub", "user_id"} {
		if s := claimString(claims, k); s != "" {
			if id, err := uuid.Parse(s); err == nil {
				c.Locals(LocUserID, id.String())
				return id, nil
			}
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

/* ============================================
   Multi-tenant getter (STRICT-ish)
   ============================================ */

func GetMasjidIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	// 1) langsung dari locals (diset middleware)
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil && len(ids) > 0 { return ids, nil }

	// 2) teacher_records → daftar masjid
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		seen := map[uuid.UUID]struct{}{}
		out := make([]uuid.UUID, 0, len(recs))
		for _, r := range recs {
			if _, ok := seen[r.MasjidID]; !ok {
				seen[r.MasjidID] = struct{}{}
				out = append(out, r.MasjidID)
			}
		}
		if len(out) > 0 { return out, nil }
	}

	// 3) student_records → daftar masjid
	if recs, err := parseStudentRecordsFromLocals(c); err == nil && len(recs) > 0 {
		seen := map[uuid.UUID]struct{}{}
		out := make([]uuid.UUID, 0, len(recs))
		for _, r := range recs {
			if _, ok := seen[r.MasjidID]; !ok {
				seen[r.MasjidID] = struct{}{}
				out = append(out, r.MasjidID)
			}
		}
		if len(out) > 0 { return out, nil }
	}

	// 4) fallback dari masjid_roles terstruktur
	if ids, err := getMasjidIDsFromMasjidRoles(c); err == nil && len(ids) > 0 { return ids, nil }

	// 5) terakhir: kumpulkan dari locals legacy per peran
	groups := []string{LocMasjidTeacherIDs, LocMasjidDKMIDs, LocMasjidAdminIDs, LocMasjidStudentIDs}
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, 4)
	var anyFound bool
	for _, key := range groups {
		v := c.Locals(key)
		if v == nil { continue }
		anyFound = true
		for _, s := range normalizeLocalsToStrings(v) {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil { return nil, fiber.NewError(fiber.StatusBadRequest, key+" berisi UUID tidak valid") }
			if id == uuid.Nil { continue }
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				out = append(out, id)
			}
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
	if err := tx.Raw(`SELECT role_id::text FROM roles WHERE LOWER(role_name)=LOWER(?) LIMIT 1`, roleName).Scan(&roleID).Error; err != nil {
		return err
	}
	if roleID == "" {
		if err := tx.Raw(`INSERT INTO roles(role_name) VALUES (LOWER(?)) RETURNING role_id::text`, roleName).Scan(&roleID).Error; err != nil {
			return err
		}
	}

	var exists bool
	if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id=?::uuid AND role_id=?::uuid AND masjid_id IS NULL AND deleted_at IS NULL)`, userID, roleID).Scan(&exists).Error; err != nil {
		return err
	}
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
	if err := tx.Raw(`SELECT role_id::text FROM roles WHERE LOWER(role_name)='dkm' LIMIT 1`).Scan(&roleID).Error; err != nil {
		return err
	}
	if roleID == "" {
		if err := tx.Raw(`INSERT INTO roles(role_name) VALUES ('dkm') RETURNING role_id::text`).Scan(&roleID).Error; err != nil {
			return err
		}
	}
	var exists bool
	if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id=?::uuid AND role_id=?::uuid AND masjid_id=?::uuid AND deleted_at IS NULL)`, userID, roleID, masjidID).Scan(&exists).Error; err != nil {
		return err
	}
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

/* ============================================
   Per-masjid legacy helpers (per masjid)
   ============================================ */

func hasUUIDClaimForMasjid(c *fiber.Ctx, key string, masjidID uuid.UUID) bool {
	if masjidID == uuid.Nil { return false }
	ids, err := parseUUIDSliceFromLocals(c, key)
	if err != nil { return false }
	for _, id := range ids { if id == masjidID { return true } }
	return false
}

func IsTeacherInMasjid(c *fiber.Ctx, masjidID uuid.UUID) bool {
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil {
		for _, r := range recs { if r.MasjidID == masjidID { return true } }
	}
	return hasUUIDClaimForMasjid(c, LocMasjidTeacherIDs, masjidID)
}

func IsStudentInMasjid(c *fiber.Ctx, masjidID uuid.UUID) bool {
	if recs, err := parseStudentRecordsFromLocals(c); err == nil {
		for _, r := range recs { if r.MasjidID == masjidID { return true } }
	}
	return hasUUIDClaimForMasjid(c, LocMasjidStudentIDs, masjidID)
}

func IsDKMInMasjid(c *fiber.Ctx, masjidID uuid.UUID) bool {
	if hasUUIDClaimForMasjid(c, LocMasjidDKMIDs, masjidID) { return true }
	if hasUUIDClaimForMasjid(c, LocMasjidAdminIDs, masjidID) { return true }
	return false
}

/* ============================================
   Presence gate (STRICT)
   ============================================ */

func isMasjidPresentInToken(c *fiber.Ctx, masjidID uuid.UUID) bool {
	if masjidID == uuid.Nil { return false }
	// 1) masjid_roles terstruktur
	if entries, err := parseMasjidRoles(c); err == nil {
		for _, e := range entries { if e.MasjidID == masjidID { return true } }
	}
	// 2) masjid_ids umum (hanya jika middleware set)
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil {
		for _, id := range ids { if id == masjidID { return true } }
	}
	return false
}

/* ============================================
   Stale-token gate (opsional)
   ============================================ */

func tokenIATUnix(c *fiber.Ctx) int64 {
	claims := readJWTClaims(c)
	switch v := claimAny(claims, "iat").(type) {
	case float64: return int64(v)
	case int64:   return v
	case json.Number:
		if n, _ := v.Int64(); n > 0 { return n }
	}
	return 0
}

func rejectIfTokenStale(c *fiber.Ctx) error {
	v := c.Locals("roles_updated_at_unix")
	if v == nil { return nil }
	rolesUpdatedAtUnix, _ := v.(int64)
	if rolesUpdatedAtUnix <= 0 { return nil }
	iat := tokenIATUnix(c)
	if iat > 0 && iat < rolesUpdatedAtUnix {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Sesi kedaluwarsa: peran Anda telah diperbarui, silakan login ulang")
	}
	return nil
}

/* ============================================
   Core authorizer (dipakai semua Ensure*)
   ============================================ */

func isPrivileged(c *fiber.Ctx) bool { return IsOwner(c) || HasGlobalRole(c, "superadmin") }

func markGuardOK(c *fiber.Ctx, masjidID uuid.UUID) {
	if masjidID != uuid.Nil { c.Locals("__masjid_guard_ok", masjidID.String()) }
}

// NOTE: nama & signature dipertahankan agar kompatibel.
func ensureRolesInMasjid(
	c *fiber.Ctx,
	masjidID uuid.UUID,
	roles []string,
	legacyFallback func() bool,
	forbidMessage string,
) error {
	if masjidID == uuid.Nil { return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id wajib") }

	// 1) Global bypass
	if isPrivileged(c) {
		markGuardOK(c, masjidID)
		return nil
	}

	// 2) Presence gate (STRICT)
	if !isMasjidPresentInToken(c, masjidID) {
		return helper.JsonError(c, fiber.StatusForbidden, "Masjid ini tidak ada dalam token Anda")
	}

	// 3) (Opsional) Stale-token gate
	if err := rejectIfTokenStale(c); err != nil { return err }

	// 4) Cek peran terstruktur
	for _, r := range roles {
		r = strings.ToLower(strings.TrimSpace(r))
		if HasRoleInMasjid(c, masjidID, r) {
			markGuardOK(c, masjidID)
			return nil
		}
	}

	// 5) Legacy per-masjid fallback
	if legacyFallback != nil && legacyFallback() {
		markGuardOK(c, masjidID)
		return nil
	}

	if strings.TrimSpace(forbidMessage) == "" { forbidMessage = "Tidak diizinkan" }
	return helper.JsonError(c, fiber.StatusForbidden, forbidMessage)
}

/* ============================================
   Publik wrappers (nama & signature tidak berubah)
   ============================================ */

func EnsureMemberMasjid(c *fiber.Ctx, masjidID uuid.UUID) error {
	roles := []string{"student", "teacher", "dkm", "admin", "bendahara"}
	legacy := func() bool { return IsStudentInMasjid(c, masjidID) || IsTeacherInMasjid(c, masjidID) || IsDKMInMasjid(c, masjidID) }
	return ensureRolesInMasjid(c, masjidID, roles, legacy, "Akses hanya untuk anggota masjid ini")
}

func EnsureStaffMasjid(c *fiber.Ctx, masjidID uuid.UUID) error {
	roles := []string{"teacher", "dkm", "admin", "bendahara"}
	legacy := func() bool { return IsTeacherInMasjid(c, masjidID) || IsDKMInMasjid(c, masjidID) }
	return ensureRolesInMasjid(c, masjidID, roles, legacy, "Hanya guru/DKM yang diizinkan")
}

func EnsureStudentMasjid(c *fiber.Ctx, masjidID uuid.UUID) error {
	roles := []string{"student"}
	legacy := func() bool { return IsStudentInMasjid(c, masjidID) }
	return ensureRolesInMasjid(c, masjidID, roles, legacy, "Hanya murid yang diizinkan")
}

func EnsureTeacherMasjid(c *fiber.Ctx, masjidID uuid.UUID) error {
	roles := []string{"teacher"}
	legacy := func() bool { return IsTeacherInMasjid(c, masjidID) }
	return ensureRolesInMasjid(c, masjidID, roles, legacy, "Hanya guru yang diizinkan")
}

func EnsureDKMMasjid(c *fiber.Ctx, masjidID uuid.UUID) error {
	roles := []string{"dkm", "admin"}
	legacy := func() bool { return IsDKMInMasjid(c, masjidID) }
	return ensureRolesInMasjid(c, masjidID, roles, legacy, "Hanya DKM yang diizinkan")
}

func EnsureDKMOrTeacherMasjid(c *fiber.Ctx, masjidID uuid.UUID) error {
	roles := []string{"dkm", "admin", "teacher"}
	legacy := func() bool { return IsDKMInMasjid(c, masjidID) || IsTeacherInMasjid(c, masjidID) }
	return ensureRolesInMasjid(c, masjidID, roles, legacy, "Hanya DKM/Guru yang diizinkan")
}

func OwnerOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !IsOwner(c) { return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan") }
		return c.Next()
	}
}

/* ============================================
   OPTIONAL: Middleware util untuk write-lock
   ============================================ */

// Pastikan semua POST/PUT/PATCH/DELETE ke /api/a/:masjid_id/* sudah lewat Ensure*Masjid
func RequireScopedMasjidWrite() fiber.Handler {
	return func(c *fiber.Ctx) error {
		m := c.Method()
		if m != fiber.MethodPost && m != fiber.MethodPut && m != fiber.MethodPatch && m != fiber.MethodDelete {
			return c.Next()
		}
		if !strings.HasPrefix(c.Path(), "/api/a/") { return c.Next() }

		pathID, err := uuid.Parse(strings.TrimSpace(c.Params("masjid_id")))
		if err != nil || pathID == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id path tidak valid")
		}
		v := c.Locals("__masjid_guard_ok")
		if v == nil {
			return helper.JsonError(c, fiber.StatusForbidden, "Guard authorization tidak terpanggil")
		}
		if s, _ := v.(string); !strings.EqualFold(s, pathID.String()) {
			return helper.JsonError(c, fiber.StatusForbidden, "Guard authorization tidak sesuai masjid")
		}
		return c.Next()
	}
}

// Helper ringan: ambil masjid_id dari path
func ParseMasjidIDFromPath(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("masjid_id")))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, helper.JsonError(c, fiber.StatusBadRequest, "masjid_id path tidak valid")
	}
	return id, nil
}
