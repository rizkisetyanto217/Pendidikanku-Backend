package user

import (
	"encoding/json"
	"log"
	"masjidku_backend/internals/features/users/user/model"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UsersProfileSeed struct {
	UserID       uuid.UUID     `json:"user_id"`
	DonationName string        `json:"donation_name"`
	FullName     string        `json:"full_name"`
	Gender       *model.Gender `json:"gender"`
	PhoneNumber  string        `json:"phone_number"`
	Bio          string        `json:"bio"`
	Location     string        `json:"location"`
	Occupation   string        `json:"occupation"`
}

func SeedUsersProfileFromJSON(db *gorm.DB, filePath string) {
	log.Println("📥 Membaca file:", filePath)

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Gagal membaca file JSON: %v", err)
	}

	var seeds []UsersProfileSeed
	if err := json.Unmarshal(file, &seeds); err != nil {
		log.Fatalf("❌ Gagal decode JSON: %v", err)
	}

	// Ambil semua user_id yang sudah ada
	var existingIDs []uuid.UUID
	if err := db.Model(&model.UsersProfileModel{}).
		Select("user_id").
		Find(&existingIDs).Error; err != nil {
		log.Fatalf("❌ Gagal ambil user_id yang sudah ada: %v", err)
	}

	existingMap := make(map[uuid.UUID]bool)
	for _, id := range existingIDs {
		existingMap[id] = true
	}

	// Kumpulkan data yang belum ada
	var newProfiles []model.UsersProfileModel
	for _, p := range seeds {
		if existingMap[p.UserID] {
			log.Printf("ℹ️ Profil user dengan ID '%s' sudah ada, dilewati.", p.UserID)
			continue
		}

		newProfiles = append(newProfiles, model.UsersProfileModel{
			UserID:       p.UserID,
			DonationName: p.DonationName,
			FullName:     p.FullName,
			Gender:       p.Gender,
			PhoneNumber:  p.PhoneNumber,
			Bio:          p.Bio,
			Location:     p.Location,
			Occupation:   p.Occupation,
		})
	}

	// Bulk insert
	if len(newProfiles) > 0 {
		if err := db.Create(&newProfiles).Error; err != nil {
			log.Fatalf("❌ Gagal bulk insert users_profile: %v", err)
		}
		log.Printf("✅ Berhasil insert %d profil user", len(newProfiles))
	} else {
		log.Println("ℹ️ Tidak ada profil baru untuk diinsert.")
	}
}
