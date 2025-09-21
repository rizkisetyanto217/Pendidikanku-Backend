package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
)

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
	// "null" -> tandai null
	if string(b) == "null" {
		f.null = true
		var zero T
		f.value = zero
		return nil
	}
	// value normal
	return json.Unmarshal(b, &f.value)
}

func (f UpdateField[T]) ShouldUpdate() bool { return f.set }
func (f UpdateField[T]) IsNull() bool       { return f.set && f.null }
func (f UpdateField[T]) Val() T             { return f.value }

/*
	==============================
	  CREATE (POST /quizzes)

==============================
*/
type CreateQuizRequest struct {
	QuizzesMasjidID     uuid.UUID  `json:"quizzes_masjid_id"`
	QuizzesAssessmentID *uuid.UUID `json:"quizzes_assessment_id" validate:"omitempty"`

	// NEW: slug (opsional; unik per tenant saat alive)
	QuizzesSlug *string `json:"quizzes_slug" validate:"omitempty,max=160"`

	QuizzesTitle        string  `json:"quizzes_title" validate:"required,max=180"`
	QuizzesDescription  *string `json:"quizzes_description" validate:"omitempty"`
	QuizzesIsPublished  *bool   `json:"quizzes_is_published" validate:"omitempty"`
	QuizzesTimeLimitSec *int    `json:"quizzes_time_limit_sec" validate:"omitempty,gte=0"`
}

// ToModel: builder model dari payload Create (GORM isi timestamps)
func (r *CreateQuizRequest) ToModel() *model.QuizModel {
	// Default sesuai DDL: false
	isPub := false
	if r.QuizzesIsPublished != nil {
		isPub = *r.QuizzesIsPublished
	}
	return &model.QuizModel{
		QuizzesMasjidID:     r.QuizzesMasjidID,
		QuizzesAssessmentID: r.QuizzesAssessmentID,

		QuizzesSlug:        trimPtr(r.QuizzesSlug),
		QuizzesTitle:       strings.TrimSpace(r.QuizzesTitle),
		QuizzesDescription: trimPtr(r.QuizzesDescription),

		QuizzesIsPublished:  isPub,
		QuizzesTimeLimitSec: r.QuizzesTimeLimitSec,
	}
}

/*
	==============================
	  PATCH (PATCH /quizzes/:id)
	  - gunakan UpdateField agar bisa null/skip/value

==============================
*/
type PatchQuizRequest struct {
	QuizzesAssessmentID UpdateField[uuid.UUID] `json:"quizzes_assessment_id"` // nullable

	// NEW: slug (nullable)
	QuizzesSlug UpdateField[string] `json:"quizzes_slug"`

	QuizzesTitle        UpdateField[string] `json:"quizzes_title"`       // NOT NULL di DB (abaikan jika null)
	QuizzesDescription  UpdateField[string] `json:"quizzes_description"` // nullable
	QuizzesIsPublished  UpdateField[bool]   `json:"quizzes_is_published"`
	QuizzesTimeLimitSec UpdateField[int]    `json:"quizzes_time_limit_sec"` // nullable
}

// ToUpdates: map untuk gorm.Model(&m).Updates(...)
func (p *PatchQuizRequest) ToUpdates() map[string]any {
	u := make(map[string]any, 7)

	// quizzes_assessment_id (nullable)
	if p.QuizzesAssessmentID.ShouldUpdate() {
		if p.QuizzesAssessmentID.IsNull() {
			u["quizzes_assessment_id"] = gorm.Expr("NULL")
		} else {
			u["quizzes_assessment_id"] = p.QuizzesAssessmentID.Val()
		}
	}

	// quizzes_slug (nullable, max 160 — validasi panjang di controller/validator)
	if p.QuizzesSlug.ShouldUpdate() {
		if p.QuizzesSlug.IsNull() {
			u["quizzes_slug"] = gorm.Expr("NULL")
		} else {
			slug := strings.TrimSpace(p.QuizzesSlug.Val())
			if slug == "" {
				u["quizzes_slug"] = gorm.Expr("NULL")
			} else {
				u["quizzes_slug"] = slug
			}
		}
	}

	// quizzes_title (NOT NULL) -> abaikan jika null
	if p.QuizzesTitle.ShouldUpdate() && !p.QuizzesTitle.IsNull() {
		title := strings.TrimSpace(p.QuizzesTitle.Val())
		if title != "" {
			u["quizzes_title"] = title
		}
	}

	// quizzes_description (nullable)
	if p.QuizzesDescription.ShouldUpdate() {
		if p.QuizzesDescription.IsNull() {
			u["quizzes_description"] = gorm.Expr("NULL")
		} else {
			desc := strings.TrimSpace(p.QuizzesDescription.Val())
			if desc == "" {
				u["quizzes_description"] = gorm.Expr("NULL")
			} else {
				u["quizzes_description"] = &desc // simpan sebagai *string
			}
		}
	}

	// quizzes_is_published (bool)
	if p.QuizzesIsPublished.ShouldUpdate() && !p.QuizzesIsPublished.IsNull() {
		u["quizzes_is_published"] = p.QuizzesIsPublished.Val()
	}

	// quizzes_time_limit_sec (nullable int)
	if p.QuizzesTimeLimitSec.ShouldUpdate() {
		if p.QuizzesTimeLimitSec.IsNull() {
			u["quizzes_time_limit_sec"] = gorm.Expr("NULL")
		} else {
			u["quizzes_time_limit_sec"] = p.QuizzesTimeLimitSec.Val()
		}
	}

	return u
}

/*
	==============================
	  QUERY (GET /quizzes)

==============================
*/
type ListQuizzesQuery struct {
	// filter dasar
	MasjidID     *uuid.UUID `query:"masjid_id" validate:"omitempty"`
	ID           *uuid.UUID `query:"id"        validate:"omitempty,uuid"` // filter by ID
	AssessmentID *uuid.UUID `query:"assessment_id" validate:"omitempty"`

	// NEW: filter by slug (exact)
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

/*
	==============================
	  RESPONSE DTOs

==============================
*/
type QuizResponse struct {
	QuizzesID           uuid.UUID  `json:"quizzes_id"`
	QuizzesMasjidID     uuid.UUID  `json:"quizzes_masjid_id"`
	QuizzesAssessmentID *uuid.UUID `json:"quizzes_assessment_id,omitempty"`

	// NEW: slug
	QuizzesSlug *string `json:"quizzes_slug,omitempty"`

	QuizzesTitle        string  `json:"quizzes_title"`
	QuizzesDescription  *string `json:"quizzes_description,omitempty"`
	QuizzesIsPublished  bool    `json:"quizzes_is_published"`
	QuizzesTimeLimitSec *int    `json:"quizzes_time_limit_sec,omitempty"`

	QuizzesCreatedAt time.Time  `json:"quizzes_created_at"`
	QuizzesUpdatedAt time.Time  `json:"quizzes_updated_at"`
	QuizzesDeletedAt *time.Time `json:"quizzes_deleted_at,omitempty"`

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
	if m.QuizzesDeletedAt.Valid {
		t := m.QuizzesDeletedAt.Time
		deletedAt = &t
	}
	return QuizResponse{
		QuizzesID:           m.QuizzesID,
		QuizzesMasjidID:     m.QuizzesMasjidID,
		QuizzesAssessmentID: m.QuizzesAssessmentID,

		QuizzesSlug:         m.QuizzesSlug,
		QuizzesTitle:        m.QuizzesTitle,
		QuizzesDescription:  m.QuizzesDescription,
		QuizzesIsPublished:  m.QuizzesIsPublished,
		QuizzesTimeLimitSec: m.QuizzesTimeLimitSec,

		QuizzesCreatedAt: m.QuizzesCreatedAt,
		QuizzesUpdatedAt: m.QuizzesUpdatedAt,
		QuizzesDeletedAt: deletedAt,
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
