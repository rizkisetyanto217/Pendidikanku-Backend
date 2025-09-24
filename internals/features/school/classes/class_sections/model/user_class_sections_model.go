// file: internals/features/school/enrolments/model/user_class_section_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserClassSection struct {
	// PK
	UserClassSectionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_section_id" json:"user_class_section_id"`

	// Enrolment & Section
	UserClassSectionUserClassID uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_user_class_id" json:"user_class_section_user_class_id"`
	UserClassSectionSectionID   uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_section_id" json:"user_class_section_section_id"`

	// Tenant (denormalized)
	UserClassSectionMasjidID uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_masjid_id" json:"user_class_section_masjid_id"`

	// Penempatan
	UserClassSectionAssignedAt   time.Time  `gorm:"type:date;not null;default:CURRENT_DATE;column:user_class_section_assigned_at" json:"user_class_section_assigned_at"`
	UserClassSectionUnassignedAt *time.Time `gorm:"type:date;column:user_class_section_unassigned_at" json:"user_class_section_unassigned_at,omitempty"`

	// Audit
	UserClassSectionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_section_created_at" json:"user_class_section_created_at"`
	UserClassSectionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_section_updated_at" json:"user_class_section_updated_at"`
	UserClassSectionDeletedAt gorm.DeletedAt `gorm:"column:user_class_section_deleted_at;index" json:"user_class_section_deleted_at,omitempty"`
}

func (UserClassSection) TableName() string { return "user_class_sections" }
