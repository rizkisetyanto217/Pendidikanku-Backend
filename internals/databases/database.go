package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	log.Println("🔌 Koneksi ke PostgreSQL (Supabase)...")

	sslmode := getenv("DB_SSLMODE", "require")
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&application_name=masjidku&options=-c statement_timeout=3000",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		sslmode,
	)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // 👍 cocok untuk PgBouncer (transaction pooling)
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Gagal konek DB: %v", err)
	}
	DB = db
	log.Println("✅ DB connected.")
}

func TunePool() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("pool tune err: %v", err)
		return
	}
	// ⚖️ Sesuaikan dengan limit Supabase/PgBouncer
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxIdleTime(60 * time.Second)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
}

// WarmUpQueries mengisi idle pool & membuka koneksi awal agar cold-start pendek
func WarmUpQueries() {
	go func() {
		time.Sleep(500 * time.Millisecond) // beri waktu server naik
		sqlDB, err := DB.DB()
		if err != nil {
			log.Printf("warm-up get DB err: %v", err)
			return
		}
		warm := 8 // <= selaras dengan SetMaxIdleConns(10)
		var wg sync.WaitGroup
		wg.Add(warm)
		for i := 0; i < warm; i++ {
			go func() {
				defer wg.Done()
				_ = sqlDB.Ping()
				// atau query ringan lain yang sering dipakai
				// DB.Exec("SELECT 1")
			}()
		}
		wg.Wait()
		log.Println("🔥 DB pool warmed.")
	}()
}

// func ping() error {
// 	sqlDB, err := DB.DB()
// 	if err != nil {
// 		return err
// 	}
// 	return sqlDB.Ping()
// }

func SQL() (*sql.DB, error) { // optional helper
	return DB.DB()
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}