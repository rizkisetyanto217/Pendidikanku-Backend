package model

import "time"

type ArticleModel struct {
	ArticleID          string    `gorm:"column:article_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"article_id"`
	ArticleTitle       string    `gorm:"column:article_title;type:varchar(255);not null" json:"article_title"`
	ArticleDescription string    `gorm:"column:article_description;type:text;not null" json:"article_description"`
	ArticleImageURL    string    `gorm:"column:article_image_url;type:text" json:"article_image_url"`
	ArticleOrderID     int       `gorm:"column:article_order_id" json:"article_order_id"`
	ArticleCreatedAt   time.Time `gorm:"column:article_created_at;autoCreateTime" json:"article_created_at"`
	ArticleUpdatedAt   time.Time `gorm:"column:article_updated_at;autoUpdateTime" json:"article_updated_at"`
}

// TableName sets the table name for ArticleModel
func (ArticleModel) TableName() string {
	return "articles"
}
