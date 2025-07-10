package scheduler

import (
	"log"
	"os"
	"strconv"
	"time"

	"masjidku_backend/internals/features/users/auth/model"

	"gorm.io/gorm"
)

func StartBlacklistCleanupScheduler(db *gorm.DB) {
	go func() {
		// Ambil TTL dari env (default: 7 hari)
		ttlDays := 7
		if val := os.Getenv("TOKEN_BLACKLIST_TTL_DAYS"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil {
				ttlDays = parsed
			}
		}

		for {
			log.Println("[CLEANUP] Menjalankan pembersihan token_blacklist...")

			deleteBefore := time.Now().Add(-time.Duration(ttlDays) * 24 * time.Hour)

			var expiredTokens []model.TokenBlacklist
			if err := db.
				Where("expired_at < ? AND deleted_at IS NULL", deleteBefore).
				Limit(100).
				Find(&expiredTokens).Error; err != nil {
				log.Printf("[CLEANUP ERROR] Gagal ambil token kadaluarsa: %v", err)
			} else if len(expiredTokens) > 0 {
				if err := db.Delete(&expiredTokens).Error; err != nil {
					log.Printf("[CLEANUP ERROR] Gagal hapus token: %v", err)
				} else {
					log.Printf("[CLEANUP] %d token kadaluarsa dihapus", len(expiredTokens))
				}
			} else {
				log.Println("[CLEANUP] Tidak ada token yang memenuhi syarat dihapus")
			}

			// Jalankan tiap 24 jam
			time.Sleep(24 * time.Hour)
		}
	}()
}
