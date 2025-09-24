package model

import (
	"time"

	"github.com/google/uuid"
)

/* =========================================================
   Enum Status Attempt
   ========================================================= */

type UserQuizAttemptStatus string

const (
	UserQuizAttemptInProgress UserQuizAttemptStatus = "in_progress"
	UserQuizAttemptSubmitted  UserQuizAttemptStatus = "submitted"
	UserQuizAttemptFinished   UserQuizAttemptStatus = "finished"
	UserQuizAttemptAbandoned  UserQuizAttemptStatus = "abandoned"
)

/* =========================================================
   UserQuizAttempt (user_quiz_attempts)
   ========================================================= */

type UserQuizAttemptModel struct {
	// PK
	UserQuizAttemptID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_quiz_attempt_id" json:"user_quiz_attempt_id"`

	// Tenant & relations
	UserQuizAttemptMasjidID  uuid.UUID `gorm:"type:uuid;not null;column:user_quiz_attempt_masjid_id;index:idx_uqa_masjid_quiz,priority:1" json:"user_quiz_attempt_masjid_id"`
	UserQuizAttemptQuizID    uuid.UUID `gorm:"type:uuid;not null;column:user_quiz_attempt_quiz_id;index:idx_uqa_quiz_student,priority:1;index:idx_uqa_quiz_student_started_desc,priority:1;index:idx_uqa_masjid_quiz,priority:2" json:"user_quiz_attempt_quiz_id"`
	UserQuizAttemptStudentID uuid.UUID `gorm:"type:uuid;not null;column:user_quiz_attempt_student_id;index:idx_uqa_quiz_student,priority:2;index:idx_uqa_student;index:idx_uqa_student_status,priority:1" json:"user_quiz_attempt_student_id"`

	// Waktu
	UserQuizAttemptStartedAt  time.Time  `gorm:"type:timestamptz;not null;default:now();column:user_quiz_attempt_started_at" json:"user_quiz_attempt_started_at"`
	UserQuizAttemptFinishedAt *time.Time `gorm:"type:timestamptz;column:user_quiz_attempt_finished_at" json:"user_quiz_attempt_finished_at,omitempty"`

	// Skor
	UserQuizAttemptScoreRaw     *float64 `gorm:"type:numeric(7,3);default:0;column:user_quiz_attempt_score_raw" json:"user_quiz_attempt_score_raw,omitempty"`
	UserQuizAttemptScorePercent *float64 `gorm:"type:numeric(6,3);default:0;column:user_quiz_attempt_score_percent" json:"user_quiz_attempt_score_percent,omitempty"`

	// Status
	UserQuizAttemptStatus UserQuizAttemptStatus `gorm:"type:varchar(16);not null;default:'in_progress';column:user_quiz_attempt_status;index:idx_uqa_status;index:idx_uqa_student_status,priority:2" json:"user_quiz_attempt_status"`

	// Timestamps (custom names)
	UserQuizAttemptCreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:user_quiz_attempt_created_at;index:brin_uqa_created_at,class:BRIN" json:"user_quiz_attempt_created_at"`
	UserQuizAttemptUpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:user_quiz_attempt_updated_at" json:"user_quiz_attempt_updated_at"`

	// Children
	Answers []UserQuizAttemptAnswerModel `gorm:"foreignKey:UserQuizAttemptAnswerAttemptID;references:UserQuizAttemptID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"answers,omitempty"`
}

func (UserQuizAttemptModel) TableName() string { return "user_quiz_attempts" }
