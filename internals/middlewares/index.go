package middlewares

import (
	loggerMiddleware "masjidku_backend/internals/middlewares/logger"

	"github.com/gofiber/fiber/v2"
)

// SetupMiddlewares menggabungkan semua middleware penting
func SetupMiddlewares(app *fiber.App) {
	app.Use(CorsMiddleware())                    // 🌐 CORS global
	app.Use(RecoveryMiddleware())                // 🔥 Tangkap panic
	app.Use(loggerMiddleware.LoggerMiddleware()) // 📝 Logger Request/Response
}
