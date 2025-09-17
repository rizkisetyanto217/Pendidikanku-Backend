// file: internals/features/masjids/model/masjid_model.go
package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ===== Enums (mirror dari DB) =====

type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

type TenantProfile string

const (
	TenantTeacherSolo       TenantProfile = "teacher_solo"
	TenantTeacherPlusSchool TenantProfile = "teacher_plus_school"
	TenantSchoolBasic       TenantProfile = "school_basic"
	TenantSchoolComplex     TenantProfile = "school_complex"
)

// ===== Model =====

// MasjidModel merepresentasikan tabel masjids (versi sesuai DDL terbaru)
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
	MasjidCity     *string `gorm:"type:varchar(80);column:masjid_city" json:"masjid_city,omitempty"`

	// Domain & Slug
	// Catatan: Domain unik case-insensitive via UNIQUE INDEX LOWER(masjid_domain) di DB.
	MasjidDomain *string `gorm:"type:varchar(50);column:masjid_domain" json:"masjid_domain,omitempty"`
	MasjidSlug   string  `gorm:"type:varchar(100);uniqueIndex;not null;column:masjid_slug" json:"masjid_slug"`

	// Status & Verifikasi
	MasjidIsActive           bool               `gorm:"not null;default:true;column:masjid_is_active" json:"masjid_is_active"`
	MasjidIsVerified         bool               `gorm:"not null;default:false;column:masjid_is_verified" json:"masjid_is_verified"`
	MasjidVerificationStatus VerificationStatus `gorm:"type:verification_status_enum;not null;default:'pending';column:masjid_verification_status" json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time         `gorm:"column:masjid_verified_at" json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  *string            `gorm:"type:text;column:masjid_verification_notes" json:"masjid_verification_notes,omitempty"`

	// Kontak & flag
	MasjidContactPersonName  *string `gorm:"type:varchar(100);column:masjid_contact_person_name" json:"masjid_contact_person_name,omitempty"`
	MasjidContactPersonPhone *string `gorm:"type:varchar(30);column:masjid_contact_person_phone" json:"masjid_contact_person_phone,omitempty"`
	MasjidIsIslamicSchool    bool    `gorm:"not null;default:false;column:masjid_is_islamic_school" json:"masjid_is_islamic_school"`

	// Peruntukan tenant (enum)
	MasjidTenantProfile TenantProfile `gorm:"type:tenant_profile_enum;not null;default:'teacher_solo';column:masjid_tenant_profile" json:"masjid_tenant_profile"`

	// Levels (JSONB tags) — pointer agar bisa NULL
	MasjidLevels *datatypes.JSON `gorm:"type:jsonb;column:masjid_levels" json:"masjid_levels,omitempty"`

	// ===== Media: LOGO (2-slot + retensi hapus) =====
	MasjidLogoURL                *string    `gorm:"type:text;column:masjid_logo_url" json:"masjid_logo_url,omitempty"`
	MasjidLogoObjectKey          *string    `gorm:"type:text;column:masjid_logo_object_key" json:"masjid_logo_object_key,omitempty"`
	MasjidLogoURLOld             *string    `gorm:"type:text;column:masjid_logo_url_old" json:"masjid_logo_url_old,omitempty"`
	MasjidLogoObjectKeyOld       *string    `gorm:"type:text;column:masjid_logo_object_key_old" json:"masjid_logo_object_key_old,omitempty"`
	MasjidLogoDeletePendingUntil *time.Time `gorm:"column:masjid_logo_delete_pending_until" json:"masjid_logo_delete_pending_until,omitempty"`

	// ===== Media: BACKGROUND (2-slot + retensi hapus) =====
	MasjidBackgroundURL                *string    `gorm:"type:text;column:masjid_background_url" json:"masjid_background_url,omitempty"`
	MasjidBackgroundObjectKey          *string    `gorm:"type:text;column:masjid_background_object_key" json:"masjid_background_object_key,omitempty"`
	MasjidBackgroundURLOld             *string    `gorm:"type:text;column:masjid_background_url_old" json:"masjid_background_url_old,omitempty"`
	MasjidBackgroundObjectKeyOld       *string    `gorm:"type:text;column:masjid_background_object_key_old" json:"masjid_background_object_key_old,omitempty"`
	MasjidBackgroundDeletePendingUntil *time.Time `gorm:"column:masjid_background_delete_pending_until" json:"masjid_background_delete_pending_until,omitempty"`

	// Audit
	MasjidCreatedAt      time.Time      `gorm:"column:masjid_created_at;autoCreateTime" json:"masjid_created_at"`
	MasjidUpdatedAt      time.Time      `gorm:"column:masjid_updated_at;autoUpdateTime"  json:"masjid_updated_at"`
	MasjidLastActivityAt *time.Time     `gorm:"column:masjid_last_activity_at"           json:"masjid_last_activity_at,omitempty"`
	MasjidDeletedAt      gorm.DeletedAt `gorm:"column:masjid_deleted_at;index"           json:"masjid_deleted_at,omitempty"`
}

func (MasjidModel) TableName() string { return "masjids" }

// -------- Helpers opsional untuk MasjidLevels (JSONB) --------

// SetLevels mengisi masjid_levels dari slice string; jika kosong → NULL
func (m *MasjidModel) SetLevels(levels []string) error {
	if len(levels) == 0 {
		m.MasjidLevels = nil
		return nil
	}
	b, err := json.Marshal(levels)
	if err != nil {
		return err
	}
	val := datatypes.JSON(b)
	m.MasjidLevels = &val
	return nil
}

// GetLevels mengembalikan masjid_levels sebagai slice string (kosong jika NULL)
func (m *MasjidModel) GetLevels() ([]string, error) {
	if m.MasjidLevels == nil || len(*m.MasjidLevels) == 0 {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal(*m.MasjidLevels, &out); err != nil {
		return nil, err
	}
	return out, nil
}
