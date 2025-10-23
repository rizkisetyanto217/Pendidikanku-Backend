// file: internals/features/school/submissions_assesments/quizzes/dto/student_quiz_attempt_answer_dto.go
package dto

import (
	"time"

	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"

	"github.com/google/uuid"
)

/* ===================== REQUESTS ===================== */

// Create — wajib: attempt_id, question_id, text.
// Catatan: quiz_id DIISI BACKEND dari konteks (bukan dari client).
type CreateStudentQuizAttemptAnswerRequest struct {
	StudentQuizAttemptAnswerAttemptID  uuid.UUID `json:"student_quiz_attempt_answer_attempt_id" validate:"required,uuid"`
	StudentQuizAttemptAnswerQuestionID uuid.UUID `json:"student_quiz_attempt_answer_question_id" validate:"required,uuid"`

	// Jawaban student — wajib
	StudentQuizAttemptAnswerText string `json:"student_quiz_attempt_answer_text" validate:"required"`

	// Opsi penilaian (biasanya diisi backend)
	StudentQuizAttemptAnswerIsCorrect         *bool      `json:"student_quiz_attempt_answer_is_correct" validate:"omitempty"`
	StudentQuizAttemptAnswerEarnedPoints      *float64   `json:"student_quiz_attempt_answer_earned_points" validate:"omitempty,gte=0,lte=9999.99"`
	StudentQuizAttemptAnswerGradedByTeacherID *uuid.UUID `json:"student_quiz_attempt_answer_graded_by_teacher_id" validate:"omitempty,uuid"`
	StudentQuizAttemptAnswerGradedAt          *time.Time `json:"student_quiz_attempt_answer_graded_at" validate:"omitempty"`
	StudentQuizAttemptAnswerFeedback          *string    `json:"student_quiz_attempt_answer_feedback" validate:"omitempty"`

	// Optional import historis
	StudentQuizAttemptAnswerAnsweredAt *time.Time `json:"student_quiz_attempt_answer_answered_at" validate:"omitempty"`
}

func (r *CreateStudentQuizAttemptAnswerRequest) ToModel() *model.StudentQuizAttemptAnswerModel {
	m := &model.StudentQuizAttemptAnswerModel{
		// Quiz ID HARUS diisi di controller: m.StudentQuizAttemptAnswerQuizID = <derived>
		StudentQuizAttemptAnswerAttemptID:  r.StudentQuizAttemptAnswerAttemptID,
		StudentQuizAttemptAnswerQuestionID: r.StudentQuizAttemptAnswerQuestionID,
		StudentQuizAttemptAnswerText:       r.StudentQuizAttemptAnswerText,
	}
	if r.StudentQuizAttemptAnswerIsCorrect != nil {
		m.StudentQuizAttemptAnswerIsCorrect = r.StudentQuizAttemptAnswerIsCorrect
	}
	if r.StudentQuizAttemptAnswerEarnedPoints != nil {
		m.StudentQuizAttemptAnswerEarnedPoints = *r.StudentQuizAttemptAnswerEarnedPoints
	}
	if r.StudentQuizAttemptAnswerGradedByTeacherID != nil {
		m.StudentQuizAttemptAnswerGradedByTeacherID = r.StudentQuizAttemptAnswerGradedByTeacherID
	}
	if r.StudentQuizAttemptAnswerGradedAt != nil {
		m.StudentQuizAttemptAnswerGradedAt = r.StudentQuizAttemptAnswerGradedAt
	}
	if r.StudentQuizAttemptAnswerFeedback != nil {
		m.StudentQuizAttemptAnswerFeedback = r.StudentQuizAttemptAnswerFeedback
	}
	if r.StudentQuizAttemptAnswerAnsweredAt != nil {
		m.StudentQuizAttemptAnswerAnsweredAt = *r.StudentQuizAttemptAnswerAnsweredAt
	}
	return m
}

// Patch/Update — semua opsional; attempt_id/question_id tidak boleh diganti lewat DTO ini.
type UpdateStudentQuizAttemptAnswerRequest struct {
	StudentQuizAttemptAnswerText              *string    `json:"student_quiz_attempt_answer_text" validate:"omitempty"`
	StudentQuizAttemptAnswerIsCorrect         *bool      `json:"student_quiz_attempt_answer_is_correct" validate:"omitempty"`
	StudentQuizAttemptAnswerEarnedPoints      *float64   `json:"student_quiz_attempt_answer_earned_points" validate:"omitempty,gte=0,lte=9999.99"`
	StudentQuizAttemptAnswerGradedByTeacherID *uuid.UUID `json:"student_quiz_attempt_answer_graded_by_teacher_id" validate:"omitempty,uuid"`
	StudentQuizAttemptAnswerGradedAt          *time.Time `json:"student_quiz_attempt_answer_graded_at" validate:"omitempty"`
	StudentQuizAttemptAnswerFeedback          *string    `json:"student_quiz_attempt_answer_feedback" validate:"omitempty"`
	StudentQuizAttemptAnswerAnsweredAt        *time.Time `json:"student_quiz_attempt_answer_answered_at" validate:"omitempty"`
}

// Apply ke model untuk PATCH (partial)
func (r *UpdateStudentQuizAttemptAnswerRequest) Apply(m *model.StudentQuizAttemptAnswerModel) {
	if r.StudentQuizAttemptAnswerText != nil {
		m.StudentQuizAttemptAnswerText = *r.StudentQuizAttemptAnswerText
	}
	if r.StudentQuizAttemptAnswerIsCorrect != nil {
		m.StudentQuizAttemptAnswerIsCorrect = r.StudentQuizAttemptAnswerIsCorrect
	}
	if r.StudentQuizAttemptAnswerEarnedPoints != nil {
		m.StudentQuizAttemptAnswerEarnedPoints = *r.StudentQuizAttemptAnswerEarnedPoints
	}
	if r.StudentQuizAttemptAnswerGradedByTeacherID != nil {
		m.StudentQuizAttemptAnswerGradedByTeacherID = r.StudentQuizAttemptAnswerGradedByTeacherID
	}
	if r.StudentQuizAttemptAnswerGradedAt != nil {
		m.StudentQuizAttemptAnswerGradedAt = r.StudentQuizAttemptAnswerGradedAt
	}
	if r.StudentQuizAttemptAnswerFeedback != nil {
		m.StudentQuizAttemptAnswerFeedback = r.StudentQuizAttemptAnswerFeedback
	}
	if r.StudentQuizAttemptAnswerAnsweredAt != nil {
		m.StudentQuizAttemptAnswerAnsweredAt = *r.StudentQuizAttemptAnswerAnsweredAt
	}
}

/* ===================== RESPONSES ===================== */

type StudentQuizAttemptAnswerResponse struct {
	StudentQuizAttemptAnswerID         uuid.UUID `json:"student_quiz_attempt_answer_id"`
	StudentQuizAttemptAnswerQuizID     uuid.UUID `json:"student_quiz_attempt_answer_quiz_id"`
	StudentQuizAttemptAnswerAttemptID  uuid.UUID `json:"student_quiz_attempt_answer_attempt_id"`
	StudentQuizAttemptAnswerQuestionID uuid.UUID `json:"student_quiz_attempt_answer_question_id"`

	StudentQuizAttemptAnswerText         string  `json:"student_quiz_attempt_answer_text"`
	StudentQuizAttemptAnswerIsCorrect    *bool   `json:"student_quiz_attempt_answer_is_correct,omitempty"`
	StudentQuizAttemptAnswerEarnedPoints float64 `json:"student_quiz_attempt_answer_earned_points"`

	StudentQuizAttemptAnswerGradedByTeacherID *uuid.UUID `json:"student_quiz_attempt_answer_graded_by_teacher_id,omitempty"`
	StudentQuizAttemptAnswerGradedAt          *time.Time `json:"student_quiz_attempt_answer_graded_at,omitempty"`
	StudentQuizAttemptAnswerFeedback          *string    `json:"student_quiz_attempt_answer_feedback,omitempty"`

	StudentQuizAttemptAnswerAnsweredAt time.Time `json:"student_quiz_attempt_answer_answered_at"`
}

func FromModelStudentQuizAttemptAnswer(m *model.StudentQuizAttemptAnswerModel) *StudentQuizAttemptAnswerResponse {
	if m == nil {
		return nil
	}
	return &StudentQuizAttemptAnswerResponse{
		StudentQuizAttemptAnswerID:                m.StudentQuizAttemptAnswerID,
		StudentQuizAttemptAnswerQuizID:            m.StudentQuizAttemptAnswerQuizID,
		StudentQuizAttemptAnswerAttemptID:         m.StudentQuizAttemptAnswerAttemptID,
		StudentQuizAttemptAnswerQuestionID:        m.StudentQuizAttemptAnswerQuestionID,
		StudentQuizAttemptAnswerText:              m.StudentQuizAttemptAnswerText,
		StudentQuizAttemptAnswerIsCorrect:         m.StudentQuizAttemptAnswerIsCorrect,
		StudentQuizAttemptAnswerEarnedPoints:      m.StudentQuizAttemptAnswerEarnedPoints,
		StudentQuizAttemptAnswerGradedByTeacherID: m.StudentQuizAttemptAnswerGradedByTeacherID,
		StudentQuizAttemptAnswerGradedAt:          m.StudentQuizAttemptAnswerGradedAt,
		StudentQuizAttemptAnswerFeedback:          m.StudentQuizAttemptAnswerFeedback,
		StudentQuizAttemptAnswerAnsweredAt:        m.StudentQuizAttemptAnswerAnsweredAt,
	}
}
