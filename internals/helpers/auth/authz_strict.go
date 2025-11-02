package helper

import (
	"strings"

	helper "schoolku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type mrEntry struct {
	SchoolID uuid.UUID
	Roles    []string
}

func parseSchoolRolesStrict(c *fiber.Ctx) ([]mrEntry, error) {
	v := c.Locals(LocSchoolRoles) // HARUS dari middleware verifikasi JWT
	if v == nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocSchoolRoles+" tidak ditemukan di token")
	}
	out := make([]mrEntry, 0)
	switch arr := v.(type) {
	case []map[string]any:
		for _, m := range arr {
			var e mrEntry
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
			if e.SchoolID != uuid.Nil && len(e.Roles) > 0 {
				out = append(out, e)
			}
		}
	case []interface{}:
		for _, it := range arr {
			if m, ok := it.(map[string]any); ok {
				var e mrEntry
				if s, ok := m["school_id"].(string); ok {
					if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
						e.SchoolID = id
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
				if e.SchoolID != uuid.Nil && len(e.Roles) > 0 {
					out = append(out, e)
				}
			}
		}
	default:
		return nil, fiber.NewError(fiber.StatusBadRequest, LocSchoolRoles+" format tidak didukung")
	}
	if len(out) == 0 {
		return nil, fiber.NewError(fiber.StatusUnauthorized, LocSchoolRoles+" kosong/invalid")
	}
	return out, nil
}

func hasRoleInSchoolStrict(c *fiber.Ctx, schoolID uuid.UUID, role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" || schoolID == uuid.Nil {
		return false
	}
	entries, err := parseSchoolRolesStrict(c)
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

func isSchoolPresentStrict(c *fiber.Ctx, schoolID uuid.UUID) bool {
	if schoolID == uuid.Nil {
		return false
	}
	entries, err := parseSchoolRolesStrict(c)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.SchoolID == schoolID {
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
func EnsureStaffSchoolStrict(c *fiber.Ctx, schoolID uuid.UUID) error {
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id wajib")
	}
	if isPrivilegedStrict(c) {
		return nil
	}
	if !isSchoolPresentStrict(c, schoolID) {
		return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] School ini tidak ada dalam token Anda")
	}
	if hasRoleInSchoolStrict(c, schoolID, "teacher") ||
		hasRoleInSchoolStrict(c, schoolID, "dkm") ||
		hasRoleInSchoolStrict(c, schoolID, "admin") ||
		hasRoleInSchoolStrict(c, schoolID, "bendahara") {
		return nil
	}
	return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Hanya guru/DKM yang diizinkan")
}

func EnsureTeacherSchoolStrict(c *fiber.Ctx, schoolID uuid.UUID) error {
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id wajib")
	}
	if isPrivilegedStrict(c) {
		return nil
	}
	if !isSchoolPresentStrict(c, schoolID) {
		return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] School ini tidak ada dalam token Anda")
	}
	if hasRoleInSchoolStrict(c, schoolID, "teacher") {
		return nil
	}
	return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Hanya guru yang diizinkan")
}

func EnsureDKMSchoolStrict(c *fiber.Ctx, schoolID uuid.UUID) error {
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id wajib")
	}
	if isPrivilegedStrict(c) {
		return nil
	}
	if !isSchoolPresentStrict(c, schoolID) {
		return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] School ini tidak ada dalam token Anda")
	}
	if hasRoleInSchoolStrict(c, schoolID, "dkm") || hasRoleInSchoolStrict(c, schoolID, "admin") {
		return nil
	}
	return helper.JsonError(c, fiber.StatusForbidden, "[AUTHZ strict] Hanya DKM yang diizinkan")
}
