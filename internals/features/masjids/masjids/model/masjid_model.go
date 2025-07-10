package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidModel struct {
	MasjidID           uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"masjid_id"`
	MasjidName         string         `gorm:"type:varchar(100);not null" json:"masjid_name"`
	MasjidBioShort     string         `gorm:"type:text" json:"masjid_bio_short"`
	MasjidLocation     string         `gorm:"type:text" json:"masjid_location"`
	MasjidLatitude     float64        `gorm:"type:decimal(9,6)" json:"masjid_latitude"`
	MasjidLongitude    float64        `gorm:"type:decimal(9,6)" json:"masjid_longitude"`
	MasjidImageURL     string         `gorm:"type:text" json:"masjid_image_url"`
	MasjidSlug         string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"masjid_slug"`
	MasjidIsVerified   bool           `gorm:"default:false" json:"masjid_is_verified"`
	MasjidInstagramURL string         `gorm:"type:text" json:"masjid_instagram_url"`
	MasjidWhatsappURL  string         `gorm:"type:text" json:"masjid_whatsapp_url"`
	MasjidYoutubeURL   string         `gorm:"type:text" json:"masjid_youtube_url"`
	MasjidCreatedAt    time.Time      `gorm:"autoCreateTime" json:"masjid_created_at"`
	MasjidUpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"masjid_updated_at"`
	MasjidDeletedAt    gorm.DeletedAt `gorm:"column:masjid_deleted_at" json:"masjid_deleted_at,omitempty"`
}

func (MasjidModel) TableName() string {
	return "masjids"
}
