package rank

import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/progress/level_rank/model"
	"os"

	"gorm.io/gorm"
)

type RankSeed struct {
	RankReqRank     int    `json:"rank_req_rank"`
	RankReqName     string `json:"rank_req_name"`
	RankReqMinLevel int    `json:"rank_req_min_level"`
	RankReqMaxLevel *int   `json:"rank_req_max_level"` // nullable
}

func SeedRanksRequirementsFromJSON(db *gorm.DB, filePath string) {
	log.Println("📥 Membaca file:", filePath)

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Gagal membaca file JSON: %v", err)
	}

	var input []RankSeed
	if err := json.Unmarshal(file, &input); err != nil {
		log.Fatalf("❌ Gagal decode JSON: %v", err)
	}

	for _, r := range input {
		var existing model.RankRequirement
		if err := db.Where("rank_req_rank = ?", r.RankReqRank).First(&existing).Error; err == nil {
			log.Printf("ℹ️ Rank %d sudah ada, lewati...", r.RankReqRank)
			continue
		}

		newRank := model.RankRequirement{
			RankReqRank:     r.RankReqRank,
			RankReqName:     r.RankReqName,
			RankReqMinLevel: r.RankReqMinLevel,
			RankReqMaxLevel: r.RankReqMaxLevel,
		}

		if err := db.Create(&newRank).Error; err != nil {
			log.Printf("❌ Gagal insert Rank %d: %v", r.RankReqRank, err)
		} else {
			log.Printf("✅ Berhasil insert Rank %d (%s)", r.RankReqRank, r.RankReqName)
		}
	}
}
