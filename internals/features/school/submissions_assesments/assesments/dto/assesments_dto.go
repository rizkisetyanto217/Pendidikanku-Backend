// file: internals/features/school/submissions_assesments/assesments/dto/assessment_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	assessModel "schoolku_backend/internals/features/school/submissions_assesments/assesments/model"
)

/* ========================================================
   REQUEST DTO
======================================================== */

type CreateAssessmentRequest struct {
	// Diisi di controller (enforce tenant)
	AssessmentSchoolID uuid.UUID `json:"assessment_school_id" validate:"-"`

	// Relasi utama
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id" validate:"omitempty,uuid4"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id" validate:"omitempty,uuid4"`

	// Identitas
	AssessmentSlug        *string `json:"assessment_slug" validate:"omitempty,max=160"`
	AssessmentTitle       string  `json:"assessment_title" validate:"omitempty,max=180"`
	AssessmentDescription *string `json:"assessment_description"`

	// Jadwal (mode date)
	AssessmentStartAt     *time.Time `json:"assessment_start_at"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at"`

	// Pengaturan
	AssessmentKind                 string  `json:"assessment_kind" validate:"omitempty,oneof=quiz assignment_upload offline survey"`
	AssessmentDurationMinutes      *int    `json:"assessment_duration_minutes" validate:"omitempty,gte=1,lte=1440"`
	AssessmentTotalAttemptsAllowed int     `json:"assessment_total_attempts_allowed" validate:"omitempty,gte=1,lte=50"`
	AssessmentMaxScore             float64 `json:"assessment_max_score" validate:"omitempty,gte=0,lte=100"`
	AssessmentIsPublished          *bool   `json:"assessment_is_published"`
	AssessmentAllowSubmission      *bool   `json:"assessment_allow_submission"`

	// Audit
	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id" validate:"omitempty,uuid4"`

	// Mode session (opsional)
	AssessmentAnnounceSessionID *uuid.UUID `json:"assessment_announce_session_id" validate:"omitempty,uuid4"`
	AssessmentCollectSessionID  *uuid.UUID `json:"assessment_collect_session_id" validate:"omitempty,uuid4"`
}

/*
Normalize:
  - trim string
  - lowercase kind
  - set default kind/attempts/max_score/is_published/allow_submission
*/
func (r *CreateAssessmentRequest) Normalize() {
	trimPtr := func(s *string) *string {
		if s == nil {
			return nil
		}
		t := strings.TrimSpace(*s)
		return &t
	}

	r.AssessmentSlug = trimPtr(r.AssessmentSlug)
	r.AssessmentDescription = trimPtr(r.AssessmentDescription)
	r.AssessmentTitle = strings.TrimSpace(r.AssessmentTitle)

	if r.AssessmentKind != "" {
		r.AssessmentKind = strings.ToLower(strings.TrimSpace(r.AssessmentKind))
	}
	if r.AssessmentKind == "" {
		r.AssessmentKind = "quiz"
	}

	// default attempts
	if r.AssessmentTotalAttemptsAllowed <= 0 {
		r.AssessmentTotalAttemptsAllowed = 1
	}

	// default max score
	if r.AssessmentMaxScore <= 0 {
		r.AssessmentMaxScore = 100
	}

	// default flags
	if r.AssessmentIsPublished == nil {
		b := true
		r.AssessmentIsPublished = &b
	}
	if r.AssessmentAllowSubmission == nil {
		b := true
		r.AssessmentAllowSubmission = &b
	}
}

// Convert Create DTO → Model
func (r *CreateAssessmentRequest) ToModel() assessModel.AssessmentModel {
	kind := assessModel.AssessmentKind(r.AssessmentKind)
	if r.AssessmentKind == "" {
		kind = assessModel.AssessmentKindQuiz
	}

	row := assessModel.AssessmentModel{
		AssessmentSchoolID:                     r.AssessmentSchoolID,
		AssessmentClassSectionSubjectTeacherID: r.AssessmentClassSectionSubjectTeacherID,
		AssessmentTypeID:                       r.AssessmentTypeID,
		AssessmentSlug:                         r.AssessmentSlug,
		AssessmentTitle:                        r.AssessmentTitle,
		AssessmentDescription:                  r.AssessmentDescription,
		AssessmentStartAt:                      r.AssessmentStartAt,
		AssessmentDueAt:                        r.AssessmentDueAt,
		AssessmentPublishedAt:                  r.AssessmentPublishedAt,
		AssessmentClosedAt:                     r.AssessmentClosedAt,
		AssessmentKind:                         kind,
		AssessmentDurationMinutes:              r.AssessmentDurationMinutes,
		AssessmentTotalAttemptsAllowed:         r.AssessmentTotalAttemptsAllowed,
		AssessmentMaxScore:                     r.AssessmentMaxScore,
		AssessmentCreatedByTeacherID:           r.AssessmentCreatedByTeacherID,
		AssessmentSubmissionMode:               assessModel.SubmissionModeDate, // akan dioverride di controller
		AssessmentIsPublished:                  *r.AssessmentIsPublished,
		AssessmentAllowSubmission:              *r.AssessmentAllowSubmission,
	}

	return row
}

/* ========================================================
   PATCH DTO
======================================================== */

type PatchAssessmentRequest struct {
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id" validate:"omitempty,uuid4"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id" validate:"omitempty,uuid4"`

	AssessmentSlug        *string `json:"assessment_slug" validate:"omitempty,max=160"`
	AssessmentTitle       *string `json:"assessment_title" validate:"omitempty,max=180"`
	AssessmentDescription *string `json:"assessment_description"`

	AssessmentStartAt     *time.Time `json:"assessment_start_at"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at"`

	AssessmentKind                 *string  `json:"assessment_kind" validate:"omitempty,oneof=quiz assignment_upload offline survey"`
	AssessmentDurationMinutes      *int     `json:"assessment_duration_minutes" validate:"omitempty,gte=1,lte=1440"`
	AssessmentTotalAttemptsAllowed *int     `json:"assessment_total_attempts_allowed" validate:"omitempty,gte=1,lte=50"`
	AssessmentMaxScore             *float64 `json:"assessment_max_score" validate:"omitempty,gte=0,lte=100"`
	AssessmentIsPublished          *bool    `json:"assessment_is_published"`
	AssessmentAllowSubmission      *bool    `json:"assessment_allow_submission"`

	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id" validate:"omitempty,uuid4"`

	AssessmentAnnounceSessionID *uuid.UUID `json:"assessment_announce_session_id" validate:"omitempty,uuid4"`
	AssessmentCollectSessionID  *uuid.UUID `json:"assessment_collect_session_id" validate:"omitempty,uuid4"`
}

func (r *PatchAssessmentRequest) Normalize() {
	trimPtr := func(s *string) *string {
		if s == nil {
			return nil
		}
		t := strings.TrimSpace(*s)
		return &t
	}

	r.AssessmentSlug = trimPtr(r.AssessmentSlug)
	r.AssessmentTitle = trimPtr(r.AssessmentTitle)
	r.AssessmentDescription = trimPtr(r.AssessmentDescription)

	if r.AssessmentKind != nil {
		k := strings.ToLower(strings.TrimSpace(*r.AssessmentKind))
		r.AssessmentKind = &k
	}
}

// Apply PATCH ke model existing
func (r *PatchAssessmentRequest) Apply(m *assessModel.AssessmentModel) {
	if r.AssessmentClassSectionSubjectTeacherID != nil {
		m.AssessmentClassSectionSubjectTeacherID = r.AssessmentClassSectionSubjectTeacherID
	}
	if r.AssessmentTypeID != nil {
		m.AssessmentTypeID = r.AssessmentTypeID
	}

	if r.AssessmentSlug != nil {
		m.AssessmentSlug = r.AssessmentSlug
	}
	if r.AssessmentTitle != nil {
		m.AssessmentTitle = strings.TrimSpace(*r.AssessmentTitle)
	}
	if r.AssessmentDescription != nil {
		m.AssessmentDescription = r.AssessmentDescription
	}

	if r.AssessmentStartAt != nil {
		m.AssessmentStartAt = r.AssessmentStartAt
	}
	if r.AssessmentDueAt != nil {
		m.AssessmentDueAt = r.AssessmentDueAt
	}
	if r.AssessmentPublishedAt != nil {
		m.AssessmentPublishedAt = r.AssessmentPublishedAt
	}
	if r.AssessmentClosedAt != nil {
		m.AssessmentClosedAt = r.AssessmentClosedAt
	}

	if r.AssessmentKind != nil && *r.AssessmentKind != "" {
		m.AssessmentKind = assessModel.AssessmentKind(*r.AssessmentKind)
	}
	if r.AssessmentDurationMinutes != nil {
		m.AssessmentDurationMinutes = r.AssessmentDurationMinutes
	}
	if r.AssessmentTotalAttemptsAllowed != nil {
		m.AssessmentTotalAttemptsAllowed = *r.AssessmentTotalAttemptsAllowed
	}
	if r.AssessmentMaxScore != nil {
		m.AssessmentMaxScore = *r.AssessmentMaxScore
	}
	if r.AssessmentIsPublished != nil {
		m.AssessmentIsPublished = *r.AssessmentIsPublished
	}
	if r.AssessmentAllowSubmission != nil {
		m.AssessmentAllowSubmission = *r.AssessmentAllowSubmission
	}

	if r.AssessmentCreatedByTeacherID != nil {
		m.AssessmentCreatedByTeacherID = r.AssessmentCreatedByTeacherID
	}

	// Session IDs sendiri di-handle di controller (finalAnnID/finalColID),
	// jadi di sini kita nggak sentuh, supaya logikanya tetap terkonsolidasi di controller.
}

/* ========================================================
   RESPONSE DTO
======================================================== */

type AssessmentResponse struct {
	AssessmentID       uuid.UUID `json:"assessment_id"`
	AssessmentSchoolID uuid.UUID `json:"assessment_school_id"`

	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id,omitempty"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id,omitempty"`

	AssessmentSlug        *string `json:"assessment_slug,omitempty"`
	AssessmentTitle       string  `json:"assessment_title"`
	AssessmentDescription *string `json:"assessment_description,omitempty"`

	AssessmentStartAt     *time.Time `json:"assessment_start_at,omitempty"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at,omitempty"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at,omitempty"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at,omitempty"`

	AssessmentKind                 string  `json:"assessment_kind"`
	AssessmentDurationMinutes      *int    `json:"assessment_duration_minutes,omitempty"`
	AssessmentTotalAttemptsAllowed int     `json:"assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64 `json:"assessment_max_score"`
	AssessmentIsPublished          bool    `json:"assessment_is_published"`
	AssessmentAllowSubmission      bool    `json:"assessment_allow_submission"`

	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id,omitempty"`

	AssessmentSubmissionMode    string     `json:"assessment_submission_mode"`
	AssessmentAnnounceSessionID *uuid.UUID `json:"assessment_announce_session_id,omitempty"`
	AssessmentCollectSessionID  *uuid.UUID `json:"assessment_collect_session_id,omitempty"`

	AssessmentCSSTSnapshot            map[string]any `json:"assessment_csst_snapshot,omitempty"`
	AssessmentAnnounceSessionSnapshot map[string]any `json:"assessment_announce_session_snapshot,omitempty"`
	AssessmentCollectSessionSnapshot  map[string]any `json:"assessment_collect_session_snapshot,omitempty"`

	AssessmentCreatedAt time.Time `json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time `json:"assessment_updated_at"`
}

// Converter Model → Response DTO
func FromModelAssesment(m assessModel.AssessmentModel) AssessmentResponse {
	// cast datatypes.JSONMap → map[string]any
	toMap := func(j any) map[string]any {
		if j == nil {
			return map[string]any{}
		}
		if mm, ok := j.(map[string]any); ok {
			return mm
		}
		// datatypes.JSONMap underlying type memang map[string]any
		return map[string]any{}
	}

	return AssessmentResponse{
		AssessmentID:       m.AssessmentID,
		AssessmentSchoolID: m.AssessmentSchoolID,

		AssessmentClassSectionSubjectTeacherID: m.AssessmentClassSectionSubjectTeacherID,
		AssessmentTypeID:                       m.AssessmentTypeID,

		AssessmentSlug:        m.AssessmentSlug,
		AssessmentTitle:       m.AssessmentTitle,
		AssessmentDescription: m.AssessmentDescription,

		AssessmentStartAt:     m.AssessmentStartAt,
		AssessmentDueAt:       m.AssessmentDueAt,
		AssessmentPublishedAt: m.AssessmentPublishedAt,
		AssessmentClosedAt:    m.AssessmentClosedAt,

		AssessmentKind:                 string(m.AssessmentKind),
		AssessmentDurationMinutes:      m.AssessmentDurationMinutes,
		AssessmentTotalAttemptsAllowed: m.AssessmentTotalAttemptsAllowed,
		AssessmentMaxScore:             m.AssessmentMaxScore,
		AssessmentIsPublished:          m.AssessmentIsPublished,
		AssessmentAllowSubmission:      m.AssessmentAllowSubmission,

		AssessmentCreatedByTeacherID: m.AssessmentCreatedByTeacherID,

		AssessmentSubmissionMode:    string(m.AssessmentSubmissionMode),
		AssessmentAnnounceSessionID: m.AssessmentAnnounceSessionID,
		AssessmentCollectSessionID:  m.AssessmentCollectSessionID,

		AssessmentCSSTSnapshot:            toMap(m.AssessmentCSSTSnapshot),
		AssessmentAnnounceSessionSnapshot: toMap(m.AssessmentAnnounceSessionSnapshot),
		AssessmentCollectSessionSnapshot:  toMap(m.AssessmentCollectSessionSnapshot),

		AssessmentCreatedAt: m.AssessmentCreatedAt,
		AssessmentUpdatedAt: m.AssessmentUpdatedAt,
	}
}
