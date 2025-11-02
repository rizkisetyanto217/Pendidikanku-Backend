// file: internals/middlewares/features/school_context.go
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

//? Digunakan untuk mengambil school_id & school_slug dari token JWT

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

type SchoolContextOpts struct {
	DB                *gorm.DB
	AppMode           AppMode
	AllowPublicNoAuth bool   // hanya untuk ModePublic & GET
	CentralRootDomain string // contoh: "schoolku.id"  → {slug}.schoolku.id
}

type schoolRow struct {
	ID     uuid.UUID `gorm:"column:school_id"`
	Slug   *string   `gorm:"column:school_slug"`
	Domain *string   `gorm:"column:school_domain"`
}

// RolesClaim minimal (menyesuaikan isi token)
type SchoolRolesEntry struct {
	SchoolID uuid.UUID `json:"school_id"`
	Roles    []string  `json:"roles"`
}
type RolesClaim struct {
	RolesGlobal []string           `json:"roles_global"`
	SchoolRoles []SchoolRolesEntry `json:"school_roles"`
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
			// school_roles
			if mr, ok := m["school_roles"].([]interface{}); ok {
				for _, it := range mr {
					if mm, ok := it.(map[string]interface{}); ok {
						var e SchoolRolesEntry
						if s, ok := mm["school_id"].(string); ok {
							if id, err := uuid.Parse(s); err == nil {
								e.SchoolID = id
							}
						}
						if rr, ok := mm["roles"].([]interface{}); ok {
							for _, r := range rr {
								if rs, ok := r.(string); ok && strings.TrimSpace(rs) != "" {
									e.Roles = append(e.Roles, rs)
								}
							}
						}
						if e.SchoolID != uuid.Nil {
							out.SchoolRoles = append(out.SchoolRoles, e)
						}
					}
				}
			}
		}
	}
	return out
}

// Auto-pick school_id dari: jwt_claims.active_school_id → legacy school_ids[0] → locals → single-tenant RC.
func pickSchoolIDAuto(c *fiber.Ctx, rc RolesClaim) string {
	if any := c.Locals("jwt_claims"); any != nil {
		if m, ok := any.(jwt.MapClaims); ok {
			if v, ok := m["active_school_id"].(string); ok && strings.TrimSpace(v) != "" {
				return v
			}
			if arr, ok := m["school_ids"].([]interface{}); ok && len(arr) > 0 {
				if s, ok := arr[0].(string); ok && strings.TrimSpace(s) != "" {
					return s
				}
			}
		}
	}
	if v, _ := c.Locals("active_school_id").(uuid.UUID); v != uuid.Nil {
		return v.String()
	}
	if s, _ := c.Locals("active_school_id").(string); strings.TrimSpace(s) != "" {
		return s
	}
	if len(rc.SchoolRoles) == 1 && rc.SchoolRoles[0].SchoolID != uuid.Nil {
		return rc.SchoolRoles[0].SchoolID.String()
	}
	return ""
}

func summarizeRC(rc RolesClaim) string {
	sb := strings.Builder{}
	for i, mr := range rc.SchoolRoles {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(mr.SchoolID.String())
		sb.WriteString(":")
		sb.WriteString(strings.Join(mr.Roles, ","))
	}
	return sb.String()
}

/* ==========================
   Middleware
========================== */

func SchoolContext(o SchoolContextOpts) fiber.Handler {
	if o.DB == nil {
		panic("SchoolContext: DB wajib diisi")
	}

	return func(c *fiber.Ctx) error {
		t0 := time.Now()
		method := strings.ToUpper(c.Method())

		log.Printf("%s 🔥 %s %s", logPrefix, method, c.OriginalURL())

		// 0) RolesClaim
		rc := inflateRolesClaimFromJWT(c)
		log.Printf("%s roles_claim from locals: [%s]", logPrefix, summarizeRC(rc))

		// 1) Sumber eksplisit
		byID := strings.TrimSpace(c.Get("X-School-ID"))
		bySlug := strings.TrimSpace(c.Get("X-School-Slug"))
		paramSlug := strings.TrimSpace(c.Params("slug"))
		if bySlug == "" && paramSlug != "" {
			bySlug = paramSlug
		}
		log.Printf("%s explicit headers: X-School-ID=%q X-School-Slug=%q (paramSlug=%q)",
			logPrefix, byID, bySlug, paramSlug)

		// 2) Auto-pick dari token/RC
		if byID == "" && bySlug == "" {
			if picked := pickSchoolIDAuto(c, rc); picked != "" {
				log.Printf("%s auto-pick from token/RC → %s", logPrefix, picked)
				byID = picked
			}
		}

		// 3) Host (diabaikan saat dev)
		host := normalizeHost(c.Hostname())
		useHost := (byID == "" && bySlug == "" && host != "" && !isLocalHostOrIP(host))
		log.Printf("%s host=%q central=%q useHost=%v", logPrefix, host, o.CentralRootDomain, useHost)

		// 4) Resolve ke DB
		var row schoolRow
		var err error

		switch {
		case byID != "":
			log.Printf("%s resolve by ID=%s", logPrefix, byID)
			id, perr := uuid.Parse(byID)
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "X-School-ID invalid")
			}
			err = o.DB.Raw(`
				SELECT school_id, school_slug, school_domain
				FROM schools
				WHERE school_id = ? AND school_deleted_at IS NULL
				LIMIT 1
			`, id).Scan(&row).Error

		case bySlug != "":
			log.Printf("%s resolve by Slug=%s", logPrefix, bySlug)
			err = o.DB.Raw(`
				SELECT school_id, school_slug, school_domain
				FROM schools
				WHERE LOWER(school_slug) = LOWER(?) AND school_deleted_at IS NULL
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
						SELECT school_id, school_slug, school_domain
						FROM schools
						WHERE LOWER(school_slug) = LOWER(?) AND school_deleted_at IS NULL
						LIMIT 1
					`, sub).Scan(&row).Error
				}
			}
			if row.ID == uuid.Nil {
				log.Printf("%s try host as custom domain", logPrefix)
				err = o.DB.Raw(`
					SELECT school_id, school_slug, school_domain
					FROM schools
					WHERE LOWER(school_domain) = LOWER(?) AND school_deleted_at IS NULL
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
			log.Printf("%s ❌ school not resolved", logPrefix)
			return fiber.NewError(fiber.StatusBadRequest, "School tidak ditemukan dari konteks")
		}

		// 5) Validasi role
		allowed := false
		if o.AppMode == ModePublic && o.AllowPublicNoAuth && method == fiber.MethodGet && row.ID == uuid.Nil {
			allowed = true
		} else {
			for _, mr := range rc.SchoolRoles {
				if mr.SchoolID == row.ID && hasRoleForApp(o.AppMode, mr.Roles) {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			log.Printf("%s ❌ forbidden: roles=%s needMode=%d resolvedSchool=%s",
				logPrefix, summarizeRC(rc), o.AppMode, row.ID)
			return fiber.NewError(fiber.StatusForbidden, "Akses ke school ini tidak diizinkan")
		}

		// 6) Set locals (baru)
		c.Locals("active_school_id", row.ID)
		if row.Slug != nil {
			c.Locals("active_school_slug", *row.Slug)
		}
		if row.Domain != nil {
			c.Locals("active_school_domain", *row.Domain)
		}

		// 6b) Compat locals untuk middleware/helper lama (role & *_ids)
		isOwner, isDKM, isTeacher := false, false, false
		for _, mr := range rc.SchoolRoles {
			if mr.SchoolID != row.ID {
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
			c.Locals("school_dkm_ids", []string{row.ID.String()})
			c.Locals("school_admin_ids", []string{row.ID.String()})
		}
		if isTeacher {
			c.Locals("school_teacher_ids", []string{row.ID.String()})
		}
		if isOwner {
			c.Locals("is_owner", true)
		}

		log.Printf("%s ✅ OK school_id=%s slug=%v domain=%v dur=%s",
			logPrefix, row.ID, row.Slug, row.Domain, time.Since(t0))
		log.Printf("%s compat locals set: role=%v dkm_ids=%v teacher_ids=%v",
			logPrefix, c.Locals("role"), c.Locals("school_dkm_ids"), c.Locals("school_teacher_ids"))

		return c.Next()
	}
}
