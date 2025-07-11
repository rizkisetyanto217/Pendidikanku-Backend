// middlewares/cors.go

package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CorsMiddleware membuat middleware CORS
func CorsMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: strings.Join([]string{
			"http://localhost:5173",
			"http://localhost:5177",
			"http://127.0.0.1:5500",
			"https://masjidkubackend-production.up.railway.app",
			"https://web-six-theta-13.vercel.app",
			"https://masjidku-web-production.up.railway.app", // ✅ tambahkan ini
		}, ", "),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	})
}
