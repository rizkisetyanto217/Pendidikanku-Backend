// file: internals/features/school/assessments/dto/assessment_type_dto.go
package dto

import (
	"strings"
	"time"

	model "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"
	"madinahsalam_backend/internals/helpers/dbtime"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* ==============================
   REQUESTS
============================== */

// Create (POST /assessment-types)
// Catatan: assessment_type_school_id tetap diisi dari token di controller.
type CreateAssessmentTypeRequest struct {
	AssessmentTypeSchoolID      uuid.UUID `json:"assessment_type_school_id" validate:"required"`
	AssessmentTypeKey           string    `json:"assessment_type_key" validate:"required,max=32"`
	AssessmentTypeName          string    `json:"assessment_type_name" validate:"required,max=120"`
	AssessmentTypeWeightPercent float64   `json:"assessment_type_weight_percent" validate:"gte=0,lte=100"`

	// Jenis besar assessment: training / daily_exam / exam
	// Optional di payload, default = "training"
	AssessmentTypeCategory *string `json:"assessment_type" validate:"omitempty,oneof=training daily_exam exam"`

	AssessmentTypeIsActive *bool `json:"assessment_type_is_active" validate:"omitempty"`
	AssessmentTypeIsGraded *bool `json:"assessment_type_is_graded" validate:"omitempty"`

	// ===== Default quiz settings =====

	AssessmentTypeShuffleQuestions       *bool `json:"assessment_type_shuffle_questions" validate:"omitempty"`
	AssessmentTypeShuffleOptions         *bool `json:"assessment_type_shuffle_options" validate:"omitempty"`
	AssessmentTypeShowCorrectAfterSubmit *bool `json:"assessment_type_show_correct_after_submit" validate:"omitempty"`

	// ✅ ganti 2 flag jadi satu strict mode
	AssessmentTypeStrictMode *bool `json:"assessment_type_strict_mode" validate:"omitempty"`

	// Waktu per soal (dalam detik); NULL = tanpa batas
	AssessmentTypeTimePerQuestionSec *int `json:"assessment_type_time_per_question_sec" validate:"omitempty,min=0"`

	AssessmentTypeAttemptsAllowed *int  `json:"assessment_type_attempts_allowed" validate:"omitempty,min=1"`
	AssessmentTypeRequireLogin    *bool `json:"assessment_type_require_login" validate:"omitempty"`

	// ===== Default late policy & scoring =====

	AssessmentTypeAllowLateSubmission *bool    `json:"assessment_type_allow_late_submission" validate:"omitempty"`
	AssessmentTypeLatePenaltyPercent  *float64 `json:"assessment_type_late_penalty_percent" validate:"omitempty,gte=0,lte=100"`
	AssessmentTypePassingScorePercent *float64 `json:"assessment_type_passing_score_percent" validate:"omitempty,gte=0,lte=100"`

	// 🔁 sekarang sudah enum: first / latest / highest / average
	AssessmentTypeScoreAggregationMode *string `json:"assessment_type_score_aggregation_mode" validate:"omitempty,oneof=first latest highest average"`

	AssessmentTypeShowScoreAfterSubmit        *bool `json:"assessment_type_show_score_after_submit" validate:"omitempty"`
	AssessmentTypeShowCorrectAfterClosed      *bool `json:"assessment_type_show_correct_after_closed" validate:"omitempty"`
	AssessmentTypeAllowReviewBeforeSubmit     *bool `json:"assessment_type_allow_review_before_submit" validate:"omitempty"`
	AssessmentTypeRequireCompleteAttempt      *bool `json:"assessment_type_require_complete_attempt" validate:"omitempty"`
	AssessmentTypeShowDetailsAfterAllAttempts *bool `json:"assessment_type_show_details_after_all_attempts" validate:"omitempty"`
}

// Patch (PATCH /assessment-types/:id) — partial update
type PatchAssessmentTypeRequest struct {
	AssessmentTypeName          *string  `json:"assessment_type_name" validate:"omitempty,max=120"`
	AssessmentTypeWeightPercent *float64 `json:"assessment_type_weight_percent" validate:"omitempty,gte=0,lte=100"`
	AssessmentTypeIsActive      *bool    `json:"assessment_type_is_active" validate:"omitempty"`

	AssessmentTypeIsGraded *bool `json:"assessment_type_is_graded" validate:"omitempty"`

	// Ubah kategori besar assessment (training / daily_exam / exam)
	AssessmentTypeCategory *string `json:"assessment_type" validate:"omitempty,oneof=training daily_exam exam"`

	AssessmentTypeShuffleQuestions       *bool `json:"assessment_type_shuffle_questions" validate:"omitempty"`
	AssessmentTypeShuffleOptions         *bool `json:"assessment_type_shuffle_options" validate:"omitempty"`
	AssessmentTypeShowCorrectAfterSubmit *bool `json:"assessment_type_show_correct_after_submit" validate:"omitempty"`
	AssessmentTypeStrictMode             *bool `json:"assessment_type_strict_mode" validate:"omitempty"`

	// Waktu per soal (detik); NULL = tanpa batas
	AssessmentTypeTimePerQuestionSec *int `json:"assessment_type_time_per_question_sec" validate:"omitempty,min=0"`

	AssessmentTypeAttemptsAllowed *int  `json:"assessment_type_attempts_allowed" validate:"omitempty,min=1"`
	AssessmentTypeRequireLogin    *bool `json:"assessment_type_require_login" validate:"omitempty"`

	AssessmentTypeAllowLateSubmission *bool    `json:"assessment_type_allow_late_submission" validate:"omitempty"`
	AssessmentTypeLatePenaltyPercent  *float64 `json:"assessment_type_late_penalty_percent" validate:"omitempty,gte=0,lte=100"`
	AssessmentTypePassingScorePercent *float64 `json:"assessment_type_passing_score_percent" validate:"omitempty,gte=0,lte=100"`

	// 🔁 sync dengan enum: first / latest / highest / average
	AssessmentTypeScoreAggregationMode *string `json:"assessment_type_score_aggregation_mode" validate:"omitempty,oneof=first latest highest average"`

	AssessmentTypeShowScoreAfterSubmit        *bool `json:"assessment_type_show_score_after_submit" validate:"omitempty"`
	AssessmentTypeShowCorrectAfterClosed      *bool `json:"assessment_type_show_correct_after_closed" validate:"omitempty"`
	AssessmentTypeAllowReviewBeforeSubmit     *bool `json:"assessment_type_allow_review_before_submit" validate:"omitempty"`
	AssessmentTypeRequireCompleteAttempt      *bool `json:"assessment_type_require_complete_attempt" validate:"omitempty"`
	AssessmentTypeShowDetailsAfterAllAttempts *bool `json:"assessment_type_show_details_after_all_attempts" validate:"omitempty"`
}

// List filter (GET /assessment-types)
type ListAssessmentTypeFilter struct {
	AssessmentTypeSchoolID uuid.UUID `query:"school_id" validate:"required"` // diisi dari token di controller
	Active                 *bool     `query:"active" validate:"omitempty"`
	Q                      *string   `query:"q" validate:"omitempty,max=120"`
	Limit                  int       `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset                 int       `query:"offset" validate:"omitempty,min=0"`
	SortBy                 *string   `query:"sort_by" validate:"omitempty,oneof=name created_at"`
	SortDir                *string   `query:"sort_dir" validate:"omitempty,oneof=asc desc"`
}

/* ==============================
   RESPONSES
============================== */

type AssessmentTypeResponse struct {
	AssessmentTypeID            uuid.UUID `json:"assessment_type_id"`
	AssessmentTypeSchoolID      uuid.UUID `json:"assessment_type_school_id"`
	AssessmentTypeKey           string    `json:"assessment_type_key"`
	AssessmentTypeName          string    `json:"assessment_type_name"`
	AssessmentTypeWeightPercent float64   `json:"assessment_type_weight_percent"`

	// Jenis besar assessment: training / daily_exam / exam
	AssessmentTypeCategory string `json:"assessment_type"`

	// Default quiz settings
	AssessmentTypeShuffleQuestions       bool    `json:"assessment_type_shuffle_questions"`
	AssessmentTypeShuffleOptions         bool    `json:"assessment_type_shuffle_options"`
	AssessmentTypeShowCorrectAfterSubmit bool    `json:"assessment_type_show_correct_after_submit"`
	AssessmentTypeStrictMode             bool    `json:"assessment_type_strict_mode"`
	AssessmentTypeTimePerQuestionSec     *int    `json:"assessment_type_time_per_question_sec,omitempty"`
	AssessmentTypeAttemptsAllowed        int     `json:"assessment_type_attempts_allowed"`
	AssessmentTypeRequireLogin           bool    `json:"assessment_type_require_login"`
	AssessmentTypeIsActive               bool    `json:"assessment_type_is_active"`
	AssessmentTypeIsGraded               bool    `json:"assessment_type_is_graded"`
	AssessmentTypeAllowLateSubmission    bool    `json:"assessment_type_allow_late_submission"`
	AssessmentTypeLatePenaltyPercent     float64 `json:"assessment_type_late_penalty_percent"`
	AssessmentTypePassingScorePercent    float64 `json:"assessment_type_passing_score_percent"`

	// Scoring & review behavior
	AssessmentTypeScoreAggregationMode        string `json:"assessment_type_score_aggregation_mode"`
	AssessmentTypeShowScoreAfterSubmit        bool   `json:"assessment_type_show_score_after_submit"`
	AssessmentTypeShowCorrectAfterClosed      bool   `json:"assessment_type_show_correct_after_closed"`
	AssessmentTypeAllowReviewBeforeSubmit     bool   `json:"assessment_type_allow_review_before_submit"`
	AssessmentTypeRequireCompleteAttempt      bool   `json:"assessment_type_require_complete_attempt"`
	AssessmentTypeShowDetailsAfterAllAttempts bool   `json:"assessment_type_show_details_after_all_attempts"`

	AssessmentTypeCreatedAt time.Time `json:"assessment_type_created_at"`
	AssessmentTypeUpdatedAt time.Time `json:"assessment_type_updated_at"`
}

type ListAssessmentTypeResponse struct {
	Data   []AssessmentTypeResponse `json:"data"`
	Total  int64                    `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

/* ==============================
   MAPPERS / HELPERS
============================== */

func (r CreateAssessmentTypeRequest) Normalize() CreateAssessmentTypeRequest {
	r.AssessmentTypeKey = strings.TrimSpace(r.AssessmentTypeKey)
	r.AssessmentTypeName = strings.TrimSpace(r.AssessmentTypeName)
	return r
}

func (r CreateAssessmentTypeRequest) ToModel() model.AssessmentTypeModel {
	// Default active = true
	isActive := true
	if r.AssessmentTypeIsActive != nil {
		isActive = *r.AssessmentTypeIsActive
	}

	// Default: type graded
	isGraded := true
	if r.AssessmentTypeIsGraded != nil {
		isGraded = *r.AssessmentTypeIsGraded
	}

	// Quiz settings default (sync dengan default di DB/model)
	shuffleQuestions := false
	if r.AssessmentTypeShuffleQuestions != nil {
		shuffleQuestions = *r.AssessmentTypeShuffleQuestions
	}

	shuffleOptions := false
	if r.AssessmentTypeShuffleOptions != nil {
		shuffleOptions = *r.AssessmentTypeShuffleOptions
	}

	showCorrect := true
	if r.AssessmentTypeShowCorrectAfterSubmit != nil {
		showCorrect = *r.AssessmentTypeShowCorrectAfterSubmit
	}

	// strict mode default false
	strictMode := false
	if r.AssessmentTypeStrictMode != nil {
		strictMode = *r.AssessmentTypeStrictMode
	}

	var timePerQuestion *int
	if r.AssessmentTypeTimePerQuestionSec != nil {
		timePerQuestion = r.AssessmentTypeTimePerQuestionSec
	}

	attempts := 1
	if r.AssessmentTypeAttemptsAllowed != nil {
		attempts = *r.AssessmentTypeAttemptsAllowed
	}

	requireLogin := true
	if r.AssessmentTypeRequireLogin != nil {
		requireLogin = *r.AssessmentTypeRequireLogin
	}

	// Late & scoring defaults (sync dengan default DB/model)
	allowLate := false
	if r.AssessmentTypeAllowLateSubmission != nil {
		allowLate = *r.AssessmentTypeAllowLateSubmission
	}

	latePenalty := 0.0
	if r.AssessmentTypeLatePenaltyPercent != nil {
		latePenalty = *r.AssessmentTypeLatePenaltyPercent
	}

	passingScore := 0.0
	if r.AssessmentTypePassingScorePercent != nil {
		passingScore = *r.AssessmentTypePassingScorePercent
	}

	// Pakai default dari konstanta model
	scoreAgg := model.AssessmentScoreAggLatest
	if r.AssessmentTypeScoreAggregationMode != nil && strings.TrimSpace(*r.AssessmentTypeScoreAggregationMode) != "" {
		scoreAgg = strings.ToLower(strings.TrimSpace(*r.AssessmentTypeScoreAggregationMode))
	}

	showScoreAfterSubmit := true
	if r.AssessmentTypeShowScoreAfterSubmit != nil {
		showScoreAfterSubmit = *r.AssessmentTypeShowScoreAfterSubmit
	}

	showCorrectAfterClosed := false
	if r.AssessmentTypeShowCorrectAfterClosed != nil {
		showCorrectAfterClosed = *r.AssessmentTypeShowCorrectAfterClosed
	}

	allowReviewBeforeSubmit := true
	if r.AssessmentTypeAllowReviewBeforeSubmit != nil {
		allowReviewBeforeSubmit = *r.AssessmentTypeAllowReviewBeforeSubmit
	}

	requireCompleteAttempt := true
	if r.AssessmentTypeRequireCompleteAttempt != nil {
		requireCompleteAttempt = *r.AssessmentTypeRequireCompleteAttempt
	}

	showDetailsAfterAllAttempts := false
	if r.AssessmentTypeShowDetailsAfterAllAttempts != nil {
		showDetailsAfterAllAttempts = *r.AssessmentTypeShowDetailsAfterAllAttempts
	}

	// ====== Category (training / daily_exam / exam) ======
	category := model.AssessmentTypeEnumTraining
	if r.AssessmentTypeCategory != nil {
		cat := strings.ToLower(strings.TrimSpace(*r.AssessmentTypeCategory))
		if cat != "" {
			category = cat
		}
	}

	return model.AssessmentTypeModel{
		AssessmentTypeSchoolID:      r.AssessmentTypeSchoolID,
		AssessmentTypeKey:           r.AssessmentTypeKey,
		AssessmentTypeName:          r.AssessmentTypeName,
		AssessmentTypeWeightPercent: r.AssessmentTypeWeightPercent,

		AssessmentTypeCategory: category,

		AssessmentTypeShuffleQuestions:       shuffleQuestions,
		AssessmentTypeShuffleOptions:         shuffleOptions,
		AssessmentTypeShowCorrectAfterSubmit: showCorrect,
		AssessmentTypeStrictMode:             strictMode,
		AssessmentTypeTimePerQuestionSec:     timePerQuestion,
		AssessmentTypeAttemptsAllowed:        attempts,
		AssessmentTypeRequireLogin:           requireLogin,

		AssessmentTypeIsActive: isActive,
		AssessmentTypeIsGraded: isGraded,

		AssessmentTypeAllowLateSubmission:         allowLate,
		AssessmentTypeLatePenaltyPercent:          latePenalty,
		AssessmentTypePassingScorePercent:         passingScore,
		AssessmentTypeScoreAggregationMode:        scoreAgg,
		AssessmentTypeShowScoreAfterSubmit:        showScoreAfterSubmit,
		AssessmentTypeShowCorrectAfterClosed:      showCorrectAfterClosed,
		AssessmentTypeAllowReviewBeforeSubmit:     allowReviewBeforeSubmit,
		AssessmentTypeRequireCompleteAttempt:      requireCompleteAttempt,
		AssessmentTypeShowDetailsAfterAllAttempts: showDetailsAfterAllAttempts,
	}
}

func (p PatchAssessmentTypeRequest) Apply(m *model.AssessmentTypeModel) {
	if p.AssessmentTypeName != nil {
		name := strings.TrimSpace(*p.AssessmentTypeName)
		m.AssessmentTypeName = name
	}
	if p.AssessmentTypeWeightPercent != nil {
		m.AssessmentTypeWeightPercent = *p.AssessmentTypeWeightPercent
	}
	if p.AssessmentTypeIsActive != nil {
		m.AssessmentTypeIsActive = *p.AssessmentTypeIsActive
	}
	if p.AssessmentTypeIsGraded != nil {
		m.AssessmentTypeIsGraded = *p.AssessmentTypeIsGraded
	}

	// Update category (training / daily_exam / exam)
	if p.AssessmentTypeCategory != nil {
		cat := strings.ToLower(strings.TrimSpace(*p.AssessmentTypeCategory))
		if cat != "" {
			m.AssessmentTypeCategory = cat
		}
	}

	if p.AssessmentTypeShuffleQuestions != nil {
		m.AssessmentTypeShuffleQuestions = *p.AssessmentTypeShuffleQuestions
	}
	if p.AssessmentTypeShuffleOptions != nil {
		m.AssessmentTypeShuffleOptions = *p.AssessmentTypeShuffleOptions
	}
	if p.AssessmentTypeShowCorrectAfterSubmit != nil {
		m.AssessmentTypeShowCorrectAfterSubmit = *p.AssessmentTypeShowCorrectAfterSubmit
	}
	if p.AssessmentTypeStrictMode != nil {
		m.AssessmentTypeStrictMode = *p.AssessmentTypeStrictMode
	}

	if p.AssessmentTypeTimePerQuestionSec != nil {
		// Catatan: sama seperti sebelumnya, ini hanya overwrite nilai.
		// Kalau mau clear ke NULL via PATCH perlu pattern khusus (mis. flag explicit).
		m.AssessmentTypeTimePerQuestionSec = p.AssessmentTypeTimePerQuestionSec
	}

	if p.AssessmentTypeAttemptsAllowed != nil {
		m.AssessmentTypeAttemptsAllowed = *p.AssessmentTypeAttemptsAllowed
	}
	if p.AssessmentTypeRequireLogin != nil {
		m.AssessmentTypeRequireLogin = *p.AssessmentTypeRequireLogin
	}

	if p.AssessmentTypeAllowLateSubmission != nil {
		m.AssessmentTypeAllowLateSubmission = *p.AssessmentTypeAllowLateSubmission
	}
	if p.AssessmentTypeLatePenaltyPercent != nil {
		m.AssessmentTypeLatePenaltyPercent = *p.AssessmentTypeLatePenaltyPercent
	}
	if p.AssessmentTypePassingScorePercent != nil {
		m.AssessmentTypePassingScorePercent = *p.AssessmentTypePassingScorePercent
	}
	if p.AssessmentTypeScoreAggregationMode != nil {
		mode := strings.ToLower(strings.TrimSpace(*p.AssessmentTypeScoreAggregationMode))
		if mode != "" {
			m.AssessmentTypeScoreAggregationMode = mode
		}
	}
	if p.AssessmentTypeShowScoreAfterSubmit != nil {
		m.AssessmentTypeShowScoreAfterSubmit = *p.AssessmentTypeShowScoreAfterSubmit
	}
	if p.AssessmentTypeShowCorrectAfterClosed != nil {
		m.AssessmentTypeShowCorrectAfterClosed = *p.AssessmentTypeShowCorrectAfterClosed
	}
	if p.AssessmentTypeAllowReviewBeforeSubmit != nil {
		m.AssessmentTypeAllowReviewBeforeSubmit = *p.AssessmentTypeAllowReviewBeforeSubmit
	}
	if p.AssessmentTypeRequireCompleteAttempt != nil {
		m.AssessmentTypeRequireCompleteAttempt = *p.AssessmentTypeRequireCompleteAttempt
	}
	if p.AssessmentTypeShowDetailsAfterAllAttempts != nil {
		m.AssessmentTypeShowDetailsAfterAllAttempts = *p.AssessmentTypeShowDetailsAfterAllAttempts
	}
}

func FromModel(m model.AssessmentTypeModel) AssessmentTypeResponse {
	return AssessmentTypeResponse{
		AssessmentTypeID:            m.AssessmentTypeID,
		AssessmentTypeSchoolID:      m.AssessmentTypeSchoolID,
		AssessmentTypeKey:           m.AssessmentTypeKey,
		AssessmentTypeName:          m.AssessmentTypeName,
		AssessmentTypeWeightPercent: m.AssessmentTypeWeightPercent,

		AssessmentTypeCategory: m.AssessmentTypeCategory,

		AssessmentTypeShuffleQuestions:       m.AssessmentTypeShuffleQuestions,
		AssessmentTypeShuffleOptions:         m.AssessmentTypeShuffleOptions,
		AssessmentTypeShowCorrectAfterSubmit: m.AssessmentTypeShowCorrectAfterSubmit,
		AssessmentTypeStrictMode:             m.AssessmentTypeStrictMode,
		AssessmentTypeTimePerQuestionSec:     m.AssessmentTypeTimePerQuestionSec,
		AssessmentTypeAttemptsAllowed:        m.AssessmentTypeAttemptsAllowed,
		AssessmentTypeRequireLogin:           m.AssessmentTypeRequireLogin,

		AssessmentTypeAllowLateSubmission:         m.AssessmentTypeAllowLateSubmission,
		AssessmentTypeLatePenaltyPercent:          m.AssessmentTypeLatePenaltyPercent,
		AssessmentTypePassingScorePercent:         m.AssessmentTypePassingScorePercent,
		AssessmentTypeScoreAggregationMode:        m.AssessmentTypeScoreAggregationMode,
		AssessmentTypeShowScoreAfterSubmit:        m.AssessmentTypeShowScoreAfterSubmit,
		AssessmentTypeShowCorrectAfterClosed:      m.AssessmentTypeShowCorrectAfterClosed,
		AssessmentTypeAllowReviewBeforeSubmit:     m.AssessmentTypeAllowReviewBeforeSubmit,
		AssessmentTypeRequireCompleteAttempt:      m.AssessmentTypeRequireCompleteAttempt,
		AssessmentTypeShowDetailsAfterAllAttempts: m.AssessmentTypeShowDetailsAfterAllAttempts,

		AssessmentTypeIsActive: m.AssessmentTypeIsActive,
		AssessmentTypeIsGraded: m.AssessmentTypeIsGraded,

		AssessmentTypeCreatedAt: m.AssessmentTypeCreatedAt,
		AssessmentTypeUpdatedAt: m.AssessmentTypeUpdatedAt,
	}
}

func FromModels(items []model.AssessmentTypeModel) []AssessmentTypeResponse {
	out := make([]AssessmentTypeResponse, 0, len(items))
	for _, it := range items {
		out = append(out, FromModel(it))
	}
	return out
}

// ==============================
//  COMPACT DTO
// ==============================

// Dipakai untuk mode=compact (dropdown / list ringan)
type AssessmentTypeCompactResponse struct {
	AssessmentTypeID       uuid.UUID `json:"assessment_type_id"`
	AssessmentTypeSchoolID uuid.UUID `json:"assessment_type_school_id"`

	AssessmentTypeKey           string  `json:"assessment_type_key"`
	AssessmentTypeName          string  `json:"assessment_type_name"`
	AssessmentTypeWeightPercent float64 `json:"assessment_type_weight_percent"`

	AssessmentTypeCategory string `json:"assessment_type"`

	AssessmentTypeIsActive bool `json:"assessment_type_is_active"`
	AssessmentTypeIsGraded bool `json:"assessment_type_is_graded"`
}

// Single model → compact
func FromModelCompact(m model.AssessmentTypeModel) AssessmentTypeCompactResponse {
	return AssessmentTypeCompactResponse{
		AssessmentTypeID:            m.AssessmentTypeID,
		AssessmentTypeSchoolID:      m.AssessmentTypeSchoolID,
		AssessmentTypeKey:           strings.TrimSpace(m.AssessmentTypeKey),
		AssessmentTypeName:          strings.TrimSpace(m.AssessmentTypeName),
		AssessmentTypeWeightPercent: m.AssessmentTypeWeightPercent,
		AssessmentTypeCategory:      m.AssessmentTypeCategory,
		AssessmentTypeIsActive:      m.AssessmentTypeIsActive,
		AssessmentTypeIsGraded:      m.AssessmentTypeIsGraded,
	}
}

// Slice model → slice compact
func FromModelsCompact(items []model.AssessmentTypeModel) []AssessmentTypeCompactResponse {
	out := make([]AssessmentTypeCompactResponse, 0, len(items))
	for _, it := range items {
		out = append(out, FromModelCompact(it))
	}
	return out
}

// ==============================
//  MAPPER DENGAN TIMEZONE SEKOLAH
// ==============================

// Versi yang awareness terhadap timezone sekolah.
// CreatedAt/UpdatedAt akan dikonversi dengan dbtime.ToSchoolTime(c, ...)
func FromModelWithCtx(c *fiber.Ctx, m model.AssessmentTypeModel) AssessmentTypeResponse {
	resp := FromModel(m)

	// override 2 field waktu pakai timezone sekolah
	resp.AssessmentTypeCreatedAt = dbtime.ToSchoolTime(c, m.AssessmentTypeCreatedAt)
	resp.AssessmentTypeUpdatedAt = dbtime.ToSchoolTime(c, m.AssessmentTypeUpdatedAt)

	return resp
}

func FromModelsWithCtx(c *fiber.Ctx, items []model.AssessmentTypeModel) []AssessmentTypeResponse {
	out := make([]AssessmentTypeResponse, 0, len(items))
	for _, it := range items {
		out = append(out, FromModelWithCtx(c, it))
	}
	return out
}
