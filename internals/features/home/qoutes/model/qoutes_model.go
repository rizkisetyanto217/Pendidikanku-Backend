package model

import (
	"time"

	"gorm.io/gorm"
)

type QuoteModel struct {
	QuoteID          string         `gorm:"column:quote_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"quote_id"`
	QuoteText        string         `gorm:"column:quote_text;type:text;not null" json:"quote_text"`
	QuoteIsPublished bool           `gorm:"column:quote_is_published;default:false" json:"quote_is_published"`
	QuoteDisplayOrder *int          `gorm:"column:quote_display_order" json:"quote_display_order,omitempty"`

	QuoteCreatedAt time.Time      `gorm:"column:quote_created_at;autoCreateTime" json:"quote_created_at"`
	QuoteUpdatedAt time.Time      `gorm:"column:quote_updated_at;autoUpdateTime" json:"quote_updated_at"`
	QuoteDeletedAt gorm.DeletedAt `gorm:"column:quote_deleted_at;index" json:"-"`
}

func (QuoteModel) TableName() string {
	return "quotes"
}
