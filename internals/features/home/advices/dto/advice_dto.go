package dto

import (
	"masjidku_backend/internals/features/home/advices/model"
	"time"
)

// ============================
// Response DTO
// ============================

type AdviceDTO struct {
	AdviceID          string    `json:"advice_id"`
	AdviceDescription string    `json:"advice_description"`
	AdviceLectureID   *string   `json:"advice_lecture_id"` // nullable
	AdviceUserID      string    `json:"advice_user_id"`
	AdviceCreatedAt   time.Time `json:"advice_created_at"`
}

// ============================
// Create Request DTO
// ============================

type CreateAdviceRequest struct {
	AdviceDescription string  `json:"advice_description" validate:"required,min=3"`
	AdviceLectureID   *string `json:"advice_lecture_id"` // optional
}

// ============================
// Converter
// ============================

func ToAdviceDTO(m model.AdviceModel) AdviceDTO {
	return AdviceDTO{
		AdviceID:          m.AdviceID,
		AdviceDescription: m.AdviceDescription,
		AdviceLectureID:   m.AdviceLectureID,
		AdviceUserID:      m.AdviceUserID,
		AdviceCreatedAt:   m.AdviceCreatedAt,
	}
}
