package model

import (
	"time"

	Masjid "masjidku_backend/internals/features/masjids/masjids/model"
	User "masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
)

type MasjidAdminModel struct {
	MasjidAdminID uuid.UUID `gorm:"column:masjid_admins_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_admins_id"`

	MasjidID uuid.UUID          `gorm:"column:masjid_admins_masjid_id;type:uuid;not null;index" json:"masjid_admins_masjid_id"`
	Masjid   Masjid.MasjidModel `gorm:"foreignKey:MasjidID;references:MasjidID" json:"masjid,omitempty"`

	UserID uuid.UUID      `gorm:"column:masjid_admins_user_id;type:uuid;not null;index" json:"masjid_admins_user_id"`
	User   User.UserModel `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`

	IsActive  bool      `gorm:"column:masjid_admins_is_active;default:true" json:"masjid_admins_is_active"`
	CreatedAt time.Time `gorm:"column:created_at;default:current_timestamp" json:"created_at"`
}

func (MasjidAdminModel) TableName() string {
	return "masjid_admins"
}
