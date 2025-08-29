package model

import (
	"time"

	// Masjid "masjidku_backend/internals/features/masjids/masjids/model"
	// User "masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
)

type UserFollowMasjidModel struct {
	// PK komposit
	UserFollowMasjidUserID   uuid.UUID `gorm:"column:user_follow_masjid_user_id;type:uuid;not null;primaryKey"   json:"user_follow_masjid_user_id"`
	UserFollowMasjidMasjidID uuid.UUID `gorm:"column:user_follow_masjid_masjid_id;type:uuid;not null;primaryKey" json:"user_follow_masjid_masjid_id"`

	// Timestamp
	UserFollowMasjidCreatedAt time.Time `gorm:"column:user_follow_masjid_created_at;not null;autoCreateTime" json:"user_follow_masjid_created_at"`

	// Relasi (opsional, kalau mau di-preload)
	// User   User.UserModel    `gorm:"foreignKey:UserFollowMasjidUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"         json:"user,omitempty"`
	// Masjid Masjid.MasjidModel`gorm:"foreignKey:UserFollowMasjidMasjidID;references:MasjidID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"masjid,omitempty"`
}

func (UserFollowMasjidModel) TableName() string {
	return "user_follow_masjid"
}
