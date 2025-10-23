package model

import (
	"time"

	"github.com/google/uuid"
)

/* =========================================================
   Enum Status Attempt (student_quiz_attempts)
   ========================================================= */

type StudentQuizAttemptStatus string

const (
	StudentQuizAttemptInProgress StudentQuizAttemptStatus = "in_progress"
	StudentQuizAttemptSubmitted  StudentQuizAttemptStatus = "submitted"
	StudentQuizAttemptFinished   StudentQuizAttemptStatus = "finished"
	StudentQuizAttemptAbandoned  StudentQuizAttemptStatus = "abandoned"
)

/* =========================================================
   StudentQuizAttempt (student_quiz_attempts)
   ========================================================= */

type StudentQuizAttemptModel struct {
	// PK
	StudentQuizAttemptID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_quiz_attempt_id" json:"student_quiz_attempt_id"`

	// Tenant & relations
	StudentQuizAttemptMasjidID  uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_masjid_id;index:idx_sqa_masjid_quiz,priority:1" json:"student_quiz_attempt_masjid_id"`
	StudentQuizAttemptQuizID    uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_quiz_id;index:idx_sqa_quiz_student,priority:1;index:idx_sqa_quiz_student_started_desc,priority:1;index:idx_sqa_masjid_quiz,priority:2" json:"student_quiz_attempt_quiz_id"`
	StudentQuizAttemptStudentID uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_student_id;index:idx_sqa_quiz_student,priority:2;index:idx_sqa_student;index:idx_sqa_student_status,priority:1" json:"student_quiz_attempt_student_id"`

	// Waktu
	StudentQuizAttemptStartedAt  time.Time  `gorm:"type:timestamptz;not null;default:now();column:student_quiz_attempt_started_at;index:brin_sqa_started_at,class:BRIN" json:"student_quiz_attempt_started_at"`
	StudentQuizAttemptFinishedAt *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_finished_at" json:"student_quiz_attempt_finished_at,omitempty"`

	// Skor
	StudentQuizAttemptScoreRaw     *float64 `gorm:"type:numeric(7,3);default:0;column:student_quiz_attempt_score_raw" json:"student_quiz_attempt_score_raw,omitempty"`
	StudentQuizAttemptScorePercent *float64 `gorm:"type:numeric(6,3);default:0;column:student_quiz_attempt_score_percent" json:"student_quiz_attempt_score_percent,omitempty"`

	// Status
	StudentQuizAttemptStatus StudentQuizAttemptStatus `gorm:"type:varchar(16);not null;default:'in_progress';column:student_quiz_attempt_status;index:idx_sqa_status;index:idx_sqa_student_status,priority:2" json:"student_quiz_attempt_status"`

	// Timestamps (custom names)
	StudentQuizAttemptCreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:student_quiz_attempt_created_at;index:brin_sqa_created_at,class:BRIN" json:"student_quiz_attempt_created_at"`
	StudentQuizAttemptUpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:student_quiz_attempt_updated_at" json:"student_quiz_attempt_updated_at"`

	// Children
	Answers []StudentQuizAttemptAnswerModel `gorm:"foreignKey:StudentQuizAttemptAnswerAttemptID;references:StudentQuizAttemptID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"answers,omitempty"`
}

func (StudentQuizAttemptModel) TableName() string { return "student_quiz_attempts" }
