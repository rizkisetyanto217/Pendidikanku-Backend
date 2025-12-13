// file: internals/features/assessments/submissions/dto/submission_dto.go
package dto

import (
	"encoding/json"
	"time"

	subModel "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/model"
	"madinahsalam_backend/internals/helpers/dbtime"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/*
PatchField adalah util 3-state untuk PATCH:
- field tidak dikirim  -> Present=false
- field dikirim nilai  -> Present=true,  Value != nil
- field dikirim null   -> Present=true,  Value == nil
CATATAN:
  - untuk kolom NOT NULL (misal submission_is_late),
    controller HARUS menolak null sebelum masuk ToUpdates
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

//
// =========================================================
// CREATE DTO
// =========================================================
//

type CreateSubmissionRequest struct {
	SubmissionSchoolID     uuid.UUID `json:"submission_school_id" validate:"required,uuid4"`
	SubmissionAssessmentID uuid.UUID `json:"submission_assessment_id" validate:"required,uuid4"`
	SubmissionStudentID    uuid.UUID `json:"submission_student_id" validate:"required,uuid4"`

	SubmissionText *string `json:"submission_text,omitempty"`
}

/*
ToModel:
- attempt_count DISET oleh backend (hasil MAX + 1)
- status default = draft
- is_late default = false (dihitung saat submit)
*/
func (r CreateSubmissionRequest) ToModel(attemptCount int) subModel.SubmissionModel {
	return subModel.SubmissionModel{
		SubmissionSchoolID:     r.SubmissionSchoolID,
		SubmissionAssessmentID: r.SubmissionAssessmentID,
		SubmissionStudentID:    r.SubmissionStudentID,

		SubmissionAttemptCount: attemptCount,

		SubmissionText:   r.SubmissionText,
		SubmissionStatus: subModel.SubmissionStatusDraft,
		SubmissionIsLate: false,
	}
}

//
// =========================================================
// PATCH DTO (Partial Update)
// =========================================================
//

type PatchSubmissionRequest struct {
	// isi & status
	SubmissionText        *PatchField[string]                    `json:"submission_text,omitempty"`
	SubmissionStatus      *PatchField[subModel.SubmissionStatus] `json:"submission_status,omitempty"`
	SubmissionSubmittedAt *PatchField[time.Time]                 `json:"submission_submitted_at,omitempty"`

	// NOT NULL → tidak boleh null
	SubmissionIsLate *PatchField[bool] `json:"submission_is_late,omitempty"`

	// penilaian
	SubmissionScore        *PatchField[float64]        `json:"submission_score,omitempty"` // 0..100
	SubmissionFeedback     *PatchField[string]         `json:"submission_feedback,omitempty"`
	SubmissionScores       *PatchField[map[string]any] `json:"submission_scores,omitempty"`
	SubmissionQuizFinished *PatchField[int]            `json:"submission_quiz_finished,omitempty"`
	SubmissionGradedBy     *PatchField[uuid.UUID]      `json:"submission_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt     *PatchField[time.Time]      `json:"submission_graded_at,omitempty"`
}

/*
ToUpdates:
- field tidak dikirim -> di-skip
- field dikirim null -> set NULL (KECUALI is_late → controller harus blok)
- field dikirim nilai -> set value
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
				// is_late tidak boleh null → asumsi controller sudah validasi
				upd[key] = *f.Value
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
					upd[key] = *f.Value
				}
			}
		}
	}

	put("submission_text", p.SubmissionText)
	put("submission_status", p.SubmissionStatus)
	put("submission_submitted_at", p.SubmissionSubmittedAt)
	put("submission_is_late", p.SubmissionIsLate)

	put("submission_score", p.SubmissionScore)
	put("submission_feedback", p.SubmissionFeedback)
	put("submission_scores", p.SubmissionScores)
	put("submission_quiz_finished", p.SubmissionQuizFinished)
	put("submission_graded_by_teacher_id", p.SubmissionGradedBy)
	put("submission_graded_at", p.SubmissionGradedAt)

	return upd
}

//
// =========================================================
// QUERY DTO
// =========================================================
//

type ListSubmissionsQuery struct {
	SchoolID     *uuid.UUID                 `query:"school_id"`
	AssessmentID *uuid.UUID                 `query:"assessment_id"`
	StudentID    *uuid.UUID                 `query:"student_id"`
	Status       *subModel.SubmissionStatus `query:"status" validate:"omitempty,oneof=draft submitted resubmitted graded returned"`

	SubmittedFrom *time.Time `query:"submitted_from"`
	SubmittedTo   *time.Time `query:"submitted_to"`

	Page    int `query:"page" validate:"omitempty,min=1"`
	PerPage int `query:"per_page" validate:"omitempty,min=1,max=200"`

	Sort string `query:"sort" validate:"omitempty,oneof=created_at desc_created_at submitted_at desc_submitted_at score desc_score"`
}

// =========================================================
// RESPONSE DTO
// =========================================================
type SubmissionResponse struct {
	SubmissionID           uuid.UUID `json:"submission_id"`
	SubmissionSchoolID     uuid.UUID `json:"submission_school_id"`
	SubmissionAssessmentID uuid.UUID `json:"submission_assessment_id"`
	SubmissionStudentID    uuid.UUID `json:"submission_student_id"`

	SubmissionAttemptCount int `json:"submission_attempt_count"`

	SubmissionText        *string                   `json:"submission_text,omitempty"`
	SubmissionStatus      subModel.SubmissionStatus `json:"submission_status"`
	SubmissionSubmittedAt *time.Time                `json:"submission_submitted_at,omitempty"`
	SubmissionIsLate      bool                      `json:"submission_is_late"`

	SubmissionScore        *float64       `json:"submission_score,omitempty"`
	SubmissionScores       map[string]any `json:"submission_scores,omitempty"`
	SubmissionQuizFinished int            `json:"submission_quiz_finished"`
	SubmissionFeedback     *string        `json:"submission_feedback,omitempty"`

	SubmissionGradedByTeacherID *uuid.UUID `json:"submission_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt          *time.Time `json:"submission_graded_at,omitempty"`

	// ✅ include=submission_urls (compact)
	SubmissionURLs []SubmissionURLDocCompact `json:"submission_urls,omitempty"`

	SubmissionCreatedAt time.Time  `json:"submission_created_at"`
	SubmissionUpdatedAt time.Time  `json:"submission_updated_at"`
	SubmissionDeletedAt *time.Time `json:"submission_deleted_at,omitempty"`
}

func FromModel(m *subModel.SubmissionModel) SubmissionResponse {
	var del *time.Time
	if m.SubmissionDeletedAt.Valid {
		t := m.SubmissionDeletedAt.Time
		del = &t
	}

	var scores map[string]any
	if m.SubmissionScores != nil {
		scores = map[string]any(m.SubmissionScores)
	}

	return SubmissionResponse{
		SubmissionID:           m.SubmissionID,
		SubmissionSchoolID:     m.SubmissionSchoolID,
		SubmissionAssessmentID: m.SubmissionAssessmentID,
		SubmissionStudentID:    m.SubmissionStudentID,

		SubmissionAttemptCount: m.SubmissionAttemptCount,

		SubmissionText:        m.SubmissionText,
		SubmissionStatus:      m.SubmissionStatus,
		SubmissionSubmittedAt: m.SubmissionSubmittedAt,
		SubmissionIsLate:      m.SubmissionIsLate,

		SubmissionScore:        m.SubmissionScore,
		SubmissionScores:       scores,
		SubmissionQuizFinished: m.SubmissionQuizFinished,
		SubmissionFeedback:     m.SubmissionFeedback,

		SubmissionGradedByTeacherID: m.SubmissionGradedByTeacherID,
		SubmissionGradedAt:          m.SubmissionGradedAt,

		SubmissionCreatedAt: m.SubmissionCreatedAt,
		SubmissionUpdatedAt: m.SubmissionUpdatedAt,
		SubmissionDeletedAt: del,
	}
}

func FromModels(list []subModel.SubmissionModel) []SubmissionResponse {
	out := make([]SubmissionResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModel(&list[i]))
	}
	return out
}

// Timezone-aware
func FromModelWithCtx(c *fiber.Ctx, m *subModel.SubmissionModel) SubmissionResponse {
	resp := FromModel(m)

	resp.SubmissionSubmittedAt = dbtime.ToSchoolTimePtr(c, m.SubmissionSubmittedAt)
	resp.SubmissionGradedAt = dbtime.ToSchoolTimePtr(c, m.SubmissionGradedAt)

	resp.SubmissionCreatedAt = dbtime.ToSchoolTime(c, m.SubmissionCreatedAt)
	resp.SubmissionUpdatedAt = dbtime.ToSchoolTime(c, m.SubmissionUpdatedAt)
	resp.SubmissionDeletedAt = dbtime.ToSchoolTimePtr(c, resp.SubmissionDeletedAt)

	return resp
}

func FromModelsWithCtx(c *fiber.Ctx, list []subModel.SubmissionModel) []SubmissionResponse {
	out := make([]SubmissionResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelWithCtx(c, &list[i]))
	}
	return out
}
