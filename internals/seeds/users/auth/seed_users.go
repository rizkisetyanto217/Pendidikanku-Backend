package user

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"masjidku_backend/internals/features/users/user/model"

	authHelper "masjidku_backend/internals/features/users/auth/helper"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserSeed struct {
	UserName         string `json:"user_name"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	Role             string `json:"role"`
	SecurityQuestion string `json:"security_question"`
	SecurityAnswer   string `json:"security_answer"`
}

func SeedUsersFromJSON(db *gorm.DB, filePath string) {
	log.Println("📥 Membaca file user:", filePath)

	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("❌ Gagal membaca file JSON: %v", err)
	}

	var inputs []UserSeed
	if err := json.Unmarshal(file, &inputs); err != nil {
		log.Fatalf("❌ Gagal decode JSON: %v", err)
	}

	for _, data := range inputs {
		var existing model.UserModel
		if err := db.Where("email = ?", data.Email).First(&existing).Error; err == nil {
			log.Printf("ℹ️ User dengan email '%s' sudah ada, dilewati.", data.Email)
			continue
		}

		// 🔐 Hash password sebelum disimpan
		hashedPassword, err := authHelper.HashPassword(data.Password)
		if err != nil {
			log.Printf("❌ Gagal hash password untuk '%s': %v", data.Email, err)
			continue
		}

		newUser := model.UserModel{
			ID:               uuid.New(),
			UserName:         data.UserName,
			Email:            data.Email,
			Password:         hashedPassword,
			Role:             data.Role,
			SecurityQuestion: data.SecurityQuestion,
			SecurityAnswer:   data.SecurityAnswer,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := db.Create(&newUser).Error; err != nil {
			log.Printf("❌ Gagal insert user '%s': %v", data.Email, err)
		} else {
			log.Printf("✅ Berhasil insert user '%s'", data.Email)
		}
	}
}
