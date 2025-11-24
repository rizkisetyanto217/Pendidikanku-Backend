// file: internals/features/assessments/submissions/dto/submission_dto.go
package dto

import (
	"encoding/json"
	"time"

	subModel "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/model"

	"github.com/google/uuid"
)

/*
PatchField adalah util 3-state untuk PATCH:
- field tidak dikirim  -> Present=false
- field dikirim nilai  -> Present=true,  Value != nil
- field dikirim null   -> Present=true,  Value == nil  (jadikan kolom NULL)
*/
type PatchField[T any] struct {
	Present bool `json:"-"`
	Value   *T   `json:"-"`
}

func (p *PatchField[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	if string(b) == "null" {
		p.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}
func (p PatchField[T]) IsNull() bool       { return p.Present && p.Value == nil }
func (p PatchField[T]) ShouldUpdate() bool { return p.Present }

/* =========================
   Create DTO
   ========================= */

type CreateSubmissionRequest struct {
	SubmissionSchoolID     uuid.UUID `json:"submission_school_id" validate:"required"`
	SubmissionAssessmentID uuid.UUID `json:"submission_assessment_id" validate:"required"`
	SubmissionStudentID    uuid.UUID `json:"submission_student_id" validate:"required"`

	SubmissionText        *string                    `json:"submission_text,omitempty"`
	SubmissionStatus      *subModel.SubmissionStatus `json:"submission_status,omitempty" validate:"omitempty,oneof=draft submitted resubmitted graded returned"`
	SubmissionSubmittedAt *time.Time                 `json:"submission_submitted_at,omitempty"`
	SubmissionIsLate      *bool                      `json:"submission_is_late,omitempty"`
}

func (r CreateSubmissionRequest) ToModel() subModel.Submission {
	status := subModel.SubmissionStatusSubmitted
	if r.SubmissionStatus != nil {
		status = *r.SubmissionStatus
	}
	return subModel.Submission{
		SubmissionSchoolID:     r.SubmissionSchoolID,
		SubmissionAssessmentID: r.SubmissionAssessmentID,
		SubmissionStudentID:    r.SubmissionStudentID,

		SubmissionText:        r.SubmissionText,
		SubmissionStatus:      status,
		SubmissionSubmittedAt: r.SubmissionSubmittedAt,
		SubmissionIsLate:      r.SubmissionIsLate,
		// submission_scores      → nil (default)
		// submission_quiz_finished → 0 (default)
		// created_at/updated_at dikelola DB (default now())
	}
}

/* =========================
   PATCH (Partial Update) DTO
   ========================= */

type PatchSubmissionRequest struct {
	// isi & status
	SubmissionText        *PatchField[string]                    `json:"submission_text,omitempty"`
	SubmissionStatus      *PatchField[subModel.SubmissionStatus] `json:"submission_status,omitempty"`
	SubmissionSubmittedAt *PatchField[time.Time]                 `json:"submission_submitted_at,omitempty"`
	SubmissionIsLate      *PatchField[bool]                      `json:"submission_is_late,omitempty"`

	// penilaian (final & breakdown)
	SubmissionScore        *PatchField[float64]        `json:"submission_score,omitempty"` // 0..100 (cek di controller)
	SubmissionFeedback     *PatchField[string]         `json:"submission_feedback,omitempty"`
	SubmissionScores       *PatchField[map[string]any] `json:"submission_scores,omitempty"`        // JSON breakdown
	SubmissionQuizFinished *PatchField[int]            `json:"submission_quiz_finished,omitempty"` // progress quiz selesai
	SubmissionGradedBy     *PatchField[uuid.UUID]      `json:"submission_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt     *PatchField[time.Time]      `json:"submission_graded_at,omitempty"`
}

/*
ToUpdates menghasilkan map[string]any untuk GORM Updates().
- Field yang tidak dikirim -> tidak dimasukkan
- Field yang dikirim null  -> dimasukkan dengan value = nil (set kolom ke NULL)
- Field yang dikirim nilai -> dimasukkan dengan nilai tsb
*/
func (p *PatchSubmissionRequest) ToUpdates() map[string]any {
	upd := map[string]any{}

	put := func(key string, pf any) {
		switch f := pf.(type) {
		case *PatchField[string]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					upd[key] = *f.Value
				}
			}
		case *PatchField[bool]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					upd[key] = *f.Value
				}
			}
		case *PatchField[float64]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					upd[key] = *f.Value
				}
			}
		case *PatchField[int]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					upd[key] = *f.Value
				}
			}
		case *PatchField[uuid.UUID]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					upd[key] = *f.Value
				}
			}
		case *PatchField[time.Time]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					upd[key] = *f.Value
				}
			}
		case *PatchField[subModel.SubmissionStatus]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					upd[key] = *f.Value
				}
			}
		case *PatchField[map[string]any]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					// langsung simpan map → akan di-marshal sebagai JSONB oleh driver
					upd[key] = *f.Value
				}
			}
		}
	}

	// isi & status
	put("submission_text", p.SubmissionText)
	put("submission_status", p.SubmissionStatus)
	put("submission_submitted_at", p.SubmissionSubmittedAt)
	put("submission_is_late", p.SubmissionIsLate)

	// penilaian
	put("submission_score", p.SubmissionScore)
	put("submission_feedback", p.SubmissionFeedback)
	put("submission_scores", p.SubmissionScores)
	put("submission_quiz_finished", p.SubmissionQuizFinished)
	put("submission_graded_by_teacher_id", p.SubmissionGradedBy)
	put("submission_graded_at", p.SubmissionGradedAt)

	return upd
}

/* =========================
   (Opsional) DTO khusus grading
   ========================= */

type GradeSubmissionRequest struct {
	SubmissionScore        *PatchField[float64]        `json:"submission_score,omitempty"` // 0..100
	SubmissionFeedback     *PatchField[string]         `json:"submission_feedback,omitempty"`
	SubmissionScores       *PatchField[map[string]any] `json:"submission_scores,omitempty"`        // breakdown komponen
	SubmissionQuizFinished *PatchField[int]            `json:"submission_quiz_finished,omitempty"` // berapa quiz/komponen yang sudah selesai
	SubmissionGradedBy     *PatchField[uuid.UUID]      `json:"submission_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt     *PatchField[time.Time]      `json:"submission_graded_at,omitempty"`
}

func (g *GradeSubmissionRequest) ToUpdates() map[string]any {
	return (&PatchSubmissionRequest{
		SubmissionScore:        g.SubmissionScore,
		SubmissionFeedback:     g.SubmissionFeedback,
		SubmissionScores:       g.SubmissionScores,
		SubmissionQuizFinished: g.SubmissionQuizFinished,
		SubmissionGradedBy:     g.SubmissionGradedBy,
		SubmissionGradedAt:     g.SubmissionGradedAt,
	}).ToUpdates()
}

/* =========================
   Query DTO (filter & paging)
   ========================= */

type ListSubmissionsQuery struct {
	// filter
	SchoolID     *uuid.UUID                 `query:"school_id"`
	AssessmentID *uuid.UUID                 `query:"assessment_id"`
	StudentID    *uuid.UUID                 `query:"student_id"`
	Status       *subModel.SubmissionStatus `query:"status" validate:"omitempty,oneof=draft submitted resubmitted graded returned"`

	// periode waktu (opsional)
	SubmittedFrom *time.Time `query:"submitted_from"`
	SubmittedTo   *time.Time `query:"submitted_to"`

	// paginate
	Page    int `query:"page" validate:"omitempty,min=1"`
	PerPage int `query:"per_page" validate:"omitempty,min=1,max=200"`

	// sorting
	// created_at | desc_created_at | submitted_at | desc_submitted_at | score | desc_score
	Sort string `query:"sort" validate:"omitempty,oneof=created_at desc_created_at submitted_at desc_submitted_at score desc_score"`
}

/* =========================
   Response DTO
   ========================= */

type SubmissionResponse struct {
	SubmissionID           uuid.UUID `json:"submission_id"`
	SubmissionSchoolID     uuid.UUID `json:"submission_school_id"`
	SubmissionAssessmentID uuid.UUID `json:"submission_assessment_id"`
	SubmissionStudentID    uuid.UUID `json:"submission_student_id"`

	SubmissionText        *string                   `json:"submission_text,omitempty"`
	SubmissionStatus      subModel.SubmissionStatus `json:"submission_status"`
	SubmissionSubmittedAt *time.Time                `json:"submission_submitted_at,omitempty"`
	SubmissionIsLate      *bool                     `json:"submission_is_late,omitempty"`

	SubmissionScore             *float64       `json:"submission_score,omitempty"`
	SubmissionScores            map[string]any `json:"submission_scores,omitempty"`
	SubmissionQuizFinished      int            `json:"submission_quiz_finished"`
	SubmissionFeedback          *string        `json:"submission_feedback,omitempty"`
	SubmissionGradedByTeacherID *uuid.UUID     `json:"submission_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt          *time.Time     `json:"submission_graded_at,omitempty"`

	SubmissionCreatedAt time.Time  `json:"submission_created_at"`
	SubmissionUpdatedAt time.Time  `json:"submission_updated_at"`
	SubmissionDeletedAt *time.Time `json:"submission_deleted_at,omitempty"`
}

func FromModel(m *subModel.Submission) SubmissionResponse {
	var del *time.Time
	if m.SubmissionDeletedAt.Valid {
		t := m.SubmissionDeletedAt.Time
		del = &t
	}

	// konversi datatypes.JSONMap → map[string]any
	var scores map[string]any
	if m.SubmissionScores != nil {
		scores = map[string]any(m.SubmissionScores)
	}

	return SubmissionResponse{
		SubmissionID:           m.SubmissionID,
		SubmissionSchoolID:     m.SubmissionSchoolID,
		SubmissionAssessmentID: m.SubmissionAssessmentID,
		SubmissionStudentID:    m.SubmissionStudentID,

		SubmissionText:        m.SubmissionText,
		SubmissionStatus:      m.SubmissionStatus,
		SubmissionSubmittedAt: m.SubmissionSubmittedAt,
		SubmissionIsLate:      m.SubmissionIsLate,

		SubmissionScore:             m.SubmissionScore,
		SubmissionScores:            scores,
		SubmissionQuizFinished:      m.SubmissionQuizFinished,
		SubmissionFeedback:          m.SubmissionFeedback,
		SubmissionGradedByTeacherID: m.SubmissionGradedByTeacherID,
		SubmissionGradedAt:          m.SubmissionGradedAt,

		SubmissionCreatedAt: m.SubmissionCreatedAt,
		SubmissionUpdatedAt: m.SubmissionUpdatedAt,
		SubmissionDeletedAt: del,
	}
}

func FromModels(list []subModel.Submission) []SubmissionResponse {
	out := make([]SubmissionResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModel(&list[i]))
	}
	return out
}
