package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type UIThemeCustomPreset struct {
	UIThemeCustomPresetID uuid.UUID `gorm:"column:ui_theme_custom_preset_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"ui_theme_custom_preset_id"`

	UIThemeCustomPresetSchoolID uuid.UUID `gorm:"column:ui_theme_custom_preset_school_id;type:uuid;not null;index:ux_custom_preset_tenant_code,unique" json:"ui_theme_custom_preset_school_id"`
	UIThemeCustomPresetCode     string    `gorm:"column:ui_theme_custom_preset_code;size:64;not null;index:ux_custom_preset_tenant_code,unique" json:"ui_theme_custom_preset_code"`
	UIThemeCustomPresetName     string    `gorm:"column:ui_theme_custom_preset_name;size:128;not null" json:"ui_theme_custom_preset_name"`

	UIThemeCustomPresetLight datatypes.JSON `gorm:"column:ui_theme_custom_preset_light;type:jsonb;not null" json:"ui_theme_custom_preset_light"`
	UIThemeCustomPresetDark  datatypes.JSON `gorm:"column:ui_theme_custom_preset_dark;type:jsonb;not null" json:"ui_theme_custom_preset_dark"`

	UIThemeCustomBasePresetID *uuid.UUID `gorm:"column:ui_theme_custom_base_preset_id;type:uuid" json:"ui_theme_custom_base_preset_id,omitempty"`

	UIThemeCustomPresetIsActive bool `gorm:"column:ui_theme_custom_preset_is_active;not null;default:true" json:"ui_theme_custom_preset_is_active"`

	UIThemeCustomPresetCreatedAt time.Time `gorm:"column:ui_theme_custom_preset_created_at;not null;default:now()" json:"ui_theme_custom_preset_created_at"`
	UIThemeCustomPresetUpdatedAt time.Time `gorm:"column:ui_theme_custom_preset_updated_at;not null;default:now()" json:"ui_theme_custom_preset_updated_at"`
}

func (UIThemeCustomPreset) TableName() string {
	return "ui_theme_custom_presets"
}
