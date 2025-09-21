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

	// Tenant & konteks
	ClassSectionSubjectTeachersMasjidID        uuid.UUID `json:"class_section_subject_teachers_masjid_id"         gorm:"column:class_section_subject_teachers_masjid_id;type:uuid;not null"`
	ClassSectionSubjectTeachersSectionID       uuid.UUID `json:"class_section_subject_teachers_section_id"        gorm:"column:class_section_subject_teachers_section_id;type:uuid;not null"`
	ClassSectionSubjectTeachersClassSubjectsID uuid.UUID `json:"class_section_subject_teachers_class_subjects_id" gorm:"column:class_section_subject_teachers_class_subjects_id;type:uuid;not null"`

	// ✅ refer ke masjid_teachers.masjid_teacher_id (BUKAN users.id)
	ClassSectionSubjectTeachersTeacherID uuid.UUID `json:"class_section_subject_teachers_teacher_id" gorm:"column:class_section_subject_teachers_teacher_id;type:uuid;not null"`

	// >>> SLUG <<<
	ClassSectionSubjectTeachersSlug *string `json:"class_section_subject_teachers_slug,omitempty" gorm:"column:class_section_subject_teachers_slug;type:varchar(160)"`

	// Deskripsi
	ClassSectionSubjectTeachersDescription *string `json:"class_section_subject_teachers_description,omitempty" gorm:"column:class_section_subject_teachers_description;type:text"`

	// 🔄 Override ruangan (opsional; default-nya dari class_sections.class_rooms_id)
	ClassSectionSubjectTeachersRoomID *uuid.UUID `json:"class_section_subject_teachers_room_id,omitempty" gorm:"column:class_section_subject_teachers_room_id;type:uuid"`

	// 🔗 Grup pelajaran (mis. WhatsApp)
	ClassSectionSubjectTeachersGroupURL *string `json:"class_section_subject_teachers_group_url,omitempty" gorm:"column:class_section_subject_teachers_group_url;type:text"`

	// Status & audit
	ClassSectionSubjectTeachersIsActive  bool           `json:"class_section_subject_teachers_is_active"  gorm:"column:class_section_subject_teachers_is_active;not null;default:true"`
	ClassSectionSubjectTeachersCreatedAt time.Time      `json:"class_section_subject_teachers_created_at" gorm:"column:class_section_subject_teachers_created_at;not null;autoCreateTime"`
	ClassSectionSubjectTeachersUpdatedAt *time.Time     `json:"class_section_subject_teachers_updated_at,omitempty" gorm:"column:class_section_subject_teachers_updated_at;autoUpdateTime"`
	ClassSectionSubjectTeachersDeletedAt gorm.DeletedAt `json:"class_section_subject_teachers_deleted_at,omitempty" gorm:"column:class_section_subject_teachers_deleted_at;index"`
}

func (ClassSectionSubjectTeacherModel) TableName() string {
	return "class_section_subject_teachers"
}
