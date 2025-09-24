package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
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
	QuizMasjidID     uuid.UUID  `json:"quiz_masjid_id" validate:"required"`
	QuizAssessmentID *uuid.UUID `json:"quiz_assessment_id" validate:"omitempty"`

	// Identitas
	QuizSlug        *string `json:"quiz_slug" validate:"omitempty,max=160"`
	QuizTitle       string  `json:"quiz_title" validate:"required,max=180"`
	QuizDescription *string `json:"quiz_description" validate:"omitempty"`

	// Pengaturan
	QuizIsPublished  *bool `json:"quiz_is_published" validate:"omitempty"`
	QuizTimeLimitSec *int  `json:"quiz_time_limit_sec" validate:"omitempty,gte=0"`
}

// ToModel: builder model dari payload Create (timestamps oleh GORM)
func (r *CreateQuizRequest) ToModel() *model.QuizModel {
	isPub := false
	if r.QuizIsPublished != nil {
		isPub = *r.QuizIsPublished
	}
	return &model.QuizModel{
		QuizMasjidID:     r.QuizMasjidID,
		QuizAssessmentID: r.QuizAssessmentID,

		QuizSlug:        trimPtr(r.QuizSlug),
		QuizTitle:       strings.TrimSpace(r.QuizTitle),
		QuizDescription: trimPtr(r.QuizDescription),

		QuizIsPublished:  isPub,
		QuizTimeLimitSec: r.QuizTimeLimitSec,
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
}

// ToUpdates: map untuk gorm.Model(&m).Updates(...)
func (p *PatchQuizRequest) ToUpdates() map[string]any {
	u := make(map[string]any, 7)

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

	return u
}

/* ==============================
   QUERY (GET /quizzes)
============================== */

type ListQuizzesQuery struct {
	// filter dasar
	MasjidID     *uuid.UUID `query:"masjid_id" validate:"omitempty"`
	ID           *uuid.UUID `query:"id" validate:"omitempty,uuid"` // quiz_id
	AssessmentID *uuid.UUID `query:"assessment_id" validate:"omitempty"`

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
	QuizMasjidID     uuid.UUID  `json:"quiz_masjid_id"`
	QuizAssessmentID *uuid.UUID `json:"quiz_assessment_id,omitempty"`

	QuizSlug *string `json:"quiz_slug,omitempty"`

	QuizTitle        string  `json:"quiz_title"`
	QuizDescription  *string `json:"quiz_description,omitempty"`
	QuizIsPublished  bool    `json:"quiz_is_published"`
	QuizTimeLimitSec *int    `json:"quiz_time_limit_sec,omitempty"`

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
		QuizMasjidID:     m.QuizMasjidID,
		QuizAssessmentID: m.QuizAssessmentID,

		QuizSlug:         m.QuizSlug,
		QuizTitle:        m.QuizTitle,
		QuizDescription:  m.QuizDescription,
		QuizIsPublished:  m.QuizIsPublished,
		QuizTimeLimitSec: m.QuizTimeLimitSec,

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
