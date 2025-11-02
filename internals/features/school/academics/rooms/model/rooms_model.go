// file: internals/features/school/classrooms/model/class_room_model.go
package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ClassRoomPlatform string

const (
	PlatformZoom           ClassRoomPlatform = "zoom"
	PlatformGoogleMeet     ClassRoomPlatform = "google_meet"
	PlatformMicrosoftTeams ClassRoomPlatform = "microsoft_teams"
	PlatformOther          ClassRoomPlatform = "other"
)

type ClassRoomModel struct {
	// PK
	ClassRoomID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_room_id" json:"class_room_id"`

	// Tenant / scope
	ClassRoomSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_room_school_id" json:"class_room_school_id"`

	// Identitas ruang
	ClassRoomName        string  `gorm:"type:text;not null;column:class_room_name" json:"class_room_name"`
	ClassRoomCode        *string `gorm:"type:text;column:class_room_code" json:"class_room_code,omitempty"`
	ClassRoomSlug        *string `gorm:"type:varchar(50);column:class_room_slug" json:"class_room_slug,omitempty"`
	ClassRoomLocation    *string `gorm:"type:text;column:class_room_location" json:"class_room_location,omitempty"`
	ClassRoomCapacity    *int    `gorm:"type:int;column:class_room_capacity" json:"class_room_capacity,omitempty"`
	ClassRoomDescription *string `gorm:"type:text;column:class_room_description" json:"class_room_description,omitempty"`

	// Karakteristik
	ClassRoomIsVirtual bool `gorm:"type:boolean;not null;default:false;column:class_room_is_virtual" json:"class_room_is_virtual"`
	ClassRoomIsActive  bool `gorm:"type:boolean;not null;default:true;column:class_room_is_active" json:"class_room_is_active"`

	// Single image (2-slot + retensi)
	ClassRoomImageURL                *string    `gorm:"type:text;column:class_room_image_url" json:"class_room_image_url,omitempty"`
	ClassRoomImageObjectKey          *string    `gorm:"type:text;column:class_room_image_object_key" json:"class_room_image_object_key,omitempty"`
	ClassRoomImageURLOld             *string    `gorm:"type:text;column:class_room_image_url_old" json:"class_room_image_url_old,omitempty"`
	ClassRoomImageObjectKeyOld       *string    `gorm:"type:text;column:class_room_image_object_key_old" json:"class_room_image_object_key_old,omitempty"`
	ClassRoomImageDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:class_room_image_delete_pending_until" json:"class_room_image_delete_pending_until,omitempty"`

	// Fitur (JSONB array; default '[]')
	ClassRoomFeatures datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_room_features" json:"class_room_features"`

	// ONLINE FIELDS (sesuai SQL)
	ClassRoomPlatform  *string `gorm:"type:varchar(30);column:class_room_platform" json:"class_room_platform,omitempty"` // gunakan nilai dari konstanta di atas
	ClassRoomJoinURL   *string `gorm:"type:text;column:class_room_join_url" json:"class_room_join_url,omitempty"`
	ClassRoomMeetingID *string `gorm:"type:text;column:class_room_meeting_id" json:"class_room_meeting_id,omitempty"`
	ClassRoomPasscode  *string `gorm:"type:text;column:class_room_passcode" json:"class_room_passcode,omitempty"`

	// JADWAL & CATATAN (JSONB) — sesuai SQL
	ClassRoomSchedule datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_room_schedule" json:"class_room_schedule"`
	ClassRoomNotes    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_room_notes" json:"class_room_notes"`

	// Timestamps standar GORM
	ClassRoomCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:class_room_created_at" json:"class_room_created_at"`
	ClassRoomUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:class_room_updated_at" json:"class_room_updated_at"`
	ClassRoomDeletedAt gorm.DeletedAt `gorm:"index;column:class_room_deleted_at" json:"class_room_deleted_at,omitempty"`
}

func (ClassRoomModel) TableName() string { return "class_rooms" }

// ---------- Helpers opsional ----------
func (cr *ClassRoomModel) GetFeatures() []string {
	if len(cr.ClassRoomFeatures) == 0 {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal(cr.ClassRoomFeatures, &arr); err != nil {
		return []string{}
	}
	return arr
}
func (cr *ClassRoomModel) SetFeatures(features []string) {
	if features == nil {
		cr.ClassRoomFeatures = datatypes.JSON([]byte("[]"))
		return
	}
	b, _ := json.Marshal(features)
	cr.ClassRoomFeatures = datatypes.JSON(b)
}
