package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	model "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
)

/* ==============================
   Helper: Tri-state updater
   - Absent  : tidak diupdate
   - null    : set kolom ke NULL
   - value   : set kolom ke value
============================== */
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

/* ==============================
   CREATE (POST /quizzes)
============================== */
type CreateQuizRequest struct {
	QuizzesMasjidID      uuid.UUID  `json:"quizzes_masjid_id"`
	QuizzesAssessmentID  *uuid.UUID `json:"quizzes_assessment_id" validate:"omitempty"`
	QuizzesTitle         string     `json:"quizzes_title" validate:"required,max=180"`
	QuizzesDescription   *string    `json:"quizzes_description" validate:"omitempty"`
	QuizzesIsPublished   *bool      `json:"quizzes_is_published" validate:"omitempty"`
	QuizzesTimeLimitSec  *int       `json:"quizzes_time_limit_sec" validate:"omitempty,gte=0"`
}



// ToModel: builder model dari payload Create (GORM isi timestamps)
func (r *CreateQuizRequest) ToModel() *model.QuizModel {
	isPub := true
	if r.QuizzesIsPublished != nil {
		isPub = *r.QuizzesIsPublished
	}
	return &model.QuizModel{
		QuizzesMasjidID:     r.QuizzesMasjidID,
		QuizzesAssessmentID: r.QuizzesAssessmentID,

		QuizzesTitle:        r.QuizzesTitle,
		QuizzesDescription:  r.QuizzesDescription,
		QuizzesIsPublished:  isPub,
		QuizzesTimeLimitSec: r.QuizzesTimeLimitSec,
	}
}

/* ==============================
   PATCH (PATCH /quizzes/:id)
   - gunakan UpdateField agar bisa null/skip/value
============================== */
type PatchQuizRequest struct {
	QuizzesAssessmentID  UpdateField[uuid.UUID] `json:"quizzes_assessment_id"`  // nullable
	QuizzesTitle         UpdateField[string]    `json:"quizzes_title"`          // NOT NULL di DB (abaikan jika null)
	QuizzesDescription   UpdateField[string]    `json:"quizzes_description"`    // nullable
	QuizzesIsPublished   UpdateField[bool]      `json:"quizzes_is_published"`   // bool (bukan null)
	QuizzesTimeLimitSec  UpdateField[int]       `json:"quizzes_time_limit_sec"` // nullable
}

// ToUpdates: map untuk gorm.Model(&m).Updates(...)
func (p *PatchQuizRequest) ToUpdates() map[string]any {
	u := make(map[string]any, 6)

	// quizzes_assessment_id (nullable)
	if p.QuizzesAssessmentID.ShouldUpdate() {
		if p.QuizzesAssessmentID.IsNull() {
			u["quizzes_assessment_id"] = gorm.Expr("NULL")
		} else {
			u["quizzes_assessment_id"] = p.QuizzesAssessmentID.Val()
		}
	}

	// quizzes_title (NOT NULL) -> abaikan jika null
	if p.QuizzesTitle.ShouldUpdate() && !p.QuizzesTitle.IsNull() {
		title := p.QuizzesTitle.Val()
		// validasi ringan panjang bisa di controller bila perlu
		u["quizzes_title"] = title
	}

	// quizzes_description (nullable)
	if p.QuizzesDescription.ShouldUpdate() {
		if p.QuizzesDescription.IsNull() {
			u["quizzes_description"] = gorm.Expr("NULL")
		} else {
			desc := p.QuizzesDescription.Val()
			u["quizzes_description"] = &desc // simpan sebagai *string agar null bisa dibedakan
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

/* ==============================
   QUERY (GET /quizzes)
============================== */
type ListQuizzesQuery struct {
	// filter dasar
	MasjidID     *uuid.UUID `query:"masjid_id" validate:"omitempty"`
	ID           *uuid.UUID `query:"id"        validate:"omitempty,uuid"` // <— filter by ID
	AssessmentID *uuid.UUID `query:"assessment_id" validate:"omitempty"`
	IsPublished  *bool      `query:"is_published" validate:"omitempty"`
	Q            string     `query:"q" validate:"omitempty,max=120"`

	// pagination & sorting
	Page    int    `query:"page" validate:"omitempty,gte=0"`
	PerPage int    `query:"per_page" validate:"omitempty,gte=0,lte=200"`
	Sort    string `query:"sort" validate:"omitempty,oneof=created_at desc_created_at title desc_title published desc_published"`

	// NEW:
	WithQuestions     bool   `query:"with_questions"`
	QuestionsLimit    int    `query:"questions_limit" validate:"omitempty,min=1,max=200"`
	QuestionsOrder    string `query:"questions_order" validate:"omitempty,oneof=created_at desc_created_at"`
	WithQuestionsCount bool  `query:"with_questions_count"`
}




/* ==============================
   RESPONSE DTOs
============================== */
type QuizResponse struct {
	QuizzesID            uuid.UUID  `json:"quizzes_id"`
	QuizzesMasjidID      uuid.UUID  `json:"quizzes_masjid_id"`
	QuizzesAssessmentID  *uuid.UUID `json:"quizzes_assessment_id,omitempty"`

	QuizzesTitle         string   `json:"quizzes_title"`
	QuizzesDescription   *string  `json:"quizzes_description,omitempty"`
	QuizzesIsPublished   bool     `json:"quizzes_is_published"`
	QuizzesTimeLimitSec  *int     `json:"quizzes_time_limit_sec,omitempty"`

	QuizzesCreatedAt     time.Time  `json:"quizzes_created_at"`
	QuizzesUpdatedAt     time.Time  `json:"quizzes_updated_at"`
	QuizzesDeletedAt     *time.Time `json:"quizzes_deleted_at,omitempty"`

	Questions       []*QuizQuestionResponse `json:"questions,omitempty"`
	QuestionsCount  *int                    `json:"questions_count,omitempty"`
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
	if m.QuizzesDeletedAt.Valid {
		t := m.QuizzesDeletedAt.Time
		deletedAt = &t
	}
	return QuizResponse{
		QuizzesID:           m.QuizzesID,
		QuizzesMasjidID:     m.QuizzesMasjidID,
		QuizzesAssessmentID: m.QuizzesAssessmentID,

		QuizzesTitle:        m.QuizzesTitle,
		QuizzesDescription:  m.QuizzesDescription,
		QuizzesIsPublished:  m.QuizzesIsPublished,
		QuizzesTimeLimitSec: m.QuizzesTimeLimitSec,

		QuizzesCreatedAt:    m.QuizzesCreatedAt,
		QuizzesUpdatedAt:    m.QuizzesUpdatedAt,
		QuizzesDeletedAt:    deletedAt,
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
