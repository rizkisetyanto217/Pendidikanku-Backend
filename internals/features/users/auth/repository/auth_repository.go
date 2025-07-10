package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	authModel "masjidku_backend/internals/features/users/auth/model"
	userModel "masjidku_backend/internals/features/users/user/model"
)

// ====================== USER ======================

func FindUserByEmailOrUsername(db *gorm.DB, identifier string) (*userModel.UserModel, error) {
	var user userModel.UserModel
	if err := db.Where("email = ? OR user_name = ?", identifier, identifier).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func FindUserByGoogleID(db *gorm.DB, googleID string) (*userModel.UserModel, error) {
	var user userModel.UserModel
	if err := db.Where("google_id = ?", googleID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func FindUserByEmailOrUsernameLight(db *gorm.DB, identifier string) (*userModel.UserModel, error) {
	var user userModel.UserModel
	if err := db.Select("id", "password", "is_active").
		Where("email = ? OR user_name = ?", identifier, identifier).
		First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func FindUserByID(db *gorm.DB, userID uuid.UUID) (*userModel.UserModel, error) {
	var user userModel.UserModel
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func FindUserByEmail(db *gorm.DB, email string) (*userModel.UserModel, error) {
	var user userModel.UserModel
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func CreateUser(db *gorm.DB, user *userModel.UserModel) error {
	return db.Create(user).Error
}

func UpdateUserPassword(db *gorm.DB, userID uuid.UUID, newPassword string) error {
	return db.Model(&userModel.UserModel{}).Where("id = ?", userID).Update("password", newPassword).Error
}

// ====================== REFRESH TOKEN ======================

func CreateRefreshToken(db *gorm.DB, token *authModel.RefreshToken) error {
	return db.Create(token).Error
}

func FindRefreshToken(db *gorm.DB, token string) (*authModel.RefreshToken, error) {
	var rt authModel.RefreshToken
	if err := db.Where("token = ?", token).First(&rt).Error; err != nil {
		return nil, err
	}
	return &rt, nil
}

func DeleteRefreshToken(db *gorm.DB, token string) error {
	return db.Where("token = ?", token).Delete(&authModel.RefreshToken{}).Error
}

// ====================== BLACKLIST TOKEN ======================

func BlacklistToken(db *gorm.DB, token string, duration time.Duration) error {
	blacklisted := authModel.TokenBlacklist{
		Token:     token,
		ExpiredAt: time.Now().Add(duration),
	}
	return db.Create(&blacklisted).Error
}