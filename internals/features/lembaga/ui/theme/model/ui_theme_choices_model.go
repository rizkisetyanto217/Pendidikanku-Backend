package model

import (
	"time"

	"github.com/google/uuid"
)

type UIThemeChoice struct {
	UIThemeChoiceID              uuid.UUID  `gorm:"column:ui_theme_choice_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"ui_theme_choice_id"`

	UIThemeChoiceMasjidID        uuid.UUID  `gorm:"column:ui_theme_choice_masjid_id;type:uuid;not null;index" json:"ui_theme_choice_masjid_id"`

	// Exactly-one-of (sesuai CHECK di DB)
	UIThemeChoicePresetID        *uuid.UUID `gorm:"column:ui_theme_choice_preset_id;type:uuid" json:"ui_theme_choice_preset_id,omitempty"`
	UIThemeChoiceCustomPresetID  *uuid.UUID `gorm:"column:ui_theme_choice_custom_preset_id;type:uuid" json:"ui_theme_choice_custom_preset_id,omitempty"`

	UIThemeChoiceIsDefault       bool       `gorm:"column:ui_theme_choice_is_default;not null;default:false" json:"ui_theme_choice_is_default"`
	UIThemeChoiceIsEnabled       bool       `gorm:"column:ui_theme_choice_is_enabled;not null;default:true" json:"ui_theme_choice_is_enabled"`

	UIThemeChoiceCreatedAt       time.Time  `gorm:"column:ui_theme_choice_created_at;not null;default:now()" json:"ui_theme_choice_created_at"`
	UIThemeChoiceUpdatedAt       time.Time  `gorm:"column:ui_theme_choice_updated_at;not null;default:now()" json:"ui_theme_choice_updated_at"`
}

func (UIThemeChoice) TableName() string {
	return "ui_theme_choices"
}
