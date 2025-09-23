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
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/utils"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	database "masjidku_backend/internals/databases"

	payment "masjidku_backend/internals/features/payment/donations/service"
	attend "masjidku_backend/internals/features/school/sessions/sessions/service"
	authsched "masjidku_backend/internals/features/users/auth/scheduler"

	osshelper "masjidku_backend/internals/helpers/oss"
	routes "masjidku_backend/internals/route"
)

/* ===============================
   Bootstrap
   =============================== */

func main() {
	// 1) Load ENV
	configs.LoadEnv()

	// 2) Init DB + pool
	db := initDB()

	// 3) Start background workers (attendance auto-seed, auth blacklist cleanup, OSS reaper)
	workersCtx, cancelWorkers := context.WithCancel(context.Background())
	startWorkers(workersCtx, db)

	// 4) Build Fiber app + routes
	app := buildApp()
	routes.SetupRoutes(app, db)

	// 5) HTTP timeouts
	app.Server().ReadTimeout = 15 * time.Second
	app.Server().WriteTimeout = 30 * time.Second
	app.Server().IdleTimeout = 90 * time.Second

	// 6) Start HTTP server (non-blocking)
	port := getPort()
	go func() {
		log.Printf("✅ Listening on :%s", port)
		if err := app.Listen("0.0.0.0:" + port); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// 7) Wait signals → graceful shutdown
	waitForShutdown(app, cancelWorkers, db)
}

/* ===============================
   HTTP (Fiber) setup
   =============================== */

func buildApp() *fiber.App {
	app := fiber.New(fiber.Config{
		JSONEncoder:           sonic.Marshal,
		JSONDecoder:           sonic.Unmarshal,
		DisableStartupMessage: true,

		// Reverse proxy safe defaults
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"0.0.0.0/0"},
	})

	// Middlewares
	app.Use(recover.New()) // proteksi panic → 500 + log
	app.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
	app.Use(etag.New())

	// Observability ringan: request-id + timing + guard timeout per request
	app.Use(func(c *fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = utils.UUID()
		}
		c.Set("X-Request-ID", id)
		c.Locals("reqid", id)

		start := time.Now()
		// Guard timeout per request → cancel context ke downstream (DB, dsb)
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		c.SetUserContext(ctx)

		err := c.Next()
		log.Printf("[REQ] id=%s %s %s status=%d dur=%s",
			id, c.Method(), c.OriginalURL(), c.Response().StatusCode(), time.Since(start))
		return err
	})

	// Health
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
	payment.InitMidtrans(configs.GetEnv("MIDTRANS_SERVER_KEY"))

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
