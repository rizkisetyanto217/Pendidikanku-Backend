package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/utils"

	"masjidku_backend/internals/configs"
	database "masjidku_backend/internals/databases"
	"masjidku_backend/internals/features/payment/donations/service"
	scheduler "masjidku_backend/internals/features/users/auth/scheduler"
	osshelper "masjidku_backend/internals/helpers/oss"
	routes "masjidku_backend/internals/route"
)

func main() {
	configs.LoadEnv()

	app := fiber.New(fiber.Config{
		JSONEncoder:             sonic.Marshal,
		JSONDecoder:             sonic.Unmarshal,
		DisableStartupMessage:   true,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"0.0.0.0/0"}, // sesuaikan sesuai kebutuhan
	})

	// Middleware dasar
	app.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
	app.Use(etag.New())

	// Observability ringan: request-id + timing + guard timeout
	app.Use(func(c *fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = utils.UUID()
		}
		c.Set("X-Request-ID", id)
		c.Locals("reqid", id)

		start := time.Now()
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		c.SetUserContext(ctx)

		err := c.Next()
		log.Printf("[REQ] id=%s %s %s status=%d dur=%s",
			id, c.Method(), c.OriginalURL(), c.Response().StatusCode(), time.Since(start))
		return err
	})

	// DB connect + tuning + warmup
	database.ConnectDB()
	database.TunePool()
	database.WarmUpQueries()

	// Schedulers lain (contoh: cleanup token blacklist)
	scheduler.StartBlacklistCleanupScheduler(database.DB)

	// Midtrans
	service.InitMidtrans(configs.GetEnv("MIDTRANS_SERVER_KEY"))

	// Healthcheck
	app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// Routes
	routes.SetupRoutes(app, database.DB)

	// Server timeouts
	app.Server().ReadTimeout = 15 * time.Second
	app.Server().WriteTimeout = 30 * time.Second
	app.Server().IdleTimeout  = 90 * time.Second

	// Cron gabungan: bersihin OSS spam/ + hard-delete rows soft-deleted
	osshelper.StartTrashReaperCron(database.DB)

	// Port
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Start server (non-blocking)
	go func() {
		log.Printf("✅ Listening on :%s", port)
		if err := app.Listen("0.0.0.0:" + port); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)

	// Tutup pool DB
	if sqlDB, err := database.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
