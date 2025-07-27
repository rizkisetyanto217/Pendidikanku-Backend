package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	// xendit "github.com/xendit/xendit-go/v7"
	// serviceXendit "masjidku_backend/internals/service" // sesuaikan path

	"masjidku_backend/internals/configs"
	database "masjidku_backend/internals/databases"
	"masjidku_backend/internals/features/donations/donations/service"
	scheduler "masjidku_backend/internals/features/users/auth/scheduler"
	middlewares "masjidku_backend/internals/middlewares"
	routes "masjidku_backend/internals/route"
)

func main() {
	configs.LoadEnv()
	app := fiber.New()

	middlewares.SetupMiddlewares(app)
	database.ConnectDB()
	scheduler.StartBlacklistCleanupScheduler(database.DB)

	// ✅ MIDTRANS setup
	service.InitMidtrans(configs.GetEnv("MIDTRANS_SERVER_KEY"))

    
	// ✅ Route
	routes.SetupRoutes(app, database.DB)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("✅ Listening on PORT: %s", port)
	log.Fatal(app.Listen("0.0.0.0:" + port))
}
