package model

import "time"

type QuoteModel struct {
	QuoteID      string    `gorm:"column:quote_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"quote_id"`
	QuoteText    string    `gorm:"column:quote_text;type:text;not null" json:"quote_text"`
	IsPublished  bool      `gorm:"column:is_published;default:false" json:"is_published"`
	DisplayOrder int       `gorm:"column:display_order" json:"display_order"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (QuoteModel) TableName() string {
	return "quotes"
}
