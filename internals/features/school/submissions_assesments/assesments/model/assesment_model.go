// file: internals/features/assessments/model/assessment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssessmentModel struct {
	AssessmentID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:assessment_id" json:"assessment_id"`
	AssessmentMasjidID uuid.UUID `gorm:"type:uuid;not null;column:assessment_masjid_id" json:"assessment_masjid_id"`

	// Relasi ke CSST (tenant-safe dijaga di DB)
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_class_section_subject_teacher_id" json:"assessment_class_section_subject_teacher_id,omitempty"`

	// Tipe penilaian (opsional)
	AssessmentTypeID *uuid.UUID `gorm:"type:uuid;column:assessment_type_id" json:"assessment_type_id,omitempty"`

	// Identitas
	AssessmentSlug        *string `gorm:"type:varchar(160);column:assessment_slug" json:"assessment_slug,omitempty"`
	AssessmentTitle       string  `gorm:"type:varchar(180);not null;column:assessment_title" json:"assessment_title"`
	AssessmentDescription *string `gorm:"type:text;column:assessment_description" json:"assessment_description,omitempty"`

	// Jadwal
	AssessmentStartAt     *time.Time `gorm:"type:timestamptz;column:assessment_start_at" json:"assessment_start_at,omitempty"`
	AssessmentDueAt       *time.Time `gorm:"type:timestamptz;column:assessment_due_at" json:"assessment_due_at,omitempty"`
	AssessmentPublishedAt *time.Time `gorm:"type:timestamptz;column:assessment_published_at" json:"assessment_published_at,omitempty"`
	AssessmentClosedAt    *time.Time `gorm:"type:timestamptz;column:assessment_closed_at" json:"assessment_closed_at,omitempty"`

	// Pengaturan
	AssessmentDurationMinutes      *int    `gorm:"column:assessment_duration_minutes" json:"assessment_duration_minutes,omitempty"`
	AssessmentTotalAttemptsAllowed int     `gorm:"not null;default:1;column:assessment_total_attempts_allowed" json:"assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64 `gorm:"type:numeric(5,2);not null;default:100;column:assessment_max_score" json:"assessment_max_score"`
	AssessmentIsPublished          bool    `gorm:"not null;default:true;column:assessment_is_published" json:"assessment_is_published"`
	AssessmentAllowSubmission      bool    `gorm:"not null;default:true;column:assessment_allow_submission" json:"assessment_allow_submission"`

	// Audit pembuat (opsional)
	AssessmentCreatedByTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_created_by_teacher_id" json:"assessment_created_by_teacher_id,omitempty"`

	AssessmentCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_created_at" json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_updated_at" json:"assessment_updated_at"`
	AssessmentDeletedAt gorm.DeletedAt `gorm:"column:assessment_deleted_at;index" json:"assessment_deleted_at,omitempty"`
}

func (AssessmentModel) TableName() string { return "assessments" }
