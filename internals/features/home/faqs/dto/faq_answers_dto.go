package dto

import (
	"masjidku_backend/internals/features/home/faqs/model"
	"time"
)

// ====================
// Response DTO
// ====================

type FaqAnswerDTO struct {
	FaqAnswerID         string    `json:"faq_answer_id"`
	FaqAnswerQuestionID string    `json:"faq_answer_question_id"`
	FaqAnswerAnsweredBy string    `json:"faq_answer_answered_by"`
	FaqAnswerText       string    `json:"faq_answer_text"`
	FaqAnswerCreatedAt  time.Time `json:"faq_answer_created_at"`
}

// ====================
// Request DTO
// ====================

type CreateFaqAnswerRequest struct {
	FaqAnswerQuestionID string `json:"faq_answer_question_id" validate:"required,uuid"`
	FaqAnswerText       string `json:"faq_answer_text" validate:"required,min=3"`
}

type UpdateFaqAnswerRequest struct {
	FaqAnswerText string `json:"faq_answer_text" validate:"required,min=3"`
}

// ====================
// Converter: Model → DTO
// ====================

func ToFaqAnswerDTO(m model.FaqAnswerModel) FaqAnswerDTO {
	return FaqAnswerDTO{
		FaqAnswerID:         m.FaqAnswerID,
		FaqAnswerQuestionID: m.FaqAnswerQuestionID,
		FaqAnswerAnsweredBy: m.FaqAnswerAnsweredBy,
		FaqAnswerText:       m.FaqAnswerText,
		FaqAnswerCreatedAt:  m.FaqAnswerCreatedAt,
	}
}

// ====================
// Converter: Request → Model
// ====================

func (r CreateFaqAnswerRequest) ToModel(answeredBy string) model.FaqAnswerModel {
	return model.FaqAnswerModel{
		FaqAnswerQuestionID: r.FaqAnswerQuestionID,
		FaqAnswerAnsweredBy: answeredBy,
		FaqAnswerText:       r.FaqAnswerText,
	}
}
