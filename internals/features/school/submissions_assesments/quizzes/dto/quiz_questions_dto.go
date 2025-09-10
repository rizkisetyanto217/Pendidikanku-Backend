package dto

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
)

/* =========================================================
   Tri-state field (absent / null / value) untuk PATCH
========================================================= */

/* =========================================================
   CREATE
========================================================= */

// Satu DTO general:
// - SINGLE: isi answers (object/array). Untuk OBJECT, isi 'correct' (A..D).
// - ESSAY : biarkan answers & correct kosong.
type CreateQuizQuestionRequest struct {
	QuizQuestionsQuizID      uuid.UUID               `json:"quiz_questions_quiz_id" validate:"required"`
	QuizQuestionsMasjidID    uuid.UUID               `json:"quiz_questions_masjid_id"` // controller boleh force override dari tenant
	QuizQuestionsType        qmodel.QuizQuestionType `json:"quiz_questions_type" validate:"required,oneof=single essay"`
	QuizQuestionsText        string                  `json:"quiz_questions_text" validate:"required"`
	QuizQuestionsPoints      *float64                `json:"quiz_questions_points" validate:"omitempty,gte=0"`
	QuizQuestionsAnswers     *json.RawMessage        `json:"quiz_questions_answers" validate:"omitempty"` // object/array (SINGLE) atau null
	QuizQuestionsCorrect     *string                 `json:"quiz_questions_correct" validate:"omitempty,oneof=A B C D a b c d"`
	QuizQuestionsExplanation *string                 `json:"quiz_questions_explanation" validate:"omitempty"`
}

func (r *CreateQuizQuestionRequest) ToModel() (*qmodel.QuizQuestionModel, error) {
	points := 1.0
	if r.QuizQuestionsPoints != nil {
		points = *r.QuizQuestionsPoints
	}

	var ans datatypes.JSON
	if r.QuizQuestionsAnswers != nil && len(*r.QuizQuestionsAnswers) > 0 {
		ans = datatypes.JSON(*r.QuizQuestionsAnswers)
	}

	var correct *string
	if r.QuizQuestionsCorrect != nil {
		c := strings.ToUpper(strings.TrimSpace(*r.QuizQuestionsCorrect))
		correct = &c
	}

	m := &qmodel.QuizQuestionModel{
		QuizQuestionsQuizID:      r.QuizQuestionsQuizID,
		QuizQuestionsMasjidID:    r.QuizQuestionsMasjidID,
		QuizQuestionsType:        r.QuizQuestionsType,
		QuizQuestionsText:        strings.TrimSpace(r.QuizQuestionsText),
		QuizQuestionsPoints:      points,
		QuizQuestionsAnswers:     ans,
		QuizQuestionsCorrect:     correct,
		QuizQuestionsExplanation: trimPtr(r.QuizQuestionsExplanation),
	}

	// Validasi bentuk data sebelum simpan (mirror CHECK DB)
	if err := m.ValidateShape(); err != nil {
		return nil, err
	}
	return m, nil
}

/* =========================================================
   PATCH (partial)
========================================================= */

type PatchQuizQuestionRequest struct {
	QuizQuestionsQuizID      UpdateField[uuid.UUID]               `json:"quiz_questions_quiz_id"`
	QuizQuestionsMasjidID    UpdateField[uuid.UUID]               `json:"quiz_questions_masjid_id"` // biasanya tidak diizinkan ubah; biarkan jika perlu
	QuizQuestionsType        UpdateField[qmodel.QuizQuestionType] `json:"quiz_questions_type"`       // single/essay
	QuizQuestionsText        UpdateField[string]                  `json:"quiz_questions_text"`
	QuizQuestionsPoints      UpdateField[float64]                 `json:"quiz_questions_points"`
	QuizQuestionsAnswers     UpdateField[json.RawMessage]         `json:"quiz_questions_answers"`    // object/array untuk SINGLE
	QuizQuestionsCorrect     UpdateField[string]                  `json:"quiz_questions_correct"`    // 'A'..'D' (OBJECT mode)
	QuizQuestionsExplanation UpdateField[string]                  `json:"quiz_questions_explanation"`
}

// Terapkan patch langsung ke model yg sudah di-load, lalu validasi shape.
func (p *PatchQuizQuestionRequest) ApplyToModel(m *qmodel.QuizQuestionModel) error {
	// IDs
	if p.QuizQuestionsQuizID.ShouldUpdate() && !p.QuizQuestionsQuizID.IsNull() {
		m.QuizQuestionsQuizID = p.QuizQuestionsQuizID.Val()
	}
	if p.QuizQuestionsMasjidID.ShouldUpdate() && !p.QuizQuestionsMasjidID.IsNull() {
		m.QuizQuestionsMasjidID = p.QuizQuestionsMasjidID.Val()
	}

	// Type
	if p.QuizQuestionsType.ShouldUpdate() && !p.QuizQuestionsType.IsNull() {
		m.QuizQuestionsType = p.QuizQuestionsType.Val()
	}

	// Text
	if p.QuizQuestionsText.ShouldUpdate() {
		if p.QuizQuestionsText.IsNull() {
			return errors.New("quiz_questions_text tidak boleh null")
		}
		m.QuizQuestionsText = strings.TrimSpace(p.QuizQuestionsText.Val())
	}

	// Points
	if p.QuizQuestionsPoints.ShouldUpdate() {
		if p.QuizQuestionsPoints.IsNull() {
			// set default (1)
			m.QuizQuestionsPoints = 1.0
		} else {
			m.QuizQuestionsPoints = p.QuizQuestionsPoints.Val()
		}
	}

	// Answers
	if p.QuizQuestionsAnswers.ShouldUpdate() {
		if p.QuizQuestionsAnswers.IsNull() {
			m.QuizQuestionsAnswers = nil
		} else {
			raw := p.QuizQuestionsAnswers.Val()
			m.QuizQuestionsAnswers = datatypes.JSON(raw)
		}
	}

	// Correct
	if p.QuizQuestionsCorrect.ShouldUpdate() {
		if p.QuizQuestionsCorrect.IsNull() {
			m.QuizQuestionsCorrect = nil
		} else {
			c := strings.ToUpper(strings.TrimSpace(p.QuizQuestionsCorrect.Val()))
			m.QuizQuestionsCorrect = &c
		}
	}

	// Explanation
	if p.QuizQuestionsExplanation.ShouldUpdate() {
		if p.QuizQuestionsExplanation.IsNull() {
			m.QuizQuestionsExplanation = nil
		} else {
			m.QuizQuestionsExplanation = trimPtr(&p.QuizQuestionsExplanation.value)
		}
	}

	// Final domain validation
	return m.ValidateShape()
}

/* =========================================================
   RESPONSE
========================================================= */

type QuizQuestionResponse struct {
	QuizQuestionsID          uuid.UUID               `json:"quiz_questions_id"`
	QuizQuestionsQuizID      uuid.UUID               `json:"quiz_questions_quiz_id"`
	QuizQuestionsMasjidID    uuid.UUID               `json:"quiz_questions_masjid_id"`
	QuizQuestionsType        qmodel.QuizQuestionType `json:"quiz_questions_type"`
	QuizQuestionsText        string                  `json:"quiz_questions_text"`
	QuizQuestionsPoints      float64                 `json:"quiz_questions_points"`
	QuizQuestionsAnswers     *json.RawMessage        `json:"quiz_questions_answers,omitempty"`
	QuizQuestionsCorrect     *string                 `json:"quiz_questions_correct,omitempty"`
	QuizQuestionsExplanation *string                 `json:"quiz_questions_explanation,omitempty"`

	QuizQuestionsCreatedAt string `json:"quiz_questions_created_at"`
	QuizQuestionsUpdatedAt string `json:"quiz_questions_updated_at"`
	// deleted_at sengaja tidak diekspos, atau bisa ditambahkan jika perlu
}

func FromModelQuizQuestion(m *qmodel.QuizQuestionModel) *QuizQuestionResponse {
	var ans *json.RawMessage
	if len(m.QuizQuestionsAnswers) > 0 {
		tmp := json.RawMessage(m.QuizQuestionsAnswers)
		ans = &tmp
	}
	return &QuizQuestionResponse{
		QuizQuestionsID:          m.QuizQuestionsID,
		QuizQuestionsQuizID:      m.QuizQuestionsQuizID,
		QuizQuestionsMasjidID:    m.QuizQuestionsMasjidID,
		QuizQuestionsType:        m.QuizQuestionsType,
		QuizQuestionsText:        m.QuizQuestionsText,
		QuizQuestionsPoints:      m.QuizQuestionsPoints,
		QuizQuestionsAnswers:     ans,
		QuizQuestionsCorrect:     m.QuizQuestionsCorrect,
		QuizQuestionsExplanation: m.QuizQuestionsExplanation,
		QuizQuestionsCreatedAt:   m.QuizQuestionsCreatedAt.UTC().Format(timeRFC3339),
		QuizQuestionsUpdatedAt:   m.QuizQuestionsUpdatedAt.UTC().Format(timeRFC3339),
	}
}

func FromModelsQuizQuestions(arr []qmodel.QuizQuestionModel) []*QuizQuestionResponse {
	out := make([]*QuizQuestionResponse, 0, len(arr))
	for i := range arr {
		out = append(out, FromModelQuizQuestion(&arr[i]))
	}
	return out
}

/* =========================================================
   Utils
========================================================= */

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	return &t
}
