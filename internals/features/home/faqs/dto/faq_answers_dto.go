package dto

import (
	"strings"
	"time"

	faqmodel "schoolku_backend/internals/features/home/faqs/model"
)

// ====================
// Response DTO
// ====================

type FaqAnswerDTO struct {
	FaqAnswerID         string     `json:"faq_answer_id"`
	FaqAnswerQuestionID string     `json:"faq_answer_question_id"`
	FaqAnswerAnsweredBy *string    `json:"faq_answer_answered_by,omitempty"` // nullable; diisi id/username penjawab jika ada
	AnsweredByName      *string    `json:"answered_by_name,omitempty"`       // opsional; human readable name dari relasi User
	FaqAnswerSchoolID   string     `json:"faq_answer_school_id"`
	FaqAnswerText       string     `json:"faq_answer_text"`
	FaqAnswerCreatedAt  time.Time  `json:"faq_answer_created_at"`
	FaqAnswerUpdatedAt  *time.Time `json:"faq_answer_updated_at,omitempty"` // nullable; hanya jika non-zero
}

// ====================
// Request DTO
// ====================

type CreateFaqAnswerRequest struct {
	FaqAnswerQuestionID string `json:"faq_answer_question_id" validate:"required,uuid"`
	FaqAnswerSchoolID   string `json:"faq_answer_school_id" validate:"required,uuid"`
	FaqAnswerText       string `json:"faq_answer_text" validate:"required,min=3"`
}

type UpdateFaqAnswerRequest struct {
	FaqAnswerText string `json:"faq_answer_text" validate:"required,min=3"`
}

// ====================
// Converters
// ====================

// Model → DTO (single)
func ToFaqAnswerDTO(m faqmodel.FaqAnswerModel) FaqAnswerDTO {
	var answeredByName *string

	// Relasi User opsional: pastikan di-preload saat query (Preload("User"))
	if m.User != nil {
		// Prioritas pakai UserName (string), fallback ke FullName (*string) bila ada
		if name := strings.TrimSpace(m.User.UserName); name != "" {
			n := name           // buat salinan lokal agar aman
			answeredByName = &n // pointer ke salinan lokal
		} else if m.User.FullName != nil {
			if fn := strings.TrimSpace(*m.User.FullName); fn != "" {
				answeredByName = m.User.FullName // sudah *string
			}
		}
	}

	// UpdatedAt optional → pointer hanya jika non-zero
	var updatedAtPtr *time.Time
	if !m.FaqAnswerUpdatedAt.IsZero() {
		ut := m.FaqAnswerUpdatedAt
		updatedAtPtr = &ut
	}

	return FaqAnswerDTO{
		FaqAnswerID:         m.FaqAnswerID,
		FaqAnswerQuestionID: m.FaqAnswerQuestionID,
		FaqAnswerAnsweredBy: m.FaqAnswerAnsweredBy, // *string
		AnsweredByName:      answeredByName,        // *string
		FaqAnswerSchoolID:   m.FaqAnswerSchoolID,
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
// `answeredBy` biasanya diisi dari token (username/id penjawab).
func (r CreateFaqAnswerRequest) ToModel(answeredBy *string) faqmodel.FaqAnswerModel {
	return faqmodel.FaqAnswerModel{
		FaqAnswerQuestionID: r.FaqAnswerQuestionID,
		FaqAnswerAnsweredBy: answeredBy, // *string
		FaqAnswerSchoolID:   r.FaqAnswerSchoolID,
		FaqAnswerText:       r.FaqAnswerText,
	}
}

// Update Request → apply ke model yang sudah di-fetch
func (r UpdateFaqAnswerRequest) ApplyToModel(m *faqmodel.FaqAnswerModel) {
	m.FaqAnswerText = r.FaqAnswerText
}
