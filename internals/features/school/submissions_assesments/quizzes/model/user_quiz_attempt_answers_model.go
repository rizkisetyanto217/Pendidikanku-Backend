package model

import (
	"time"

	"github.com/google/uuid"
)

/* =========================================================
   UserQuizAttemptAnswer (user_quiz_attempt_answers)
   ========================================================= */

type UserQuizAttemptAnswerModel struct {
	// PK
	UserQuizAttemptAnswerID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_quiz_attempt_answer_id" json:"user_quiz_attempt_answer_id"`

	// Diisi backend (bukan trigger)
	UserQuizAttemptAnswerQuizID uuid.UUID `gorm:"type:uuid;not null;column:user_quiz_attempt_answer_quiz_id;index:idx_uqaa_quiz" json:"user_quiz_attempt_answer_quiz_id"`

	// Relasi attempt & question
	UserQuizAttemptAnswerAttemptID  uuid.UUID `gorm:"type:uuid;not null;column:user_quiz_attempt_answer_attempt_id;index:idx_uqaa_attempt" json:"user_quiz_attempt_answer_attempt_id"`
	UserQuizAttemptAnswerQuestionID uuid.UUID `gorm:"type:uuid;not null;column:user_quiz_attempt_answer_question_id;index:idx_uqaa_question" json:"user_quiz_attempt_answer_question_id"`

	// Unique: 1 attempt hanya 1 jawaban per soal
	_ struct{} `gorm:"uniqueIndex:uq_uqaa_attempt_question,priority:1"`

	// Jawaban user
	UserQuizAttemptAnswerText string `gorm:"type:text;not null;column:user_quiz_attempt_answer_text" json:"user_quiz_attempt_answer_text"`

	// Hasil penilaian
	UserQuizAttemptAnswerIsCorrect    *bool    `gorm:"column:user_quiz_attempt_answer_is_correct" json:"user_quiz_attempt_answer_is_correct,omitempty"`
	UserQuizAttemptAnswerEarnedPoints float64  `gorm:"type:numeric(6,2);not null;default:0;column:user_quiz_attempt_answer_earned_points" json:"user_quiz_attempt_answer_earned_points"`
	UserQuizAttemptAnswerGradedByTeacherID *uuid.UUID `gorm:"type:uuid;column:user_quiz_attempt_answer_graded_by_teacher_id" json:"user_quiz_attempt_answer_graded_by_teacher_id,omitempty"`
	UserQuizAttemptAnswerGradedAt          *time.Time `gorm:"type:timestamptz;column:user_quiz_attempt_answer_graded_at;index:idx_uqaa_need_grading,sort:asc" json:"user_quiz_attempt_answer_graded_at,omitempty"`
	UserQuizAttemptAnswerFeedback          *string    `gorm:"type:text;column:user_quiz_attempt_answer_feedback" json:"user_quiz_attempt_answer_feedback,omitempty"`

	// Waktu menjawab
	UserQuizAttemptAnswerAnsweredAt time.Time `gorm:"type:timestamptz;not null;default:now();column:user_quiz_attempt_answer_answered_at;index:brin_uqaa_answered_at,class:BRIN" json:"user_quiz_attempt_answer_answered_at"`

	/* =====================================================
	   Composite FK (tenant-safe consistency by quiz_id)
	   ===================================================== */

	// Attempt & Quiz harus match (FK komposit → user_quiz_attempts)
	// FOREIGN KEY (attempt_id, quiz_id) REFERENCES user_quiz_attempts(id, quiz_id) ON DELETE CASCADE
	Attempt *UserQuizAttemptModel `gorm:"foreignKey:UserQuizAttemptAnswerAttemptID,UserQuizAttemptAnswerQuizID;references:UserQuizAttemptID,UserQuizAttemptQuizID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"attempt,omitempty"`

	// Question & Quiz harus match (FK komposit → quiz_questions)
	// FOREIGN KEY (question_id, quiz_id) REFERENCES quiz_questions(id, quiz_id) ON DELETE CASCADE
	Question *QuizQuestionModel `gorm:"foreignKey:UserQuizAttemptAnswerQuestionID,UserQuizAttemptAnswerQuizID;references:QuizQuestionID,QuizQuestionQuizID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"question,omitempty"`
}

func (UserQuizAttemptAnswerModel) TableName() string { return "user_quiz_attempt_answers" }