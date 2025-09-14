package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassParentModel struct {
	// PK & tenant
	ClassParentID       uuid.UUID      `gorm:"column:class_parent_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"class_parent_id"`
	ClassParentMasjidID uuid.UUID      `gorm:"column:class_parent_masjid_id;type:uuid;not null;index" json:"class_parent_masjid_id"`

	// Identitas
	ClassParentName string  `gorm:"column:class_parent_name;type:varchar(120);not null" json:"class_parent_name"`
	ClassParentCode string  `gorm:"column:class_parent_code;type:varchar(40)" json:"class_parent_code"`

	// Detail
	ClassParentDescription string  `gorm:"column:class_parent_description;type:text" json:"class_parent_description"`
	ClassParentLevel       *int16  `gorm:"column:class_parent_level;type:smallint" json:"class_parent_level"`

	// Status
	ClassParentIsActive bool `gorm:"column:class_parent_is_active;not null;default:true" json:"class_parent_is_active"`

	// Audit
	ClassParentCreatedAt time.Time      `gorm:"column:class_parent_created_at;not null;autoCreateTime" json:"class_parent_created_at"`
	ClassParentUpdatedAt time.Time      `gorm:"column:class_parent_updated_at;not null;autoUpdateTime" json:"class_parent_updated_at"`
	ClassParentDeletedAt gorm.DeletedAt `gorm:"column:class_parent_deleted_at;index" json:"class_parent_deleted_at,omitempty"`
}

func (ClassParentModel) TableName() string {
	return "class_parents"
}
