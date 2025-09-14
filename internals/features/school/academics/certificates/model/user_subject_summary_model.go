// models/user_subject_summary.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type UserSubjectSummary struct {
	UserSubjectSummaryID                 uuid.UUID         `gorm:"column:user_subject_summary_id;type:uuid;default:gen_random_uuid();primaryKey" json:"user_subject_summary_id"`
	UserSubjectSummaryMasjidID           uuid.UUID         `gorm:"column:user_subject_summary_masjid_id;type:uuid;not null" json:"user_subject_summary_masjid_id"`
	UserSubjectSummaryMasjidStudentID    uuid.UUID         `gorm:"column:user_subject_summary_masjid_student_id;type:uuid;not null" json:"user_subject_summary_masjid_student_id"`
	UserSubjectSummaryClassSubjectsID    uuid.UUID         `gorm:"column:user_subject_summary_class_subjects_id;type:uuid;not null" json:"user_subject_summary_class_subjects_id"`
	UserSubjectSummaryCSSTID             *uuid.UUID        `gorm:"column:user_subject_summary_csst_id;type:uuid" json:"user_subject_summary_csst_id,omitempty"`
	UserSubjectSummaryTermID             *uuid.UUID        `gorm:"column:user_subject_summary_term_id;type:uuid" json:"user_subject_summary_term_id,omitempty"`
	UserSubjectSummaryFinalAssessmentID  *uuid.UUID        `gorm:"column:user_subject_summary_final_assessment_id;type:uuid" json:"user_subject_summary_final_assessment_id,omitempty"`

	UserSubjectSummaryFinalScore         *float64          `gorm:"column:user_subject_summary_final_score;type:numeric(5,2)" json:"user_subject_summary_final_score,omitempty"`
	UserSubjectSummaryPassThreshold      float64           `gorm:"column:user_subject_summary_pass_threshold;type:numeric(5,2);not null;default:70" json:"user_subject_summary_pass_threshold"`
	UserSubjectSummaryPassed             bool              `gorm:"column:user_subject_summary_passed;not null;default:false" json:"user_subject_summary_passed"`

	// snapshot komponen (tugas/kuis/uts/uas) — diisi backend
	UserSubjectSummaryBreakdown          datatypes.JSONMap `gorm:"column:user_subject_summary_breakdown;type:jsonb" json:"user_subject_summary_breakdown,omitempty"`

	// metrik progres (opsional)
	UserSubjectSummaryTotalAssessments        *int       `gorm:"column:user_subject_summary_total_assessments" json:"user_subject_summary_total_assessments,omitempty"`
	UserSubjectSummaryTotalCompletedAttempts  *int       `gorm:"column:user_subject_summary_total_completed_attempts" json:"user_subject_summary_total_completed_attempts,omitempty"`
	UserSubjectSummaryLastAssessedAt          *time.Time `gorm:"column:user_subject_summary_last_assessed_at" json:"user_subject_summary_last_assessed_at,omitempty"`

	// sertifikat & catatan
	UserSubjectSummaryCertificateGenerated bool       `gorm:"column:user_subject_summary_certificate_generated;not null;default:false" json:"user_subject_summary_certificate_generated"`
	UserSubjectSummaryNote                *string    `gorm:"column:user_subject_summary_note" json:"user_subject_summary_note,omitempty"`

	UserSubjectSummaryCreatedAt           time.Time  `gorm:"column:user_subject_summary_created_at;not null;default:now()" json:"user_subject_summary_created_at"`
	UserSubjectSummaryUpdatedAt           time.Time  `gorm:"column:user_subject_summary_updated_at;not null;default:now()" json:"user_subject_summary_updated_at"`
	UserSubjectSummaryDeletedAt           *time.Time `gorm:"column:user_subject_summary_deleted_at" json:"user_subject_summary_deleted_at,omitempty"`
}

func (UserSubjectSummary) TableName() string { return "user_subject_summary" }
