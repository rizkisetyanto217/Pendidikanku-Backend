// file: internals/features/school/model/school_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   Enums (mapped as string)
   — values enforced by ENUM & CHECK in DB
========================= */

type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

type TenantProfile string

const (
	TenantProfileStudent     TenantProfile = "student"
	TenantProfileTeacherSolo TenantProfile = "teacher_solo"
	TenantProfileTeacherPlus TenantProfile = "teacher_plus"
	TenantProfileSchoolBasic TenantProfile = "school_basic"
	TenantProfileSchoolPlus  TenantProfile = "school_plus"
)

// Go mapping for attendance_entry_mode_enum
type AttendanceEntryMode string

const (
	AttendanceEntryTeacherOnly AttendanceEntryMode = "teacher_only"
	AttendanceEntryStudentOnly AttendanceEntryMode = "student_only"
	AttendanceEntryBoth        AttendanceEntryMode = "both"
)

/* =========================
   School model
========================= */

type SchoolModel struct {
	// PK
	SchoolID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:school_id" json:"school_id"`

	// Running number (global sequence)
	SchoolNumber int64 `gorm:"type:bigint;column:school_number" json:"school_number"`

	// Relations
	SchoolYayasanID     *uuid.UUID `gorm:"type:uuid;column:school_yayasan_id" json:"school_yayasan_id,omitempty"`
	SchoolCurrentPlanID *uuid.UUID `gorm:"type:uuid;column:school_current_plan_id" json:"school_current_plan_id,omitempty"`

	// Identity & short location
	SchoolName     string  `gorm:"type:varchar(100);not null;column:school_name" json:"school_name"`
	SchoolBioShort *string `gorm:"type:text;column:school_bio_short" json:"school_bio_short,omitempty"`
	SchoolLocation *string `gorm:"type:text;column:school_location" json:"school_location,omitempty"`
	SchoolCity     *string `gorm:"type:varchar(80);column:school_city" json:"school_city,omitempty"`

	// Domain & slug
	SchoolDomain *string `gorm:"type:varchar(50);column:school_domain" json:"school_domain,omitempty"`
	SchoolSlug   string  `gorm:"type:varchar(100);not null;column:school_slug" json:"school_slug"`

	// Status & verification
	SchoolIsActive           bool               `gorm:"type:boolean;not null;default:true;column:school_is_active" json:"school_is_active"`
	SchoolIsVerified         bool               `gorm:"type:boolean;not null;default:false;column:school_is_verified" json:"school_is_verified"`
	SchoolVerificationStatus VerificationStatus `gorm:"type:verification_status_enum;not null;default:'pending';column:school_verification_status" json:"school_verification_status"`
	SchoolVerifiedAt         *time.Time         `gorm:"type:timestamptz;column:school_verified_at" json:"school_verified_at,omitempty"`
	SchoolVerificationNotes  *string            `gorm:"type:text;column:school_verification_notes" json:"school_verification_notes,omitempty"`

	// Contact & admin
	SchoolContactPersonName  *string `gorm:"type:varchar(100);column:school_contact_person_name" json:"school_contact_person_name,omitempty"`
	SchoolContactPersonPhone *string `gorm:"type:varchar(30);column:school_contact_person_phone" json:"school_contact_person_phone,omitempty"`

	// Flag
	SchoolIsIslamicSchool bool `gorm:"type:boolean;not null;default:false;column:school_is_islamic_school" json:"school_is_islamic_school"`

	// Tenant profile (sync with DB ENUM)
	SchoolTenantProfile TenantProfile `gorm:"type:tenant_profile_enum;not null;default:'school_basic';column:school_tenant_profile" json:"school_tenant_profile"`

	// Levels (JSONB array/tag-style)
	SchoolLevels datatypes.JSON `gorm:"type:jsonb;column:school_levels" json:"school_levels,omitempty"`

	// Teacher invite/join code
	SchoolTeacherCodeHash  []byte     `gorm:"type:bytea;column:school_teacher_code_hash" json:"school_teacher_code_hash,omitempty"`
	SchoolTeacherCodeSetAt *time.Time `gorm:"type:timestamptz;column:school_teacher_code_set_at" json:"school_teacher_code_set_at,omitempty"`

	// Media: icon (2-slot + retention)
	SchoolIconURL                *string    `gorm:"type:text;column:school_icon_url" json:"school_icon_url,omitempty"`
	SchoolIconObjectKey          *string    `gorm:"type:text;column:school_icon_object_key" json:"school_icon_object_key,omitempty"`
	SchoolIconURLOld             *string    `gorm:"type:text;column:school_icon_url_old" json:"school_icon_url_old,omitempty"`
	SchoolIconObjectKeyOld       *string    `gorm:"type:text;column:school_icon_object_key_old" json:"school_icon_object_key_old,omitempty"`
	SchoolIconDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:school_icon_delete_pending_until" json:"school_icon_delete_pending_until,omitempty"`

	// Media: logo (2-slot + retention)
	SchoolLogoURL                *string    `gorm:"type:text;column:school_logo_url" json:"school_logo_url,omitempty"`
	SchoolLogoObjectKey          *string    `gorm:"type:text;column:school_logo_object_key" json:"school_logo_object_key,omitempty"`
	SchoolLogoURLOld             *string    `gorm:"type:text;column:school_logo_url_old" json:"school_logo_url_old,omitempty"`
	SchoolLogoObjectKeyOld       *string    `gorm:"type:text;column:school_logo_object_key_old" json:"school_logo_object_key_old,omitempty"`
	SchoolLogoDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:school_logo_delete_pending_until" json:"school_logo_delete_pending_until,omitempty"`

	// Media: background (2-slot + retention)
	SchoolBackgroundURL                *string    `gorm:"type:text;column:school_background_url" json:"school_background_url,omitempty"`
	SchoolBackgroundObjectKey          *string    `gorm:"type:text;column:school_background_object_key" json:"school_background_object_key,omitempty"`
	SchoolBackgroundURLOld             *string    `gorm:"type:text;column:school_background_url_old" json:"school_background_url_old,omitempty"`
	SchoolBackgroundObjectKeyOld       *string    `gorm:"type:text;column:school_background_object_key_old" json:"school_background_object_key_old,omitempty"`
	SchoolBackgroundDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:school_background_delete_pending_until" json:"school_background_delete_pending_until,omitempty"`

	// Default attendance mode
	SchoolDefaultAttendanceEntryMode AttendanceEntryMode `gorm:"type:attendance_entry_mode_enum;not null;default:'both';column:school_default_attendance_entry_mode" json:"school_default_attendance_entry_mode"`

	// Global school settings
	SchoolTimezone               *string `gorm:"type:varchar(50);column:school_timezone" json:"school_timezone,omitempty"`
	SchoolDefaultMinPassingScore *int    `gorm:"type:int;column:school_default_min_passing_score" json:"school_default_min_passing_score,omitempty"`

	// 🆕 Default number of students per class (school-wide)
	SchoolDefaultClassQouta *int `gorm:"type:int;column:school_default_class_qouta" json:"school_default_class_qouta,omitempty"`

	SchoolSettings datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'::jsonb;column:school_settings" json:"school_settings"`

	// Relation to profile (if any)
	SchoolProfile *SchoolProfileModel `gorm:"foreignKey:SchoolProfileSchoolID;references:SchoolID" json:"school_profile,omitempty"`

	// Audit & soft delete
	SchoolCreatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:school_created_at" json:"school_created_at"`
	SchoolUpdatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:school_updated_at" json:"school_updated_at"`
	SchoolLastActivityAt *time.Time     `gorm:"type:timestamptz;column:school_last_activity_at" json:"school_last_activity_at,omitempty"`
	SchoolDeletedAt      gorm.DeletedAt `gorm:"index;column:school_deleted_at" json:"school_deleted_at,omitempty"`
}

func (SchoolModel) TableName() string { return "schools" }
