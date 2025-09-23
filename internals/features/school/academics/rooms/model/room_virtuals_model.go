// file: internals/features/school/class_rooms/virtual_links/model/class_room_virtual_link_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===================== ENUM ===================== */

type VirtualPlatform string

const (
	VirtualPlatformZoom           VirtualPlatform = "zoom"
	VirtualPlatformGoogleMeet     VirtualPlatform = "google_meet"
	VirtualPlatformMicrosoftTeams VirtualPlatform = "microsoft_teams"
	VirtualPlatformOther          VirtualPlatform = "other"
)

/* ===================== MODEL ===================== */

type ClassRoomVirtualLinkModel struct {
	// PK
	ClassRoomVirtualLinkID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_room_virtual_link_id" json:"class_room_virtual_link_id"`

	// Scope / FK
	ClassRoomVirtualLinkMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_room_virtual_link_masjid_id" json:"class_room_virtual_link_masjid_id"`
	ClassRoomVirtualLinkRoomID   uuid.UUID `gorm:"type:uuid;not null;column:class_room_virtual_link_room_id"   json:"class_room_virtual_link_room_id"`

	// Identitas link
	ClassRoomVirtualLinkLabel     string  `gorm:"type:text;not null;column:class_room_virtual_link_label"       json:"class_room_virtual_link_label"`
	ClassRoomVirtualLinkJoinURL   string  `gorm:"type:text;not null;column:class_room_virtual_link_join_url"    json:"class_room_virtual_link_join_url"`
	ClassRoomVirtualLinkHostURL   *string `gorm:"type:text;column:class_room_virtual_link_host_url"              json:"class_room_virtual_link_host_url,omitempty"`
	ClassRoomVirtualLinkMeetingID *string `gorm:"type:text;column:class_room_virtual_link_meeting_id"            json:"class_room_virtual_link_meeting_id,omitempty"`
	ClassRoomVirtualLinkPasscode  *string `gorm:"type:text;column:class_room_virtual_link_passcode"              json:"class_room_virtual_link_passcode,omitempty"`
	ClassRoomVirtualLinkNotes     *string `gorm:"type:text;column:class_room_virtual_link_notes"                 json:"class_room_virtual_link_notes,omitempty"`

	// Platform (Postgres ENUM)
	ClassRoomVirtualLinkPlatform VirtualPlatform `gorm:"type:virtual_platform_enum;not null;column:class_room_virtual_link_platform" json:"class_room_virtual_link_platform"`

	// Status
	ClassRoomVirtualLinkIsActive bool `gorm:"type:boolean;not null;default:true;column:class_room_virtual_link_is_active" json:"class_room_virtual_link_is_active"`

	// Timestamps
	ClassRoomVirtualLinkCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_room_virtual_link_created_at" json:"class_room_virtual_link_created_at"`
	ClassRoomVirtualLinkUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_room_virtual_link_updated_at" json:"class_room_virtual_link_updated_at"`
	ClassRoomVirtualLinkDeletedAt gorm.DeletedAt `gorm:"column:class_room_virtual_link_deleted_at;index"`
}

func (ClassRoomVirtualLinkModel) TableName() string { return "class_room_virtual_links" }
