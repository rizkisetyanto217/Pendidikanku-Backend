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
	MasjidAdminID uuid.UUID `gorm:"column:masjid_admin_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_admin_id"`

	// FK (kolom sama persis dgn SQL)
	MasjidAdminMasjidID uuid.UUID `gorm:"column:masjid_admin_masjid_id;type:uuid;not null;index" json:"masjid_admin_masjid_id"`
	MasjidAdminUserID   uuid.UUID `gorm:"column:masjid_admin_user_id;type:uuid;not null;index"   json:"masjid_admin_user_id"`

	// Relasi (foreignKey mengacu ke nama field di atas)
	Masjid Masjid.MasjidModel `gorm:"foreignKey:MasjidAdminMasjidID;references:MasjidID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"masjid,omitempty"`
	User   User.UserModel      `gorm:"foreignKey:MasjidAdminUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"        json:"user,omitempty"`

	// Status
	MasjidAdminIsActive bool `gorm:"column:masjid_admin_is_active;not null;default:true;index" json:"masjid_admin_is_active"`

	// Timestamps (ikut nama kolom SQL)
	MasjidAdminCreatedAt time.Time      `gorm:"column:masjid_admin_created_at;autoCreateTime" json:"masjid_admin_created_at"`
	MasjidAdminUpdatedAt time.Time      `gorm:"column:masjid_admin_updated_at;autoUpdateTime" json:"masjid_admin_updated_at"`
	MasjidAdminDeletedAt gorm.DeletedAt `gorm:"column:masjid_admin_deleted_at;index"          json:"masjid_admin_deleted_at,omitempty"`
}

// Nama tabel tetap plural sesuai SQL
func (MasjidAdminModel) TableName() string {
	return "masjid_admins"
}
