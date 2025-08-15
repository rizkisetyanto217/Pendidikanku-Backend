// internals/features/lembaga/announcements/model/announcement_theme_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnnouncementThemeModel struct {
	AnnouncementThemesID       uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:announcement_themes_id" json:"announcement_themes_id"`
	AnnouncementThemesMasjidID uuid.UUID      `gorm:"type:uuid;not null;column:announcement_themes_masjid_id" json:"announcement_themes_masjid_id"`

	AnnouncementThemesName     string         `gorm:"size:80;not null;column:announcement_themes_name" json:"announcement_themes_name"`
	AnnouncementThemesSlug     string         `gorm:"size:120;not null;column:announcement_themes_slug" json:"announcement_themes_slug"`
	AnnouncementThemesDescription *string        `gorm:"type:text;column:announcement_themes_description" json:"announcement_themes_description,omitempty"`
	AnnouncementThemesColor    *string        `gorm:"size:20;column:announcement_themes_color" json:"announcement_themes_color,omitempty"`
	AnnouncementThemesIsActive bool           `gorm:"not null;default:true;column:announcement_themes_is_active" json:"announcement_themes_is_active"`

	AnnouncementThemesCreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;column:announcement_themes_created_at" json:"announcement_themes_created_at"`
	AnnouncementThemesUpdatedAt *time.Time     `gorm:"column:announcement_themes_updated_at" json:"announcement_themes_updated_at,omitempty"`
	AnnouncementThemesDeletedAt gorm.DeletedAt `gorm:"column:announcement_themes_deleted_at;index" json:"-"`
}

func (AnnouncementThemeModel) TableName() string { return "announcement_themes" }
