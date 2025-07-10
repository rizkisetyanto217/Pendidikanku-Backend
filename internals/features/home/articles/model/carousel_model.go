package model

import (
	"time"

	"github.com/google/uuid"
)

type CarouselModel struct {
	CarouselID         uuid.UUID  `gorm:"column:carousel_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"carousel_id"`
	CarouselTitle      string     `gorm:"column:carousel_title;type:varchar(255)" json:"carousel_title"`
	CarouselCaption    string     `gorm:"column:carousel_caption;type:text" json:"carousel_caption"`
	CarouselImageURL   string     `gorm:"column:carousel_image_url;type:text;not null" json:"carousel_image_url"`
	CarouselTargetURL  string     `gorm:"column:carousel_target_url;type:text" json:"carousel_target_url"`
	CarouselType       string     `gorm:"column:carousel_type;type:varchar(50)" json:"carousel_type"`
	CarouselOrder      int        `gorm:"column:carousel_order" json:"carousel_order"`
	CarouselIsActive   bool       `gorm:"column:carousel_is_active;default:true" json:"carousel_is_active"`
	CarouselArticleID  *uuid.UUID `gorm:"column:carousel_article_id;type:uuid" json:"carousel_article_id"` // optional

	// Optional relasi ke artikel (preload jika dibutuhkan)
	Article *ArticleModel `gorm:"foreignKey:CarouselArticleID;references:ArticleID" json:"article,omitempty"`

	CarouselCreatedAt time.Time `gorm:"column:carousel_created_at;autoCreateTime" json:"carousel_created_at"`
	CarouselUpdatedAt time.Time `gorm:"column:carousel_updated_at;autoUpdateTime" json:"carousel_updated_at"`
}

func (CarouselModel) TableName() string {
	return "carousels"
}
