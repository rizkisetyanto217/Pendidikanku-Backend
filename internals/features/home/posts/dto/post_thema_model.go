package dto

import (
	"schoolku_backend/internals/features/home/posts/model"
	"time"
)

// ============================
// Response DTO
// ============================

type PostThemeDTO struct {
	PostThemeID          string    `json:"post_theme_id"`
	PostThemeName        string    `json:"post_theme_name"`
	PostThemeDescription string    `json:"post_theme_description"`
	PostThemeSchoolID    string    `json:"post_theme_school_id"`
	PostThemeCreatedAt   time.Time `json:"post_theme_created_at"`
}

// ============================
// Create Request DTO
// ============================

type CreatePostThemeRequest struct {
	PostThemeName        string `json:"post_theme_name" validate:"required,min=3"`
	PostThemeDescription string `json:"post_theme_description"`
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
		PostThemeSchoolID:    m.PostThemeSchoolID,
		PostThemeCreatedAt:   m.PostThemeCreatedAt,
	}
}
