// file: internals/helpers/dbtime/dbtime.go
package dbtime

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Nama locals mengikuti yg di-set di middleware AuthJWT
const (
	LocSchoolTimezone = "school_timezone" // string, misal "Asia/Jakarta"
	LocSchoolLoc      = "school_loc"      // *time.Location
)

// Ambil *time.Location berdasarkan token:
// 1) Prioritas: c.Locals("school_loc") yang diisi middleware
// 2) Kalau belum ada: coba baca "school_timezone" (string) lalu LoadLocation
// 3) Fallback: Asia/Jakarta
// 4) Fallback terakhir: time.UTC
func GetSchoolLocation(c *fiber.Ctx) *time.Location {
	if c == nil {
		return time.UTC
	}

	// 1) Kalau middleware sudah set "school_loc"
	if v := c.Locals(LocSchoolLoc); v != nil {
		if loc, ok := v.(*time.Location); ok && loc != nil {
			return loc
		}
	}

	// 2) Kalau cuma punya "school_timezone" (string)
	if v := c.Locals(LocSchoolTimezone); v != nil {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			s = strings.TrimSpace(s)
			if loc, err := time.LoadLocation(s); err == nil {
				// cache ke locals biar next call lebih murah
				c.Locals(LocSchoolLoc, loc)
				return loc
			}
		}
	}

	// 3) Fallback ke Asia/Jakarta
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		c.Locals(LocSchoolLoc, loc)
		return loc
	}

	// 4) Fallback terakhir
	return time.UTC
}

// ToSchoolTime mengonversi waktu (biasanya dari DB = UTC) ke timezone sekolah.
// Kalau t.IsZero() → dikembalikan apa adanya.
func ToSchoolTime(c *fiber.Ctx, t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	loc := GetSchoolLocation(c)
	if loc == nil {
		return t
	}
	return t.In(loc)
}

// Versi pointer, biar gampang dipakai di DTO yg pakai *time.Time
func ToSchoolTimePtr(c *fiber.Ctx, t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := ToSchoolTime(c, *t)
	return &v
}

// Helper kecil untuk "sekarang di timezone sekolah"
func NowInSchool(c *fiber.Ctx) time.Time {
	return time.Now().In(GetSchoolLocation(c))
}

// Signature ini yang dipakai di controller:
// now, err := dbtime.GetDBTime(c)
func GetDBTime(c *fiber.Ctx) (time.Time, error) {
	// Kalau suatu saat mau beneran ambil dari DB (SELECT NOW())
	// di sini aja yang diganti, signature-nya tetap.
	return NowInSchool(c), nil
}
