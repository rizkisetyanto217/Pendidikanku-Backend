// models/donation_like_model.go

package model

import (
	"time"

	"github.com/google/uuid"
)

type DonationLikeModel struct {
	DonationLikeID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"donation_like_id"`
	DonationLikeIsLiked   bool       `gorm:"default:true" json:"donation_like_is_liked"`
	DonationLikeDonationID uuid.UUID `gorm:"type:uuid;not null" json:"donation_like_donation_id"`
	DonationLikeUserID    uuid.UUID  `gorm:"type:uuid;not null" json:"donation_like_user_id"`
	DonationLikeUpdatedAt time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"donation_like_updated_at"`
	DonationLikeMasjidID  *uuid.UUID `gorm:"type:uuid" json:"donation_like_masjid_id"`

	// Optional: Relasi
	// Donation DonationModel `gorm:"foreignKey:DonationLikeDonationID" json:"-"`
	// User     UserModel     `gorm:"foreignKey:DonationLikeUserID" json:"-"`
	// Masjid   MasjidModel   `gorm:"foreignKey:DonationLikeMasjidID" json:"-"`
}

func (DonationLikeModel) TableName() string {
	return "donation_likes"
}