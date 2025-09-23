// internals/features/lembaga/announcements/model/announcement_theme_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnnouncementThemeModel struct {
	AnnouncementThemesID         uuid.UUID      `gorm:"column:announcement_themes_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"announcement_themes_id"`
	AnnouncementThemesMasjidID   uuid.UUID      `gorm:"column:announcement_themes_masjid_id;type:uuid;not null" json:"announcement_themes_masjid_id"`

	AnnouncementThemesName       string         `gorm:"column:announcement_themes_name;type:varchar(80);not null" json:"announcement_themes_name"`
	AnnouncementThemesSlug       string         `gorm:"column:announcement_themes_slug;type:varchar(120);not null" json:"announcement_themes_slug"`
	AnnouncementThemesDescription *string        `gorm:"column:announcement_themes_description;type:text" json:"announcement_themes_description,omitempty"`
	AnnouncementThemesColor      *string        `gorm:"column:announcement_themes_color;type:varchar(20)" json:"announcement_themes_color,omitempty"`
	AnnouncementThemesIsActive   bool           `gorm:"column:announcement_themes_is_active;not null;default:true" json:"announcement_themes_is_active"`

	// DDL: NOT NULL DEFAULT NOW()  → pakai time.Time + autoCreate/autoUpdate
	AnnouncementThemesCreatedAt  time.Time      `gorm:"column:announcement_themes_created_at;type:timestamptz;not null;autoCreateTime" json:"announcement_themes_created_at"`
	AnnouncementThemesUpdatedAt  time.Time      `gorm:"column:announcement_themes_updated_at;type:timestamptz;not null;autoUpdateTime" json:"announcement_themes_updated_at"`

	// DDL: nullable → pakai gorm.DeletedAt; index sudah diset di migration
	AnnouncementThemesDeletedAt  gorm.DeletedAt `gorm:"column:announcement_themes_deleted_at;index" json:"-"`
}

func (AnnouncementThemeModel) TableName() string { return "announcement_themes" }
