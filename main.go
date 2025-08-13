package main

import (
	"context"
	"fmt"
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
	"masjidku_backend/internals/features/donations/donations/service"
	scheduler "masjidku_backend/internals/features/users/auth/scheduler"
	middlewares "masjidku_backend/internals/middlewares"
	routes "masjidku_backend/internals/route"
)

func main() {
	configs.LoadEnv()

	app := fiber.New(fiber.Config{
		// 🚀 JSON super cepat
		JSONEncoder:            sonic.Marshal,
		JSONDecoder:            sonic.Unmarshal,
		DisableStartupMessage:  true,
		ProxyHeader:            fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:         []string{"0.0.0.0/0"}, // sesuaikan dengan CIDR Cloudflare jika perlu
	})

	// ⚙️ middleware dasar + performa
	app.Use(compress.New(compress.Config{Level: compress.LevelDefault})) // gzip
	app.Use(etag.New())                                                  // 304 caching

	// 🔎 Request-ID + timing (observability ringan)
	app.Use(func(c *fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = utils.UUID()
		}
		c.Set("X-Request-ID", id)
		c.Locals("reqid", id)
		start := time.Now()
		// HTTP timeout guard (selaras dengan statement_timeout di DB)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		c.SetUserContext(ctx)
		err := c.Next()
		dur := time.Since(start)
		log.Printf("[REQ] id=%s %s %s status=%d dur=%s", id, c.Method(), c.OriginalURL(), c.Response().StatusCode(), dur)
		return err
	})

	middlewares.SetupMiddlewares(app)

	// 🔌 DB connect + pool + warm-up
	database.ConnectDB()
	database.TunePool()
	database.WarmUpQueries()

	// ⏱ scheduler setelah DB siap
	scheduler.StartBlacklistCleanupScheduler(database.DB)

	// ✅ MIDTRANS
	service.InitMidtrans(configs.GetEnv("MIDTRANS_SERVER_KEY"))

	// ❤️ Health check (anti-cold start)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// ✅ Routes
	routes.SetupRoutes(app, database.DB)

	// 🔒 Keep-Alive & timeout koneksi server
	app.Server().ReadTimeout = 15 * time.Second
	app.Server().WriteTimeout = 30 * time.Second
	app.Server().IdleTimeout = 90 * time.Second

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Start server non-blocking
	go func() {
		log.Printf("✅ Listening on :%s", port)
		if err := app.Listen("0.0.0.0:" + port); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown + tutup pool DB
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)

	if sqlDB, err := database.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// Helper opsional untuk set Cache-Control publik
func setPublicCache(c *fiber.Ctx, seconds int) {
	c.Set("Cache-Control", fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d", seconds, seconds*2))
}