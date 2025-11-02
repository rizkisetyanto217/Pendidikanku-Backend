// file: internals/helpers/auth/school_context_resolver.go
package helper

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchoolContext struct {
	ID   uuid.UUID
	Slug string
}

var (
	ErrSchoolContextMissing   = fiber.NewError(fiber.StatusBadRequest, "School context tidak ditemukan. Sertakan :school_id di path atau header X-Active-School-ID / query ?school_id.")
	ErrSchoolContextAmbiguous = fiber.NewError(fiber.StatusConflict, "School context ambigu untuk user multi-tenant. Sertakan identitas school eksplisit.")
	ErrSchoolContextForbidden = fiber.NewError(fiber.StatusForbidden, "Anda tidak memiliki akses ke school ini atau bukan DKM.")
)

/*
	============================
	  Resolver slug → ID (via DB)

============================
*/
func GetSchoolIDBySlug(c *fiber.Ctx, slug string) (uuid.UUID, error) {
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
		SELECT school_id
		FROM schools
		WHERE LOWER(school_slug) = LOWER(?) AND school_deleted_at IS NULL
		LIMIT 1
	`, strings.TrimSpace(slug)).Scan(&id).Error; err != nil {
		return uuid.Nil, err
	}
	if id == uuid.Nil {
		return uuid.Nil, gorm.ErrRecordNotFound
	}
	return id, nil
}

/*
	==========================================
	  Resolve context: path → header → cookie → query → host → token

==========================================
*/
func ResolveSchoolContext(c *fiber.Ctx) (SchoolContext, error) {
	// 1) path
	if id := strings.TrimSpace(c.Params("school_id")); id != "" {
		if uid, err := uuid.Parse(id); err == nil {
			return SchoolContext{ID: uid}, nil
		}
	}
	if slug := strings.TrimSpace(c.Params("school_slug")); slug != "" {
		return SchoolContext{Slug: slug}, nil
	}

	// 2) header
	if h := strings.TrimSpace(c.Get("X-Active-School-ID")); h != "" {
		if uid, err := uuid.Parse(h); err == nil {
			return SchoolContext{ID: uid}, nil
		}
	}
	if h := strings.TrimSpace(c.Get("X-Active-School-Slug")); h != "" {
		return SchoolContext{Slug: h}, nil
	}

	// 3) cookie (opsional, membantu saat test di Postman)
	if v := strings.TrimSpace(c.Cookies("X-Active-School-ID")); v != "" {
		if uid, err := uuid.Parse(v); err == nil {
			return SchoolContext{ID: uid}, nil
		}
	}
	if v := strings.TrimSpace(c.Cookies("X-Active-School-Slug")); v != "" {
		return SchoolContext{Slug: v}, nil
	}

	// 4) query
	q := c.Context().QueryArgs()
	if b := q.Peek("school_id"); len(b) > 0 {
		if uid, err := uuid.Parse(string(b)); err == nil {
			return SchoolContext{ID: uid}, nil
		}
	}
	if b := q.Peek("school_slug"); len(b) > 0 {
		if s, _ := url.QueryUnescape(string(b)); s != "" {
			return SchoolContext{Slug: s}, nil
		}
	}

	// 5) host/subdomain
	host := c.Hostname()
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		sub := parts[0]
		if sub != "www" && sub != "app" && sub != "" {
			return SchoolContext{Slug: sub}, nil
		}
	}

	// 6) fallback token (biasanya hanya single-tenant)
	if id, err := GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		return SchoolContext{ID: id}, nil
	}

	return SchoolContext{}, ErrSchoolContextMissing
}

func EnsureSchoolAccessDKM(c *fiber.Ctx, mc SchoolContext) (uuid.UUID, error) {
	var schoolID uuid.UUID

	// slug → id
	if mc.ID == uuid.Nil && mc.Slug != "" {
		id, er := GetSchoolIDBySlug(c, mc.Slug)
		if er != nil {
			return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		schoolID = id
	} else {
		schoolID = mc.ID
	}

	// ✅ 1) Role check DULU (role DKM/Admin di school ini ⇒ otomatis member)
	if err := EnsureDKMSchool(c, schoolID); err == nil {
		return schoolID, nil
	}

	// ❓ 2) Kalau bukan DKM/Admin, cek apakah memang bukan member
	if !UserHasSchool(c, schoolID) {
		return uuid.Nil, fiber.NewError(fiber.StatusForbidden, "Anda tidak terdaftar pada school ini (membership).")
	}

	// ❌ 3) Member, tapi bukan DKM/Admin
	return uuid.Nil, fiber.NewError(fiber.StatusForbidden, "Anda bukan DKM untuk school ini (role).")
}

func UserHasSchool(c *fiber.Ctx, id uuid.UUID) bool {
	if id == uuid.Nil {
		return false
	}
	// ✅ Anggap member jika ada entry school_roles untuk school ini
	if entries, err := parseSchoolRoles(c); err == nil {
		for _, e := range entries {
			if e.SchoolID == id {
				return true
			}
		}
	}
	// Fallback ke daftar school_ids (jika ada)
	if ids, _ := GetSchoolIDsFromToken(c); len(ids) > 0 {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
	}
	return false
}
