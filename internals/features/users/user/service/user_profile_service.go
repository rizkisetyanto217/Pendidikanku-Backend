package service

import (
	"log"

	"masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateInitialUserProfile(db *gorm.DB, userID uuid.UUID) {
	profile := model.UsersProfileModel{
		UserID: userID,
		Gender: nil, // atau models.Male jika mau default
	}
	if err := db.Create(&profile).Error; err != nil {
		log.Printf("[ERROR] Failed to create user profile: %v", err)
	}
}