// middlewares/cors.go

package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// SetupMiddlewareCors membuat middleware CORS
func CorsMiddleware() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5500, http://127.0.0.1:5500, https://masjidkubackend-production.up.railway.app, https://web-six-theta-13.vercel.app", // sesuaikan
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	})
}
