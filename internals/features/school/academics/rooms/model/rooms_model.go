// file: internals/features/school/classrooms/model/class_room_model.go
package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================================================
   ENUM-like: Platform untuk virtual links
========================================================= */

type ClassRoomPlatform string

const (
	PlatformZoom           ClassRoomPlatform = "zoom"
	PlatformGoogleMeet     ClassRoomPlatform = "google_meet"
	PlatformMicrosoftTeams ClassRoomPlatform = "microsoft_teams"
	PlatformOther          ClassRoomPlatform = "other"
)

/* =========================================================
   Typed shapes untuk JSONB (opsional)
   - Sesuai validasi jsonpath di DDL
========================================================= */

type ClassRoomTimeWindow struct {
	From string `json:"from"` // string; format bebas di app
	To   string `json:"to"`
}

type ClassRoomScheduleItem struct {
	Weekday  string `json:"weekday"`  // "MON".."SUN"
	Start    string `json:"start"`    // "HH:MM" (disarankan)
	End      string `json:"end"`      // "HH:MM"
	Timezone string `json:"timezone"` // IANA tz, mis. "Asia/Jakarta"
}

type ClassRoomVirtualLink struct {
	Label      *string                 `json:"label,omitempty"`
	Platform   ClassRoomPlatform       `json:"platform"` // wajib & valid
	JoinURL    string                  `json:"join_url"` // wajib
	HostURL    *string                 `json:"host_url,omitempty"`
	MeetingID  *string                 `json:"meeting_id,omitempty"`
	Passcode   *string                 `json:"passcode,omitempty"`
	Notes      *string                 `json:"notes,omitempty"`
	IsActive   *bool                   `json:"is_active,omitempty"`
	Tags       []string                `json:"tags,omitempty"`
	TimeWindow *ClassRoomTimeWindow    `json:"time_window,omitempty"`
	Schedule   []ClassRoomScheduleItem `json:"schedule,omitempty"`
}

/* =========================================================
   (Opsional) JSONB Typed Wrapper agar mudah pakai GORM
   - Bisa langsung dipakai menggantikan datatypes.JSON
   - Mengimplementasikan sql.Scanner & driver.Valuer
========================================================= */

type JSONBVirtualLinks []ClassRoomVirtualLink

func (v *JSONBVirtualLinks) Scan(value any) error {
	if value == nil {
		*v = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("JSONBVirtualLinks: type assertion to []byte failed")
	}
	var tmp []ClassRoomVirtualLink
	if len(b) == 0 {
		*v = nil
		return nil
	}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*v = JSONBVirtualLinks(tmp)
	return nil
}

func (v JSONBVirtualLinks) Value() (driver.Value, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return json.Marshal(v)
}

/* =========================================================
   Model: ClassRoom
   - Index & unique partial dibuat via migration (DDL).
========================================================= */

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
	// Pilih salah satu: datatypes.JSON (raw) ATAU typed wrapper JSONBVirtualLinks untuk virtual_links.
	ClassRoomFeatures datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_room_features" json:"class_room_features"`

	// Virtual meeting links (JSONB array)
	// Opsi A (raw):
	// ClassRoomVirtualLinks datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_room_virtual_links" json:"class_room_virtual_links"`
	// Opsi B (typed wrapper):
	ClassRoomVirtualLinks JSONBVirtualLinks `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_room_virtual_links" json:"class_room_virtual_links"`

	// Timestamps standar GORM
	ClassRoomCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:class_room_created_at" json:"class_room_created_at"`
	ClassRoomUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:class_room_updated_at" json:"class_room_updated_at"`
	ClassRoomDeletedAt gorm.DeletedAt `gorm:"index;column:class_room_deleted_at" json:"class_room_deleted_at,omitempty"`
}

// TableName menegaskan nama tabel eksplisit
func (ClassRoomModel) TableName() string { return "class_rooms" }

/* =========================================================
   Helpers: Features (JSONB → []string) hanya jika pakai raw datatypes.JSON
   (Kalau ingin typed penuh, ubah ClassRoomFeatures ke []string + Scanner/Valuer)
========================================================= */

// GetFeatures mengembalikan []string dari JSONB features.
// Jika JSON invalid/empty, mengembalikan slice kosong.
func (cr *ClassRoomModel) GetFeatures() []string {
	// cukup cek len(...) == 0
	if len(cr.ClassRoomFeatures) == 0 {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal(cr.ClassRoomFeatures, &arr); err != nil {
		return []string{}
	}
	return arr
}

// SetFeatures menyetel features dari []string ke JSONB.
func (cr *ClassRoomModel) SetFeatures(features []string) error {
	if features == nil {
		cr.ClassRoomFeatures = datatypes.JSON([]byte("[]"))
		return nil
	}
	b, err := json.Marshal(features)
	if err != nil {
		return err
	}
	cr.ClassRoomFeatures = datatypes.JSON(b)
	return nil
}

/* =========================================================
   Helpers: VirtualLinks (hanya jika pakai raw datatypes.JSON)
   (Di sini kita sudah pakai JSONBVirtualLinks typed; jadi helper
    raw tidak dibutuhkan. Jika suatu saat berpindah ke raw JSON:
    gunakan helper di bawah ini.)
========================================================= */

// // GetVirtualLinksRaw contoh jika field bertipe datatypes.JSON:
// func (cr *ClassRoom) GetVirtualLinksRaw() []ClassRoomVirtualLink {
// 	if cr.ClassRoomVirtualLinksRaw == nil || len(cr.ClassRoomVirtualLinksRaw) == 0 {
// 		return []ClassRoomVirtualLink{}
// 	}
// 	var arr []ClassRoomV
