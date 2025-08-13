// internals/features/lembaga/classes/sections/main/model/class_section_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ClassSectionModel struct {
	ClassSectionID uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_sections_id" json:"class_sections_id"`
	ClassID        uuid.UUID      `gorm:"type:uuid;not null;column:class_sections_class_id" json:"class_sections_class_id"`
	MasjidID       *uuid.UUID     `gorm:"type:uuid;column:class_sections_masjid_id" json:"class_sections_masjid_id,omitempty"`

	Slug     string         `gorm:"size:160;uniqueIndex:idx_sections_slug;not null;column:class_sections_slug" json:"class_sections_slug"`
	Name     string         `gorm:"size:100;not null;column:class_sections_name" json:"class_sections_name"`
	Code     *string        `gorm:"size:50;column:class_sections_code" json:"class_sections_code,omitempty"`
	Capacity *int           `gorm:"column:class_sections_capacity" json:"class_sections_capacity,omitempty"`
	Schedule datatypes.JSON `gorm:"type:jsonb;column:class_sections_schedule" json:"class_sections_schedule,omitempty"`

	IsActive  bool       `gorm:"not null;default:true;column:class_sections_is_active" json:"class_sections_is_active"`
	CreatedAt time.Time  `gorm:"column:class_sections_created_at;autoCreateTime" json:"class_sections_created_at"`
	UpdatedAt *time.Time `gorm:"column:class_sections_updated_at;autoUpdateTime" json:"class_sections_updated_at,omitempty"`
	DeletedAt *time.Time `gorm:"column:class_sections_deleted_at" json:"class_sections_deleted_at,omitempty"`
}

func (ClassSectionModel) TableName() string {
	return "class_sections"
}
