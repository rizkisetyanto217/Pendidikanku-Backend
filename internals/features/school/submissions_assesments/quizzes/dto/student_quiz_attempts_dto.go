package dto

import (
	"encoding/json"
	"time"

	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* ==========================================================================================
   REQUEST — CREATE (membuat summary row student × quiz)
   Biasanya dipanggil saat student pertama kali mulai quiz.
   Server boleh meng-derive school_id / student_id dari context.
========================================================================================== */

type CreateStudentQuizAttemptRequest struct {
	// Opsional: bisa diisi server dari context
	StudentQuizAttemptSchoolID *uuid.UUID `json:"student_quiz_attempt_school_id" validate:"omitempty,uuid"`

	// Wajib: quiz yang dimaksud
	StudentQuizAttemptQuizID uuid.UUID `json:"student_quiz_attempt_quiz_id" validate:"required,uuid"`

	// Opsional (untuk admin/dkm/teacher); untuk self-attempt bisa diisi server
	StudentQuizAttemptStudentID *uuid.UUID `json:"student_quiz_attempt_student_id" validate:"omitempty,uuid"`
}

func (r *CreateStudentQuizAttemptRequest) ToModel() *qmodel.StudentQuizAttemptModel {
	m := &qmodel.StudentQuizAttemptModel{
		StudentQuizAttemptQuizID: r.StudentQuizAttemptQuizID,
		// History default: [] (sudah di tag GORM)
		// Count default: 0
	}

	if r.StudentQuizAttemptSchoolID != nil {
		m.StudentQuizAttemptSchoolID = *r.StudentQuizAttemptSchoolID
	}
	if r.StudentQuizAttemptStudentID != nil {
		m.StudentQuizAttemptStudentID = *r.StudentQuizAttemptStudentID
	}

	return m
}

/* ==========================================================================================
   REQUEST — UPDATE/PATCH (PARTIAL)
   Gunakan pointer supaya field yang tidak dikirim tidak diubah.
   Biasanya dipakai internal (service) untuk update summary:
   - history JSON
   - count
   - best_*
   - last_*
========================================================================================== */

type UpdateStudentQuizAttemptRequest struct {
	StudentQuizAttemptSchoolID  *uuid.UUID `json:"student_quiz_attempt_school_id" validate:"omitempty,uuid"`
	StudentQuizAttemptQuizID    *uuid.UUID `json:"student_quiz_attempt_quiz_id" validate:"omitempty,uuid"`
	StudentQuizAttemptStudentID *uuid.UUID `json:"student_quiz_attempt_student_id" validate:"omitempty,uuid"`

	// Full history (opsional, biasanya diisi backend)
	StudentQuizAttemptHistory *json.RawMessage `json:"student_quiz_attempt_history" validate:"omitempty"`

	// Summary: total attempt
	StudentQuizAttemptCount *int `json:"student_quiz_attempt_count" validate:"omitempty,gte=0"`

	// Summary: nilai terbaik
	StudentQuizAttemptBestRaw        *float64   `json:"student_quiz_attempt_best_raw" validate:"omitempty"`
	StudentQuizAttemptBestPercent    *float64   `json:"student_quiz_attempt_best_percent" validate:"omitempty"`
	StudentQuizAttemptBestStartedAt  *time.Time `json:"student_quiz_attempt_best_started_at" validate:"omitempty"`
	StudentQuizAttemptBestFinishedAt *time.Time `json:"student_quiz_attempt_best_finished_at" validate:"omitempty"`

	// Summary: nilai terakhir
	StudentQuizAttemptLastRaw        *float64   `json:"student_quiz_attempt_last_raw" validate:"omitempty"`
	StudentQuizAttemptLastPercent    *float64   `json:"student_quiz_attempt_last_percent" validate:"omitempty"`
	StudentQuizAttemptLastStartedAt  *time.Time `json:"student_quiz_attempt_last_started_at" validate:"omitempty"`
	StudentQuizAttemptLastFinishedAt *time.Time `json:"student_quiz_attempt_last_finished_at" validate:"omitempty"`
}

// ApplyToModel — patch ke model yang sudah di-load.
// Business logic (recompute best/last dari history) bisa ditaruh di service,
// di luar DTO ini, supaya lebih bersih.
func (r *UpdateStudentQuizAttemptRequest) ApplyToModel(m *qmodel.StudentQuizAttemptModel) error {
	if r.StudentQuizAttemptSchoolID != nil {
		m.StudentQuizAttemptSchoolID = *r.StudentQuizAttemptSchoolID
	}
	if r.StudentQuizAttemptQuizID != nil {
		m.StudentQuizAttemptQuizID = *r.StudentQuizAttemptQuizID
	}
	if r.StudentQuizAttemptStudentID != nil {
		m.StudentQuizAttemptStudentID = *r.StudentQuizAttemptStudentID
	}

	if r.StudentQuizAttemptHistory != nil {
		// datatypes.JSON di model adalah []byte, json.RawMessage juga []byte → langsung cast
		m.StudentQuizAttemptHistory = JSONFromRaw(*r.StudentQuizAttemptHistory)
	}

	if r.StudentQuizAttemptCount != nil {
		m.StudentQuizAttemptCount = *r.StudentQuizAttemptCount
	}

	// Best summary
	if r.StudentQuizAttemptBestRaw != nil {
		m.StudentQuizAttemptBestRaw = r.StudentQuizAttemptBestRaw
	}
	if r.StudentQuizAttemptBestPercent != nil {
		m.StudentQuizAttemptBestPercent = r.StudentQuizAttemptBestPercent
	}
	if r.StudentQuizAttemptBestStartedAt != nil {
		m.StudentQuizAttemptBestStartedAt = r.StudentQuizAttemptBestStartedAt
	}
	if r.StudentQuizAttemptBestFinishedAt != nil {
		m.StudentQuizAttemptBestFinishedAt = r.StudentQuizAttemptBestFinishedAt
	}

	// Last summary
	if r.StudentQuizAttemptLastRaw != nil {
		m.StudentQuizAttemptLastRaw = r.StudentQuizAttemptLastRaw
	}
	if r.StudentQuizAttemptLastPercent != nil {
		m.StudentQuizAttemptLastPercent = r.StudentQuizAttemptLastPercent
	}
	if r.StudentQuizAttemptLastStartedAt != nil {
		m.StudentQuizAttemptLastStartedAt = r.StudentQuizAttemptLastStartedAt
	}
	if r.StudentQuizAttemptLastFinishedAt != nil {
		m.StudentQuizAttemptLastFinishedAt = r.StudentQuizAttemptLastFinishedAt
	}

	return nil
}

/* ==========================================================================================
   RESPONSE DTO
   Ini yang dikirim ke FE, sudah sesuai dengan model JSON summary.
========================================================================================== */

type StudentQuizAttemptResponse struct {
	StudentQuizAttemptID        uuid.UUID `json:"student_quiz_attempt_id"`
	StudentQuizAttemptSchoolID  uuid.UUID `json:"student_quiz_attempt_school_id"`
	StudentQuizAttemptQuizID    uuid.UUID `json:"student_quiz_attempt_quiz_id"`
	StudentQuizAttemptStudentID uuid.UUID `json:"student_quiz_attempt_student_id"`

	// History full (biar FE bisa tampilkan riwayat attempt + jawaban)
	StudentQuizAttemptHistory json.RawMessage `json:"student_quiz_attempt_history"`

	// Summary
	StudentQuizAttemptCount int `json:"student_quiz_attempt_count"`

	StudentQuizAttemptBestRaw        *float64   `json:"student_quiz_attempt_best_raw,omitempty"`
	StudentQuizAttemptBestPercent    *float64   `json:"student_quiz_attempt_best_percent,omitempty"`
	StudentQuizAttemptBestStartedAt  *time.Time `json:"student_quiz_attempt_best_started_at,omitempty"`
	StudentQuizAttemptBestFinishedAt *time.Time `json:"student_quiz_attempt_best_finished_at,omitempty"`

	StudentQuizAttemptLastRaw        *float64   `json:"student_quiz_attempt_last_raw,omitempty"`
	StudentQuizAttemptLastPercent    *float64   `json:"student_quiz_attempt_last_percent,omitempty"`
	StudentQuizAttemptLastStartedAt  *time.Time `json:"student_quiz_attempt_last_started_at,omitempty"`
	StudentQuizAttemptLastFinishedAt *time.Time `json:"student_quiz_attempt_last_finished_at,omitempty"`

	StudentQuizAttemptCreatedAt time.Time `json:"student_quiz_attempt_created_at"`
	StudentQuizAttemptUpdatedAt time.Time `json:"student_quiz_attempt_updated_at"`
}

func FromModelStudentQuizAttempt(m *qmodel.StudentQuizAttemptModel) *StudentQuizAttemptResponse {
	return &StudentQuizAttemptResponse{
		StudentQuizAttemptID:        m.StudentQuizAttemptID,
		StudentQuizAttemptSchoolID:  m.StudentQuizAttemptSchoolID,
		StudentQuizAttemptQuizID:    m.StudentQuizAttemptQuizID,
		StudentQuizAttemptStudentID: m.StudentQuizAttemptStudentID,

		StudentQuizAttemptHistory: json.RawMessage(m.StudentQuizAttemptHistory),

		StudentQuizAttemptCount: m.StudentQuizAttemptCount,

		StudentQuizAttemptBestRaw:        m.StudentQuizAttemptBestRaw,
		StudentQuizAttemptBestPercent:    m.StudentQuizAttemptBestPercent,
		StudentQuizAttemptBestStartedAt:  m.StudentQuizAttemptBestStartedAt,
		StudentQuizAttemptBestFinishedAt: m.StudentQuizAttemptBestFinishedAt,

		StudentQuizAttemptLastRaw:        m.StudentQuizAttemptLastRaw,
		StudentQuizAttemptLastPercent:    m.StudentQuizAttemptLastPercent,
		StudentQuizAttemptLastStartedAt:  m.StudentQuizAttemptLastStartedAt,
		StudentQuizAttemptLastFinishedAt: m.StudentQuizAttemptLastFinishedAt,

		StudentQuizAttemptCreatedAt: m.StudentQuizAttemptCreatedAt,
		StudentQuizAttemptUpdatedAt: m.StudentQuizAttemptUpdatedAt,
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

func JSONFromRaw(raw json.RawMessage) datatypes.JSON {
	return datatypes.JSON(raw)
}
