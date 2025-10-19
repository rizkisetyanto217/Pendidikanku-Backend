// file: internals/features/assessments/model/assessment_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Enum submission mode (mengikuti CHECK di SQL)
type AssessmentSubmissionMode string

const (
	SubmissionModeDate    AssessmentSubmissionMode = "date"
	SubmissionModeSession AssessmentSubmissionMode = "session"
)

type AssessmentModel struct {
	// PK & Tenant
	AssessmentID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:assessment_id" json:"assessment_id"`
	AssessmentMasjidID uuid.UUID `gorm:"type:uuid;not null;column:assessment_masjid_id" json:"assessment_masjid_id"`

	// Relasi ke CSST (tenant-safe dijaga via composite FK di DB)
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_class_section_subject_teacher_id" json:"assessment_class_section_subject_teacher_id,omitempty"`

	// Tipe penilaian (tenant-safe via composite FK: assessment_type_id + assessment_masjid_id)
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

	// Pengaturan
	AssessmentDurationMinutes      *int    `gorm:"column:assessment_duration_minutes" json:"assessment_duration_minutes,omitempty"`
	AssessmentTotalAttemptsAllowed int     `gorm:"not null;default:1;column:assessment_total_attempts_allowed" json:"assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64 `gorm:"type:numeric(5,2);not null;default:100;column:assessment_max_score" json:"assessment_max_score"`
	AssessmentIsPublished          bool    `gorm:"not null;default:true;column:assessment_is_published" json:"assessment_is_published"`
	AssessmentAllowSubmission      bool    `gorm:"not null;default:true;column:assessment_allow_submission" json:"assessment_allow_submission"`

	// Audit pembuat (opsional)
	AssessmentCreatedByTeacherID *uuid.UUID `gorm:"type:uuid;column:assessment_created_by_teacher_id" json:"assessment_created_by_teacher_id,omitempty"`

	// Mode pengumpulan (by date / by session)
	AssessmentSubmissionMode    AssessmentSubmissionMode `gorm:"type:text;not null;default:'date';column:assessment_submission_mode" json:"assessment_submission_mode"`
	AssessmentAnnounceSessionID *uuid.UUID               `gorm:"type:uuid;column:assessment_announce_session_id" json:"assessment_announce_session_id,omitempty"`
	AssessmentCollectSessionID  *uuid.UUID               `gorm:"type:uuid;column:assessment_collect_session_id" json:"assessment_collect_session_id,omitempty"`

	// 🔹 Snapshots (JSONB, NOT NULL, default '{}'::jsonb)
	AssessmentCSSTSnapshot            datatypes.JSONMap `gorm:"type:jsonb;not null;default:'{}';column:assessment_csst_snapshot" json:"assessment_csst_snapshot,omitempty"`
	AssessmentAnnounceSessionSnapshot datatypes.JSONMap `gorm:"type:jsonb;not null;default:'{}';column:assessment_announce_session_snapshot" json:"assessment_announce_session_snapshot,omitempty"`
	AssessmentCollectSessionSnapshot  datatypes.JSONMap `gorm:"type:jsonb;not null;default:'{}';column:assessment_collect_session_snapshot" json:"assessment_collect_session_snapshot,omitempty"`

	// Timestamps & soft delete
	AssessmentCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_created_at" json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_updated_at" json:"assessment_updated_at"`
	AssessmentDeletedAt gorm.DeletedAt `gorm:"column:assessment_deleted_at;index" json:"assessment_deleted_at,omitempty"`
}

func (AssessmentModel) TableName() string { return "assessments" }
