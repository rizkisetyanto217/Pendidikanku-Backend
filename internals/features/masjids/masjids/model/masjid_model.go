package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MasjidModel merepresentasikan tabel masjids
type MasjidModel struct {
	MasjidID                    uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"masjid_id"`
	MasjidName                  string          `gorm:"type:varchar(100);not null" json:"masjid_name"`
	MasjidBioShort              string          `gorm:"type:text" json:"masjid_bio_short"`
	MasjidLocation              string          `gorm:"type:text" json:"masjid_location"`
	MasjidLatitude              *float64        `gorm:"type:decimal(9,6)" json:"masjid_latitude,omitempty"`
	MasjidLongitude             *float64        `gorm:"type:decimal(9,6)" json:"masjid_longitude,omitempty"`

	// Media & Maps
	MasjidImageURL              string     `gorm:"type:text" json:"masjid_image_url"`
	MasjidImageTrashURL         *string    `gorm:"type:text" json:"masjid_image_trash_url,omitempty"`
	MasjidImageDeletePendingUntil *time.Time `gorm:"type:timestamp" json:"masjid_image_delete_pending_until,omitempty"`
	MasjidGoogleMapsURL         string     `gorm:"type:text" json:"masjid_google_maps_url"`

	// Domain & Slug
	MasjidDomain                *string    `gorm:"type:varchar(50)" json:"masjid_domain,omitempty"`
	MasjidSlug                  string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"masjid_slug"`

	// Status & Verifikasi
	MasjidIsActive              bool       `gorm:"not null;default:true" json:"masjid_is_active"`
	MasjidIsVerified            bool       `gorm:"not null;default:false" json:"masjid_is_verified"`
	MasjidVerificationStatus    string     `gorm:"type:verification_status_enum;default:'pending'" json:"masjid_verification_status"`
	MasjidVerifiedAt            *time.Time `json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes     string     `gorm:"type:text" json:"masjid_verification_notes"`

	// Paket aktif (relasi ke masjid_service_plans)
	MasjidCurrentPlanID         *uuid.UUID `gorm:"type:uuid" json:"masjid_current_plan_id,omitempty"`

	// Sosial Media
	MasjidInstagramURL          string `gorm:"type:text" json:"masjid_instagram_url"`
	MasjidWhatsappURL           string `gorm:"type:text" json:"masjid_whatsapp_url"`
	MasjidYoutubeURL            string `gorm:"type:text" json:"masjid_youtube_url"`
	MasjidFacebookURL           string `gorm:"type:text" json:"masjid_facebook_url"`
	MasjidTiktokURL             string `gorm:"type:text" json:"masjid_tiktok_url"`
	MasjidWhatsappGroupIkhwanURL string `gorm:"type:text" json:"masjid_whatsapp_group_ikhwan_url"`
	MasjidWhatsappGroupAkhwatURL string `gorm:"type:text" json:"masjid_whatsapp_group_akhwat_url"`

	// Full-text search gabungan
	MasjidSearch                string `gorm:"type:tsvector;->;<-:false" json:"masjid_search,omitempty"`

	// Audit
	MasjidCreatedAt             time.Time      `gorm:"autoCreateTime" json:"masjid_created_at"`
	MasjidUpdatedAt             time.Time      `gorm:"autoUpdateTime" json:"masjid_updated_at"`
	MasjidDeletedAt             gorm.DeletedAt `gorm:"column:masjid_deleted_at" json:"masjid_deleted_at,omitempty"`
}

func (MasjidModel) TableName() string {
	return "masjids"
}
