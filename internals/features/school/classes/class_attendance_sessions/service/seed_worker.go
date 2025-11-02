package service

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
)

type dueSession struct {
	SessionID string
	SchoolID  string
}

func RunSeedWorker(ctx context.Context, db *gorm.DB, cfg Config) {
	if !cfg.Enabled {
		log.Println("[seed-worker] disabled")
		return
	}

	ticker := time.NewTicker(cfg.PollEvery)
	defer ticker.Stop()

	log.Printf("[seed-worker] start window=%dmin every=%s batch=%d",
		cfg.WindowMins, cfg.PollEvery, cfg.BatchSize)

	for {
		select {
		case <-ctx.Done():
			log.Println("[seed-worker] stop")
			return
		case <-ticker.C:
			if err := runOnce(ctx, db, cfg); err != nil {
				log.Printf("[seed-worker] runOnce error: %v", err)
			}
		}
	}
}

func runOnce(ctx context.Context, db *gorm.DB, cfg Config) error {
	// grace: sesi yang sudah lewat <= 15 menit tetap disiapkan (jaga-jaga worker sempat mati)
	const graceMins = 15

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// per-TX guard (hindari deadlock lama & query nyangkut)
		if err := tx.Exec(`SET LOCAL lock_timeout = '2s'; SET LOCAL statement_timeout = '5s';`).Error; err != nil {
			return err
		}
		rows, err := tx.Raw(`
			SELECT s.class_attendance_sessions_id   AS session_id,
				s.class_attendance_sessions_school_id AS school_id
			FROM class_attendance_sessions s
			WHERE s.class_attendance_sessions_deleted_at IS NULL
			AND s.class_attendance_sessions_status IN ('scheduled','open')
			AND s.class_attendance_sessions_start_at IS NOT NULL
			AND s.class_attendance_sessions_start_at
					BETWEEN NOW() - (? * INTERVAL '1 minute')
						AND NOW() + (? * INTERVAL '1 minute')
			ORDER BY s.class_attendance_sessions_start_at
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		`, graceMins, cfg.WindowMins, cfg.BatchSize).Rows()

		if err != nil {
			return err
		}
		defer rows.Close()

		var (
			okCount int
			ds      dueSession
		)

		for rows.Next() {
			if err := rows.Scan(&ds.SessionID, &ds.SchoolID); err != nil {
				return err
			}
			if err := ensureSessionSeededTx(tx, ds.SessionID, ds.SchoolID, cfg.AutoOpen); err != nil {
				log.Printf("[seed-worker] seed %s err: %v", ds.SessionID, err)
				continue
			}
			okCount++
		}
		if err := rows.Err(); err != nil {
			return err
		}

		if okCount > 0 {
			log.Printf("[seed-worker] seeded %d session(s)", okCount)
		}
		return nil
	})
}
