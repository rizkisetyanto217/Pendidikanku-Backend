package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"

	"github.com/google/uuid"
)


type AuthJWTOpts struct {
	Secret              string
	BlacklistChecker    func(rawToken string) (bool, error) // return true if blacklisted
	AllowCookieFallback bool                                // pakai cookie access_token jika tidak ada Bearer
}

func AuthJWT(o AuthJWTOpts) fiber.Handler {
	secret := strings.TrimSpace(o.Secret)
	if secret == "" {
		panic("AuthJWT: Secret wajib diisi")
	}

	return func(c *fiber.Ctx) error {
		// Ambil token: Authorization: Bearer xxx  (atau cookie jika diizinkan)
		raw := ""
		authz := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			raw = strings.TrimSpace(authz[7:])
		} else if o.AllowCookieFallback {
			raw = strings.TrimSpace(c.Cookies("access_token"))
		}
		if raw == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		}

		// Cek blacklist (opsional)
		if o.BlacklistChecker != nil {
			if black, err := o.BlacklistChecker(raw); err == nil && black {
				return fiber.NewError(fiber.StatusUnauthorized, "Token revoked")
			}
		}

		// Parse
		tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !tok.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		}

		claims, ok := tok.Claims.(jwt.MapClaims)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token claims")
		}

		// Simpan raw claims
		c.Locals("jwt_claims", claims)

		// Build RolesClaim (roles_global + masjid_roles)
		var rc RolesClaim
		// roles_global
		if rg, ok := claims["roles_global"].([]any); ok {
			for _, v := range rg {
				if s, ok := v.(string); ok && s != "" {
					rc.RolesGlobal = append(rc.RolesGlobal, s)
				}
			}
		}
		// masjid_roles
		if mrRaw, ok := claims["masjid_roles"].([]any); ok {
			for _, it := range mrRaw {
				if m, ok := it.(map[string]any); ok {
					var entry MasjidRolesEntry
					if idStr, ok := m["masjid_id"].(string); ok {
						if uid, err := uuid.Parse(idStr); err == nil {
							entry.MasjidID = uid
						}
					}
					if roles, ok := m["roles"].([]any); ok {
						for _, r := range roles {
							if s, ok := r.(string); ok && s != "" {
								entry.Roles = append(entry.Roles, s)
							}
						}
					}
					if entry.MasjidID != uuid.Nil {
						rc.MasjidRoles = append(rc.MasjidRoles, entry)
					}
				}
			}
		}

		c.Locals("roles_claim", rc)
		return c.Next()
	}
}
