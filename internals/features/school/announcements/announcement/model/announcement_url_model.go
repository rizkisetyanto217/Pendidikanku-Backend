package model

import (
	"time"

	"github.com/google/uuid"
)

type AnnouncementURLModel struct {
	AnnouncementURLID               uuid.UUID  `gorm:"column:announcement_url_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"announcement_url_id"`
	AnnouncementURLMasjidID         uuid.UUID  `gorm:"column:announcement_url_masjid_id;type:uuid;not null" json:"announcement_url_masjid_id"`
	AnnouncementURLAnnouncementID   uuid.UUID  `gorm:"column:announcement_url_announcement_id;type:uuid;not null" json:"announcement_url_announcement_id"`

	AnnouncementURLLabel            *string    `gorm:"column:announcement_url_label;type:varchar(120)" json:"announcement_url_label,omitempty"`
	AnnouncementURLHref             string     `gorm:"column:announcement_url_href;type:text;not null" json:"announcement_url_href"`
	AnnouncementURLTrashURL         *string    `gorm:"column:announcement_url_trash_url;type:text" json:"announcement_url_trash_url,omitempty"`
	AnnouncementURLDeletePendingUntil *time.Time `gorm:"column:announcement_url_delete_pending_until;type:timestamptz" json:"announcement_url_delete_pending_until,omitempty"`

	AnnouncementURLCreatedAt        time.Time  `gorm:"column:announcement_url_created_at;type:timestamptz;not null;autoCreateTime" json:"announcement_url_created_at"`
	AnnouncementURLUpdatedAt        time.Time  `gorm:"column:announcement_url_updated_at;type:timestamptz;not null;autoUpdateTime" json:"announcement_url_updated_at"`
	AnnouncementURLDeletedAt        *time.Time `gorm:"column:announcement_url_deleted_at;type:timestamptz" json:"announcement_url_deleted_at,omitempty"`
}

func (AnnouncementURLModel) TableName() string { return "announcement_urls" }
