// internals/features/users/auth/repository/repository.go
package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	authModel "masjidku_backend/internals/features/users/auth/model"
	userModel "masjidku_backend/internals/features/users/users/model"
)

/* ====================== USER ====================== */

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

/* ====================== REFRESH TOKEN ====================== */

func CreateRefreshToken(db *gorm.DB, token *authModel.RefreshTokenModel) error {
	return db.Create(token).Error
}

func FindRefreshToken(db *gorm.DB, token string) (*authModel.RefreshTokenModel, error) {
	var rt authModel.RefreshTokenModel
	// CATATAN: kolom model kamu bertipe []byte (bytea). Kalau memang disimpan hash,
	// hash dulu `token` sebelum query, atau ubah model.Token -> string biar cocok dengan query ini.
	if err := db.Where("token = ?", token).First(&rt).Error; err != nil {
		return nil, err
	}
	return &rt, nil
}

func DeleteRefreshToken(db *gorm.DB, token string) error {
	// Sama seperti FindRefreshToken: pastikan tipe & nilai yang dicari sesuai penyimpanan (hash/plain).
	return db.Where("token = ?", token).Delete(&authModel.RefreshTokenModel{}).Error
}

/* ====================== BLACKLIST TOKEN ====================== */

func BlacklistToken(db *gorm.DB, token string, ttl time.Duration) error {
	return db.Create(&authModel.TokenBlacklistModel{
		Token:     token,
		ExpiredAt: time.Now().UTC().Add(ttl), // <- pakai field yang benar
	}).Error
}

func CleanupExpiredBlacklist(db *gorm.DB) (int64, error) {
	// Kolom di DB: expired_at (bukan expires_at)
	res := db.Exec(`DELETE FROM token_blacklist WHERE expired_at <= ?`, time.Now().UTC())
	return res.RowsAffected, res.Error
}





// IsUsernameTaken — cek apakah username sudah dipakai
func IsUsernameTaken(db *gorm.DB, username string) (bool, error) {
	if username == "" {
		return false, errors.New("username cannot be empty")
	}

	var exists bool
	err := db.
		Raw(`SELECT EXISTS(SELECT 1 FROM users WHERE user_name = ? AND deleted_at IS NULL)`, username).
		Scan(&exists).Error
	if err != nil {
		return false, err
	}
	return exists, nil
}

