package model

import (
	"time"

	"github.com/google/uuid"
)

type EventModel struct {
	EventID          uuid.UUID `gorm:"column:event_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"event_id"`
	EventTitle       string    `gorm:"column:event_title;type:varchar(255);not null" json:"event_title"`
	EventSlug        string    `gorm:"column:event_slug;type:varchar(100);not null" json:"event_slug"`
	EventDescription string    `gorm:"column:event_description;type:text" json:"event_description"`
	EventLocation    string    `gorm:"column:event_location;type:varchar(255)" json:"event_location"`
	EventMasjidID    uuid.UUID `gorm:"column:event_masjid_id;type:uuid;not null" json:"event_masjid_id"`
	EventCreatedAt   time.Time `gorm:"column:event_created_at;autoCreateTime" json:"event_created_at"`
}

func (EventModel) TableName() string {
	return "events"
}
