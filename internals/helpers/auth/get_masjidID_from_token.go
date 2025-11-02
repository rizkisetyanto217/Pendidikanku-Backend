// file: internals/helpers/helper.go
package helper

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	helper "schoolku_backend/internals/helpers"

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
	LocSchoolIDs        = "school_ids"         // []string | []uuid.UUID
	LocSchoolAdminIDs   = "school_admin_ids"   // []string | []uuid.UUID
	LocSchoolDKMIDs     = "school_dkm_ids"     // []string | []uuid.UUID
	LocSchoolTeacherIDs = "school_teacher_ids" // []string | []uuid.UUID
	LocSchoolStudentIDs = "school_student_ids" // []string | []uuid.UUID

	// New structured claims (from verified middleware → set to locals)
	LocRolesGlobal    = "roles_global"     // []string
	LocSchoolRoles    = "school_roles"     // []SchoolRolesEntry | []map[string]any
	LocIsOwner        = "is_owner"         // bool | "true"/"false"
	LocActiveSchoolID = "active_school_id" // string UUID
	LocTeacherRecords = "teacher_records"  // []TeacherRecordEntry | []map[string]any
	LocStudentRecords = "student_records"  // []StudentRecordEntry | []map[string]any
)

/* ============================================
   Types for structured claims
   ============================================ */

type SchoolRolesEntry struct {
	SchoolID uuid.UUID `json:"school_id"`
	Roles    []string  `json:"roles"`
}

type RolesClaim struct {
	RolesGlobal []string           `json:"roles_global"`
	SchoolRoles []SchoolRolesEntry `json:"school_roles"`
}

type TeacherRecordEntry struct {
	SchoolTeacherID uuid.UUID `json:"school_teacher_id"`
	SchoolID        uuid.UUID `json:"school_id"`
}

type StudentRecordEntry struct {
	SchoolStudentID uuid.UUID `json:"school_student_id"`
	SchoolID        uuid.UUID `json:"school_id"`
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
				if len(m) > 0 {
					return m
				}
			}
		}
	}
	return nil
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
   roles_global & school_roles (STRICT: locals only)
   ============================================ */

func GetRolesGlobal(c *fiber.Ctx) []string {
	v := c.Locals(LocRolesGlobal) // ← hanya locals terverifikasi
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
				if s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func HasGlobalRole(c *fiber.Ctx, role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return false
	}
	for _, r := range GetRolesGlobal(c) {
		if r == role {
			return true
		}
	}
	return false
}

func GetRole(c *fiber.Ctx) string {
	if v := c.Locals(LocRole); v != nil {
		if s, ok := v.(string); ok {
			return strings.ToLower(strings.TrimSpace(s))
		}
	}
	return ""
}

// --- school_roles parsing (STRICT: NO claims-first) ---

func parseSchoolRoles(c *fiber.Ctx) ([]SchoolRolesEntry, error) {
	v := c.Locals(LocSchoolRoles) // ← hanya locals dari middleware
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocSchoolRoles+" tidak ditemukan di token")
	}

	drain := func(m map[string]any) (SchoolRolesEntry, bool) {
		var e SchoolRolesEntry
		if s, ok := m["school_id"].(string); ok {
			if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
				e.SchoolID = id
			}
		}
		if rr, ok := m["roles"].([]interface{}); ok {
			for _, it := range rr {
				if rs, ok := it.(string); ok {
					rs = strings.ToLower(strings.TrimSpace(rs))
					if rs != "" {
						e.Roles = append(e.Roles, rs)
					}
				}
			}
		}
		return e, e.SchoolID != uuid.Nil && len(e.Roles) > 0
	}

	switch t := v.(type) {
	case []SchoolRolesEntry:
		out := make([]SchoolRolesEntry, 0, len(t))
		for _, mr := range t {
			if mr.SchoolID != uuid.Nil && len(mr.Roles) > 0 {
				out = append(out, mr)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocSchoolRoles+" kosong")
		}
		return out, nil
	case []map[string]any:
		out := make([]SchoolRolesEntry, 0, len(t))
		for _, m := range t {
			if e, ok := drain(m); ok {
				out = append(out, e)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocSchoolRoles+" kosong/invalid")
		}
		return out, nil
	case []interface{}:
		out := make([]SchoolRolesEntry, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok {
				if e, ok := drain(m); ok {
					out = append(out, e)
				}
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocSchoolRoles+" kosong/invalid")
		}
		return out, nil
	}
	return nil, fiber.NewError(fiber.StatusBadRequest, LocSchoolRoles+" format tidak didukung")
}

func getSchoolIDsFromSchoolRoles(c *fiber.Ctx) ([]uuid.UUID, error) {
	entries, err := parseSchoolRoles(c)
	if err != nil {
		return nil, err
	}
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		if _, ok := seen[e.SchoolID]; !ok {
			seen[e.SchoolID] = struct{}{}
			out = append(out, e.SchoolID)
		}
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "school_id tidak ada pada school_roles")
	}
	return out, nil
}

func HasRoleInSchool(c *fiber.Ctx, schoolID uuid.UUID, role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" || schoolID == uuid.Nil {
		return false
	}
	entries, err := parseSchoolRoles(c)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.SchoolID == schoolID {
			for _, r := range e.Roles {
				if r == role {
					return true
				}
			}
		}
	}
	return false
}

/* ============================================
   active_school_id & role flags (STRICT: locals)
   ============================================ */

func GetActiveSchoolIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals(LocActiveSchoolID)
	if v == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, LocActiveSchoolID+" tidak ditemukan di token")
	}
	switch t := v.(type) {
	case string:
		id, err := uuid.Parse(strings.TrimSpace(t))
		if err != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, LocActiveSchoolID+" tidak valid")
		}
		return id, nil
	case uuid.UUID:
		if t != uuid.Nil {
			return t, nil
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, LocActiveSchoolID+" tidak valid")
}

func GetActiveSchoolID(c *fiber.Ctx) (uuid.UUID, error) { return GetActiveSchoolIDFromToken(c) }

func IsOwner(c *fiber.Ctx) bool {
	if v := c.Locals(LocIsOwner); v != nil {
		if b, ok := v.(bool); ok && b {
			return true
		}
		if s, ok := v.(string); ok && strings.EqualFold(s, "true") {
			return true
		}
	}
	if HasGlobalRole(c, "owner") {
		return true
	}
	return strings.ToLower(GetRole(c)) == "owner"
}

func roleExistsInAnySchool(c *fiber.Ctx, role string) bool {
	ids, err := getSchoolIDsFromSchoolRoles(c)
	if err == nil {
		for _, id := range ids {
			if HasRoleInSchool(c, id, role) {
				return true
			}
		}
	}
	return false
}

func IsDKM(c *fiber.Ctx) bool {
	if strings.ToLower(GetRole(c)) == "dkm" {
		return true
	}
	if roleExistsInAnySchool(c, "dkm") {
		return true
	}
	return HasUUIDClaim(c, LocSchoolDKMIDs) || HasUUIDClaim(c, LocSchoolAdminIDs)
}

func IsTeacher(c *fiber.Ctx) bool {
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		return true
	}
	if strings.EqualFold(GetRole(c), "teacher") || HasGlobalRole(c, "teacher") {
		return true
	}
	return roleExistsInAnySchool(c, "teacher")
}

func IsStudent(c *fiber.Ctx) bool {
	if recs, err := parseStudentRecordsFromLocals(c); err == nil && len(recs) > 0 {
		return true
	}
	if strings.EqualFold(GetRole(c), "student") || HasGlobalRole(c, "student") {
		return true
	}
	if roleExistsInAnySchool(c, "student") {
		return true
	}
	return HasUUIDClaim(c, LocSchoolStudentIDs)
}

/* ============================================
   teacher_records / student_records (STRICT: locals)
   ============================================ */

type recordPair struct{ SchoolID, SecondID uuid.UUID }

func drainPair(m map[string]any, schoolKey, secondKey string) (recordPair, bool) {
	var rp recordPair
	if s, ok := m[schoolKey].(string); ok {
		if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
			rp.SchoolID = id
		}
	}
	if s, ok := m[secondKey].(string); ok {
		if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
			rp.SecondID = id
		}
	}
	return rp, rp.SchoolID != uuid.Nil && rp.SecondID != uuid.Nil
}

func parsePairsFromLocals(c *fiber.Ctx, localsKey, schoolKey, secondKey string) ([]recordPair, error) {
	// STRICT: hanya locals, tidak mengambil dari JWT mentah
	v := c.Locals(localsKey)
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" tidak ditemukan di token")
	}
	switch t := v.(type) {
	case []map[string]any:
		out := make([]recordPair, 0, len(t))
		for _, m := range t {
			if rp, ok := drainPair(m, schoolKey, secondKey); ok {
				out = append(out, rp)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong/invalid")
		}
		return out, nil
	case []interface{}:
		out := make([]recordPair, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok {
				if rp, ok := drainPair(m, schoolKey, secondKey); ok {
					out = append(out, rp)
				}
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong/invalid")
		}
		return out, nil
	case []TeacherRecordEntry:
		if localsKey == LocTeacherRecords {
			out := make([]recordPair, 0, len(t))
			for _, r := range t {
				if r.SchoolID != uuid.Nil && r.SchoolTeacherID != uuid.Nil {
					out = append(out, recordPair{r.SchoolID, r.SchoolTeacherID})
				}
			}
			if len(out) == 0 {
				return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong")
			}
			return out, nil
		}
	case []StudentRecordEntry:
		if localsKey == LocStudentRecords {
			out := make([]recordPair, 0, len(t))
			for _, r := range t {
				if r.SchoolID != uuid.Nil && r.SchoolStudentID != uuid.Nil {
					out = append(out, recordPair{r.SchoolID, r.SchoolStudentID})
				}
			}
			if len(out) == 0 {
				return nil, fiber.NewError(fiber.StatusUnauthorized, localsKey+" kosong")
			}
			return out, nil
		}
	}
	return nil, fiber.NewError(fiber.StatusBadRequest, localsKey+" format tidak didukung")
}

// ---- TEACHER ----
func parseTeacherRecordsFromLocals(c *fiber.Ctx) ([]TeacherRecordEntry, error) {
	pairs, err := parsePairsFromLocals(c, LocTeacherRecords, "school_id", "school_teacher_id")
	if err != nil {
		return nil, err
	}
	out := make([]TeacherRecordEntry, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, TeacherRecordEntry{SchoolTeacherID: p.SecondID, SchoolID: p.SchoolID})
	}
	return out, nil
}

func GetTeacherRecordsFromToken(c *fiber.Ctx) ([]TeacherRecordEntry, error) {
	return parseTeacherRecordsFromLocals(c)
}

func GetSchoolTeacherIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(recs))
	seen := map[uuid.UUID]struct{}{}
	for _, r := range recs {
		if r.SchoolTeacherID == uuid.Nil {
			continue
		}
		if _, ok := seen[r.SchoolTeacherID]; !ok {
			seen[r.SchoolTeacherID] = struct{}{}
			out = append(out, r.SchoolTeacherID)
		}
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "school_teacher_id tidak ditemukan di token")
	}
	return out, nil
}

func GetSchoolTeacherIDForSchool(c *fiber.Ctx, schoolID uuid.UUID) (uuid.UUID, error) {
	if schoolID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id wajib")
	}
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil {
		return uuid.Nil, err
	}
	for _, r := range recs {
		if r.SchoolID == schoolID && r.SchoolTeacherID != uuid.Nil {
			return r.SchoolTeacherID, nil
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "school_teacher_id untuk school tersebut tidak ada di token")
}

func GetPrimarySchoolTeacherID(c *fiber.Ctx) (uuid.UUID, error) {
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil {
		return uuid.Nil, err
	}
	if act, err2 := GetActiveSchoolIDFromToken(c); err2 == nil && act != uuid.Nil {
		if mt, e := GetSchoolTeacherIDForSchool(c, act); e == nil {
			return mt, nil
		}
	}
	if len(recs) > 0 && recs[0].SchoolTeacherID != uuid.Nil {
		return recs[0].SchoolTeacherID, nil
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "school_teacher_id tidak tersedia")
}

// ---- STUDENT ----
func parseStudentRecordsFromLocals(c *fiber.Ctx) ([]StudentRecordEntry, error) {
	pairs, err := parsePairsFromLocals(c, LocStudentRecords, "school_id", "school_student_id")
	if err != nil {
		return nil, err
	}
	out := make([]StudentRecordEntry, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, StudentRecordEntry{SchoolStudentID: p.SecondID, SchoolID: p.SchoolID})
	}
	return out, nil
}

func GetStudentRecordsFromToken(c *fiber.Ctx) ([]StudentRecordEntry, error) {
	return parseStudentRecordsFromLocals(c)
}

func GetSchoolStudentIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(recs))
	seen := map[uuid.UUID]struct{}{}
	for _, r := range recs {
		if r.SchoolStudentID == uuid.Nil {
			continue
		}
		if _, ok := seen[r.SchoolStudentID]; !ok {
			seen[r.SchoolStudentID] = struct{}{}
			out = append(out, r.SchoolStudentID)
		}
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "school_student_id tidak ditemukan di token")
	}
	return out, nil
}

func GetSchoolStudentIDForSchool(c *fiber.Ctx, schoolID uuid.UUID) (uuid.UUID, error) {
	if schoolID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id wajib")
	}
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil {
		return uuid.Nil, err
	}
	for _, r := range recs {
		if r.SchoolID == schoolID && r.SchoolStudentID != uuid.Nil {
			return r.SchoolStudentID, nil
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "school_student_id untuk school tersebut tidak ada di token")
}

func GetPrimarySchoolStudentID(c *fiber.Ctx) (uuid.UUID, error) {
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil {
		return uuid.Nil, err
	}
	if act, err2 := GetActiveSchoolIDFromToken(c); err2 == nil && act != uuid.Nil {
		if sid, e := GetSchoolStudentIDForSchool(c, act); e == nil {
			return sid, nil
		}
	}
	if len(recs) > 0 && recs[0].SchoolStudentID != uuid.Nil {
		return recs[0].SchoolStudentID, nil
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "school_student_id tidak tersedia")
}

// Helper alias for controller
func GetStudentIDFromToken(c *fiber.Ctx, schoolID uuid.UUID) (uuid.UUID, error) {
	return GetSchoolStudentIDForSchool(c, schoolID)
}

/* ============================================
   Single-tenant getters (compat + new token)
   ============================================ */

func GetUserIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocUserID); err == nil && id != uuid.Nil {
		return id, nil
	}
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
func GetSchoolIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocSchoolDKMIDs); err == nil {
		return id, nil
	}
	return parseFirstUUIDFromLocals(c, LocSchoolAdminIDs)
}

// Prefer teacher scope
func GetSchoolIDFromTokenPreferTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		if act, err2 := GetActiveSchoolIDFromToken(c); err2 == nil && act != uuid.Nil {
			for _, r := range recs {
				if r.SchoolID == act {
					return act, nil
				}
			}
		}
		return recs[0].SchoolID, nil
	}
	if id, err := GetSchoolIDFromToken(c); err == nil {
		return id, nil
	}
	if ids, err := parseUUIDSliceFromLocals(c, LocSchoolIDs); err == nil && len(ids) > 0 {
		return ids[0], nil
	}
	if ids, err := getSchoolIDsFromSchoolRoles(c); err == nil && len(ids) > 0 {
		return ids[0], nil
	}
	return parseFirstUUIDFromLocals(c, LocSchoolAdminIDs)
}

/* ============================================
   Multi-tenant getter (STRICT-ish)
   ============================================ */

func GetSchoolIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	// 1) langsung dari locals (diset middleware)
	if ids, err := parseUUIDSliceFromLocals(c, LocSchoolIDs); err == nil && len(ids) > 0 {
		return ids, nil
	}

	// 2) teacher_records → daftar school
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		seen := map[uuid.UUID]struct{}{}
		out := make([]uuid.UUID, 0, len(recs))
		for _, r := range recs {
			if _, ok := seen[r.SchoolID]; !ok {
				seen[r.SchoolID] = struct{}{}
				out = append(out, r.SchoolID)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}

	// 3) student_records → daftar school
	if recs, err := parseStudentRecordsFromLocals(c); err == nil && len(recs) > 0 {
		seen := map[uuid.UUID]struct{}{}
		out := make([]uuid.UUID, 0, len(recs))
		for _, r := range recs {
			if _, ok := seen[r.SchoolID]; !ok {
				seen[r.SchoolID] = struct{}{}
				out = append(out, r.SchoolID)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}

	// 4) fallback dari school_roles terstruktur
	if ids, err := getSchoolIDsFromSchoolRoles(c); err == nil && len(ids) > 0 {
		return ids, nil
	}

	// 5) terakhir: kumpulkan dari locals legacy per peran
	groups := []string{LocSchoolTeacherIDs, LocSchoolDKMIDs, LocSchoolAdminIDs, LocSchoolStudentIDs}
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, 4)
	var anyFound bool
	for _, key := range groups {
		v := c.Locals(key)
		if v == nil {
			continue
		}
		anyFound = true
		for _, s := range normalizeLocalsToStrings(v) {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil {
				return nil, fiber.NewError(fiber.StatusBadRequest, key+" berisi UUID tidak valid")
			}
			if id == uuid.Nil {
				continue
			}
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				out = append(out, id)
			}
		}
	}
	if !anyFound || len(out) == 0 {
		if id, err := GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			return []uuid.UUID{id}, nil
		}
		return nil, fiber.NewError(fiber.StatusUnauthorized, "School ID tidak ditemukan di token")
	}
	return out, nil
}

/* ============================================
   DB-role helpers (unchanged logic)
   ============================================ */

func EnsureGlobalRole(tx *gorm.DB, userID uuid.UUID, roleName string, assignedBy *uuid.UUID) error {
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") {
		return nil
	}
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
	if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id=?::uuid AND role_id=?::uuid AND school_id IS NULL AND deleted_at IS NULL)`, userID, roleID).Scan(&exists).Error; err != nil {
		return err
	}
	if exists {
		return nil
	}

	if assignedBy != nil {
		return tx.Exec(`INSERT INTO user_roles(user_id, role_id, school_id, assigned_at, assigned_by) VALUES (?::uuid, ?::uuid, NULL, now(), ?::uuid)`, userID, roleID, *assignedBy).Error
	}
	return tx.Exec(`INSERT INTO user_roles(user_id, role_id, school_id, assigned_at) VALUES (?::uuid, ?::uuid, NULL, now())`, userID, roleID).Error
}

func GrantScopedRoleDKM(tx *gorm.DB, userID, schoolID uuid.UUID) error {
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") {
		return nil
	}
	if quickHasFunction(tx, "fn_grant_role") {
		var idStr string
		return tx.Raw(`SELECT fn_grant_role(?::uuid, 'dkm'::text, ?::uuid, ?::uuid)::text`, userID, schoolID, userID).Scan(&idStr).Error
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
	if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id=?::uuid AND role_id=?::uuid AND school_id=?::uuid AND deleted_at IS NULL)`, userID, roleID, schoolID).Scan(&exists).Error; err != nil {
		return err
	}
	if exists {
		return nil
	}
	return tx.Exec(`INSERT INTO user_roles(user_id, role_id, school_id, assigned_at, assigned_by) VALUES (?::uuid, ?::uuid, ?::uuid, now(), ?::uuid)`, userID, roleID, schoolID, userID).Error
}

/* ============================================
   Misc
   ============================================ */

func HasUUIDClaim(c *fiber.Ctx, key string) bool {
	ids, err := parseUUIDSliceFromLocals(c, key)
	return err == nil && len(ids) > 0
}

func GetActiveSchoolIDIfSingle(rc RolesClaim) *string {
	if len(rc.SchoolRoles) == 1 && rc.SchoolRoles[0].SchoolID != uuid.Nil {
		id := rc.SchoolRoles[0].SchoolID.String()
		return &id
	}
	return nil
}

/* ============================================
   Per-school legacy helpers (per school)
   ============================================ */

func hasUUIDClaimForSchool(c *fiber.Ctx, key string, schoolID uuid.UUID) bool {
	if schoolID == uuid.Nil {
		return false
	}
	ids, err := parseUUIDSliceFromLocals(c, key)
	if err != nil {
		return false
	}
	for _, id := range ids {
		if id == schoolID {
			return true
		}
	}
	return false
}

func IsTeacherInSchool(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil {
		for _, r := range recs {
			if r.SchoolID == schoolID {
				return true
			}
		}
	}
	return hasUUIDClaimForSchool(c, LocSchoolTeacherIDs, schoolID)
}

func IsStudentInSchool(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if recs, err := parseStudentRecordsFromLocals(c); err == nil {
		for _, r := range recs {
			if r.SchoolID == schoolID {
				return true
			}
		}
	}
	return hasUUIDClaimForSchool(c, LocSchoolStudentIDs, schoolID)
}

func IsDKMInSchool(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if hasUUIDClaimForSchool(c, LocSchoolDKMIDs, schoolID) {
		return true
	}
	if hasUUIDClaimForSchool(c, LocSchoolAdminIDs, schoolID) {
		return true
	}
	return false
}

/* ============================================
   Presence gate (STRICT)
   ============================================ */

func isSchoolPresentInToken(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if schoolID == uuid.Nil {
		return false
	}
	// 1) school_roles terstruktur
	if entries, err := parseSchoolRoles(c); err == nil {
		for _, e := range entries {
			if e.SchoolID == schoolID {
				return true
			}
		}
	}
	// 2) school_ids umum (hanya jika middleware set)
	if ids, err := parseUUIDSliceFromLocals(c, LocSchoolIDs); err == nil {
		for _, id := range ids {
			if id == schoolID {
				return true
			}
		}
	}
	return false
}

/* ============================================
   Stale-token gate (opsional)
   ============================================ */

func tokenIATUnix(c *fiber.Ctx) int64 {
	claims := readJWTClaims(c)
	switch v := claimAny(claims, "iat").(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case json.Number:
		if n, _ := v.Int64(); n > 0 {
			return n
		}
	}
	return 0
}

func rejectIfTokenStale(c *fiber.Ctx) error {
	v := c.Locals("roles_updated_at_unix")
	if v == nil {
		return nil
	}
	rolesUpdatedAtUnix, _ := v.(int64)
	if rolesUpdatedAtUnix <= 0 {
		return nil
	}
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

func markGuardOK(c *fiber.Ctx, schoolID uuid.UUID) {
	if schoolID != uuid.Nil {
		c.Locals("__school_guard_ok", schoolID.String())
	}
}

// NOTE: nama & signature dipertahankan agar kompatibel.
func ensureRolesInSchool(
	c *fiber.Ctx,
	schoolID uuid.UUID,
	roles []string,
	legacyFallback func() bool,
	forbidMessage string,
) error {
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id wajib")
	}

	// 1) Global bypass
	if isPrivileged(c) {
		markGuardOK(c, schoolID)
		return nil
	}

	// 2) Presence gate (STRICT)
	if !isSchoolPresentInToken(c, schoolID) {
		return helper.JsonError(c, fiber.StatusForbidden, "School ini tidak ada dalam token Anda")
	}

	// 3) (Opsional) Stale-token gate
	if err := rejectIfTokenStale(c); err != nil {
		return err
	}

	// 4) Cek peran terstruktur
	for _, r := range roles {
		r = strings.ToLower(strings.TrimSpace(r))
		if HasRoleInSchool(c, schoolID, r) {
			markGuardOK(c, schoolID)
			return nil
		}
	}

	// 5) Legacy per-school fallback
	if legacyFallback != nil && legacyFallback() {
		markGuardOK(c, schoolID)
		return nil
	}

	if strings.TrimSpace(forbidMessage) == "" {
		forbidMessage = "Tidak diizinkan"
	}
	return helper.JsonError(c, fiber.StatusForbidden, forbidMessage)
}

/* ============================================
   Publik wrappers (nama & signature tidak berubah)
   ============================================ */

func EnsureMemberSchool(c *fiber.Ctx, schoolID uuid.UUID) error {
	roles := []string{"student", "teacher", "dkm", "admin", "bendahara"}
	legacy := func() bool {
		return IsStudentInSchool(c, schoolID) || IsTeacherInSchool(c, schoolID) || IsDKMInSchool(c, schoolID)
	}
	return ensureRolesInSchool(c, schoolID, roles, legacy, "Akses hanya untuk anggota school ini")
}

func EnsureStaffSchool(c *fiber.Ctx, schoolID uuid.UUID) error {
	roles := []string{"teacher", "dkm", "admin", "bendahara"}
	legacy := func() bool { return IsTeacherInSchool(c, schoolID) || IsDKMInSchool(c, schoolID) }
	return ensureRolesInSchool(c, schoolID, roles, legacy, "Hanya guru/DKM yang diizinkan")
}

func EnsureStudentSchool(c *fiber.Ctx, schoolID uuid.UUID) error {
	roles := []string{"student"}
	legacy := func() bool { return IsStudentInSchool(c, schoolID) }
	return ensureRolesInSchool(c, schoolID, roles, legacy, "Hanya murid yang diizinkan")
}

func EnsureTeacherSchool(c *fiber.Ctx, schoolID uuid.UUID) error {
	roles := []string{"teacher"}
	legacy := func() bool { return IsTeacherInSchool(c, schoolID) }
	return ensureRolesInSchool(c, schoolID, roles, legacy, "Hanya guru yang diizinkan")
}

func EnsureDKMSchool(c *fiber.Ctx, schoolID uuid.UUID) error {
	roles := []string{"dkm", "admin"}
	legacy := func() bool { return IsDKMInSchool(c, schoolID) }
	return ensureRolesInSchool(c, schoolID, roles, legacy, "Hanya DKM yang diizinkan")
}

func EnsureDKMOrTeacherSchool(c *fiber.Ctx, schoolID uuid.UUID) error {
	roles := []string{"dkm", "admin", "teacher"}
	legacy := func() bool { return IsDKMInSchool(c, schoolID) || IsTeacherInSchool(c, schoolID) }
	return ensureRolesInSchool(c, schoolID, roles, legacy, "Hanya DKM/Guru yang diizinkan")
}

func OwnerOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !IsOwner(c) {
			return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan")
		}
		return c.Next()
	}
}

/* ============================================
   OPTIONAL: Middleware util untuk write-lock
   ============================================ */

// Pastikan semua POST/PUT/PATCH/DELETE ke /api/a/:school_id/* sudah lewat Ensure*School
func RequireScopedSchoolWrite() fiber.Handler {
	return func(c *fiber.Ctx) error {
		m := c.Method()
		if m != fiber.MethodPost && m != fiber.MethodPut && m != fiber.MethodPatch && m != fiber.MethodDelete {
			return c.Next()
		}
		if !strings.HasPrefix(c.Path(), "/api/a/") {
			return c.Next()
		}

		pathID, err := uuid.Parse(strings.TrimSpace(c.Params("school_id")))
		if err != nil || pathID == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_id path tidak valid")
		}
		v := c.Locals("__school_guard_ok")
		if v == nil {
			return helper.JsonError(c, fiber.StatusForbidden, "Guard authorization tidak terpanggil")
		}
		if s, _ := v.(string); !strings.EqualFold(s, pathID.String()) {
			return helper.JsonError(c, fiber.StatusForbidden, "Guard authorization tidak sesuai school")
		}
		return c.Next()
	}
}

// Helper ringan: ambil school_id dari path
func ParseSchoolIDFromPath(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("school_id")))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, helper.JsonError(c, fiber.StatusBadRequest, "school_id path tidak valid")
	}
	return id, nil
}
