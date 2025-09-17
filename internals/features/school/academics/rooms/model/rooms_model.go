// file: internals/features/school/class_rooms/model/class_room_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ClassRoomModel merepresentasikan tabel class_rooms
// ClassRoomModel merepresentasikan tabel class_rooms
type ClassRoomModel struct {
    ClassRoomID        uuid.UUID        `json:"class_room_id" gorm:"type:uuid;primaryKey;column:class_room_id;default:gen_random_uuid()"`
    ClassRoomsMasjidID uuid.UUID        `json:"class_rooms_masjid_id" gorm:"type:uuid;not null;column:class_rooms_masjid_id"`

    ClassRoomsName        string   `json:"class_rooms_name" gorm:"type:text;not null;column:class_rooms_name"`
    ClassRoomsCode        *string  `json:"class_rooms_code,omitempty" gorm:"type:text;column:class_rooms_code"`
    ClassRoomsSlug        *string  `json:"class_rooms_slug,omitempty" gorm:"type:varchar(50);column:class_rooms_slug"` // ← DITAMBAH
    ClassRoomsLocation    *string  `json:"class_rooms_location,omitempty" gorm:"type:text;column:class_rooms_location"`
    // ClassRoomsFloor     *int     `json:"class_rooms_floor,omitempty" gorm:"column:class_rooms_floor"` // ← HAPUS jika tak dipakai
    ClassRoomsCapacity    *int     `json:"class_rooms_capacity,omitempty" gorm:"column:class_rooms_capacity"`
    ClassRoomsDescription *string  `json:"class_rooms_description,omitempty" gorm:"type:text;column:class_rooms_description"`

    ClassRoomsIsVirtual bool            `json:"class_rooms_is_virtual" gorm:"not null;default:false;column:class_rooms_is_virtual"`
    ClassRoomsIsActive  bool            `json:"class_rooms_is_active"  gorm:"not null;default:true;column:class_rooms_is_active"`

    ClassRoomsFeatures datatypes.JSON   `json:"class_rooms_features" gorm:"type:jsonb;not null;default:'[]';column:class_rooms_features"`

    ClassRoomsCreatedAt time.Time      `json:"class_rooms_created_at" gorm:"column:class_rooms_created_at;autoCreateTime"`
    ClassRoomsUpdatedAt time.Time      `json:"class_rooms_updated_at" gorm:"column:class_rooms_updated_at;autoUpdateTime"`
    ClassRoomsDeletedAt gorm.DeletedAt `json:"class_rooms_deleted_at,omitempty" gorm:"column:class_rooms_deleted_at;index"`
}


// TableName mengikat model ke tabel class_rooms
func (ClassRoomModel) TableName() string { return "class_rooms" }
