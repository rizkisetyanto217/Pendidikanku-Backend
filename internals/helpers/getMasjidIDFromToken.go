package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// --- util kecil biar gak duplikasi parsing ---
func firstUUIDFromLocals(c *fiber.Ctx, key string) (uuid.UUID, error) {
	v := c.Locals(key)
	if v == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" tidak ditemukan di token")
	}

	switch t := v.(type) {
	case []string:
		if len(t) == 0 || strings.TrimSpace(t[0]) == "" {
			return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token")
		}
		return uuid.Parse(strings.TrimSpace(t[0]))
	case []interface{}:
		if len(t) == 0 {
			return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token")
		}
		if s, ok := t[0].(string); ok {
			return uuid.Parse(strings.TrimSpace(s))
		}
	case interface{}:
		if s, ok := t.(string); ok {
			if strings.TrimSpace(s) == "" {
				return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, key+" kosong di token")
			}
			return uuid.Parse(strings.TrimSpace(s))
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Format "+key+" tidak valid di token")
}

// === Existing (ADMIN) ===
func GetMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return firstUUIDFromLocals(c, "masjid_admin_ids")
}

// === TEACHER ===
func GetTeacherMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	return firstUUIDFromLocals(c, "masjid_teacher_ids")
}

// === (Opsional) Prefer TEACHER lalu fallback ke ADMIN/UNION ===
func GetMasjidIDFromTokenPreferTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	if id, err := firstUUIDFromLocals(c, "masjid_teacher_ids"); err == nil {
		return id, nil
	}
	// kalau middleware kamu juga set "masjid_ids" (union), ini bisa dijadikan fallback kedua
	if id, err := firstUUIDFromLocals(c, "masjid_ids"); err == nil {
		return id, nil
	}
	return firstUUIDFromLocals(c, "masjid_admin_ids")
}
