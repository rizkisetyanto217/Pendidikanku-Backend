// file: internals/features/school/submissions_assesments/quizzes/model/quiz_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
   Model: quizzes
   ========================= */

type QuizModel struct {
	// PK & Tenant
	QuizID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:quiz_id" json:"quiz_id"`
	QuizSchoolID uuid.UUID `gorm:"type:uuid;not null;column:quiz_school_id" json:"quiz_school_id"`

	// Relasi ke assessment (opsional)
	QuizAssessmentID *uuid.UUID `gorm:"type:uuid;column:quiz_assessment_id" json:"quiz_assessment_id,omitempty"`

	// Relasi langsung ke assessment_types (opsional)
	QuizAssessmentTypeID *uuid.UUID `gorm:"type:uuid;column:quiz_assessment_type_id" json:"quiz_assessment_type_id,omitempty"`

	// Identitas
	QuizSlug        *string `gorm:"type:varchar(160);column:quiz_slug" json:"quiz_slug,omitempty"`
	QuizTitle       string  `gorm:"type:varchar(180);not null;column:quiz_title" json:"quiz_title"`
	QuizDescription *string `gorm:"type:text;column:quiz_description" json:"quiz_description,omitempty"`

	// Pengaturan dasar
	QuizIsPublished    bool `gorm:"not null;default:false;column:quiz_is_published" json:"quiz_is_published"`
	QuizTimeLimitSec   *int `gorm:"type:int;column:quiz_time_limit_sec" json:"quiz_time_limit_sec,omitempty"`
	QuizTotalQuestions int  `gorm:"type:int;not null;default:0;column:quiz_total_questions" json:"quiz_total_questions"`

	// Timestamps & soft delete
	QuizCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:quiz_created_at" json:"quiz_created_at"`
	QuizUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:quiz_updated_at" json:"quiz_updated_at"`
	QuizDeletedAt gorm.DeletedAt `gorm:"column:quiz_deleted_at;index" json:"quiz_deleted_at,omitempty"`
}

func (QuizModel) TableName() string { return "quizzes" }
