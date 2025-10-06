package helper

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/*
   =========================================================
   SCHEMA (opsional, sesuai skema kamu)
   =========================================================
*/

// EnsureSchema: biarkan no-op supaya tidak bentrok dengan migration kamu.
// Kalau tetap mau safety, boleh cek kolom minimal.
func EnsureSchema(db *gorm.DB) error {
	return nil
}

/*
   =========================================================
   LOW-LEVEL UTILS
   =========================================================
*/

func hmacHex(msg, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil)) // cocok ke kolom TEXT
}

// ambil token dari Authorization: Bearer ... atau cookie access_token
func getRawAccessToken(c *fiber.Ctx) string {
	authHeader := strings.TrimSpace(c.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	return strings.TrimSpace(c.Cookies("access_token"))
}

/*
   =========================================================
   CORE API (sesuai tabel: token TEXT, expired_at, deleted_at)
   =========================================================
*/

// Add: simpan HMAC(access_token) (hex) ke kolom token TEXT.
func Add(ctx context.Context, db *gorm.DB, rawAccessToken, jwtSecret string, expiresAt time.Time) error {
	if db == nil || strings.TrimSpace(rawAccessToken) == "" || strings.TrimSpace(jwtSecret) == "" {
		return nil
	}
	tokenHex := hmacHex(rawAccessToken, jwtSecret)
	// ON CONFLICT sesuai unique(token) di skema kamu
	return db.WithContext(ctx).Exec(`
		INSERT INTO token_blacklist (token, expired_at)
		VALUES (?, ?)
		ON CONFLICT (token) DO UPDATE
		SET expired_at = EXCLUDED.expired_at,
		    deleted_at = NULL
	`, tokenHex, expiresAt).Error
}

// IsBlacklisted: ada baris aktif dan belum expired?
func IsBlacklisted(ctx context.Context, db *gorm.DB, rawAccessToken, jwtSecret string) (bool, error) {
	if db == nil || strings.TrimSpace(rawAccessToken) == "" || strings.TrimSpace(jwtSecret) == "" {
		return false, nil
	}
	tokenHex := hmacHex(rawAccessToken, jwtSecret)
	var exists bool
	err := db.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM token_blacklist
		  WHERE token = ?
		    AND deleted_at IS NULL
		    AND expired_at > NOW()
		)
	`, tokenHex).Scan(&exists).Error
	return exists, err
}

// PurgeExpired: hapus yang sudah lewat (atau ganti ke soft-delete sesuai preferensi)
func PurgeExpired(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	// Hard delete:
	return db.WithContext(ctx).Exec(`DELETE FROM token_blacklist WHERE expired_at <= NOW()`).Error

	// Atau jika mau soft delete:
	// return db.WithContext(ctx).Exec(`
	//   UPDATE token_blacklist
	//   SET deleted_at = NOW()
	//   WHERE deleted_at IS NULL AND expired_at <= NOW()
	// `).Error
}

/*
   =========================================================
   MIDDLEWARE
   =========================================================
*/

// Pasang ini DI DEPAN middleware JWT-mu.
func MiddlewareBlacklistOnly(db *gorm.DB, jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := getRawAccessToken(c)
		if strings.TrimSpace(raw) == "" {
			return c.Next()
		}
		bl, err := IsBlacklisted(c.Context(), db, raw, jwtSecret)
		if err == nil && bl {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi sudah keluar. Silakan login lagi.",
			})
		}
		return c.Next()
	}
}
