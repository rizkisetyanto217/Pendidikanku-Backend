// file: internals/features/school/submissions_assesments/quizzes/dto/user_quiz_attempt_dto.go
package dto

import (
	"fmt"
	"time"

	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"

	"github.com/google/uuid"
)

/* ==========================================================================================
   REQUEST — CREATE (memulai attempt)
   Server boleh meng-derive masjid_id / student_id dari context.
========================================================================================== */

type CreateUserQuizAttemptRequest struct {
	// Opsional: bisa diisi server dari context
	UserQuizAttemptMasjidID *uuid.UUID `json:"user_quiz_attempt_masjid_id" validate:"omitempty,uuid"`

	// Wajib
	UserQuizAttemptQuizID uuid.UUID `json:"user_quiz_attempt_quiz_id" validate:"required,uuid"`

	// Opsional (untuk admin/dkm/teacher); untuk self-attempt bisa diisi server
	UserQuizAttemptStudentID *uuid.UUID `json:"user_quiz_attempt_student_id" validate:"omitempty,uuid"`

	// Opsional
	UserQuizAttemptStartedAt *time.Time `json:"user_quiz_attempt_started_at" validate:"omitempty"`
}

func (r *CreateUserQuizAttemptRequest) ToModel() *qmodel.UserQuizAttemptModel {
	m := &qmodel.UserQuizAttemptModel{
		UserQuizAttemptQuizID: r.UserQuizAttemptQuizID,
	}
	if r.UserQuizAttemptMasjidID != nil {
		m.UserQuizAttemptMasjidID = *r.UserQuizAttemptMasjidID
	}
	if r.UserQuizAttemptStudentID != nil {
		m.UserQuizAttemptStudentID = *r.UserQuizAttemptStudentID
	}
	if r.UserQuizAttemptStartedAt != nil {
		m.UserQuizAttemptStartedAt = *r.UserQuizAttemptStartedAt
	}
	return m
}

/* ==========================================================================================
   REQUEST — UPDATE/PATCH (PARTIAL)
   Gunakan pointer supaya field yang tidak dikirim tidak diubah.
========================================================================================== */

type UpdateUserQuizAttemptRequest struct {
	UserQuizAttemptMasjidID  *uuid.UUID `json:"user_quiz_attempt_masjid_id" validate:"omitempty,uuid"`
	UserQuizAttemptQuizID    *uuid.UUID `json:"user_quiz_attempt_quiz_id" validate:"omitempty,uuid"`
	UserQuizAttemptStudentID *uuid.UUID `json:"user_quiz_attempt_student_id" validate:"omitempty,uuid"`

	UserQuizAttemptStartedAt  *time.Time `json:"user_quiz_attempt_started_at" validate:"omitempty"`
	UserQuizAttemptFinishedAt *time.Time `json:"user_quiz_attempt_finished_at" validate:"omitempty"`

	UserQuizAttemptScoreRaw     *float64 `json:"user_quiz_attempt_score_raw" validate:"omitempty"`
	UserQuizAttemptScorePercent *float64 `json:"user_quiz_attempt_score_percent" validate:"omitempty"`

	UserQuizAttemptStatus *qmodel.UserQuizAttemptStatus `json:"user_quiz_attempt_status" validate:"omitempty,oneof=in_progress submitted finished abandoned"`
}

func isValidStatus(s qmodel.UserQuizAttemptStatus) bool {
	switch s {
	case qmodel.UserQuizAttemptInProgress,
		qmodel.UserQuizAttemptSubmitted,
		qmodel.UserQuizAttemptFinished,
		qmodel.UserQuizAttemptAbandoned:
		return true
	default:
		return false
	}
}

// ApplyToModel — patch ke model yang sudah di-load lalu cek konsistensi sederhana.
func (r *UpdateUserQuizAttemptRequest) ApplyToModel(m *qmodel.UserQuizAttemptModel) error {
	if r.UserQuizAttemptMasjidID != nil {
		m.UserQuizAttemptMasjidID = *r.UserQuizAttemptMasjidID
	}
	if r.UserQuizAttemptQuizID != nil {
		m.UserQuizAttemptQuizID = *r.UserQuizAttemptQuizID
	}
	if r.UserQuizAttemptStudentID != nil {
		m.UserQuizAttemptStudentID = *r.UserQuizAttemptStudentID
	}
	if r.UserQuizAttemptStartedAt != nil {
		m.UserQuizAttemptStartedAt = *r.UserQuizAttemptStartedAt
	}
	if r.UserQuizAttemptFinishedAt != nil {
		m.UserQuizAttemptFinishedAt = r.UserQuizAttemptFinishedAt
	}
	if r.UserQuizAttemptScoreRaw != nil {
		m.UserQuizAttemptScoreRaw = r.UserQuizAttemptScoreRaw
	}
	if r.UserQuizAttemptScorePercent != nil {
		m.UserQuizAttemptScorePercent = r.UserQuizAttemptScorePercent
	}
	if r.UserQuizAttemptStatus != nil {
		if !isValidStatus(*r.UserQuizAttemptStatus) {
			return fmt.Errorf("invalid status: %s", string(*r.UserQuizAttemptStatus))
		}
		m.UserQuizAttemptStatus = *r.UserQuizAttemptStatus
	}

	// Konsistensi sederhana:
	// Jika status final dan finished_at masih nil → set sekarang.
	if m.UserQuizAttemptStatus == qmodel.UserQuizAttemptSubmitted ||
		m.UserQuizAttemptStatus == qmodel.UserQuizAttemptFinished ||
		m.UserQuizAttemptStatus == qmodel.UserQuizAttemptAbandoned {
		if m.UserQuizAttemptFinishedAt == nil {
			now := time.Now()
			m.UserQuizAttemptFinishedAt = &now
		}
	}

	return nil
}

/* ==========================================================================================
   RESPONSE DTO
========================================================================================== */

type UserQuizAttemptResponse struct {
	UserQuizAttemptID        uuid.UUID `json:"user_quiz_attempt_id"`
	UserQuizAttemptMasjidID  uuid.UUID `json:"user_quiz_attempt_masjid_id"`
	UserQuizAttemptQuizID    uuid.UUID `json:"user_quiz_attempt_quiz_id"`
	UserQuizAttemptStudentID uuid.UUID `json:"user_quiz_attempt_student_id"`

	UserQuizAttemptStartedAt  time.Time  `json:"user_quiz_attempt_started_at"`
	UserQuizAttemptFinishedAt *time.Time `json:"user_quiz_attempt_finished_at,omitempty"`

	UserQuizAttemptScoreRaw     *float64 `json:"user_quiz_attempt_score_raw,omitempty"`
	UserQuizAttemptScorePercent *float64 `json:"user_quiz_attempt_score_percent,omitempty"`

	UserQuizAttemptStatus qmodel.UserQuizAttemptStatus `json:"user_quiz_attempt_status"`

	UserQuizAttemptCreatedAt time.Time `json:"user_quiz_attempt_created_at"`
	UserQuizAttemptUpdatedAt time.Time `json:"user_quiz_attempt_updated_at"`
}

func FromModelUserQuizAttempt(m *qmodel.UserQuizAttemptModel) *UserQuizAttemptResponse {
	return &UserQuizAttemptResponse{
		UserQuizAttemptID:           m.UserQuizAttemptID,
		UserQuizAttemptMasjidID:     m.UserQuizAttemptMasjidID,
		UserQuizAttemptQuizID:       m.UserQuizAttemptQuizID,
		UserQuizAttemptStudentID:    m.UserQuizAttemptStudentID,
		UserQuizAttemptStartedAt:    m.UserQuizAttemptStartedAt,
		UserQuizAttemptFinishedAt:   m.UserQuizAttemptFinishedAt,
		UserQuizAttemptScoreRaw:     m.UserQuizAttemptScoreRaw,
		UserQuizAttemptScorePercent: m.UserQuizAttemptScorePercent,
		UserQuizAttemptStatus:       m.UserQuizAttemptStatus,
		UserQuizAttemptCreatedAt:    m.UserQuizAttemptCreatedAt,
		UserQuizAttemptUpdatedAt:    m.UserQuizAttemptUpdatedAt,
	}
}

func FromModelsUserQuizAttempts(items []qmodel.UserQuizAttemptModel) []*UserQuizAttemptResponse {
	out := make([]*UserQuizAttemptResponse, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, FromModelUserQuizAttempt(&item))
	}
	return out
}
