package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings" // ✅ dipakai untuk guard path
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/utils"
	"gorm.io/gorm"

	"schoolku_backend/internals/configs"
	database "schoolku_backend/internals/databases"

	attend "schoolku_backend/internals/features/school/classes/class_attendance_sessions/service"
	authsched "schoolku_backend/internals/features/users/auth/scheduler"

	osshelper "schoolku_backend/internals/helpers/oss"
	routes "schoolku_backend/internals/route"

	payctl "schoolku_backend/internals/features/finance/payments/controller"
	helperAuth "schoolku_backend/internals/helpers/auth"
	middlewares "schoolku_backend/internals/middlewares"
)

func main() {
	configs.LoadEnv()
	db := initDB()

	workersCtx, cancelWorkers := context.WithCancel(context.Background())
	startWorkers(workersCtx, db)

	app := buildApp()

	// === PUBLIC: Midtrans webhook (payments) — tanpa middleware apapun ===
	midtransServerKey := os.Getenv("MIDTRANS_SERVER_KEY")
	useProd := strings.EqualFold(os.Getenv("MIDTRANS_USE_PROD"), "true")
	paymentWebhookCtrl := payctl.NewPaymentController(db, midtransServerKey, useProd)

	// (opsional) GET ping buat test dari dashboard Midtrans
	app.Get("/public/donations/midtrans/webhook", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// POST notifikasi transaksi (akan update status)
	app.Post("/public/donations/midtrans/webhook", paymentWebhookCtrl.MidtransWebhook)

	// ✅ Auth middleware dengan guard: jangan halangi OPTIONS & /api/auth/*
	app.Use(func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}
		p := c.Path()
		if strings.HasPrefix(p, "/health") ||
			strings.HasPrefix(p, "/api/auth/") ||
			strings.HasPrefix(p, "/public/donations/midtrans/webhook") { // ✅ whitelist baru
			return c.Next()
		}

		return helperAuth.MiddlewareBlacklistOnly(db, os.Getenv("JWT_SECRET"))(c)
	})

	routes.SetupRoutes(app, db)

	app.Server().ReadTimeout = 15 * time.Second
	app.Server().WriteTimeout = 30 * time.Second
	app.Server().IdleTimeout = 90 * time.Second

	port := getPort()
	go func() {
		log.Printf("✅ Listening on :%s", port)
		if err := app.Listen("0.0.0.0:" + port); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	waitForShutdown(app, cancelWorkers, db)
}

func buildApp() *fiber.App {
	app := fiber.New(fiber.Config{
		JSONEncoder:             sonic.Marshal,
		JSONDecoder:             sonic.Unmarshal,
		DisableStartupMessage:   true,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"0.0.0.0/0"},
	})

	// urutan middleware
	app.Use(recover.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
	app.Use(etag.New())

	// ✅ Pakai CORS dari middleware terpisah
	app.Use(middlewares.CorsMiddleware())

	// ✅ Pastikan preflight dihandle cepat (header CORS sudah dipasang oleh middleware di atas)
	app.Options("/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	// observability ringkas
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

	app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

/* ===============================
   DB init & tuning
   =============================== */

func initDB() *gorm.DB {
	database.ConnectDB()
	database.TunePool()
	database.WarmUpQueries()

	// (Opsional) Naikkan pool sedikit untuk handle spike 08:00
	if sqlDB, err := database.DB.DB(); err == nil {
		sqlDB.SetMaxOpenConns(40) // default kamu 20 → naikkan jika perlu
		sqlDB.SetMaxIdleConns(20) // default kamu 10
		sqlDB.SetConnMaxLifetime(10 * time.Minute)
	}
	return database.DB
}

/* ===============================
   Workers
   =============================== */

func startWorkers(ctx context.Context, db *gorm.DB) {
	// 1) Attendance auto-seed (T-60m; polling & batch via ENV)
	attCfg, err := attend.LoadConfig()
	if err != nil {
		log.Fatalf("attendance config error: %v", err)
	}
	go attend.RunSeedWorker(ctx, db, attCfg)

	// 2) Auth: cleanup token blacklist
	authsched.StartBlacklistCleanupScheduler(db)

	// 3) Payments: init Midtrans
	// payment.InitMidtrans(configs.GetEnv("MIDTRANS_SERVER_KEY"))

	// 4) OSS trash reaper (gabungan cron pembersih)
	osshelper.StartTrashReaperCron(db)
}

/* ===============================
   Shutdown
   =============================== */

func waitForShutdown(app *fiber.App, cancelWorkers context.CancelFunc, db *gorm.DB) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 1) Stop workers
	cancelWorkers()

	// 2) Shutdown HTTP server (grace)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)

	// 3) Close DB pool
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	log.Println("👋 Shutdown complete.")
}

/* ===============================
   Utils
   =============================== */

func getPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "3000"
}
