// file: internals/helpers/auth/helper.go
package helper

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	helper "madinahsalam_backend/internals/helpers"

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
	LocSchoolIDs        = "school_ids"         // []string | []uuid.UUID (optional)
	LocSchoolAdminIDs   = "school_admin_ids"   // legacy (optional)
	LocSchoolDKMIDs     = "school_dkm_ids"     // legacy (optional)
	LocSchoolTeacherIDs = "school_teacher_ids" // legacy (optional)
	LocSchoolStudentIDs = "school_student_ids" // legacy (optional)

	// New structured claims (from verified middleware → set to locals)
	LocRolesGlobal    = "roles_global"     // []string
	LocSchoolRoles    = "school_roles"     // []SchoolRolesEntry | []map[string]any
	LocIsOwner        = "is_owner"         // bool | "true"/"false"
	LocActiveSchoolID = "active_school_id" // string UUID (optional)

	// Convenience (disarankan middleware ikut set dari JWT claim)
	LocSchoolID  = "school_id"  // string UUID
	LocTeacherID = "teacher_id" // string UUID
	LocStudentID = "student_id" // string UUID
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
   Tiny shared DB helpers
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

/* ============================================
   Locals normalizer
   ============================================ */

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
   JWT payload reader (NO signature verify)
   Hanya sebagai fallback / data non-kritis
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
	if len(m) == 0 {
		return nil
	}
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
   roles_global & school_roles
   ============================================ */

func GetRolesGlobal(c *fiber.Ctx) []string {
	v := c.Locals(LocRolesGlobal)
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

// --- school_roles parsing ---

func parseSchoolRoles(c *fiber.Ctx) ([]SchoolRolesEntry, error) {
	v := c.Locals(LocSchoolRoles)
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
   active_school_id & school_id helpers
   ============================================ */

func GetActiveSchoolIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Prefer locals "active_school_id"
	if v := c.Locals(LocActiveSchoolID); v != nil {
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
	}

	// 2) Coba dari school_roles (dengan asumsi 1 sekolah per sesi)
	if entries, err := parseSchoolRoles(c); err == nil && len(entries) > 0 {
		return entries[0].SchoolID, nil
	}

	// 3) Fallback dari claim school_id
	claims := readJWTClaims(c)
	if s := claimString(claims, "school_id"); s != "" {
		id, err := uuid.Parse(s)
		if err == nil && id != uuid.Nil {
			return id, nil
		}
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id di token tidak valid")
	}

	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "school_id tidak ditemukan di token")
}

func GetActiveSchoolID(c *fiber.Ctx) (uuid.UUID, error) { return GetActiveSchoolIDFromToken(c) }

// Admin/DKM legacy helper – sekarang sama saja: 1 sesi = 1 sekolah
func GetSchoolIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return GetActiveSchoolIDFromToken(c)
}

// Prefer teacher scope → sekarang cukup sama dengan aktif
func GetSchoolIDFromTokenPreferTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	return GetActiveSchoolIDFromToken(c)
}

// Multi-school legacy helper – sekarang selalu 0 atau 1
func GetSchoolIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	id, err := GetActiveSchoolIDFromToken(c)
	if err != nil || id == uuid.Nil {
		return nil, err
	}
	return []uuid.UUID{id}, nil
}

/* ============================================
   User ID
   ============================================ */

func GetUserIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) locals (diset middleware)
	if id, err := parseFirstUUIDFromLocals(c, LocUserID); err == nil && id != uuid.Nil {
		return id, nil
	}

	// 2) legacy locals "sub"/"id"
	for _, key := range []string{"sub", "id"} {
		if v := c.Locals(key); v != nil {
			if s, ok := v.(string); ok {
				if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
					c.Locals(LocUserID, id.String())
					return id, nil
				}
			}
		}
	}

	// 3) Fallback: baca langsung JWT payload (tanpa verify)
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

/* ============================================
   Role flags (global)
   ============================================ */

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

func isAnySchoolRole(c *fiber.Ctx, role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return false
	}
	entries, err := parseSchoolRoles(c)
	if err != nil {
		return false
	}
	for _, e := range entries {
		for _, r := range e.Roles {
			if r == role {
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
	if HasGlobalRole(c, "dkm") || HasGlobalRole(c, "admin") {
		return true
	}
	if isAnySchoolRole(c, "dkm") || isAnySchoolRole(c, "admin") {
		return true
	}
	// Legacy locals fallback
	return HasUUIDClaim(c, LocSchoolDKMIDs) || HasUUIDClaim(c, LocSchoolAdminIDs)
}

func IsTeacher(c *fiber.Ctx) bool {
	if strings.EqualFold(GetRole(c), "teacher") || HasGlobalRole(c, "teacher") {
		return true
	}
	if isAnySchoolRole(c, "teacher") {
		return true
	}
	if _, err := GetTeacherIDFromToken(c); err == nil {
		return true
	}
	return HasUUIDClaim(c, LocSchoolTeacherIDs)
}

func IsStudent(c *fiber.Ctx) bool {
	if strings.EqualFold(GetRole(c), "student") || HasGlobalRole(c, "student") {
		return true
	}
	if isAnySchoolRole(c, "student") {
		return true
	}
	if _, err := GetStudentIDFromToken(c, uuid.Nil); err == nil {
		return true
	}
	return HasUUIDClaim(c, LocSchoolStudentIDs)
}

/* ============================================
   teacher_id / student_id helpers (single)
   ============================================ */

func getUUIDFromLocalsOrClaim(c *fiber.Ctx, localsKey, claimKey, humanName string) (uuid.UUID, error) {
	// 1) locals
	if v := c.Locals(localsKey); v != nil {
		switch t := v.(type) {
		case string:
			id, err := uuid.Parse(strings.TrimSpace(t))
			if err == nil && id != uuid.Nil {
				return id, nil
			}
		case uuid.UUID:
			if t != uuid.Nil {
				return t, nil
			}
		}
	}

	// 2) fallback JWT payload
	claims := readJWTClaims(c)
	if s := claimString(claims, claimKey); s != "" {
		id, err := uuid.Parse(s)
		if err == nil && id != uuid.Nil {
			return id, nil
		}
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, humanName+" tidak valid di token")
	}

	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, humanName+" tidak ditemukan di token")
}

func GetTeacherIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return getUUIDFromLocalsOrClaim(c, LocTeacherID, "teacher_id", "teacher_id")
}

// NOTE: schoolID di-ignore karena 1 sesi = 1 sekolah.
func GetStudentIDFromToken(c *fiber.Ctx, _ uuid.UUID) (uuid.UUID, error) {
	return getUUIDFromLocalsOrClaim(c, LocStudentID, "student_id", "student_id")
}

/* ============================================
   Teacher / Student “records” (compat, 1 sekolah)
   ============================================ */

func GetTeacherRecordsFromToken(c *fiber.Ctx) ([]TeacherRecordEntry, error) {
	tid, err := GetTeacherIDFromToken(c)
	if err != nil {
		return nil, err
	}
	sid, _ := GetActiveSchoolIDFromToken(c)
	return []TeacherRecordEntry{
		{SchoolTeacherID: tid, SchoolID: sid},
	}, nil
}

func GetSchoolTeacherIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	tid, err := GetTeacherIDFromToken(c)
	if err != nil {
		return nil, err
	}
	return []uuid.UUID{tid}, nil
}

func GetSchoolTeacherIDForSchool(c *fiber.Ctx, schoolID uuid.UUID) (uuid.UUID, error) {
	if schoolID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id wajib")
	}
	act, err := GetActiveSchoolIDFromToken(c)
	if err != nil {
		return uuid.Nil, err
	}
	if act != schoolID {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "school_teacher_id untuk school tersebut tidak ada di token")
	}
	return GetTeacherIDFromToken(c)
}

func GetPrimarySchoolTeacherID(c *fiber.Ctx) (uuid.UUID, error) {
	return GetTeacherIDFromToken(c)
}

func GetStudentRecordsFromToken(c *fiber.Ctx) ([]StudentRecordEntry, error) {
	sid, err := GetStudentIDFromToken(c, uuid.Nil)
	if err != nil {
		return nil, err
	}
	schoolID, _ := GetActiveSchoolIDFromToken(c)
	return []StudentRecordEntry{
		{SchoolStudentID: sid, SchoolID: schoolID},
	}, nil
}

func GetSchoolStudentIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	sid, err := GetStudentIDFromToken(c, uuid.Nil)
	if err != nil {
		return nil, err
	}
	return []uuid.UUID{sid}, nil
}

func GetSchoolStudentIDForSchool(c *fiber.Ctx, schoolID uuid.UUID) (uuid.UUID, error) {
	if schoolID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "school_id wajib")
	}
	act, err := GetActiveSchoolIDFromToken(c)
	if err != nil {
		return uuid.Nil, err
	}
	if act != schoolID {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "school_student_id untuk school tersebut tidak ada di token")
	}
	return GetStudentIDFromToken(c, schoolID)
}

func GetPrimarySchoolStudentID(c *fiber.Ctx) (uuid.UUID, error) {
	return GetStudentIDFromToken(c, uuid.Nil)
}

/* Alias yang dipakai controller lama */
func GetStudentIDFromTokenCompat(c *fiber.Ctx, schoolID uuid.UUID) (uuid.UUID, error) {
	return GetStudentIDFromToken(c, schoolID)
}

/* ============================================
   Single-tenant helpers (compat)
   ============================================ */

func HasUUIDClaim(c *fiber.Ctx, key string) bool {
	ids, err := parseUUIDSliceFromLocals(c, key)
	return err == nil && len(ids) > 0
}

/* ============================================
   Multi-tenant claim helper (untuk RolesClaim struct)
   ============================================ */

func GetActiveSchoolIDIfSingle(rc RolesClaim) *string {
	if len(rc.SchoolRoles) == 1 && rc.SchoolRoles[0].SchoolID != uuid.Nil {
		id := rc.SchoolRoles[0].SchoolID.String()
		return &id
	}
	return nil
}

/* ============================================
   Per-school helpers (compat)
   ============================================ */

func IsTeacherInSchool(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if schoolID == uuid.Nil {
		return false
	}
	// 1) Jika token aktif di school ini dan punya teacher_id → true
	if act, err := GetActiveSchoolIDFromToken(c); err == nil && act == schoolID {
		if _, err := GetTeacherIDFromToken(c); err == nil {
			return true
		}
	}
	// 2) legacy locals
	return HasUUIDClaim(c, LocSchoolTeacherIDs)
}

func IsStudentInSchool(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if schoolID == uuid.Nil {
		return false
	}
	if act, err := GetActiveSchoolIDFromToken(c); err == nil && act == schoolID {
		if _, err := GetStudentIDFromToken(c, schoolID); err == nil {
			return true
		}
	}
	return HasUUIDClaim(c, LocSchoolStudentIDs)
}

func IsDKMInSchool(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if schoolID == uuid.Nil {
		return false
	}
	// Dengan model baru, presence cukup dicek via roles
	if HasRoleInSchool(c, schoolID, "dkm") || HasRoleInSchool(c, schoolID, "admin") {
		return true
	}
	return HasUUIDClaim(c, LocSchoolDKMIDs) || HasUUIDClaim(c, LocSchoolAdminIDs)
}

/* ============================================
   Presence gate (STRICT)
   ============================================ */

func isSchoolPresentInToken(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if schoolID == uuid.Nil {
		return false
	}
	act, err := GetActiveSchoolIDFromToken(c)
	if err != nil || act == uuid.Nil {
		return false
	}
	return act == schoolID
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

	// 2) Presence gate
	if !isSchoolPresentInToken(c, schoolID) {
		return helper.JsonError(c, fiber.StatusForbidden, "School ini tidak ada dalam token Anda")
	}

	// 3) (Opsional) Stale-token gate
	if err := rejectIfTokenStale(c); err != nil {
		return err
	}

	// 4) Cek peran di school_roles
	for _, r := range roles {
		r = strings.ToLower(strings.TrimSpace(r))
		if HasRoleInSchool(c, schoolID, r) {
			markGuardOK(c, schoolID)
			return nil
		}
	}

	// 5) Legacy fallback
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
   Publik wrappers (nama & signature dipertahankan)
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

func ResolveSchoolIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Prioritas: school_id dari token (mode guru dulu)
	if id, err := GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		return id, nil
	}

	// 2) Fallback: active-school (kalau kamu pakai mekanisme ini)
	if id, err := GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		return id, nil
	}

	// 3) Kalau tetap nggak ada, unauthorized / bad context
	return uuid.Nil, helper.JsonError(c, fiber.StatusUnauthorized, "school context not found in token")
}

/* ============================================
   Middleware util untuk write-lock
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
			return tx.Raw(`SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, ?::uuid)::text`,
				userID, strings.ToLower(roleName), *assignedBy).Scan(&idStr).Error
		}
		return tx.Raw(`SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, NULL::uuid)::text`,
			userID, strings.ToLower(roleName)).Scan(&idStr).Error
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
	if err := tx.Raw(`SELECT EXISTS(
		SELECT 1 FROM user_roles
		WHERE user_id=?::uuid AND role_id=?::uuid AND school_id IS NULL AND deleted_at IS NULL
	)`, userID, roleID).Scan(&exists).Error; err != nil {
		return err
	}
	if exists {
		return nil
	}

	if assignedBy != nil {
		return tx.Exec(`INSERT INTO user_roles(user_id, role_id, school_id, assigned_at, assigned_by)
			VALUES (?::uuid, ?::uuid, NULL, now(), ?::uuid)`, userID, roleID, *assignedBy).Error
	}
	return tx.Exec(`INSERT INTO user_roles(user_id, role_id, school_id, assigned_at)
		VALUES (?::uuid, ?::uuid, NULL, now())`, userID, roleID).Error
}

func GrantScopedRoleDKM(tx *gorm.DB, userID, schoolID uuid.UUID) error {
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") {
		return nil
	}
	if quickHasFunction(tx, "fn_grant_role") {
		var idStr string
		return tx.Raw(`SELECT fn_grant_role(?::uuid, 'dkm'::text, ?::uuid, ?::uuid)::text`,
			userID, schoolID, userID).Scan(&idStr).Error
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
	if err := tx.Raw(`SELECT EXISTS(
		SELECT 1 FROM user_roles
		WHERE user_id=?::uuid AND role_id=?::uuid AND school_id=?::uuid AND deleted_at IS NULL
	)`, userID, roleID, schoolID).Scan(&exists).Error; err != nil {
		return err
	}
	if exists {
		return nil
	}

	return tx.Exec(`INSERT INTO user_roles(user_id, role_id, school_id, assigned_at, assigned_by)
		VALUES (?::uuid, ?::uuid, ?::uuid, now(), ?::uuid)`,
		userID, roleID, schoolID, userID).Error
}
