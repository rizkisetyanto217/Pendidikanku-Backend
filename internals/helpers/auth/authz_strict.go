package helper

import (
	"strings"

	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type mrEntry struct {
	MasjidID uuid.UUID
	Roles    []string
}

func parseMasjidRolesStrict(c *fiber.Ctx) ([]mrEntry, error) {
	v := c.Locals(LocMasjidRoles) // HARUS dari middleware verifikasi JWT
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" tidak ditemukan di token")
	}
	out := make([]mrEntry, 0)
	switch arr := v.(type) {
	case []map[string]any:
		for _, m := range arr {
			var e mrEntry
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
	case []interface{}:
		for _, it := range arr {
			if m, ok := it.(map[string]any); ok {
				var e mrEntry
				if s, ok := m["masjid_id"].(string); ok {
					if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
						e.MasjidID = id
					}
				}
				if rr, ok := m["roles"].([]interface{}); ok {
					for _, it2 := range rr {
						if rs, ok := it2.(string); ok {
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
	default:
		return nil, fiber.NewError(fiber.StatusBadRequest, LocMasjidRoles+" format tidak didukung")
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocMasjidRoles+" kosong/invalid")
	}
	return out, nil
}

func hasRoleInMasjidStrict(c *fiber.Ctx, masjidID uuid.UUID, role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" || masjidID == uuid.Nil {
		return false
	}
	entries, err := parseMasjidRolesStrict(c)
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

func isMasjidPresentStrict(c *fiber.Ctx, masjidID uuid.UUID) bool {
	if masjidID == uuid.Nil {
		return false
	}
	entries, err := parseMasjidRolesStrict(c)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.MasjidID == masjidID {
			return true
		}
	}
	return false
}

func isPrivilegedStrict(c *fiber.Ctx) bool {
	// Owner/superadmin boleh bypass
	if v := c.Locals(LocRolesGlobal); v != nil {
		if arr, ok := v.([]string); ok {
			for _, r := range arr {
				if strings.EqualFold(r, "superadmin") || strings.EqualFold(r, "owner") {
					return true
				}
			}
		}
	}
	if s, _ := c.Locals("role").(string); strings.EqualFold(s, "owner") {
		return true
	}
	return false
}

// ===== Strict wrappers (tidak ada legacy fallback) =====
func EnsureStaffMasjidStrict(c *fiber.Ctx, masjidID uuid.UUID) error {
	if masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id wajib")
	}
	if isPrivilegedStrict(c) {
		return nil
	}
	if !isMasjidPresentStrict(c, masjidID) {
		return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Masjid ini tidak ada dalam token Anda")
	}
	if hasRoleInMasjidStrict(c, masjidID, "teacher") ||
		hasRoleInMasjidStrict(c, masjidID, "dkm") ||
		hasRoleInMasjidStrict(c, masjidID, "admin") ||
		hasRoleInMasjidStrict(c, masjidID, "bendahara") {
		return nil
	}
	return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Hanya guru/DKM yang diizinkan")
}

func EnsureTeacherMasjidStrict(c *fiber.Ctx, masjidID uuid.UUID) error {
	if masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id wajib")
	}
	if isPrivilegedStrict(c) {
		return nil
	}
	if !isMasjidPresentStrict(c, masjidID) {
		return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Masjid ini tidak ada dalam token Anda")
	}
	if hasRoleInMasjidStrict(c, masjidID, "teacher") {
		return nil
	}
	return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Hanya guru yang diizinkan")
}

func EnsureDKMMasjidStrict(c *fiber.Ctx, masjidID uuid.UUID) error {
	if masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id wajib")
	}
	if isPrivilegedStrict(c) {
		return nil
	}
	if !isMasjidPresentStrict(c, masjidID) {
		return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Masjid ini tidak ada dalam token Anda")
	}
	if hasRoleInMasjidStrict(c, masjidID, "dkm") || hasRoleInMasjidStrict(c, masjidID, "admin") {
		return nil
	}
	return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Hanya DKM yang diizinkan")
}
