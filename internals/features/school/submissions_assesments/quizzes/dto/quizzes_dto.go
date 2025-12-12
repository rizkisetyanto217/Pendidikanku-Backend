// file: internals/features/school/submissions_assesments/quizzes/dto/quiz_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	model "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	"madinahsalam_backend/internals/helpers/dbtime"
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
	QuizSchoolID         uuid.UUID  `json:"quiz_school_id"`
	QuizAssessmentID     *uuid.UUID `json:"quiz_assessment_id" validate:"omitempty,uuid4"`
	QuizAssessmentTypeID *uuid.UUID `json:"quiz_assessment_type_id" validate:"omitempty,uuid4"`

	// Identitas
	QuizSlug        *string `json:"quiz_slug" validate:"omitempty,max=160"`
	QuizTitle       string  `json:"quiz_title" validate:"required,max=180"`
	QuizDescription *string `json:"quiz_description" validate:"omitempty"`

	// Pengaturan dasar
	QuizIsPublished  *bool `json:"quiz_is_published" validate:"omitempty"`
	QuizTimeLimitSec *int  `json:"quiz_time_limit_sec" validate:"omitempty,gte=0"`

	// Remedial flags (opsional, biasanya diisi dari backend saat clone)
	QuizIsRemedial    *bool      `json:"quiz_is_remedial,omitempty" validate:"omitempty"`
	QuizParentQuizID  *uuid.UUID `json:"quiz_parent_quiz_id,omitempty" validate:"omitempty,uuid4"`
	QuizRemedialRound *int       `json:"quiz_remedial_round,omitempty" validate:"omitempty,gte=1"`
}

// ToModel: builder model dari payload Create (timestamps oleh GORM)
func (r *CreateQuizRequest) ToModel() *model.QuizModel {
	// publish flag
	isPub := false
	if r.QuizIsPublished != nil {
		isPub = *r.QuizIsPublished
	}

	// remedial flag
	isRemedial := false
	if r.QuizIsRemedial != nil {
		isRemedial = *r.QuizIsRemedial
	}

	return &model.QuizModel{
		QuizSchoolID:         r.QuizSchoolID,
		QuizAssessmentID:     r.QuizAssessmentID,
		QuizAssessmentTypeID: r.QuizAssessmentTypeID,

		QuizSlug:        trimPtr(r.QuizSlug),
		QuizTitle:       strings.TrimSpace(r.QuizTitle),
		QuizDescription: trimPtr(r.QuizDescription),

		QuizIsPublished:  isPub,
		QuizTimeLimitSec: r.QuizTimeLimitSec,

		// Remedial flags
		QuizIsRemedial:    isRemedial,
		QuizParentQuizID:  r.QuizParentQuizID,
		QuizRemedialRound: r.QuizRemedialRound,
		// QuizTotalQuestions pakai default 0 dari DB / diupdate dari service questions
	}
}

/* ==============================
   PATCH (PATCH /quizzes/:id)
   - gunakan UpdateField agar bisa null/skip/value
============================== */

type PatchQuizRequest struct {
	QuizAssessmentID     UpdateField[uuid.UUID] `json:"quiz_assessment_id"`      // nullable
	QuizAssessmentTypeID UpdateField[uuid.UUID] `json:"quiz_assessment_type_id"` // nullable

	QuizSlug        UpdateField[string] `json:"quiz_slug"`        // nullable
	QuizTitle       UpdateField[string] `json:"quiz_title"`       // NOT NULL di DB (abaikan jika null/empty)
	QuizDescription UpdateField[string] `json:"quiz_description"` // nullable

	QuizIsPublished  UpdateField[bool] `json:"quiz_is_published"`
	QuizTimeLimitSec UpdateField[int]  `json:"quiz_time_limit_sec"` // nullable

	// Remedial flags
	QuizIsRemedial    UpdateField[bool]      `json:"quiz_is_remedial"`
	QuizParentQuizID  UpdateField[uuid.UUID] `json:"quiz_parent_quiz_id"`
	QuizRemedialRound UpdateField[int]       `json:"quiz_remedial_round"`
}

// ToUpdates: map untuk gorm.Model(&m).Updates(...)
func (p *PatchQuizRequest) ToUpdates() map[string]any {
	u := make(map[string]any, 16)

	// quiz_assessment_id (nullable)
	if p.QuizAssessmentID.ShouldUpdate() {
		if p.QuizAssessmentID.IsNull() {
			u["quiz_assessment_id"] = gorm.Expr("NULL")
		} else {
			u["quiz_assessment_id"] = p.QuizAssessmentID.Val()
		}
	}

	// quiz_assessment_type_id (nullable)
	if p.QuizAssessmentTypeID.ShouldUpdate() {
		if p.QuizAssessmentTypeID.IsNull() {
			u["quiz_assessment_type_id"] = gorm.Expr("NULL")
		} else {
			u["quiz_assessment_type_id"] = p.QuizAssessmentTypeID.Val()
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

	// quiz_is_published (bool, non-nullable)
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

	// ========== Remedial flags ==========

	// quiz_is_remedial (bool, non-nullable)
	if p.QuizIsRemedial.ShouldUpdate() && !p.QuizIsRemedial.IsNull() {
		u["quiz_is_remedial"] = p.QuizIsRemedial.Val()
	}

	// quiz_parent_quiz_id (nullable uuid)
	if p.QuizParentQuizID.ShouldUpdate() {
		if p.QuizParentQuizID.IsNull() {
			u["quiz_parent_quiz_id"] = gorm.Expr("NULL")
		} else {
			u["quiz_parent_quiz_id"] = p.QuizParentQuizID.Val()
		}
	}

	// quiz_remedial_round (nullable int)
	if p.QuizRemedialRound.ShouldUpdate() {
		if p.QuizRemedialRound.IsNull() {
			u["quiz_remedial_round"] = gorm.Expr("NULL")
		} else {
			u["quiz_remedial_round"] = p.QuizRemedialRound.Val()
		}
	}

	return u
}

/* ==============================
   QUERY (GET /quizzes)
============================== */

type ListQuizzesQuery struct {
	// filter dasar
	SchoolID         *uuid.UUID `query:"school_id" validate:"omitempty,uuid4"`
	ID               *uuid.UUID `query:"id" validate:"omitempty,uuid4"` // quiz_id
	AssessmentID     *uuid.UUID `query:"assessment_id" validate:"omitempty,uuid4"`
	AssessmentTypeID *uuid.UUID `query:"assessment_type_id" validate:"omitempty,uuid4"`

	// filter by slug (exact)
	Slug *string `query:"slug" validate:"omitempty,max=160"`

	IsPublished *bool  `query:"is_published" validate:"omitempty"`
	Q           string `query:"q" validate:"omitempty,max=120"`

	// remedial filters (opsional)
	IsRemedial    *bool      `query:"is_remedial" validate:"omitempty"`
	ParentQuizID  *uuid.UUID `query:"parent_quiz_id" validate:"omitempty,uuid4"`
	RemedialRound *int       `query:"remedial_round" validate:"omitempty,gte=1"`

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

/*
==============================
  RESPONSE DTOs
==============================
*/

type QuizResponse struct {
	QuizID           uuid.UUID  `json:"quiz_id"`
	QuizSchoolID     uuid.UUID  `json:"quiz_school_id"`
	QuizAssessmentID *uuid.UUID `json:"quiz_assessment_id,omitempty"`

	// relasi langsung ke assessment type
	QuizAssessmentTypeID *uuid.UUID `json:"quiz_assessment_type_id,omitempty"`

	QuizSlug *string `json:"quiz_slug,omitempty"`

	QuizTitle        string  `json:"quiz_title"`
	QuizDescription  *string `json:"quiz_description,omitempty"`
	QuizIsPublished  bool    `json:"quiz_is_published"`
	QuizTimeLimitSec *int    `json:"quiz_time_limit_sec,omitempty"`

	// total waktu untuk mengerjakan semua soal (detik)
	// quiz_time_limit_sec_all = quiz_time_limit_sec * jumlah_soal
	QuizTimeLimitSecAll *int `json:"quiz_time_limit_sec_all,omitempty"`

	// denorm jumlah soal
	QuizTotalQuestions int `json:"quiz_total_questions"`

	// remedial flags
	QuizIsRemedial    bool       `json:"quiz_is_remedial"`
	QuizParentQuizID  *uuid.UUID `json:"quiz_parent_quiz_id,omitempty"`
	QuizRemedialRound *int       `json:"quiz_remedial_round,omitempty"`

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

/*
==============================
  MAPPERS
==============================
*/

func FromModel(m *model.QuizModel) QuizResponse {
	var deletedAt *time.Time
	if m.QuizDeletedAt.Valid {
		t := m.QuizDeletedAt.Time
		deletedAt = &t
	}

	// base total time dari denorm (bisa dioverride di controller kalau punya len(questions))
	var totalAll *int
	if m.QuizTimeLimitSec != nil && m.QuizTotalQuestions > 0 {
		v := (*m.QuizTimeLimitSec) * m.QuizTotalQuestions
		totalAll = &v
	}

	return QuizResponse{
		QuizID:           m.QuizID,
		QuizSchoolID:     m.QuizSchoolID,
		QuizAssessmentID: m.QuizAssessmentID,

		QuizAssessmentTypeID: m.QuizAssessmentTypeID,

		QuizSlug:        m.QuizSlug,
		QuizTitle:       m.QuizTitle,
		QuizDescription: m.QuizDescription,
		QuizIsPublished: m.QuizIsPublished,
		QuizTimeLimitSec: func() *int {
			if m.QuizTimeLimitSec == nil {
				return nil
			}
			return m.QuizTimeLimitSec
		}(),

		QuizTimeLimitSecAll: totalAll,

		QuizTotalQuestions: m.QuizTotalQuestions,

		// remedial flags
		QuizIsRemedial:    m.QuizIsRemedial,
		QuizParentQuizID:  m.QuizParentQuizID,
		QuizRemedialRound: m.QuizRemedialRound,

		QuizCreatedAt: m.QuizCreatedAt,
		QuizUpdatedAt: m.QuizUpdatedAt,
		QuizDeletedAt: deletedAt,

		// Questions & QuestionsCount diisi di service/controller kalau perlu
	}
}

func FromModels(rows []model.QuizModel) []QuizResponse {
	out := make([]QuizResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromModel(&rows[i]))
	}
	return out
}

// ==============================
//  MAPPERS DENGAN TIMEZONE SEKOLAH
// ==============================

// Versi aware timezone sekolah untuk Quiz.
// - QuizCreatedAt / QuizUpdatedAt / QuizDeletedAt dikonversi via dbtime.ToSchoolTime
func FromModelWithCtx(c *fiber.Ctx, m *model.QuizModel) QuizResponse {
	// Pakai mapper lama dulu
	resp := FromModel(m)

	// Override waktu dengan timezone sekolah
	resp.QuizCreatedAt = dbtime.ToSchoolTime(c, m.QuizCreatedAt)
	resp.QuizUpdatedAt = dbtime.ToSchoolTime(c, m.QuizUpdatedAt)

	if m.QuizDeletedAt.Valid {
		t := dbtime.ToSchoolTime(c, m.QuizDeletedAt.Time)
		resp.QuizDeletedAt = &t
	} else {
		resp.QuizDeletedAt = nil
	}

	return resp
}

func FromModelsWithCtx(c *fiber.Ctx, rows []model.QuizModel) []QuizResponse {
	out := make([]QuizResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromModelWithCtx(c, &rows[i]))
	}
	return out
}
