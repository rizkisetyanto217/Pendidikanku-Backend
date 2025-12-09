// file: internals/features/school/submissions_assesments/quizzes/dto/quiz_question_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
)

/* =========================================================
   CREATE
========================================================= */

// SINGLE: isi answers (object) + correct (key, misal "A").
// ESSAY : biarkan answers & correct kosong.
type CreateQuizQuestionRequest struct {
	QuizQuestionQuizID      uuid.UUID               `json:"quiz_question_quiz_id" validate:"required,uuid4"`
	QuizQuestionSchoolID    uuid.UUID               `json:"quiz_question_school_id"` // controller boleh force override dari tenant
	QuizQuestionType        qmodel.QuizQuestionType `json:"quiz_question_type" validate:"required,oneof=single essay"`
	QuizQuestionText        string                  `json:"quiz_question_text" validate:"required"`
	QuizQuestionPoints      *float64                `json:"quiz_question_points" validate:"omitempty,gte=0"`
	QuizQuestionAnswers     *json.RawMessage        `json:"quiz_question_answers" validate:"omitempty"` // object untuk SINGLE
	QuizQuestionCorrect     *string                 `json:"quiz_question_correct" validate:"omitempty"` // key di answers, misal "A"
	QuizQuestionExplanation *string                 `json:"quiz_question_explanation" validate:"omitempty"`
}

func (r *CreateQuizQuestionRequest) ToModel() (*qmodel.QuizQuestionModel, error) {
	points := 1.0
	if r.QuizQuestionPoints != nil {
		points = *r.QuizQuestionPoints
	}

	var ans datatypes.JSON
	if r.QuizQuestionAnswers != nil && len(*r.QuizQuestionAnswers) > 0 {
		ans = datatypes.JSON(*r.QuizQuestionAnswers)
	}

	var correct *string
	if r.QuizQuestionCorrect != nil {
		c := strings.TrimSpace(*r.QuizQuestionCorrect)
		if c != "" {
			correct = &c
		}
	}

	m := &qmodel.QuizQuestionModel{
		QuizQuestionQuizID:      r.QuizQuestionQuizID,
		QuizQuestionSchoolID:    r.QuizQuestionSchoolID,
		QuizQuestionType:        r.QuizQuestionType,
		QuizQuestionText:        strings.TrimSpace(r.QuizQuestionText),
		QuizQuestionPoints:      points,
		QuizQuestionAnswers:     ans,
		QuizQuestionCorrect:     correct,
		QuizQuestionExplanation: trimPtr(r.QuizQuestionExplanation),
		// Version dan History pakai default DB (version=1, history=[])
	}

	// Domain-level validation
	if err := m.ValidateShape(); err != nil {
		return nil, err
	}
	return m, nil
}

/* =========================================================
   PATCH (partial)
========================================================= */

type PatchQuizQuestionRequest struct {
	QuizQuestionQuizID      UpdateField[uuid.UUID]               `json:"quiz_question_quiz_id"`
	QuizQuestionSchoolID    UpdateField[uuid.UUID]               `json:"quiz_question_school_id"` // biasanya tidak diizinkan ubah
	QuizQuestionType        UpdateField[qmodel.QuizQuestionType] `json:"quiz_question_type"`      // single/essay
	QuizQuestionText        UpdateField[string]                  `json:"quiz_question_text"`
	QuizQuestionPoints      UpdateField[float64]                 `json:"quiz_question_points"`
	QuizQuestionAnswers     UpdateField[json.RawMessage]         `json:"quiz_question_answers"` // object untuk SINGLE
	QuizQuestionCorrect     UpdateField[string]                  `json:"quiz_question_correct"` // key di answers
	QuizQuestionExplanation UpdateField[string]                  `json:"quiz_question_explanation"`

	// "major" → simpan snapshot ke history + naikkan version
	// "minor" (atau kosong) → update tanpa history
	ChangeKind string `json:"change_kind"` // optional: "major" / "minor"
}

// Terapkan patch langsung ke model yang sudah di-load, lalu validasi shape.
// Di sini juga di-handle logika major/minor untuk history.
func (p *PatchQuizQuestionRequest) ApplyToModel(m *qmodel.QuizQuestionModel) error {
	// 0) Handle history untuk perubahan mayor
	kind := strings.ToLower(strings.TrimSpace(p.ChangeKind))
	if kind == "major" {
		if err := m.AppendHistorySnapshot("major"); err != nil {
			return err
		}
	}

	// 1) IDs
	if p.QuizQuestionQuizID.ShouldUpdate() && !p.QuizQuestionQuizID.IsNull() {
		m.QuizQuestionQuizID = p.QuizQuestionQuizID.Val()
	}
	if p.QuizQuestionSchoolID.ShouldUpdate() && !p.QuizQuestionSchoolID.IsNull() {
		m.QuizQuestionSchoolID = p.QuizQuestionSchoolID.Val()
	}

	// 2) Type
	if p.QuizQuestionType.ShouldUpdate() && !p.QuizQuestionType.IsNull() {
		m.QuizQuestionType = p.QuizQuestionType.Val()
	}

	// 3) Text
	if p.QuizQuestionText.ShouldUpdate() {
		if p.QuizQuestionText.IsNull() {
			return errors.New("quiz_question_text tidak boleh null")
		}
		m.QuizQuestionText = strings.TrimSpace(p.QuizQuestionText.Val())
	}

	// 4) Points
	if p.QuizQuestionPoints.ShouldUpdate() {
		if p.QuizQuestionPoints.IsNull() {
			m.QuizQuestionPoints = 1.0
		} else {
			m.QuizQuestionPoints = p.QuizQuestionPoints.Val()
		}
	}

	// 5) Answers
	if p.QuizQuestionAnswers.ShouldUpdate() {
		if p.QuizQuestionAnswers.IsNull() {
			m.QuizQuestionAnswers = nil
		} else {
			raw := p.QuizQuestionAnswers.Val()
			m.QuizQuestionAnswers = datatypes.JSON(raw)
		}
	}

	// 6) Correct
	if p.QuizQuestionCorrect.ShouldUpdate() {
		if p.QuizQuestionCorrect.IsNull() {
			m.QuizQuestionCorrect = nil
		} else {
			c := strings.TrimSpace(p.QuizQuestionCorrect.Val())
			if c == "" {
				m.QuizQuestionCorrect = nil
			} else {
				m.QuizQuestionCorrect = &c
			}
		}
	}

	// 7) Explanation
	if p.QuizQuestionExplanation.ShouldUpdate() {
		if p.QuizQuestionExplanation.IsNull() {
			m.QuizQuestionExplanation = nil
		} else {
			v := p.QuizQuestionExplanation.Val()
			m.QuizQuestionExplanation = trimPtr(&v)
		}
	}

	// 8) Final domain validation
	return m.ValidateShape()
}

/* =========================================================
   LIST QUERY (GET /quiz-questions)
========================================================= */

type ListQuizQuestionsQuery struct {
	SchoolID *uuid.UUID `query:"school_id" validate:"omitempty,uuid4"`
	ID       *uuid.UUID `query:"id" validate:"omitempty,uuid4"`      // quiz_question_id
	QuizID   *uuid.UUID `query:"quiz_id" validate:"omitempty,uuid4"` // filter by quiz

	Type string `query:"type" validate:"omitempty,oneof=single essay"`
	Q    string `query:"q" validate:"omitempty,max=200"` // search text/explanation

	Page    int    `query:"page" validate:"omitempty,gte=0"`
	PerPage int    `query:"per_page" validate:"omitempty,gte=0,lte=200"`
	Sort    string `query:"sort" validate:"omitempty,oneof=created_at desc_created_at points desc_points type desc_type"`

	WithQuiz bool `query:"with_quiz"` // kalau true → preload quiz parent & embed di response
}

/*
	=========================================================
	  Lite Quiz info (embed di question jika with_quiz=true)

=========================================================
*/
type QuizLiteResponse struct {
	QuizID           uuid.UUID  `json:"quiz_id"`
	QuizSchoolID     uuid.UUID  `json:"quiz_school_id"`
	QuizAssessmentID *uuid.UUID `json:"quiz_assessment_id,omitempty"`

	// NEW: relasi langsung ke assessment type
	QuizAssessmentTypeID *uuid.UUID `json:"quiz_assessment_type_id,omitempty"`

	QuizSlug *string `json:"quiz_slug,omitempty"`

	QuizTitle        string  `json:"quiz_title"`
	QuizDescription  *string `json:"quiz_description,omitempty"`
	QuizIsPublished  bool    `json:"quiz_is_published"`
	QuizTimeLimitSec *int    `json:"quiz_time_limit_sec,omitempty"`

	// denorm jumlah soal
	QuizTotalQuestions int `json:"quiz_total_questions"`

	QuizCreatedAt time.Time  `json:"quiz_created_at"`
	QuizUpdatedAt time.Time  `json:"quiz_updated_at"`
	QuizDeletedAt *time.Time `json:"quiz_deleted_at,omitempty"`
}

/*
	=========================================================
	  RESPONSE
	=========================================================
*/

type QuizQuestionResponse struct {
	QuizQuestionID          uuid.UUID               `json:"quiz_question_id"`
	QuizQuestionQuizID      uuid.UUID               `json:"quiz_question_quiz_id"`
	QuizQuestionSchoolID    uuid.UUID               `json:"quiz_question_school_id"`
	QuizQuestionType        qmodel.QuizQuestionType `json:"quiz_question_type"`
	QuizQuestionText        string                  `json:"quiz_question_text"`
	QuizQuestionPoints      float64                 `json:"quiz_question_points"`
	QuizQuestionAnswers     *json.RawMessage        `json:"quiz_question_answers,omitempty"`
	QuizQuestionCorrect     *string                 `json:"quiz_question_correct,omitempty"`
	QuizQuestionExplanation *string                 `json:"quiz_question_explanation,omitempty"`

	QuizQuestionCreatedAt string `json:"quiz_question_created_at"`
	QuizQuestionUpdatedAt string `json:"quiz_question_updated_at"`

	// ➕ Tambahan
	QuizQuestionVersion int              `json:"quiz_question_version"`
	QuizQuestionHistory *json.RawMessage `json:"quiz_question_history,omitempty"` // optional kalau mau ditampilkan juga

	// Optional: parent quiz (jika with_quiz=true dan sudah di-Preload)
	Quiz *QuizLiteResponse `json:"quiz,omitempty"`
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func FromModelQuizQuestion(m *qmodel.QuizQuestionModel) *QuizQuestionResponse {
	var ans *json.RawMessage
	if len(m.QuizQuestionAnswers) > 0 {
		tmp := json.RawMessage(m.QuizQuestionAnswers)
		ans = &tmp
	}

	var history *json.RawMessage
	if len(m.QuizQuestionHistory) > 0 {
		tmp := json.RawMessage(m.QuizQuestionHistory)
		history = &tmp
	}

	// Build lite quiz jika di-preload
	var quizLite *QuizLiteResponse
	if m.Quiz != nil {
		var deletedAt *time.Time
		if m.Quiz.QuizDeletedAt.Valid {
			t := m.Quiz.QuizDeletedAt.Time
			deletedAt = &t
		}

		quizLite = &QuizLiteResponse{
			QuizID:               m.Quiz.QuizID,
			QuizSchoolID:         m.Quiz.QuizSchoolID,
			QuizAssessmentID:     m.Quiz.QuizAssessmentID,
			QuizAssessmentTypeID: m.Quiz.QuizAssessmentTypeID,

			QuizSlug:         m.Quiz.QuizSlug,
			QuizTitle:        m.Quiz.QuizTitle,
			QuizDescription:  m.Quiz.QuizDescription,
			QuizIsPublished:  m.Quiz.QuizIsPublished,
			QuizTimeLimitSec: m.Quiz.QuizTimeLimitSec,

			QuizTotalQuestions: m.Quiz.QuizTotalQuestions,

			QuizCreatedAt: m.Quiz.QuizCreatedAt,
			QuizUpdatedAt: m.Quiz.QuizUpdatedAt,
			QuizDeletedAt: deletedAt,
		}
	}

	return &QuizQuestionResponse{
		QuizQuestionID:          m.QuizQuestionID,
		QuizQuestionQuizID:      m.QuizQuestionQuizID,
		QuizQuestionSchoolID:    m.QuizQuestionSchoolID,
		QuizQuestionType:        m.QuizQuestionType,
		QuizQuestionText:        m.QuizQuestionText,
		QuizQuestionPoints:      m.QuizQuestionPoints,
		QuizQuestionAnswers:     ans,
		QuizQuestionCorrect:     m.QuizQuestionCorrect,
		QuizQuestionExplanation: m.QuizQuestionExplanation,
		QuizQuestionCreatedAt:   m.QuizQuestionCreatedAt.UTC().Format(timeRFC3339),
		QuizQuestionUpdatedAt:   m.QuizQuestionUpdatedAt.UTC().Format(timeRFC3339),

		QuizQuestionVersion: m.QuizQuestionVersion,
		QuizQuestionHistory: history,

		Quiz: quizLite,
	}
}

func FromModelsQuizQuestions(arr []qmodel.QuizQuestionModel) []*QuizQuestionResponse {
	out := make([]*QuizQuestionResponse, 0, len(arr))
	for i := range arr {
		out = append(out, FromModelQuizQuestion(&arr[i]))
	}
	return out
}
