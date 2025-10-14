// middlewares/cors.go
package middlewares

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CorsMiddleware: baca CORS_ALLOW_ORIGINS dari ENV (kalau ada),
// kalau tidak ada pakai default (dev + domain produksi kamu).
func CorsMiddleware() fiber.Handler {
	origins := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if origins == "" {
		origins = strings.Join([]string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"https://masjidku.org",
			"https://www.masjidku.org",
			// tambahkan staging lain di sini bila perlu:
			// "https://masjidku-web-production.up.railway.app",
			// "https://web-six-theta-13.vercel.app",
		}, ",")
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-User-Id, X-Requested-With",
		ExposeHeaders:    "Content-Type, Authorization",
		AllowCredentials: true, // pakai Bearer token, bukan cookie
		MaxAge:           86400, // cache preflight 24 jam
	})
}
