// file: internals/features/masjid/model/masjid_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   Enums (mapped as string)
   — nilai dijaga oleh ENUM & CHECK di DB
========================= */

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

/* =========================
   Masjid model
========================= */

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

	// Domain & slug
	MasjidDomain *string `gorm:"type:varchar(50);column:masjid_domain" json:"masjid_domain,omitempty"`
	MasjidSlug   string  `gorm:"type:varchar(100);uniqueIndex;not null;column:masjid_slug" json:"masjid_slug"`

	// Status & verifikasi
	MasjidIsActive           bool               `gorm:"type:boolean;not null;default:true;column:masjid_is_active" json:"masjid_is_active"`
	MasjidIsVerified         bool               `gorm:"type:boolean;not null;default:false;column:masjid_is_verified" json:"masjid_is_verified"`
	MasjidVerificationStatus VerificationStatus `gorm:"type:verification_status_enum;not null;default:'pending';column:masjid_verification_status" json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time         `gorm:"type:timestamptz;column:masjid_verified_at" json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  *string            `gorm:"type:text;column:masjid_verification_notes" json:"masjid_verification_notes,omitempty"`

	// Kontak & admin
	MasjidContactPersonName  *string `gorm:"type:varchar(100);column:masjid_contact_person_name" json:"masjid_contact_person_name,omitempty"`
	MasjidContactPersonPhone *string `gorm:"type:varchar(30);column:masjid_contact_person_phone" json:"masjid_contact_person_phone,omitempty"`

	// Flag
	MasjidIsIslamicSchool bool `gorm:"type:boolean;not null;default:false;column:masjid_is_islamic_school" json:"masjid_is_islamic_school"`

	// Peruntukan tenant
	MasjidTenantProfile TenantProfile `gorm:"type:tenant_profile_enum;not null;default:'teacher_solo';column:masjid_tenant_profile" json:"masjid_tenant_profile"`

	// Levels (JSONB array/tag-style)
	MasjidLevels datatypes.JSON `gorm:"type:jsonb;column:masjid_levels" json:"masjid_levels,omitempty"`
	// Alternatif yang lebih ketat (butuh GORM dengan JSONType):
	// MasjidLevels datatypes.JSONType[[]string] `gorm:"type:jsonb;column:masjid_levels" json:"masjid_levels,omitempty"`

	// Media: icon (2-slot + retensi)
	MasjidIconURL                *string    `gorm:"type:text;column:masjid_icon_url" json:"masjid_icon_url,omitempty"`
	MasjidIconObjectKey          *string    `gorm:"type:text;column:masjid_icon_object_key" json:"masjid_icon_object_key,omitempty"`
	MasjidIconURLOld             *string    `gorm:"type:text;column:masjid_icon_url_old" json:"masjid_icon_url_old,omitempty"`
	MasjidIconObjectKeyOld       *string    `gorm:"type:text;column:masjid_icon_object_key_old" json:"masjid_icon_object_key_old,omitempty"`
	MasjidIconDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:masjid_icon_delete_pending_until" json:"masjid_icon_delete_pending_until,omitempty"`

	// Media: logo (2-slot + retensi)
	MasjidLogoURL                *string    `gorm:"type:text;column:masjid_logo_url" json:"masjid_logo_url,omitempty"`
	MasjidLogoObjectKey          *string    `gorm:"type:text;column:masjid_logo_object_key" json:"masjid_logo_object_key,omitempty"`
	MasjidLogoURLOld             *string    `gorm:"type:text;column:masjid_logo_url_old" json:"masjid_logo_url_old,omitempty"`
	MasjidLogoObjectKeyOld       *string    `gorm:"type:text;column:masjid_logo_object_key_old" json:"masjid_logo_object_key_old,omitempty"`
	MasjidLogoDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:masjid_logo_delete_pending_until" json:"masjid_logo_delete_pending_until,omitempty"`

	// Media: background (2-slot + retensi)
	MasjidBackgroundURL                *string    `gorm:"type:text;column:masjid_background_url" json:"masjid_background_url,omitempty"`
	MasjidBackgroundObjectKey          *string    `gorm:"type:text;column:masjid_background_object_key" json:"masjid_background_object_key,omitempty"`
	MasjidBackgroundURLOld             *string    `gorm:"type:text;column:masjid_background_url_old" json:"masjid_background_url_old,omitempty"`
	MasjidBackgroundObjectKeyOld       *string    `gorm:"type:text;column:masjid_background_object_key_old" json:"masjid_background_object_key_old,omitempty"`
	MasjidBackgroundDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:masjid_background_delete_pending_until" json:"masjid_background_delete_pending_until,omitempty"`

	// Audit & soft delete
	MasjidCreatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:masjid_created_at" json:"masjid_created_at"`
	MasjidUpdatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:masjid_updated_at" json:"masjid_updated_at"`
	MasjidLastActivityAt *time.Time     `gorm:"type:timestamptz;column:masjid_last_activity_at" json:"masjid_last_activity_at,omitempty"`
	MasjidDeletedAt      gorm.DeletedAt `gorm:"index;column:masjid_deleted_at" json:"masjid_deleted_at,omitempty"`
}

func (MasjidModel) TableName() string { return "masjids" }
