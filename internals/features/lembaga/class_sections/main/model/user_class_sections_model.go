// internals/features/lembaga/classes/user_class_sections/main/model/user_class_section_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	UserClassSectionStatusActive   = "active"
	UserClassSectionStatusInactive = "inactive"
	UserClassSectionStatusEnded    = "ended"
)

type UserClassSectionsModel struct {
	UserClassSectionsID        uuid.UUID  `json:"user_class_sections_id"          gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_sections_id"`
	UserClassSectionsUserClassID uuid.UUID  `json:"user_class_sections_user_class_id" gorm:"type:uuid;not null;column:user_class_sections_user_class_id"`
	UserClassSectionsSectionID uuid.UUID  `json:"user_class_sections_section_id"  gorm:"type:uuid;not null;column:user_class_sections_section_id"`
	UserClassSectionsMasjidID  uuid.UUID  `json:"user_class_sections_masjid_id"   gorm:"type:uuid;not null;column:user_class_sections_masjid_id"`

	UserClassSectionsAssignedAt time.Time  `json:"user_class_sections_assigned_at" gorm:"type:date;not null;column:user_class_sections_assigned_at"`
	UserClassSectionsUnassignedAt *time.Time `json:"user_class_sections_unassigned_at,omitempty" gorm:"type:date;column:user_class_sections_unassigned_at"`

	UserClassSectionsCreatedAt time.Time  `json:"user_class_sections_created_at"  gorm:"column:user_class_sections_created_at;autoCreateTime"`
	UserClassSectionsUpdatedAt *time.Time `json:"user_class_sections_updated_at,omitempty" gorm:"column:user_class_sections_updated_at;autoUpdateTime"`

	// --- Relasi opsional (hindari import cycle). Uncomment & sesuaikan import bila dibutuhkan ---
	// UserClass *userclassesmodel.UserClassesModel `gorm:"foreignKey:UserClassSectionsUserClassID;references:UserClassesID"`
	// Section   *sectionmodel.ClassSectionModel    `gorm:"foreignKey:UserClassSectionsSectionID;references:ClassSectionID"`
}

func (UserClassSectionsModel) TableName() string {
	return "user_class_sections"
}
