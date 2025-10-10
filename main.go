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
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/utils"
	"gorm.io/gorm"

	"masjidku_backend/internals/configs"
	database "masjidku_backend/internals/databases"

	payment "masjidku_backend/internals/features/payment/donations/service"
	attend "masjidku_backend/internals/features/school/classes/class_attendance_sessions/service"
	authsched "masjidku_backend/internals/features/users/auth/scheduler"

	osshelper "masjidku_backend/internals/helpers/oss"
	routes "masjidku_backend/internals/route"

	helperAuth "masjidku_backend/internals/helpers/auth"
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
	// 4) Build Fiber app + routes
	app := buildApp()

	// ⬇️ tambahkan dua baris ini
	if err := helperAuth.EnsureSchema(db); err != nil {
		log.Fatalf("ensure blacklist schema: %v", err)
	}
	app.Use(helperAuth.MiddlewareBlacklistOnly(db, os.Getenv("JWT_SECRET")))

	// baru set routes
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

/*
===============================

	HTTP (Fiber) setup
	===============================
*/
func buildApp() *fiber.App {
	app := fiber.New(fiber.Config{
		JSONEncoder:           sonic.Marshal,
		JSONDecoder:           sonic.Unmarshal,
		DisableStartupMessage: true,

		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"0.0.0.0/0"},
	})

	// --- urutan middleware yang baik ---
	app.Use(recover.New())
	app.Use(compress.New(compress.Config{Level: compress.LevelDefault}))
	app.Use(etag.New())

	// ====== CORS (fix preflight) ======
	origins := os.Getenv("CORS_ALLOW_ORIGINS")
	if origins == "" {
		// default dev + contoh domain produksi (ganti sesuai domainmu)
		origins = "http://localhost:5173,http://127.0.0.1:5173,https://app.sekolahislamku.com"
	}
	allowHeaders := os.Getenv("CORS_ALLOW_HEADERS")
	if allowHeaders == "" {
		allowHeaders = "Origin, Content-Type, Accept, Authorization, X-Requested-With"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins, // spesifik, bukan "*"
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     allowHeaders, // termasuk "Authorization"
		ExposeHeaders:    "Content-Type, Authorization",
		AllowCredentials: true, // <-- UBAH ke true
		MaxAge:           86400,
	}))

	// Pastikan semua preflight OPTIONS dibalas 204 agar browser happy
	app.Options("/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	// Observability ringan: request-id + timing + guard timeout per request
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
