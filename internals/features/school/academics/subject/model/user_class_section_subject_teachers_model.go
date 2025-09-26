// file: internals/features/school/sectionsubjectteachers/model/user_class_section_subject_teacher_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserClassSectionSubjectTeacher struct {
	// PK
	UserClassSectionSubjectTeacherID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_section_subject_teacher_id" json:"user_class_section_subject_teacher_id"`

	// Tenant
	UserClassSectionSubjectTeacherMasjidID uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_subject_teacher_masjid_id" json:"user_class_section_subject_teacher_masjid_id"`

	// Relations (IDs saja)
	UserClassSectionSubjectTeacherSectionID      uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_subject_teacher_section_id" json:"user_class_section_subject_teacher_section_id"`
	UserClassSectionSubjectTeacherClassSubjectID uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_subject_teacher_class_subject_id" json:"user_class_section_subject_teacher_class_subject_id"`
	UserClassSectionSubjectTeacherTeacherID      uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_subject_teacher_teacher_id" json:"user_class_section_subject_teacher_teacher_id"`

	// Status & audit
	UserClassSectionSubjectTeacherIsActive  bool           `gorm:"not null;default:true;column:user_class_section_subject_teacher_is_active" json:"user_class_section_subject_teacher_is_active"`
	UserClassSectionSubjectTeacherCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_section_subject_teacher_created_at" json:"user_class_section_subject_teacher_created_at"`
	UserClassSectionSubjectTeacherUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:user_class_section_subject_teacher_updated_at" json:"user_class_section_subject_teacher_updated_at"`
	UserClassSectionSubjectTeacherDeletedAt gorm.DeletedAt `gorm:"column:user_class_section_subject_teacher_deleted_at;index" json:"user_class_section_subject_teacher_deleted_at,omitempty"`
}

func (UserClassSectionSubjectTeacher) TableName() string {
	return "user_class_section_subject_teachers"
}
