// file: internals/features/lembaga/classes/model/class_room_virtual_link_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassRoomVirtualLinkModel struct {
	// PK
	ClassRoomVirtualLinkID uuid.UUID `gorm:"column:class_room_virtual_link_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"class_room_virtual_link_id"`

	// Scope
	ClassRoomVirtualLinkMasjidID uuid.UUID `gorm:"column:class_room_virtual_link_masjid_id;type:uuid;not null" json:"class_room_virtual_link_masjid_id"`
	ClassRoomVirtualLinkRoomID   uuid.UUID `gorm:"column:class_room_virtual_link_room_id;type:uuid;not null" json:"class_room_virtual_link_room_id"`

	// Identitas link
	ClassRoomVirtualLinkLabel     string  `gorm:"column:class_room_virtual_link_label;type:text;not null" json:"class_room_virtual_link_label"`
	ClassRoomVirtualLinkJoinURL   string  `gorm:"column:class_room_virtual_link_join_url;type:text;not null" json:"class_room_virtual_link_join_url"`
	ClassRoomVirtualLinkHostURL   *string `gorm:"column:class_room_virtual_link_host_url;type:text" json:"class_room_virtual_link_host_url,omitempty"`
	ClassRoomVirtualLinkMeetingID *string `gorm:"column:class_room_virtual_link_meeting_id;type:text" json:"class_room_virtual_link_meeting_id,omitempty"`
	ClassRoomVirtualLinkPasscode  *string `gorm:"column:class_room_virtual_link_passcode;type:text" json:"class_room_virtual_link_passcode,omitempty"`
	ClassRoomVirtualLinkNotes     *string `gorm:"column:class_room_virtual_link_notes;type:text" json:"class_room_virtual_link_notes,omitempty"`

	// Status
	ClassRoomVirtualLinkIsActive bool `gorm:"column:class_room_virtual_link_is_active;type:boolean;not null;default:true" json:"class_room_virtual_link_is_active"`

	// Timestamps
	ClassRoomVirtualLinkCreatedAt time.Time      `gorm:"column:class_room_virtual_link_created_at;type:timestamptz;not null;default:now()" json:"class_room_virtual_link_created_at"`
	ClassRoomVirtualLinkUpdatedAt time.Time      `gorm:"column:class_room_virtual_link_updated_at;type:timestamptz;not null;default:now()" json:"class_room_virtual_link_updated_at"`
	ClassRoomVirtualLinkDeletedAt gorm.DeletedAt `gorm:"column:class_room_virtual_link_deleted_at;index" json:"class_room_virtual_link_deleted_at,omitempty"`
}

// TableName overrides default table name.
func (ClassRoomVirtualLinkModel) TableName() string {
	return "class_room_virtual_links"
}

// BeforeUpdate: pastikan updated_at terisi current timestamp saat update lewat GORM.
func (m *ClassRoomVirtualLinkModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.ClassRoomVirtualLinkUpdatedAt = time.Now()
	return nil
}
