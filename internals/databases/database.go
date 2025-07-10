package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	fmt.Println("🔌 Memulai koneksi ke Supabase PostgreSQL...")

	// Bangun DSN
	dsn := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)
	fmt.Println("🔍 DSN:", dsn)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})

	if err != nil {
		log.Fatalf("❌ Gagal koneksi ke Supabase:\n%v", err)
	}

	DB = db
	fmt.Println("✅ Berhasil konek ke Supabase PostgreSQL sekarang!")
}
