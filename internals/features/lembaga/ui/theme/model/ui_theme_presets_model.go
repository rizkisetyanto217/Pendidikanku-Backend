package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type UIThemePreset struct {
	UIThemePresetID   uuid.UUID      `gorm:"column:ui_theme_preset_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"ui_theme_preset_id"`
	UIThemePresetCode string         `gorm:"column:ui_theme_preset_code;size:64;unique;not null" json:"ui_theme_preset_code"`
	UIThemePresetName string         `gorm:"column:ui_theme_preset_name;size:128;not null" json:"ui_theme_preset_name"`

	UIThemePresetLight datatypes.JSON `gorm:"column:ui_theme_preset_light;type:jsonb;not null" json:"ui_theme_preset_light"`
	UIThemePresetDark  datatypes.JSON `gorm:"column:ui_theme_preset_dark;type:jsonb;not null" json:"ui_theme_preset_dark"`

	UIThemePresetCreatedAt time.Time  `gorm:"column:ui_theme_preset_created_at;not null;default:now()" json:"ui_theme_preset_created_at"`
	UIThemePresetUpdatedAt time.Time  `gorm:"column:ui_theme_preset_updated_at;not null;default:now()" json:"ui_theme_preset_updated_at"`
	UIThemePresetDeletedAt *time.Time `gorm:"column:ui_theme_preset_deleted_at" json:"ui_theme_preset_deleted_at,omitempty"`
}

func (UIThemePreset) TableName() string {
	return "ui_theme_presets"
}
