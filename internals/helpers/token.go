// helpers/token.go (atau gabung ke file helper kamu yang sudah ada)
package helper

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Simpan raw JWT di Locals dari middleware (opsional, tapi enak buat reuse)
const LocRawToken = "raw_token"

// GetRawAccessToken mengembalikan access token dari:
// 1) cookie "access_token"
// 2) Locals("raw_token") yang diset middleware
// 3) Authorization header "Bearer <token>"
func GetRawAccessToken(c *fiber.Ctx) string {
	// 1) Cookie
	if v := strings.TrimSpace(c.Cookies("access_token")); v != "" {
		return v
	}
	// 2) Locals (diisi middleware sesudah verifikasi header/cookie)
	if v, ok := c.Locals(LocRawToken).(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	// 3) Authorization: Bearer <token>
	const p = "Bearer "
	auth := c.Get("Authorization")
	if len(auth) > len(p) && strings.HasPrefix(auth, p) {
		return strings.TrimSpace(auth[len(p):])
	}
	return ""
}

// (Opsional) Ambil refresh token dari cookie
func GetRefreshTokenFromCookie(c *fiber.Ctx) string {
	return strings.TrimSpace(c.Cookies("refresh_token"))
}

// (Opsional) Set raw token ke Locals dari middleware auth
func SetRawAccessToken(c *fiber.Ctx, raw string) {
	if strings.TrimSpace(raw) != "" {
		c.Locals(LocRawToken, strings.TrimSpace(raw))
	}
}

// (Opsional) CSRF check bila request membawa cookie (double-submit token):
// header: X-CSRF-Token harus sama dengan cookie: csrf_token
func CheckCSRFCookieHeader(c *fiber.Ctx) error {
	csrfCookie := strings.TrimSpace(c.Cookies("csrf_token"))
	if csrfCookie == "" {
		return fiber.NewError(fiber.StatusForbidden, "CSRF token missing (cookie)")
	}
	csrfHeader := strings.TrimSpace(c.Get("X-CSRF-Token"))
	if csrfHeader == "" {
		return fiber.NewError(fiber.StatusForbidden, "CSRF token missing (header)")
	}
	if csrfCookie != csrfHeader {
		return fiber.NewError(fiber.StatusForbidden, "CSRF token mismatch")
	}
	return nil
}
