package model

import (
	"time"

	"github.com/google/uuid"
)

type EventSessionModel struct {
	EventSessionID                   uuid.UUID `gorm:"column:event_session_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"event_session_id"`
	EventSessionEventID              uuid.UUID `gorm:"column:event_session_event_id;type:uuid;not null" json:"event_session_event_id"`
	EventSessionTitle                string    `gorm:"column:event_session_title;type:varchar(255);not null" json:"event_session_title"`
	EventSessionDescription          string    `gorm:"column:event_session_description;type:text" json:"event_session_description"`
	EventSessionStartTime            time.Time `gorm:"column:event_session_start_time;not null" json:"event_session_start_time"`
	EventSessionEndTime              time.Time `gorm:"column:event_session_end_time;not null" json:"event_session_end_time"`
	EventSessionLocation             string    `gorm:"column:event_session_location;type:varchar(255)" json:"event_session_location"`
	EventSessionImageURL             string    `gorm:"column:event_session_image_url;type:text" json:"event_session_image_url"`
	EventSessionCapacity             int       `gorm:"column:event_session_capacity" json:"event_session_capacity"`
	EventSessionIsPublic             bool      `gorm:"column:event_session_is_public;default:true" json:"event_session_is_public"`
	EventSessionIsRegistrationNeeded bool      `gorm:"column:event_session_is_registration_required;default:false" json:"event_session_is_registration_required"`
	EventSessionMasjidID             uuid.UUID `gorm:"column:event_session_masjid_id;type:uuid;not null" json:"event_session_masjid_id"` // <<--- TAMBAHAN INI
	EventSessionCreatedBy *uuid.UUID `gorm:"column:event_session_created_by;type:uuid" json:"event_session_created_by"` // boleh null
	EventSessionCreatedAt time.Time  `gorm:"column:event_session_created_at;autoCreateTime" json:"event_session_created_at"`
	EventSessionUpdatedAt time.Time  `gorm:"column:event_session_updated_at;autoUpdateTime" json:"event_session_updated_at"`
}

func (EventSessionModel) TableName() string {
	return "event_sessions"
}
