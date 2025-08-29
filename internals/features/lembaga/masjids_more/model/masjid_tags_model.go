package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidTagModel struct {
	MasjidTagID          uuid.UUID      `gorm:"column:masjid_tag_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_tag_id"`
	MasjidTagName        string         `gorm:"column:masjid_tag_name;type:varchar(50);not null" json:"masjid_tag_name"`
	MasjidTagDescription *string        `gorm:"column:masjid_tag_description;type:text" json:"masjid_tag_description,omitempty"`

	MasjidTagCreatedAt   time.Time      `gorm:"column:masjid_tag_created_at;not null;autoCreateTime" json:"masjid_tag_created_at"`
	MasjidTagUpdatedAt   time.Time      `gorm:"column:masjid_tag_updated_at;not null;autoUpdateTime"  json:"masjid_tag_updated_at"`
	MasjidTagDeletedAt   gorm.DeletedAt `gorm:"column:masjid_tag_deleted_at;index"                   json:"masjid_tag_deleted_at,omitempty"`
}

func (MasjidTagModel) TableName() string { return "masjid_tags" }
