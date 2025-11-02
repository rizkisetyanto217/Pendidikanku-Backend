package school

// import (
// 	"encoding/json"
// 	"log"
// 	"schoolku_backend/internals/features/schools/schools/model"
// 	"os"
// 	"time"

// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// // Struktur sesuai dengan dto.SchoolRequest
// type SchoolSeed struct {
// 	SchoolName         string  `json:"school_name"`
// 	SchoolBioShort     string  `json:"school_bio_short"`
// 	SchoolLocation     string  `json:"school_location"`
// 	SchoolLatitude     float64 `json:"school_latitude"`
// 	SchoolLongitude    float64 `json:"school_longitude"`
// 	SchoolImageURL     string  `json:"school_image_url"`
// 	SchoolSlug         string  `json:"school_slug"`
// 	SchoolIsVerified   bool    `json:"school_is_verified"`
// 	SchoolInstagramURL string  `json:"school_instagram_url"`
// 	SchoolWhatsappURL  string  `json:"school_whatsapp_url"`
// 	SchoolYoutubeURL   string  `json:"school_youtube_url"`
// }

// func SeedSchoolsFromJSON(db *gorm.DB, filePath string) {
// 	log.Println("📥 Membaca file:", filePath)

// 	file, err := os.ReadFile(filePath)
// 	if err != nil {
// 		log.Fatalf("❌ Gagal membaca file JSON: %v", err)
// 	}

// 	var schools []SchoolSeed
// 	if err := json.Unmarshal(file, &schools); err != nil {
// 		log.Fatalf("❌ Gagal decode JSON: %v", err)
// 	}

// 	for _, m := range schools {
// 		var existing model.SchoolModel
// 		if err := db.Where("school_slug = ?", m.SchoolSlug).First(&existing).Error; err == nil {
// 			log.Printf("ℹ️ School dengan slug %s sudah ada, lewati...", m.SchoolSlug)
// 			continue
// 		}

// 		newSchool := model.SchoolModel{
// 			SchoolID:           uuid.New(),
// 			SchoolName:         m.SchoolName,
// 			SchoolBioShort:     m.SchoolBioShort,
// 			SchoolLocation:     m.SchoolLocation,
// 			// SchoolLatitude:     m.SchoolLatitude,
// 			// SchoolLongitude:    m.SchoolLongitude,
// 			SchoolImageURL:     m.SchoolImageURL,
// 			SchoolSlug:         m.SchoolSlug,
// 			SchoolIsVerified:   m.SchoolIsVerified,
// 			SchoolInstagramURL: m.SchoolInstagramURL,
// 			SchoolWhatsappURL:  m.SchoolWhatsappURL,
// 			SchoolYoutubeURL:   m.SchoolYoutubeURL,
// 			SchoolCreatedAt:    time.Now(),
// 			SchoolUpdatedAt:    time.Now(),
// 		}

// 		if err := db.Create(&newSchool).Error; err != nil {
// 			log.Printf("❌ Gagal insert school %s: %v", m.SchoolSlug, err)
// 		} else {
// 			log.Printf("✅ Berhasil insert school %s (%s)", newSchool.SchoolName, newSchool.SchoolSlug)
// 		}
// 	}
// }
