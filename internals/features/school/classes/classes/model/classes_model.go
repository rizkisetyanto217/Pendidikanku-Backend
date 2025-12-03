package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ClassModel merepresentasikan tabel `classes`.
type ClassModel struct {
	// =====================================================
	// PK & Tenant
	// =====================================================

	ClassID       uuid.UUID `json:"class_id"        gorm:"column:class_id;type:uuid;default:gen_random_uuid();primaryKey"`
	ClassSchoolID uuid.UUID `json:"class_school_id" gorm:"column:class_school_id;type:uuid;not null"`

	// =====================================================
	// Identitas
	// =====================================================

	ClassName *string `json:"class_name,omitempty" gorm:"column:class_name;type:varchar(160)"`
	ClassSlug string  `json:"class_slug"           gorm:"column:class_slug;type:varchar(160);not null"`

	// =====================================================
	// Periode kelas (DATE)
	// =====================================================

	ClassStartDate *time.Time `json:"class_start_date,omitempty" gorm:"column:class_start_date;type:date"`
	ClassEndDate   *time.Time `json:"class_end_date,omitempty"   gorm:"column:class_end_date;type:date"`

	// =====================================================
	// Registrasi
	// =====================================================

	ClassRegistrationOpensAt  *time.Time `json:"class_registration_opens_at,omitempty"  gorm:"column:class_registration_opens_at;type:timestamptz"`
	ClassRegistrationClosesAt *time.Time `json:"class_registration_closes_at,omitempty" gorm:"column:class_registration_closes_at;type:timestamptz"`

	// =====================================================
	// Kuota
	//  - ClassQuotaTotal → kapasitas maksimal (limit)
	//  - ClassQuotaTaken → sudah terpakai (count)
	// =====================================================

	ClassQuotaTotal *int `json:"class_quota_total,omitempty" gorm:"column:class_quota_total"`                    // CHECK (>=0)
	ClassQuotaTaken int  `json:"class_quota_taken"           gorm:"column:class_quota_taken;not null;default:0"` // CHECK (>=0)

	// =====================================================
	// Catatan & Meta biaya
	// =====================================================

	ClassNotes   *string           `json:"class_notes,omitempty"    gorm:"column:class_notes;type:text"`
	ClassFeeMeta datatypes.JSONMap `json:"class_fee_meta,omitempty" gorm:"column:class_fee_meta;type:jsonb"`

	// =====================================================
	// Mode & Status
	// =====================================================

	ClassDeliveryMode *string    `json:"class_delivery_mode,omitempty" gorm:"column:class_delivery_mode;type:class_delivery_mode_enum"`
	ClassStatus       string     `json:"class_status"                  gorm:"column:class_status;type:class_status_enum;not null;default:'active'"`
	ClassCompletedAt  *time.Time `json:"class_completed_at,omitempty"  gorm:"column:class_completed_at;type:timestamptz"`

	// =====================================================
	// Single image (2-slot + retensi 30 hari)
	// =====================================================

	ClassImageURL                *string    `json:"class_image_url,omitempty"                  gorm:"column:class_image_url;type:text"`
	ClassImageObjectKey          *string    `json:"class_image_object_key,omitempty"           gorm:"column:class_image_object_key;type:text"`
	ClassImageURLOld             *string    `json:"class_image_url_old,omitempty"              gorm:"column:class_image_url_old;type:text"`
	ClassImageObjectKeyOld       *string    `json:"class_image_object_key_old,omitempty"       gorm:"column:class_image_object_key_old;type:text"`
	ClassImageDeletePendingUntil *time.Time `json:"class_image_delete_pending_until,omitempty" gorm:"column:class_image_delete_pending_until;type:timestamptz"`

	// =====================================================
	// Snapshot Class Parent (denormalisasi label)
	// =====================================================

	ClassClassParentID         uuid.UUID `json:"class_class_parent_id"                    gorm:"column:class_class_parent_id;type:uuid;not null"`
	ClassClassParentCodeCache  *string   `json:"class_class_parent_code_cache,omitempty"  gorm:"column:class_class_parent_code_cache;type:varchar(40)"`
	ClassClassParentNameCache  *string   `json:"class_class_parent_name_cache,omitempty"  gorm:"column:class_class_parent_name_cache;type:varchar(80)"`
	ClassClassParentSlugCache  *string   `json:"class_class_parent_slug_cache,omitempty"  gorm:"column:class_class_parent_slug_cache;type:varchar(160)"`
	ClassClassParentLevelCache *int16    `json:"class_class_parent_level_cache,omitempty" gorm:"column:class_class_parent_level_cache;type:smallint"`
	ClassClassParentURLCache   *string   `json:"class_class_parent_url_cache,omitempty"   gorm:"column:class_class_parent_url_cache;type:varchar(160)"`

	// =====================================================
	// Snapshot Academic Term (denormalisasi label)
	// =====================================================

	ClassAcademicTermID                *uuid.UUID `json:"class_academic_term_id,omitempty"                        gorm:"column:class_academic_term_id;type:uuid"`
	ClassAcademicTermAcademicYearCache *string    `json:"class_academic_term_academic_year_cache,omitempty"       gorm:"column:class_academic_term_academic_year_cache;type:varchar(40)"`
	ClassAcademicTermNameCache         *string    `json:"class_academic_term_name_cache,omitempty"                gorm:"column:class_academic_term_name_cache;type:varchar(100)"`
	ClassAcademicTermSlugCache         *string    `json:"class_academic_term_slug_cache,omitempty"                gorm:"column:class_academic_term_slug_cache;type:varchar(160)"`
	ClassAcademicTermAngkatanCache     *string    `json:"class_academic_term_angkatan_cache,omitempty"            gorm:"column:class_academic_term_angkatan_cache;type:varchar(40)"`

	// =====================================================
	// STATS (per class - ALL) → COUNT
	// =====================================================

	ClassClassSectionCount    int `json:"class_class_section_count"      gorm:"column:class_class_section_count;not null;default:0"`
	ClassStudentCount         int `json:"class_student_count"            gorm:"column:class_student_count;not null;default:0"`
	ClassStudentMaleCount     int `json:"class_student_male_count"       gorm:"column:class_student_male_count;not null;default:0"`
	ClassStudentFemaleCount   int `json:"class_student_female_count"     gorm:"column:class_student_female_count;not null;default:0"`
	ClassTeacherCount         int `json:"class_teacher_count"            gorm:"column:class_teacher_count;not null;default:0"`
	ClassClassEnrollmentCount int `json:"class_class_enrollment_count"   gorm:"column:class_class_enrollment_count;not null;default:0"`

	// =====================================================
	// STATS (per class - ACTIVE ONLY) → COUNT
	// =====================================================

	ClassClassSectionActiveCount    int `json:"class_class_section_active_count"      gorm:"column:class_class_section_active_count;not null;default:0"`
	ClassStudentActiveCount         int `json:"class_student_active_count"            gorm:"column:class_student_active_count;not null;default:0"`
	ClassStudentMaleActiveCount     int `json:"class_student_male_active_count"       gorm:"column:class_student_male_active_count;not null;default:0"`
	ClassStudentFemaleActiveCount   int `json:"class_student_female_active_count"     gorm:"column:class_student_female_active_count;not null;default:0"`
	ClassTeacherActiveCount         int `json:"class_teacher_active_count"            gorm:"column:class_teacher_active_count;not null;default:0"`
	ClassClassEnrollmentActiveCount int `json:"class_class_enrollment_active_count"   gorm:"column:class_class_enrollment_active_count;not null;default:0"`

	// =====================================================
	// Stats extra (JSONB)
	// =====================================================

	ClassStats datatypes.JSONMap `json:"class_stats,omitempty" gorm:"column:class_stats;type:jsonb"`

	// =====================================================
	// Audit & soft delete
	// =====================================================

	ClassCreatedAt time.Time      `json:"class_created_at"           gorm:"column:class_created_at;type:timestamptz;not null;default:now();autoCreateTime"`
	ClassUpdatedAt time.Time      `json:"class_updated_at"           gorm:"column:class_updated_at;type:timestamptz;not null;default:now();autoUpdateTime"`
	ClassDeletedAt gorm.DeletedAt `json:"class_deleted_at,omitempty" gorm:"column:class_deleted_at;type:timestamptz;index"`
}

func (ClassModel) TableName() string { return "classes" }

// (Opsional) konstanta untuk nilai enum—berguna di layer service/DTO.
const (
	ClassDeliveryModeOffline = "offline"
	ClassDeliveryModeOnline  = "online"
	ClassDeliveryModeHybrid  = "hybrid"

	ClassStatusActive    = "active"
	ClassStatusInactive  = "inactive"
	ClassStatusCompleted = "completed"
)
