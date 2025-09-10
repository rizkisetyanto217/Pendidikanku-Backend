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
	LocRolesGlobal     = "roles_global"      // []string
	LocMasjidRoles     = "masjid_roles"      // []MasjidRolesEntry | []map[string]any
	LocIsOwner         = "is_owner"          // bool | "true"/"false"
	LocActiveMasjidID  = "active_masjid_id"  // string UUID
	LocTeacherRecords  = "teacher_records"   // []TeacherRecordEntry | []map[string]any
	LocStudentRecords  = "student_records"   // []StudentRecordEntry | []map[string]any (BARU)
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

// BARU: student_records
type StudentRecordEntry struct {
	MasjidStudentID uuid.UUID `json:"masjid_student_id"`
	MasjidID        uuid.UUID `json:"masjid_id"`
}

/* ============================================
   Quick schema helpers (LOCAL)
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

// Cek apakah di Locals ada klaim UUID untuk key tertentu (mis. LocMasjidAdminIDs/LocMasjidDKMIDs)
func HasUUIDClaim(c *fiber.Ctx, key string) bool {
	ids, err := parseUUIDSliceFromLocals(c, key)
	return err == nil && len(ids) > 0
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
   Locals parsers (robust to various shapes)
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
   helpers: roles_global & masjid_roles
   ============================================ */

func GetRolesGlobal(c *fiber.Ctx) []string {
	// 1) locals
	v := c.Locals(LocRolesGlobal)
	if v == nil {
		// 2) JWT fallback
		if arr := claimAny(readJWTClaims(c), "roles_global"); arr != nil {
			v = arr
			// cache ke locals
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

// Parse masjid_roles → []MasjidRolesEntry (robust)
func parseMasjidRoles(c *fiber.Ctx) ([]MasjidRolesEntry, error) {
	v := c.Locals(LocMasjidRoles)
	if v == nil {
		// JWT fallback
		if any := claimAny(readJWTClaims(c), "masjid_roles"); any != nil {
			v = any
			c.Locals(LocMasjidRoles, any)
		}
	}
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" tidak ditemukan di token")
	}

	switch t := v.(type) {
	case []MasjidRolesEntry:
		out := make([]MasjidRolesEntry, 0, len(t))
		for _, mr := range t {
			if mr.MasjidID != uuid.Nil && len(mr.Roles) > 0 {
				out = append(out, mr)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" kosong")
		}
		return out, nil

	case []map[string]interface{}:
		out := make([]MasjidRolesEntry, 0, len(t))
		for _, m := range t {
			var e MasjidRolesEntry
			if s, ok := m["masjid_id"].(string); ok {
				if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
					e.MasjidID = id
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
			if e.MasjidID != uuid.Nil && len(e.Roles) > 0 {
				out = append(out, e)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" kosong/invalid")
		}
		return out, nil

	case []interface{}:
		// kadang JWT decode kasih []any
		out := make([]MasjidRolesEntry, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok {
				var e MasjidRolesEntry
				if s, ok := m["masjid_id"].(string); ok {
					if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
						e.MasjidID = id
					}
				}
				if rr, ok := m["roles"].([]interface{}); ok {
					for _, r := range rr {
						if rs, ok := r.(string); ok {
							rs = strings.ToLower(strings.TrimSpace(rs))
							if rs != "" {
								e.Roles = append(e.Roles, rs)
							}
						}
					}
				}
				if e.MasjidID != uuid.Nil && len(e.Roles) > 0 {
					out = append(out, e)
				}
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" kosong/invalid")
		}
		return out, nil
	}
	return nil, fiber.NewError(fiber.StatusBadRequest, LocMasjidRoles+" format tidak didukung")
}

func getMasjidIDsFromMasjidRoles(c *fiber.Ctx) ([]uuid.UUID, error) {
	entries, err := parseMasjidRoles(c)
	if err != nil {
		return nil, err
	}
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
	if role == "" || masjidID == uuid.Nil {
		return false
	}
	entries, err := parseMasjidRoles(c)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.MasjidID == masjidID {
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
   teacher_records & active_masjid_id
   ============================================ */

func GetActiveMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals(LocActiveMasjidID)
	if v == nil {
		// JWT fallback
		if s := claimString(readJWTClaims(c), "active_masjid_id"); s != "" {
			c.Locals(LocActiveMasjidID, s)
			v = s
		}
	}
	if v == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, LocActiveMasjidID+" tidak ditemukan di token")
	}
	s := ""
	switch t := v.(type) {
	case string:
		s = t
	case uuid.UUID:
		if t != uuid.Nil {
			return t, nil
		}
	}
	id, err := uuid.Parse(strings.TrimSpace(s))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, LocActiveMasjidID+" tidak valid")
	}
	return id, nil
}

// Alias (supaya seragam dengan pemanggilan di controller)
func GetActiveMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
	return GetActiveMasjidIDFromToken(c)
}

func parseTeacherRecordsFromLocals(c *fiber.Ctx) ([]TeacherRecordEntry, error) {
	v := c.Locals(LocTeacherRecords)
	if v == nil {
		// JWT fallback
		if any := claimAny(readJWTClaims(c), "teacher_records"); any != nil {
			v = any
			c.Locals(LocTeacherRecords, any)
		}
	}
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocTeacherRecords+" tidak ditemukan di token")
	}

	switch t := v.(type) {
	case []TeacherRecordEntry:
		out := make([]TeacherRecordEntry, 0, len(t))
		for _, tr := range t {
			if tr.MasjidID != uuid.Nil && tr.MasjidTeacherID != uuid.Nil {
				out = append(out, tr)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocTeacherRecords+" kosong")
		}
		return out, nil

	case []map[string]interface{}:
		out := make([]TeacherRecordEntry, 0, len(t))
		for _, m := range t {
			var tr TeacherRecordEntry
			if s, ok := m["masjid_id"].(string); ok {
				if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
					tr.MasjidID = id
				}
			}
			if s, ok := m["masjid_teacher_id"].(string); ok {
				if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
					tr.MasjidTeacherID = id
				}
			}
			if tr.MasjidID != uuid.Nil && tr.MasjidTeacherID != uuid.Nil {
				out = append(out, tr)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocTeacherRecords+" kosong/invalid")
		}
		return out, nil

	case []interface{}:
		out := make([]TeacherRecordEntry, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]interface{}); ok {
				var tr TeacherRecordEntry
				if s, ok := m["masjid_id"].(string); ok {
					if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
						tr.MasjidID = id
					}
				}
				if s, ok := m["masjid_teacher_id"].(string); ok {
					if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
						tr.MasjidTeacherID = id
					}
				}
				if tr.MasjidID != uuid.Nil && tr.MasjidTeacherID != uuid.Nil {
					out = append(out, tr)
				}
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocTeacherRecords+" kosong/invalid")
		}
		return out, nil
	}
	return nil, fiber.NewError(fiber.StatusBadRequest, LocTeacherRecords+" format tidak didukung")
}

func GetTeacherRecordsFromToken(c *fiber.Ctx) ([]TeacherRecordEntry, error) {
	return parseTeacherRecordsFromLocals(c)
}

func GetMasjidTeacherIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(recs))
	seen := map[uuid.UUID]struct{}{}
	for _, r := range recs {
		if r.MasjidTeacherID == uuid.Nil {
			continue
		}
		if _, ok := seen[r.MasjidTeacherID]; !ok {
			seen[r.MasjidTeacherID] = struct{}{}
			out = append(out, r.MasjidTeacherID)
		}
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_id tidak ditemukan di token")
	}
	return out, nil
}

func GetMasjidTeacherIDForMasjid(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) {
	if masjidID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib")
	}
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil {
		return uuid.Nil, err
	}
	for _, r := range recs {
		if r.MasjidID == masjidID && r.MasjidTeacherID != uuid.Nil {
			return r.MasjidTeacherID, nil
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_id untuk masjid tersebut tidak ada di token")
}

func GetPrimaryMasjidTeacherID(c *fiber.Ctx) (uuid.UUID, error) {
	recs, err := parseTeacherRecordsFromLocals(c)
	if err != nil {
		return uuid.Nil, err
	}
	if act, err2 := GetActiveMasjidIDFromToken(c); err2 == nil && act != uuid.Nil {
		if mt, e := GetMasjidTeacherIDForMasjid(c, act); e == nil {
			return mt, nil
		}
	}
	if len(recs) > 0 && recs[0].MasjidTeacherID != uuid.Nil {
		return recs[0].MasjidTeacherID, nil
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_id tidak tersedia")
}

/* ============================================
   STUDENT records (BARU)
   ============================================ */

func parseStudentRecordsFromLocals(c *fiber.Ctx) ([]StudentRecordEntry, error) {
	v := c.Locals(LocStudentRecords)
	if v == nil {
		// JWT fallback
		if any := claimAny(readJWTClaims(c), "student_records"); any != nil {
			v = any
			c.Locals(LocStudentRecords, any)
		}
	}
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocStudentRecords+" tidak ditemukan di token")
	}

	switch t := v.(type) {
	case []StudentRecordEntry:
		out := make([]StudentRecordEntry, 0, len(t))
		for _, sr := range t {
			if sr.MasjidID != uuid.Nil && sr.MasjidStudentID != uuid.Nil {
				out = append(out, sr)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocStudentRecords+" kosong")
		}
		return out, nil

	case []map[string]interface{}:
		out := make([]StudentRecordEntry, 0, len(t))
		for _, m := range t {
			var sr StudentRecordEntry
			if s, ok := m["masjid_id"].(string); ok {
				if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
					sr.MasjidID = id
				}
			}
			if s, ok := m["masjid_student_id"].(string); ok {
				if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
					sr.MasjidStudentID = id
				}
			}
			if sr.MasjidID != uuid.Nil && sr.MasjidStudentID != uuid.Nil {
				out = append(out, sr)
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocStudentRecords+" kosong/invalid")
		}
		return out, nil

	case []interface{}:
		out := make([]StudentRecordEntry, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok {
				var sr StudentRecordEntry
				if s, ok := m["masjid_id"].(string); ok {
					if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
						sr.MasjidID = id
					}
				}
				if s, ok := m["masjid_student_id"].(string); ok {
					if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
						sr.MasjidStudentID = id
					}
				}
				if sr.MasjidID != uuid.Nil && sr.MasjidStudentID != uuid.Nil {
					out = append(out, sr)
				}
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusUnauthorized, LocStudentRecords+" kosong/invalid")
		}
		return out, nil
	}
	return nil, fiber.NewError(fiber.StatusBadRequest, LocStudentRecords+" format tidak didukung")
}

func GetStudentRecordsFromToken(c *fiber.Ctx) ([]StudentRecordEntry, error) {
	return parseStudentRecordsFromLocals(c)
}

func GetMasjidStudentIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(recs))
	seen := map[uuid.UUID]struct{}{}
	for _, r := range recs {
		if r.MasjidStudentID == uuid.Nil {
			continue
		}
		if _, ok := seen[r.MasjidStudentID]; !ok {
			seen[r.MasjidStudentID] = struct{}{}
			out = append(out, r.MasjidStudentID)
		}
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_student_id tidak ditemukan di token")
	}
	return out, nil
}

func GetMasjidStudentIDForMasjid(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) {
	if masjidID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib")
	}
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil {
		return uuid.Nil, err
	}
	for _, r := range recs {
		if r.MasjidID == masjidID && r.MasjidStudentID != uuid.Nil {
			return r.MasjidStudentID, nil
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_student_id untuk masjid tersebut tidak ada di token")
}

func GetPrimaryMasjidStudentID(c *fiber.Ctx) (uuid.UUID, error) {
	recs, err := parseStudentRecordsFromLocals(c)
	if err != nil {
		return uuid.Nil, err
	}
	if act, err2 := GetActiveMasjidIDFromToken(c); err2 == nil && act != uuid.Nil {
		if sid, e := GetMasjidStudentIDForMasjid(c, act); e == nil {
			return sid, nil
		}
	}
	if len(recs) > 0 && recs[0].MasjidStudentID != uuid.Nil {
		return recs[0].MasjidStudentID, nil
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "masjid_student_id tidak tersedia")
}

// Helper yang kamu butuhkan di controller: cari student_id untuk masjid tertentu.
func GetStudentIDFromToken(c *fiber.Ctx, masjidID uuid.UUID) (uuid.UUID, error) {
	return GetMasjidStudentIDForMasjid(c, masjidID)
}

/* ============================================
   Single-tenant getters (compat + new token)
   ============================================ */

func GetUserIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) standar: user_id
	if id, err := parseFirstUUIDFromLocals(c, LocUserID); err == nil && id != uuid.Nil {
		return id, nil
	}
	// 2) fallback: sub
	if v := c.Locals("sub"); v != nil {
		if s, ok := v.(string); ok {
			if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
				c.Locals(LocUserID, id.String())
				return id, nil
			}
		}
	}
	// 3) fallback: id
	if v := c.Locals("id"); v != nil {
		if s, ok := v.(string); ok {
			if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
				c.Locals(LocUserID, id.String())
				return id, nil
			}
		}
	}
	// 4) JWT fallback: id/sub/user_id
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

// Admin/DKM only (legacy behavior) — jangan melebar ke teacher.
func GetMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocMasjidDKMIDs); err == nil {
		return id, nil
	}
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

// Prefer teacher scope: pakai teacher_records + active_masjid_id → fallbacks terurut.
func GetMasjidIDFromTokenPreferTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) teacher_records (baru)
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		if act, err2 := GetActiveMasjidIDFromToken(c); err2 == nil && act != uuid.Nil {
			for _, r := range recs {
				if r.MasjidID == act {
					return act, nil
				}
			}
		}
		return recs[0].MasjidID, nil
	}
	// 2) DKM/Admin (legacy)
	if id, err := GetMasjidIDFromToken(c); err == nil {
		return id, nil
	}
	// 3) masjid_ids (generic)
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil && len(ids) > 0 {
		return ids[0], nil
	}
	// 4) masjid_roles (ambil yang pertama)
	if ids, err := getMasjidIDsFromMasjidRoles(c); err == nil && len(ids) > 0 {
		return ids[0], nil
	}
	// 5) fallback admin
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

// Untuk kode lama yang mengira "TeacherMasjidID" adalah masjid_id tempat dia mengajar.
func GetTeacherMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		if act, err2 := GetActiveMasjidIDFromToken(c); err2 == nil && act != uuid.Nil {
			for _, r := range recs {
				if r.MasjidID == act {
					return act, nil
				}
			}
		}
		return recs[0].MasjidID, nil
	}
	// legacy
	return parseFirstUUIDFromLocals(c, LocMasjidTeacherIDs)
}

/* ============================================
   Multi-tenant getter
   ============================================ */

func GetMasjidIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	// 1) explicit masjid_ids
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil && len(ids) > 0 {
		return ids, nil
	}
	// 2) teacher_records
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		seen := map[uuid.UUID]struct{}{}
		out := make([]uuid.UUID, 0, len(recs))
		for _, r := range recs {
			if _, ok := seen[r.MasjidID]; !ok {
				seen[r.MasjidID] = struct{}{}
				out = append(out, r.MasjidID)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	// 3) student_records (BARU)
	if recs, err := parseStudentRecordsFromLocals(c); err == nil && len(recs) > 0 {
		seen := map[uuid.UUID]struct{}{}
		out := make([]uuid.UUID, 0, len(recs))
		for _, r := range recs {
			if _, ok := seen[r.MasjidID]; !ok {
				seen[r.MasjidID] = struct{}{}
				out = append(out, r.MasjidID)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	// 4) masjid_roles
	if ids, err := getMasjidIDsFromMasjidRoles(c); err == nil && len(ids) > 0 {
		return ids, nil
	}
	// 5) legacy grouped claims
	groups := []string{LocMasjidTeacherIDs, LocMasjidDKMIDs, LocMasjidAdminIDs, LocMasjidStudentIDs}
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
		if id, err := GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			return []uuid.UUID{id}, nil
		}
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	return out, nil
}

/* ============================================
   Role helpers (aware of masjid_roles, teacher_records, student_records)
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

func IsDKM(c *fiber.Ctx) bool {
	if strings.ToLower(GetRole(c)) == "dkm" {
		return true
	}
	// cek masjid_roles untuk role "dkm"
	if ids, err := getMasjidIDsFromMasjidRoles(c); err == nil && len(ids) > 0 {
		for _, id := range ids {
			if HasRoleInMasjid(c, id, "dkm") {
				return true
			}
		}
	}
	// legacy locals
	return HasUUIDClaim(c, LocMasjidDKMIDs) || HasUUIDClaim(c, LocMasjidAdminIDs)
}

func IsTeacher(c *fiber.Ctx) bool {
	// kalau ada teacher_records → teacher
	if recs, err := parseTeacherRecordsFromLocals(c); err == nil && len(recs) > 0 {
		return true
	}
	if strings.EqualFold(GetRole(c), "teacher") {
		return true
	}
	if HasGlobalRole(c, "teacher") {
		return true
	}
	// cek masjid_roles
	if ids, err := getMasjidIDsFromMasjidRoles(c); err == nil && len(ids) > 0 {
		for _, id := range ids {
			if HasRoleInMasjid(c, id, "teacher") {
				return true
			}
		}
	}
	return false
}

func IsStudent(c *fiber.Ctx) bool {
	// BARU: kalau ada student_records → student
	if recs, err := parseStudentRecordsFromLocals(c); err == nil && len(recs) > 0 {
		return true
	}
	if strings.EqualFold(GetRole(c), "student") {
		return true
	}
	if HasGlobalRole(c, "student") {
		return true
	}
	if ids, err := getMasjidIDsFromMasjidRoles(c); err == nil && len(ids) > 0 {
		for _, id := range ids {
			if HasRoleInMasjid(c, id, "student") {
				return true
			}
		}
	}
	return HasUUIDClaim(c, LocMasjidStudentIDs)
}

/* ============================================
   DB-role helpers (unchanged)
   ============================================ */

func EnsureGlobalRole(tx *gorm.DB, userID uuid.UUID, roleName string, assignedBy *uuid.UUID) error {
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") {
		return nil
	}
	if quickHasFunction(tx, "fn_grant_role") {
		var idStr string
		if assignedBy != nil {
			return tx.Raw(`
				SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, ?::uuid)::text
			`, userID, strings.ToLower(roleName), *assignedBy).Scan(&idStr).Error
		}
		return tx.Raw(`
			SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, NULL::uuid)::text
		`, userID, strings.ToLower(roleName)).Scan(&idStr).Error
	}

	var roleID string
	if err := tx.Raw(`SELECT role_id::text FROM roles WHERE LOWER(role_name)=LOWER(?) LIMIT 1`,
		roleName).Scan(&roleID).Error; err != nil {
		return err
	}
	if roleID == "" {
		if err := tx.Raw(`INSERT INTO roles(role_name) VALUES (LOWER(?)) RETURNING role_id::text`,
			roleName).Scan(&roleID).Error; err != nil {
			return err
		}
	}

	var exists bool
	if err := tx.Raw(`
		SELECT EXISTS(
		  SELECT 1 FROM user_roles
		  WHERE user_id=?::uuid AND role_id=?::uuid AND masjid_id IS NULL AND deleted_at IS NULL
		)
	`, userID, roleID).Scan(&exists).Error; err != nil {
		return err
	}
	if exists {
		return nil
	}

	if assignedBy != nil {
		return tx.Exec(`
			INSERT INTO user_roles(user_id, role_id, masjid_id, assigned_at, assigned_by)
			VALUES (?::uuid, ?::uuid, NULL, now(), ?::uuid)
		`, userID, roleID, *assignedBy).Error
	}
	return tx.Exec(`
		INSERT INTO user_roles(user_id, role_id, masjid_id, assigned_at)
		VALUES (?::uuid, ?::uuid, NULL, now())
	`, userID, roleID).Error
}

func GrantScopedRoleDKM(tx *gorm.DB, userID, masjidID uuid.UUID) error {
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") {
		return nil
	}
	if quickHasFunction(tx, "fn_grant_role") {
		var idStr string
		return tx.Raw(`
			SELECT fn_grant_role(?::uuid, 'dkm'::text, ?::uuid, ?::uuid)::text
		`, userID, masjidID, userID).Scan(&idStr).Error
	}

	var roleID string
	if err := tx.Raw(`SELECT role_id::text FROM roles WHERE LOWER(role_name)='dkm' LIMIT 1`).
		Scan(&roleID).Error; err != nil {
		return err
	}

	if roleID == "" {
		if err := tx.Raw(`INSERT INTO roles(role_name) VALUES ('dkm') RETURNING role_id::text`).
			Scan(&roleID).Error; err != nil {
			return err
		}
	}

	var exists bool
	if err := tx.Raw(`
		SELECT EXISTS(
		  SELECT 1 FROM user_roles
		  WHERE user_id=?::uuid AND role_id=?::uuid AND masjid_id=?::uuid AND deleted_at IS NULL
		)
	`, userID, roleID, masjidID).Scan(&exists).Error; err != nil {
		return err
	}
	if exists {
		return nil
	}

	return tx.Exec(`
		INSERT INTO user_roles(user_id, role_id, masjid_id, assigned_at, assigned_by)
		VALUES (?::uuid, ?::uuid, ?::uuid, now(), ?::uuid)
	`, userID, roleID, masjidID, userID).Error
}

/* ============================================
   Misc
   ============================================ */

func GetActiveMasjidIDIfSingle(rc RolesClaim) *string {
	if len(rc.MasjidRoles) == 1 && rc.MasjidRoles[0].MasjidID != uuid.Nil {
		id := rc.MasjidRoles[0].MasjidID.String()
		return &id
	}
	return nil
}
