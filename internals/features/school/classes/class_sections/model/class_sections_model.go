// file: internals/features/school/classes/model/class_sections_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSectionModel struct {
	// PK & tenant
	ClassSectionsID       uuid.UUID      `gorm:"column:class_sections_id;type:uuid;primaryKey" json:"class_sections_id"`
	ClassSectionsMasjidID uuid.UUID      `gorm:"column:class_sections_masjid_id;type:uuid;not null" json:"class_sections_masjid_id"`

	// Relasi (FK komposit tenant-safe; dikontrol di migration)
	ClassSectionsClassID            uuid.UUID  `gorm:"column:class_sections_class_id;type:uuid;not null" json:"class_sections_class_id"`
	ClassSectionsTeacherID          *uuid.UUID `gorm:"column:class_sections_teacher_id;type:uuid" json:"class_sections_teacher_id"`
	ClassSectionsAssistantTeacherID *uuid.UUID `gorm:"column:class_sections_assistant_teacher_id;type:uuid" json:"class_sections_assistant_teacher_id"`
	ClassSectionsClassRoomID        *uuid.UUID `gorm:"column:class_sections_class_room_id;type:uuid" json:"class_sections_class_room_id"`

	// Leader (ketua kelas) → masjid_students
	ClassSectionsLeaderStudentID *uuid.UUID `gorm:"column:class_sections_leader_student_id;type:uuid" json:"class_sections_leader_student_id"`

	// Identitas
	ClassSectionsSlug string  `gorm:"column:class_sections_slug;type:varchar(160);not null" json:"class_sections_slug"`
	ClassSectionsName string  `gorm:"column:class_sections_name;type:varchar(100);not null" json:"class_sections_name"`
	ClassSectionsCode *string `gorm:"column:class_sections_code;type:varchar(50)" json:"class_sections_code"`

	// Jadwal sederhana
	ClassSectionsSchedule *string `gorm:"column:class_sections_schedule;type:text" json:"class_sections_schedule"`

	// Kapasitas & counter
	ClassSectionsCapacity      *int `gorm:"column:class_sections_capacity" json:"class_sections_capacity"`
	ClassSectionsTotalStudents int  `gorm:"column:class_sections_total_students;not null;default:0" json:"class_sections_total_students"`

	// Meeting / Group
	ClassSectionsGroupURL *string `gorm:"column:class_sections_group_url;type:text" json:"class_sections_group_url"`

	// Image (2-slot + retensi)
	ClassSectionsImageURL                 *string    `gorm:"column:class_sections_image_url" json:"class_sections_image_url"`
	ClassSectionsImageObjectKey           *string    `gorm:"column:class_sections_image_object_key" json:"class_sections_image_object_key"`
	ClassSectionsImageURLOld              *string    `gorm:"column:class_sections_image_url_old" json:"class_sections_image_url_old"`
	ClassSectionsImageObjectKeyOld        *string    `gorm:"column:class_sections_image_object_key_old" json:"class_sections_image_object_key_old"`
	ClassSectionsImageDeletePendingUntil  *time.Time `gorm:"column:class_sections_image_delete_pending_until" json:"class_sections_image_delete_pending_until"`

	// Status & audit
	ClassSectionsIsActive  bool            `gorm:"column:class_sections_is_active;not null;default:true" json:"class_sections_is_active"`
	ClassSectionsCreatedAt time.Time       `gorm:"column:class_sections_created_at;not null;autoCreateTime" json:"class_sections_created_at"`
	ClassSectionsUpdatedAt time.Time       `gorm:"column:class_sections_updated_at;not null;autoUpdateTime" json:"class_sections_updated_at"`
	ClassSectionsDeletedAt gorm.DeletedAt  `gorm:"column:class_sections_deleted_at;index" json:"class_sections_deleted_at"`
}

func (ClassSectionModel) TableName() string {
	return "class_sections"
}