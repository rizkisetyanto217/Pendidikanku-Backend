// package: internals/helpers/helper.go  (atau sesuai path "package helper")

package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	LocRole   = "role"
	LocUserID = "user_id"

	LocMasjidIDs        = "masjid_ids"
	LocMasjidAdminIDs   = "masjid_admin_ids"
	LocMasjidDKMIDs     = "masjid_dkm_ids"
	LocMasjidTeacherIDs = "masjid_teacher_ids"
	LocMasjidStudentIDs = "masjid_student_ids"

	LocRolesGlobal = "roles_global"
	LocMasjidRoles = "masjid_roles"
	LocIsOwner     = "is_owner"
)


type MasjidRolesEntry struct {
	MasjidID uuid.UUID `json:"masjid_id"`
	Roles    []string  `json:"roles"`
}

type RolesClaim struct {
	RolesGlobal []string           `json:"roles_global"`
	MasjidRoles []MasjidRolesEntry `json:"masjid_roles"`
}


// ---------------------------
// Quick schema helpers (LOCAL)
// ---------------------------
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

// ---------------------------
// Locals parsers
// ---------------------------

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

// ---------------------------
// Claims helpers
// ---------------------------

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

func HasUUIDClaim(c *fiber.Ctx, key string) bool {
	ids, err := parseUUIDSliceFromLocals(c, key)
	return err == nil && len(ids) > 0
}

// ---------------------------
// Single-tenant getters
// ---------------------------

func GetUserIDFromToken(c *fiber.Ctx) (uuid.UUID, error) { return parseFirstUUIDFromLocals(c, LocUserID) }

func GetMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocMasjidDKMIDs); err == nil {
		return id, nil
	}
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

func GetDKMMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocMasjidDKMIDs); err == nil {
		return id, nil
	}
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

func GetTeacherMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return parseFirstUUIDFromLocals(c, LocMasjidTeacherIDs)
}

func GetMasjidIDFromTokenPreferTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocMasjidTeacherIDs); err == nil {
		return id, nil
	}
	if id, err := GetDKMMasjidIDFromToken(c); err == nil {
		return id, nil
	}
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil && len(ids) > 0 {
		return ids[0], nil
	}
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

// ---------------------------
// Multi-tenant getter
// ---------------------------

func GetMasjidIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil {
		return ids, nil
	}
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

// ---------------------------
// Role helpers
// ---------------------------

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
	role := strings.ToLower(GetRole(c))
	if role == "dkm" {
		return true
	}
	return HasUUIDClaim(c, LocMasjidDKMIDs) || HasUUIDClaim(c, LocMasjidAdminIDs)
}

func IsTeacher(c *fiber.Ctx) bool {
	if strings.EqualFold(GetRole(c), "teacher") {
		return true
	}
	return HasGlobalRole(c, "teacher")
}

func IsStudent(c *fiber.Ctx) bool {
	if strings.EqualFold(GetRole(c), "student") {
		return true
	}
	return HasGlobalRole(c, "student")
}

func IsAdmin(c *fiber.Ctx) bool {
	if IsOwner(c) || IsDKM(c) {
		return true
	}
	role := strings.ToLower(GetRole(c))
	if role == "admin" {
		return true
	}
	return HasUUIDClaim(c, LocMasjidAdminIDs) || HasUUIDClaim(c, LocMasjidDKMIDs)
}



// pastikan import:
// "github.com/google/uuid"
// "gorm.io/gorm"
// "strings"

func EnsureGlobalRole(tx *gorm.DB, userID uuid.UUID, roleName string, assignedBy *uuid.UUID) error {
	// kalau table belum ada, skip
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") {
		return nil
	}

	// Prefer function
	if quickHasFunction(tx, "fn_grant_role") {
		var idStr string
		// cast ke text supaya scan ke string mulus
		if assignedBy != nil {
			return tx.Raw(`
				SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, ?::uuid)::text
			`, userID, strings.ToLower(roleName), *assignedBy).
				Scan(&idStr).Error
		}
		return tx.Raw(`
			SELECT fn_grant_role(?::uuid, ?::text, NULL::uuid, NULL::uuid)::text
		`, userID, strings.ToLower(roleName)).
			Scan(&idStr).Error
	}

	// --- fallback manual idempotent ---
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
	// kalau table belum ada, skip
	if !quickHasTable(tx, "roles") || !quickHasTable(tx, "user_roles") {
		return nil
	}

	// Prefer function
	if quickHasFunction(tx, "fn_grant_role") {
		var idStr string
		return tx.Raw(`
			SELECT fn_grant_role(?::uuid, 'dkm'::text, ?::uuid, ?::uuid)::text
		`, userID, masjidID, userID).Scan(&idStr).Error
	}

	// --- fallback manual idempotent ---
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



// getActiveMasjidIDIfSingle mengembalikan masjid_id jika user hanya punya satu masjid.
func GetActiveMasjidIDIfSingle(rc RolesClaim) *string {
	if len(rc.MasjidRoles) == 1 && rc.MasjidRoles[0].MasjidID != uuid.Nil {
		id := rc.MasjidRoles[0].MasjidID.String()
		return &id
	}
	return nil
}
