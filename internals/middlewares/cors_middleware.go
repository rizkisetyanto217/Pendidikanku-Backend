// middlewares/cors.go
package middlewares

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// sanitizeOrigins: trim spasi, buang trailing slash, filter empty
func sanitizeOrigins(csv string) string {
	if strings.TrimSpace(csv) == "" {
		return ""
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// buang trailing slash biar match persis dengan Origin
		p = strings.TrimSuffix(p, "/")
		if p == "" {
			continue
		}
		// dedup
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, ",")
}

func CorsMiddleware() fiber.Handler {
	// Default origins (tanpa trailing slash)
	def := strings.Join([]string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"https://masjidku.org",
		"https://www.masjidku.org",
		"https://pendidikanku-frontend-2-production.up.railway.app",
		// tambahkan staging lain di sini bila perlu
	}, ",")

	env := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if env == "" {
		env = def
	}
	origins := sanitizeOrigins(env)

	return cors.New(cors.Config{
		// Harus daftar origin spesifik saat AllowCredentials=true
		AllowOrigins: origins,

		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",

		// Tambahkan X-CSRF-Token untuk double-submit
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-User-Id, X-CSRF-Token",

		// (Opsional) kalau kamu mau baca header tertentu dari FE
		ExposeHeaders: "Content-Type, Authorization, X-Request-Id",

		// WAJIB true untuk kirim cookie refresh_token cross-site
		AllowCredentials: true,

		// Cache preflight 24 jam
		MaxAge: 86400,
	})
}
