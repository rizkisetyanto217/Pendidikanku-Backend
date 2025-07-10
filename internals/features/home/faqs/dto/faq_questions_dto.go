package dto

import (
	"masjidku_backend/internals/features/home/faqs/model"
	"time"
)

// ====================
// Response DTO
// ====================

type FaqQuestionDTO struct {
	FaqQuestionID               string    `json:"faq_question_id"`
	FaqQuestionUserID           string    `json:"faq_question_user_id"`
	FaqQuestionText             string    `json:"faq_question_text"`
	FaqQuestionLectureID        *string   `json:"faq_question_lecture_id,omitempty"`
	FaqQuestionLectureSessionID *string   `json:"faq_question_lecture_session_id,omitempty"`
	FaqQuestionIsAnswered       bool      `json:"faq_question_is_answered"`
	FaqQuestionCreatedAt        time.Time `json:"faq_question_created_at"`
}

// ====================
// Request DTO
// ====================

type CreateFaqQuestionRequest struct {
	FaqQuestionUserID           string  `json:"faq_question_user_id" validate:"required,uuid"`
	FaqQuestionText             string  `json:"faq_question_text" validate:"required,min=5"`
	FaqQuestionLectureID        *string `json:"faq_question_lecture_id,omitempty"`
	FaqQuestionLectureSessionID *string `json:"faq_question_lecture_session_id,omitempty"`
}

type UpdateFaqQuestionRequest struct {
	FaqQuestionText             string  `json:"faq_question_text" validate:"required,min=5"`
	FaqQuestionLectureID        *string `json:"faq_question_lecture_id,omitempty"`
	FaqQuestionLectureSessionID *string `json:"faq_question_lecture_session_id,omitempty"`
	FaqQuestionIsAnswered       *bool   `json:"faq_question_is_answered,omitempty"`
}

// ====================
// Converter
// ====================

func ToFaqQuestionDTO(f model.FaqQuestionModel) FaqQuestionDTO {
	return FaqQuestionDTO{
		FaqQuestionID:               f.FaqQuestionID,
		FaqQuestionUserID:           f.FaqQuestionUserID,
		FaqQuestionText:             f.FaqQuestionText,
		FaqQuestionLectureID:        f.FaqQuestionLectureID,
		FaqQuestionLectureSessionID: f.FaqQuestionLectureSessionID,
		FaqQuestionIsAnswered:       f.FaqQuestionIsAnswered,
		FaqQuestionCreatedAt:        f.FaqQuestionCreatedAt,
	}
}

func (r CreateFaqQuestionRequest) ToModel(userID string) model.FaqQuestionModel {
	return model.FaqQuestionModel{
		FaqQuestionUserID:           userID,
		FaqQuestionText:             r.FaqQuestionText,
		FaqQuestionLectureID:        r.FaqQuestionLectureID,
		FaqQuestionLectureSessionID: r.FaqQuestionLectureSessionID,
	}
}
