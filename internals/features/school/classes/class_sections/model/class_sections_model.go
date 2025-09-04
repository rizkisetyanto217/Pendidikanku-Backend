// file: internals/features/school/class_sections/model/class_section_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ClassSectionModel struct {
	// PK
	ClassSectionsID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_sections_id" json:"class_sections_id"`

	// Relasi wajib
	ClassSectionsClassID  uuid.UUID `gorm:"type:uuid;not null;column:class_sections_class_id" json:"class_sections_class_id"`
	ClassSectionsMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_sections_masjid_id" json:"class_sections_masjid_id"`

	// Teacher mengarah ke masjid_teachers.*
	ClassSectionsTeacherID *uuid.UUID `gorm:"type:uuid;column:class_sections_teacher_id" json:"class_sections_teacher_id,omitempty"`

	// Identitas
	ClassSectionsSlug string  `gorm:"size:160;not null;column:class_sections_slug" json:"class_sections_slug"`
	ClassSectionsName string  `gorm:"size:100;not null;column:class_sections_name" json:"class_sections_name"`
	ClassSectionsCode *string `gorm:"size:50;column:class_sections_code" json:"class_sections_code,omitempty"`

	// Kapasitas & jadwal tersimpan
	ClassSectionsCapacity *int           `gorm:"column:class_sections_capacity" json:"class_sections_capacity,omitempty"`
	ClassSectionsSchedule datatypes.JSON `gorm:"type:jsonb;column:class_sections_schedule" json:"class_sections_schedule,omitempty"`

	// Denormalized counter siswa
	ClassSectionsTotalStudents int `gorm:"not null;default:0;column:class_sections_total_students" json:"class_sections_total_students"`

	// Status aktif
	ClassSectionsIsActive bool `gorm:"not null;default:true;column:class_sections_is_active" json:"class_sections_is_active"`

	// Timestamps eksplisit
	ClassSectionsCreatedAt time.Time      `gorm:"column:class_sections_created_at;autoCreateTime" json:"class_sections_created_at"`
	ClassSectionsUpdatedAt time.Time      `gorm:"column:class_sections_updated_at;autoUpdateTime" json:"class_sections_updated_at"`
	ClassSectionsDeletedAt gorm.DeletedAt `gorm:"column:class_sections_deleted_at;index" json:"class_sections_deleted_at,omitempty"`
}

func (ClassSectionModel) TableName() string {
	return "class_sections"
}
