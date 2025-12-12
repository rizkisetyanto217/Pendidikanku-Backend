// file: internals/features/school/submissions_assesments/quizzes/model/student_quiz_attempt_model.go
package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/*
=========================================================

	STUDENT QUIZ ATTEMPTS (JSON VERSION)
	1 row = 1 student × 1 quiz
	- history      : semua attempt dalam JSONB
	- count        : total attempt
	- best_*       : nilai terbaik
	- last_*       : nilai attempt terakhir
	- first_*      : nilai attempt pertama
	- avg_*        : rata-rata nilai semua attempt

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

/* =========================================================
   HISTORY STRUCTS
========================================================= */

// Satu soal yang dijawab dalam satu attempt
type StudentQuizAttemptQuestionItem struct {
	QuizID              uuid.UUID        `json:"quiz_id"`
	QuizQuestionID      uuid.UUID        `json:"quiz_question_id"`
	QuizQuestionVersion int              `json:"quiz_question_version"`
	QuizQuestionType    QuizQuestionType `json:"quiz_question_type"`

	// Jawaban murid
	AnswerSingle *string `json:"answer_single,omitempty"` // untuk single choice (A/B/C/...)
	AnswerEssay  *string `json:"answer_essay,omitempty"`  // untuk essay (teks bebas)

	// Penilaian
	IsCorrect    *bool   `json:"is_correct,omitempty"` // boleh null (misal essay belum dinilai)
	Points       float64 `json:"points"`               // bobot soal
	PointsEarned float64 `json:"points_earned"`        // 0 / points / parsial
}

// Satu attempt lengkap (1 kali pengerjaan quiz)
type StudentQuizAttemptHistoryItem struct {
	AttemptNo         int       `json:"attempt_no"`
	AttemptStartedAt  time.Time `json:"attempt_started_at"`
	AttemptFinishedAt time.Time `json:"attempt_finished_at"`

	AttemptRawScore float64 `json:"attempt_raw_score"` // total PointsEarned
	AttemptPercent  float64 `json:"attempt_percent"`   // 0–100

	Items []StudentQuizAttemptQuestionItem `json:"items"`
}

/*
=========================================================

	MODEL

=========================================================
*/
type StudentQuizAttemptModel struct {
	// PK teknis
	StudentQuizAttemptID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_quiz_attempt_id" json:"student_quiz_attempt_id"`

	// Tenant & identitas
	StudentQuizAttemptSchoolID  uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_school_id" json:"student_quiz_attempt_school_id"`
	StudentQuizAttemptQuizID    uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_quiz_id" json:"student_quiz_attempt_quiz_id"`
	StudentQuizAttemptStudentID uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_student_id" json:"student_quiz_attempt_student_id"`

	// Cache user profile & siswa (snapshot, optional)
	StudentQuizAttemptUserProfileNameSnapshot        *string `gorm:"type:varchar(80);column:student_quiz_attempt_user_profile_name_snapshot" json:"student_quiz_attempt_user_profile_name_snapshot,omitempty"`
	StudentQuizAttemptUserProfileAvatarURLSnapshot   *string `gorm:"type:varchar(255);column:student_quiz_attempt_user_profile_avatar_url_snapshot" json:"student_quiz_attempt_user_profile_avatar_url_snapshot,omitempty"`
	StudentQuizAttemptUserProfileWhatsappURLSnapshot *string `gorm:"type:varchar(50);column:student_quiz_attempt_user_profile_whatsapp_url_snapshot" json:"student_quiz_attempt_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentQuizAttemptUserProfileGenderSnapshot      *string `gorm:"type:varchar(20);column:student_quiz_attempt_user_profile_gender_snapshot" json:"student_quiz_attempt_user_profile_gender_snapshot,omitempty"`
	StudentQuizAttemptSchoolStudentCodeCache         *string `gorm:"type:varchar(50);column:student_quiz_attempt_school_student_code_cache" json:"student_quiz_attempt_school_student_code_cache,omitempty"`

	// Status attempt saat ini (dipakai di List + filter active_only)
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

	// ====== NILAI PERTAMA (FIRST) ======
	StudentQuizAttemptFirstRaw        *float64   `gorm:"type:numeric(7,3);column:student_quiz_attempt_first_raw" json:"student_quiz_attempt_first_raw,omitempty"`
	StudentQuizAttemptFirstPercent    *float64   `gorm:"type:numeric(6,3);column:student_quiz_attempt_first_percent" json:"student_quiz_attempt_first_percent,omitempty"`
	StudentQuizAttemptFirstStartedAt  *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_first_started_at" json:"student_quiz_attempt_first_started_at,omitempty"`
	StudentQuizAttemptFirstFinishedAt *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_first_finished_at" json:"student_quiz_attempt_first_finished_at,omitempty"`

	// ====== NILAI RATA-RATA (AVERAGE) ======
	StudentQuizAttemptAvgRaw     *float64 `gorm:"type:numeric(7,3);column:student_quiz_attempt_avg_raw" json:"student_quiz_attempt_avg_raw,omitempty"`
	StudentQuizAttemptAvgPercent *float64 `gorm:"type:numeric(6,3);column:student_quiz_attempt_avg_percent" json:"student_quiz_attempt_avg_percent,omitempty"`

	// Timestamps
	StudentQuizAttemptCreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:student_quiz_attempt_created_at" json:"student_quiz_attempt_created_at"`
	StudentQuizAttemptUpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:student_quiz_attempt_updated_at" json:"student_quiz_attempt_updated_at"`
}

/* =========================================================
   HISTORY HELPER
========================================================= */

// AppendAttemptHistory dipanggil dari service submit,
// bukan dari controller langsung.
func (m *StudentQuizAttemptModel) AppendAttemptHistory(
	startedAt, finishedAt time.Time,
	items []StudentQuizAttemptQuestionItem,
) error {
	// 1) Parse history lama
	var history []StudentQuizAttemptHistoryItem
	if len(m.StudentQuizAttemptHistory) > 0 {
		if err := json.Unmarshal(m.StudentQuizAttemptHistory, &history); err != nil {
			return fmt.Errorf("invalid student_quiz_attempt_history json: %w", err)
		}
	}

	// 2) Hitung skor total attempt baru
	var totalPoints, totalEarned float64
	for _, it := range items {
		totalPoints += it.Points
		totalEarned += it.PointsEarned
	}

	percent := 0.0
	if totalPoints > 0 {
		percent = (totalEarned / totalPoints) * 100.0
	}

	// 3) Buat attempt baru
	attempt := StudentQuizAttemptHistoryItem{
		AttemptNo:         len(history) + 1,
		AttemptStartedAt:  startedAt,
		AttemptFinishedAt: finishedAt,
		AttemptRawScore:   totalEarned,
		AttemptPercent:    percent,
		Items:             items,
	}

	history = append(history, attempt)

	// 4) Serialize kembali ke JSONB
	buf, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal student_quiz_attempt_history: %w", err)
	}

	m.StudentQuizAttemptHistory = datatypes.JSON(buf)
	m.StudentQuizAttemptCount = len(history)

	// 5) Update summary LAST
	m.StudentQuizAttemptLastRaw = &attempt.AttemptRawScore
	m.StudentQuizAttemptLastPercent = &attempt.AttemptPercent
	m.StudentQuizAttemptLastStartedAt = &attempt.AttemptStartedAt
	m.StudentQuizAttemptLastFinishedAt = &attempt.AttemptFinishedAt

	// 6) Update summary BEST (kalau lebih bagus)
	if m.StudentQuizAttemptBestPercent == nil || attempt.AttemptPercent > *m.StudentQuizAttemptBestPercent {
		m.StudentQuizAttemptBestRaw = &attempt.AttemptRawScore
		m.StudentQuizAttemptBestPercent = &attempt.AttemptPercent
		m.StudentQuizAttemptBestStartedAt = &attempt.AttemptStartedAt
		m.StudentQuizAttemptBestFinishedAt = &attempt.AttemptFinishedAt
	}

	// 7) FIRST: kalau belum pernah di-set, atau nil (row lama sebelum kolom ada)
	if m.StudentQuizAttemptFirstRaw == nil || m.StudentQuizAttemptFirstPercent == nil ||
		m.StudentQuizAttemptFirstStartedAt == nil || m.StudentQuizAttemptFirstFinishedAt == nil {

		if len(history) > 0 {
			first := history[0]
			m.StudentQuizAttemptFirstRaw = &first.AttemptRawScore
			m.StudentQuizAttemptFirstPercent = &first.AttemptPercent
			m.StudentQuizAttemptFirstStartedAt = &first.AttemptStartedAt
			m.StudentQuizAttemptFirstFinishedAt = &first.AttemptFinishedAt
		}
	}

	// 8) AVERAGE: hitung rata-rata dari seluruh history
	var sumRaw, sumPercent float64
	for _, h := range history {
		sumRaw += h.AttemptRawScore
		sumPercent += h.AttemptPercent
	}
	n := float64(len(history))
	if n > 0 {
		avgRaw := sumRaw / n
		avgPercent := sumPercent / n
		m.StudentQuizAttemptAvgRaw = &avgRaw
		m.StudentQuizAttemptAvgPercent = &avgPercent
	}

	return nil
}

// TableName override default GORM → pakai nama tabel nyata di DB
func (StudentQuizAttemptModel) TableName() string {
	return "student_quiz_attempts"
}
