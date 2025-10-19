package helper

import (
	"masjidku_backend/internals/constants"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetUserUUID(c *fiber.Ctx) uuid.UUID {
	// 🟢 Default: Guest user
	userUUID := constants.DummyUserID

	// 🔍 Coba dari Locals
	if userIDRaw := c.Locals("user_id"); userIDRaw != nil {
		if userIDStr, ok := userIDRaw.(string); ok {
			if parsed, err := uuid.Parse(userIDStr); err == nil {
				return parsed
			}
		}
	}

	// 🔍 Coba dari header fallback (optional)
	if header := c.Get("X-User-Id"); header != "" {
		if parsed, err := uuid.Parse(header); err == nil {
			return parsed
		}
	}

	return userUUID
}

func ParseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(c.Params(name)))
}
