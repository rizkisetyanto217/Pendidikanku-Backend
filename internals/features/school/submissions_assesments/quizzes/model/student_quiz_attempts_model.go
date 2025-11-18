// file: internals/features/school/submissions_assesments/quizzes/model/student_quiz_attempt_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/*
=========================================================

	STUDENT QUIZ ATTEMPTS (JSON VERSION)
	1 row = 1 student × 1 quiz
	- history: semua attempt dalam JSONB
	- best_* : nilai terbaik
	- last_* : nilai attempt terakhir
	- count  : total attempt

=========================================================
*/

// Enum status attempt (dipakai controller: qmodel.StudentQuizAttemptStatus)
type StudentQuizAttemptStatus string

const (
	StudentQuizAttemptInProgress StudentQuizAttemptStatus = "in_progress"
	StudentQuizAttemptSubmitted  StudentQuizAttemptStatus = "submitted"
	StudentQuizAttemptFinished   StudentQuizAttemptStatus = "finished"
	StudentQuizAttemptAbandoned  StudentQuizAttemptStatus = "abandoned"
)

type StudentQuizAttemptModel struct {
	// PK teknis
	StudentQuizAttemptID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_quiz_attempt_id" json:"student_quiz_attempt_id"`

	// Tenant & identitas
	StudentQuizAttemptSchoolID  uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_school_id" json:"student_quiz_attempt_school_id"`
	StudentQuizAttemptQuizID    uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_quiz_id" json:"student_quiz_attempt_quiz_id"`
	StudentQuizAttemptStudentID uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_student_id" json:"student_quiz_attempt_student_id"`

	// Status attempt saat ini (dipakai di List + filter active_only)
	// Sesuaikan "type:" dengan tipe enum di DB kamu
	StudentQuizAttemptStatus StudentQuizAttemptStatus `gorm:"type:student_quiz_attempt_status_enum;not null;default:'in_progress';column:student_quiz_attempt_status" json:"student_quiz_attempt_status"`

	// Waktu attempt terakhir dimulai & selesai (dipakai untuk sorting)
	StudentQuizAttemptStartedAt  *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_started_at" json:"student_quiz_attempt_started_at,omitempty"`
	StudentQuizAttemptFinishedAt *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_finished_at" json:"student_quiz_attempt_finished_at,omitempty"`

	// Riwayat attempt lengkap (termasuk jawaban) dalam JSONB
	StudentQuizAttemptHistory datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:student_quiz_attempt_history" json:"student_quiz_attempt_history"`

	// Total attempt yang pernah dilakukan
	StudentQuizAttemptCount int `gorm:"type:int;not null;default:0;column:student_quiz_attempt_count" json:"student_quiz_attempt_count"`

	// ====== NILAI TERBAIK ======
	StudentQuizAttemptBestRaw        *float64   `gorm:"type:numeric(7,3);column:student_quiz_attempt_best_raw" json:"student_quiz_attempt_best_raw,omitempty"`
	StudentQuizAttemptBestPercent    *float64   `gorm:"type:numeric(6,3);column:student_quiz_attempt_best_percent" json:"student_quiz_attempt_best_percent,omitempty"`
	StudentQuizAttemptBestStartedAt  *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_best_started_at" json:"student_quiz_attempt_best_started_at,omitempty"`
	StudentQuizAttemptBestFinishedAt *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_best_finished_at" json:"student_quiz_attempt_best_finished_at,omitempty"`

	// ====== NILAI TERAKHIR ======
	StudentQuizAttemptLastRaw        *float64   `gorm:"type:numeric(7,3);column:student_quiz_attempt_last_raw" json:"student_quiz_attempt_last_raw,omitempty"`
	StudentQuizAttemptLastPercent    *float64   `gorm:"type:numeric(6,3);column:student_quiz_attempt_last_percent" json:"student_quiz_attempt_last_percent,omitempty"`
	StudentQuizAttemptLastStartedAt  *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_last_started_at" json:"student_quiz_attempt_last_started_at,omitempty"`
	StudentQuizAttemptLastFinishedAt *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_last_finished_at" json:"student_quiz_attempt_last_finished_at,omitempty"`

	// Timestamps
	StudentQuizAttemptCreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:student_quiz_attempt_created_at" json:"student_quiz_attempt_created_at"`
	StudentQuizAttemptUpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:student_quiz_attempt_updated_at" json:"student_quiz_attempt_updated_at"`
}

func (StudentQuizAttemptModel) TableName() string {
	return "student_quiz_attempts"
}
