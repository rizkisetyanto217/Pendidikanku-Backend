package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

//
// ========== Keys (hindari typo) ==========
//

const (
	LocRole             = "role"
	LocUserID           = "user_id" // ✅ tambahan: dipakai banyak controller
	LocMasjidIDs        = "masjid_ids"         // union semua peran (opsional)
	LocMasjidAdminIDs   = "masjid_admin_ids"
	LocMasjidTeacherIDs = "masjid_teacher_ids"
	LocMasjidStudentIDs = "masjid_student_ids"
)

//
// ========== Utilities ==========
//

// normalizeLocalsToStrings mengekstrak daftar string dari tipe umum Locals:
// []string, []interface{}(string), string, uuid.UUID, []uuid.UUID.
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
			if s, ok := it.(string); ok {
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, s)
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

// parseFirstUUIDFromLocals mengembalikan UUID pertama dari Locals[key].
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

// parseUUIDSliceFromLocals mengembalikan slice UUID dari Locals[key].
// Error 400 jika ada item yang bukan UUID, 401 jika key ada tapi kosong.
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

//
// ========== Claim helpers generik ==========
//

// GetRole mengembalikan role (lowercased). Kosong jika tidak ada.
func GetRole(c *fiber.Ctx) string {
	if v := c.Locals(LocRole); v != nil {
		if s, ok := v.(string); ok {
			return strings.ToLower(strings.TrimSpace(s))
		}
	}
	return ""
}

// HasUUIDClaim true jika Locals[key] berisi minimal 1 UUID valid.
func HasUUIDClaim(c *fiber.Ctx, key string) bool {
	ids, err := parseUUIDSliceFromLocals(c, key)
	return err == nil && len(ids) > 0
}

//
// ========== Single-tenant getters (Create/Update/Delete) ==========
//

// ✅ User ID (sering dibutuhkan)
func GetUserIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return parseFirstUUIDFromLocals(c, LocUserID)
}

// Admin-only
func GetMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

// Teacher-only
func GetTeacherMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return parseFirstUUIDFromLocals(c, LocMasjidTeacherIDs)
}

// Prefer TEACHER -> UNION masjid_ids -> ADMIN
func GetMasjidIDFromTokenPreferTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := parseFirstUUIDFromLocals(c, LocMasjidTeacherIDs); err == nil {
		return id, nil
	}
	if id, err := parseFirstUUIDFromLocals(c, LocMasjidIDs); err == nil {
		return id, nil
	}
	return parseFirstUUIDFromLocals(c, LocMasjidAdminIDs)
}

//
// ========== Multi-tenant getter (List/Read) ==========
//

// Ambil semua masjid yang berhak diakses user dari token.
func GetMasjidIDsFromToken(c *fiber.Ctx) ([]uuid.UUID, error) {
	// 1) union langsung (jika ada)
	if ids, err := parseUUIDSliceFromLocals(c, LocMasjidIDs); err == nil {
		return ids, nil
	}

	// 2) fallback gabungan role-role
	groups := []string{LocMasjidTeacherIDs, LocMasjidAdminIDs, LocMasjidStudentIDs}
	seen := map[uuid.UUID]struct{}{}
	out := make([]uuid.UUID, 0, 4)

	var anyFound bool
	for _, key := range groups {
		v := c.Locals(key)
		if v == nil {
			continue
		}
		anyFound = true
		raw := normalizeLocalsToStrings(v)
		for _, s := range raw {
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
		// terakhir: fallback single preferTeacher
		if id, err := GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			return []uuid.UUID{id}, nil
		}
		return nil, fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	return out, nil
}

//
// ========== Role helpers ==========
//

// IsAdmin true jika role=admin ATAU owner ATAU punya masjid_admin_ids (scoped).
func IsAdmin(c *fiber.Ctx) bool {
	role := strings.ToLower(GetRole(c)) // pastikan middleware set Locals("role")
	if role == "admin" || role == "owner" {
		return true
	}
	// fallback: punya hak admin scoped ke masjid
	return HasUUIDClaim(c, LocMasjidAdminIDs)
}

func IsOwner(c *fiber.Ctx) bool   { return strings.ToLower(GetRole(c)) == "owner" }
func IsTeacher(c *fiber.Ctx) bool { return strings.EqualFold(GetRole(c), "teacher") }
func IsStudent(c *fiber.Ctx) bool { return strings.EqualFold(GetRole(c), "student") }
