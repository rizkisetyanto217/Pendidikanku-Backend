package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Ambil user_id dari c.Locals("user_id")
// Return 401 kalau belum login, 400 kalau formatnya tidak valid.
func GetUserIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals("user_id")
	if v == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "User belum login")
	}

	switch t := v.(type) {
	case uuid.UUID:
		if t == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "User belum login")
		}
		return t, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "User belum login")
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "User ID pada token tidak valid")
		}
		return id, nil
	case []byte:
		s := strings.TrimSpace(string(t))
		if s == "" {
			return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "User belum login")
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "User ID pada token tidak valid")
		}
		return id, nil
	default:
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "User ID pada token tidak valid")
	}
}

