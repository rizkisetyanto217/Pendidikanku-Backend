// internals/features/lembaga/class_section_subject_teachers/model/csst_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSectionSubjectTeacherModel struct {
	// PK
	ClassSectionSubjectTeachersID uuid.UUID `json:"class_section_subject_teachers_id" gorm:"column:class_section_subject_teachers_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// FK & data inti
	ClassSectionSubjectTeachersMasjidID  uuid.UUID `json:"class_section_subject_teachers_masjid_id"  gorm:"column:class_section_subject_teachers_masjid_id;type:uuid;not null"`
	ClassSectionSubjectTeachersSectionID uuid.UUID `json:"class_section_subject_teachers_section_id" gorm:"column:class_section_subject_teachers_section_id;type:uuid;not null"`
	ClassSectionSubjectTeachersSubjectID uuid.UUID `json:"class_section_subject_teachers_subject_id" gorm:"column:class_section_subject_teachers_subject_id;type:uuid;not null"`

	// ✅ GANTI: mengacu ke masjid_teachers.masjid_teacher_id (BUKAN users.id)
	ClassSectionSubjectTeachersTeacherID uuid.UUID `json:"class_section_subject_teachers_teacher_id" gorm:"column:class_section_subject_teachers_teacher_id;type:uuid;not null"`

	// Status & audit
	ClassSectionSubjectTeachersIsActive  bool           `json:"class_section_subject_teachers_is_active"  gorm:"column:class_section_subject_teachers_is_active;not null;default:true"`
	ClassSectionSubjectTeachersCreatedAt time.Time      `json:"class_section_subject_teachers_created_at" gorm:"column:class_section_subject_teachers_created_at;not null;autoCreateTime"`
	ClassSectionSubjectTeachersUpdatedAt *time.Time     `json:"class_section_subject_teachers_updated_at" gorm:"column:class_section_subject_teachers_updated_at;autoUpdateTime"`
	ClassSectionSubjectTeachersDeletedAt gorm.DeletedAt `json:"class_section_subject_teachers_deleted_at" gorm:"column:class_section_subject_teachers_deleted_at;index"`
}

func (ClassSectionSubjectTeacherModel) TableName() string {
	return "class_section_subject_teachers"
}
