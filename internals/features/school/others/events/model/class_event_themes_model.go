// file: internals/features/school/events/themes/model/class_event_theme_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassEventTheme struct {
	ClassEventThemesID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_event_themes_id" json:"class_event_themes_id"`
	ClassEventThemesMasjidID    uuid.UUID `gorm:"type:uuid;not null;column:class_event_themes_masjid_id;uniqueIndex:uq_cet_masjid_code;index:idx_cet_masjid_active,priority:1;index:idx_cet_masjid_name,priority:1" json:"class_event_themes_masjid_id"`
	ClassEventThemesCode        string    `gorm:"type:varchar(64);not null;column:class_event_themes_code;uniqueIndex:uq_cet_masjid_code" json:"class_event_themes_code"`
	ClassEventThemesName        string    `gorm:"type:varchar(120);not null;column:class_event_themes_name;index:idx_cet_masjid_name,priority:2" json:"class_event_themes_name"`
	ClassEventThemesColor       *string   `gorm:"type:varchar(32);column:class_event_themes_color" json:"class_event_themes_color"`
	ClassEventThemesCustomColor *string   `gorm:"type:varchar(16);column:class_event_themes_custom_color" json:"class_event_themes_custom_color"`
	ClassEventThemesIsActive    bool      `gorm:"not null;default:true;column:class_event_themes_is_active;index:idx_cet_masjid_active,priority:2" json:"class_event_themes_is_active"`

	ClassEventThemesCreatedAt time.Time      `gorm:"column:class_event_themes_created_at;autoCreateTime" json:"class_event_themes_created_at"`
	ClassEventThemesUpdatedAt time.Time      `gorm:"column:class_event_themes_updated_at;autoUpdateTime" json:"class_event_themes_updated_at"`
	ClassEventThemesDeletedAt gorm.DeletedAt `gorm:"column:class_event_themes_deleted_at;index" json:"class_event_themes_deleted_at,omitempty"`
}

func (ClassEventTheme) TableName() string { return "class_event_themes" }
