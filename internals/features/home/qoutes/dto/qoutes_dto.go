package dto

import (
	"time"

	"schoolku_backend/internals/features/home/qoutes/model"
)

// ============================
// Response DTO
// ============================

type QuoteDTO struct {
	QuoteID           string    `json:"quote_id"`
	QuoteText         string    `json:"quote_text"`
	QuoteIsPublished  bool      `json:"quote_is_published"`
	QuoteDisplayOrder *int      `json:"quote_display_order,omitempty"`
	QuoteCreatedAt    time.Time `json:"quote_created_at"`
	QuoteUpdatedAt    time.Time `json:"quote_updated_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateQuoteRequest struct {
	QuoteText         string `json:"quote_text" validate:"required,min=3"`
	QuoteIsPublished  bool   `json:"quote_is_published"`
	QuoteDisplayOrder *int   `json:"quote_display_order,omitempty"` // boleh null
}

// ============================
// Update Request DTO (partial)
// ============================

type UpdateQuoteRequest struct {
	QuoteText         *string `json:"quote_text,omitempty" validate:"omitempty,min=3"`
	QuoteIsPublished  *bool   `json:"quote_is_published,omitempty"`
	QuoteDisplayOrder *int    `json:"quote_display_order,omitempty"` // kirim null untuk clear
}

// ============================
//
// Converters
//
// ============================

func ToQuoteDTO(m model.QuoteModel) QuoteDTO {
	return QuoteDTO{
		QuoteID:           m.QuoteID,
		QuoteText:         m.QuoteText,
		QuoteIsPublished:  m.QuoteIsPublished,
		QuoteDisplayOrder: m.QuoteDisplayOrder,
		QuoteCreatedAt:    m.QuoteCreatedAt,
		QuoteUpdatedAt:    m.QuoteUpdatedAt,
	}
}

func (r *CreateQuoteRequest) ToModel() *model.QuoteModel {
	return &model.QuoteModel{
		QuoteText:         r.QuoteText,
		QuoteIsPublished:  r.QuoteIsPublished,
		QuoteDisplayOrder: r.QuoteDisplayOrder, // bisa nil
	}
}
