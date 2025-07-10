package tooltip

import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/utils/tooltips/model"
	"os"

	"gorm.io/gorm"
)

type TooltipSeed struct {
	TooltipKeyword          string `json:"tooltip_keyword"`
	TooltipDescriptionShort string `json:"tooltip_description_short"`
	TooltipDescriptionLong  string `json:"tooltip_description_long"`
}

func SeedTooltipsFromJSON(db *gorm.DB, filePath string) {
	log.Println("📥 Membaca file:", filePath)

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Gagal membaca file JSON: %v", err)
	}

	var seeds []TooltipSeed
	if err := json.Unmarshal(file, &seeds); err != nil {
		log.Fatalf("❌ Gagal decode JSON: %v", err)
	}

	for _, seed := range seeds {
		var existing model.Tooltip
		if err := db.Where("tooltip_keyword = ?", seed.TooltipKeyword).First(&existing).Error; err == nil {
			log.Printf("ℹ️ Tooltip '%s' sudah ada, lewati...", seed.TooltipKeyword)
			continue
		}

		tooltip := model.Tooltip{
			TooltipKeyword:          seed.TooltipKeyword,
			TooltipDescriptionShort: seed.TooltipDescriptionShort,
			TooltipDescriptionLong:  seed.TooltipDescriptionLong,
		}

		if err := db.Create(&tooltip).Error; err != nil {
			log.Printf("❌ Gagal insert '%s': %v", seed.TooltipKeyword, err)
		} else {
			log.Printf("✅ Berhasil insert '%s'", seed.TooltipKeyword)
		}
	}
}
