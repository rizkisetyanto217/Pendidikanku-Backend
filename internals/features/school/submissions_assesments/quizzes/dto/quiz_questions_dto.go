// file: internals/features/school/submissions_assesments/quizzes/dto/quiz_question_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	qmodel "schoolku_backend/internals/features/school/submissions_assesments/quizzes/model"
)

/* =========================================================
   CREATE
========================================================= */

// SINGLE: isi answers (object/array) + correct ('A'..'D').
// ESSAY : biarkan answers & correct kosong.
type CreateQuizQuestionRequest struct {
	QuizQuestionQuizID      uuid.UUID               `json:"quiz_question_quiz_id" validate:"required"`
	QuizQuestionSchoolID    uuid.UUID               `json:"quiz_question_school_id"` // controller boleh force override dari tenant
	QuizQuestionType        qmodel.QuizQuestionType `json:"quiz_question_type" validate:"required,oneof=single essay"`
	QuizQuestionText        string                  `json:"quiz_question_text" validate:"required"`
	QuizQuestionPoints      *float64                `json:"quiz_question_points" validate:"omitempty,gte=0"`
	QuizQuestionAnswers     *json.RawMessage        `json:"quiz_question_answers" validate:"omitempty"` // object/array (SINGLE) atau null
	QuizQuestionCorrect     *string                 `json:"quiz_question_correct" validate:"omitempty,oneof=A B C D a b c d"`
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
		c := strings.ToUpper(strings.TrimSpace(*r.QuizQuestionCorrect))
		correct = &c
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
	}

	// Jika model punya validator domain-level, panggil di sini.
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
	QuizQuestionAnswers     UpdateField[json.RawMessage]         `json:"quiz_question_answers"` // object/array untuk SINGLE
	QuizQuestionCorrect     UpdateField[string]                  `json:"quiz_question_correct"` // 'A'..'D' (OBJECT mode)
	QuizQuestionExplanation UpdateField[string]                  `json:"quiz_question_explanation"`
}

// Terapkan patch langsung ke model yang sudah di-load, lalu validasi shape.
func (p *PatchQuizQuestionRequest) ApplyToModel(m *qmodel.QuizQuestionModel) error {
	// IDs
	if p.QuizQuestionQuizID.ShouldUpdate() && !p.QuizQuestionQuizID.IsNull() {
		m.QuizQuestionQuizID = p.QuizQuestionQuizID.Val()
	}
	if p.QuizQuestionSchoolID.ShouldUpdate() && !p.QuizQuestionSchoolID.IsNull() {
		m.QuizQuestionSchoolID = p.QuizQuestionSchoolID.Val()
	}

	// Type
	if p.QuizQuestionType.ShouldUpdate() && !p.QuizQuestionType.IsNull() {
		m.QuizQuestionType = p.QuizQuestionType.Val()
	}

	// Text
	if p.QuizQuestionText.ShouldUpdate() {
		if p.QuizQuestionText.IsNull() {
			return errors.New("quiz_question_text tidak boleh null")
		}
		m.QuizQuestionText = strings.TrimSpace(p.QuizQuestionText.Val())
	}

	// Points
	if p.QuizQuestionPoints.ShouldUpdate() {
		if p.QuizQuestionPoints.IsNull() {
			m.QuizQuestionPoints = 1.0
		} else {
			m.QuizQuestionPoints = p.QuizQuestionPoints.Val()
		}
	}

	// Answers
	if p.QuizQuestionAnswers.ShouldUpdate() {
		if p.QuizQuestionAnswers.IsNull() {
			m.QuizQuestionAnswers = nil
		} else {
			raw := p.QuizQuestionAnswers.Val()
			m.QuizQuestionAnswers = datatypes.JSON(raw)
		}
	}

	// Correct
	if p.QuizQuestionCorrect.ShouldUpdate() {
		if p.QuizQuestionCorrect.IsNull() {
			m.QuizQuestionCorrect = nil
		} else {
			c := strings.ToUpper(strings.TrimSpace(p.QuizQuestionCorrect.Val()))
			m.QuizQuestionCorrect = &c
		}
	}

	// Explanation
	if p.QuizQuestionExplanation.ShouldUpdate() {
		if p.QuizQuestionExplanation.IsNull() {
			m.QuizQuestionExplanation = nil
		} else {
			v := p.QuizQuestionExplanation.Val()
			m.QuizQuestionExplanation = trimPtr(&v)
		}
	}

	// Final domain validation
	return m.ValidateShape()
}

/* =========================================================
   RESPONSE
========================================================= */

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
}

func FromModelQuizQuestion(m *qmodel.QuizQuestionModel) *QuizQuestionResponse {
	var ans *json.RawMessage
	if len(m.QuizQuestionAnswers) > 0 {
		tmp := json.RawMessage(m.QuizQuestionAnswers)
		ans = &tmp
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
