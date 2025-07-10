package routes

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	databases "masjidku_backend/internals/databases"
)

func BaseRoutes(app *fiber.App, db *gorm.DB) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Fiber & Supabase PostgreSQL connected successfully 🚀")
	})

	app.Get("/panic-test", func(c *fiber.Ctx) error {
		panic("Simulasi panic error!") // testing panic handler
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		sqlDB, err := databases.DB.DB()
		dbStatus := "Connected"
		serverStatus := "OK"
		httpStatus := fiber.StatusOK

		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "Database connection error"
			serverStatus = "DOWN"
			httpStatus = fiber.StatusServiceUnavailable
		}

		uptime := time.Since(startTime).Seconds()

		return c.Status(httpStatus).JSON(fiber.Map{
			"status":         serverStatus,
			"database":       dbStatus,
			"server_time":    time.Now().Format(time.RFC3339),
			"uptime_seconds": int(uptime),
			"environment":    os.Getenv("RAILWAY_ENVIRONMENT"),
		})
	})
}
