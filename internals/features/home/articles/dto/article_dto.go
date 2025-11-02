package dto

import (
	"schoolku_backend/internals/features/home/articles/model"
	"time"
)

// ============================
// Response DTO
// ============================

type ArticleDTO struct {
	ArticleID          string     `json:"article_id"`
	ArticleTitle       string     `json:"article_title"`
	ArticleDescription string     `json:"article_description"`
	ArticleImageURL    string     `json:"article_image_url"`
	ArticleOrderID     int        `json:"article_order_id"`
	ArticleSchoolID    string     `json:"article_school_id"`
	ArticleCreatedAt   time.Time  `json:"article_created_at"`
	ArticleUpdatedAt   time.Time  `json:"article_updated_at"`
	ArticleDeletedAt   *time.Time `json:"article_deleted_at,omitempty"`
}

// ============================
// Create & Update Request DTO
// ============================

type CreateArticleRequest struct {
	ArticleTitle       string `json:"article_title" validate:"required,min=3"`
	ArticleDescription string `json:"article_description" validate:"required"`
	ArticleImageURL    string `json:"article_image_url"`
	ArticleOrderID     int    `json:"article_order_id"`
	ArticleSchoolID    string `json:"article_school_id" validate:"required,uuid"`
}

type UpdateArticleRequest struct {
	ArticleTitle       string `json:"article_title" validate:"required,min=3"`
	ArticleDescription string `json:"article_description" validate:"required"`
	ArticleImageURL    string `json:"article_image_url"`
	ArticleOrderID     int    `json:"article_order_id"`
	ArticleSchoolID    string `json:"article_school_id" validate:"required,uuid"`
}

// ============================
// Converter
// ============================

func ToArticleDTO(m model.ArticleModel) ArticleDTO {
	return ArticleDTO{
		ArticleID:          m.ArticleID,
		ArticleTitle:       m.ArticleTitle,
		ArticleDescription: m.ArticleDescription,
		ArticleImageURL:    m.ArticleImageURL,
		ArticleOrderID:     m.ArticleOrderID,
		ArticleSchoolID:    m.ArticleSchoolID,
		ArticleCreatedAt:   m.ArticleCreatedAt,
		ArticleUpdatedAt:   m.ArticleUpdatedAt,
		ArticleDeletedAt:   m.ArticleDeletedAt,
	}
}
