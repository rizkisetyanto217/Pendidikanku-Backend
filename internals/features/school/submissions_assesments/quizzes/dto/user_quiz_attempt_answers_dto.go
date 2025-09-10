// file: internals/features/quiz/user_attempts/dto/user_quiz_attempt_answer_dto.go
package dto

import (
	"time"

	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

// Create — wajib attempt_id & question_id, dan persis salah satu: selected_option_id ATAU text
type CreateUserQuizAttemptAnswerRequest struct {
	UserQuizAttemptAnswersAttemptID  uuid.UUID  `json:"user_quiz_attempt_answers_attempt_id" validate:"required"`
	UserQuizAttemptAnswersQuestionID uuid.UUID  `json:"user_quiz_attempt_answers_question_id" validate:"required"`

	// XOR rules:
	// - selected_option_id required_without=text, dan excluded_with=text
	// - text required_without=selected_option_id, dan excluded_with=selected_option_id
	UserQuizAttemptAnswersSelectedOptionID *uuid.UUID `json:"user_quiz_attempt_answers_selected_option_id" validate:"omitempty,required_without=UserQuizAttemptAnswersText,excluded_with=UserQuizAttemptAnswersText"`
	UserQuizAttemptAnswersText             *string    `json:"user_quiz_attempt_answers_text" validate:"omitempty,required_without=UserQuizAttemptAnswersSelectedOptionID,excluded_with=UserQuizAttemptAnswersSelectedOptionID"`

	// Optional (biasanya diisi backend saat auto grade / manual grade)
	UserQuizAttemptAnswersIsCorrect   *bool      `json:"user_quiz_attempt_answers_is_correct" validate:"omitempty"`
	UserQuizAttemptAnswersEarnedPoints *float64  `json:"user_quiz_attempt_answers_earned_points" validate:"omitempty,gte=0,lte=9999.99"`
	UserQuizAttemptAnswersGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answers_graded_by_teacher_id" validate:"omitempty"`
	UserQuizAttemptAnswersGradedAt          *time.Time `json:"user_quiz_attempt_answers_graded_at" validate:"omitempty"`
	UserQuizAttemptAnswersFeedback          *string    `json:"user_quiz_attempt_answers_feedback" validate:"omitempty"`

	// answered_at biarkan default dari DB; boleh diisi jika perlu import data historis
	UserQuizAttemptAnswersAnsweredAt *time.Time `json:"user_quiz_attempt_answers_answered_at" validate:"omitempty"`
}

// Patch/Update — semua opsional; tetap jaga XOR bila salah satu diubah.
// Catatan: jika keduanya kosong (tidak diubah), validator tidak memaksa.
type UpdateUserQuizAttemptAnswerRequest struct {
	// Tidak mengizinkan ganti attempt_id / question_id demi konsistensi UNIQUE(attempt,question)
	// Jika sangat perlu, lakukan via delete+create.

	UserQuizAttemptAnswersSelectedOptionID *uuid.UUID `json:"user_quiz_attempt_answers_selected_option_id" validate:"omitempty,excluded_with=UserQuizAttemptAnswersText"`
	UserQuizAttemptAnswersText             *string    `json:"user_quiz_attempt_answers_text" validate:"omitempty,excluded_with=UserQuizAttemptAnswersSelectedOptionID"`

	UserQuizAttemptAnswersIsCorrect   *bool      `json:"user_quiz_attempt_answers_is_correct" validate:"omitempty"`
	UserQuizAttemptAnswersEarnedPoints *float64  `json:"user_quiz_attempt_answers_earned_points" validate:"omitempty,gte=0,lte=9999.99"`
	UserQuizAttemptAnswersGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answers_graded_by_teacher_id" validate:"omitempty"`
	UserQuizAttemptAnswersGradedAt          *time.Time `json:"user_quiz_attempt_answers_graded_at" validate:"omitempty"`
	UserQuizAttemptAnswersFeedback          *string    `json:"user_quiz_attempt_answers_feedback" validate:"omitempty"`

	UserQuizAttemptAnswersAnsweredAt *time.Time `json:"user_quiz_attempt_answers_answered_at" validate:"omitempty"`
}

/* ===================== RESPONSES ===================== */

type UserQuizAttemptAnswerResponse struct {
	UserQuizAttemptAnswersID              uuid.UUID  `json:"user_quiz_attempt_answers_id"`
	UserQuizAttemptAnswersAttemptID       uuid.UUID  `json:"user_quiz_attempt_answers_attempt_id"`
	UserQuizAttemptAnswersQuestionID      uuid.UUID  `json:"user_quiz_attempt_answers_question_id"`
	UserQuizAttemptAnswersSelectedOptionID *uuid.UUID `json:"user_quiz_attempt_answers_selected_option_id,omitempty"`
	UserQuizAttemptAnswersText            *string    `json:"user_quiz_attempt_answers_text,omitempty"`
	UserQuizAttemptAnswersIsCorrect       *bool      `json:"user_quiz_attempt_answers_is_correct,omitempty"`
	UserQuizAttemptAnswersEarnedPoints    float64    `json:"user_quiz_attempt_answers_earned_points"`
	UserQuizAttemptAnswersGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answers_graded_by_teacher_id,omitempty"`
	UserQuizAttemptAnswersGradedAt        *time.Time `json:"user_quiz_attempt_answers_graded_at,omitempty"`
	UserQuizAttemptAnswersFeedback        *string    `json:"user_quiz_attempt_answers_feedback,omitempty"`
	UserQuizAttemptAnswersAnsweredAt      time.Time  `json:"user_quiz_attempt_answers_answered_at"`
}

/* ===================== CONVERTERS ===================== */

func ToUserQuizAttemptAnswerResponse(m *model.UserQuizAttemptAnswerModel) *UserQuizAttemptAnswerResponse {
	if m == nil {
		return nil
	}
	return &UserQuizAttemptAnswerResponse{
		UserQuizAttemptAnswersID:               m.UserQuizAttemptAnswersID,
		UserQuizAttemptAnswersAttemptID:        m.UserQuizAttemptAnswersAttemptID,
		UserQuizAttemptAnswersQuestionID:       m.UserQuizAttemptAnswersQuestionID,
		UserQuizAttemptAnswersSelectedOptionID: m.UserQuizAttemptAnswersSelectedOptionID,
		UserQuizAttemptAnswersText:             m.UserQuizAttemptAnswersText,
		UserQuizAttemptAnswersIsCorrect:        m.UserQuizAttemptAnswersIsCorrect,
		UserQuizAttemptAnswersEarnedPoints:     m.UserQuizAttemptAnswersEarnedPoints,
		UserQuizAttemptAnswersGradedByTeacherID: m.UserQuizAttemptAnswersGradedByTeacherID,
		UserQuizAttemptAnswersGradedAt:         m.UserQuizAttemptAnswersGradedAt,
		UserQuizAttemptAnswersFeedback:         m.UserQuizAttemptAnswersFeedback,
		UserQuizAttemptAnswersAnsweredAt:       m.UserQuizAttemptAnswersAnsweredAt,
	}
}

func (r *CreateUserQuizAttemptAnswerRequest) ToModel() *model.UserQuizAttemptAnswerModel {
	m := &model.UserQuizAttemptAnswerModel{
		UserQuizAttemptAnswersAttemptID:        r.UserQuizAttemptAnswersAttemptID,
		UserQuizAttemptAnswersQuestionID:       r.UserQuizAttemptAnswersQuestionID,
		UserQuizAttemptAnswersSelectedOptionID: r.UserQuizAttemptAnswersSelectedOptionID,
		UserQuizAttemptAnswersText:             r.UserQuizAttemptAnswersText,
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
	if r.UserQuizAttemptAnswersSelectedOptionID != nil {
		m.UserQuizAttemptAnswersSelectedOptionID = r.UserQuizAttemptAnswersSelectedOptionID
	}
	if r.UserQuizAttemptAnswersText != nil {
		m.UserQuizAttemptAnswersText = r.UserQuizAttemptAnswersText
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
