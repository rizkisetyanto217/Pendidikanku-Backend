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
	// Default PROD origins — HTTPS & domain spesifik
	// (tambahkan/kurangi sesuai domain kamu)
	def := strings.Join([]string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"https://masjidku.org",
		"https://www.masjidku.org",
		"https://pendidikanku-frontend-2-production.up.railway.app",
	}, ",")

	// Override via ENV saat perlu (CI/CD, staging, blue/green)
	env := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if env == "" {
		env = def
	}
	origins := sanitizeOrigins(env)

	return cors.New(cors.Config{
		// NOTE: Saat AllowCredentials=true, HARUS daftar origin spesifik (bukan *)
		AllowOrigins: origins,

		// Metode umum
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",

		// Header yang diizinkan dari FE (whitelist)
		// Sertakan semua yang kamu pakai di FE (Authorization untuk Bearer, X-CSRF-Token untuk double-submit, dsb.)
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-User-Id, X-CSRF-Token, X-XSRF-Token, X-XSRF-TOKEN",

		// Header yang boleh dibaca FE dari response (opsional)
		ExposeHeaders: "Content-Type, Authorization, X-Request-Id",

		// WAJIB: karena kita kirim cookie refresh_token (HttpOnly) lintas origin
		AllowCredentials: true,

		// Cache preflight 24 jam
		MaxAge: 86400,
	})
}
