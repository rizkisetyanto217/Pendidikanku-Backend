// file: internals/helpers/auth/masjid_context_resolver.go
package helper

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidContext struct {
	ID   uuid.UUID
	Slug string
}

var (
	ErrMasjidContextMissing   = fiber.NewError(fiber.StatusBadRequest, "Masjid context tidak ditemukan. Sertakan :masjid_id di path atau header X-Active-Masjid-ID / query ?masjid_id.")
	ErrMasjidContextAmbiguous = fiber.NewError(fiber.StatusConflict, "Masjid context ambigu untuk user multi-tenant. Sertakan identitas masjid eksplisit.")
	ErrMasjidContextForbidden = fiber.NewError(fiber.StatusForbidden, "Anda tidak memiliki akses ke masjid ini atau bukan DKM.")
)

/* ============================
   Resolver slug → ID (via DB)
============================ */
func GetMasjidIDBySlug(c *fiber.Ctx, slug string) (uuid.UUID, error) {
	dbAny := c.Locals("DB")
	if dbAny == nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "DB context tidak tersedia")
	}
	db, ok := dbAny.(*gorm.DB)
	if !ok {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "DB context invalid")
	}

	var id uuid.UUID
	// case-insensitive & only alive
	if err := db.Raw(`
		SELECT masjid_id
		FROM masjids
		WHERE LOWER(masjid_slug) = LOWER(?) AND masjid_deleted_at IS NULL
		LIMIT 1
	`, strings.TrimSpace(slug)).Scan(&id).Error; err != nil {
		return uuid.Nil, err
	}
	if id == uuid.Nil {
		return uuid.Nil, gorm.ErrRecordNotFound
	}
	return id, nil
}

/* ==========================================
   Resolve context: path → header → cookie → query → host → token
========================================== */
func ResolveMasjidContext(c *fiber.Ctx) (MasjidContext, error) {
	// 1) path
	if id := strings.TrimSpace(c.Params("masjid_id")); id != "" {
		if uid, err := uuid.Parse(id); err == nil {
			return MasjidContext{ID: uid}, nil
		}
	}
	if slug := strings.TrimSpace(c.Params("masjid_slug")); slug != "" {
		return MasjidContext{Slug: slug}, nil
	}

	// 2) header
	if h := strings.TrimSpace(c.Get("X-Active-Masjid-ID")); h != "" {
		if uid, err := uuid.Parse(h); err == nil {
			return MasjidContext{ID: uid}, nil
		}
	}
	if h := strings.TrimSpace(c.Get("X-Active-Masjid-Slug")); h != "" {
		return MasjidContext{Slug: h}, nil
	}

	// 3) cookie (opsional, membantu saat test di Postman)
	if v := strings.TrimSpace(c.Cookies("X-Active-Masjid-ID")); v != "" {
		if uid, err := uuid.Parse(v); err == nil {
			return MasjidContext{ID: uid}, nil
		}
	}
	if v := strings.TrimSpace(c.Cookies("X-Active-Masjid-Slug")); v != "" {
		return MasjidContext{Slug: v}, nil
	}

	// 4) query
	q := c.Context().QueryArgs()
	if b := q.Peek("masjid_id"); len(b) > 0 {
		if uid, err := uuid.Parse(string(b)); err == nil {
			return MasjidContext{ID: uid}, nil
		}
	}
	if b := q.Peek("masjid_slug"); len(b) > 0 {
		if s, _ := url.QueryUnescape(string(b)); s != "" {
			return MasjidContext{Slug: s}, nil
		}
	}

	// 5) host/subdomain
	host := c.Hostname()
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		sub := parts[0]
		if sub != "www" && sub != "app" && sub != "" {
			return MasjidContext{Slug: sub}, nil
		}
	}

	// 6) fallback token (biasanya hanya single-tenant)
	if id, err := GetActiveMasjidID(c); err == nil && id != uuid.Nil {
		return MasjidContext{ID: id}, nil
	}

	return MasjidContext{}, ErrMasjidContextMissing
}


func EnsureMasjidAccessDKM(c *fiber.Ctx, mc MasjidContext) (uuid.UUID, error) {
    var masjidID uuid.UUID

    // slug → id
    if mc.ID == uuid.Nil && mc.Slug != "" {
        id, er := GetMasjidIDBySlug(c, mc.Slug)
        if er != nil {
            return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
        }
        masjidID = id
    } else {
        masjidID = mc.ID
    }

    // ✅ 1) Role check DULU (role DKM/Admin di masjid ini ⇒ otomatis member)
    if err := EnsureDKMMasjid(c, masjidID); err == nil {
        return masjidID, nil
    }

    // ❓ 2) Kalau bukan DKM/Admin, cek apakah memang bukan member
    if !UserHasMasjid(c, masjidID) {
        return uuid.Nil, fiber.NewError(fiber.StatusForbidden, "Anda tidak terdaftar pada masjid ini (membership).")
    }

    // ❌ 3) Member, tapi bukan DKM/Admin
    return uuid.Nil, fiber.NewError(fiber.StatusForbidden, "Anda bukan DKM untuk masjid ini (role).")
}

func UserHasMasjid(c *fiber.Ctx, id uuid.UUID) bool {
    if id == uuid.Nil {
        return false
    }
    // ✅ Anggap member jika ada entry masjid_roles untuk masjid ini
    if entries, err := parseMasjidRoles(c); err == nil {
        for _, e := range entries {
            if e.MasjidID == id {
                return true
            }
        }
    }
    // Fallback ke daftar masjid_ids (jika ada)
    if ids, _ := GetMasjidIDsFromToken(c); len(ids) > 0 {
        for _, v := range ids {
            if v == id {
                return true
            }
        }
    }
    return false
}