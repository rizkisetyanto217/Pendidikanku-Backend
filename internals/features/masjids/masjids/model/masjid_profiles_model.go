package model

import (
	"time"

	"github.com/google/uuid"
)

type MasjidProfileModel struct {
	MasjidProfileID             uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"masjid_profile_id"`
	MasjidProfileStory         string     `gorm:"type:text;column:masjid_profile_story" json:"masjid_profile_story"`
	MasjidProfileVisi          string     `gorm:"type:text;column:masjid_profile_visi" json:"masjid_profile_visi"`
	MasjidProfileMisi          string     `gorm:"type:text;column:masjid_profile_misi" json:"masjid_profile_misi"`
	MasjidProfileOther         string     `gorm:"type:text;column:masjid_profile_other" json:"masjid_profile_other"`
	MasjidProfileFoundedYear   int        `gorm:"type:int;column:masjid_profile_founded_year" json:"masjid_profile_founded_year"`
	MasjidProfileMasjidID      uuid.UUID  `gorm:"type:uuid;unique;column:masjid_profile_masjid_id" json:"masjid_profile_masjid_id"`
	MasjidProfileLogoURL       string     `gorm:"type:text;column:masjid_profile_logo_url" json:"masjid_profile_logo_url"`
	MasjidProfileStampURL      string     `gorm:"type:text;column:masjid_profile_stamp_url" json:"masjid_profile_stamp_url"`
	MasjidProfileTTDKetuaDKMURL string    `gorm:"type:text;column:masjid_profile_ttd_ketua_dkm_url" json:"masjid_profile_ttd_ketua_dkm_url"`
	MasjidProfileCreatedAt     time.Time  `gorm:"autoCreateTime;column:masjid_profile_created_at" json:"masjid_profile_created_at"`
	MasjidProfileUpdatedAt     time.Time  `gorm:"autoUpdateTime;column:masjid_profile_updated_at" json:"masjid_profile_updated_at"`
	MasjidProfileDeletedAt     *time.Time `gorm:"column:masjid_profile_deleted_at" json:"masjid_profile_deleted_at,omitempty"`
}

func (MasjidProfileModel) TableName() string {
	return "masjids_profiles"
}