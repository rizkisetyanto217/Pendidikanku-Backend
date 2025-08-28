package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventModel struct {
	EventID          uuid.UUID      `gorm:"column:event_id;type:uuid;default:gen_random_uuid();primaryKey" json:"event_id"`
	EventTitle       string         `gorm:"column:event_title;type:varchar(255);not null"                json:"event_title"`
	EventSlug        string         `gorm:"column:event_slug;type:varchar(100);not null"                 json:"event_slug"`
	EventDescription string         `gorm:"column:event_description;type:text"                            json:"event_description"`
	EventLocation    string         `gorm:"column:event_location;type:varchar(255)"                       json:"event_location"`
	EventMasjidID    uuid.UUID      `gorm:"column:event_masjid_id;type:uuid;not null;index:idx_events_masjid_id" json:"event_masjid_id"`

	// Timestamps (DB sudah punya trigger untuk updated_at)
	EventCreatedAt time.Time      `gorm:"column:event_created_at;type:timestamptz;autoCreateTime" json:"event_created_at"`
	EventUpdatedAt time.Time      `gorm:"column:event_updated_at;type:timestamptz;autoUpdateTime" json:"event_updated_at"`

	// Soft delete mengikuti kolom event_deleted_at
	EventDeletedAt gorm.DeletedAt `gorm:"column:event_deleted_at;type:timestamptz;index"          json:"event_deleted_at,omitempty"`

	// NOTE:
	// - Unik slug per masjid (case-insensitive) dibuat lewat migration:
	//   CREATE UNIQUE INDEX ux_events_slug_per_masjid_lower ON events (event_masjid_id, LOWER(event_slug));
	//   Tidak bisa diekspresikan langsung via tag GORM.
	// - Kolom generated tsvector event_search_tsv sengaja tidak dimodelkan.
}

func (EventModel) TableName() string {
	return "events"
}
