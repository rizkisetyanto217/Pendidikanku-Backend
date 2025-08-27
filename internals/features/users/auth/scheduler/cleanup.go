// internals/features/users/auth/scheduler/blacklist_cleanup.go
package scheduler

import (
	"log"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"
)

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Jalan periodik: hapus token_blacklist yang sudah expired (expired_at <= now)
func StartBlacklistCleanupScheduler(db *gorm.DB) {
	intervalSec := envInt("BLACKLIST_CLEANUP_INTERVAL_SEC", 604800) // default: 7 day (1 week)
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)

	go func() {
		log.Printf("[BL-CLEANUP] started interval=%ds", intervalSec)
		for range ticker.C {
			now := time.Now().UTC()
			res := db.Exec(`DELETE FROM token_blacklist WHERE expired_at <= ?`, now)
			if res.Error != nil {
				log.Printf("[BL-CLEANUP] error: %v", res.Error)
				continue
			}
			if res.RowsAffected > 0 {
				log.Printf("[BL-CLEANUP] deleted=%d rows (<= %s)", res.RowsAffected, now.Format(time.RFC3339))
			} else {
				log.Printf("[BL-CLEANUP] nothing to delete")
			}
		}
	}()
}
