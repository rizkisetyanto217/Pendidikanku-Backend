package model

import (
	MasjidModel "masjidku_backend/internals/features/masjids/masjids/model"
	"time"
)

type PostThemeModel struct {
	PostThemeID          string    `gorm:"column:post_theme_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"post_theme_id"`
	PostThemeName        string    `gorm:"column:post_theme_name;type:varchar(100);not null" json:"post_theme_name"`
	PostThemeDescription string    `gorm:"column:post_theme_description;type:text" json:"post_theme_description"`

	PostThemeMasjidID    string    `gorm:"column:post_theme_masjid_id;type:uuid;not null" json:"post_theme_masjid_id"`

	PostThemeCreatedAt   time.Time `gorm:"column:post_theme_created_at;autoCreateTime" json:"post_theme_created_at"`

	// Optional: relation
	Masjid *MasjidModel.MasjidModel `gorm:"foreignKey:PostThemeMasjidID"`
	// Posts  []PostModel              `gorm:"foreignKey:PostThemeID"` // if you want reverse relation
}

func (PostThemeModel) TableName() string {
	return "post_themes"
}
