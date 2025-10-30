// middlewares/cors.go
package middlewares

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// sanitizeOrigins: trim spasi, buang trailing slash, filter empty, dedup
func sanitizeOrigins(csv string) string {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return ""
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimSuffix(p, "/") // match persis dgn Origin header
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, ",")
}

func CorsMiddleware() fiber.Handler {
	// Default origins (DEV + beberapa domain PROD)
	def := strings.Join([]string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"https://masjidku.org",
		"https://www.masjidku.org",
		"https://pendidikanku-frontend-2-production.up.railway.app",
	}, ",")

	// Bisa override via ENV (CORS_ALLOW_ORIGINS="https://foo,https://bar")
	env := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if env == "" {
		env = def
	}
	origins := sanitizeOrigins(env)

	return cors.New(cors.Config{
		// NOTE: AllowCredentials=true → tidak boleh "*"
		AllowOrigins: origins,

		// Metode umum
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",

		// ✅ Header yang diizinkan dari FE (whitelist)
		// Sertakan SEMUA header kustom yang mungkin dikirim FE.
		AllowHeaders: strings.Join([]string{
			"Origin",
			"Accept",
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"X-CSRF-Token", // dipakai utk double-submit
			"X-Masjid-ID",  // ⬅️ PENTING: inilah penyebab error kamu
		}, ","),

		// Opsional: header yang diekspos ke FE
		ExposeHeaders: "Content-Type, Authorization, X-Request-Id",

		// Harus true karena pakai cookie HttpOnly refresh_token lintas origin
		AllowCredentials: true,

		// Cache preflight 24 jam (detik)
		MaxAge: 86400,
	})
}
