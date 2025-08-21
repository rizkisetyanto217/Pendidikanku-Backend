// internals/middlewares/auth/claims_utils.go
package auth

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ======== Extractors ======== */
// ======== Extractors (perbarui ini) ========

func extractBearerToken(c *fiber.Ctx) (string, error) {
	// 1) Ambil dari Authorization header atau fallback cookie
	auth := strings.TrimSpace(c.Get("Authorization"))
	if auth == "" {
		if cookieTok := c.Cookies("access_token"); cookieTok != "" {
			auth = "Bearer " + cookieTok
			log.Println("[DEBUG] Authorization dari Cookie")
		}
	}
	if auth == "" {
		return "", fmt.Errorf("unauthorized - No token provided")
	}

	// 2) Robust split: toleransi spasi ganda & case-insensitive
	fields := strings.Fields(auth) // pecah berdasarkan whitespace berturut
	if len(fields) < 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", fmt.Errorf("unauthorized - Invalid token format")
	}
	tok := fields[1]

	// 3) Sanitasi: buang kutip di kiri/kanan & spasi
	tok = strings.TrimSpace(tok)
	tok = strings.Trim(tok, "\"'")

	if tok == "" {
		return "", fmt.Errorf("unauthorized - Empty token")
	}
	return tok, nil
}

func validateTokenExpiry(claims jwt.MapClaims, skew time.Duration) error {
	expVal, ok := claims["exp"]
	if !ok {
		return fmt.Errorf("token has no exp")
	}

	var expUnix int64
	switch t := expVal.(type) {
	case float64:
		expUnix = int64(t)
	case int64:
		expUnix = t
	case string:
		if n, err := parseInt64(strings.TrimSpace(t)); err == nil {
			expUnix = n
		} else {
			return fmt.Errorf("invalid exp format")
		}
	default:
		// coba best-effort untuk tipe numeric lain (mis. json.Number via interface{})
		if s := fmt.Sprintf("%v", t); s != "" {
			if n, err := parseInt64(s); err == nil {
				expUnix = n
			} else {
				return fmt.Errorf("invalid exp type")
			}
		} else {
			return fmt.Errorf("invalid exp type")
		}
	}

	now := time.Now().UTC()
	expTime := time.Unix(expUnix, 0).UTC()
	if now.After(expTime.Add(skew)) {
		return fmt.Errorf("token expired at %v", expTime)
	}
	return nil
}


func extractUserID(claims jwt.MapClaims) (uuid.UUID, error) {
	idRaw, ok := claims["id"]
	if !ok {
		return uuid.Nil, fmt.Errorf("no user id")
	}
	switch v := idRaw.(type) {
	case string:
		return uuid.Parse(strings.TrimSpace(v))
	default:
		return uuid.Nil, fmt.Errorf("invalid user id type")
	}
}

func ensureUserActive(db *gorm.DB, userID uuid.UUID) error {
	var user struct {
		IsActive bool
	}
	if err := db.Table("users").Select("is_active").Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if !user.IsActive {
		return errors.New("user inactive")
	}
	return nil
}

/* ======== Store claims to Locals ======== */

func storeBasicClaimsToLocals(c *fiber.Ctx, claims jwt.MapClaims) {
	if role, ok := claims["role"].(string); ok {
		c.Locals("userRole", role)
	}
	if userName, ok := claims["user_name"].(string); ok {
		c.Locals("user_name", userName)
	}
}

func storeMasjidIDsToLocals(c *fiber.Ctx, claims jwt.MapClaims) {
	adminIDs := toStringSlice(claims["masjid_admin_ids"])
	teacherIDs := toStringSlice(claims["masjid_teacher_ids"])
	unionIDs := toStringSlice(claims["masjid_ids"])

	if len(adminIDs) > 0 {
		c.Locals("masjid_admin_ids", adminIDs)
	}
	if len(teacherIDs) > 0 {
		c.Locals("masjid_teacher_ids", teacherIDs)
	}
	if len(unionIDs) > 0 {
		c.Locals("masjid_ids", unionIDs)
	}

	// pilih aktif: teacher → admin → union
	active := ""
	switch {
	case len(teacherIDs) > 0:
		active = teacherIDs[0]
	case len(adminIDs) > 0:
		active = adminIDs[0]
	case len(unionIDs) > 0:
		active = unionIDs[0]
	}

	if active != "" {
		c.Locals("masjid_id", active)
		log.Println("[SUCCESS] Active masjid_id stored:", active)
	} else {
		log.Println("[INFO] Tidak ada masjid_id terdeteksi di token")
	}
}

/* ======== Helpers ======== */

func toStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, it := range t {
			if s, ok := it.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func parseInt64(s string) (int64, error) {
	// kecilkan depedensi: simple parser untuk angka desimal
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("non-digit")
		}
		n = n*10 + int64(ch-'0')
	}
	return n, nil
}
