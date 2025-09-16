package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	// <-- pakai konstanta & helper kamu
	helperAuth "masjidku_backend/internals/helpers/auth"
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
		// 1) Ambil token: Authorization: Bearer xxx (atau cookie jika diizinkan)
		raw := ""
		if authz := strings.TrimSpace(c.Get(fiber.HeaderAuthorization)); strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			raw = strings.TrimSpace(authz[7:])
		} else if o.AllowCookieFallback {
			raw = strings.TrimSpace(c.Cookies("access_token"))
		}
		if raw == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		}

		// 2) Cek blacklist (opsional)
		if o.BlacklistChecker != nil {
			if black, err := o.BlacklistChecker(raw); err == nil && black {
				return fiber.NewError(fiber.StatusUnauthorized, "Token revoked")
			}
		}

		// 3) Parse + verifikasi algoritma
		tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !tok.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		}

		claims, ok := tok.Claims.(jwt.MapClaims)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token claims")
		}

		// Simpan raw claims (opsional)
		c.Locals("jwt_claims", claims)

		// === HYDRATE LOCALS YANG DIHARAPKAN HELPER ===

		// roles_global
		if v, ok := claims["roles_global"]; ok {
			c.Locals(helperAuth.LocRolesGlobal, v) // helperAuth.GetRolesGlobal bisa handling []any atau []string
		}

		// masjid_roles
		if v, ok := claims["masjid_roles"]; ok {
			c.Locals(helperAuth.LocMasjidRoles, v) // parseMasjidRoles bisa handling []map[string]any / []any
		}

		// teacher_records
		if v, ok := claims["teacher_records"]; ok {
			c.Locals(helperAuth.LocTeacherRecords, v) // parseTeacherRecordsFromLocals bisa handling bentuk generic
		}

		// active_masjid_id (harus string untuk helperAuth)
		if s, ok := claims["active_masjid_id"].(string); ok && strings.TrimSpace(s) != "" {
			c.Locals(helperAuth.LocActiveMasjidID, strings.TrimSpace(s))
		}

		// user_id: ambil id/sub/user_id dalam urutan preferensi
		switch {
		case strClaim(claims, "id") != "":
			c.Locals(helperAuth.LocUserID, strClaim(claims, "id"))
		case strClaim(claims, "sub") != "":
			c.Locals(helperAuth.LocUserID, strClaim(claims, "sub"))
		case strClaim(claims, "user_id") != "":
			c.Locals(helperAuth.LocUserID, strClaim(claims, "user_id"))
		}

		// (opsional) Validasi cepat bahwa user_id berbentuk UUID, biar fail-fast
		if v := c.Locals(helperAuth.LocUserID); v != nil {
			if s, ok := v.(string); ok {
				if _, err := uuid.Parse(strings.TrimSpace(s)); err != nil {
					// biarkan helperAuth yang menolak nanti; atau kamu bisa langsung return 401 di sini
				}
			}
		}

		// === Build & set roles_claim (struct) ===
		rc := helperAuth.RolesClaim{
			RolesGlobal: readStringSlice(claims["roles_global"]),
			MasjidRoles: make([]helperAuth.MasjidRolesEntry, 0),
		}
		if v, ok := claims["masjid_roles"]; ok {
			switch arr := v.(type) {
			case []any:
				for _, it := range arr {
					m, ok := it.(map[string]any)
					if !ok {
						continue
					}
					var mid uuid.UUID
					if s, ok := m["masjid_id"].(string); ok {
						if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
							mid = id
						}
					}
					roles := readStringSlice(m["roles"])
					rc.MasjidRoles = append(rc.MasjidRoles, helperAuth.MasjidRolesEntry{
						MasjidID: mid,
						Roles:    roles,
					})
				}
			}
		}
		c.Locals("roles_claim", rc) // ✅ penting untuk IsOwnerGlobal & middleware lain

		// Turunkan "role" legacy supaya guard lama tidak error "Role not found"
		EnsureLegacyRoleLocal(c)

		return c.Next()
	}
}

// util kecil untuk ambil string claim
func strClaim(m jwt.MapClaims, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}


// letakkan di file yang sama (package middleware), di bawah kode AuthJWT

// EnsureLegacyRoleLocal mengisi c.Locals("role") (legacy) dari klaim modern.
// Prioritas: masjid_roles (dkm > admin > teacher > student > user)
// lalu roles_global, lalu teacher_records, terakhir fallback "user".
func EnsureLegacyRoleLocal(c *fiber.Ctx) {
	// jika sudah ada role legacy, biarkan
	if v := c.Locals("role"); v != nil {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return
		}
	}

	pick := func(list []string, wanted ...string) string {
		if len(list) == 0 {
			return ""
		}
		has := map[string]struct{}{}
		for _, r := range list {
			r = strings.ToLower(strings.TrimSpace(r))
			if r != "" {
				has[r] = struct{}{}
			}
		}
		for _, w := range wanted {
			if _, ok := has[w]; ok {
				return w
			}
		}
		return ""
	}

	// 1) dari masjid_roles
	if mr := c.Locals(helperAuth.LocMasjidRoles); mr != nil {
		switch t := mr.(type) {
		case []map[string]any:
			for _, m := range t {
				roles := readStringSlice(m["roles"])
				if r := pick(roles, "dkm", "admin", "teacher", "student", "user"); r != "" {
					c.Locals("role", r)
					return
				}
			}
		case []any:
			for _, it := range t {
				if m, ok := it.(map[string]any); ok {
					roles := readStringSlice(m["roles"])
					if r := pick(roles, "dkm", "admin", "teacher", "student", "user"); r != "" {
						c.Locals("role", r)
						return
					}
				}
			}
		case []helperAuth.MasjidRolesEntry:
			for _, e := range t {
				if r := pick(e.Roles, "dkm", "admin", "teacher", "student", "user"); r != "" {
					c.Locals("role", r)
					return
				}
			}
		}
	}

	// 2) dari roles_global
	if rg := c.Locals(helperAuth.LocRolesGlobal); rg != nil {
		roles := readStringSlice(rg)
		if r := pick(roles, "owner", "dkm", "admin", "teacher", "student", "user"); r != "" {
			c.Locals("role", r)
			return
		}
	}

	// 3) jika ada teacher_records → set "teacher"
	if tr := c.Locals(helperAuth.LocTeacherRecords); tr != nil {
		switch t := tr.(type) {
		case []any:
			if len(t) > 0 {
				c.Locals("role", "teacher")
				return
			}
		case []map[string]any:
			if len(t) > 0 {
				c.Locals("role", "teacher")
				return
			}
		case []helperAuth.TeacherRecordEntry:
			if len(t) > 0 {
				c.Locals("role", "teacher")
				return
			}
		}
	}

	// 4) fallback
	c.Locals("role", "user")
}

// util: ubah nilai interface{} → []string (robust untuk []string atau []any)
func readStringSlice(v any) []string {
	out := make([]string, 0)
	switch t := v.(type) {
	case []string:
		for _, s := range t {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
	case []any:
		for _, it := range t {
			if s, ok := it.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

