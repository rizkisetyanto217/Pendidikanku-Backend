package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchoolTagModel struct {
	SchoolTagID          uuid.UUID `gorm:"column:school_tag_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"school_tag_id"`
	SchoolTagName        string    `gorm:"column:school_tag_name;type:varchar(50);not null" json:"school_tag_name"`
	SchoolTagDescription *string   `gorm:"column:school_tag_description;type:text" json:"school_tag_description,omitempty"`

	SchoolTagCreatedAt time.Time      `gorm:"column:school_tag_created_at;not null;autoCreateTime" json:"school_tag_created_at"`
	SchoolTagUpdatedAt time.Time      `gorm:"column:school_tag_updated_at;not null;autoUpdateTime"  json:"school_tag_updated_at"`
	SchoolTagDeletedAt gorm.DeletedAt `gorm:"column:school_tag_deleted_at;index"                   json:"school_tag_deleted_at,omitempty"`
}

func (SchoolTagModel) TableName() string { return "school_tags" }
