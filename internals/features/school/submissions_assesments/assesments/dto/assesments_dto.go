// file: internals/features/school/submissions_assesments/assesments/dto/assessment_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	dbtime "madinahsalam_backend/internals/helpers/dbtime"

	sessionModel "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
	assessModel "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"
	assessService "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/service"
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
	AssessmentStatus               *string `json:"assessment_status" validate:"omitempty,oneof=draft published archived"`
	AssessmentDurationMinutes      *int    `json:"assessment_duration_minutes" validate:"omitempty,gte=1,lte=1440"`
	AssessmentTotalAttemptsAllowed int     `json:"assessment_total_attempts_allowed" validate:"omitempty,gte=1,lte=50"`
	AssessmentMaxScore             float64 `json:"assessment_max_score" validate:"omitempty,gte=0,lte=100"`

	// total quiz/komponen quiz di assessment ini (global, sama utk semua siswa)
	AssessmentQuizTotal *int `json:"assessment_quiz_total" validate:"omitempty,gte=0,lte=255"`

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

	// normalize status (lowercase)
	if r.AssessmentStatus != nil {
		s := strings.ToLower(strings.TrimSpace(*r.AssessmentStatus))
		if s == "" {
			r.AssessmentStatus = nil
		} else {
			r.AssessmentStatus = &s
		}
	}

	// Default status kalau tetap kosong → draft
	if r.AssessmentStatus == nil {
		s := "draft"
		r.AssessmentStatus = &s
	}

	// quiz_total tidak dipaksa di sini
	if r.AssessmentQuizTotal != nil && *r.AssessmentQuizTotal < 0 {
		z := 0
		r.AssessmentQuizTotal = &z
	}
}

// Convert Create DTO → Model
func (r *CreateAssessmentRequest) ToModel() assessModel.AssessmentModel {
	kind := assessModel.AssessmentKind(r.AssessmentKind)
	if r.AssessmentKind == "" {
		kind = assessModel.AssessmentKindQuiz
	}

	status := assessModel.AssessmentStatusDraft
	if r.AssessmentStatus != nil && *r.AssessmentStatus != "" {
		status = assessModel.AssessmentStatus(*r.AssessmentStatus)
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
		AssessmentStatus:                       status,
		AssessmentDurationMinutes:              r.AssessmentDurationMinutes,
		AssessmentTotalAttemptsAllowed:         r.AssessmentTotalAttemptsAllowed,
		AssessmentMaxScore:                     r.AssessmentMaxScore,
		AssessmentQuizTotal:                    quizTotal,
		AssessmentCreatedByTeacherID:           r.AssessmentCreatedByTeacherID,
		AssessmentSubmissionMode:               assessModel.SubmissionModeDate, // akan dioverride di controller
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
	AssessmentStatus               *string  `json:"assessment_status" validate:"omitempty,oneof=draft published archived"`
	AssessmentDurationMinutes      *int     `json:"assessment_duration_minutes" validate:"omitempty,gte=1,lte=1440"`
	AssessmentTotalAttemptsAllowed *int     `json:"assessment_total_attempts_allowed" validate:"omitempty,gte=1,lte=50"`
	AssessmentMaxScore             *float64 `json:"assessment_max_score" validate:"omitempty,gte=0,lte=100"`

	AssessmentQuizTotal *int `json:"assessment_quiz_total" validate:"omitempty,gte=0,lte=255"`

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

	if r.AssessmentStatus != nil {
		s := strings.ToLower(strings.TrimSpace(*r.AssessmentStatus))
		if s == "" {
			r.AssessmentStatus = nil
		} else {
			r.AssessmentStatus = &s
		}
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
	if r.AssessmentStatus != nil && *r.AssessmentStatus != "" {
		m.AssessmentStatus = assessModel.AssessmentStatus(*r.AssessmentStatus)
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
		m.AssessmentQuizTotal = *r.AssessmentQuizTotal
	}

	if r.AssessmentCreatedByTeacherID != nil {
		m.AssessmentCreatedByTeacherID = r.AssessmentCreatedByTeacherID
	}
}

/*
========================================================

	RESPONSE DTO (FULL)

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

	AssessmentKind   string `json:"assessment_kind"`
	AssessmentStatus string `json:"assessment_status"`

	AssessmentDurationMinutes      *int    `json:"assessment_duration_minutes,omitempty"`
	AssessmentTotalAttemptsAllowed int     `json:"assessment_total_attempts_allowed"`
	AssessmentMaxScore             float64 `json:"assessment_max_score"`

	AssessmentQuizTotal int `json:"assessment_quiz_total"`

	// counter submissions
	AssessmentSubmissionsTotal       int `json:"assessment_submissions_total"`
	AssessmentSubmissionsGradedTotal int `json:"assessment_submissions_graded_total"`

	AssessmentCreatedByTeacherID *uuid.UUID `json:"assessment_created_by_teacher_id,omitempty"`

	AssessmentSubmissionMode    string     `json:"assessment_submission_mode"`
	AssessmentAnnounceSessionID *uuid.UUID `json:"assessment_announce_session_id,omitempty"`
	AssessmentCollectSessionID  *uuid.UUID `json:"assessment_collect_session_id,omitempty"`

	AssessmentCreatedAt time.Time `json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time `json:"assessment_updated_at"`

	// 🔥 Computed field utama
	AssessmentIsOpen bool `json:"assessment_is_open"`
}

// Shared builder
func buildAssessmentResponse(m assessModel.AssessmentModel, isOpen bool) AssessmentResponse {
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

		AssessmentKind:   string(m.AssessmentKind),
		AssessmentStatus: string(m.AssessmentStatus),

		AssessmentDurationMinutes:      m.AssessmentDurationMinutes,
		AssessmentTotalAttemptsAllowed: m.AssessmentTotalAttemptsAllowed,
		AssessmentMaxScore:             m.AssessmentMaxScore,
		AssessmentQuizTotal:            m.AssessmentQuizTotal,

		AssessmentSubmissionsTotal:       m.AssessmentSubmissionsTotal,
		AssessmentSubmissionsGradedTotal: m.AssessmentSubmissionsGradedTotal,

		AssessmentCreatedByTeacherID: m.AssessmentCreatedByTeacherID,

		AssessmentSubmissionMode:    string(m.AssessmentSubmissionMode),
		AssessmentAnnounceSessionID: m.AssessmentAnnounceSessionID,
		AssessmentCollectSessionID:  m.AssessmentCollectSessionID,

		AssessmentCreatedAt: m.AssessmentCreatedAt,
		AssessmentUpdatedAt: m.AssessmentUpdatedAt,

		AssessmentIsOpen: isOpen,
	}
}

// Converter Model → Response DTO (tanpa session collect) — versi lama (UTC)
func FromModelAssesment(m assessModel.AssessmentModel) AssessmentResponse {
	now := time.Now().UTC()
	isOpen := assessService.ComputeIsOpen(&m, now)
	return buildAssessmentResponse(m, isOpen)
}

func FromAssesmentModels(rows []assessModel.AssessmentModel) []AssessmentResponse {
	out := make([]AssessmentResponse, 0, len(rows))
	now := time.Now().UTC()

	for i := range rows {
		m := rows[i]
		isOpen := assessService.ComputeIsOpen(&m, now)
		out = append(out, buildAssessmentResponse(m, isOpen))
	}

	return out
}

// Versi baru: FULL DTO pakai dbtime (timezone sekolah)
func FromModelAssesmentWithSchoolTime(c *fiber.Ctx, m assessModel.AssessmentModel) AssessmentResponse {
	now, _ := dbtime.GetDBTime(c)
	isOpen := assessService.ComputeIsOpen(&m, now)

	resp := buildAssessmentResponse(m, isOpen)

	// override waktu ke timezone sekolah
	resp.AssessmentStartAt = dbtime.ToSchoolTimePtr(c, m.AssessmentStartAt)
	resp.AssessmentDueAt = dbtime.ToSchoolTimePtr(c, m.AssessmentDueAt)
	resp.AssessmentPublishedAt = dbtime.ToSchoolTimePtr(c, m.AssessmentPublishedAt)
	resp.AssessmentClosedAt = dbtime.ToSchoolTimePtr(c, m.AssessmentClosedAt)

	resp.AssessmentCreatedAt = dbtime.ToSchoolTime(c, m.AssessmentCreatedAt)
	resp.AssessmentUpdatedAt = dbtime.ToSchoolTime(c, m.AssessmentUpdatedAt)

	return resp
}

func FromAssesmentModelsWithSchoolTime(c *fiber.Ctx, rows []assessModel.AssessmentModel) []AssessmentResponse {
	out := make([]AssessmentResponse, 0, len(rows))
	now, _ := dbtime.GetDBTime(c)

	for i := range rows {
		m := rows[i]
		isOpen := assessService.ComputeIsOpen(&m, now)

		resp := buildAssessmentResponse(m, isOpen)

		resp.AssessmentStartAt = dbtime.ToSchoolTimePtr(c, m.AssessmentStartAt)
		resp.AssessmentDueAt = dbtime.ToSchoolTimePtr(c, m.AssessmentDueAt)
		resp.AssessmentPublishedAt = dbtime.ToSchoolTimePtr(c, m.AssessmentPublishedAt)
		resp.AssessmentClosedAt = dbtime.ToSchoolTimePtr(c, m.AssessmentClosedAt)

		resp.AssessmentCreatedAt = dbtime.ToSchoolTime(c, m.AssessmentCreatedAt)
		resp.AssessmentUpdatedAt = dbtime.ToSchoolTime(c, m.AssessmentUpdatedAt)

		out = append(out, resp)
	}

	return out
}

// Versi dengan CollectSession (masih pakai UTC, kalau mau dibikin school time bisa ditambah varian baru)
func FromModelAssesmentWithCollectSession(
	m assessModel.AssessmentModel,
	sess *sessionModel.ClassAttendanceSessionModel,
) AssessmentResponse {
	now := time.Now().UTC()
	isOpen := assessService.ComputeIsOpenWithCollectSession(&m, sess, now)
	return buildAssessmentResponse(m, isOpen)
}

func FromAssesmentModelsWithCollectSessions(
	rows []assessModel.AssessmentModel,
	collectSessions map[uuid.UUID]*sessionModel.ClassAttendanceSessionModel,
) []AssessmentResponse {
	out := make([]AssessmentResponse, 0, len(rows))
	now := time.Now().UTC()

	for i := range rows {
		m := rows[i]

		var sess *sessionModel.ClassAttendanceSessionModel
		if m.AssessmentCollectSessionID != nil {
			if s, ok := collectSessions[*m.AssessmentCollectSessionID]; ok {
				sess = s
			}
		}

		isOpen := assessService.ComputeIsOpenWithCollectSession(&m, sess, now)
		out = append(out, buildAssessmentResponse(m, isOpen))
	}

	return out
}

/* ========================================================
   COMBINED DTO: Assessment + Quiz(es)
======================================================== */

type CreateAssessmentWithQuizzesRequest struct {
	Assessment CreateAssessmentRequest `json:"assessment" validate:"required"`
	Quiz       *CreateQuizInline       `json:"quiz,omitempty"`
	Quizzes    []CreateQuizInline      `json:"quizzes,omitempty"`
}

func (r *CreateAssessmentWithQuizzesRequest) Normalize() {
	r.Assessment.Normalize()

	if r.Quiz != nil {
		r.Quiz.Normalize()
	}
	for i := range r.Quizzes {
		r.Quizzes[i].Normalize()
	}
}

func (r *CreateAssessmentWithQuizzesRequest) FlattenQuizzes() []CreateQuizInline {
	if len(r.Quizzes) > 0 {
		return r.Quizzes
	}
	if r.Quiz != nil {
		return []CreateQuizInline{*r.Quiz}
	}
	return nil
}

/*
	========================================================
	  COMPACT DTO (untuk list / dropdown / kartu ringan)
	========================================================
*/

type AssessmentCompactResponse struct {
	AssessmentID uuid.UUID `json:"assessment_id"`

	AssessmentClassSectionSubjectTeacherID *uuid.UUID `json:"assessment_class_section_subject_teacher_id,omitempty"`
	AssessmentTypeID                       *uuid.UUID `json:"assessment_type_id,omitempty"`

	AssessmentSlug   *string `json:"assessment_slug,omitempty"`
	AssessmentTitle  string  `json:"assessment_title"`
	AssessmentKind   string  `json:"assessment_kind"`
	AssessmentStatus string  `json:"assessment_status"`

	AssessmentStartAt *time.Time `json:"assessment_start_at,omitempty"`
	AssessmentDueAt   *time.Time `json:"assessment_due_at,omitempty"`

	AssessmentMaxScore             float64 `json:"assessment_max_score"`
	AssessmentQuizTotal            int     `json:"assessment_quiz_total"`
	AssessmentTotalAttemptsAllowed int     `json:"assessment_total_attempts_allowed" validate:"omitempty,gte=1,lte=50"`

	AssessmentIsOpen bool `json:"assessment_is_open"`

	AssessmentCreatedAt time.Time `json:"assessment_created_at"`
	AssessmentUpdatedAt time.Time `json:"assessment_updated_at"`
}

// Versi compact lama (UTC)
func FromAssessmentModelCompact(m assessModel.AssessmentModel) AssessmentCompactResponse {
	now := time.Now().UTC()
	isOpen := assessService.ComputeIsOpen(&m, now)

	return AssessmentCompactResponse{
		AssessmentID: m.AssessmentID,

		AssessmentClassSectionSubjectTeacherID: m.AssessmentClassSectionSubjectTeacherID,
		AssessmentTypeID:                       m.AssessmentTypeID,

		AssessmentSlug:   m.AssessmentSlug,
		AssessmentTitle:  m.AssessmentTitle,
		AssessmentKind:   string(m.AssessmentKind),
		AssessmentStatus: string(m.AssessmentStatus),

		AssessmentStartAt: m.AssessmentStartAt,
		AssessmentDueAt:   m.AssessmentDueAt,

		AssessmentMaxScore:             m.AssessmentMaxScore,
		AssessmentQuizTotal:            m.AssessmentQuizTotal,
		AssessmentTotalAttemptsAllowed: m.AssessmentTotalAttemptsAllowed,
		AssessmentIsOpen:               isOpen,

		AssessmentCreatedAt: m.AssessmentCreatedAt,
		AssessmentUpdatedAt: m.AssessmentUpdatedAt,
	}
}

func FromAssessmentModelsCompact(rows []assessModel.AssessmentModel) []AssessmentCompactResponse {
	out := make([]AssessmentCompactResponse, 0, len(rows))
	now := time.Now().UTC()

	for i := range rows {
		m := rows[i]
		isOpen := assessService.ComputeIsOpen(&m, now)

		out = append(out, AssessmentCompactResponse{
			AssessmentID: m.AssessmentID,

			AssessmentClassSectionSubjectTeacherID: m.AssessmentClassSectionSubjectTeacherID,
			AssessmentTypeID:                       m.AssessmentTypeID,

			AssessmentSlug:   m.AssessmentSlug,
			AssessmentTitle:  m.AssessmentTitle,
			AssessmentKind:   string(m.AssessmentKind),
			AssessmentStatus: string(m.AssessmentStatus),

			AssessmentStartAt: m.AssessmentStartAt,
			AssessmentDueAt:   m.AssessmentDueAt,

			AssessmentMaxScore:             m.AssessmentMaxScore,
			AssessmentQuizTotal:            m.AssessmentQuizTotal,
			AssessmentTotalAttemptsAllowed: m.AssessmentTotalAttemptsAllowed,

			AssessmentIsOpen: isOpen,

			AssessmentCreatedAt: m.AssessmentCreatedAt,
			AssessmentUpdatedAt: m.AssessmentUpdatedAt,
		})
	}

	return out
}

// Versi compact + dbtime (dipakai di controller List)
func FromAssessmentModelsCompactWithSchoolTime(c *fiber.Ctx, rows []assessModel.AssessmentModel) []AssessmentCompactResponse {
	out := make([]AssessmentCompactResponse, 0, len(rows))
	now, _ := dbtime.GetDBTime(c)

	for i := range rows {
		m := rows[i]
		isOpen := assessService.ComputeIsOpen(&m, now)

		item := AssessmentCompactResponse{
			AssessmentID: m.AssessmentID,

			AssessmentClassSectionSubjectTeacherID: m.AssessmentClassSectionSubjectTeacherID,
			AssessmentTypeID:                       m.AssessmentTypeID,

			AssessmentSlug:   m.AssessmentSlug,
			AssessmentTitle:  m.AssessmentTitle,
			AssessmentKind:   string(m.AssessmentKind),
			AssessmentStatus: string(m.AssessmentStatus),

			AssessmentStartAt: dbtime.ToSchoolTimePtr(c, m.AssessmentStartAt),
			AssessmentDueAt:   dbtime.ToSchoolTimePtr(c, m.AssessmentDueAt),

			AssessmentMaxScore:             m.AssessmentMaxScore,
			AssessmentQuizTotal:            m.AssessmentQuizTotal,
			AssessmentTotalAttemptsAllowed: m.AssessmentTotalAttemptsAllowed,

			AssessmentIsOpen: isOpen,

			AssessmentCreatedAt: dbtime.ToSchoolTime(c, m.AssessmentCreatedAt),
			AssessmentUpdatedAt: dbtime.ToSchoolTime(c, m.AssessmentUpdatedAt),
		}

		out = append(out, item)
	}

	return out
}
