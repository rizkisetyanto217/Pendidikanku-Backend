package helper

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type TrashReaperConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	Prefix          string
	RetentionDays   int
	CronSchedule    string
	DryRun          bool
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	default:
		return false
	}
}
func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// ── ENTRYPOINT: panggil dari main.go
func StartTrashReaperCron(db *gorm.DB) {
	log.Printf("[DEBUG] DRY_RUN=%q", os.Getenv("DRY_RUN"))

	cfg := TrashReaperConfig{
		Endpoint:        normalizeEndpoint(os.Getenv("ALI_OSS_ENDPOINT")),
		AccessKeyID:     os.Getenv("ALI_OSS_ACCESS_KEY"),
		AccessKeySecret: os.Getenv("ALI_OSS_SECRET_KEY"),
		Bucket:          os.Getenv("ALI_OSS_BUCKET"),
		Prefix:          getEnvOrDefault("REAPER_PREFIX", "spam/"),
		RetentionDays:   getEnvInt("RETENTION_DAYS", 30),
		CronSchedule:    getEnvOrDefault("CRON_SCHEDULE", "15 2 * * *"),
		DryRun:          getEnvBool("DRY_RUN", false),
	}

	// OSS client (optional)
	var bucket *oss.Bucket
	if cfg.Endpoint != "" && cfg.AccessKeyID != "" && cfg.AccessKeySecret != "" && cfg.Bucket != "" {
		if cli, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret); err == nil {
			if b, e := cli.Bucket(cfg.Bucket); e == nil {
				bucket = b
			} else {
				log.Printf("[TRASH-REAPER] get bucket gagal: %v", e)
			}
		} else {
			log.Printf("[TRASH-REAPER] OSS init gagal: %v", err)
		}
	} else {
		log.Printf("[TRASH-REAPER] ENV ALI_OSS_* tidak lengkap — skip OSS reaper, jalankan DB reaper saja")
	}

	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))

	_, err := c.AddFunc(cfg.CronSchedule, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		retention := time.Duration(cfg.RetentionDays) * 24 * time.Hour

		// 1) OSS spam cleaner
		if bucket != nil {
			if err := runOSSReaper(ctx, bucket, cfg.Prefix, retention, cfg.DryRun); err != nil {
				log.Printf("[TRASH-REAPER] OSS error: %v", err)
			}
		}

		// 2) DB soft-delete cleaner
		if err := runDBReaper(ctx, db, retention); err != nil {
			log.Printf("[TRASH-REAPER] DB error: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("[TRASH-REAPER] add cron gagal: %v", err)
	}
	log.Printf("[TRASH-REAPER] started schedule=%q prefix=%q retention=%dd dryRun=%v",
		cfg.CronSchedule, cfg.Prefix, cfg.RetentionDays, cfg.DryRun)
	c.Start()
}

// ── Bagian OSS (dipakai ulang)
func runOSSReaper(ctx context.Context, bucket *oss.Bucket, prefix string, retention time.Duration, dryRun bool) error {
	threshold := time.Now().Add(-retention)
	log.Printf("[OSS-REAPER] scanning prefix=%q threshold=%s dry=%v", prefix, threshold.Format(time.RFC3339), dryRun)

	marker := oss.Marker("")
	var keysToDelete []string
	total := 0

	for {
		lor, err := bucket.ListObjects(oss.Prefix(prefix), marker, oss.MaxKeys(1000))
		if err != nil {
			return err
		}
		for _, obj := range lor.Objects {
			total++
			if obj.Key == "" {
				continue
			}
			if obj.LastModified.Before(threshold) {
				keysToDelete = append(keysToDelete, obj.Key)
			}
		}
		if lor.IsTruncated {
			marker = oss.Marker(lor.NextMarker)
		} else {
			break
		}
	}

	if len(keysToDelete) == 0 {
		log.Printf("[OSS-REAPER] nothing to delete; scanned=%d under %q", total, prefix)
		return nil
	}
	if dryRun {
		log.Printf("[OSS-REAPER] DRY-RUN would delete %d/%d objects under %q", len(keysToDelete), total, prefix)
		return nil
	}

	deleted := 0
	for i := 0; i < len(keysToDelete); i += 1000 {
		end := i + 1000
		if end > len(keysToDelete) {
			end = len(keysToDelete)
		}
		batch := keysToDelete[i:end]
		if _, err := bucket.DeleteObjects(batch, oss.DeleteObjectsQuiet(true)); err != nil {
			log.Printf("[OSS-REAPER] delete batch %d-%d gagal: %v", i, end, err)
			continue
		}
		deleted += len(batch)
	}
	log.Printf("[OSS-REAPER] deleted %d objects (scanned=%d) under %q", deleted, total, prefix)
	return nil
}

// ── Bagian DB: hard-delete semua row yg soft-deleted lebih tua dari cutoff
func runDBReaper(ctx context.Context, db *gorm.DB, retention time.Duration) error {
	if db == nil {
		return nil
	}
	cutoff := time.Now().Add(-retention)

	type target struct{ Table, Col string }
	targets := []target{
		{Table: "masjids", Col: "masjid_deleted_at"},
		{Table: "masjid_profiles", Col: "masjid_profile_deleted_at"},
		// tambah tabel soft-delete lain di sini…
	}

	totalDeleted := 0
	for _, t := range targets {
		res := db.WithContext(ctx).Exec(
			`DELETE FROM `+t.Table+` WHERE `+t.Col+` IS NOT NULL AND `+t.Col+` < ?`,
			cutoff,
		)
		if err := res.Error; err != nil {
			log.Printf("[DB-REAPER] %s: delete error: %v", t.Table, err)
			continue
		}
		ra := res.RowsAffected
		totalDeleted += int(ra)
		if ra > 0 {
			log.Printf("[DB-REAPER] %s: hard-deleted %d rows older than %s", t.Table, ra, cutoff.Format(time.RFC3339))
		}
	}
	if totalDeleted == 0 {
		log.Printf("[DB-REAPER] nothing to delete (cutoff=%s)", cutoff.Format(time.RFC3339))
	}
	return nil
}

func GetRetentionDuration() time.Duration {
	return time.Duration(getEnvInt("RETENTION_DAYS", 30)) * 24 * time.Hour
}
