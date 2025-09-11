// file: internals/features/quiz/user_attempts/dto/user_quiz_attempt_answer_dto.go
package dto

import (
	"time"

	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

// Create — wajib: attempt_id, question_id, dan text (SINGLE: label/A-D; ESSAY: uraian)
type CreateUserQuizAttemptAnswerRequest struct {
	UserQuizAttemptAnswersAttemptID  uuid.UUID `json:"user_quiz_attempt_answers_attempt_id" validate:"required"`
	UserQuizAttemptAnswersQuestionID uuid.UUID `json:"user_quiz_attempt_answers_question_id" validate:"required"`

	// Jawaban user — wajib, akan ditrim/validasi lagi di controller bila perlu
	UserQuizAttemptAnswersText string `json:"user_quiz_attempt_answers_text" validate:"required"`

	// Opsi penilaian (biasanya backend yang isi)
	UserQuizAttemptAnswersIsCorrect    *bool      `json:"user_quiz_attempt_answers_is_correct" validate:"omitempty"`
	UserQuizAttemptAnswersEarnedPoints *float64   `json:"user_quiz_attempt_answers_earned_points" validate:"omitempty,gte=0,lte=9999.99"`
	UserQuizAttemptAnswersGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answers_graded_by_teacher_id" validate:"omitempty"`
	UserQuizAttemptAnswersGradedAt          *time.Time `json:"user_quiz_attempt_answers_graded_at" validate:"omitempty"`
	UserQuizAttemptAnswersFeedback          *string    `json:"user_quiz_attempt_answers_feedback" validate:"omitempty"`

	// Optional import historis
	UserQuizAttemptAnswersAnsweredAt *time.Time `json:"user_quiz_attempt_answers_answered_at" validate:"omitempty"`
}

// Patch/Update — semua opsional; attempt_id/question_id tidak boleh diganti.
type UpdateUserQuizAttemptAnswerRequest struct {
	// Jawaban baru (opsional). Controller boleh trim & cek non-empty bila dikirim.
	UserQuizAttemptAnswersText *string `json:"user_quiz_attempt_answers_text" validate:"omitempty"`

	UserQuizAttemptAnswersIsCorrect    *bool      `json:"user_quiz_attempt_answers_is_correct" validate:"omitempty"`
	UserQuizAttemptAnswersEarnedPoints *float64   `json:"user_quiz_attempt_answers_earned_points" validate:"omitempty,gte=0,lte=9999.99"`
	UserQuizAttemptAnswersGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answers_graded_by_teacher_id" validate:"omitempty"`
	UserQuizAttemptAnswersGradedAt          *time.Time `json:"user_quiz_attempt_answers_graded_at" validate:"omitempty"`
	UserQuizAttemptAnswersFeedback          *string    `json:"user_quiz_attempt_answers_feedback" validate:"omitempty"`

	UserQuizAttemptAnswersAnsweredAt *time.Time `json:"user_quiz_attempt_answers_answered_at" validate:"omitempty"`
}

/* ===================== RESPONSES ===================== */

type UserQuizAttemptAnswerResponse struct {
	UserQuizAttemptAnswersID         uuid.UUID `json:"user_quiz_attempt_answers_id"`
	UserQuizAttemptAnswersQuizID     uuid.UUID `json:"user_quiz_attempt_answers_quiz_id"`
	UserQuizAttemptAnswersAttemptID  uuid.UUID `json:"user_quiz_attempt_answers_attempt_id"`
	UserQuizAttemptAnswersQuestionID uuid.UUID `json:"user_quiz_attempt_answers_question_id"`

	UserQuizAttemptAnswersText       string     `json:"user_quiz_attempt_answers_text"`
	UserQuizAttemptAnswersIsCorrect  *bool      `json:"user_quiz_attempt_answers_is_correct,omitempty"`
	UserQuizAttemptAnswersEarnedPoints float64  `json:"user_quiz_attempt_answers_earned_points"`

	UserQuizAttemptAnswersGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answers_graded_by_teacher_id,omitempty"`
	UserQuizAttemptAnswersGradedAt          *time.Time `json:"user_quiz_attempt_answers_graded_at,omitempty"`
	UserQuizAttemptAnswersFeedback          *string    `json:"user_quiz_attempt_answers_feedback,omitempty"`

	UserQuizAttemptAnswersAnsweredAt time.Time `json:"user_quiz_attempt_answers_answered_at"`
}

/* ===================== CONVERTERS ===================== */

func ToUserQuizAttemptAnswerResponse(m *model.UserQuizAttemptAnswerModel) *UserQuizAttemptAnswerResponse {
	if m == nil {
		return nil
	}
	var qid uuid.UUID
	if m.UserQuizAttemptAnswersQuizID != nil {
		qid = *m.UserQuizAttemptAnswersQuizID
	}
	return &UserQuizAttemptAnswerResponse{
		UserQuizAttemptAnswersID:          m.UserQuizAttemptAnswersID,
		UserQuizAttemptAnswersQuizID:      qid,
		UserQuizAttemptAnswersAttemptID:   m.UserQuizAttemptAnswersAttemptID,
		UserQuizAttemptAnswersQuestionID:  m.UserQuizAttemptAnswersQuestionID,
		UserQuizAttemptAnswersText:        m.UserQuizAttemptAnswersText,
		UserQuizAttemptAnswersIsCorrect:   m.UserQuizAttemptAnswersIsCorrect,
		UserQuizAttemptAnswersEarnedPoints: m.UserQuizAttemptAnswersEarnedPoints,
		UserQuizAttemptAnswersGradedByTeacherID: m.UserQuizAttemptAnswersGradedByTeacherID,
		UserQuizAttemptAnswersGradedAt:    m.UserQuizAttemptAnswersGradedAt,
		UserQuizAttemptAnswersFeedback:    m.UserQuizAttemptAnswersFeedback,
		UserQuizAttemptAnswersAnsweredAt:  m.UserQuizAttemptAnswersAnsweredAt,
	}
}

func (r *CreateUserQuizAttemptAnswerRequest) ToModel() *model.UserQuizAttemptAnswerModel {
	m := &model.UserQuizAttemptAnswerModel{
		UserQuizAttemptAnswersAttemptID: r.UserQuizAttemptAnswersAttemptID,
		UserQuizAttemptAnswersQuestionID: r.UserQuizAttemptAnswersQuestionID,
		// QuizID dibiarkan nil: akan diisi trigger dari attempt_id
		UserQuizAttemptAnswersText: r.UserQuizAttemptAnswersText,
	}
	if r.UserQuizAttemptAnswersIsCorrect != nil {
		m.UserQuizAttemptAnswersIsCorrect = r.UserQuizAttemptAnswersIsCorrect
	}
	if r.UserQuizAttemptAnswersEarnedPoints != nil {
		m.UserQuizAttemptAnswersEarnedPoints = *r.UserQuizAttemptAnswersEarnedPoints
	}
	if r.UserQuizAttemptAnswersGradedByTeacherID != nil {
		m.UserQuizAttemptAnswersGradedByTeacherID = r.UserQuizAttemptAnswersGradedByTeacherID
	}
	if r.UserQuizAttemptAnswersGradedAt != nil {
		m.UserQuizAttemptAnswersGradedAt = r.UserQuizAttemptAnswersGradedAt
	}
	if r.UserQuizAttemptAnswersFeedback != nil {
		m.UserQuizAttemptAnswersFeedback = r.UserQuizAttemptAnswersFeedback
	}
	if r.UserQuizAttemptAnswersAnsweredAt != nil {
		m.UserQuizAttemptAnswersAnsweredAt = *r.UserQuizAttemptAnswersAnsweredAt
	}
	return m
}

// Apply ke model untuk PATCH (partial)
func (r *UpdateUserQuizAttemptAnswerRequest) Apply(m *model.UserQuizAttemptAnswerModel) {
	if r.UserQuizAttemptAnswersText != nil {
		m.UserQuizAttemptAnswersText = *r.UserQuizAttemptAnswersText
	}
	if r.UserQuizAttemptAnswersIsCorrect != nil {
		m.UserQuizAttemptAnswersIsCorrect = r.UserQuizAttemptAnswersIsCorrect
	}
	if r.UserQuizAttemptAnswersEarnedPoints != nil {
		m.UserQuizAttemptAnswersEarnedPoints = *r.UserQuizAttemptAnswersEarnedPoints
	}
	if r.UserQuizAttemptAnswersGradedByTeacherID != nil {
		m.UserQuizAttemptAnswersGradedByTeacherID = r.UserQuizAttemptAnswersGradedByTeacherID
	}
	if r.UserQuizAttemptAnswersGradedAt != nil {
		m.UserQuizAttemptAnswersGradedAt = r.UserQuizAttemptAnswersGradedAt
	}
	if r.UserQuizAttemptAnswersFeedback != nil {
		m.UserQuizAttemptAnswersFeedback = r.UserQuizAttemptAnswersFeedback
	}
	if r.UserQuizAttemptAnswersAnsweredAt != nil {
		m.UserQuizAttemptAnswersAnsweredAt = *r.UserQuizAttemptAnswersAnsweredAt
	}
}
