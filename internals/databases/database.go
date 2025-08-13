// package database

// import (
// 	"fmt"
// 	"log"
// 	"os"

// 	"gorm.io/driver/postgres"
// 	"gorm.io/gorm"
// )

// var DB *gorm.DB

// func ConnectDB() {
// 	fmt.Println("🔌 Memulai koneksi ke Supabase PostgreSQL...")

// 	// Bangun DSN
// 	dsn := fmt.Sprintf(
// 		"user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
// 		os.Getenv("DB_USER"),
// 		os.Getenv("DB_PASSWORD"),
// 		os.Getenv("DB_HOST"),
// 		os.Getenv("DB_PORT"),
// 		os.Getenv("DB_NAME"),
// 		os.Getenv("DB_SSLMODE"),
// 	)
// 	fmt.Println("🔍 DSN:", dsn)

// 	db, err := gorm.Open(postgres.New(postgres.Config{
// 		DSN:                  dsn,
// 		PreferSimpleProtocol: true,
// 	}), &gorm.Config{})

// 	if err != nil {
// 		log.Fatalf("❌ Gagal koneksi ke Supabase:\n%v", err)
// 	}

// 	DB = db
// 	fmt.Println("✅ Berhasil konek ke Supabase PostgreSQL sekarang!")
// }

package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	log.Println("🔌 Koneksi ke PostgreSQL (Supabase)...")

	// ✅ Gunakan URL/DSN lengkap + statement_timeout
	// Catatan: kalau pakai PgBouncer, ganti host/port ke port PgBouncer (mis. 6543) dan biarkan PreferSimpleProtocol=true
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

func WarmUpQueries() {
	// jalankan ringan supaya koneksi/pool “keisi” & siap
	go func() {
		time.Sleep(500 * time.Millisecond) // beri waktu server naik
		if err := ping(); err != nil {
			log.Printf("warm-up ping err: %v", err)
		}
		// tambahkan query ringan yang paling sering dipakai, mis. cek masjid by slug
		// DB.Exec("SELECT 1")
	}()
}

func ping() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
