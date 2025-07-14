package dto

import (
	"masjidku_backend/internals/features/home/posts/model"
	"time"
)

// ============================
// Response DTO
// ============================

type PostThemeDTO struct {
	PostThemeID          string    `json:"post_theme_id"`
	PostThemeName        string    `json:"post_theme_name"`
	PostThemeDescription string    `json:"post_theme_description"`
	PostThemeMasjidID    string    `json:"post_theme_masjid_id"`
	PostThemeCreatedAt   time.Time `json:"post_theme_created_at"`
}

// ============================
// Create Request DTO
// ============================

type CreatePostThemeRequest struct {
	PostThemeName        string `json:"post_theme_name" validate:"required,min=3"`
	PostThemeDescription string `json:"post_theme_description"`
	PostThemeMasjidID    string `json:"post_theme_masjid_id" validate:"required,uuid"`
}

// ============================
// Update Request DTO
// ============================

type UpdatePostThemeRequest struct {
	PostThemeName        string `json:"post_theme_name" validate:"required,min=3"`
	PostThemeDescription string `json:"post_theme_description"`
}

// ============================
// Converter
// ============================

func ToPostThemeDTO(m model.PostThemeModel) PostThemeDTO {
	return PostThemeDTO{
		PostThemeID:          m.PostThemeID,
		PostThemeName:        m.PostThemeName,
		PostThemeDescription: m.PostThemeDescription,
		PostThemeMasjidID:    m.PostThemeMasjidID,
		PostThemeCreatedAt:   m.PostThemeCreatedAt,
	}
}
