// file: internals/features/assessments/model/assessment_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssessmentTypeModel struct {
	AssessmentTypeID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:assessment_type_id" json:"assessment_type_id"`
	AssessmentTypeSchoolID uuid.UUID `gorm:"type:uuid;not null;column:assessment_type_school_id" json:"assessment_type_school_id"`

	AssessmentTypeKey  string `gorm:"type:varchar(32);not null;column:assessment_type_key" json:"assessment_type_key"`
	AssessmentTypeName string `gorm:"type:varchar(120);not null;column:assessment_type_name" json:"assessment_type_name"`

	// Bobot nilai akhir
	AssessmentTypeWeightPercent float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_type_weight_percent" json:"assessment_type_weight_percent"`

	// ============== Default Quiz Settings (dari QuizSettings React) ==============

	// Acak urutan pertanyaan
	AssessmentTypeShuffleQuestions bool `gorm:"not null;default:false;column:assessment_type_shuffle_questions" json:"assessment_type_shuffle_questions"`

	// Acak urutan opsi jawaban
	AssessmentTypeShuffleOptions bool `gorm:"not null;default:false;column:assessment_type_shuffle_options" json:"assessment_type_shuffle_options"`

	// Tampilkan jawaban benar / review setelah submit
	AssessmentTypeShowCorrectAfterSubmit bool `gorm:"not null;default:true;column:assessment_type_show_correct_after_submit" json:"assessment_type_show_correct_after_submit"`

	// Satu pertanyaan per halaman
	AssessmentTypeOneQuestionPerPage bool `gorm:"not null;default:false;column:assessment_type_one_question_per_page" json:"assessment_type_one_question_per_page"`

	// Batas waktu (menit); NULL = tanpa batas
	AssessmentTypeTimeLimitMin *int `gorm:"type:int;column:assessment_type_time_limit_min" json:"assessment_type_time_limit_min"`

	// Maksimal percobaan (min 1)
	AssessmentTypeAttemptsAllowed int `gorm:"type:int;not null;default:1;column:assessment_type_attempts_allowed" json:"assessment_type_attempts_allowed"`

	// Wajib login
	AssessmentTypeRequireLogin bool `gorm:"not null;default:true;column:assessment_type_require_login" json:"assessment_type_require_login"`

	// Blok tombol back (kurangi kecurangan)
	AssessmentTypePreventBackNavigation bool `gorm:"not null;default:false;column:assessment_type_prevent_back_navigation" json:"assessment_type_prevent_back_navigation"`

	// Status aktif type ini
	AssessmentTypeIsActive bool `gorm:"not null;default:true;column:assessment_type_is_active" json:"assessment_type_is_active"`

	AssessmentTypeCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_type_created_at" json:"assessment_type_created_at"`
	AssessmentTypeUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_type_updated_at" json:"assessment_type_updated_at"`
	AssessmentTypeDeletedAt gorm.DeletedAt `gorm:"column:assessment_type_deleted_at;index" json:"assessment_type_deleted_at,omitempty"`
}

func (AssessmentTypeModel) TableName() string { return "assessment_types" }
