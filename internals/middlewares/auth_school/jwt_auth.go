package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	helperAuth "madinahsalam_backend/internals/helpers/auth"
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

		// === HYDRATE LOCALS YANG DIHARAPKAN HELPER BARU ===

		// roles_global → LocRolesGlobal
		if v, ok := claims["roles_global"]; ok {
			c.Locals(helperAuth.LocRolesGlobal, v)
		}

		// school_roles → LocSchoolRoles
		if v, ok := claims["school_roles"]; ok {
			c.Locals(helperAuth.LocSchoolRoles, v)
		}

		// is_owner → LocIsOwner (kalau ada)
		if v, ok := claims["is_owner"]; ok {
			switch t := v.(type) {
			case bool:
				c.Locals(helperAuth.LocIsOwner, t)
			case string:
				s := strings.ToLower(strings.TrimSpace(t))
				if s == "true" || s == "1" || s == "yes" {
					c.Locals(helperAuth.LocIsOwner, true)
				}
			}
		}

		// school_id (single session) → LocActiveSchoolID + LocSchoolID
		if sid := strClaim(claims, "school_id"); sid != "" {
			c.Locals(helperAuth.LocActiveSchoolID, sid)
			c.Locals(helperAuth.LocSchoolID, sid)
		}

		// teacher_id → LocTeacherID
		if tid := strClaim(claims, "teacher_id"); tid != "" {
			c.Locals(helperAuth.LocTeacherID, tid)
		}

		// student_id → LocStudentID
		if sid := strClaim(claims, "student_id"); sid != "" {
			c.Locals(helperAuth.LocStudentID, sid)
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
					// boleh langsung 401 kalau mau strict:
					// return fiber.NewError(fiber.StatusUnauthorized, "user_id tidak valid")
					// sekarang kita biarkan helper yang menolak nanti.
				}
			}
		}

		// === (OPSIONAL) Build & set roles_claim struct untuk pemakaian lain ===
		rc := helperAuth.RolesClaim{
			RolesGlobal: readStringSlice(claims["roles_global"]),
			SchoolRoles: make([]helperAuth.SchoolRolesEntry, 0),
		}
		if v, ok := claims["school_roles"]; ok {
			switch arr := v.(type) {
			case []any:
				for _, it := range arr {
					m, ok := it.(map[string]any)
					if !ok {
						continue
					}
					var mid uuid.UUID
					if s, ok := m["school_id"].(string); ok {
						if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
							mid = id
						}
					}
					roles := readStringSlice(m["roles"])
					rc.SchoolRoles = append(rc.SchoolRoles, helperAuth.SchoolRolesEntry{
						SchoolID: mid,
						Roles:    roles,
					})
				}
			}
		}
		c.Locals("roles_claim", rc)

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

// EnsureLegacyRoleLocal mengisi c.Locals("role") (legacy) dari klaim modern.
// Prioritas: school_roles (dkm > admin > teacher > student > user)
// lalu roles_global, terakhir fallback "user".
func EnsureLegacyRoleLocal(c *fiber.Ctx) {
	// kalau sudah ada "role" dan tidak kosong, jangan diutak-atik
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

	// 1) dari school_roles
	if mr := c.Locals(helperAuth.LocSchoolRoles); mr != nil {
		switch t := mr.(type) {
		case []helperAuth.SchoolRolesEntry:
			for _, e := range t {
				if r := pick(e.Roles, "dkm", "admin", "teacher", "student", "user"); r != "" {
					c.Locals("role", r)
					return
				}
			}
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

	// 3) fallback: user
	c.Locals("role", "user")
}
