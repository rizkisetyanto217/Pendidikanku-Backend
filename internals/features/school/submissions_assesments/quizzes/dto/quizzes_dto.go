// file: internals/features/school/submissions_assesments/quizzes/dto/quiz_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	model "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
)

/* ==============================
   Helpers
============================== */

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

/*
==============================

	Helper: Tri-state updater
	- Absent  : tidak diupdate
	- null    : set kolom ke NULL
	- value   : set kolom ke value

==============================
*/
type UpdateField[T any] struct {
	set   bool
	null  bool
	value T
}

func (f *UpdateField[T]) UnmarshalJSON(b []byte) error {
	f.set = true
	if string(b) == "null" {
		f.null = true
		var zero T
		f.value = zero
		return nil
	}
	return json.Unmarshal(b, &f.value)
}

func (f UpdateField[T]) ShouldUpdate() bool { return f.set }
func (f UpdateField[T]) IsNull() bool       { return f.set && f.null }
func (f UpdateField[T]) Val() T             { return f.value }

/* ==============================
   CREATE (POST /quizzes)
============================== */

type CreateQuizRequest struct {
	// Tenant & relasi
	QuizSchoolID     uuid.UUID  `json:"quiz_school_id" validate:"required,uuid4"`
	QuizAssessmentID *uuid.UUID `json:"quiz_assessment_id" validate:"omitempty,uuid4"`

	// Identitas
	QuizSlug        *string `json:"quiz_slug" validate:"omitempty,max=160"`
	QuizTitle       string  `json:"quiz_title" validate:"required,max=180"`
	QuizDescription *string `json:"quiz_description" validate:"omitempty"`

	// Pengaturan dasar
	QuizIsPublished  *bool `json:"quiz_is_published" validate:"omitempty"`
	QuizTimeLimitSec *int  `json:"quiz_time_limit_sec" validate:"omitempty,gte=0"`

	// Snapshot behaviour & scoring (opsional, kalau kosong pakai default)
	QuizShuffleQuestionsSnapshot            *bool   `json:"quiz_shuffle_questions_snapshot" validate:"omitempty"`
	QuizShuffleOptionsSnapshot              *bool   `json:"quiz_shuffle_options_snapshot" validate:"omitempty"`
	QuizShowCorrectAfterSubmitSnapshot      *bool   `json:"quiz_show_correct_after_submit_snapshot" validate:"omitempty"`
	QuizStrictModeSnapshot                  *bool   `json:"quiz_strict_mode_snapshot" validate:"omitempty"`
	QuizTimeLimitMinSnapshot                *int    `json:"quiz_time_limit_min_snapshot" validate:"omitempty,gte=0"`
	QuizRequireLoginSnapshot                *bool   `json:"quiz_require_login_snapshot" validate:"omitempty"`
	QuizShowScoreAfterSubmitSnapshot        *bool   `json:"quiz_show_score_after_submit_snapshot" validate:"omitempty"`
	QuizShowCorrectAfterClosedSnapshot      *bool   `json:"quiz_show_correct_after_closed_snapshot" validate:"omitempty"`
	QuizAllowReviewBeforeSubmitSnapshot     *bool   `json:"quiz_allow_review_before_submit_snapshot" validate:"omitempty"`
	QuizRequireCompleteAttemptSnapshot      *bool   `json:"quiz_require_complete_attempt_snapshot" validate:"omitempty"`
	QuizShowDetailsAfterAllAttemptsSnapshot *bool   `json:"quiz_show_details_after_all_attempts_snapshot" validate:"omitempty"`
	QuizAttemptsAllowedSnapshot             *int    `json:"quiz_attempts_allowed_snapshot" validate:"omitempty,gte=1"`
	QuizScoreAggregationModeSnapshot        *string `json:"quiz_score_aggregation_mode_snapshot" validate:"omitempty,oneof=latest highest average first"`
}

// ToModel: builder model dari payload Create (timestamps oleh GORM)
func (r *CreateQuizRequest) ToModel() *model.QuizModel {
	// pub flag
	isPub := false
	if r.QuizIsPublished != nil {
		isPub = *r.QuizIsPublished
	}

	// default behaviour & scoring (sync dengan DDL)
	shuffleQuestions := false
	if r.QuizShuffleQuestionsSnapshot != nil {
		shuffleQuestions = *r.QuizShuffleQuestionsSnapshot
	}

	shuffleOptions := false
	if r.QuizShuffleOptionsSnapshot != nil {
		shuffleOptions = *r.QuizShuffleOptionsSnapshot
	}

	showCorrectAfterSubmit := true
	if r.QuizShowCorrectAfterSubmitSnapshot != nil {
		showCorrectAfterSubmit = *r.QuizShowCorrectAfterSubmitSnapshot
	}

	strictMode := false
	if r.QuizStrictModeSnapshot != nil {
		strictMode = *r.QuizStrictModeSnapshot
	}

	requireLogin := true
	if r.QuizRequireLoginSnapshot != nil {
		requireLogin = *r.QuizRequireLoginSnapshot
	}

	showScoreAfterSubmit := true
	if r.QuizShowScoreAfterSubmitSnapshot != nil {
		showScoreAfterSubmit = *r.QuizShowScoreAfterSubmitSnapshot
	}

	showCorrectAfterClosed := false
	if r.QuizShowCorrectAfterClosedSnapshot != nil {
		showCorrectAfterClosed = *r.QuizShowCorrectAfterClosedSnapshot
	}

	allowReviewBeforeSubmit := true
	if r.QuizAllowReviewBeforeSubmitSnapshot != nil {
		allowReviewBeforeSubmit = *r.QuizAllowReviewBeforeSubmitSnapshot
	}

	requireCompleteAttempt := true
	if r.QuizRequireCompleteAttemptSnapshot != nil {
		requireCompleteAttempt = *r.QuizRequireCompleteAttemptSnapshot
	}

	showDetailsAfterAllAttempts := false
	if r.QuizShowDetailsAfterAllAttemptsSnapshot != nil {
		showDetailsAfterAllAttempts = *r.QuizShowDetailsAfterAllAttemptsSnapshot
	}

	attemptsAllowed := 1
	if r.QuizAttemptsAllowedSnapshot != nil {
		attemptsAllowed = *r.QuizAttemptsAllowedSnapshot
	}

	aggMode := "latest"
	if r.QuizScoreAggregationModeSnapshot != nil {
		if v := strings.TrimSpace(*r.QuizScoreAggregationModeSnapshot); v != "" {
			aggMode = v
		}
	}

	return &model.QuizModel{
		QuizSchoolID:     r.QuizSchoolID,
		QuizAssessmentID: r.QuizAssessmentID,

		QuizSlug:        trimPtr(r.QuizSlug),
		QuizTitle:       strings.TrimSpace(r.QuizTitle),
		QuizDescription: trimPtr(r.QuizDescription),

		QuizIsPublished:  isPub,
		QuizTimeLimitSec: r.QuizTimeLimitSec,

		// snapshot behaviour
		QuizShuffleQuestionsSnapshot:            shuffleQuestions,
		QuizShuffleOptionsSnapshot:              shuffleOptions,
		QuizShowCorrectAfterSubmitSnapshot:      showCorrectAfterSubmit,
		QuizStrictModeSnapshot:                  strictMode,
		QuizTimeLimitMinSnapshot:                r.QuizTimeLimitMinSnapshot,
		QuizRequireLoginSnapshot:                requireLogin,
		QuizShowScoreAfterSubmitSnapshot:        showScoreAfterSubmit,
		QuizShowCorrectAfterClosedSnapshot:      showCorrectAfterClosed,
		QuizAllowReviewBeforeSubmitSnapshot:     allowReviewBeforeSubmit,
		QuizRequireCompleteAttemptSnapshot:      requireCompleteAttempt,
		QuizShowDetailsAfterAllAttemptsSnapshot: showDetailsAfterAllAttempts,

		QuizAttemptsAllowedSnapshot:      attemptsAllowed,
		QuizScoreAggregationModeSnapshot: aggMode,
	}
}

/* ==============================
   PATCH (PATCH /quizzes/:id)
   - gunakan UpdateField agar bisa null/skip/value
============================== */

type PatchQuizRequest struct {
	QuizAssessmentID UpdateField[uuid.UUID] `json:"quiz_assessment_id"` // nullable

	QuizSlug        UpdateField[string] `json:"quiz_slug"`        // nullable
	QuizTitle       UpdateField[string] `json:"quiz_title"`       // NOT NULL di DB (abaikan jika null/empty)
	QuizDescription UpdateField[string] `json:"quiz_description"` // nullable

	QuizIsPublished  UpdateField[bool] `json:"quiz_is_published"`
	QuizTimeLimitSec UpdateField[int]  `json:"quiz_time_limit_sec"` // nullable

	// behaviour & scoring config
	QuizShuffleQuestionsSnapshot            UpdateField[bool]   `json:"quiz_shuffle_questions_snapshot"`
	QuizShuffleOptionsSnapshot              UpdateField[bool]   `json:"quiz_shuffle_options_snapshot"`
	QuizShowCorrectAfterSubmitSnapshot      UpdateField[bool]   `json:"quiz_show_correct_after_submit_snapshot"`
	QuizStrictModeSnapshot                  UpdateField[bool]   `json:"quiz_strict_mode_snapshot"`
	QuizTimeLimitMinSnapshot                UpdateField[int]    `json:"quiz_time_limit_min_snapshot"` // nullable
	QuizRequireLoginSnapshot                UpdateField[bool]   `json:"quiz_require_login_snapshot"`
	QuizShowScoreAfterSubmitSnapshot        UpdateField[bool]   `json:"quiz_show_score_after_submit_snapshot"`
	QuizShowCorrectAfterClosedSnapshot      UpdateField[bool]   `json:"quiz_show_correct_after_closed_snapshot"`
	QuizAllowReviewBeforeSubmitSnapshot     UpdateField[bool]   `json:"quiz_allow_review_before_submit_snapshot"`
	QuizRequireCompleteAttemptSnapshot      UpdateField[bool]   `json:"quiz_require_complete_attempt_snapshot"`
	QuizShowDetailsAfterAllAttemptsSnapshot UpdateField[bool]   `json:"quiz_show_details_after_all_attempts_snapshot"`
	QuizAttemptsAllowedSnapshot             UpdateField[int]    `json:"quiz_attempts_allowed_snapshot"`
	QuizScoreAggregationModeSnapshot        UpdateField[string] `json:"quiz_score_aggregation_mode_snapshot"`
}

// ToUpdates: map untuk gorm.Model(&m).Updates(...)
func (p *PatchQuizRequest) ToUpdates() map[string]any {
	u := make(map[string]any, 24)

	// quiz_assessment_id (nullable)
	if p.QuizAssessmentID.ShouldUpdate() {
		if p.QuizAssessmentID.IsNull() {
			u["quiz_assessment_id"] = gorm.Expr("NULL")
		} else {
			u["quiz_assessment_id"] = p.QuizAssessmentID.Val()
		}
	}

	// quiz_slug (nullable)
	if p.QuizSlug.ShouldUpdate() {
		if p.QuizSlug.IsNull() {
			u["quiz_slug"] = gorm.Expr("NULL")
		} else {
			slug := strings.TrimSpace(p.QuizSlug.Val())
			if slug == "" {
				u["quiz_slug"] = gorm.Expr("NULL")
			} else {
				u["quiz_slug"] = slug
			}
		}
	}

	// quiz_title (NOT NULL) -> abaikan jika null/empty
	if p.QuizTitle.ShouldUpdate() && !p.QuizTitle.IsNull() {
		title := strings.TrimSpace(p.QuizTitle.Val())
		if title != "" {
			u["quiz_title"] = title
		}
	}

	// quiz_description (nullable)
	if p.QuizDescription.ShouldUpdate() {
		if p.QuizDescription.IsNull() {
			u["quiz_description"] = gorm.Expr("NULL")
		} else {
			desc := strings.TrimSpace(p.QuizDescription.Val())
			if desc == "" {
				u["quiz_description"] = gorm.Expr("NULL")
			} else {
				u["quiz_description"] = &desc
			}
		}
	}

	// quiz_is_published (bool)
	if p.QuizIsPublished.ShouldUpdate() && !p.QuizIsPublished.IsNull() {
		u["quiz_is_published"] = p.QuizIsPublished.Val()
	}

	// quiz_time_limit_sec (nullable int)
	if p.QuizTimeLimitSec.ShouldUpdate() {
		if p.QuizTimeLimitSec.IsNull() {
			u["quiz_time_limit_sec"] = gorm.Expr("NULL")
		} else {
			u["quiz_time_limit_sec"] = p.QuizTimeLimitSec.Val()
		}
	}

	// ==============================
	// behaviour & scoring
	// ==============================

	if p.QuizShuffleQuestionsSnapshot.ShouldUpdate() && !p.QuizShuffleQuestionsSnapshot.IsNull() {
		u["quiz_shuffle_questions_snapshot"] = p.QuizShuffleQuestionsSnapshot.Val()
	}

	if p.QuizShuffleOptionsSnapshot.ShouldUpdate() && !p.QuizShuffleOptionsSnapshot.IsNull() {
		u["quiz_shuffle_options_snapshot"] = p.QuizShuffleOptionsSnapshot.Val()
	}

	if p.QuizShowCorrectAfterSubmitSnapshot.ShouldUpdate() && !p.QuizShowCorrectAfterSubmitSnapshot.IsNull() {
		u["quiz_show_correct_after_submit_snapshot"] = p.QuizShowCorrectAfterSubmitSnapshot.Val()
	}

	if p.QuizStrictModeSnapshot.ShouldUpdate() && !p.QuizStrictModeSnapshot.IsNull() {
		u["quiz_strict_mode_snapshot"] = p.QuizStrictModeSnapshot.Val()
	}

	// quiz_time_limit_min_snapshot (nullable int)
	if p.QuizTimeLimitMinSnapshot.ShouldUpdate() {
		if p.QuizTimeLimitMinSnapshot.IsNull() {
			u["quiz_time_limit_min_snapshot"] = gorm.Expr("NULL")
		} else {
			u["quiz_time_limit_min_snapshot"] = p.QuizTimeLimitMinSnapshot.Val()
		}
	}

	if p.QuizRequireLoginSnapshot.ShouldUpdate() && !p.QuizRequireLoginSnapshot.IsNull() {
		u["quiz_require_login_snapshot"] = p.QuizRequireLoginSnapshot.Val()
	}

	if p.QuizShowScoreAfterSubmitSnapshot.ShouldUpdate() && !p.QuizShowScoreAfterSubmitSnapshot.IsNull() {
		u["quiz_show_score_after_submit_snapshot"] = p.QuizShowScoreAfterSubmitSnapshot.Val()
	}

	if p.QuizShowCorrectAfterClosedSnapshot.ShouldUpdate() && !p.QuizShowCorrectAfterClosedSnapshot.IsNull() {
		u["quiz_show_correct_after_closed_snapshot"] = p.QuizShowCorrectAfterClosedSnapshot.Val()
	}

	if p.QuizAllowReviewBeforeSubmitSnapshot.ShouldUpdate() && !p.QuizAllowReviewBeforeSubmitSnapshot.IsNull() {
		u["quiz_allow_review_before_submit_snapshot"] = p.QuizAllowReviewBeforeSubmitSnapshot.Val()
	}

	if p.QuizRequireCompleteAttemptSnapshot.ShouldUpdate() && !p.QuizRequireCompleteAttemptSnapshot.IsNull() {
		u["quiz_require_complete_attempt_snapshot"] = p.QuizRequireCompleteAttemptSnapshot.Val()
	}

	if p.QuizShowDetailsAfterAllAttemptsSnapshot.ShouldUpdate() && !p.QuizShowDetailsAfterAllAttemptsSnapshot.IsNull() {
		u["quiz_show_details_after_all_attempts_snapshot"] = p.QuizShowDetailsAfterAllAttemptsSnapshot.Val()
	}

	if p.QuizAttemptsAllowedSnapshot.ShouldUpdate() && !p.QuizAttemptsAllowedSnapshot.IsNull() {
		u["quiz_attempts_allowed_snapshot"] = p.QuizAttemptsAllowedSnapshot.Val()
	}

	if p.QuizScoreAggregationModeSnapshot.ShouldUpdate() && !p.QuizScoreAggregationModeSnapshot.IsNull() {
		mode := strings.TrimSpace(p.QuizScoreAggregationModeSnapshot.Val())
		if mode != "" {
			u["quiz_score_aggregation_mode_snapshot"] = mode
		}
	}

	return u
}

/* ==============================
   QUERY (GET /quizzes)
============================== */

type ListQuizzesQuery struct {
	// filter dasar
	SchoolID     *uuid.UUID `query:"school_id" validate:"omitempty,uuid4"`
	ID           *uuid.UUID `query:"id" validate:"omitempty,uuid4"` // quiz_id
	AssessmentID *uuid.UUID `query:"assessment_id" validate:"omitempty,uuid4"`

	// filter by slug (exact)
	Slug *string `query:"slug" validate:"omitempty,max=160"`

	IsPublished *bool  `query:"is_published" validate:"omitempty"`
	Q           string `query:"q" validate:"omitempty,max=120"`

	// pagination & sorting
	Page    int    `query:"page" validate:"omitempty,gte=0"`
	PerPage int    `query:"per_page" validate:"omitempty,gte=0,lte=200"`
	Sort    string `query:"sort" validate:"omitempty,oneof=created_at desc_created_at title desc_title published desc_published"`

	// embedding questions
	WithQuestions      bool   `query:"with_questions"`
	QuestionsLimit     int    `query:"questions_limit" validate:"omitempty,min=1,max=200"`
	QuestionsOrder     string `query:"questions_order" validate:"omitempty,oneof=created_at desc_created_at"`
	WithQuestionsCount bool   `query:"with_questions_count"`
}

/* ==============================
   RESPONSE DTOs
============================== */

type QuizResponse struct {
	QuizID           uuid.UUID  `json:"quiz_id"`
	QuizSchoolID     uuid.UUID  `json:"quiz_school_id"`
	QuizAssessmentID *uuid.UUID `json:"quiz_assessment_id,omitempty"`

	QuizSlug *string `json:"quiz_slug,omitempty"`

	QuizTitle        string  `json:"quiz_title"`
	QuizDescription  *string `json:"quiz_description,omitempty"`
	QuizIsPublished  bool    `json:"quiz_is_published"`
	QuizTimeLimitSec *int    `json:"quiz_time_limit_sec,omitempty"`

	// behaviour & scoring snapshot
	QuizShuffleQuestionsSnapshot            bool   `json:"quiz_shuffle_questions_snapshot"`
	QuizShuffleOptionsSnapshot              bool   `json:"quiz_shuffle_options_snapshot"`
	QuizShowCorrectAfterSubmitSnapshot      bool   `json:"quiz_show_correct_after_submit_snapshot"`
	QuizStrictModeSnapshot                  bool   `json:"quiz_strict_mode_snapshot"`
	QuizTimeLimitMinSnapshot                *int   `json:"quiz_time_limit_min_snapshot,omitempty"`
	QuizRequireLoginSnapshot                bool   `json:"quiz_require_login_snapshot"`
	QuizShowScoreAfterSubmitSnapshot        bool   `json:"quiz_show_score_after_submit_snapshot"`
	QuizShowCorrectAfterClosedSnapshot      bool   `json:"quiz_show_correct_after_closed_snapshot"`
	QuizAllowReviewBeforeSubmitSnapshot     bool   `json:"quiz_allow_review_before_submit_snapshot"`
	QuizRequireCompleteAttemptSnapshot      bool   `json:"quiz_require_complete_attempt_snapshot"`
	QuizShowDetailsAfterAllAttemptsSnapshot bool   `json:"quiz_show_details_after_all_attempts_snapshot"`
	QuizAttemptsAllowedSnapshot             int    `json:"quiz_attempts_allowed_snapshot"`
	QuizScoreAggregationModeSnapshot        string `json:"quiz_score_aggregation_mode_snapshot"`

	QuizCreatedAt time.Time  `json:"quiz_created_at"`
	QuizUpdatedAt time.Time  `json:"quiz_updated_at"`
	QuizDeletedAt *time.Time `json:"quiz_deleted_at,omitempty"`

	Questions      []*QuizQuestionResponse `json:"questions,omitempty"`
	QuestionsCount *int                    `json:"questions_count,omitempty"`
}

type ListQuizResponse struct {
	Data       []QuizResponse `json:"data"`
	Pagination any            `json:"pagination"`
}

/* ==============================
   MAPPERS
============================== */

func FromModel(m *model.QuizModel) QuizResponse {
	var deletedAt *time.Time
	if m.QuizDeletedAt.Valid {
		t := m.QuizDeletedAt.Time
		deletedAt = &t
	}
	return QuizResponse{
		QuizID:           m.QuizID,
		QuizSchoolID:     m.QuizSchoolID,
		QuizAssessmentID: m.QuizAssessmentID,

		QuizSlug:         m.QuizSlug,
		QuizTitle:        m.QuizTitle,
		QuizDescription:  m.QuizDescription,
		QuizIsPublished:  m.QuizIsPublished,
		QuizTimeLimitSec: m.QuizTimeLimitSec,

		QuizShuffleQuestionsSnapshot:            m.QuizShuffleQuestionsSnapshot,
		QuizShuffleOptionsSnapshot:              m.QuizShuffleOptionsSnapshot,
		QuizShowCorrectAfterSubmitSnapshot:      m.QuizShowCorrectAfterSubmitSnapshot,
		QuizStrictModeSnapshot:                  m.QuizStrictModeSnapshot,
		QuizTimeLimitMinSnapshot:                m.QuizTimeLimitMinSnapshot,
		QuizRequireLoginSnapshot:                m.QuizRequireLoginSnapshot,
		QuizShowScoreAfterSubmitSnapshot:        m.QuizShowScoreAfterSubmitSnapshot,
		QuizShowCorrectAfterClosedSnapshot:      m.QuizShowCorrectAfterClosedSnapshot,
		QuizAllowReviewBeforeSubmitSnapshot:     m.QuizAllowReviewBeforeSubmitSnapshot,
		QuizRequireCompleteAttemptSnapshot:      m.QuizRequireCompleteAttemptSnapshot,
		QuizShowDetailsAfterAllAttemptsSnapshot: m.QuizShowDetailsAfterAllAttemptsSnapshot,
		QuizAttemptsAllowedSnapshot:             m.QuizAttemptsAllowedSnapshot,
		QuizScoreAggregationModeSnapshot:        m.QuizScoreAggregationModeSnapshot,

		QuizCreatedAt: m.QuizCreatedAt,
		QuizUpdatedAt: m.QuizUpdatedAt,
		QuizDeletedAt: deletedAt,
	}
}

func FromModels(ms []model.QuizModel) []QuizResponse {
	out := make([]QuizResponse, 0, len(ms))
	for i := range ms {
		out = append(out, FromModel(&ms[i]))
	}
	return out
}

func FromModelWithQuestions(m *model.QuizModel) QuizResponse {
	resp := FromModel(m)
	if len(m.Questions) > 0 {
		arr := make([]*QuizQuestionResponse, 0, len(m.Questions))
		for i := range m.Questions {
			arr = append(arr, FromModelQuizQuestion(&m.Questions[i]))
		}
		resp.Questions = arr
	}
	return resp
}
