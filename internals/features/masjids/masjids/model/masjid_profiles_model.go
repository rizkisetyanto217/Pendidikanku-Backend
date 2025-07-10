package model

import (
	"time"

	"github.com/google/uuid"

)

type MasjidProfileModel struct {
	MasjidProfileID          uint       `gorm:"primaryKey;column:masjid_profile_id" json:"masjid_profile_id"`
	MasjidProfileStory       string     `gorm:"type:text;column:masjid_profile_story" json:"masjid_profile_story"`
	MasjidProfileVisi        string     `gorm:"type:text;column:masjid_profile_visi" json:"masjid_profile_visi"`
	MasjidProfileMisi        string     `gorm:"type:text;column:masjid_profile_misi" json:"masjid_profile_misi"`
	MasjidProfileOther       string     `gorm:"type:text;column:masjid_profile_other" json:"masjid_profile_other"`
	MasjidProfileFoundedYear int        `gorm:"type:int;column:masjid_profile_founded_year" json:"masjid_profile_founded_year"`
	MasjidProfileMasjidID    uuid.UUID  `gorm:"type:uuid;unique;column:masjid_profile_masjid_id" json:"masjid_profile_masjid_id"`
	MasjidProfileCreatedAt   time.Time  `gorm:"autoCreateTime;column:masjid_profile_created_at" json:"masjid_profile_created_at"`
	MasjidProfileUpdatedAt   time.Time  `gorm:"autoUpdateTime;column:masjid_profile_updated_at" json:"masjid_profile_updated_at"`
	MasjidProfileDeletedAt   *time.Time `gorm:"column:masjid_profile_deleted_at" json:"masjid_profile_deleted_at,omitempty"`

}

func (MasjidProfileModel) TableName() string {
	return "masjids_profiles"
}
