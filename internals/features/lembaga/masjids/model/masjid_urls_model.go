// file: internals/features/lembaga/masjids/model/masjid_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
   ENUM HELPER
   ========================= */

type MasjidURLType string

const (
	MasjidURLTypeLogo          MasjidURLType = "logo"
	MasjidURLTypeStempel       MasjidURLType = "stempel"
	MasjidURLTypeTTDKetua      MasjidURLType = "ttd_ketua"
	MasjidURLTypeBanner        MasjidURLType = "banner"
	MasjidURLTypeProfileCover  MasjidURLType = "profile_cover"
	MasjidURLTypeGallery       MasjidURLType = "gallery"
	MasjidURLTypeQR            MasjidURLType = "qr"
	MasjidURLTypeOther         MasjidURLType = "other"
	MasjidURLTypeBgBehindMain  MasjidURLType = "bg_behind_main"
	MasjidURLTypeMain          MasjidURLType = "main"
	MasjidURLTypeLinktreeBg    MasjidURLType = "linktree_bg"
)

/* =========================
   MODEL
   ========================= */

type MasjidURL struct {
	MasjidURLID        uuid.UUID     `json:"masjid_url_id"        gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:masjid_url_id"`
	MasjidURLMasjidID  uuid.UUID     `json:"masjid_url_masjid_id" gorm:"type:uuid;not null;column:masjid_url_masjid_id"`
	MasjidURLType      MasjidURLType `json:"masjid_url_type"      gorm:"type:masjid_url_type_enum;not null;column:masjid_url_type"`
	MasjidURLFileURL   string        `json:"masjid_url_file_url"  gorm:"type:text;not null;column:masjid_url_file_url"`
	MasjidURLIsPrimary bool          `json:"masjid_url_is_primary" gorm:"not null;default:false;column:masjid_url_is_primary"`
	MasjidURLIsActive  bool          `json:"masjid_url_is_active"  gorm:"not null;default:true;column:masjid_url_is_active"`

	MasjidURLCreatedAt time.Time      `json:"masjid_url_created_at" gorm:"column:masjid_url_created_at;autoCreateTime"`
	MasjidURLUpdatedAt time.Time      `json:"masjid_url_updated_at" gorm:"column:masjid_url_updated_at;autoUpdateTime"`
	MasjidURLDeletedAt gorm.DeletedAt `json:"masjid_url_deleted_at" gorm:"column:masjid_url_deleted_at;index"`
}

// Pastikan nama tabel sesuai
func (MasjidURL) TableName() string { return "masjid_urls" }

/*
Optional: relasi ke Masjid (pakai model yang sudah kamu punya)
----------------------------------------------------------------
type Masjid struct {
	MasjidID   uuid.UUID `gorm:"column:masjid_id;type:uuid;primaryKey"`
	MasjidName string    `gorm:"column:masjid_name"`
}
func (Masjid) TableName() string { return "masjids" }

func (m *MasjidURL) Masjid() *Masjid {
	return nil // gunakan association GORM di tempat lain jika perlu
}
// Atau pakai field association:
//   Masjid *Masjid `gorm:"foreignKey:MasjidURLMasjidID;references:MasjidID" json:"masjid,omitempty"`
*/
