// file: internals/features/school/submissions_assesments/quizzes/dto/user_quiz_attempt_dto.go
package dto

import (
	"fmt"
	"time"

	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"

	"github.com/google/uuid"
)

/* ==========================================================================================
   REQUEST — CREATE
   Use case: memulai attempt (status default 'in_progress')
========================================================================================== */

// file: .../dto/user_quiz_attempt_dto.go

type CreateUserQuizAttemptRequest struct {
	// Opsional: server akan derive dari quiz
	UserQuizAttemptsMasjidID   *uuid.UUID `json:"user_quiz_attempts_masjid_id" validate:"omitempty,uuid"`

	// Wajib
	UserQuizAttemptsQuizID     uuid.UUID  `json:"user_quiz_attempts_quiz_id" validate:"required,uuid"`

	// Opsional: siswa tidak perlu kirim; admin/dkm/teacher wajib kirim
	UserQuizAttemptsStudentID  *uuid.UUID `json:"user_quiz_attempts_student_id" validate:"omitempty,uuid"`

	// Opsional
	UserQuizAttemptsStartedAt  *time.Time `json:"user_quiz_attempts_started_at" validate:"omitempty"`
}

func (r *CreateUserQuizAttemptRequest) ToModel() *qmodel.UserQuizAttemptModel {
	m := &qmodel.UserQuizAttemptModel{
		UserQuizAttemptsQuizID: r.UserQuizAttemptsQuizID,
	}
	if r.UserQuizAttemptsMasjidID != nil {
		m.UserQuizAttemptsMasjidID = *r.UserQuizAttemptsMasjidID
	}
	if r.UserQuizAttemptsStudentID != nil {
		m.UserQuizAttemptsStudentID = *r.UserQuizAttemptsStudentID
	}
	if r.UserQuizAttemptsStartedAt != nil {
		m.UserQuizAttemptsStartedAt = *r.UserQuizAttemptsStartedAt
	}
	return m
}

/* ==========================================================================================
   REQUEST — UPDATE/PATCH (PARTIAL)
   - Gunakan pointer supaya field yang tidak dikirim tidak diubah.
   - Ada helper konsistensi sederhana (status final → isi finished_at kalau belum).
========================================================================================== */

type UpdateUserQuizAttemptRequest struct {
	UserQuizAttemptsMasjidID  *uuid.UUID                     `json:"user_quiz_attempts_masjid_id" validate:"omitempty"`
	UserQuizAttemptsQuizID    *uuid.UUID                     `json:"user_quiz_attempts_quiz_id" validate:"omitempty"`
	UserQuizAttemptsStudentID *uuid.UUID                     `json:"user_quiz_attempts_student_id" validate:"omitempty"`

	UserQuizAttemptsStartedAt  *time.Time                    `json:"user_quiz_attempts_started_at" validate:"omitempty"`
	UserQuizAttemptsFinishedAt *time.Time                    `json:"user_quiz_attempts_finished_at" validate:"omitempty"`

	UserQuizAttemptsScoreRaw     *float64                    `json:"user_quiz_attempts_score_raw" validate:"omitempty"`
	UserQuizAttemptsScorePercent *float64                    `json:"user_quiz_attempts_score_percent" validate:"omitempty"`

	UserQuizAttemptsStatus *qmodel.UserQuizAttemptStatus     `json:"user_quiz_attempts_status" validate:"omitempty,oneof=in_progress submitted finished abandoned"`
}

// ApplyToModel — patch ke model yang sudah di-load lalu cek konsistensi.
func (r *UpdateUserQuizAttemptRequest) ApplyToModel(m *qmodel.UserQuizAttemptModel) error {
	if r.UserQuizAttemptsMasjidID != nil {
		m.UserQuizAttemptsMasjidID = *r.UserQuizAttemptsMasjidID
	}
	if r.UserQuizAttemptsQuizID != nil {
		m.UserQuizAttemptsQuizID = *r.UserQuizAttemptsQuizID
	}
	if r.UserQuizAttemptsStudentID != nil {
		m.UserQuizAttemptsStudentID = *r.UserQuizAttemptsStudentID
	}
	if r.UserQuizAttemptsStartedAt != nil {
		m.UserQuizAttemptsStartedAt = *r.UserQuizAttemptsStartedAt
	}
	if r.UserQuizAttemptsFinishedAt != nil {
		m.UserQuizAttemptsFinishedAt = r.UserQuizAttemptsFinishedAt
	}
	if r.UserQuizAttemptsScoreRaw != nil {
		m.UserQuizAttemptsScoreRaw = r.UserQuizAttemptsScoreRaw
	}
	if r.UserQuizAttemptsScorePercent != nil {
		m.UserQuizAttemptsScorePercent = r.UserQuizAttemptsScorePercent
	}
	if r.UserQuizAttemptsStatus != nil {
		// validasi enum
		if !r.UserQuizAttemptsStatus.Valid() {
			return fmt.Errorf("invalid status: %s", r.UserQuizAttemptsStatus.String())
		}
		m.UserQuizAttemptsStatus = *r.UserQuizAttemptsStatus
	}

	// Konsistensi sederhana:
	// - Jika status submitted/finished/abandoned dan finished_at masih nil → set ke now.
	// - Jika status in_progress namun finished_at terisi, biarkan (kasus edge: undo manual).
	if m.UserQuizAttemptsStatus == qmodel.UserAttemptSubmitted ||
		m.UserQuizAttemptsStatus == qmodel.UserAttemptFinished ||
		m.UserQuizAttemptsStatus == qmodel.UserAttemptAbandoned {
		if m.UserQuizAttemptsFinishedAt == nil {
			now := time.Now()
			m.UserQuizAttemptsFinishedAt = &now
		}
	}

	return nil
}

/* ==========================================================================================
   RESPONSE DTO
========================================================================================== */

type UserQuizAttemptResponse struct {
	UserQuizAttemptsID        uuid.UUID                    `json:"user_quiz_attempts_id"`
	UserQuizAttemptsMasjidID  uuid.UUID                    `json:"user_quiz_attempts_masjid_id"`
	UserQuizAttemptsQuizID    uuid.UUID                    `json:"user_quiz_attempts_quiz_id"`
	UserQuizAttemptsStudentID uuid.UUID                    `json:"user_quiz_attempts_student_id"`

	UserQuizAttemptsStartedAt  time.Time                   `json:"user_quiz_attempts_started_at"`
	UserQuizAttemptsFinishedAt *time.Time                  `json:"user_quiz_attempts_finished_at,omitempty"`

	UserQuizAttemptsScoreRaw     *float64                  `json:"user_quiz_attempts_score_raw,omitempty"`
	UserQuizAttemptsScorePercent *float64                  `json:"user_quiz_attempts_score_percent,omitempty"`

	UserQuizAttemptsStatus qmodel.UserQuizAttemptStatus    `json:"user_quiz_attempts_status"`

	UserQuizAttemptsCreatedAt time.Time                    `json:"user_quiz_attempts_created_at"`
	UserQuizAttemptsUpdatedAt time.Time                    `json:"user_quiz_attempts_updated_at"`
}

func FromModelUserQuizAttempt(m *qmodel.UserQuizAttemptModel) *UserQuizAttemptResponse {
	return &UserQuizAttemptResponse{
		UserQuizAttemptsID:         m.UserQuizAttemptsID,
		UserQuizAttemptsMasjidID:   m.UserQuizAttemptsMasjidID,
		UserQuizAttemptsQuizID:     m.UserQuizAttemptsQuizID,
		UserQuizAttemptsStudentID:  m.UserQuizAttemptsStudentID,
		UserQuizAttemptsStartedAt:  m.UserQuizAttemptsStartedAt,
		UserQuizAttemptsFinishedAt: m.UserQuizAttemptsFinishedAt,
		UserQuizAttemptsScoreRaw:   m.UserQuizAttemptsScoreRaw,
		UserQuizAttemptsScorePercent: m.UserQuizAttemptsScorePercent,
		UserQuizAttemptsStatus:     m.UserQuizAttemptsStatus,
		UserQuizAttemptsCreatedAt:  m.UserQuizAttemptsCreatedAt,
		UserQuizAttemptsUpdatedAt:  m.UserQuizAttemptsUpdatedAt,
	}
}

func FromModelsUserQuizAttempts(items []*qmodel.UserQuizAttemptModel) []*UserQuizAttemptResponse {
	out := make([]*UserQuizAttemptResponse, 0, len(items))
	for _, it := range items {
		out = append(out, FromModelUserQuizAttempt(it))
	}
	return out
}
