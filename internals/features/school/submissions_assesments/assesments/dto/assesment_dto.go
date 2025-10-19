// file: internals/features/school/assessments/dto/assessment_dto.go
package dto

import (
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"

	"github.com/google/uuid"
)

/* ==============================
   CREATE (POST /assessments)
============================== */

type CreateAssessmentRequest struct {
	// Tenant (diisi dari context di controller)
	AssessmentMasjidID uuid.UUID `json:"assessment_masjid_id" validate:"required"`

	// Relasi
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id" validate:"omitempty,uuid"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id" validate:"omitempty,uuid"`

	// Identitas
	AssessmentSlug        *string `json:"assessment_slug" validate:"omitempty,max=160"`
	AssessmentTitle       string  `json:"assessment_title" validate:"required,max=180"`
	AssessmentDescription *string `json:"assessment_description" validate:"omitempty"`

	// Jadwal (mode 'date')
	AssessmentStartAt     *time.Time `json:"assessment_start_at" validate:"omitempty"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at" validate:"omitempty"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at" validate:"omitempty"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at" validate:"omitempty"`

	// Pengaturan
	AssessmentDurationMinutes      *int     `json:"assessment_duration_minutes" validate:"omitempty,min=1,max=1440"`
	AssessmentTotalAttemptsAllowed *int     `json:"assessment_total_attempts_allowed" validate:"omitempty,min=1,max=50"`
	AssessmentMaxScore             *float64 `json:"assessment_max_score" validate:"omitempty,gte=0,lte=100"`
	AssessmentIsPublished          *bool    `json:"assessment_is_published" validate:"omitempty"`
	AssessmentAllowSubmission      *bool    `json:"assessment_allow_submission" validate:"omitempty"`

	// Audit pembuat
	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id" validate:"omitempty,uuid"`

	// Mode berbasis sesi (opsional)
	AssessmentSubmissionMode    *model.AssessmentSubmissionMode `json:"assessment_submission_mode" validate:"omitempty,oneof=date session"`
	AssessmentAnnounceSessionID *uuid.UUID                      `json:"assessment_announce_session_id" validate:"omitempty,uuid"`
	AssessmentCollectSessionID  *uuid.UUID                      `json:"assessment_collect_session_id" validate:"omitempty,uuid"`
}

func (r *CreateAssessmentRequest) Normalize() {
	if r.AssessmentSlug != nil {
		s := strings.TrimSpace(*r.AssessmentSlug)
		if s == "" {
			r.AssessmentSlug = nil
		} else {
			r.AssessmentSlug = &s
		}
	}
	r.AssessmentTitle = strings.TrimSpace(r.AssessmentTitle)
	if r.AssessmentDescription != nil {
		d := strings.TrimSpace(*r.AssessmentDescription)
		if d == "" {
			r.AssessmentDescription = nil
		} else {
			r.AssessmentDescription = &d
		}
	}
}

func (r CreateAssessmentRequest) ToModel() model.AssessmentModel {
	// Defaults boolean
	isPublished := true
	if r.AssessmentIsPublished != nil {
		isPublished = *r.AssessmentIsPublished
	}
	allowSubmission := true
	if r.AssessmentAllowSubmission != nil {
		allowSubmission = *r.AssessmentAllowSubmission
	}

	// Defaults numeric
	maxScore := 100.0
	if r.AssessmentMaxScore != nil {
		maxScore = *r.AssessmentMaxScore
	}
	totalAttempts := 1
	if r.AssessmentTotalAttemptsAllowed != nil {
		totalAttempts = *r.AssessmentTotalAttemptsAllowed
	}

	// Default submission mode = "date"
	mode := model.SubmissionModeDate
	if r.AssessmentSubmissionMode != nil && strings.TrimSpace(string(*r.AssessmentSubmissionMode)) != "" {
		mode = *r.AssessmentSubmissionMode
	}

	return model.AssessmentModel{
		AssessmentMasjidID:                     r.AssessmentMasjidID,
		AssessmentClassSectionSubjectTeacherID: r.AssessmentClassSectionSubjectTeacherID,
		AssessmentTypeID:                       r.AssessmentTypeID,

		AssessmentSlug:        r.AssessmentSlug,
		AssessmentTitle:       r.AssessmentTitle,
		AssessmentDescription: r.AssessmentDescription,

		AssessmentStartAt:     r.AssessmentStartAt,
		AssessmentDueAt:       r.AssessmentDueAt,
		AssessmentPublishedAt: r.AssessmentPublishedAt,
		AssessmentClosedAt:    r.AssessmentClosedAt,

		AssessmentDurationMinutes:      r.AssessmentDurationMinutes,
		AssessmentTotalAttemptsAllowed: totalAttempts,
		AssessmentMaxScore:             maxScore,
		AssessmentIsPublished:          isPublished,
		AssessmentAllowSubmission:      allowSubmission,

		AssessmentCreatedByTeacherID: r.AssessmentCreatedByTeacherID,

		// Mode session (kalau diisi di controller akan juga di-set)
		AssessmentSubmissionMode:    mode,
		AssessmentAnnounceSessionID: r.AssessmentAnnounceSessionID,
		AssessmentCollectSessionID:  r.AssessmentCollectSessionID,
	}
}

/* ==============================
   PATCH (PATCH /assessments/:id)
============================== */

type PatchAssessmentRequest struct {
	// Identitas
	AssessmentSlug        *string `json:"assessment_slug" validate:"omitempty,max=160"`
	AssessmentTitle       *string `json:"assessment_title" validate:"omitempty,max=180"`
	AssessmentDescription *string `json:"assessment_description" validate:"omitempty"`

	// Jadwal
	AssessmentStartAt     *time.Time `json:"assessment_start_at" validate:"omitempty"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at" validate:"omitempty"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at" validate:"omitempty"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at" validate:"omitempty"`

	// Pengaturan
	AssessmentDurationMinutes      *int     `json:"assessment_duration_minutes" validate:"omitempty,min=1,max=1440"`
	AssessmentTotalAttemptsAllowed *int     `json:"assessment_total_attempts_allowed" validate:"omitempty,min=1,max=50"`
	AssessmentMaxScore             *float64 `json:"assessment_max_score" validate:"omitempty,gte=0,lte=100"`
	AssessmentIsPublished          *bool    `json:"assessment_is_published" validate:"omitempty"`
	AssessmentAllowSubmission      *bool    `json:"assessment_allow_submission" validate:"omitempty"`

	// Relasi
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id" validate:"omitempty,uuid"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id" validate:"omitempty,uuid"`

	// Audit pembuat
	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id" validate:"omitempty,uuid"`

	// Mode session
	AssessmentSubmissionMode    *model.AssessmentSubmissionMode `json:"assessment_submission_mode" validate:"omitempty,oneof=date session"`
	AssessmentAnnounceSessionID *uuid.UUID                      `json:"assessment_announce_session_id" validate:"omitempty,uuid"`
	AssessmentCollectSessionID  *uuid.UUID                      `json:"assessment_collect_session_id" validate:"omitempty,uuid"`
}

func (p *PatchAssessmentRequest) Normalize() {
	if p.AssessmentSlug != nil {
		s := strings.TrimSpace(*p.AssessmentSlug)
		if s == "" {
			p.AssessmentSlug = nil
		} else {
			p.AssessmentSlug = &s
		}
	}
	if p.AssessmentTitle != nil {
		t := strings.TrimSpace(*p.AssessmentTitle)
		if t == "" {
			p.AssessmentTitle = nil
		} else {
			p.AssessmentTitle = &t
		}
	}
	if p.AssessmentDescription != nil {
		d := strings.TrimSpace(*p.AssessmentDescription)
		if d == "" {
			p.AssessmentDescription = nil
		} else {
			p.AssessmentDescription = &d
		}
	}
}

// Apply menerapkan PATCH ke model (tanpa menyimpan ke DB)
func (p PatchAssessmentRequest) Apply(m *model.AssessmentModel) {
	if p.AssessmentSlug != nil {
		m.AssessmentSlug = p.AssessmentSlug
	}
	if p.AssessmentTitle != nil {
		m.AssessmentTitle = strings.TrimSpace(*p.AssessmentTitle)
	}
	if p.AssessmentDescription != nil {
		m.AssessmentDescription = p.AssessmentDescription
	}

	if p.AssessmentStartAt != nil {
		m.AssessmentStartAt = p.AssessmentStartAt
	}
	if p.AssessmentDueAt != nil {
		m.AssessmentDueAt = p.AssessmentDueAt
	}
	if p.AssessmentPublishedAt != nil {
		m.AssessmentPublishedAt = p.AssessmentPublishedAt
	}
	if p.AssessmentClosedAt != nil {
		m.AssessmentClosedAt = p.AssessmentClosedAt
	}

	if p.AssessmentDurationMinutes != nil {
		m.AssessmentDurationMinutes = p.AssessmentDurationMinutes
	}
	if p.AssessmentTotalAttemptsAllowed != nil {
		m.AssessmentTotalAttemptsAllowed = *p.AssessmentTotalAttemptsAllowed
	}
	if p.AssessmentMaxScore != nil {
		m.AssessmentMaxScore = *p.AssessmentMaxScore
	}
	if p.AssessmentIsPublished != nil {
		m.AssessmentIsPublished = *p.AssessmentIsPublished
	}
	if p.AssessmentAllowSubmission != nil {
		m.AssessmentAllowSubmission = *p.AssessmentAllowSubmission
	}

	if p.AssessmentClassSectionSubjectTeacherID != nil {
		m.AssessmentClassSectionSubjectTeacherID = p.AssessmentClassSectionSubjectTeacherID
	}
	if p.AssessmentTypeID != nil {
		m.AssessmentTypeID = p.AssessmentTypeID
	}
	if p.AssessmentCreatedByTeacherID != nil {
		m.AssessmentCreatedByTeacherID = p.AssessmentCreatedByTeacherID
	}

	// Mode session & sesi terkait
	if p.AssessmentSubmissionMode != nil {
		m.AssessmentSubmissionMode = *p.AssessmentSubmissionMode
	}
	if p.AssessmentAnnounceSessionID != nil {
		m.AssessmentAnnounceSessionID = p.AssessmentAnnounceSessionID
	}
	if p.AssessmentCollectSessionID != nil {
		m.AssessmentCollectSessionID = p.AssessmentCollectSessionID
	}
}

/* ==============================
   RESPONSES
============================== */

type AssessmentResponse struct {
	AssessmentID                           uuid.UUID  `json:"assessment_id"`
	AssessmentMasjidID                     uuid.UUID  `json:"assessment_masjid_id"`
	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id,omitempty"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id,omitempty"`

	AssessmentSlug        *string `json:"assessment_slug,omitempty"`
	AssessmentTitle       string  `json:"assessment_title"`
	AssessmentDescription *string `json:"assessment_description,omitempty"`

	AssessmentStartAt     *time.Time `json:"assessment_start_at,omitempty"`
	AssessmentDueAt       *time.Time `json:"assessment_due_at,omitempty"`
	AssessmentPublishedAt *time.Time `json:"assessment_published_at,omitempty"`
	AssessmentClosedAt    *time.Time `json:"assessment_closed_at,omitempty"`

	AssessmentDurationMinutes      *int    `json:"assessment_duration_minutes,omitempty"`
	AssessmentTotalAttemptsAllowed int     `json:"assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64 `json:"assessment_max_score"`
	AssessmentIsPublished          bool    `json:"assessment_is_published"`
	AssessmentAllowSubmission      bool    `json:"assessment_allow_submission"`

	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id,omitempty"`

	// Mode session
	AssessmentSubmissionMode    string     `json:"assessment_submission_mode"`
	AssessmentAnnounceSessionID *uuid.UUID `json:"assessment_announce_session_id,omitempty"`
	AssessmentCollectSessionID  *uuid.UUID `json:"assessment_collect_session_id,omitempty"`

	// 🔹 Snapshots (read-only)
	AssessmentCSSTSnapshot            map[string]any `json:"assessment_csst_snapshot"`
	AssessmentAnnounceSessionSnapshot map[string]any `json:"assessment_announce_session_snapshot"`
	AssessmentCollectSessionSnapshot  map[string]any `json:"assessment_collect_session_snapshot"`

	AssessmentCreatedAt time.Time  `json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time  `json:"assessment_updated_at"`
	AssessmentDeletedAt *time.Time `json:"assessment_deleted_at,omitempty"`
}

type ListAssessmentResponse struct {
	Data   []AssessmentResponse `json:"data"`
	Total  int64                `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

func FromModelAssesment(m model.AssessmentModel) AssessmentResponse {
	// DeletedAt → *time.Time
	var deletedAt *time.Time
	if m.AssessmentDeletedAt.Valid {
		t := m.AssessmentDeletedAt.Time
		deletedAt = &t
	}

	// JSONB snapshots (datatypes.JSONMap adalah map[string]any)
	var csstSnap map[string]any
	if m.AssessmentCSSTSnapshot != nil {
		csstSnap = make(map[string]any, len(m.AssessmentCSSTSnapshot))
		for k, v := range m.AssessmentCSSTSnapshot {
			csstSnap[k] = v
		}
	} else {
		csstSnap = map[string]any{}
	}

	var annSnap map[string]any
	if m.AssessmentAnnounceSessionSnapshot != nil {
		annSnap = make(map[string]any, len(m.AssessmentAnnounceSessionSnapshot))
		for k, v := range m.AssessmentAnnounceSessionSnapshot {
			annSnap[k] = v
		}
	} else {
		annSnap = map[string]any{}
	}

	var colSnap map[string]any
	if m.AssessmentCollectSessionSnapshot != nil {
		colSnap = make(map[string]any, len(m.AssessmentCollectSessionSnapshot))
		for k, v := range m.AssessmentCollectSessionSnapshot {
			colSnap[k] = v
		}
	} else {
		colSnap = map[string]any{}
	}

	return AssessmentResponse{
		AssessmentID:                           m.AssessmentID,
		AssessmentMasjidID:                     m.AssessmentMasjidID,
		AssessmentClassSectionSubjectTeacherID: m.AssessmentClassSectionSubjectTeacherID,
		AssessmentTypeID:                       m.AssessmentTypeID,

		AssessmentSlug:        m.AssessmentSlug,
		AssessmentTitle:       m.AssessmentTitle,
		AssessmentDescription: m.AssessmentDescription,

		AssessmentStartAt:     m.AssessmentStartAt,
		AssessmentDueAt:       m.AssessmentDueAt,
		AssessmentPublishedAt: m.AssessmentPublishedAt,
		AssessmentClosedAt:    m.AssessmentClosedAt,

		AssessmentDurationMinutes:      m.AssessmentDurationMinutes,
		AssessmentTotalAttemptsAllowed: m.AssessmentTotalAttemptsAllowed,
		AssessmentMaxScore:             m.AssessmentMaxScore,
		AssessmentIsPublished:          m.AssessmentIsPublished,
		AssessmentAllowSubmission:      m.AssessmentAllowSubmission,

		AssessmentCreatedByTeacherID: m.AssessmentCreatedByTeacherID,

		AssessmentSubmissionMode:    string(m.AssessmentSubmissionMode),
		AssessmentAnnounceSessionID: m.AssessmentAnnounceSessionID,
		AssessmentCollectSessionID:  m.AssessmentCollectSessionID,

		AssessmentCSSTSnapshot:            csstSnap,
		AssessmentAnnounceSessionSnapshot: annSnap,
		AssessmentCollectSessionSnapshot:  colSnap,

		AssessmentCreatedAt: m.AssessmentCreatedAt,
		AssessmentUpdatedAt: m.AssessmentUpdatedAt,
		AssessmentDeletedAt: deletedAt,
	}
}

func FromModelsAssesments(items []model.AssessmentModel) []AssessmentResponse {
	out := make([]AssessmentResponse, 0, len(items))
	for _, it := range items {
		out = append(out, FromModelAssesment(it))
	}
	return out
}
