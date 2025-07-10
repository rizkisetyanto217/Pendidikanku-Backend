package model

import (
	"time"

	"github.com/google/uuid"
)

type MasjidTagModel struct {
	MasjidTagID          uuid.UUID `gorm:"column:masjid_tag_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"masjid_tag_id"`
	MasjidTagName        string    `gorm:"column:masjid_tag_name;type:varchar(50);not null" json:"masjid_tag_name"`
	MasjidTagDescription string    `gorm:"column:masjid_tag_description;type:text" json:"masjid_tag_description"`
	MasjidTagCreatedAt   time.Time `gorm:"column:masjid_tag_created_at;autoCreateTime" json:"masjid_tag_created_at"`
}

func (MasjidTagModel) TableName() string {
	return "masjid_tags"
}
