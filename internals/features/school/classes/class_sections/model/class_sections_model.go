package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ClassSectionModel struct {
	ClassSectionsID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_sections_id" json:"class_sections_id"`
	ClassSectionsClassID   uuid.UUID  `gorm:"type:uuid;not null;column:class_sections_class_id" json:"class_sections_class_id"`
	ClassSectionsMasjidID  *uuid.UUID `gorm:"type:uuid;column:class_sections_masjid_id" json:"class_sections_masjid_id,omitempty"`
	// Mengganti dari teacher_id yang sebelumnya mengarah ke users(id) menjadi mengarah ke masjid_teachers(id)
	ClassSectionsTeacherID *uuid.UUID `gorm:"type:uuid;column:class_sections_teacher_id" json:"class_sections_teacher_id,omitempty"`

	ClassSectionsSlug     string         `gorm:"size:160;uniqueIndex:idx_sections_slug;not null;column:class_sections_slug" json:"class_sections_slug"`
	ClassSectionsName     string         `gorm:"size:100;not null;column:class_sections_name" json:"class_sections_name"`
	ClassSectionsCode     *string        `gorm:"size:50;column:class_sections_code" json:"class_sections_code,omitempty"`
	ClassSectionsCapacity *int           `gorm:"column:class_sections_capacity" json:"class_sections_capacity,omitempty"`
	ClassSectionsSchedule datatypes.JSON `gorm:"type:jsonb;column:class_sections_schedule" json:"class_sections_schedule,omitempty"`

	ClassSectionsIsActive  bool       `gorm:"not null;default:true;column:class_sections_is_active" json:"class_sections_is_active"`
	ClassSectionsCreatedAt time.Time  `gorm:"column:class_sections_created_at;autoCreateTime" json:"class_sections_created_at"`
	ClassSectionsUpdatedAt *time.Time `gorm:"column:class_sections_updated_at;autoUpdateTime" json:"class_sections_updated_at,omitempty"`
	ClassSectionsDeletedAt *time.Time `gorm:"column:class_sections_deleted_at" json:"class_sections_deleted_at,omitempty"`
}

// Menentukan nama tabel secara eksplisit
func (ClassSectionModel) TableName() string {
	return "class_sections"
}
