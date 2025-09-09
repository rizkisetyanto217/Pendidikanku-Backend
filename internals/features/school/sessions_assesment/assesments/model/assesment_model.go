// file: internals/features/school/assessments/model/assessment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssessmentModel merepresentasikan tabel `assessments`
type AssessmentModel struct {
	// =========================
	// Primary Key
	// =========================
	AssessmentsID uuid.UUID `json:"assessments_id" gorm:"column:assessments_id;type:uuid;primaryKey"`

	// =========================
	// Tenant / Masjid
	// =========================
	AssessmentsMasjidID uuid.UUID `json:"assessments_masjid_id" gorm:"column:assessments_masjid_id;type:uuid;not null;index:idx_assessments_masjid_created_at,priority:1"`

	// =========================
	// Relasi (FK)
	// =========================
	AssessmentsClassSectionSubjectTeacherID *uuid.UUID `json:"assessments_class_section_subject_teacher_id" gorm:"column:assessments_class_section_subject_teacher_id;type:uuid;index:idx_assessments_csst"`
	AssessmentsTypeID                       *uuid.UUID `json:"assessments_type_id" gorm:"column:assessments_type_id;type:uuid;index:idx_assessments_type_id"`
	AssessmentsCreatedByTeacherID           *uuid.UUID `json:"assessments_created_by_teacher_id" gorm:"column:assessments_created_by_teacher_id;type:uuid;index:idx_assessments_created_by_teacher"`

	// =========================
	// Data Utama
	// =========================
	AssessmentsTitle       string  `json:"assessments_title" gorm:"column:assessments_title;type:varchar(180);not null"`
	AssessmentsDescription *string `json:"assessments_description" gorm:"column:assessments_description;type:text"`

	AssessmentsStartAt *time.Time `json:"assessments_start_at" gorm:"column:assessments_start_at;type:timestamptz"`
	AssessmentsDueAt   *time.Time `json:"assessments_due_at"   gorm:"column:assessments_due_at;type:timestamptz"`

	AssessmentsMaxScore float64 `json:"assessments_max_score" gorm:"column:assessments_max_score;type:numeric(5,2);not null;default:100"`

	AssessmentsIsPublished     bool `json:"assessments_is_published" gorm:"column:assessments_is_published;not null;default:true"`
	AssessmentsAllowSubmission bool `json:"assessments_allow_submission" gorm:"column:assessments_allow_submission;not null;default:true"`

	// =========================
	// Timestamps
	// =========================
	AssessmentsCreatedAt time.Time      `json:"assessments_created_at" gorm:"column:assessments_created_at;not null;autoCreateTime;index:idx_assessments_masjid_created_at,priority:2,sort:desc"`
	AssessmentsUpdatedAt time.Time      `json:"assessments_updated_at" gorm:"column:assessments_updated_at;not null;autoUpdateTime"`
	AssessmentsDeletedAt gorm.DeletedAt `json:"assessments_deleted_at" gorm:"column:assessments_deleted_at;index"`
}

// TableName memastikan mapping ke tabel `assessments`
func (AssessmentModel) TableName() string {
	return "assessments"
}
