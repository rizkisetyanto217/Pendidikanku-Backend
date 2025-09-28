package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ========================= ENUMS =========================

type UserClassSectionStatus string
type UserClassSectionResult string

const (
	// Status
	UserClassSectionActive    UserClassSectionStatus = "active"
	UserClassSectionInactive  UserClassSectionStatus = "inactive"
	UserClassSectionCompleted UserClassSectionStatus = "completed"

	// Result
	UserClassSectionPassed UserClassSectionResult = "passed"
	UserClassSectionFailed UserClassSectionResult = "failed"
)

// ========================= MODEL =========================

type UserClassSection struct {
	UserClassSectionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_section_id" json:"user_class_section_id"`

	// Identitas siswa & tenant
	UserClassSectionMasjidStudentID uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_masjid_student_id" json:"user_class_section_masjid_student_id"`
	UserClassSectionSectionID       uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_section_id" json:"user_class_section_section_id"`
	UserClassSectionMasjidID        uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_masjid_id" json:"user_class_section_masjid_id"`

	// Lifecycle enrolment
	UserClassSectionStatus UserClassSectionStatus  `gorm:"type:text;not null;default:'active';column:user_class_section_status" json:"user_class_section_status"`
	UserClassSectionResult *UserClassSectionResult `gorm:"type:text;column:user_class_section_result" json:"user_class_section_result,omitempty"`

	// Snapshot biaya
	UserClassSectionFeeSnapshot datatypes.JSON `gorm:"type:jsonb;column:user_class_section_fee_snapshot" json:"user_class_section_fee_snapshot,omitempty"`

	// Jejak waktu
	UserClassSectionAssignedAt   time.Time  `gorm:"type:date;not null;default:current_date;column:user_class_section_assigned_at" json:"user_class_section_assigned_at"`
	UserClassSectionUnassignedAt *time.Time `gorm:"type:date;column:user_class_section_unassigned_at" json:"user_class_section_unassigned_at,omitempty"`
	UserClassSectionCompletedAt  *time.Time `gorm:"type:timestamptz;column:user_class_section_completed_at" json:"user_class_section_completed_at,omitempty"`

	// Audit
	UserClassSectionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:user_class_section_created_at" json:"user_class_section_created_at"`
	UserClassSectionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:user_class_section_updated_at" json:"user_class_section_updated_at"`
	UserClassSectionDeletedAt gorm.DeletedAt `gorm:"index;column:user_class_section_deleted_at" json:"user_class_section_deleted_at,omitempty"`
}

// TableName override
func (UserClassSection) TableName() string {
	return "user_class_sections"
}
