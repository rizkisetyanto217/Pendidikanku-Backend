package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	Masjid "masjidku_backend/internals/features/masjids/masjids/model"
	User "masjidku_backend/internals/features/users/user/model"
)

type MasjidProfileTeacherDkmModel struct {
	// PK
	MasjidProfileTeacherDkmID uuid.UUID `gorm:"column:masjid_profile_teacher_dkm_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_profile_teacher_dkm_id"`

	// FK ke masjids (NOT NULL, ON DELETE CASCADE)
	MasjidProfileTeacherDkmMasjidID uuid.UUID          `gorm:"column:masjid_profile_teacher_dkm_masjid_id;type:uuid;not null;index" json:"masjid_profile_teacher_dkm_masjid_id"`
	Masjid                          Masjid.MasjidModel `gorm:"foreignKey:MasjidProfileTeacherDkmMasjidID;references:MasjidID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"masjid,omitempty"`

	// FK ke users (NULLABLE, ON DELETE SET NULL)
	MasjidProfileTeacherDkmUserID *uuid.UUID        `gorm:"column:masjid_profile_teacher_dkm_user_id;type:uuid;index" json:"masjid_profile_teacher_dkm_user_id,omitempty"`
	User                          *User.UserModel   `gorm:"foreignKey:MasjidProfileTeacherDkmUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user,omitempty"`

	// Data profil
	MasjidProfileTeacherDkmName        string   `gorm:"column:masjid_profile_teacher_dkm_name;type:varchar(100);not null" json:"masjid_profile_teacher_dkm_name"`
	MasjidProfileTeacherDkmRole        string   `gorm:"column:masjid_profile_teacher_dkm_role;type:varchar(100);not null" json:"masjid_profile_teacher_dkm_role"`
	MasjidProfileTeacherDkmDescription *string  `gorm:"column:masjid_profile_teacher_dkm_description;type:text" json:"masjid_profile_teacher_dkm_description,omitempty"`
	MasjidProfileTeacherDkmMessage     *string  `gorm:"column:masjid_profile_teacher_dkm_message;type:text" json:"masjid_profile_teacher_dkm_message,omitempty"`
	MasjidProfileTeacherDkmImageURL    *string  `gorm:"column:masjid_profile_teacher_dkm_image_url;type:text" json:"masjid_profile_teacher_dkm_image_url,omitempty"`

	// Timestamps
	MasjidProfileTeacherDkmCreatedAt time.Time      `gorm:"column:masjid_profile_teacher_dkm_created_at;not null;autoCreateTime" json:"masjid_profile_teacher_dkm_created_at"`
	MasjidProfileTeacherDkmUpdatedAt time.Time      `gorm:"column:masjid_profile_teacher_dkm_updated_at;not null;autoUpdateTime" json:"masjid_profile_teacher_dkm_updated_at"`
	MasjidProfileTeacherDkmDeletedAt gorm.DeletedAt `gorm:"column:masjid_profile_teacher_dkm_deleted_at;index" json:"masjid_profile_teacher_dkm_deleted_at,omitempty"`
}

func (MasjidProfileTeacherDkmModel) TableName() string {
	return "masjid_profile_teacher_dkm"
}
