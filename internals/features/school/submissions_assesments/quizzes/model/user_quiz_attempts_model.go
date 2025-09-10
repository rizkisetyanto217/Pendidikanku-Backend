// file: internals/features/school/submissions_assesments/quizzes/model/user_quiz_attempt_model.go
package model

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

/* =============================================================================
   ENUM-like: Attempts Status ('in_progress','submitted','finished','abandoned')
============================================================================= */
type UserQuizAttemptStatus string

const (
	UserAttemptInProgress UserQuizAttemptStatus = "in_progress"
	UserAttemptSubmitted  UserQuizAttemptStatus = "submitted"
	UserAttemptFinished   UserQuizAttemptStatus = "finished"
	UserAttemptAbandoned  UserQuizAttemptStatus = "abandoned"
)

func (s UserQuizAttemptStatus) String() string { return string(s) }
func (s UserQuizAttemptStatus) Valid() bool {
	switch s {
	case UserAttemptInProgress, UserAttemptSubmitted, UserAttemptFinished, UserAttemptAbandoned:
		return true
	default:
		return false
	}
}

// sql.Scanner + driver.Valuer (aman saat scan ke enum)
func (s *UserQuizAttemptStatus) Scan(value any) error {
	if value == nil {
		*s = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = UserQuizAttemptStatus(v)
	case []byte:
		*s = UserQuizAttemptStatus(string(v))
	default:
		return fmt.Errorf("unsupported type for UserQuizAttemptStatus: %T", value)
	}
	if !s.Valid() {
		return fmt.Errorf("invalid UserQuizAttemptStatus: %q", *s)
	}
	return nil
}
func (s UserQuizAttemptStatus) Value() (driver.Value, error) {
	if s == "" {
		return nil, nil
	}
	if !s.Valid() {
		return nil, fmt.Errorf("invalid UserQuizAttemptStatus: %q", s)
	}
	return string(s), nil
}

/* =============================================================================
   MODEL: user_quiz_attempts
   Catatan:
   - numeric(7,3)/(6,3) → float64 (ganti ke decimal bila perlu presisi penuh).
   - Index tags mengikuti DDL (untuk dokumentasi; DDL-mu sudah buat indeksnya).
============================================================================= */
type UserQuizAttemptModel struct {
	// PK
	UserQuizAttemptsID uuid.UUID `json:"user_quiz_attempts_id" gorm:"column:user_quiz_attempts_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// Tenant
	UserQuizAttemptsMasjidID uuid.UUID `json:"user_quiz_attempts_masjid_id" gorm:"column:user_quiz_attempts_masjid_id;type:uuid;not null;index:idx_uqa_masjid_quiz,priority:1"`

	// FK
	UserQuizAttemptsQuizID    uuid.UUID `json:"user_quiz_attempts_quiz_id" gorm:"column:user_quiz_attempts_quiz_id;type:uuid;not null;index:idx_uqa_quiz_student,priority:1;index:idx_uqa_quiz_student_started_desc,priority:1;index:idx_uqa_quiz_active,where:user_quiz_attempts_status IN ('in_progress','submitted');index:idx_uqa_masjid_quiz,priority:2"`
	UserQuizAttemptsStudentID uuid.UUID `json:"user_quiz_attempts_student_id" gorm:"column:user_quiz_attempts_student_id;type:uuid;not null;index:idx_uqa_quiz_student,priority:2;index:idx_uqa_quiz_student_started_desc,priority:2;index:idx_uqa_student;index:idx_uqa_student_status,priority:1"`

	// Waktu
	UserQuizAttemptsStartedAt  time.Time  `json:"user_quiz_attempts_started_at" gorm:"column:user_quiz_attempts_started_at;type:timestamptz;not null;default:now();index:brin_uqa_started_at,priority:1,using:brin"`
	UserQuizAttemptsFinishedAt *time.Time `json:"user_quiz_attempts_finished_at,omitempty" gorm:"column:user_quiz_attempts_finished_at;type:timestamptz"`

	// Skor
	UserQuizAttemptsScoreRaw     *float64 `json:"user_quiz_attempts_score_raw,omitempty" gorm:"column:user_quiz_attempts_score_raw;type:numeric(7,3);default:0"`
	UserQuizAttemptsScorePercent *float64 `json:"user_quiz_attempts_score_percent,omitempty" gorm:"column:user_quiz_attempts_score_percent;type:numeric(6,3);default:0"`

	// Status
	UserQuizAttemptsStatus UserQuizAttemptStatus `json:"user_quiz_attempts_status" gorm:"column:user_quiz_attempts_status;type:varchar(16);not null;default:'in_progress';index:idx_uqa_status;index:idx_uqa_student_status,priority:2"`

	// Audit
	UserQuizAttemptsCreatedAt time.Time `json:"user_quiz_attempts_created_at" gorm:"column:user_quiz_attempts_created_at;type:timestamptz;not null;default:now();index:brin_uqa_created_at,using:brin"`
	UserQuizAttemptsUpdatedAt time.Time `json:"user_quiz_attempts_updated_at" gorm:"column:user_quiz_attempts_updated_at;type:timestamptz;not null;default:now()"`
}

// Nama tabel eksplisit
func (UserQuizAttemptModel) TableName() string { return "user_quiz_attempts" }

/* =============================================================================
   Hooks — jaga updated_at
============================================================================= */
func (m *UserQuizAttemptModel) BeforeSave(_ any) error {
	m.UserQuizAttemptsUpdatedAt = time.Now()
	return nil
}

/* ===================================================================
   Helper methods
=================================================================== */
func (m *UserQuizAttemptModel) IsActive() bool {
	return m.UserQuizAttemptsStatus == UserAttemptInProgress || m.UserQuizAttemptsStatus == UserAttemptSubmitted
}

func (m *UserQuizAttemptModel) MarkSubmitted(scoreRaw, scorePct *float64, finishedAt *time.Time) {
	m.UserQuizAttemptsStatus = UserAttemptSubmitted
	if finishedAt != nil {
		m.UserQuizAttemptsFinishedAt = finishedAt
	}
	if scoreRaw != nil {
		m.UserQuizAttemptsScoreRaw = scoreRaw
	}
	if scorePct != nil {
		m.UserQuizAttemptsScorePercent = scorePct
	}
}

func (m *UserQuizAttemptModel) MarkFinished(scoreRaw, scorePct *float64, finishedAt *time.Time) {
	m.UserQuizAttemptsStatus = UserAttemptFinished
	if finishedAt != nil {
		m.UserQuizAttemptsFinishedAt = finishedAt
	}
	if scoreRaw != nil {
		m.UserQuizAttemptsScoreRaw = scoreRaw
	}
	if scorePct != nil {
		m.UserQuizAttemptsScorePercent = scorePct
	}
}

func (m *UserQuizAttemptModel) MarkAbandoned(finishedAt *time.Time) {
	m.UserQuizAttemptsStatus = UserAttemptAbandoned
	if finishedAt != nil {
		m.UserQuizAttemptsFinishedAt = finishedAt
	}
}
