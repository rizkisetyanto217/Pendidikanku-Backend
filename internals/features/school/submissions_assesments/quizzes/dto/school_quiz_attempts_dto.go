// file: internals/features/school/submissions_assesments/quizzes/dto/student_quiz_attempt_dto.go
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

type CreateStudentQuizAttemptRequest struct {
	// Opsional: bisa diisi server dari context
	StudentQuizAttemptMasjidID *uuid.UUID `json:"student_quiz_attempt_masjid_id" validate:"omitempty,uuid"`

	// Wajib
	StudentQuizAttemptQuizID uuid.UUID `json:"student_quiz_attempt_quiz_id" validate:"required,uuid"`

	// Opsional (untuk admin/dkm/teacher); untuk self-attempt bisa diisi server
	StudentQuizAttemptStudentID *uuid.UUID `json:"student_quiz_attempt_student_id" validate:"omitempty,uuid"`

	// Opsional
	StudentQuizAttemptStartedAt *time.Time `json:"student_quiz_attempt_started_at" validate:"omitempty"`
}

func (r *CreateStudentQuizAttemptRequest) ToModel() *qmodel.StudentQuizAttemptModel {
	m := &qmodel.StudentQuizAttemptModel{
		StudentQuizAttemptQuizID: r.StudentQuizAttemptQuizID,
	}
	if r.StudentQuizAttemptMasjidID != nil {
		m.StudentQuizAttemptMasjidID = *r.StudentQuizAttemptMasjidID
	}
	if r.StudentQuizAttemptStudentID != nil {
		m.StudentQuizAttemptStudentID = *r.StudentQuizAttemptStudentID
	}
	if r.StudentQuizAttemptStartedAt != nil {
		m.StudentQuizAttemptStartedAt = *r.StudentQuizAttemptStartedAt
	}
	return m
}

/* ==========================================================================================
   REQUEST — UPDATE/PATCH (PARTIAL)
   Gunakan pointer supaya field yang tidak dikirim tidak diubah.
========================================================================================== */

type UpdateStudentQuizAttemptRequest struct {
	StudentQuizAttemptMasjidID  *uuid.UUID `json:"student_quiz_attempt_masjid_id" validate:"omitempty,uuid"`
	StudentQuizAttemptQuizID    *uuid.UUID `json:"student_quiz_attempt_quiz_id" validate:"omitempty,uuid"`
	StudentQuizAttemptStudentID *uuid.UUID `json:"student_quiz_attempt_student_id" validate:"omitempty,uuid"`

	StudentQuizAttemptStartedAt  *time.Time `json:"student_quiz_attempt_started_at" validate:"omitempty"`
	StudentQuizAttemptFinishedAt *time.Time `json:"student_quiz_attempt_finished_at" validate:"omitempty"`

	StudentQuizAttemptScoreRaw     *float64 `json:"student_quiz_attempt_score_raw" validate:"omitempty"`
	StudentQuizAttemptScorePercent *float64 `json:"student_quiz_attempt_score_percent" validate:"omitempty"`

	StudentQuizAttemptStatus *qmodel.StudentQuizAttemptStatus `json:"student_quiz_attempt_status" validate:"omitempty,oneof=in_progress submitted finished abandoned"`
}

func isValidStatus(s qmodel.StudentQuizAttemptStatus) bool {
	switch s {
	case qmodel.StudentQuizAttemptInProgress,
		qmodel.StudentQuizAttemptSubmitted,
		qmodel.StudentQuizAttemptFinished,
		qmodel.StudentQuizAttemptAbandoned:
		return true
	default:
		return false
	}
}

// ApplyToModel — patch ke model yang sudah di-load lalu cek konsistensi sederhana.
func (r *UpdateStudentQuizAttemptRequest) ApplyToModel(m *qmodel.StudentQuizAttemptModel) error {
	if r.StudentQuizAttemptMasjidID != nil {
		m.StudentQuizAttemptMasjidID = *r.StudentQuizAttemptMasjidID
	}
	if r.StudentQuizAttemptQuizID != nil {
		m.StudentQuizAttemptQuizID = *r.StudentQuizAttemptQuizID
	}
	if r.StudentQuizAttemptStudentID != nil {
		m.StudentQuizAttemptStudentID = *r.StudentQuizAttemptStudentID
	}
	if r.StudentQuizAttemptStartedAt != nil {
		m.StudentQuizAttemptStartedAt = *r.StudentQuizAttemptStartedAt
	}
	if r.StudentQuizAttemptFinishedAt != nil {
		m.StudentQuizAttemptFinishedAt = r.StudentQuizAttemptFinishedAt
	}
	if r.StudentQuizAttemptScoreRaw != nil {
		m.StudentQuizAttemptScoreRaw = r.StudentQuizAttemptScoreRaw
	}
	if r.StudentQuizAttemptScorePercent != nil {
		m.StudentQuizAttemptScorePercent = r.StudentQuizAttemptScorePercent
	}
	if r.StudentQuizAttemptStatus != nil {
		if !isValidStatus(*r.StudentQuizAttemptStatus) {
			return fmt.Errorf("invalid status: %s", string(*r.StudentQuizAttemptStatus))
		}
		m.StudentQuizAttemptStatus = *r.StudentQuizAttemptStatus
	}

	// Konsistensi sederhana:
	// Jika status final dan finished_at masih nil → set sekarang.
	if m.StudentQuizAttemptStatus == qmodel.StudentQuizAttemptSubmitted ||
		m.StudentQuizAttemptStatus == qmodel.StudentQuizAttemptFinished ||
		m.StudentQuizAttemptStatus == qmodel.StudentQuizAttemptAbandoned {
		if m.StudentQuizAttemptFinishedAt == nil {
			now := time.Now()
			m.StudentQuizAttemptFinishedAt = &now
		}
	}

	return nil
}

/* ==========================================================================================
   RESPONSE DTO
========================================================================================== */

type StudentQuizAttemptResponse struct {
	StudentQuizAttemptID        uuid.UUID `json:"student_quiz_attempt_id"`
	StudentQuizAttemptMasjidID  uuid.UUID `json:"student_quiz_attempt_masjid_id"`
	StudentQuizAttemptQuizID    uuid.UUID `json:"student_quiz_attempt_quiz_id"`
	StudentQuizAttemptStudentID uuid.UUID `json:"student_quiz_attempt_student_id"`

	StudentQuizAttemptStartedAt  time.Time  `json:"student_quiz_attempt_started_at"`
	StudentQuizAttemptFinishedAt *time.Time `json:"student_quiz_attempt_finished_at,omitempty"`

	StudentQuizAttemptScoreRaw     *float64 `json:"student_quiz_attempt_score_raw,omitempty"`
	StudentQuizAttemptScorePercent *float64 `json:"student_quiz_attempt_score_percent,omitempty"`

	StudentQuizAttemptStatus qmodel.StudentQuizAttemptStatus `json:"student_quiz_attempt_status"`

	StudentQuizAttemptCreatedAt time.Time `json:"student_quiz_attempt_created_at"`
	StudentQuizAttemptUpdatedAt time.Time `json:"student_quiz_attempt_updated_at"`
}

func FromModelStudentQuizAttempt(m *qmodel.StudentQuizAttemptModel) *StudentQuizAttemptResponse {
	return &StudentQuizAttemptResponse{
		StudentQuizAttemptID:           m.StudentQuizAttemptID,
		StudentQuizAttemptMasjidID:     m.StudentQuizAttemptMasjidID,
		StudentQuizAttemptQuizID:       m.StudentQuizAttemptQuizID,
		StudentQuizAttemptStudentID:    m.StudentQuizAttemptStudentID,
		StudentQuizAttemptStartedAt:    m.StudentQuizAttemptStartedAt,
		StudentQuizAttemptFinishedAt:   m.StudentQuizAttemptFinishedAt,
		StudentQuizAttemptScoreRaw:     m.StudentQuizAttemptScoreRaw,
		StudentQuizAttemptScorePercent: m.StudentQuizAttemptScorePercent,
		StudentQuizAttemptStatus:       m.StudentQuizAttemptStatus,
		StudentQuizAttemptCreatedAt:    m.StudentQuizAttemptCreatedAt,
		StudentQuizAttemptUpdatedAt:    m.StudentQuizAttemptUpdatedAt,
	}
}

func FromModelsStudentQuizAttempts(items []qmodel.StudentQuizAttemptModel) []*StudentQuizAttemptResponse {
	out := make([]*StudentQuizAttemptResponse, 0, len(items))
	for i := range items {
		item := items[i]
		out = append(out, FromModelStudentQuizAttempt(&item))
	}
	return out
}
