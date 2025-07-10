package model

import (
	"time"

	"github.com/google/uuid"
	User "masjidku_backend/internals/features/users/user/model"
	Masjid "masjidku_backend/internals/features/masjids/masjids/model"
)

type MasjidProfileTeacherDkmModel struct {
	MasjidProfileTeacherDkmID        uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_profile_teacher_dkm_id"`
	MasjidProfileTeacherDkmMasjidID  uuid.UUID           `gorm:"type:uuid;not null;index" json:"masjid_profile_teacher_dkm_masjid_id"`
	Masjid                           Masjid.MasjidModel  `gorm:"foreignKey:MasjidProfileTeacherDkmMasjidID;references:MasjidID" json:"masjid,omitempty"`

	MasjidProfileTeacherDkmUserID    *uuid.UUID          `gorm:"type:uuid;index" json:"masjid_profile_teacher_dkm_user_id,omitempty"`
	User                             *User.UserModel     `gorm:"foreignKey:MasjidProfileTeacherDkmUserID;references:ID" json:"user,omitempty"`

	MasjidProfileTeacherDkmName      string              `gorm:"type:varchar(100);not null" json:"masjid_profile_teacher_dkm_name"`
	MasjidProfileTeacherDkmRole      string              `gorm:"type:varchar(100);not null" json:"masjid_profile_teacher_dkm_role"`
	MasjidProfileTeacherDkmDescription string            `gorm:"type:text" json:"masjid_profile_teacher_dkm_description"`
	MasjidProfileTeacherDkmMessage   string              `gorm:"type:text" json:"masjid_profile_teacher_dkm_message"`
	MasjidProfileTeacherDkmImageURL  string              `gorm:"type:text" json:"masjid_profile_teacher_dkm_image_url"`
	MasjidProfileTeacherDkmCreatedAt time.Time           `gorm:"default:current_timestamp" json:"masjid_profile_teacher_dkm_created_at"`
}

func (MasjidProfileTeacherDkmModel) TableName() string {
	return "masjid_profile_teacher_dkm"
}
