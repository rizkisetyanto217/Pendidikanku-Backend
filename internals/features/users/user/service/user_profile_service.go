package service

import (
	"errors"
	"log"

	profilemodel "masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateInitialUserProfile membuat profile kosong untuk user tertentu.
// Mengembalikan error agar bisa di-handle di caller.
func CreateInitialUserProfile(db *gorm.DB, userID uuid.UUID) error {
	// Cek apakah sudah ada profil untuk user ini
	var count int64
	if err := db.Model(&profilemodel.UserProfileModel{}).
		Where("users_profile_user_id = ?", userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		// sudah ada, tidak perlu buat lagi
		return nil
	}

	profile := profilemodel.UserProfileModel{
		UserProfileUserID: userID,
		// Kolom lain biarkan default DB (is_public_profile = true, is_verified = false, dst)
		UserProfileGender: nil, // atau pointer ke profilemodel.Male/Female jika mau default
	}

	if err := db.Create(&profile).Error; err != nil {
		log.Printf("[ERROR] Failed to create users_profile: %v", err)
		return err
	}

	// Validasi sanity: pastikan row tercipta
	if profile.UserProfileID == uuid.Nil {
		return errors.New("users_profile creation failed: empty users_profile_id")
	}
	return nil
}
