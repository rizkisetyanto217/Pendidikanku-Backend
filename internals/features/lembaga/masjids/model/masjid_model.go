// file: internals/features/masjids/model/masjid_model.go
package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Enum helper agar konsisten dengan tipe verification_status_enum di DB
type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

// MasjidModel merepresentasikan tabel masjids (versi tanpa koordinat)
type MasjidModel struct {
	// PK
	MasjidID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:masjid_id" json:"masjid_id"`

	// Relasi
	MasjidYayasanID     *uuid.UUID `gorm:"type:uuid;column:masjid_yayasan_id" json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `gorm:"type:uuid;column:masjid_current_plan_id" json:"masjid_current_plan_id,omitempty"`

	// Identitas & lokasi ringkas
	MasjidName     string  `gorm:"type:varchar(100);not null;column:masjid_name" json:"masjid_name"`
	MasjidBioShort *string `gorm:"type:text;column:masjid_bio_short" json:"masjid_bio_short,omitempty"`
	MasjidLocation *string `gorm:"type:text;column:masjid_location" json:"masjid_location,omitempty"`

	// Domain & Slug
	MasjidDomain *string `gorm:"type:varchar(50);column:masjid_domain" json:"masjid_domain,omitempty"`
	MasjidSlug   string  `gorm:"type:varchar(100);uniqueIndex;not null;column:masjid_slug" json:"masjid_slug"`

	// Status & Verifikasi
	MasjidIsActive           bool               `gorm:"not null;default:true;column:masjid_is_active" json:"masjid_is_active"`
	MasjidIsVerified         bool               `gorm:"not null;default:false;column:masjid_is_verified" json:"masjid_is_verified"`
	MasjidVerificationStatus VerificationStatus `gorm:"type:verification_status_enum;default:'pending';column:masjid_verification_status" json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time         `gorm:"column:masjid_verified_at" json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  *string            `gorm:"type:text;column:masjid_verification_notes" json:"masjid_verification_notes,omitempty"`

	// Flag sekolah/pesantren
	MasjidIsIslamicSchool bool `gorm:"not null;default:false;column:masjid_is_islamic_school" json:"masjid_is_islamic_school"`

	// Levels (JSONB tags)
	MasjidLevels datatypes.JSON `gorm:"type:jsonb;column:masjid_levels" json:"masjid_levels,omitempty"`

	// Full-text search (generated; read-only)
	MasjidSearch string `gorm:"type:tsvector;->;<-:false;column:masjid_search" json:"masjid_search,omitempty"`

	// Audit
	MasjidCreatedAt time.Time      `gorm:"column:masjid_created_at;autoCreateTime" json:"masjid_created_at"`
	MasjidUpdatedAt time.Time      `gorm:"column:masjid_updated_at;autoUpdateTime"  json:"masjid_updated_at"`
	MasjidDeletedAt gorm.DeletedAt `gorm:"column:masjid_deleted_at;index"           json:"masjid_deleted_at,omitempty"`
}

func (MasjidModel) TableName() string { return "masjids" }

// -------- Helpers opsional untuk MasjidLevels (JSONB) --------

// SetLevels mengisi masjid_levels dari slice string (mis. ["kursus","ilmu_quran"])
func (m *MasjidModel) SetLevels(levels []string) error {
	b, err := json.Marshal(levels)
	if err != nil {
		return err
	}
	m.MasjidLevels = datatypes.JSON(b)
	return nil
}

// GetLevels mengembalikan masjid_levels sebagai slice string
func (m *MasjidModel) GetLevels() ([]string, error) {
	if len(m.MasjidLevels) == 0 {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal(m.MasjidLevels, &out); err != nil {
		return nil, err
	}
	return out, nil
}
