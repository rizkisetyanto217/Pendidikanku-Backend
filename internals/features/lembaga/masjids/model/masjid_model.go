// file: internals/features/masjids/model/masjid_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// (Opsional) Enum helper supaya konsisten di kode
type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

// MasjidModel merepresentasikan tabel masjids
type MasjidModel struct {
	// PK
	MasjidID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"masjid_id"`

	// Relasi
	MasjidYayasanID    *uuid.UUID `gorm:"type:uuid" json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `gorm:"type:uuid" json:"masjid_current_plan_id,omitempty"`

	// Identitas & lokasi
	MasjidName     string   `gorm:"type:varchar(100);not null" json:"masjid_name"`
	MasjidBioShort *string  `gorm:"type:text" json:"masjid_bio_short,omitempty"`
	MasjidLocation *string  `gorm:"type:text" json:"masjid_location,omitempty"`
	MasjidLatitude *float64 `gorm:"type:decimal(9,6)" json:"masjid_latitude,omitempty"`
	MasjidLongitude *float64 `gorm:"type:decimal(9,6)" json:"masjid_longitude,omitempty"`

	// Media (default)
	MasjidImageURL               *string    `gorm:"type:text" json:"masjid_image_url,omitempty"`
	MasjidImageTrashURL          *string    `gorm:"type:text" json:"masjid_image_trash_url,omitempty"`
	MasjidImageDeletePendingUntil *time.Time `gorm:"type:timestamp" json:"masjid_image_delete_pending_until,omitempty"`

	// Media (MAIN)
	MasjidImageMainURL               *string    `gorm:"column:masjid_image_main_url;type:text" json:"masjid_image_main_url,omitempty"`
	MasjidImageMainTrashURL          *string    `gorm:"column:masjid_image_main_trash_url;type:text" json:"masjid_image_main_trash_url,omitempty"`
	MasjidImageMainDeletePendingUntil *time.Time `gorm:"column:masjid_image_main_delete_pending_until;type:timestamp" json:"masjid_image_main_delete_pending_until,omitempty"`

	// Media (BACKGROUND)
	MasjidImageBgURL               *string    `gorm:"column:masjid_image_bg_url;type:text" json:"masjid_image_bg_url,omitempty"`
	MasjidImageBgTrashURL          *string    `gorm:"column:masjid_image_bg_trash_url;type:text" json:"masjid_image_bg_trash_url,omitempty"`
	MasjidImageBgDeletePendingUntil *time.Time `gorm:"column:masjid_image_bg_delete_pending_until;type:timestamp" json:"masjid_image_bg_delete_pending_until,omitempty"`

	// Maps & sosial
	MasjidGoogleMapsURL          *string `gorm:"type:text" json:"masjid_google_maps_url,omitempty"`
	MasjidInstagramURL           *string `gorm:"type:text" json:"masjid_instagram_url,omitempty"`
	MasjidWhatsappURL            *string `gorm:"type:text" json:"masjid_whatsapp_url,omitempty"`
	MasjidYoutubeURL             *string `gorm:"type:text" json:"masjid_youtube_url,omitempty"`
	MasjidFacebookURL            *string `gorm:"type:text" json:"masjid_facebook_url,omitempty"`
	MasjidTiktokURL              *string `gorm:"type:text" json:"masjid_tiktok_url,omitempty"`
	MasjidWhatsappGroupIkhwanURL *string `gorm:"type:text" json:"masjid_whatsapp_group_ikhwan_url,omitempty"`
	MasjidWhatsappGroupAkhwatURL *string `gorm:"type:text" json:"masjid_whatsapp_group_akhwat_url,omitempty"`

	// Domain & Slug
	MasjidDomain *string `gorm:"type:varchar(50)" json:"masjid_domain,omitempty"`
	MasjidSlug   string  `gorm:"type:varchar(100);uniqueIndex;not null" json:"masjid_slug"`

	// Status & Verifikasi
	MasjidIsActive           bool               `gorm:"not null;default:true" json:"masjid_is_active"`
	MasjidIsVerified         bool               `gorm:"not null;default:false" json:"masjid_is_verified"`
	MasjidVerificationStatus VerificationStatus `gorm:"type:verification_status_enum;default:'pending'" json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time         `json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  *string            `gorm:"type:text" json:"masjid_verification_notes,omitempty"`

	// Flag sekolah/pesantren
	MasjidIsIslamicSchool bool `gorm:"not null;default:false" json:"masjid_is_islamic_school"`

	// Full-text search (generated column; read-only)
	MasjidSearch string `gorm:"type:tsvector;->;<-:false" json:"masjid_search,omitempty"`

	// Audit
	MasjidCreatedAt time.Time      `gorm:"column:masjid_created_at;autoCreateTime" json:"masjid_created_at"`
	MasjidUpdatedAt time.Time      `gorm:"column:masjid_updated_at;autoUpdateTime" json:"masjid_updated_at"`
	MasjidDeletedAt gorm.DeletedAt `gorm:"column:masjid_deleted_at;index" json:"masjid_deleted_at,omitempty"`
}

func (MasjidModel) TableName() string { return "masjids" }
