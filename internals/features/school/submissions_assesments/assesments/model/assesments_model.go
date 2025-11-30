// file: internals/features/assessments/model/assessment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// =========================
// Enum: Submission Mode
// =========================

type AssessmentSubmissionMode string

const (
	SubmissionModeDate    AssessmentSubmissionMode = "date"
	SubmissionModeSession AssessmentSubmissionMode = "session"
)

// =========================
// Enum: Assessment Kind
// =========================

type AssessmentKind string

const (
	AssessmentKindQuiz             AssessmentKind = "quiz"
	AssessmentKindAssignmentUpload AssessmentKind = "assignment_upload"
	AssessmentKindOffline          AssessmentKind = "offline"
	AssessmentKindSurvey           AssessmentKind = "survey"
)

// =========================
// Model: assessments
// =========================

type AssessmentModel struct {
	// PK & Tenant
	AssessmentID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:assessment_id" json:"assessment_id"`
	AssessmentSchoolID uuid.UUID `gorm:"type:uuid;not null;column:assessment_school_id" json:"assessment_school_id"`

	// Relasi ke CSST (single FK)
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_class_section_subject_teacher_id" json:"assessment_class_section_subject_teacher_id,omitempty"`

	// Tipe penilaian
	AssessmentTypeID *uuid.UUID `gorm:"type:uuid;column:assessment_type_id" json:"assessment_type_id,omitempty"`

	// Identitas
	AssessmentSlug        *string `gorm:"type:varchar(160);column:assessment_slug" json:"assessment_slug,omitempty"`
	AssessmentTitle       string  `gorm:"type:varchar(180);not null;column:assessment_title" json:"assessment_title"`
	AssessmentDescription *string `gorm:"type:text;column:assessment_description" json:"assessment_description,omitempty"`

	// Jadwal (mode 'date')
	AssessmentStartAt     *time.Time `gorm:"type:timestamptz;column:assessment_start_at" json:"assessment_start_at,omitempty"`
	AssessmentDueAt       *time.Time `gorm:"type:timestamptz;column:assessment_due_at" json:"assessment_due_at,omitempty"`
	AssessmentPublishedAt *time.Time `gorm:"type:timestamptz;column:assessment_published_at" json:"assessment_published_at,omitempty"`
	AssessmentClosedAt    *time.Time `gorm:"type:timestamptz;column:assessment_closed_at" json:"assessment_closed_at,omitempty"`

	// Pengaturan dasar assessment
	AssessmentKind                 AssessmentKind `gorm:"type:assessment_kind_enum;not null;default:'quiz';column:assessment_kind" json:"assessment_kind"`
	AssessmentDurationMinutes      *int           `gorm:"column:assessment_duration_minutes" json:"assessment_duration_minutes,omitempty"`
	AssessmentTotalAttemptsAllowed int            `gorm:"not null;default:1;column:assessment_total_attempts_allowed" json:"assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64        `gorm:"type:numeric(5,2);not null;default:100;column:assessment_max_score" json:"assessment_max_score"`

	// total quiz/komponen quiz di assessment ini
	AssessmentQuizTotal int `gorm:"type:smallint;not null;default:0;column:assessment_quiz_total" json:"assessment_quiz_total"`

	// agregat submissions (diupdate dari service)
	AssessmentSubmissionsTotal       int `gorm:"not null;default:0;column:assessment_submissions_total" json:"assessment_submissions_total"`
	AssessmentSubmissionsGradedTotal int `gorm:"not null;default:0;column:assessment_submissions_graded_total" json:"assessment_submissions_graded_total"`

	AssessmentIsPublished     bool `gorm:"not null;default:true;column:assessment_is_published" json:"assessment_is_published"`
	AssessmentAllowSubmission bool `gorm:"not null;default:true;column:assessment_allow_submission" json:"assessment_allow_submission"`

	// Flag apakah assessment type ini menghasilkan nilai (graded) — snapshot
	AssessmentTypeIsGradedSnapshot bool `gorm:"not null;default:false;column:assessment_type_is_graded_snapshot" json:"assessment_type_is_graded_snapshot"`

	// =========================
	// Snapshot aturan dari AssessmentType (per assessment)
	// =========================

	// Quiz behaviour
	AssessmentShuffleQuestionsSnapshot       bool   `gorm:"not null;default:false;column:assessment_shuffle_questions_snapshot" json:"assessment_shuffle_questions_snapshot"`
	AssessmentShuffleOptionsSnapshot         bool   `gorm:"not null;default:false;column:assessment_shuffle_options_snapshot" json:"assessment_shuffle_options_snapshot"`
	AssessmentShowCorrectAfterSubmitSnapshot bool   `gorm:"not null;default:true;column:assessment_show_correct_after_submit_snapshot" json:"assessment_show_correct_after_submit_snapshot"`
	AssessmentStrictModeSnapshot             bool   `gorm:"not null;default:false;column:assessment_strict_mode_snapshot" json:"assessment_strict_mode_snapshot"`
	AssessmentTimeLimitMinSnapshot           *int   `gorm:"column:assessment_time_limit_min_snapshot" json:"assessment_time_limit_min_snapshot,omitempty"`
	AssessmentAttemptsAllowedSnapshot        int    `gorm:"not null;default:1;column:assessment_attempts_allowed_snapshot" json:"assessment_attempts_allowed_snapshot"`
	AssessmentRequireLoginSnapshot           bool   `gorm:"not null;default:true;column:assessment_require_login_snapshot" json:"assessment_require_login_snapshot"`
	AssessmentScoreAggregationModeSnapshot   string `gorm:"type:varchar(20);not null;default:'latest';column:assessment_score_aggregation_mode_snapshot" json:"assessment_score_aggregation_mode_snapshot"`

	// Late policy & visibility
	AssessmentAllowLateSubmissionSnapshot         bool    `gorm:"not null;default:false;column:assessment_allow_late_submission_snapshot" json:"assessment_allow_late_submission_snapshot"`
	AssessmentLatePenaltyPercentSnapshot          float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_late_penalty_percent_snapshot" json:"assessment_late_penalty_percent_snapshot"`
	AssessmentPassingScorePercentSnapshot         float64 `gorm:"type:numeric(5,2);not null;default:0;column:assessment_passing_score_percent_snapshot" json:"assessment_passing_score_percent_snapshot"`
	AssessmentShowScoreAfterSubmitSnapshot        bool    `gorm:"not null;default:true;column:assessment_show_score_after_submit_snapshot" json:"assessment_show_score_after_submit_snapshot"`
	AssessmentShowCorrectAfterClosedSnapshot      bool    `gorm:"not null;default:false;column:assessment_show_correct_after_closed_snapshot" json:"assessment_show_correct_after_closed_snapshot"`
	AssessmentAllowReviewBeforeSubmitSnapshot     bool    `gorm:"not null;default:true;column:assessment_allow_review_before_submit_snapshot" json:"assessment_allow_review_before_submit_snapshot"`
	AssessmentRequireCompleteAttemptSnapshot      bool    `gorm:"not null;default:true;column:assessment_require_complete_attempt_snapshot" json:"assessment_require_complete_attempt_snapshot"`
	AssessmentShowDetailsAfterAllAttemptsSnapshot bool    `gorm:"not null;default:false;column:assessment_show_details_after_all_attempts_snapshot" json:"assessment_show_details_after_all_attempts_snapshot"`

	// Audit pembuat (opsional)
	AssessmentCreatedByTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_created_by_teacher_id" json:"assessment_created_by_teacher_id,omitempty"`

	// Snapshots relasi (CSST & sesi kehadiran)
	AssessmentCSSTSnapshot            datatypes.JSONMap `gorm:"type:jsonb;not null;default:'{}';column:assessment_csst_snapshot" json:"assessment_csst_snapshot,omitempty"`
	AssessmentAnnounceSessionSnapshot datatypes.JSONMap `gorm:"type:jsonb;not null;default:'{}';column:assessment_announce_session_snapshot" json:"assessment_announce_session_snapshot,omitempty"`
	AssessmentCollectSessionSnapshot  datatypes.JSONMap `gorm:"type:jsonb;not null;default:'{}';column:assessment_collect_session_snapshot" json:"assessment_collect_session_snapshot,omitempty"`

	// Mode pengumpulan (by date / by session)
	AssessmentSubmissionMode    AssessmentSubmissionMode `gorm:"type:text;not null;default:'date';column:assessment_submission_mode" json:"assessment_submission_mode"`
	AssessmentAnnounceSessionID *uuid.UUID               `gorm:"type:uuid;column:assessment_announce_session_id" json:"assessment_announce_session_id,omitempty"`
	AssessmentCollectSessionID  *uuid.UUID               `gorm:"type:uuid;column:assessment_collect_session_id" json:"assessment_collect_session_id,omitempty"`

	// Timestamps & soft delete
	AssessmentCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_created_at" json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_updated_at" json:"assessment_updated_at"`
	AssessmentDeletedAt gorm.DeletedAt `gorm:"column:assessment_deleted_at;index" json:"assessment_deleted_at,omitempty"`
}

func (AssessmentModel) TableName() string { return "assessments" }
