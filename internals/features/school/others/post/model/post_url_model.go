// file: internals/features/announcements/urls/model/announcement_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type AnnouncementURLModel struct {
	AnnouncementURLId             uuid.UUID `gorm:"column:announcement_url_id;type:uuid;default:gen_random_uuid();primaryKey"`
	AnnouncementURLMasjidId       uuid.UUID `gorm:"column:announcement_url_masjid_id;type:uuid;not null"`
	AnnouncementURLAnnouncementId uuid.UUID `gorm:"column:announcement_url_announcement_id;type:uuid;not null"`

	AnnouncementURLKind string `gorm:"column:announcement_url_kind;type:varchar(24);not null"`

	AnnouncementURLHref         *string `gorm:"column:announcement_url_href;type:text"`
	AnnouncementURLObjectKey    *string `gorm:"column:announcement_url_object_key;type:text"`
	AnnouncementURLObjectKeyOld *string `gorm:"column:announcement_url_object_key_old;type:text"`

	AnnouncementURLLabel     *string `gorm:"column:announcement_url_label;type:varchar(160)"`
	AnnouncementURLOrder     int     `gorm:"column:announcement_url_order;type:int;not null;default:0"`
	AnnouncementURLIsPrimary bool    `gorm:"column:announcement_url_is_primary;type:boolean;not null;default:false"`

	AnnouncementURLCreatedAt          time.Time  `gorm:"column:announcement_url_created_at;type:timestamptz;not null;default:now()"`
	AnnouncementURLUpdatedAt          time.Time  `gorm:"column:announcement_url_updated_at;type:timestamptz;not null;default:now()"`
	AnnouncementURLDeletedAt          *time.Time `gorm:"column:announcement_url_deleted_at;type:timestamptz"`
	AnnouncementURLDeletePendingUntil *time.Time `gorm:"column:announcement_url_delete_pending_until;type:timestamptz"`
}

func (AnnouncementURLModel) TableName() string {
	return "announcement_urls"
}
