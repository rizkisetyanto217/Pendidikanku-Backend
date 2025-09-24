// file: internals/features/school/submissions_assesments/quizzes/dto/user_quiz_attempt_answer_dto.go
package dto

import (
	"time"

	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

// Create — wajib: attempt_id, question_id, text.
// Catatan: quiz_id DIISI BACKEND dari konteks (bukan dari client).
type CreateUserQuizAttemptAnswerRequest struct {
	UserQuizAttemptAnswerAttemptID  uuid.UUID `json:"user_quiz_attempt_answer_attempt_id" validate:"required,uuid"`
	UserQuizAttemptAnswerQuestionID uuid.UUID `json:"user_quiz_attempt_answer_question_id" validate:"required,uuid"`

	// Jawaban user — wajib
	UserQuizAttemptAnswerText string `json:"user_quiz_attempt_answer_text" validate:"required"`

	// Opsi penilaian (biasanya diisi backend)
	UserQuizAttemptAnswerIsCorrect         *bool      `json:"user_quiz_attempt_answer_is_correct" validate:"omitempty"`
	UserQuizAttemptAnswerEarnedPoints      *float64   `json:"user_quiz_attempt_answer_earned_points" validate:"omitempty,gte=0,lte=9999.99"`
	UserQuizAttemptAnswerGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answer_graded_by_teacher_id" validate:"omitempty,uuid"`
	UserQuizAttemptAnswerGradedAt          *time.Time `json:"user_quiz_attempt_answer_graded_at" validate:"omitempty"`
	UserQuizAttemptAnswerFeedback          *string    `json:"user_quiz_attempt_answer_feedback" validate:"omitempty"`

	// Optional import historis
	UserQuizAttemptAnswerAnsweredAt *time.Time `json:"user_quiz_attempt_answer_answered_at" validate:"omitempty"`
}

func (r *CreateUserQuizAttemptAnswerRequest) ToModel() *model.UserQuizAttemptAnswerModel {
	m := &model.UserQuizAttemptAnswerModel{
		// Quiz ID HARUS diisi di controller: m.UserQuizAttemptAnswerQuizID = <derived>
		UserQuizAttemptAnswerAttemptID:  r.UserQuizAttemptAnswerAttemptID,
		UserQuizAttemptAnswerQuestionID: r.UserQuizAttemptAnswerQuestionID,
		UserQuizAttemptAnswerText:       r.UserQuizAttemptAnswerText,
	}
	if r.UserQuizAttemptAnswerIsCorrect != nil {
		m.UserQuizAttemptAnswerIsCorrect = r.UserQuizAttemptAnswerIsCorrect
	}
	if r.UserQuizAttemptAnswerEarnedPoints != nil {
		m.UserQuizAttemptAnswerEarnedPoints = *r.UserQuizAttemptAnswerEarnedPoints
	}
	if r.UserQuizAttemptAnswerGradedByTeacherID != nil {
		m.UserQuizAttemptAnswerGradedByTeacherID = r.UserQuizAttemptAnswerGradedByTeacherID
	}
	if r.UserQuizAttemptAnswerGradedAt != nil {
		m.UserQuizAttemptAnswerGradedAt = r.UserQuizAttemptAnswerGradedAt
	}
	if r.UserQuizAttemptAnswerFeedback != nil {
		m.UserQuizAttemptAnswerFeedback = r.UserQuizAttemptAnswerFeedback
	}
	if r.UserQuizAttemptAnswerAnsweredAt != nil {
		m.UserQuizAttemptAnswerAnsweredAt = *r.UserQuizAttemptAnswerAnsweredAt
	}
	return m
}

// Patch/Update — semua opsional; attempt_id/question_id tidak boleh diganti lewat DTO ini.
type UpdateUserQuizAttemptAnswerRequest struct {
	UserQuizAttemptAnswerText              *string    `json:"user_quiz_attempt_answer_text" validate:"omitempty"`
	UserQuizAttemptAnswerIsCorrect         *bool      `json:"user_quiz_attempt_answer_is_correct" validate:"omitempty"`
	UserQuizAttemptAnswerEarnedPoints      *float64   `json:"user_quiz_attempt_answer_earned_points" validate:"omitempty,gte=0,lte=9999.99"`
	UserQuizAttemptAnswerGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answer_graded_by_teacher_id" validate:"omitempty,uuid"`
	UserQuizAttemptAnswerGradedAt          *time.Time `json:"user_quiz_attempt_answer_graded_at" validate:"omitempty"`
	UserQuizAttemptAnswerFeedback          *string    `json:"user_quiz_attempt_answer_feedback" validate:"omitempty"`
	UserQuizAttemptAnswerAnsweredAt        *time.Time `json:"user_quiz_attempt_answer_answered_at" validate:"omitempty"`
}

// Apply ke model untuk PATCH (partial)
func (r *UpdateUserQuizAttemptAnswerRequest) Apply(m *model.UserQuizAttemptAnswerModel) {
	if r.UserQuizAttemptAnswerText != nil {
		m.UserQuizAttemptAnswerText = *r.UserQuizAttemptAnswerText
	}
	if r.UserQuizAttemptAnswerIsCorrect != nil {
		m.UserQuizAttemptAnswerIsCorrect = r.UserQuizAttemptAnswerIsCorrect
	}
	if r.UserQuizAttemptAnswerEarnedPoints != nil {
		m.UserQuizAttemptAnswerEarnedPoints = *r.UserQuizAttemptAnswerEarnedPoints
	}
	if r.UserQuizAttemptAnswerGradedByTeacherID != nil {
		m.UserQuizAttemptAnswerGradedByTeacherID = r.UserQuizAttemptAnswerGradedByTeacherID
	}
	if r.UserQuizAttemptAnswerGradedAt != nil {
		m.UserQuizAttemptAnswerGradedAt = r.UserQuizAttemptAnswerGradedAt
	}
	if r.UserQuizAttemptAnswerFeedback != nil {
		m.UserQuizAttemptAnswerFeedback = r.UserQuizAttemptAnswerFeedback
	}
	if r.UserQuizAttemptAnswerAnsweredAt != nil {
		m.UserQuizAttemptAnswerAnsweredAt = *r.UserQuizAttemptAnswerAnsweredAt
	}
}

/* ===================== RESPONSES ===================== */

type UserQuizAttemptAnswerResponse struct {
	UserQuizAttemptAnswerID         uuid.UUID `json:"user_quiz_attempt_answer_id"`
	UserQuizAttemptAnswerQuizID     uuid.UUID `json:"user_quiz_attempt_answer_quiz_id"`
	UserQuizAttemptAnswerAttemptID  uuid.UUID `json:"user_quiz_attempt_answer_attempt_id"`
	UserQuizAttemptAnswerQuestionID uuid.UUID `json:"user_quiz_attempt_answer_question_id"`

	UserQuizAttemptAnswerText         string  `json:"user_quiz_attempt_answer_text"`
	UserQuizAttemptAnswerIsCorrect    *bool   `json:"user_quiz_attempt_answer_is_correct,omitempty"`
	UserQuizAttemptAnswerEarnedPoints float64 `json:"user_quiz_attempt_answer_earned_points"`

	UserQuizAttemptAnswerGradedByTeacherID *uuid.UUID `json:"user_quiz_attempt_answer_graded_by_teacher_id,omitempty"`
	UserQuizAttemptAnswerGradedAt          *time.Time `json:"user_quiz_attempt_answer_graded_at,omitempty"`
	UserQuizAttemptAnswerFeedback          *string    `json:"user_quiz_attempt_answer_feedback,omitempty"`

	UserQuizAttemptAnswerAnsweredAt time.Time `json:"user_quiz_attempt_answer_answered_at"`
}

func FromModelUserQuizAttemptAnswer(m *model.UserQuizAttemptAnswerModel) *UserQuizAttemptAnswerResponse {
	if m == nil {
		return nil
	}
	return &UserQuizAttemptAnswerResponse{
		UserQuizAttemptAnswerID:                m.UserQuizAttemptAnswerID,
		UserQuizAttemptAnswerQuizID:            m.UserQuizAttemptAnswerQuizID,
		UserQuizAttemptAnswerAttemptID:         m.UserQuizAttemptAnswerAttemptID,
		UserQuizAttemptAnswerQuestionID:        m.UserQuizAttemptAnswerQuestionID,
		UserQuizAttemptAnswerText:              m.UserQuizAttemptAnswerText,
		UserQuizAttemptAnswerIsCorrect:         m.UserQuizAttemptAnswerIsCorrect,
		UserQuizAttemptAnswerEarnedPoints:      m.UserQuizAttemptAnswerEarnedPoints,
		UserQuizAttemptAnswerGradedByTeacherID: m.UserQuizAttemptAnswerGradedByTeacherID,
		UserQuizAttemptAnswerGradedAt:          m.UserQuizAttemptAnswerGradedAt,
		UserQuizAttemptAnswerFeedback:          m.UserQuizAttemptAnswerFeedback,
		UserQuizAttemptAnswerAnsweredAt:        m.UserQuizAttemptAnswerAnsweredAt,
	}
}
