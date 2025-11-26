// file: internals/features/school/academics/classes/model/class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ClassModel merepresentasikan tabel `classes` sesuai DDL terbaru.
type ClassModel struct {
	// PK & Tenant
	ClassID       uuid.UUID `json:"class_id"        gorm:"column:class_id;type:uuid;default:gen_random_uuid();primaryKey"`
	ClassSchoolID uuid.UUID `json:"class_school_id" gorm:"column:class_school_id;type:uuid;not null"`

	// Identitas
	// (DDL: class_name boleh NULL → pakai *string)
	ClassName *string `json:"class_name,omitempty" gorm:"column:class_name;type:varchar(160)"`
	ClassSlug string  `json:"class_slug"           gorm:"column:class_slug;type:varchar(160);not null"`

	// Periode kelas (DATE di DB → time.Time, pakai bagian tanggal)
	ClassStartDate *time.Time `json:"class_start_date,omitempty" gorm:"column:class_start_date;type:date"`
	ClassEndDate   *time.Time `json:"class_end_date,omitempty"   gorm:"column:class_end_date;type:date"`

	// Registrasi
	ClassRegistrationOpensAt  *time.Time `json:"class_registration_opens_at,omitempty"  gorm:"column:class_registration_opens_at;type:timestamptz"`
	ClassRegistrationClosesAt *time.Time `json:"class_registration_closes_at,omitempty" gorm:"column:class_registration_closes_at;type:timestamptz"`

	// Kuota
	ClassQuotaTotal *int `json:"class_quota_total,omitempty" gorm:"column:class_quota_total"`
	ClassQuotaTaken int  `json:"class_quota_taken"           gorm:"column:class_quota_taken;not null;default:0"`

	// Catatan & Meta biaya (informasional)
	ClassNotes   *string           `json:"class_notes,omitempty"    gorm:"column:class_notes;type:text"`
	ClassFeeMeta datatypes.JSONMap `json:"class_fee_meta,omitempty" gorm:"column:class_fee_meta;type:jsonb"`

	// Mode & Status (ENUM di DB)
	ClassDeliveryMode *string    `json:"class_delivery_mode,omitempty" gorm:"column:class_delivery_mode;type:class_delivery_mode_enum"`
	ClassStatus       string     `json:"class_status"                  gorm:"column:class_status;type:class_status_enum;not null;default:'active'"`
	ClassCompletedAt  *time.Time `json:"class_completed_at,omitempty"  gorm:"column:class_completed_at;type:timestamptz"`

	// Single image (2-slot + retensi 30 hari)
	ClassImageURL                *string    `json:"class_image_url,omitempty"                  gorm:"column:class_image_url;type:text"`
	ClassImageObjectKey          *string    `json:"class_image_object_key,omitempty"           gorm:"column:class_image_object_key;type:text"`
	ClassImageURLOld             *string    `json:"class_image_url_old,omitempty"              gorm:"column:class_image_url_old;type:text"`
	ClassImageObjectKeyOld       *string    `json:"class_image_object_key_old,omitempty"       gorm:"column:class_image_object_key_old;type:text"`
	ClassImageDeletePendingUntil *time.Time `json:"class_image_delete_pending_until,omitempty" gorm:"column:class_image_delete_pending_until;type:timestamptz"`

	// Snapshot Class Parent (FIXED) — per DDL pakai prefix class_class_parent_*
	ClassClassParentID       uuid.UUID `json:"class_class_parent_id"                 gorm:"column:class_class_parent_id;type:uuid;not null"`
	ClassParentCodeSnapshot  *string   `json:"class_parent_code_snapshot,omitempty"  gorm:"column:class_class_parent_code_snapshot;type:varchar(40)"`
	ClassParentNameSnapshot  *string   `json:"class_parent_name_snapshot,omitempty"  gorm:"column:class_class_parent_name_snapshot;type:varchar(80)"`
	ClassParentSlugSnapshot  *string   `json:"class_parent_slug_snapshot,omitempty"  gorm:"column:class_class_parent_slug_snapshot;type:varchar(160)"`
	ClassParentLevelSnapshot *int16    `json:"class_parent_level_snapshot,omitempty" gorm:"column:class_class_parent_level_snapshot;type:smallint"`
	ClassParentURLSnapshot   *string   `json:"class_parent_url_snapshot,omitempty"   gorm:"column:class_class_parent_url_snapshot;type:varchar(160)"`

	// Snapshot Academic Term (FIXED) — per DDL pakai prefix class_academic_term_*
	ClassAcademicTermID                   *uuid.UUID `json:"class_academic_term_id,omitempty"                     gorm:"column:class_academic_term_id;type:uuid"`
	ClassAcademicTermAcademicYearSnapshot *string    `json:"class_academic_term_academic_year_snapshot,omitempty" gorm:"column:class_academic_term_academic_year_snapshot;type:varchar(40)"`
	ClassAcademicTermNameSnapshot         *string    `json:"class_academic_term_name_snapshot,omitempty"          gorm:"column:class_academic_term_name_snapshot;type:varchar(100)"`
	ClassAcademicTermSlugSnapshot         *string    `json:"class_academic_term_slug_snapshot,omitempty"          gorm:"column:class_academic_term_slug_snapshot;type:varchar(160)"`
	ClassAcademicTermAngkatanSnapshot     *string    `json:"class_academic_term_angkatan_snapshot,omitempty"      gorm:"column:class_academic_term_angkatan_snapshot;type:varchar(40)"`

	// Stats / aggregates (hasil ALTER TABLE terbaru)
	ClassTotalClassSections    int               `json:"class_total_class_sections"    gorm:"column:class_total_class_sections;not null;default:0"`
	ClassTotalStudents         int               `json:"class_total_students"          gorm:"column:class_total_students;not null;default:0"`
	ClassTotalStudentsMale     int               `json:"class_total_students_male"     gorm:"column:class_total_students_male;not null;default:0"`
	ClassTotalStudentsFemale   int               `json:"class_total_students_female"   gorm:"column:class_total_students_female;not null;default:0"`
	ClassTotalTeachers         int               `json:"class_total_teachers"          gorm:"column:class_total_teachers;not null;default:0"`
	ClassTotalClassEnrollments int               `json:"class_total_class_enrollments" gorm:"column:class_total_class_enrollments;not null;default:0"`
	ClassStats                 datatypes.JSONMap `json:"class_stats,omitempty"         gorm:"column:class_stats;type:jsonb"`

	// Audit & soft delete
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
