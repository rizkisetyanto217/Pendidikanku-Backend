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

	// Bobot nilai akhir (0–100, boleh >100 kalau mau over-weight)
	AssessmentTypeWeightPercent float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_type_weight_percent" json:"assessment_type_weight_percent"`

	// ============== Default Quiz Settings (dari QuizSettings React) ==============

	// Acak urutan pertanyaan
	AssessmentTypeShuffleQuestions bool `gorm:"not null;default:false;column:assessment_type_shuffle_questions" json:"assessment_type_shuffle_questions"`

	// Acak urutan opsi jawaban
	AssessmentTypeShuffleOptions bool `gorm:"not null;default:false;column:assessment_type_shuffle_options" json:"assessment_type_shuffle_options"`

	// Tampilkan jawaban benar / review setelah submit
	AssessmentTypeShowCorrectAfterSubmit bool `gorm:"not null;default:true;column:assessment_type_show_correct_after_submit" json:"assessment_type_show_correct_after_submit"`

	// ❌ DULU ada one_question_per_page & prevent_back_navigation di sini
	// ✅ Sekarang diganti jadi strict mode
	// Mode ketat (strict) — nanti di FE/BE bisa di-artikan:
	// - satu soal per halaman
	// - tidak boleh back
	// - tidak tampilkan kunci sebelum close
	// dsb (aturan detail diatur di layer lain)
	AssessmentTypeStrictMode bool `gorm:"not null;default:false;column:assessment_type_strict_mode" json:"assessment_type_strict_mode"`

	// Batas waktu (menit); NULL = tanpa batas
	AssessmentTypeTimeLimitMin *int `gorm:"type:int;column:assessment_type_time_limit_min" json:"assessment_type_time_limit_min"`

	// Maksimal percobaan (min 1)
	AssessmentTypeAttemptsAllowed int `gorm:"type:int;not null;default:1;column:assessment_type_attempts_allowed" json:"assessment_type_attempts_allowed"`

	// Wajib login
	AssessmentTypeRequireLogin bool `gorm:"not null;default:true;column:assessment_type_require_login" json:"assessment_type_require_login"`

	// Status aktif type ini
	AssessmentTypeIsActive bool `gorm:"not null;default:true;column:assessment_type_is_active" json:"assessment_type_is_active"`

	// Type ini menghasilkan nilai (graded) atau cuma hadir / survey / polling
	AssessmentTypeIsGraded bool `gorm:"not null;default:true;column:assessment_type_is_graded" json:"assessment_type_is_graded"`

	// ============== Default Late Policy ==============

	AssessmentTypeAllowLateSubmission bool    `gorm:"not null;default:false;column:assessment_type_allow_late_submission" json:"assessment_type_allow_late_submission"`
	AssessmentTypeLatePenaltyPercent  float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_type_late_penalty_percent" json:"assessment_type_late_penalty_percent"`
	AssessmentTypePassingScorePercent float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_type_passing_score_percent" json:"assessment_type_passing_score_percent"`

	AssessmentTypeScoreAggregationMode string `gorm:"type:varchar(20);not null;default:'latest';column:assessment_type_score_aggregation_mode" json:"assessment_type_score_aggregation_mode"`

	AssessmentTypeShowScoreAfterSubmit        bool `gorm:"not null;default:true;column:assessment_type_show_score_after_submit" json:"assessment_type_show_score_after_submit"`
	AssessmentTypeShowCorrectAfterClosed      bool `gorm:"not null;default:false;column:assessment_type_show_correct_after_closed" json:"assessment_type_show_correct_after_closed"`
	AssessmentTypeAllowReviewBeforeSubmit     bool `gorm:"not null;default:true;column:assessment_type_allow_review_before_submit" json:"assessment_type_allow_review_before_submit"`
	AssessmentTypeRequireCompleteAttempt      bool `gorm:"not null;default:true;column:assessment_type_require_complete_attempt" json:"assessment_type_require_complete_attempt"`
	AssessmentTypeShowDetailsAfterAllAttempts bool `gorm:"not null;default:false;column:assessment_type_show_details_after_all_attempts" json:"assessment_type_show_details_after_all_attempts"`

	AssessmentTypeCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_type_created_at" json:"assessment_type_created_at"`
	AssessmentTypeUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_type_updated_at" json:"assessment_type_updated_at"`
	AssessmentTypeDeletedAt gorm.DeletedAt `gorm:"column:assessment_type_deleted_at;index" json:"assessment_type_deleted_at,omitempty"`
}

func (AssessmentTypeModel) TableName() string { return "assessment_types" }
