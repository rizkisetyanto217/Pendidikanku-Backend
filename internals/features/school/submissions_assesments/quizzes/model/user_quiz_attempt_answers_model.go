// file: internals/features/quiz/user_attempts/model/user_quiz_attempt_answer_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// UserQuizAttemptAnswerModel merepresentasikan baris pada tabel user_quiz_attempt_answers
// Versi ini mengikuti DDL terbaru (no selected_option_id, strong FK pakai quiz_id, text wajib).
type UserQuizAttemptAnswerModel struct {
	// PK
	UserQuizAttemptAnswersID uuid.UUID `json:"user_quiz_attempt_answers_id" gorm:"column:user_quiz_attempt_answers_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// Diisi otomatis via trigger dari attempt_id (harus pointer agar INSERT mengirim NULL -> trigger jalan)
	UserQuizAttemptAnswersQuizID *uuid.UUID `json:"user_quiz_attempt_answers_quiz_id" gorm:"column:user_quiz_attempt_answers_quiz_id;type:uuid;not null"`

	// FK -> user_quiz_attempts(user_quiz_attempts_id)
	UserQuizAttemptAnswersAttemptID uuid.UUID `json:"user_quiz_attempt_answers_attempt_id" gorm:"column:user_quiz_attempt_answers_attempt_id;type:uuid;not null;index;uniqueIndex:uqa_attempt_question"`

	// FK logis -> quiz_questions(quiz_questions_id)
	UserQuizAttemptAnswersQuestionID uuid.UUID `json:"user_quiz_attempt_answers_question_id" gorm:"column:user_quiz_attempt_answers_question_id;type:uuid;not null;uniqueIndex:uqa_attempt_question"`

	// Jawaban user (SINGLE: label/teks opsi / 'A'..'D'; ESSAY: uraian) — NOT NULL di DB
	UserQuizAttemptAnswersText string `json:"user_quiz_attempt_answers_text" gorm:"column:user_quiz_attempt_answers_text;type:text;not null"`

	// Hasil penilaian (SINGLE auto; ESSAY manual). Boleh NULL jika belum dinilai.
	UserQuizAttemptAnswersIsCorrect   *bool    `json:"user_quiz_attempt_answers_is_correct" gorm:"column:user_quiz_attempt_answers_is_correct"`
	UserQuizAttemptAnswersEarnedPoints float64 `json:"user_quiz_attempt_answers_earned_points" gorm:"column:user_quiz_attempt_answers_earned_points;type:numeric(6,2);not null;default:0"`

	// Penilaian manual (ESSAY)
	UserQuizAttemptAnswersGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answers_graded_by_teacher_id" gorm:"column:user_quiz_attempt_answers_graded_by_teacher_id;type:uuid"`
	UserQuizAttemptAnswersGradedAt          *time.Time `json:"user_quiz_attempt_answers_graded_at" gorm:"column:user_quiz_attempt_answers_graded_at"`
	UserQuizAttemptAnswersFeedback          *string    `json:"user_quiz_attempt_answers_feedback" gorm:"column:user_quiz_attempt_answers_feedback;type:text"`

	// Time-series
	UserQuizAttemptAnswersAnsweredAt time.Time `json:"user_quiz_attempt_answers_answered_at" gorm:"column:user_quiz_attempt_answers_answered_at;type:timestamptz;not null;default:now()"`
}

// TableName memastikan nama tabel sesuai DDL.
func (UserQuizAttemptAnswerModel) TableName() string {
	return "user_quiz_attempt_answers"
}
