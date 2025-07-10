package levels

import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/progress/level_rank/model"
	"os"

	"gorm.io/gorm"
)

type LevelSeed struct {
	LevelReqLevel     int    `json:"level_req_level"`
	LevelReqName      string `json:"level_req_name"`
	LevelReqMinPoints int    `json:"level_req_min_points"`
	LevelReqMaxPoints *int   `json:"level_req_max_points"` // bisa null
}

func SeedLevelRequirementsFromJSON(db *gorm.DB, filePath string) {
	log.Println("📥 Membaca file:", filePath)

	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Gagal baca file JSON: %v", err)
	}

	var data []LevelSeed
	if err := json.Unmarshal(content, &data); err != nil {
		log.Fatalf("❌ Gagal decode JSON: %v", err)
	}

	for _, item := range data {
		var existing model.LevelRequirement
		if err := db.Where("level_req_level = ?", item.LevelReqLevel).First(&existing).Error; err == nil {
			log.Printf("ℹ️ Level %d sudah ada, lewati...", item.LevelReqLevel)
			continue
		}

		record := model.LevelRequirement{
			LevelReqLevel:     item.LevelReqLevel,
			LevelReqName:      item.LevelReqName,
			LevelReqMinPoints: item.LevelReqMinPoints,
			LevelReqMaxPoints: item.LevelReqMaxPoints,
		}

		if err := db.Create(&record).Error; err != nil {
			log.Printf("❌ Gagal insert Level %d: %v", item.LevelReqLevel, err)
		} else {
			log.Printf("✅ Berhasil insert Level %d", item.LevelReqLevel)
		}
	}
}
