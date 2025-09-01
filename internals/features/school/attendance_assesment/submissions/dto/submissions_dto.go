// file: internals/features/school/submissions/dto/submission_dto.go
package dto

import (
	"encoding/json"
	"time"

	"masjidku_backend/internals/features/school/attendance_assesment/submissions/model"

	"github.com/google/uuid"
)

/*
PatchField adalah util 3-state untuk PATCH:
- field tidak dikirim  -> Present=false
- field dikirim nilai  -> Present=true,  Value != nil
- field dikirim null   -> Present=true,  Value == nil  (jadikan kolom NULL)

Contoh JSON:
{ "submissions_text": "abc" }         -> Present:true,  Value:"abc"
{ "submissions_text": null }          -> Present:true,  Value:nil
{ /* tidak ada key submissions_text */ /*} -> Present:false
 */
type PatchField[T any] struct {
	Present bool `json:"-"`
	Value   *T   `json:"-"`
}

// UnmarshalJSON menangkap 3-state di atas
func (p *PatchField[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	// null?
	if string(b) == "null" {
		p.Value = nil
		return nil
	}
	// value
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

// Helper: apakah ingin set kolom ke NULL?
func (p PatchField[T]) IsNull() bool {
	return p.Present && p.Value == nil
}

// Helper: apakah ingin update kolom (nilai atau null)
func (p PatchField[T]) ShouldUpdate() bool {
	return p.Present
}

/* =========================
   Create DTO
   ========================= */

type CreateSubmissionRequest struct {
	SubmissionMasjidID     uuid.UUID             `json:"submissions_masjid_id" validate:"required"`
	SubmissionAssessmentID uuid.UUID             `json:"submissions_assessment_id" validate:"required"`
	SubmissionUserID       uuid.UUID             `json:"submissions_user_id" validate:"required"`

	SubmissionText        *string                 `json:"submissions_text,omitempty"`
	SubmissionStatus      *model.SubmissionStatus `json:"submissions_status,omitempty" validate:"omitempty,oneof=draft submitted resubmitted graded returned"`
	SubmissionSubmittedAt *time.Time              `json:"submissions_submitted_at,omitempty"`
	SubmissionIsLate      *bool                   `json:"submissions_is_late,omitempty"`
}

/* =========================
   PATCH (Partial Update) DTO
   Semua kolom opsional & 3-state via PatchField
   ========================= */

type PatchSubmissionRequest struct {
	// isi & status
	SubmissionText        *PatchField[string]                 `json:"submissions_text,omitempty"`
	SubmissionStatus      *PatchField[model.SubmissionStatus] `json:"submissions_status,omitempty"` // validate di controller
	SubmissionSubmittedAt *PatchField[time.Time]              `json:"submissions_submitted_at,omitempty"`
	SubmissionIsLate      *PatchField[bool]                   `json:"submissions_is_late,omitempty"`

	// penilaian
	SubmissionScore    *PatchField[float64]  `json:"submissions_score,omitempty"` // 0..100 (cek di controller)
	SubmissionFeedback *PatchField[string]   `json:"submissions_feedback,omitempty"`
	SubmissionGradedBy *PatchField[uuid.UUID] `json:"submissions_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt *PatchField[time.Time] `json:"submissions_graded_at,omitempty"`
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
		case *PatchField[model.SubmissionStatus]:
			if f != nil && f.ShouldUpdate() {
				if f.IsNull() {
					upd[key] = nil
				} else {
					upd[key] = *f.Value
				}
			}
		}
	}

	// isi & status
	put("submissions_text", p.SubmissionText)
	put("submissions_status", p.SubmissionStatus)
	put("submissions_submitted_at", p.SubmissionSubmittedAt)
	put("submissions_is_late", p.SubmissionIsLate)

	// penilaian
	put("submissions_score", p.SubmissionScore)
	put("submissions_feedback", p.SubmissionFeedback)
	put("submissions_graded_by_teacher_id", p.SubmissionGradedBy)
	put("submissions_graded_at", p.SubmissionGradedAt)

	return upd
}

/* =========================
   (Opsional) DTO khusus grading
   Bisa dipakai endpoint terpisah /grade
   ========================= */
type GradeSubmissionRequest struct {
	SubmissionScore    *PatchField[float64]  `json:"submissions_score,omitempty"` // 0..100
	SubmissionFeedback *PatchField[string]   `json:"submissions_feedback,omitempty"`
	SubmissionGradedBy *PatchField[uuid.UUID] `json:"submissions_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt *PatchField[time.Time] `json:"submissions_graded_at,omitempty"`
}

func (g *GradeSubmissionRequest) ToUpdates() map[string]any {
	return (&PatchSubmissionRequest{
		SubmissionScore:    g.SubmissionScore,
		SubmissionFeedback: g.SubmissionFeedback,
		SubmissionGradedBy: g.SubmissionGradedBy,
		SubmissionGradedAt: g.SubmissionGradedAt,
	}).ToUpdates()
}

/* =========================
   Query DTO (filter & paging)
   ========================= */

type ListSubmissionsQuery struct {
	// filter
	MasjidID     *uuid.UUID              `query:"masjid_id"`
	AssessmentID *uuid.UUID              `query:"assessment_id"`
	UserID       *uuid.UUID              `query:"user_id"`
	Status       *model.SubmissionStatus `query:"status" validate:"omitempty,oneof=draft submitted resubmitted graded returned"`

	// periode waktu (opsional)
	SubmittedFrom *time.Time `query:"submitted_from"`
	SubmittedTo   *time.Time `query:"submitted_to"`

	// paginate
	Page    int `query:"page" validate:"omitempty,min=1"`
	PerPage int `query:"per_page" validate:"omitempty,min=1,max=200"`

	// sorting
	Sort string `query:"sort" validate:"omitempty,oneof=created_at desc_created_at submitted_at desc_submitted_at score desc_score"`
}

/* =========================
   Response DTO
   ========================= */

type SubmissionResponse struct {
	SubmissionID           uuid.UUID              `json:"submissions_id"`
	SubmissionMasjidID     uuid.UUID              `json:"submissions_masjid_id"`
	SubmissionAssessmentID uuid.UUID              `json:"submissions_assessment_id"`
	SubmissionUserID       uuid.UUID              `json:"submissions_user_id"`

	SubmissionText        *string                 `json:"submissions_text,omitempty"`
	SubmissionStatus      model.SubmissionStatus  `json:"submissions_status"`
	SubmissionSubmittedAt *time.Time              `json:"submissions_submitted_at,omitempty"`
	SubmissionIsLate      *bool                   `json:"submissions_is_late,omitempty"`

	SubmissionScore    *float64   `json:"submissions_score,omitempty"`
	SubmissionFeedback *string    `json:"submissions_feedback,omitempty"`
	SubmissionGradedBy *uuid.UUID `json:"submissions_graded_by_teacher_id,omitempty"`
	SubmissionGradedAt *time.Time `json:"submissions_graded_at,omitempty"`

	SubmissionCreatedAt time.Time `json:"submissions_created_at"`
	SubmissionUpdatedAt time.Time `json:"submissions_updated_at"`
}

func FromModel(m *model.Submission) SubmissionResponse {
	return SubmissionResponse{
		SubmissionID:           m.SubmissionID,
		SubmissionMasjidID:     m.SubmissionMasjidID,
		SubmissionAssessmentID: m.SubmissionAssessmentID,
		SubmissionUserID:       m.SubmissionUserID,

		SubmissionText:        m.SubmissionText,
		SubmissionStatus:      m.SubmissionStatus,
		SubmissionSubmittedAt: m.SubmissionSubmittedAt,
		SubmissionIsLate:      m.SubmissionIsLate,

		SubmissionScore:    m.SubmissionScore,
		SubmissionFeedback: m.SubmissionFeedback,
		SubmissionGradedBy: m.SubmissionGradedBy,
		SubmissionGradedAt: m.SubmissionGradedAt,

		SubmissionCreatedAt: m.SubmissionCreatedAt,
		SubmissionUpdatedAt: m.SubmissionUpdatedAt,
	}
}
