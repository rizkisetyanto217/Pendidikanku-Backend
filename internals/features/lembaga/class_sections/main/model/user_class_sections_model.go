// internals/features/lembaga/classes/user_class_sections/main/model/user_class_section_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserClassSectionsModel struct {
	UserClassSectionsID           uuid.UUID      `json:"user_class_sections_id"              gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_sections_id"`
	UserClassSectionsUserClassID  uuid.UUID      `json:"user_class_sections_user_class_id"   gorm:"type:uuid;not null;column:user_class_sections_user_class_id"`
	UserClassSectionsSectionID    uuid.UUID      `json:"user_class_sections_section_id"      gorm:"type:uuid;not null;column:user_class_sections_section_id"`
	UserClassSectionsMasjidID     uuid.UUID      `json:"user_class_sections_masjid_id"       gorm:"type:uuid;not null;column:user_class_sections_masjid_id"`

	// DATE di DB → pakai time.Time / *time.Time
	UserClassSectionsAssignedAt   time.Time      `json:"user_class_sections_assigned_at"     gorm:"type:date;not null;column:user_class_sections_assigned_at"` // DEFAULT di-set di DB
	UserClassSectionsUnassignedAt *time.Time     `json:"user_class_sections_unassigned_at,omitempty" gorm:"type:date;column:user_class_sections_unassigned_at"`

	// Timestamps (TIMESTAMPTZ di DB)
	UserClassSectionsCreatedAt    time.Time      `json:"user_class_sections_created_at"      gorm:"column:user_class_sections_created_at;autoCreateTime"`
	UserClassSectionsUpdatedAt    time.Time      `json:"user_class_sections_updated_at"      gorm:"column:user_class_sections_updated_at;autoUpdateTime"`

	// Soft delete
	UserClassSectionsDeletedAt    gorm.DeletedAt `json:"user_class_sections_deleted_at,omitempty" gorm:"column:user_class_sections_deleted_at;index"`
}

func (UserClassSectionsModel) TableName() string { return "user_class_sections" }
