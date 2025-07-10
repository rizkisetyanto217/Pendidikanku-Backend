package masjid


import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/masjids/masjids/model"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Struktur sesuai dengan dto.MasjidRequest
type MasjidSeed struct {
	MasjidName         string  `json:"masjid_name"`
	MasjidBioShort     string  `json:"masjid_bio_short"`
	MasjidLocation     string  `json:"masjid_location"`
	MasjidLatitude     float64 `json:"masjid_latitude"`
	MasjidLongitude    float64 `json:"masjid_longitude"`
	MasjidImageURL     string  `json:"masjid_image_url"`
	MasjidSlug         string  `json:"masjid_slug"`
	MasjidIsVerified   bool    `json:"masjid_is_verified"`
	MasjidInstagramURL string  `json:"masjid_instagram_url"`
	MasjidWhatsappURL  string  `json:"masjid_whatsapp_url"`
	MasjidYoutubeURL   string  `json:"masjid_youtube_url"`
}

func SeedMasjidsFromJSON(db *gorm.DB, filePath string) {
	log.Println("📥 Membaca file:", filePath)

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Gagal membaca file JSON: %v", err)
	}

	var masjids []MasjidSeed
	if err := json.Unmarshal(file, &masjids); err != nil {
		log.Fatalf("❌ Gagal decode JSON: %v", err)
	}

	for _, m := range masjids {
		var existing model.MasjidModel
		if err := db.Where("masjid_slug = ?", m.MasjidSlug).First(&existing).Error; err == nil {
			log.Printf("ℹ️ Masjid dengan slug %s sudah ada, lewati...", m.MasjidSlug)
			continue
		}

		newMasjid := model.MasjidModel{
			MasjidID:           uuid.New(),
			MasjidName:         m.MasjidName,
			MasjidBioShort:     m.MasjidBioShort,
			MasjidLocation:     m.MasjidLocation,
			MasjidLatitude:     m.MasjidLatitude,
			MasjidLongitude:    m.MasjidLongitude,
			MasjidImageURL:     m.MasjidImageURL,
			MasjidSlug:         m.MasjidSlug,
			MasjidIsVerified:   m.MasjidIsVerified,
			MasjidInstagramURL: m.MasjidInstagramURL,
			MasjidWhatsappURL:  m.MasjidWhatsappURL,
			MasjidYoutubeURL:   m.MasjidYoutubeURL,
			MasjidCreatedAt:    time.Now(),
			MasjidUpdatedAt:    time.Now(),
		}

		if err := db.Create(&newMasjid).Error; err != nil {
			log.Printf("❌ Gagal insert masjid %s: %v", m.MasjidSlug, err)
		} else {
			log.Printf("✅ Berhasil insert masjid %s (%s)", newMasjid.MasjidName, newMasjid.MasjidSlug)
		}
	}
}
