package model

import "time"

type Tooltip struct {
	TooltipID               uint      `gorm:"column:tooltip_id;primaryKey" json:"tooltip_id"`                                       // ID unik
	TooltipKeyword          string    `gorm:"column:tooltip_keyword;type:text;not null;unique" json:"tooltip_keyword"`              // Kata kunci tooltip
	TooltipDescriptionShort string    `gorm:"column:tooltip_description_short;type:text;not null" json:"tooltip_description_short"` // Deskripsi ringkas
	TooltipDescriptionLong  string    `gorm:"column:tooltip_description_long;type:text;not null" json:"tooltip_description_long"`   // Deskripsi panjang
	CreatedAt               time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                                   // Waktu dibuat
	UpdatedAt               time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                                   // Waktu update terakhir
}

// TableName memastikan nama tabel konsisten
func (Tooltip) TableName() string {
	return "tooltips"
}