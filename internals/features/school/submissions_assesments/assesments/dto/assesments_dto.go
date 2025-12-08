// file: internals/features/school/submissions_assesments/assesments/dto/assessment_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	assessModel "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"
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

	// total quiz/komponen quiz di assessment ini (global, sama utk semua siswa)
	// opsional; kalau kosong, bisa diisi di controller dari jumlah quiz inline
	AssessmentQuizTotal *int `json:"assessment_quiz_total" validate:"omitempty,gte=0,lte=255"`

	AssessmentIsPublished     *bool `json:"assessment_is_published"`
	AssessmentAllowSubmission *bool `json:"assessment_allow_submission"`

	// Audit
	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id" validate:"omitempty,uuid4"`

	// Mode session (opsional)
	AssessmentAnnounceSessionID *uuid.UUID `json:"assessment_announce_session_id" validate:"omitempty,uuid4"`
	AssessmentCollectSessionID  *uuid.UUID `json:"assessment_collect_session_id" validate:"omitempty,uuid4"`
}

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

	// quiz_total tidak dipaksa di sini; bisa diisi dari jumlah quiz inline di controller
	if r.AssessmentQuizTotal != nil && *r.AssessmentQuizTotal < 0 {
		z := 0
		r.AssessmentQuizTotal = &z
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

	quizTotal := 0
	if r.AssessmentQuizTotal != nil && *r.AssessmentQuizTotal > 0 {
		quizTotal = *r.AssessmentQuizTotal
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
		AssessmentQuizTotal:                    quizTotal,
		AssessmentCreatedByTeacherID:           r.AssessmentCreatedByTeacherID,
		AssessmentSubmissionMode:               assessModel.SubmissionModeDate, // akan dioverride di controller
		AssessmentIsPublished:                  *r.AssessmentIsPublished,
		AssessmentAllowSubmission:              *r.AssessmentAllowSubmission,
		// Snapshot fields & counters pakai default DB / diisi di service
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

	// boleh PATCH total quiz juga kalau guru edit struktur penilaian
	AssessmentQuizTotal *int `json:"assessment_quiz_total" validate:"omitempty,gte=0,lte=255"`

	AssessmentIsPublished     *bool `json:"assessment_is_published"`
	AssessmentAllowSubmission *bool `json:"assessment_allow_submission"`

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

	if r.AssessmentQuizTotal != nil && *r.AssessmentQuizTotal < 0 {
		z := 0
		r.AssessmentQuizTotal = &z
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

	if r.AssessmentQuizTotal != nil {
		// udah dinormalize minimal 0
		m.AssessmentQuizTotal = *r.AssessmentQuizTotal
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

	// Session IDs + submission_mode biasanya di-handle di controller,
	// karena terkait logic snapshot & mode 'date' vs 'session'.
}

/*
========================================================

	RESPONSE DTO

========================================================
*/

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

	// total quiz/komponen quiz global untuk assessment ini
	AssessmentQuizTotal       int  `json:"assessment_quiz_total"`
	AssessmentIsPublished     bool `json:"assessment_is_published"`
	AssessmentAllowSubmission bool `json:"assessment_allow_submission"`

	// counter submissions (read-only; diisi dari backend)
	AssessmentSubmissionsTotal       int `json:"assessment_submissions_total"`
	AssessmentSubmissionsGradedTotal int `json:"assessment_submissions_graded_total"`

	// flag hasil grading tipe assessment (snapshot dari AssessmentType)
	AssessmentTypeIsGradedSnapshot bool `json:"assessment_type_is_graded_snapshot"`

	// Snapshot aturan dari AssessmentType (sesuai SQL terbaru: hanya late policy & passing score)
	AssessmentAllowLateSubmissionSnapshot bool    `json:"assessment_allow_late_submission_snapshot"`
	AssessmentLatePenaltyPercentSnapshot  float64 `json:"assessment_late_penalty_percent_snapshot"`
	AssessmentPassingScorePercentSnapshot float64 `json:"assessment_passing_score_percent_snapshot"`

	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id,omitempty"`

	AssessmentSubmissionMode    string     `json:"assessment_submission_mode"`
	AssessmentAnnounceSessionID *uuid.UUID `json:"assessment_announce_session_id,omitempty"`
	AssessmentCollectSessionID  *uuid.UUID `json:"assessment_collect_session_id,omitempty"`

	AssessmentCreatedAt time.Time `json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time `json:"assessment_updated_at"`
}

// Converter Model → Response DTO
func FromModelAssesment(m assessModel.AssessmentModel) AssessmentResponse {

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
		AssessmentQuizTotal:            m.AssessmentQuizTotal,
		AssessmentIsPublished:          m.AssessmentIsPublished,
		AssessmentAllowSubmission:      m.AssessmentAllowSubmission,

		// counters
		AssessmentSubmissionsTotal:       m.AssessmentSubmissionsTotal,
		AssessmentSubmissionsGradedTotal: m.AssessmentSubmissionsGradedTotal,

		// flag hasil grading type
		AssessmentTypeIsGradedSnapshot: m.AssessmentTypeIsGradedSnapshot,

		// snapshot aturan dari AssessmentType (late policy + passing score)
		AssessmentAllowLateSubmissionSnapshot: m.AssessmentAllowLateSubmissionSnapshot,
		AssessmentLatePenaltyPercentSnapshot:  m.AssessmentLatePenaltyPercentSnapshot,
		AssessmentPassingScorePercentSnapshot: m.AssessmentPassingScorePercentSnapshot,

		AssessmentCreatedByTeacherID: m.AssessmentCreatedByTeacherID,

		AssessmentSubmissionMode:    string(m.AssessmentSubmissionMode),
		AssessmentAnnounceSessionID: m.AssessmentAnnounceSessionID,
		AssessmentCollectSessionID:  m.AssessmentCollectSessionID,

		AssessmentCreatedAt: m.AssessmentCreatedAt,
		AssessmentUpdatedAt: m.AssessmentUpdatedAt,
	}
}

/* ========================================================
   COMBINED DTO: Assessment + Quiz(es)
======================================================== */

type CreateAssessmentWithQuizzesRequest struct {
	Assessment CreateAssessmentRequest `json:"assessment" validate:"required"`
	Quiz       *CreateQuizInline       `json:"quiz,omitempty"`
	Quizzes    []CreateQuizInline      `json:"quizzes,omitempty"`
}

// Normalize: normalize assessment + semua quiz inline
func (r *CreateAssessmentWithQuizzesRequest) Normalize() {
	r.Assessment.Normalize()

	if r.Quiz != nil {
		r.Quiz.Normalize()
	}
	for i := range r.Quizzes {
		r.Quizzes[i].Normalize()
	}
}

// FlattenQuizzes:
// - Kalau ada "quizzes" dan len>0 → pakai itu
// - Else kalau ada "quiz" tunggal → jadikan slice 1 elemen
// - Else → nil
func (r *CreateAssessmentWithQuizzesRequest) FlattenQuizzes() []CreateQuizInline {
	if len(r.Quizzes) > 0 {
		return r.Quizzes
	}
	if r.Quiz != nil {
		return []CreateQuizInline{*r.Quiz}
	}
	return nil
}

/* ========================================================
   COMPACT DTO (untuk list / dropdown / kartu ringan)
======================================================== */

type AssessmentCompactResponse struct {
	AssessmentID       uuid.UUID `json:"assessment_id"`
	AssessmentSchoolID uuid.UUID `json:"assessment_school_id"`

	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id,omitempty"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id,omitempty"`

	AssessmentSlug  *string `json:"assessment_slug,omitempty"`
	AssessmentTitle string  `json:"assessment_title"`
	AssessmentKind  string  `json:"assessment_kind"`

	AssessmentStartAt *time.Time `json:"assessment_start_at,omitempty"`
	AssessmentDueAt   *time.Time `json:"assessment_due_at,omitempty"`

	AssessmentMaxScore        float64 `json:"assessment_max_score"`
	AssessmentQuizTotal       int     `json:"assessment_quiz_total"`
	AssessmentIsPublished     bool    `json:"assessment_is_published"`
	AssessmentAllowSubmission bool    `json:"assessment_allow_submission"`

	// Ringkas tapi tetap ada info progress
	AssessmentSubmissionsTotal       int `json:"assessment_submissions_total"`
	AssessmentSubmissionsGradedTotal int `json:"assessment_submissions_graded_total"`

	AssessmentCreatedAt time.Time `json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time `json:"assessment_updated_at"`
}

// Single mapper: Model → Compact DTO
func FromAssessmentModelCompact(m assessModel.AssessmentModel) AssessmentCompactResponse {
	return AssessmentCompactResponse{
		AssessmentID:       m.AssessmentID,
		AssessmentSchoolID: m.AssessmentSchoolID,

		AssessmentClassSectionSubjectTeacherID: m.AssessmentClassSectionSubjectTeacherID,
		AssessmentTypeID:                       m.AssessmentTypeID,

		AssessmentSlug:  m.AssessmentSlug,
		AssessmentTitle: m.AssessmentTitle,
		AssessmentKind:  string(m.AssessmentKind),

		AssessmentStartAt: m.AssessmentStartAt,
		AssessmentDueAt:   m.AssessmentDueAt,

		AssessmentMaxScore:        m.AssessmentMaxScore,
		AssessmentQuizTotal:       m.AssessmentQuizTotal,
		AssessmentIsPublished:     m.AssessmentIsPublished,
		AssessmentAllowSubmission: m.AssessmentAllowSubmission,

		AssessmentSubmissionsTotal:       m.AssessmentSubmissionsTotal,
		AssessmentSubmissionsGradedTotal: m.AssessmentSubmissionsGradedTotal,

		AssessmentCreatedAt: m.AssessmentCreatedAt,
		AssessmentUpdatedAt: m.AssessmentUpdatedAt,
	}
}

// Slice mapper: []Model → []Compact DTO
func FromAssessmentModelsCompact(rows []assessModel.AssessmentModel) []AssessmentCompactResponse {
	out := make([]AssessmentCompactResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromAssessmentModelCompact(rows[i]))
	}
	return out
}
