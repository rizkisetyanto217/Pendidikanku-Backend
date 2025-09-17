// internals/middlewares/auth/claims_utils.go
// package auth

// import (
// 	"errors"
// 	"fmt"
// 	"log"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/golang-jwt/jwt/v4"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"

// 	helper "masjidku_backend/internals/helpers/auth"
// )

// /* =======================
//    Extractors
//    ======================= */

// // Ambil bearer token dari Authorization header, fallback cookie "access_token".
// func extractBearerToken(c *fiber.Ctx) (string, error) {
// 	// 1) Header atau fallback cookie
// 	auth := strings.TrimSpace(c.Get("Authorization"))
// 	if auth == "" {
// 		if cookieTok := c.Cookies("access_token"); cookieTok != "" {
// 			auth = "Bearer " + cookieTok
// 			log.Println("[AUTH] [DEBUG] Authorization diambil dari Cookie")
// 		}
// 	}
// 	if auth == "" {
// 		return "", fmt.Errorf("unauthorized - No token provided")
// 	}

// 	// 2) Robust split & case-insensitive
// 	fields := strings.Fields(auth) // pecah by multiple whitespace
// 	if len(fields) < 2 || !strings.EqualFold(fields[0], "Bearer") {
// 		return "", fmt.Errorf("unauthorized - Invalid token format")
// 	}
// 	tok := strings.TrimSpace(fields[1])
// 	tok = strings.Trim(tok, "\"'") // sanitasi jika ada quote

// 	if tok == "" {
// 		return "", fmt.Errorf("unauthorized - Empty token")
// 	}
// 	return tok, nil
// }

// // Validasi exp dengan toleransi skew (clock drift).
// func validateTokenExpiry(claims jwt.MapClaims, skew time.Duration) error {
// 	expVal, ok := claims["exp"]
// 	if !ok {
// 		return fmt.Errorf("token has no exp")
// 	}

// 	var expUnix int64
// 	switch v := expVal.(type) {
// 	case float64:
// 		expUnix = int64(v)
// 	case int64:
// 		expUnix = v
// 	case int:
// 		expUnix = int64(v)
// 	case string:
// 		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
// 		if err != nil {
// 			return fmt.Errorf("invalid exp format")
// 		}
// 		expUnix = n
// 	default:
// 		// best-effort: stringify lalu parse
// 		s := strings.TrimSpace(fmt.Sprintf("%v", v))
// 		if s == "" {
// 			return fmt.Errorf("invalid exp type")
// 		}
// 		n, err := strconv.ParseInt(s, 10, 64)
// 		if err != nil {
// 			return fmt.Errorf("invalid exp type")
// 		}
// 		expUnix = n
// 	}

// 	now := time.Now().UTC()
// 	expTime := time.Unix(expUnix, 0).UTC()
// 	if now.After(expTime.Add(skew)) {
// 		return fmt.Errorf("token expired at %v", expTime)
// 	}
// 	return nil
// }

// // Ambil user ID dari klaim "id", fallback ke "sub". Keduanya di-parse ke UUID.
// func extractUserID(claims jwt.MapClaims) (uuid.UUID, error) {
// 	// utama: "id"
// 	if idRaw, ok := claims["id"]; ok {
// 		if s, ok := idRaw.(string); ok {
// 			return uuid.Parse(strings.TrimSpace(s))
// 		}
// 		return uuid.Nil, fmt.Errorf("invalid user id type")
// 	}
// 	// fallback: "sub" (sering dipakai untuk subject/user id)
// 	if subRaw, ok := claims["sub"]; ok {
// 		if s, ok := subRaw.(string); ok {
// 			return uuid.Parse(strings.TrimSpace(s))
// 		}
// 	}
// 	return uuid.Nil, fmt.Errorf("no user id")
// }

// // Pastikan user aktif di DB.
// func ensureUserActive(db *gorm.DB, userID uuid.UUID) error {
// 	var user struct {
// 		IsActive bool
// 	}
// 	if err := db.Table("users").
// 		Select("is_active").
// 		Where("id = ?", userID).
// 		First(&user).Error; err != nil {
// 		return err
// 	}
// 	if !user.IsActive {
// 		return errors.New("user inactive")
// 	}
// 	return nil
// }

// /* =======================
//    Store claims to Locals
//    ======================= */

// // Simpan role (lowercased) → Locals("role"), plus user_name & sub (opsional).
// // auth/claims_utils.go
// func storeBasicClaimsToLocals(c *fiber.Ctx, claims jwt.MapClaims) {
// 	if v, ok := claims["role"].(string); ok {
// 		role := strings.ToLower(strings.TrimSpace(v))
// 		if role != "" {
// 			// single-source of truth + alias kompatibilitas
// 			c.Locals(helper.LocRole, role) // biasanya "role"
// 			c.Locals("role", role)         // alias eksplisit
// 			c.Locals("userRole", role)     // alias legacy

// 			log.Printf("[AUTH] locals role set: %s (keys: %q, %q, %q)",
// 				role, helper.LocRole, "role", "userRole")
// 		}
// 	}
// 	if userName, ok := claims["user_name"].(string); ok {
// 		c.Locals("user_name", strings.TrimSpace(userName))
// 	}
// 	if sub, ok := claims["sub"].(string); ok && strings.TrimSpace(sub) != "" {
// 		c.Locals("sub", strings.TrimSpace(sub))
// 	}

// 	// (opsional) log tetap boleh
// 	log.Printf("[AUTH] locals set: role=%v user_name=%v sub=%v",
// 		c.Locals(helper.LocRole), c.Locals("user_name"), c.Locals("sub"))
// }


// // Simpan IDs: admin/teacher/student/union → Locals, dan set "masjid_id" aktif.
// // Prioritas aktif: TEACHER → UNION → ADMIN (selaras helper.GetMasjidIDFromTokenPreferTeacher).
// func storeMasjidIDsToLocals(c *fiber.Ctx, claims jwt.MapClaims) {
// 	adminIDs := toStringSlice(claims["masjid_admin_ids"])
// 	teacherIDs := toStringSlice(claims["masjid_teacher_ids"])
// 	studentIDs := toStringSlice(claims["masjid_student_ids"])
// 	unionIDs := toStringSlice(claims["masjid_ids"])

// 	if len(adminIDs) > 0 {
// 		c.Locals(helper.LocMasjidAdminIDs, adminIDs)
// 	}
// 	if len(teacherIDs) > 0 {
// 		c.Locals(helper.LocMasjidTeacherIDs, teacherIDs)
// 	}
// 	if len(studentIDs) > 0 {
// 		c.Locals(helper.LocMasjidStudentIDs, studentIDs)
// 	}
// 	if len(unionIDs) > 0 {
// 		c.Locals(helper.LocMasjidIDs, unionIDs)
// 	}

// 	// pilih aktif: TEACHER -> UNION -> ADMIN
// 	active := ""
// 	switch {
// 	case len(teacherIDs) > 0:
// 		active = teacherIDs[0]
// 	case len(unionIDs) > 0:
// 		active = unionIDs[0]
// 	case len(adminIDs) > 0:
// 		active = adminIDs[0]
// 	}
// 	if strings.TrimSpace(active) != "" {
// 		c.Locals("masjid_id", active) // backward-compat untuk kode lama
// 		log.Println("[AUTH] [SUCCESS] Active masjid_id stored:", active)
// 	} else {
// 		log.Println("[AUTH] [INFO] Tidak ada masjid_id terdeteksi di token")
// 	}

// 	log.Printf("[AUTH] locals masjid_ids set: admin=%v teacher=%v student=%v union=%v active=%v",
// 		c.Locals(helper.LocMasjidAdminIDs),
// 		c.Locals(helper.LocMasjidTeacherIDs),
// 		c.Locals(helper.LocMasjidStudentIDs),
// 		c.Locals(helper.LocMasjidIDs),
// 		c.Locals("masjid_id"),
// 	)
// }

/* =======================
   Helpers
   ======================= */

// Konversi klaim menjadi []string, tahan tipe []string, []any, atau string tunggal.
// func toStringSlice(v interface{}) []string {
// 	switch t := v.(type) {
// 	case []string:
// 		out := make([]string, 0, len(t))
// 		for _, s := range t {
// 			s = strings.TrimSpace(s)
// 			if s != "" {
// 				out = append(out, s)
// 			}
// 		}
// 		return out
// 	case []interface{}:
// 		out := make([]string, 0, len(t))
// 		for _, it := range t {
// 			if s, ok := it.(string); ok {
// 				s = strings.TrimSpace(s)
// 				if s != "" {
// 					out = append(out, s)
// 				}
// 			}
// 		}
// 		return out
// 	case string:
// 		s := strings.TrimSpace(t)
// 		if s == "" {
// 			return nil
// 		}
// 		return []string{s}
// 	default:
// 		return nil
// 	}
// }
