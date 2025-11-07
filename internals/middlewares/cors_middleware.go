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

// middlewares/cors.go
func CorsMiddleware() fiber.Handler {
	origins := sanitizeOrigins(os.Getenv("FRONTEND_ORIGINS"))
	if origins == "" {
		origins = strings.Join([]string{
			"http://localhost:5174",
			"http://127.0.0.1:5174",
			"http://localhost:5175",
			"http://127.0.0.1:5175",
			"http://localhost:5176",
			"http://127.0.0.1:5176",
			"http://localhost:5177",
			"http://127.0.0.1:5177",
			"https://masjidku.org",
			"https://www.masjidku.org",
			"https://madinahsalam.up.railway.app/",
			"https://pendidikanku-frontend-2-production.up.railway.app",
		}, ",")
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Accept, Content-Type, Authorization, X-Requested-With, X-CSRF-Token, X-School-ID",
		ExposeHeaders:    "Content-Type, Authorization, X-Request-Id",
		AllowCredentials: true,
		MaxAge:           86400,
	})
}
