// package main

// import (
// 	"log"
// 	"os"

// 	"github.com/gofiber/fiber/v2"
// 	// xendit "github.com/xendit/xendit-go/v7"
// 	// serviceXendit "masjidku_backend/internals/service" // sesuaikan path

// 	"masjidku_backend/internals/configs"
// 	database "masjidku_backend/internals/databases"
// 	"masjidku_backend/internals/features/donations/donations/service"
// 	scheduler "masjidku_backend/internals/features/users/auth/scheduler"
// 	middlewares "masjidku_backend/internals/middlewares"
// 	routes "masjidku_backend/internals/route"
// )

// func main() {
// 	configs.LoadEnv()
// 	app := fiber.New()

// 	middlewares.SetupMiddlewares(app)
// 	database.ConnectDB()
// 	scheduler.StartBlacklistCleanupScheduler(database.DB)

// 	// ✅ MIDTRANS setup
// 	service.InitMidtrans(configs.GetEnv("MIDTRANS_SERVER_KEY"))

// 	// ✅ Route
// 	routes.SetupRoutes(app, database.DB)

// 	port := os.Getenv("PORT")
// 	if port == "" {
// 		port = "3000"
// 	}
// 	log.Printf("✅ Listening on PORT: %s", port)
// 	log.Fatal(app.Listen("0.0.0.0:" + port))
// }

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	// "github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"

	"masjidku_backend/internals/configs"
	database "masjidku_backend/internals/databases"
	"masjidku_backend/internals/features/donations/donations/service"
	scheduler "masjidku_backend/internals/features/users/auth/scheduler"
	middlewares "masjidku_backend/internals/middlewares"
	routes "masjidku_backend/internals/route"
)

func main() {
	configs.LoadEnv()

	app := fiber.New(fiber.Config{
		// 🚀 JSON super cepat
		// JSONEncoder:        sonic.Marshal,
		// JSONDecoder:        sonic.Unmarshal,
		DisableStartupMessage: true,
		// (opsional) naikkan body limit kalau butuh upload
	})

	// ⚙️ middleware dasar + performa
	app.Use(compress.New()) // gzip
	app.Use(etag.New())

	middlewares.SetupMiddlewares(app)

	// 🔌 DB connect + pool + warm-up
	database.ConnectDB()
	database.TunePool()     // <-- tambahkan fungsi ini (di bawah)
	database.WarmUpQueries() // <-- dan ini

	// ⏱ scheduler setelah DB siap
	scheduler.StartBlacklistCleanupScheduler(database.DB)

	// ✅ MIDTRANS
	service.InitMidtrans(configs.GetEnv("MIDTRANS_SERVER_KEY"))

	// ❤️ Health check (buat anti cold start)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// ✅ Routes
	routes.SetupRoutes(app, database.DB)

	// 🔒 Keep-Alive & timeout biar koneksi stabil
	app.Server().ReadTimeout = 15 * time.Second
	app.Server().WriteTimeout = 30 * time.Second
	app.Server().IdleTimeout = 90 * time.Second

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	go func() {
		log.Printf("✅ Listening on :%s", port)
		if err := app.Listen("0.0.0.0:" + port); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)
}
