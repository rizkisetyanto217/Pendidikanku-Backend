// file: internals/features/school/assessments/dto/assessment_type_dto.go
package dto

import (
	"strings"
	"time"

	model "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"

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

	AssessmentTypeIsActive *bool `json:"assessment_type_is_active" validate:"omitempty"`
	AssessmentTypeIsGraded *bool `json:"assessment_type_is_graded" validate:"omitempty"` // 👈 baru

	// ===== Default quiz settings (optional di request; pakai default kalau null) =====

	AssessmentTypeShuffleQuestions       *bool `json:"assessment_type_shuffle_questions" validate:"omitempty"`
	AssessmentTypeShuffleOptions         *bool `json:"assessment_type_shuffle_options" validate:"omitempty"`
	AssessmentTypeShowCorrectAfterSubmit *bool `json:"assessment_type_show_correct_after_submit" validate:"omitempty"`
	AssessmentTypeOneQuestionPerPage     *bool `json:"assessment_type_one_question_per_page" validate:"omitempty"`
	AssessmentTypeTimeLimitMin           *int  `json:"assessment_type_time_limit_min" validate:"omitempty,min=0"`
	AssessmentTypeAttemptsAllowed        *int  `json:"assessment_type_attempts_allowed" validate:"omitempty,min=1"`
	AssessmentTypeRequireLogin           *bool `json:"assessment_type_require_login" validate:"omitempty"`
	AssessmentTypePreventBackNavigation  *bool `json:"assessment_type_prevent_back_navigation" validate:"omitempty"`
}

// Patch (PATCH /assessment-types/:id) — partial update
type PatchAssessmentTypeRequest struct {
	AssessmentTypeName          *string  `json:"assessment_type_name" validate:"omitempty,max=120"`
	AssessmentTypeWeightPercent *float64 `json:"assessment_type_weight_percent" validate:"omitempty,gte=0,lte=100"`
	AssessmentTypeIsActive      *bool    `json:"assessment_type_is_active" validate:"omitempty"`
	AssessmentTypeIsGraded      *bool    `json:"assessment_type_is_graded" validate:"omitempty"` // 👈 baru

	AssessmentTypeShuffleQuestions       *bool `json:"assessment_type_shuffle_questions" validate:"omitempty"`
	AssessmentTypeShuffleOptions         *bool `json:"assessment_type_shuffle_options" validate:"omitempty"`
	AssessmentTypeShowCorrectAfterSubmit *bool `json:"assessment_type_show_correct_after_submit" validate:"omitempty"`
	AssessmentTypeOneQuestionPerPage     *bool `json:"assessment_type_one_question_per_page" validate:"omitempty"`
	AssessmentTypeTimeLimitMin           *int  `json:"assessment_type_time_limit_min" validate:"omitempty,min=0"`
	AssessmentTypeAttemptsAllowed        *int  `json:"assessment_type_attempts_allowed" validate:"omitempty,min=1"`
	AssessmentTypeRequireLogin           *bool `json:"assessment_type_require_login" validate:"omitempty"`
	AssessmentTypePreventBackNavigation  *bool `json:"assessment_type_prevent_back_navigation" validate:"omitempty"`
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

	// Default quiz settings (dibaca frontend buat seed QuizSettings)
	AssessmentTypeShuffleQuestions       bool `json:"assessment_type_shuffle_questions"`
	AssessmentTypeShuffleOptions         bool `json:"assessment_type_shuffle_options"`
	AssessmentTypeShowCorrectAfterSubmit bool `json:"assessment_type_show_correct_after_submit"`
	AssessmentTypeOneQuestionPerPage     bool `json:"assessment_type_one_question_per_page"`
	AssessmentTypeTimeLimitMin           *int `json:"assessment_type_time_limit_min,omitempty"`
	AssessmentTypeAttemptsAllowed        int  `json:"assessment_type_attempts_allowed"`
	AssessmentTypeRequireLogin           bool `json:"assessment_type_require_login"`
	AssessmentTypePreventBackNavigation  bool `json:"assessment_type_prevent_back_navigation"`

	AssessmentTypeIsActive bool `json:"assessment_type_is_active"`
	AssessmentTypeIsGraded bool `json:"assessment_type_is_graded"` // 👈 baru

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
	// Default active = true agar tidak menimpa default DB dengan false (zero value)
	isActive := true
	if r.AssessmentTypeIsActive != nil {
		isActive = *r.AssessmentTypeIsActive
	}

	// Default: secara umum assessment type adalah graded
	isGraded := true
	if r.AssessmentTypeIsGraded != nil {
		isGraded = *r.AssessmentTypeIsGraded
	}

	// Default untuk quiz settings — harus sync dengan default di SQL
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

	onePerPage := false
	if r.AssessmentTypeOneQuestionPerPage != nil {
		onePerPage = *r.AssessmentTypeOneQuestionPerPage
	}

	var timeLimit *int
	if r.AssessmentTypeTimeLimitMin != nil {
		timeLimit = r.AssessmentTypeTimeLimitMin
	}

	attempts := 1
	if r.AssessmentTypeAttemptsAllowed != nil {
		attempts = *r.AssessmentTypeAttemptsAllowed
	}

	requireLogin := true
	if r.AssessmentTypeRequireLogin != nil {
		requireLogin = *r.AssessmentTypeRequireLogin
	}

	preventBack := false
	if r.AssessmentTypePreventBackNavigation != nil {
		preventBack = *r.AssessmentTypePreventBackNavigation
	}

	return model.AssessmentTypeModel{
		AssessmentTypeSchoolID:      r.AssessmentTypeSchoolID,
		AssessmentTypeKey:           r.AssessmentTypeKey,
		AssessmentTypeName:          r.AssessmentTypeName,
		AssessmentTypeWeightPercent: r.AssessmentTypeWeightPercent,

		AssessmentTypeShuffleQuestions:       shuffleQuestions,
		AssessmentTypeShuffleOptions:         shuffleOptions,
		AssessmentTypeShowCorrectAfterSubmit: showCorrect,
		AssessmentTypeOneQuestionPerPage:     onePerPage,
		AssessmentTypeTimeLimitMin:           timeLimit,
		AssessmentTypeAttemptsAllowed:        attempts,
		AssessmentTypeRequireLogin:           requireLogin,
		AssessmentTypePreventBackNavigation:  preventBack,

		AssessmentTypeIsActive: isActive,
		AssessmentTypeIsGraded: isGraded, // 👈 baru masuk model
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
	if p.AssessmentTypeIsGraded != nil { // 👈 baru
		m.AssessmentTypeIsGraded = *p.AssessmentTypeIsGraded
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
	if p.AssessmentTypeOneQuestionPerPage != nil {
		m.AssessmentTypeOneQuestionPerPage = *p.AssessmentTypeOneQuestionPerPage
	}
	if p.AssessmentTypeTimeLimitMin != nil {
		// Catatan: dengan desain ini kita belum bisa "clear" jadi NULL lewat PATCH (hanya ubah nilai).
		// Kalau butuh clear, nanti bisa ditambah flag khusus.
		m.AssessmentTypeTimeLimitMin = p.AssessmentTypeTimeLimitMin
	}
	if p.AssessmentTypeAttemptsAllowed != nil {
		m.AssessmentTypeAttemptsAllowed = *p.AssessmentTypeAttemptsAllowed
	}
	if p.AssessmentTypeRequireLogin != nil {
		m.AssessmentTypeRequireLogin = *p.AssessmentTypeRequireLogin
	}
	if p.AssessmentTypePreventBackNavigation != nil {
		m.AssessmentTypePreventBackNavigation = *p.AssessmentTypePreventBackNavigation
	}
}

func FromModel(m model.AssessmentTypeModel) AssessmentTypeResponse {
	return AssessmentTypeResponse{
		AssessmentTypeID:            m.AssessmentTypeID,
		AssessmentTypeSchoolID:      m.AssessmentTypeSchoolID,
		AssessmentTypeKey:           m.AssessmentTypeKey,
		AssessmentTypeName:          m.AssessmentTypeName,
		AssessmentTypeWeightPercent: m.AssessmentTypeWeightPercent,

		AssessmentTypeShuffleQuestions:       m.AssessmentTypeShuffleQuestions,
		AssessmentTypeShuffleOptions:         m.AssessmentTypeShuffleOptions,
		AssessmentTypeShowCorrectAfterSubmit: m.AssessmentTypeShowCorrectAfterSubmit,
		AssessmentTypeOneQuestionPerPage:     m.AssessmentTypeOneQuestionPerPage,
		AssessmentTypeTimeLimitMin:           m.AssessmentTypeTimeLimitMin,
		AssessmentTypeAttemptsAllowed:        m.AssessmentTypeAttemptsAllowed,
		AssessmentTypeRequireLogin:           m.AssessmentTypeRequireLogin,
		AssessmentTypePreventBackNavigation:  m.AssessmentTypePreventBackNavigation,

		AssessmentTypeIsActive: m.AssessmentTypeIsActive,
		AssessmentTypeIsGraded: m.AssessmentTypeIsGraded, // 👈 baru ikut ke response

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
