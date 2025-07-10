package dto

import (
	"masjidku_backend/internals/features/home/articles/model"
	"time"
)

type CarouselResponse struct {
	CarouselID        string          `json:"carousel_id"`
	CarouselTitle     string          `json:"carousel_title"`
	CarouselCaption   string          `json:"carousel_caption"`
	CarouselImageURL  string          `json:"carousel_image_url"`
	CarouselTargetURL string          `json:"carousel_target_url"`
	CarouselType      string          `json:"carousel_type"`
	CarouselOrder     int             `json:"carousel_order"`
	CarouselIsActive  bool            `json:"carousel_is_active"`
	CarouselArticleID *string         `json:"carousel_article_id,omitempty"`
	Article           *ArticlePreview `json:"article,omitempty"`
	CarouselCreatedAt string          `json:"carousel_created_at"`
	CarouselUpdatedAt string          `json:"carousel_updated_at"`
}

type ArticlePreview struct {
	ArticleID          string `json:"article_id"`
	ArticleTitle       string `json:"article_title"`
	ArticleDescription string `json:"article_description"`
	ArticleImageURL    string `json:"article_image_url"`
}

func ConvertCarouselToDTO(c model.CarouselModel) CarouselResponse {
	var article *ArticlePreview
	if c.Article != nil {
		article = &ArticlePreview{
			ArticleID:          c.Article.ArticleID,
			ArticleTitle:       c.Article.ArticleTitle,
			ArticleDescription: c.Article.ArticleDescription,
			ArticleImageURL:    c.Article.ArticleImageURL,
		}
	}

	var articleID *string
	if c.CarouselArticleID != nil {
		str := c.CarouselArticleID.String()
		articleID = &str
	}

	return CarouselResponse{
		CarouselID:        c.CarouselID.String(),
		CarouselTitle:     c.CarouselTitle,
		CarouselCaption:   c.CarouselCaption,
		CarouselImageURL:  c.CarouselImageURL,
		CarouselTargetURL: c.CarouselTargetURL,
		CarouselType:      c.CarouselType,
		CarouselOrder:     c.CarouselOrder,
		CarouselIsActive:  c.CarouselIsActive,
		CarouselArticleID: articleID,
		Article:           article,
		CarouselCreatedAt: c.CarouselCreatedAt.Format(time.RFC3339),
		CarouselUpdatedAt: c.CarouselUpdatedAt.Format(time.RFC3339),
	}
}

func ConvertCarouselListToDTO(carousels []model.CarouselModel) []CarouselResponse {
	result := make([]CarouselResponse, 0, len(carousels))
	for _, c := range carousels {
		result = append(result, ConvertCarouselToDTO(c))
	}
	return result
}
