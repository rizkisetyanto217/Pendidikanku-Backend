// file: internals/features/school/classrooms/model/class_room_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ClassRoomModel struct {
	// PK
	ClassRoomID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_room_id" json:"class_room_id"`

	// Tenant / scope
	ClassRoomMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_room_masjid_id" json:"class_room_masjid_id"`

	// Identitas ruang
	ClassRoomName        string  `gorm:"type:text;not null;column:class_room_name" json:"class_room_name"`
	ClassRoomCode        *string `gorm:"type:text;column:class_room_code" json:"class_room_code,omitempty"`
	ClassRoomSlug        *string `gorm:"type:varchar(50);column:class_room_slug" json:"class_room_slug,omitempty"`
	ClassRoomLocation    *string `gorm:"type:text;column:class_room_location" json:"class_room_location,omitempty"`
	ClassRoomCapacity    *int    `gorm:"column:class_room_capacity" json:"class_room_capacity,omitempty"`
	ClassRoomDescription *string `gorm:"type:text;column:class_room_description" json:"class_room_description,omitempty"`

	// Karakteristik
	ClassRoomIsVirtual bool `gorm:"not null;default:false;column:class_room_is_virtual" json:"class_room_is_virtual"`
	ClassRoomIsActive  bool `gorm:"not null;default:true;column:class_room_is_active" json:"class_room_is_active"`

	// Fitur (JSONB array)
	ClassRoomFeatures datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_room_features" json:"class_room_features"`

	// Timestamps (dikelola aplikasi)
	ClassRoomCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_room_created_at" json:"class_room_created_at"`
	ClassRoomUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_room_updated_at" json:"class_room_updated_at"`
	ClassRoomDeletedAt gorm.DeletedAt `gorm:"column:class_room_deleted_at;index" json:"class_room_deleted_at,omitempty"`
}

func (ClassRoomModel) TableName() string { return "class_rooms" }
