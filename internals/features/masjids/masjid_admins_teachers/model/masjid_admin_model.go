package model

import (
	"time"

	Masjid "masjidku_backend/internals/features/masjids/masjids/model"
	User "masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidAdminModel struct {
	// PK
	MasjidAdminsID uuid.UUID `gorm:"column:masjid_admins_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_admins_id"`

	// FK (kolom sama persis dengan SQL)
	MasjidAdminsMasjidID uuid.UUID `gorm:"column:masjid_admins_masjid_id;type:uuid;not null;index" json:"masjid_admins_masjid_id"`
	MasjidAdminsUserID   uuid.UUID `gorm:"column:masjid_admins_user_id;type:uuid;not null;index"   json:"masjid_admins_user_id"`

	// Relasi (gunakan nama field FK di atas)
	Masjid Masjid.MasjidModel `gorm:"foreignKey:MasjidAdminsMasjidID;references:MasjidID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"masjid,omitempty"`
	User   User.UserModel      `gorm:"foreignKey:MasjidAdminsUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"      json:"user,omitempty"`

	// Status
	MasjidAdminsIsActive bool `gorm:"column:masjid_admins_is_active;not null;default:true;index" json:"masjid_admins_is_active"`

	// Timestamps (eksplisit sesuai migrasi)
	MasjidAdminCreatedAt time.Time      `gorm:"column:masjid_admin_created_at;autoCreateTime" json:"masjid_admin_created_at"`
	MasjidAdminUpdatedAt time.Time      `gorm:"column:masjid_admin_updated_at;autoUpdateTime" json:"masjid_admin_updated_at"`
	MasjidAdminDeletedAt gorm.DeletedAt `gorm:"column:masjid_admin_deleted_at;index"          json:"masjid_admin_deleted_at,omitempty"`
}

func (MasjidAdminModel) TableName() string {
	return "masjid_admins"
}
