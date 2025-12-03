// internals/databases/database.go
package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	log.Println("🔌 Init PostgreSQL connection...")

	// 1) Prioritas: DATABASE_URL penuh
	if url := strings.TrimSpace(os.Getenv("DATABASE_URL")); url != "" {
		openWithDSN(url)
		return
	}

	// 2) Fallback: komponenan DB_* (harus lengkap)
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	sslmode := getenv("DB_SSLMODE", "require") // Supabase pooler: require

	if host == "" || port == "" || user == "" || name == "" {
		log.Fatalf(
			"❌ ENV DB_* tidak lengkap. Butuh DB_HOST, DB_PORT, DB_USER, DB_NAME (DB_PASSWORD optional). " +
				"Atau set DATABASE_URL langsung.",
		)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&application_name=schoolku&options=-c statement_timeout=3000",
		urlEncode(user),
		urlEncode(pass),
		host,
		port,
		name,
		sslmode,
	)

	openWithDSN(dsn)
}

func openWithDSN(dsn string) {
	log.Printf("🔎 Connecting with DSN: %s", redactDSN(dsn))

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // cocok utk PgBouncer (transaction pooling)
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Gagal konek DB: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ Gagal ambil sql.DB dari GORM: %v", err)
	}

	// Pool tuning — sesuaikan dg limit Supabase/PgBouncer
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxIdleTime(60 * time.Second)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)

	DB = db
	log.Println("✅ DB connected.")
}

func WarmUpQueries() {
	go func() {
		time.Sleep(500 * time.Millisecond)
		sqlDB, err := DB.DB()
		if err != nil {
			log.Printf("warm-up get DB err: %v", err)
			return
		}
		warm := 8
		var wg sync.WaitGroup
		wg.Add(warm)
		for i := 0; i < warm; i++ {
			go func() { defer wg.Done(); _ = sqlDB.Ping() }()
		}
		wg.Wait()
		log.Println("🔥 DB pool warmed.")
	}()
}

// Tambahkan di internals/databases/database.go

func TunePool() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("pool tune err: %v", err)
		return
	}

	// Bisa diatur via ENV; fallback nilai aman untuk Supabase/PgBouncer
	maxOpen := getenvInt("DB_MAX_OPEN_CONNS", 20)
	maxIdle := getenvInt("DB_MAX_IDLE_CONNS", 10)
	idleSec := getenvInt("DB_CONN_MAX_IDLE_TIME_SEC", 60)
	lifeMin := getenvInt("DB_CONN_MAX_LIFETIME_MIN", 10)

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxIdleTime(time.Duration(idleSec) * time.Second)
	sqlDB.SetConnMaxLifetime(time.Duration(lifeMin) * time.Minute)

	log.Printf("✅ DB pool tuned: maxOpen=%d maxIdle=%d idle=%ds life=%dm",
		maxOpen, maxIdle, idleSec, lifeMin)
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func SQL() (*sql.DB, error) { return DB.DB() }

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// --- helpers ---

func redactDSN(dsn string) string {
	// URL style: postgresql://user:pass@host:port/db?...
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		at := strings.Index(dsn, "@")
		if at > 0 {
			prefix := dsn[:at] // scheme://user:pass
			colon := strings.LastIndex(prefix, ":")
			if colon > 0 {
				return prefix[:colon] + ":****" + dsn[at:]
			}
		}
		return dsn
	}
	// keyword style: host=... user=... password=...
	return strings.ReplaceAll(dsn, "password="+extractKV(dsn, "password"), "password=****")
}

func extractKV(s, key string) string {
	for _, part := range strings.Fields(s) {
		if strings.HasPrefix(part, key+"=") {
			return strings.TrimPrefix(part, key+"=")
		}
	}
	return ""
}

func urlEncode(s string) string {
	// minimal encode untuk karakter khusus pada user/pass
	r := strings.ReplaceAll(s, "@", "%40")
	r = strings.ReplaceAll(r, ":", "%3A")
	return r
}