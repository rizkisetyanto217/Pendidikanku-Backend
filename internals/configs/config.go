package configs

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

var (
	JWTSecret          string
	JWTRefreshSecret   string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
	DB                 *gorm.DB
)

// =======================
// ENV LOADER
// =======================
func LoadEnv() {
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		if err := godotenv.Load(); err != nil {
			log.Println("⚠️ Tidak menemukan .env file, menggunakan ENV dari sistem")
		} else {
			log.Println("✅ .env file berhasil dimuat!")
		}
	} else {
		log.Println("🚀 Running in Railway, menggunakan ENV dari sistem")
	}

	JWTSecret = GetEnv("JWT_SECRET")
	JWTRefreshSecret = GetEnv("JWT_REFRESH_SECRET")
	GoogleClientID = GetEnv("GOOGLE_CLIENT_ID")
	GoogleClientSecret = GetEnv("GOOGLE_CLIENT_SECRET")
	GoogleRedirectURI = GetEnv("GOOGLE_REDIRECT_URI")

	if JWTSecret == "" {
		log.Println("❌ JWT_SECRET belum diset!")
	} else {
		log.Println("✅ JWT_SECRET berhasil dimuat.")
	}

	if JWTRefreshSecret == "" {
		log.Println("❌ JWT_REFRESH_SECRET belum diset!")
	} else {
		log.Println("✅ JWT_REFRESH_SECRET berhasil dimuat.")
	}

	if GoogleClientID == "" {
		log.Println("❌ GOOGLE_CLIENT_ID belum diset!")
	} else {
		log.Println("✅ GOOGLE_CLIENT_ID berhasil dimuat.")
	}
}

func GetEnv(key string, defaultValue ...string) string {
	value, exists := os.LookupEnv(key)
	if !exists && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return value
}

// =======================
// DATABASE CONNECTOR
// =======================
func InitSeederDB() *gorm.DB {
	dbUser := GetEnv("DB_USER")
	dbPassword := GetEnv("DB_PASSWORD")
	dbHost := GetEnv("DB_HOST")
	dbPort := GetEnv("DB_PORT")
	dbName := GetEnv("DB_NAME")
	dbSSL := GetEnv("DB_SSLMODE", "require")

	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSL)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // ✅ hindari cache prepared statement
	}), &gorm.Config{
		Logger: NewGormLogger(),
	})
	if err != nil {
		log.Fatalf("❌ Gagal koneksi ke database (Seeder): %v", err)
	}
	log.Println("✅ Database (Seeder) terkoneksi.")
	return db
}

// =======================
// GORM LOGGER CUSTOM
// =======================
type GormLogger struct {
	SlowThreshold time.Duration
	LogLevel      gormLogger.LogLevel
}

func NewGormLogger() gormLogger.Interface {
	return &GormLogger{
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      gormLogger.Info,
	}
}

func (l *GormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	l.LogLevel = level
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[INFO] "+msg, data...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[WARN] "+msg, data...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[ERROR] "+msg, data...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	file := utils.FileWithLineNum()

	if err != nil {
		log.Printf("[ERROR] %s | %v | %s | %d rows | %s", file, err, elapsed, rows, sql)
	} else if elapsed > l.SlowThreshold {
		log.Printf("[SLOW SQL] %s | %s | %d rows | %s", file, elapsed, rows, sql)
	} else {
		log.Printf("[QUERY] %s | %s | %d rows | %s", file, elapsed, rows, sql)
	}
}
