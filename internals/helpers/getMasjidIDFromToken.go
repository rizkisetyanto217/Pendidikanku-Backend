package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals("masjid_admin_ids")
	if v == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	switch t := v.(type) {
	case []string:
		if len(t) == 0 || strings.TrimSpace(t[0]) == "" {
			return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Masjid ID kosong di token")
		}
		return uuid.Parse(t[0])
	case []interface{}:
		if len(t) == 0 {
			return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "Masjid ID kosong di token")
		}
		if s, ok := t[0].(string); ok {
			return uuid.Parse(strings.TrimSpace(s))
		}
	case interface{}:
		// Kalau di Locals disimpan langsung sebagai string
		if s, ok := t.(string); ok {
			return uuid.Parse(strings.TrimSpace(s))
		}
	}

	return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Format Masjid ID tidak valid di token")
}
