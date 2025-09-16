// internals/features/lembaga/yayasans/model/yayasan_model.go
package model

import (
	"database/sql/driver"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
Verifikasi (mengikuti ENUM di DB):
- "pending"
- "approved"
- "rejected"
*/
type YayasanVerificationStatus string

const (
	YayasanVerificationPending  YayasanVerificationStatus = "pending"
	YayasanVerificationApproved YayasanVerificationStatus = "approved"
	YayasanVerificationRejected YayasanVerificationStatus = "rejected"
)

// Normalisasi lower-case saat scan/save (aman bila sumber mixed-case)
func (s *YayasanVerificationStatus) Scan(value any) error {
	switch v := value.(type) {
	case string:
		*s = YayasanVerificationStatus(strings.ToLower(strings.TrimSpace(v)))
	case []byte:
		*s = YayasanVerificationStatus(strings.ToLower(strings.TrimSpace(string(v))))
	case nil:
		*s = ""
	default:
		*s = YayasanVerificationStatus(strings.ToLower(strings.TrimSpace(v.(string))))
	}
	return nil
}
func (s YayasanVerificationStatus) Value() (driver.Value, error) {
	return string(YayasanVerificationStatus(strings.ToLower(strings.TrimSpace(string(s))))), nil
}


type YayasanModel struct {
	// PK
	YayasanID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:yayasan_id" json:"yayasan_id"`

	// Identitas
	YayasanName        string  `gorm:"type:varchar(150);not null;column:yayasan_name" json:"yayasan_name"`
	YayasanDescription *string `gorm:"column:yayasan_description" json:"yayasan_description,omitempty"`
	YayasanBio         *string `gorm:"column:yayasan_bio" json:"yayasan_bio,omitempty"`

	// Kontak & lokasi
	YayasanAddress   *string  `gorm:"column:yayasan_address" json:"yayasan_address,omitempty"`
	YayasanCity      *string  `gorm:"column:yayasan_city" json:"yayasan_city,omitempty"`
	YayasanProvince  *string  `gorm:"column:yayasan_province" json:"yayasan_province,omitempty"`
	YayasanLatitude  *float64 `gorm:"type:double precision;column:yayasan_latitude" json:"yayasan_latitude,omitempty"`
	YayasanLongitude *float64 `gorm:"type:double precision;column:yayasan_longitude" json:"yayasan_longitude,omitempty"`

	// Media & maps
	YayasanGoogleMapsURL *string `gorm:"column:yayasan_google_maps_url" json:"yayasan_google_maps_url,omitempty"`

	// Logo (single file, 2-slot + retensi 30 hari)
	YayasanLogoURL                 *string    `gorm:"column:yayasan_logo_url" json:"yayasan_logo_url,omitempty"`
	YayasanLogoObjectKey           *string    `gorm:"column:yayasan_logo_object_key" json:"yayasan_logo_object_key,omitempty"`
	YayasanLogoURLOld              *string    `gorm:"column:yayasan_logo_url_old" json:"yayasan_logo_url_old,omitempty"`
	YayasanLogoObjectKeyOld        *string    `gorm:"column:yayasan_logo_object_key_old" json:"yayasan_logo_object_key_old,omitempty"`
	YayasanLogoDeletePendingUntil  *time.Time `gorm:"column:yayasan_logo_delete_pending_until" json:"yayasan_logo_delete_pending_until,omitempty"`

	// Domain & slug
	YayasanDomain *string `gorm:"type:varchar(80);column:yayasan_domain" json:"yayasan_domain,omitempty"`
	YayasanSlug   string  `gorm:"type:varchar(120);unique;not null;column:yayasan_slug" json:"yayasan_slug"`

	// Status & verifikasi
	YayasanIsActive           bool                      `gorm:"not null;default:true;column:yayasan_is_active" json:"yayasan_is_active"`
	YayasanIsVerified         bool                      `gorm:"not null;default:false;column:yayasan_is_verified" json:"yayasan_is_verified"`
	YayasanVerificationStatus YayasanVerificationStatus `gorm:"type:verification_status_enum;not null;default:'pending';column:yayasan_verification_status" json:"yayasan_verification_status"`
	YayasanVerifiedAt         *time.Time                `gorm:"column:yayasan_verified_at" json:"yayasan_verified_at,omitempty"`
	YayasanVerificationNotes  *string                   `gorm:"column:yayasan_verification_notes" json:"yayasan_verification_notes,omitempty"`

	// Search (generated tsvector) – read-only di GORM
	YayasanSearch *string `gorm:"type:tsvector;column:yayasan_search;->" json:"-"`

	// Audit
	YayasanCreatedAt time.Time      `gorm:"column:yayasan_created_at;autoCreateTime" json:"yayasan_created_at"`
	YayasanUpdatedAt time.Time      `gorm:"column:yayasan_updated_at;autoUpdateTime"  json:"yayasan_updated_at"`
	YayasanDeletedAt gorm.DeletedAt `gorm:"column:yayasan_deleted_at;index"           json:"yayasan_deleted_at,omitempty"`
}

func (YayasanModel) TableName() string { return "yayasans" }
