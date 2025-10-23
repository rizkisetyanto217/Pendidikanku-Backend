package model

import (
	"time"

	"github.com/google/uuid"
)

/* =========================================================
   StudentQuizAttemptAnswer (student_quiz_attempt_answers)
   ========================================================= */

type StudentQuizAttemptAnswerModel struct {
	// PK
	StudentQuizAttemptAnswerID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_quiz_attempt_answer_id" json:"student_quiz_attempt_answer_id"`

	// Diisi backend (bukan trigger)
	StudentQuizAttemptAnswerQuizID uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_answer_quiz_id;index:idx_sqaa_quiz" json:"student_quiz_attempt_answer_quiz_id"`

	// Relasi attempt & question
	StudentQuizAttemptAnswerAttemptID  uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_answer_attempt_id;index:idx_sqaa_attempt" json:"student_quiz_attempt_answer_attempt_id"`
	StudentQuizAttemptAnswerQuestionID uuid.UUID `gorm:"type:uuid;not null;column:student_quiz_attempt_answer_question_id;index:idx_sqaa_question" json:"student_quiz_attempt_answer_question_id"`

	// Unique: 1 attempt hanya 1 jawaban per soal
	_ struct{} `gorm:"uniqueIndex:uq_sqaa_attempt_question,priority:1"`

	// Jawaban student
	StudentQuizAttemptAnswerText string `gorm:"type:text;not null;column:student_quiz_attempt_answer_text" json:"student_quiz_attempt_answer_text"`

	// Hasil penilaian
	StudentQuizAttemptAnswerIsCorrect         *bool      `gorm:"column:student_quiz_attempt_answer_is_correct" json:"student_quiz_attempt_answer_is_correct,omitempty"`
	StudentQuizAttemptAnswerEarnedPoints      float64    `gorm:"type:numeric(6,2);not null;default:0;column:student_quiz_attempt_answer_earned_points" json:"student_quiz_attempt_answer_earned_points"`
	StudentQuizAttemptAnswerGradedByTeacherID *uuid.UUID `gorm:"type:uuid;column:student_quiz_attempt_answer_graded_by_teacher_id" json:"student_quiz_attempt_answer_graded_by_teacher_id,omitempty"`
	StudentQuizAttemptAnswerGradedAt          *time.Time `gorm:"type:timestamptz;column:student_quiz_attempt_answer_graded_at;index:idx_sqaa_need_grading,sort:asc" json:"student_quiz_attempt_answer_graded_at,omitempty"`
	StudentQuizAttemptAnswerFeedback          *string    `gorm:"type:text;column:student_quiz_attempt_answer_feedback" json:"student_quiz_attempt_answer_feedback,omitempty"`

	// Waktu menjawab
	StudentQuizAttemptAnswerAnsweredAt time.Time `gorm:"type:timestamptz;not null;default:now();column:student_quiz_attempt_answer_answered_at;index:brin_sqaa_answered_at,class:BRIN" json:"student_quiz_attempt_answer_answered_at"`

	/* =====================================================
	   Composite FK (tenant-safe consistency by quiz_id)
	   ===================================================== */

	// Attempt & Quiz harus match (FK komposit → student_quiz_attempts)
	// FOREIGN KEY (attempt_id, quiz_id) REFERENCES student_quiz_attempts(id, quiz_id) ON DELETE CASCADE
	Attempt *StudentQuizAttemptModel `gorm:"foreignKey:StudentQuizAttemptAnswerAttemptID,StudentQuizAttemptAnswerQuizID;references:StudentQuizAttemptID,StudentQuizAttemptQuizID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"attempt,omitempty"`

	// Question & Quiz harus match (FK komposit → quiz_questions)
	// FOREIGN KEY (question_id, quiz_id) REFERENCES quiz_questions(id, quiz_id) ON DELETE CASCADE
	Question *QuizQuestionModel `gorm:"foreignKey:StudentQuizAttemptAnswerQuestionID,StudentQuizAttemptAnswerQuizID;references:QuizQuestionID,QuizQuestionQuizID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"question,omitempty"`
}

func (StudentQuizAttemptAnswerModel) TableName() string { return "student_quiz_attempt_answers" }
