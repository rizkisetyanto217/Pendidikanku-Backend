// file: internals/features/school/submissions_assesments/quizzes/dto/quiz_inline_dto.go
package dto

import (
	"strings"

	"github.com/google/uuid"

	model "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
)

/* ========================================================
   Helpers (pakai trimPtr yang sudah ada di file ini)
   ======================================================== */

// NOTE: trimPtr sudah ada di file DTO kamu:
// func trimPtr(s *string) *string { ... }
func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

/* ========================================================
   Inline DTO: dipakai bareng Assessment
   ======================================================== */

type CreateQuizInline struct {
	QuizSlug        *string `json:"quiz_slug" validate:"omitempty,max=160"`
	QuizTitle       string  `json:"quiz_title" validate:"required,max=180"`
	QuizDescription *string `json:"quiz_description" validate:"omitempty"`

	QuizIsPublished  *bool `json:"quiz_is_published" validate:"omitempty"`
	QuizTimeLimitSec *int  `json:"quiz_time_limit_sec" validate:"omitempty,gte=0"`
}

// Normalize: trim string & fallback title minimal
func (q *CreateQuizInline) Normalize() {
	q.QuizSlug = trimPtr(q.QuizSlug)
	q.QuizDescription = trimPtr(q.QuizDescription)
	q.QuizTitle = strings.TrimSpace(q.QuizTitle)
}

// ToModel: build QuizModel dari inline DTO + school & assessment
func (q *CreateQuizInline) ToModel(schoolID uuid.UUID, assessmentID uuid.UUID) *model.QuizModel {
	isPub := false
	if q.QuizIsPublished != nil {
		isPub = *q.QuizIsPublished
	}

	return &model.QuizModel{
		QuizSchoolID:     schoolID,
		QuizAssessmentID: &assessmentID,

		QuizSlug:        trimPtr(q.QuizSlug),
		QuizTitle:       strings.TrimSpace(q.QuizTitle),
		QuizDescription: trimPtr(q.QuizDescription),

		QuizIsPublished:  isPub,
		QuizTimeLimitSec: q.QuizTimeLimitSec,
	}
}
