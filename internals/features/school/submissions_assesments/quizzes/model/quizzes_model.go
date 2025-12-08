package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
   Quiz (quizzes)
   ========================================================= */

type QuizModel struct {
	// PK
	QuizID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:quiz_id" json:"quiz_id"`

	// Tenant
	QuizSchoolID uuid.UUID `gorm:"type:uuid;not null;column:quiz_school_id" json:"quiz_school_id"`

	// Opsional relasi ke assessments
	QuizAssessmentID *uuid.UUID `gorm:"type:uuid;column:quiz_assessment_id" json:"quiz_assessment_id,omitempty"`

	// Slug (opsional; unik per tenant saat alive via migration)
	QuizSlug *string `gorm:"type:varchar(160);column:quiz_slug" json:"quiz_slug,omitempty"`

	// Konten
	QuizTitle        string  `gorm:"type:varchar(180);not null;column:quiz_title" json:"quiz_title"`
	QuizDescription  *string `gorm:"type:text;column:quiz_description" json:"quiz_description,omitempty"`
	QuizIsPublished  bool    `gorm:"type:boolean;not null;default:false;column:quiz_is_published" json:"quiz_is_published"`
	QuizTimeLimitSec *int    `gorm:"type:int;column:quiz_time_limit_sec" json:"quiz_time_limit_sec,omitempty"`

	// ==============================
	// Snapshot quiz behaviour & scoring
	// ==============================

	// tampilan & UX pengerjaan
	QuizShuffleQuestionsSnapshot            bool `gorm:"type:boolean;not null;default:false;column:quiz_shuffle_questions_snapshot" json:"quiz_shuffle_questions_snapshot"`
	QuizShuffleOptionsSnapshot              bool `gorm:"type:boolean;not null;default:false;column:quiz_shuffle_options_snapshot" json:"quiz_shuffle_options_snapshot"`
	QuizShowCorrectAfterSubmitSnapshot      bool `gorm:"type:boolean;not null;default:true;column:quiz_show_correct_after_submit_snapshot" json:"quiz_show_correct_after_submit_snapshot"`
	QuizStrictModeSnapshot                  bool `gorm:"type:boolean;not null;default:false;column:quiz_strict_mode_snapshot" json:"quiz_strict_mode_snapshot"`
	QuizTimeLimitMinSnapshot                *int `gorm:"type:int;column:quiz_time_limit_min_snapshot" json:"quiz_time_limit_min_snapshot,omitempty"`
	QuizRequireLoginSnapshot                bool `gorm:"type:boolean;not null;default:true;column:quiz_require_login_snapshot" json:"quiz_require_login_snapshot"`
	QuizShowScoreAfterSubmitSnapshot        bool `gorm:"type:boolean;not null;default:true;column:quiz_show_score_after_submit_snapshot" json:"quiz_show_score_after_submit_snapshot"`
	QuizShowCorrectAfterClosedSnapshot      bool `gorm:"type:boolean;not null;default:false;column:quiz_show_correct_after_closed_snapshot" json:"quiz_show_correct_after_closed_snapshot"`
	QuizAllowReviewBeforeSubmitSnapshot     bool `gorm:"type:boolean;not null;default:true;column:quiz_allow_review_before_submit_snapshot" json:"quiz_allow_review_before_submit_snapshot"`
	QuizRequireCompleteAttemptSnapshot      bool `gorm:"type:boolean;not null;default:true;column:quiz_require_complete_attempt_snapshot" json:"quiz_require_complete_attempt_snapshot"`
	QuizShowDetailsAfterAllAttemptsSnapshot bool `gorm:"type:boolean;not null;default:false;column:quiz_show_details_after_all_attempts_snapshot" json:"quiz_show_details_after_all_attempts_snapshot"`

	// attempts & agregasi nilai (final score dari attempts quiz)
	QuizAttemptsAllowedSnapshot      int    `gorm:"type:int;not null;default:1;column:quiz_attempts_allowed_snapshot" json:"quiz_attempts_allowed_snapshot"`
	QuizScoreAggregationModeSnapshot string `gorm:"type:varchar(20);not null;default:'latest';column:quiz_score_aggregation_mode_snapshot" json:"quiz_score_aggregation_mode_snapshot"`

	// 🔢 snapshot jumlah soal (denorm)
	QuizTotalQuestions int `gorm:"type:int;not null;default:0;column:quiz_total_questions" json:"quiz_total_questions"`

	// Timestamps (custom names)
	QuizCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:quiz_created_at" json:"quiz_created_at"`
	QuizUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:quiz_updated_at" json:"quiz_updated_at"`
	QuizDeletedAt gorm.DeletedAt `gorm:"column:quiz_deleted_at;index" json:"quiz_deleted_at,omitempty"`

	// Children
	Questions []QuizQuestionModel `gorm:"foreignKey:QuizQuestionQuizID;references:QuizID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"questions,omitempty"`
}

func (QuizModel) TableName() string { return "quizzes" }
