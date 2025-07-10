package service

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	levelRequirement "masjidku_backend/internals/features/progress/level_rank/model"
	userLogPoint "masjidku_backend/internals/features/progress/points/model"
	userProgress "masjidku_backend/internals/features/progress/progress/model"
)

func AddUserPointLogAndUpdateProgress(db *gorm.DB, userID uuid.UUID, sourceType int, sourceID int, points int) error {
	log.Printf("[SERVICE] AddUserPointLogAndUpdateProgress - userID: %s sourceType: %d sourceID: %d point: %d",
		userID.String(), sourceType, sourceID, points)

	// 1. Simpan log poin
	logEntry := userLogPoint.UserPointLog{
		UserPointLogUserID:     userID,
		UserPointLogPoints:     points,
		UserPointLogSourceType: sourceType,
		UserPointLogSourceID:   sourceID,
		CreatedAt:              time.Now(),
	}
	if err := db.Create(&logEntry).Error; err != nil {
		log.Println("[ERROR] Gagal insert user_point_log:", err)
		return err
	}

	// 2. Tambahkan poin ke user_progress
	if err := db.Model(&userProgress.UserProgress{}).
		Where("user_progress_user_id = ?", userID).
		Updates(map[string]interface{}{
			"user_progress_total_points": gorm.Expr("user_progress_total_points + ?", points),
			"last_updated":               time.Now(),
		}).Error; err != nil {
		log.Println("[ERROR] Gagal update user_progress:", err)
		return err
	}

	// 3. Ambil user_progress terbaru
	var progress userProgress.UserProgress
	if err := db.Where("user_progress_user_id = ?", userID).First(&progress).Error; err != nil {
		log.Println("[ERROR] Gagal ambil user_progress setelah update:", err)
		return err
	}

	// 4. Cari level berdasarkan total poin
	var level levelRequirement.LevelRequirement
	if err := db.Where("level_req_min_points <= ? AND (level_req_max_points IS NULL OR level_req_max_points >= ?)",
		progress.UserProgressTotalPoints, progress.UserProgressTotalPoints).
		Order("level_req_level DESC").
		First(&level).Error; err != nil {
		log.Println("[ERROR] Gagal cari level yang sesuai:", err)
		return err
	}

	// 5. Update level jika berubah
	if level.LevelReqLevel != progress.UserProgressLevel {
		if err := db.Model(&userProgress.UserProgress{}).
			Where("user_progress_user_id = ?", userID).
			Update("user_progress_level", level.LevelReqLevel).Error; err != nil {
			log.Println("[ERROR] Gagal update level user_progress:", err)
			return err
		}
		log.Printf("[LEVEL-UP] User %s naik ke level %d", userID.String(), level.LevelReqLevel)
		progress.UserProgressLevel = level.LevelReqLevel
	}

	// 6. Ambil rank berdasarkan level terbaru
	var rank levelRequirement.RankRequirement
	if err := db.Where("rank_req_min_level <= ? AND (rank_req_max_level IS NULL OR rank_req_max_level >= ?)",
		progress.UserProgressLevel, progress.UserProgressLevel).
		Order("rank_req_rank DESC").
		First(&rank).Error; err != nil {
		log.Println("[ERROR] Gagal cari rank yang sesuai:", err)
		return err
	}

	// 7. Update rank
	if err := db.Model(&userProgress.UserProgress{}).
		Where("user_progress_user_id = ?", userID).
		Update("user_progress_rank", rank.RankReqRank).Error; err != nil {
		log.Println("[ERROR] Gagal update rank user_progress:", err)
		return err
	}
	log.Printf("[RANK-UP] User %s naik ke rank %d (%s)", userID.String(), rank.RankReqRank, rank.RankReqName)

	log.Printf("[SUCCESS] Poin berhasil ditambahkan: %d poin", points)
	return nil
}
