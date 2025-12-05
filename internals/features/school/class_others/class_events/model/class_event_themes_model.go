// file: internals/features/school/class_events/model/class_event_theme_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassEventThemeModel struct {
	ClassEventThemeID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_event_theme_id" json:"class_event_theme_id"`
	ClassEventThemeSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_event_theme_school_id" json:"class_event_theme_school_id"`

	// Identitas tema
	ClassEventThemeCode string `gorm:"type:varchar(64);not null;column:class_event_theme_code" json:"class_event_theme_code"`
	ClassEventThemeName string `gorm:"type:varchar(120);not null;column:class_event_theme_name" json:"class_event_theme_name"`

	// Warna: preset atau custom hex
	ClassEventThemeColor       *string `gorm:"type:varchar(32);column:class_event_theme_color" json:"class_event_theme_color,omitempty"`
	ClassEventThemeCustomColor *string `gorm:"type:varchar(16);column:class_event_theme_custom_color" json:"class_event_theme_custom_color,omitempty"`

	ClassEventThemeIsActive bool `gorm:"not null;default:true;column:class_event_theme_is_active" json:"class_event_theme_is_active"`

	// Audit
	ClassEventThemeCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:class_event_theme_created_at" json:"class_event_theme_created_at"`
	ClassEventThemeUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:class_event_theme_updated_at" json:"class_event_theme_updated_at"`
	ClassEventThemeDeletedAt gorm.DeletedAt `gorm:"column:class_event_theme_deleted_at;index" json:"class_event_theme_deleted_at,omitempty"`
}

func (ClassEventThemeModel) TableName() string { return "class_event_themes" }
