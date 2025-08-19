// internals/features/lembaga/class_section_subject_teachers/model/csst_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSectionSubjectTeacherModel struct {
	ClassSectionSubjectTeachersID uuid.UUID `json:"class_section_subject_teachers_id" gorm:"column:class_section_subject_teachers_id;type:uuid;default:gen_random_uuid();primaryKey"`

	ClassSectionSubjectTeacherModelMasjidID     uuid.UUID      `json:"class_section_subject_teachers_masjid_id" gorm:"column:class_section_subject_teachers_masjid_id;type:uuid;not null"`
	ClassSectionSubjectTeacherModelSectionID    uuid.UUID      `json:"class_section_subject_teachers_section_id" gorm:"column:class_section_subject_teachers_section_id;type:uuid;not null"`
	ClassSectionSubjectTeacherModelSubjectID    uuid.UUID      `json:"class_section_subject_teachers_subject_id" gorm:"column:class_section_subject_teachers_subject_id;type:uuid;not null"`
	ClassSectionSubjectTeacherModelTeacherUserID uuid.UUID     `json:"class_section_subject_teachers_teacher_user_id" gorm:"column:class_section_subject_teachers_teacher_user_id;type:uuid;not null"`

	ClassSectionSubjectTeacherModelIsActive  bool           `json:"class_section_subject_teachers_is_active"  gorm:"column:class_section_subject_teachers_is_active;not null;default:true"`
	ClassSectionSubjectTeacherModelCreatedAt time.Time      `json:"class_section_subject_teachers_created_at" gorm:"column:class_section_subject_teachers_created_at;not null;default:CURRENT_TIMESTAMP"`
	ClassSectionSubjectTeacherModelUpdatedAt *time.Time     `json:"class_section_subject_teachers_updated_at" gorm:"column:class_section_subject_teachers_updated_at"`
	ClassSectionSubjectTeacherModelDeletedAt gorm.DeletedAt `json:"class_section_subject_teachers_deleted_at" gorm:"column:class_section_subject_teachers_deleted_at;index"`
}

func (ClassSectionSubjectTeacherModel) TableName() string {
	return "class_section_subject_teachers"
}
