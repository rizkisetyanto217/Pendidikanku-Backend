package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)


func GetMasjidIDFromToken(c *fiber.Ctx) (uuid.UUID, error) {
	// Validasi user login
	if c.Locals("user_id") == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "User belum login")
	}
	ids, ok := c.Locals("masjid_admin_ids").([]string)
	if !ok || len(ids) == 0 {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak ditemukan di token")
	}
	return uuid.Parse(ids[0])
}

// simple slug normalizer (optional, sesuaikan dgn kebutuhanmu)
func NormalizeSlug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "--", "-")
	return s
}