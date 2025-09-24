// file: internals/features/school/sections/model/class_section_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSectionModel struct {
	// PK
	ClassSectionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_id" json:"class_section_id"`

	// Tenant & relasi inti (IDs saja)
	ClassSectionMasjidID           uuid.UUID  `gorm:"type:uuid;not null;column:class_section_masjid_id" json:"class_section_masjid_id"`
	ClassSectionClassID            uuid.UUID  `gorm:"type:uuid;not null;column:class_section_class_id" json:"class_section_class_id"`
	ClassSectionTeacherID          *uuid.UUID `gorm:"type:uuid;column:class_section_teacher_id" json:"class_section_teacher_id,omitempty"`
	ClassSectionAssistantTeacherID *uuid.UUID `gorm:"type:uuid;column:class_section_assistant_teacher_id" json:"class_section_assistant_teacher_id,omitempty"`
	ClassSectionClassRoomID        *uuid.UUID `gorm:"type:uuid;column:class_section_class_room_id" json:"class_section_class_room_id,omitempty"`
	ClassSectionLeaderStudentID    *uuid.UUID `gorm:"type:uuid;column:class_section_leader_student_id" json:"class_section_leader_student_id,omitempty"`

	// Identitas
	ClassSectionSlug string  `gorm:"type:varchar(160);not null;column:class_section_slug" json:"class_section_slug"`
	ClassSectionName string  `gorm:"type:varchar(100);not null;column:class_section_name" json:"class_section_name"`
	ClassSectionCode *string `gorm:"type:varchar(50);column:class_section_code" json:"class_section_code,omitempty"`

	// Jadwal simple
	ClassSectionSchedule *string `gorm:"type:text;column:class_section_schedule" json:"class_section_schedule,omitempty"`

	// Kapasitas & counter
	ClassSectionCapacity      *int `gorm:"column:class_section_capacity" json:"class_section_capacity,omitempty"`
	ClassSectionTotalStudents int  `gorm:"not null;default:0;column:class_section_total_students" json:"class_section_total_students"`

	// Meeting / Group
	ClassSectionGroupURL *string `gorm:"type:text;column:class_section_group_url" json:"class_section_group_url,omitempty"`

	// Image (2-slot + retensi 30 hari)
	ClassSectionImageURL                *string    `gorm:"type:text;column:class_section_image_url" json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey          *string    `gorm:"type:text;column:class_section_image_object_key" json:"class_section_image_object_key,omitempty"`
	ClassSectionImageURLOld             *string    `gorm:"type:text;column:class_section_image_url_old" json:"class_section_image_url_old,omitempty"`
	ClassSectionImageObjectKeyOld       *string    `gorm:"type:text;column:class_section_image_object_key_old" json:"class_section_image_object_key_old,omitempty"`
	ClassSectionImageDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:class_section_image_delete_pending_until" json:"class_section_image_delete_pending_until,omitempty"`

	// Status & audit
	ClassSectionIsActive  bool           `gorm:"not null;default:true;column:class_section_is_active" json:"class_section_is_active"`
	ClassSectionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_created_at" json:"class_section_created_at"`
	ClassSectionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_updated_at" json:"class_section_updated_at"`
	ClassSectionDeletedAt gorm.DeletedAt `gorm:"column:class_section_deleted_at;index" json:"class_section_deleted_at,omitempty"`
}

func (ClassSectionModel) TableName() string { return "class_sections" }
