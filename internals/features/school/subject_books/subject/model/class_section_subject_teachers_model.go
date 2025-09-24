// file: internals/features/school/sectionsubjectteachers/model/class_section_subject_teacher_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSectionSubjectTeacherModel struct {
	// PK
	ClassSectionSubjectTeacherID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_subject_teacher_id" json:"class_section_subject_teacher_id"`

	// Tenant
	ClassSectionSubjectTeacherMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_masjid_id" json:"class_section_subject_teacher_masjid_id"`

	// Relations (IDs saja)
	ClassSectionSubjectTeacherSectionID     uuid.UUID  `gorm:"type:uuid;not null;column:class_section_subject_teacher_section_id" json:"class_section_subject_teacher_section_id"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_class_subject_id" json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherTeacherID     uuid.UUID  `gorm:"type:uuid;not null;column:class_section_subject_teacher_teacher_id" json:"class_section_subject_teacher_teacher_id"`

	// Identitas/opsional
	ClassSectionSubjectTeacherSlug        *string   `gorm:"type:varchar(160);column:class_section_subject_teacher_slug" json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string   `gorm:"type:text;column:class_section_subject_teacher_description" json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID`gorm:"type:uuid;column:class_section_subject_teacher_room_id" json:"class_section_subject_teacher_room_id,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string   `gorm:"type:text;column:class_section_subject_teacher_group_url" json:"class_section_subject_teacher_group_url,omitempty"`

	// Status & audit
	ClassSectionSubjectTeacherIsActive  bool            `gorm:"not null;default:true;column:class_section_subject_teacher_is_active" json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCreatedAt time.Time       `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_created_at" json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt time.Time       `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_updated_at" json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt gorm.DeletedAt  `gorm:"column:class_section_subject_teacher_deleted_at;index" json:"class_section_subject_teacher_deleted_at,omitempty"`
}

func (ClassSectionSubjectTeacherModel) TableName() string { return "class_section_subject_teachers" }