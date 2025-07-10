package dto

import (
	"masjidku_backend/internals/features/home/qoutes/model"
	"time"
)

// ============================
// Response DTO
// ============================

type QuoteDTO struct {
	QuoteID      string    `json:"quote_id"`
	QuoteText    string    `json:"quote_text"`
	IsPublished  bool      `json:"is_published"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateQuoteRequest struct {
	QuoteText    string `json:"quote_text" validate:"required,min=3"`
	IsPublished  bool   `json:"is_published"`
	DisplayOrder int    `json:"display_order"`
}

// ============================
// Update Request DTO
// ============================

type UpdateQuoteRequest struct {
	QuoteText    string `json:"quote_text" validate:"required,min=3"`
	IsPublished  bool   `json:"is_published"`
	DisplayOrder int    `json:"display_order"`
}

// ============================
// Converter
// ============================

func ToQuoteDTO(m model.QuoteModel) QuoteDTO {
	return QuoteDTO{
		QuoteID:      m.QuoteID,
		QuoteText:    m.QuoteText,
		IsPublished:  m.IsPublished,
		DisplayOrder: m.DisplayOrder,
		CreatedAt:    m.CreatedAt,
	}
}

func (r *CreateQuoteRequest) ToModel() *model.QuoteModel {
	return &model.QuoteModel{
		QuoteText:    r.QuoteText,
		IsPublished:  r.IsPublished,
		DisplayOrder: r.DisplayOrder,
	}
}
