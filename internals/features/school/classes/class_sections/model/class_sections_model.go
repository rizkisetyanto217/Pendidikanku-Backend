// file: internals/features/school/class_sections/model/class_section_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSectionModel struct {
	// PK
	ClassSectionsID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_sections_id" json:"class_sections_id"`

	// Relasi wajib
	ClassSectionsClassID  uuid.UUID `gorm:"type:uuid;not null;column:class_sections_class_id;index:idx_sections_class" json:"class_sections_class_id"`
	ClassSectionsMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_sections_masjid_id;index:idx_sections_masjid" json:"class_sections_masjid_id"`

	// Relasi opsional
	ClassSectionsTeacherID    *uuid.UUID `gorm:"type:uuid;column:class_sections_teacher_id;index:idx_sections_teacher" json:"class_sections_teacher_id,omitempty"`
	ClassSectionsClassRoomID  *uuid.UUID `gorm:"type:uuid;column:class_sections_class_room_id;index:idx_sections_class_room" json:"class_sections_class_room_id,omitempty"`

	// Identitas
	ClassSectionsSlug string  `gorm:"size:160;not null;column:class_sections_slug;index:idx_sections_slug" json:"class_sections_slug"`
	ClassSectionsName string  `gorm:"size:100;not null;column:class_sections_name" json:"class_sections_name"`
	ClassSectionsCode *string `gorm:"size:50;column:class_sections_code" json:"class_sections_code,omitempty"`

	// Jadwal simple (teks bebas)
	ClassSectionsSchedule *string `gorm:"size:200;column:class_sections_schedule" json:"class_sections_schedule,omitempty"`

	// Kapasitas & counter
	ClassSectionsCapacity      *int  `gorm:"column:class_sections_capacity" json:"class_sections_capacity,omitempty"`
	ClassSectionsTotalStudents int   `gorm:"not null;default:0;column:class_sections_total_students" json:"class_sections_total_students"`

	// Group link (URL)
	ClassSectionsGroupURL *string `gorm:"column:class_sections_group_url;type:text" json:"class_sections_group_url,omitempty"`

	// Status
	ClassSectionsIsActive bool `gorm:"not null;default:true;column:class_sections_is_active;index:idx_sections_active" json:"class_sections_is_active"`

	// Timestamps (soft delete)
	ClassSectionsCreatedAt time.Time      `gorm:"column:class_sections_created_at;autoCreateTime;index:idx_sections_created_at,sort:desc" json:"class_sections_created_at"`
	ClassSectionsUpdatedAt time.Time      `gorm:"column:class_sections_updated_at;autoUpdateTime" json:"class_sections_updated_at"`
	ClassSectionsDeletedAt gorm.DeletedAt `gorm:"column:class_sections_deleted_at;index" json:"class_sections_deleted_at,omitempty"`
}

func (ClassSectionModel) TableName() string { return "class_sections" }
