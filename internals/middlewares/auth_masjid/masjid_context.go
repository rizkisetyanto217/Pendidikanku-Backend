// file: internals/middlewares/features/masjid_context.go
package middleware

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

//? Digunakan untuk mengambil masjid_id & masjid_slug dari token JWT

/* ==========================
   Consts & Types
========================== */

const logPrefix = "[MASJID_CTX]"

type AppMode int

const (
	ModeDKM AppMode = iota
	ModeTeacher
	ModePublic // GET publik diperbolehkan (opsional)
)

type MasjidContextOpts struct {
	DB                *gorm.DB
	AppMode           AppMode
	AllowPublicNoAuth bool   // hanya untuk ModePublic & GET
	CentralRootDomain string // contoh: "masjidku.id"  → {slug}.masjidku.id
}

type masjidRow struct {
	ID     uuid.UUID `gorm:"column:masjid_id"`
	Slug   *string   `gorm:"column:masjid_slug"`
	Domain *string   `gorm:"column:masjid_domain"`
}

// RolesClaim minimal (menyesuaikan isi token)
type MasjidRolesEntry struct {
	MasjidID uuid.UUID `json:"masjid_id"`
	Roles    []string  `json:"roles"`
}
type RolesClaim struct {
	RolesGlobal []string           `json:"roles_global"`
	MasjidRoles []MasjidRolesEntry `json:"masjid_roles"`
}

/* ==========================
   Helpers (host/strings)
========================== */

func normalizeHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	if h == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	}
	h = strings.TrimPrefix(h, "www.")
	return h
}

func isLocalHostOrIP(h string) bool {
	if h == "localhost" || h == "localhost.localdomain" {
		return true
	}
	return net.ParseIP(h) != nil
}

func hasRoleForApp(mode AppMode, roles []string) bool {
	for _, r := range roles {
		lr := strings.ToLower(strings.TrimSpace(r))
		switch mode {
		case ModeDKM:
			if lr == "dkm" || lr == "owner" {
				return true
			}
		case ModeTeacher:
			if lr == "teacher" || lr == "owner" {
				return true
			}
		case ModePublic:
			return true
		}
	}
	return false
}

/* ==========================
   Helpers (claims)
========================== */

// Ambil RolesClaim dari locals, kalau belum ada inflate dari jwt_claims.
func inflateRolesClaimFromJWT(c *fiber.Ctx) RolesClaim {
	if any := c.Locals("roles_claim"); any != nil {
		if rc, ok := any.(RolesClaim); ok {
			return rc
		}
	}

	var out RolesClaim
	if any := c.Locals("jwt_claims"); any != nil {
		if m, ok := any.(jwt.MapClaims); ok {
			// roles_global
			if rg, ok := m["roles_global"].([]interface{}); ok {
				for _, v := range rg {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						out.RolesGlobal = append(out.RolesGlobal, s)
					}
				}
			}
			// masjid_roles
			if mr, ok := m["masjid_roles"].([]interface{}); ok {
				for _, it := range mr {
					if mm, ok := it.(map[string]interface{}); ok {
						var e MasjidRolesEntry
						if s, ok := mm["masjid_id"].(string); ok {
							if id, err := uuid.Parse(s); err == nil {
								e.MasjidID = id
							}
						}
						if rr, ok := mm["roles"].([]interface{}); ok {
							for _, r := range rr {
								if rs, ok := r.(string); ok && strings.TrimSpace(rs) != "" {
                                    e.Roles = append(e.Roles, rs)
								}
							}
						}
						if e.MasjidID != uuid.Nil {
							out.MasjidRoles = append(out.MasjidRoles, e)
						}
					}
				}
			}
		}
	}
	return out
}

// Auto-pick masjid_id dari: jwt_claims.active_masjid_id → legacy masjid_ids[0] → locals → single-tenant RC.
func pickMasjidIDAuto(c *fiber.Ctx, rc RolesClaim) string {
	if any := c.Locals("jwt_claims"); any != nil {
		if m, ok := any.(jwt.MapClaims); ok {
			if v, ok := m["active_masjid_id"].(string); ok && strings.TrimSpace(v) != "" {
				return v
			}
			if arr, ok := m["masjid_ids"].([]interface{}); ok && len(arr) > 0 {
				if s, ok := arr[0].(string); ok && strings.TrimSpace(s) != "" {
					return s
				}
			}
		}
	}
	if v, _ := c.Locals("active_masjid_id").(uuid.UUID); v != uuid.Nil {
		return v.String()
	}
	if s, _ := c.Locals("active_masjid_id").(string); strings.TrimSpace(s) != "" {
		return s
	}
	if len(rc.MasjidRoles) == 1 && rc.MasjidRoles[0].MasjidID != uuid.Nil {
		return rc.MasjidRoles[0].MasjidID.String()
	}
	return ""
}

func summarizeRC(rc RolesClaim) string {
	sb := strings.Builder{}
	for i, mr := range rc.MasjidRoles {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(mr.MasjidID.String())
		sb.WriteString(":")
		sb.WriteString(strings.Join(mr.Roles, ","))
	}
	return sb.String()
}

/* ==========================
   Middleware
========================== */

func MasjidContext(o MasjidContextOpts) fiber.Handler {
	if o.DB == nil {
		panic("MasjidContext: DB wajib diisi")
	}

	return func(c *fiber.Ctx) error {
		t0 := time.Now()
		method := strings.ToUpper(c.Method())

		log.Printf("%s 🔥 %s %s", logPrefix, method, c.OriginalURL())

		// 0) RolesClaim
		rc := inflateRolesClaimFromJWT(c)
		log.Printf("%s roles_claim from locals: [%s]", logPrefix, summarizeRC(rc))

		// 1) Sumber eksplisit
		byID := strings.TrimSpace(c.Get("X-Masjid-ID"))
		bySlug := strings.TrimSpace(c.Get("X-Masjid-Slug"))
		paramSlug := strings.TrimSpace(c.Params("slug"))
		if bySlug == "" && paramSlug != "" {
			bySlug = paramSlug
		}
		log.Printf("%s explicit headers: X-Masjid-ID=%q X-Masjid-Slug=%q (paramSlug=%q)",
			logPrefix, byID, bySlug, paramSlug)

		// 2) Auto-pick dari token/RC
		if byID == "" && bySlug == "" {
			if picked := pickMasjidIDAuto(c, rc); picked != "" {
				log.Printf("%s auto-pick from token/RC → %s", logPrefix, picked)
				byID = picked
			}
		}

		// 3) Host (diabaikan saat dev)
		host := normalizeHost(c.Hostname())
		useHost := (byID == "" && bySlug == "" && host != "" && !isLocalHostOrIP(host))
		log.Printf("%s host=%q central=%q useHost=%v", logPrefix, host, o.CentralRootDomain, useHost)

		// 4) Resolve ke DB
		var row masjidRow
		var err error

		switch {
		case byID != "":
			log.Printf("%s resolve by ID=%s", logPrefix, byID)
			id, perr := uuid.Parse(byID)
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "X-Masjid-ID invalid")
			}
			err = o.DB.Raw(`
				SELECT masjid_id, masjid_slug, masjid_domain
				FROM masjids
				WHERE masjid_id = ? AND masjid_deleted_at IS NULL
				LIMIT 1
			`, id).Scan(&row).Error

		case bySlug != "":
			log.Printf("%s resolve by Slug=%s", logPrefix, bySlug)
			err = o.DB.Raw(`
				SELECT masjid_id, masjid_slug, masjid_domain
				FROM masjids
				WHERE LOWER(masjid_slug) = LOWER(?) AND masjid_deleted_at IS NULL
				LIMIT 1
			`, bySlug).Scan(&row).Error

		case useHost:
			log.Printf("%s resolve by Host=%s", logPrefix, host)
			if o.CentralRootDomain != "" && strings.HasSuffix(host, "."+o.CentralRootDomain) {
				parts := strings.Split(host, ".")
				if len(parts) >= 3 {
					sub := parts[0]
					log.Printf("%s host looks like subdomain → slug=%s", logPrefix, sub)
					err = o.DB.Raw(`
						SELECT masjid_id, masjid_slug, masjid_domain
						FROM masjids
						WHERE LOWER(masjid_slug) = LOWER(?) AND masjid_deleted_at IS NULL
						LIMIT 1
					`, sub).Scan(&row).Error
				}
			}
			if row.ID == uuid.Nil {
				log.Printf("%s try host as custom domain", logPrefix)
				err = o.DB.Raw(`
					SELECT masjid_id, masjid_slug, masjid_domain
					FROM masjids
					WHERE LOWER(masjid_domain) = LOWER(?) AND masjid_deleted_at IS NULL
					LIMIT 1
				`, host).Scan(&row).Error
			}

		default:
			if o.AppMode == ModePublic && o.AllowPublicNoAuth && method == fiber.MethodGet {
				log.Printf("%s public GET without context → pass-through", logPrefix)
				return c.Next()
			}
		}

		if err != nil {
			log.Printf("%s SQL error: %v", logPrefix, err)
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		if row.ID == uuid.Nil && !(o.AppMode == ModePublic && o.AllowPublicNoAuth && method == fiber.MethodGet) {
			log.Printf("%s ❌ masjid not resolved", logPrefix)
			return fiber.NewError(fiber.StatusBadRequest, "Masjid tidak ditemukan dari konteks")
		}

		// 5) Validasi role
		allowed := false
		if o.AppMode == ModePublic && o.AllowPublicNoAuth && method == fiber.MethodGet && row.ID == uuid.Nil {
			allowed = true
		} else {
			for _, mr := range rc.MasjidRoles {
				if mr.MasjidID == row.ID && hasRoleForApp(o.AppMode, mr.Roles) {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			log.Printf("%s ❌ forbidden: roles=%s needMode=%d resolvedMasjid=%s",
				logPrefix, summarizeRC(rc), o.AppMode, row.ID)
			return fiber.NewError(fiber.StatusForbidden, "Akses ke masjid ini tidak diizinkan")
		}

		// 6) Set locals (baru)
		c.Locals("active_masjid_id", row.ID)
		if row.Slug != nil {
			c.Locals("active_masjid_slug", *row.Slug)
		}
		if row.Domain != nil {
			c.Locals("active_masjid_domain", *row.Domain)
		}

		// 6b) Compat locals untuk middleware/helper lama (role & *_ids)
		isOwner, isDKM, isTeacher := false, false, false
		for _, mr := range rc.MasjidRoles {
			if mr.MasjidID != row.ID {
				continue
			}
			for _, r := range mr.Roles {
				lr := strings.ToLower(strings.TrimSpace(r))
				switch lr {
				case "owner":
					isOwner = true
				case "dkm":
					isDKM = true
				case "teacher":
					isTeacher = true
				}
			}
		}
		// set "role" legacy sesuai Mode
		switch o.AppMode {
		case ModeDKM:
			if isOwner {
				c.Locals("role", "owner")
			} else if isDKM {
				c.Locals("role", "dkm")
			}
		case ModeTeacher:
			if isOwner {
				c.Locals("role", "owner")
			} else if isTeacher {
				c.Locals("role", "teacher")
			}
		}
		// set daftar *_ids (string slice)
		if isOwner || isDKM {
			c.Locals("masjid_dkm_ids", []string{row.ID.String()})
			c.Locals("masjid_admin_ids", []string{row.ID.String()})
		}
		if isTeacher {
			c.Locals("masjid_teacher_ids", []string{row.ID.String()})
		}
		if isOwner {
			c.Locals("is_owner", true)
		}

		log.Printf("%s ✅ OK masjid_id=%s slug=%v domain=%v dur=%s",
			logPrefix, row.ID, row.Slug, row.Domain, time.Since(t0))
		log.Printf("%s compat locals set: role=%v dkm_ids=%v teacher_ids=%v",
			logPrefix, c.Locals("role"), c.Locals("masjid_dkm_ids"), c.Locals("masjid_teacher_ids"))

		return c.Next()
	}
}
