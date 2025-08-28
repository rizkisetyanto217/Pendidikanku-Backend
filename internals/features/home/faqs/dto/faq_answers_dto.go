package dto

import (
	"time"

	faqmodel "masjidku_backend/internals/features/home/faqs/model"
)

// ====================
// Response DTO
// ====================

type FaqAnswerDTO struct {
	FaqAnswerID         string     `json:"faq_answer_id"`
	FaqAnswerQuestionID string     `json:"faq_answer_question_id"`
	FaqAnswerAnsweredBy *string    `json:"faq_answer_answered_by,omitempty"` // nullable
	AnsweredByName      *string    `json:"answered_by_name,omitempty"`       // opsional, dari relasi User
	FaqAnswerMasjidID   string     `json:"faq_answer_masjid_id"`
	FaqAnswerText       string     `json:"faq_answer_text"`
	FaqAnswerCreatedAt  time.Time  `json:"faq_answer_created_at"`
	FaqAnswerUpdatedAt  *time.Time `json:"faq_answer_updated_at,omitempty"` // opsional kalau mau tampilkan updated_at
}

// ====================
// Request DTO
// ====================

type CreateFaqAnswerRequest struct {
	FaqAnswerQuestionID string `json:"faq_answer_question_id" validate:"required,uuid"`
	FaqAnswerMasjidID   string `json:"faq_answer_masjid_id" validate:"required,uuid"`
	FaqAnswerText       string `json:"faq_answer_text" validate:"required,min=3"`
}

type UpdateFaqAnswerRequest struct {
	FaqAnswerText string `json:"faq_answer_text" validate:"required,min=3"`
}

// ====================
/* Converters */
// ====================

// Model → DTO (single)
func ToFaqAnswerDTO(m faqmodel.FaqAnswerModel) FaqAnswerDTO {
	var answeredByName *string
	if m.User != nil {
		// Pilih field nama yang kamu pakai di user; contoh: UserName
		if m.User.UserName != "" {
			n := m.User.UserName
			answeredByName = &n
		} else if m.User.FullName != "" {
			n := m.User.FullName
			answeredByName = &n
		}
	}

	// UpdatedAt optional: kalau zero value, biarkan nil
	var updatedAtPtr *time.Time
	if !m.FaqAnswerUpdatedAt.IsZero() {
		ut := m.FaqAnswerUpdatedAt
		updatedAtPtr = &ut
	}

	return FaqAnswerDTO{
		FaqAnswerID:         m.FaqAnswerID,
		FaqAnswerQuestionID: m.FaqAnswerQuestionID,
		FaqAnswerAnsweredBy: m.FaqAnswerAnsweredBy, // *string
		AnsweredByName:      answeredByName,
		FaqAnswerMasjidID:   m.FaqAnswerMasjidID,
		FaqAnswerText:       m.FaqAnswerText,
		FaqAnswerCreatedAt:  m.FaqAnswerCreatedAt,
		FaqAnswerUpdatedAt:  updatedAtPtr,
	}
}

// Model list → DTO list
func ToFaqAnswerDTOs(list []faqmodel.FaqAnswerModel) []FaqAnswerDTO {
	out := make([]FaqAnswerDTO, 0, len(list))
	for _, m := range list {
		out = append(out, ToFaqAnswerDTO(m))
	}
	return out
}

// Create Request → Model
// `answeredBy` ambil dari token; boleh nil jika sistem mengizinkan jawaban anonim/moderator sistem
func (r CreateFaqAnswerRequest) ToModel(answeredBy *string) faqmodel.FaqAnswerModel {
	return faqmodel.FaqAnswerModel{
		FaqAnswerQuestionID: r.FaqAnswerQuestionID,
		FaqAnswerAnsweredBy: answeredBy, // *string
		FaqAnswerMasjidID:   r.FaqAnswerMasjidID,
		FaqAnswerText:       r.FaqAnswerText,
	}
}

// Update Request → apply ke model yang sudah di-fetch
func (r UpdateFaqAnswerRequest) ApplyToModel(m *faqmodel.FaqAnswerModel) {
	m.FaqAnswerText = r.FaqAnswerText
}
