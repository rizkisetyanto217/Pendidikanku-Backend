// file: internals/features/school/assessments/model/assessment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssessmentModel merepresentasikan tabel `assessments`
type AssessmentModel struct {
	// PK
	ID uuid.UUID `json:"assessments_id" gorm:"column:assessments_id;type:uuid;primaryKey"`

	// Tenant
	MasjidID uuid.UUID `json:"assessments_masjid_id" gorm:"column:assessments_masjid_id;type:uuid;not null;index:idx_assessments_masjid_created_at,priority:1"`

	// Relasi opsional (boleh NULL)
	ClassSectionID                    *uuid.UUID `json:"assessments_class_section_id" gorm:"column:assessments_class_section_id;type:uuid;index:idx_assessments_section"`
	ClassSubjectsID                   *uuid.UUID `json:"assessments_class_subjects_id" gorm:"column:assessments_class_subjects_id;type:uuid;index:idx_assessments_subject"`
	ClassSectionSubjectTeacherID      *uuid.UUID `json:"assessments_class_section_subject_teacher_id" gorm:"column:assessments_class_section_subject_teacher_id;type:uuid;index:idx_assessments_csst"`

	// Tipe penilaian (FK ke assessment_types)
	TypeID *uuid.UUID `json:"assessments_type_id" gorm:"column:assessments_type_id;type:uuid;index:idx_assessments_type_id"`

	// Data utama
	Title       string  `json:"assessments_title" gorm:"column:assessments_title;type:varchar(180);not null"`
	Description *string `json:"assessments_description" gorm:"column:assessments_description;type:text"`

	StartAt *time.Time `json:"assessments_start_at" gorm:"column:assessments_start_at;type:timestamptz"`
	DueAt   *time.Time `json:"assessments_due_at"   gorm:"column:assessments_due_at;type:timestamptz"`

	MaxScore float32 `json:"assessments_max_score" gorm:"column:assessments_max_score;type:numeric(5,2);not null;default:100"`

	IsPublished     bool `json:"assessments_is_published" gorm:"column:assessments_is_published;not null;default:true"`
	AllowSubmission bool `json:"assessments_allow_submission" gorm:"column:assessments_allow_submission;not null;default:true"`

	// Creator (FK ke masjid_teachers, global guru — beda dengan CSST)
	CreatedByTeacherID *uuid.UUID `json:"assessments_created_by_teacher_id" gorm:"column:assessments_created_by_teacher_id;type:uuid;index:idx_assessments_created_by_teacher"`

	// Timestamps
	CreatedAt time.Time      `json:"assessments_created_at" gorm:"column:assessments_created_at;not null;autoCreateTime;index:idx_assessments_masjid_created_at,priority:2,sort:desc"`
	UpdatedAt time.Time      `json:"assessments_updated_at" gorm:"column:assessments_updated_at;not null;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"assessments_deleted_at" gorm:"column:assessments_deleted_at;index"`
}

// TableName memastikan nama tabel sesuai DDL
func (AssessmentModel) TableName() string { return "assessments" }
