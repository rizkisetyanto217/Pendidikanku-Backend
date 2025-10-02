// file: internals/features/school/classrooms/dto/class_room_dto.go
package dto

import (
	"encoding/json"
	"time"

	"masjidku_backend/internals/features/school/academics/rooms/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

//
// ====== ALIASES & TIPE BANTU ======
//

type ClassRoomPlatform = model.ClassRoomPlatform

const (
	PlatformZoom           ClassRoomPlatform = model.PlatformZoom
	PlatformGoogleMeet     ClassRoomPlatform = model.PlatformGoogleMeet
	PlatformMicrosoftTeams ClassRoomPlatform = model.PlatformMicrosoftTeams
	PlatformOther          ClassRoomPlatform = model.PlatformOther
)

// Bentuk longgar utk schedule/notes (boleh apa saja berbentuk object)
// Contoh schedule item: {"weekday":"MON","start":"07:30","end":"09:00","timezone":"Asia/Jakarta"}
type AnyObject = map[string]any

//
// ========== CREATE ==========
//

type CreateClassRoomRequest struct {
	// Tenant
	ClassRoomMasjidID uuid.UUID `json:"class_room_masjid_id" validate:"required"`

	// Identitas
	ClassRoomName        string  `json:"class_room_name" validate:"required"`
	ClassRoomCode        *string `json:"class_room_code" validate:"omitempty,max=120"`
	ClassRoomSlug        *string `json:"class_room_slug" validate:"omitempty,max=50"`
	ClassRoomLocation    *string `json:"class_room_location" validate:"omitempty,max=500"`
	ClassRoomCapacity    *int    `json:"class_room_capacity" validate:"omitempty,min=0"`
	ClassRoomDescription *string `json:"class_room_description" validate:"omitempty"`

	// Karakteristik
	ClassRoomIsVirtual *bool `json:"class_room_is_virtual" validate:"omitempty"`
	ClassRoomIsActive  *bool `json:"class_room_is_active" validate:"omitempty"`

	// Image (opsional)
	ClassRoomImageURL                *string    `json:"class_room_image_url" validate:"omitempty,url"`
	ClassRoomImageObjectKey          *string    `json:"class_room_image_object_key" validate:"omitempty"`
	ClassRoomImageURLOld             *string    `json:"class_room_image_url_old" validate:"omitempty,url"`
	ClassRoomImageObjectKeyOld       *string    `json:"class_room_image_object_key_old" validate:"omitempty"`
	ClassRoomImageDeletePendingUntil *time.Time `json:"class_room_image_delete_pending_until" validate:"omitempty"`

	// ONLINE FIELDS
	ClassRoomPlatform  *string `json:"class_room_platform" validate:"omitempty,oneof=zoom google_meet microsoft_teams other"`
	ClassRoomJoinURL   *string `json:"class_room_join_url" validate:"omitempty,url"`
	ClassRoomMeetingID *string `json:"class_room_meeting_id" validate:"omitempty"`
	ClassRoomPasscode  *string `json:"class_room_passcode" validate:"omitempty"`

	// JSONB ergonomis di DTO → slice native
	ClassRoomFeatures []string    `json:"class_room_features" validate:"omitempty,dive,printascii"`
	ClassRoomSchedule []AnyObject `json:"class_room_schedule" validate:"omitempty,dive"`
	ClassRoomNotes    []AnyObject `json:"class_room_notes" validate:"omitempty,dive"`
}

func (r CreateClassRoomRequest) ToModel() (model.ClassRoomModel, error) {
	m := model.ClassRoomModel{
		ClassRoomMasjidID:  r.ClassRoomMasjidID,
		ClassRoomName:      r.ClassRoomName,
		ClassRoomIsVirtual: false,
		ClassRoomIsActive:  true,
	}

	// Identitas
	m.ClassRoomCode = r.ClassRoomCode
	m.ClassRoomSlug = r.ClassRoomSlug
	m.ClassRoomLocation = r.ClassRoomLocation
	m.ClassRoomCapacity = r.ClassRoomCapacity
	m.ClassRoomDescription = r.ClassRoomDescription

	// Flags
	if r.ClassRoomIsVirtual != nil {
		m.ClassRoomIsVirtual = *r.ClassRoomIsVirtual
	}
	if r.ClassRoomIsActive != nil {
		m.ClassRoomIsActive = *r.ClassRoomIsActive
	}

	// Image
	m.ClassRoomImageURL = r.ClassRoomImageURL
	m.ClassRoomImageObjectKey = r.ClassRoomImageObjectKey
	m.ClassRoomImageURLOld = r.ClassRoomImageURLOld
	m.ClassRoomImageObjectKeyOld = r.ClassRoomImageObjectKeyOld
	m.ClassRoomImageDeletePendingUntil = r.ClassRoomImageDeletePendingUntil

	// Online fields
	m.ClassRoomPlatform = r.ClassRoomPlatform
	m.ClassRoomJoinURL = r.ClassRoomJoinURL
	m.ClassRoomMeetingID = r.ClassRoomMeetingID
	m.ClassRoomPasscode = r.ClassRoomPasscode

	// JSONB
	if err := setJSONFromStrings(&m.ClassRoomFeatures, r.ClassRoomFeatures); err != nil {
		return m, err
	}
	if err := setJSONFromAnySlice(&m.ClassRoomSchedule, r.ClassRoomSchedule); err != nil {
		return m, err
	}
	if err := setJSONFromAnySlice(&m.ClassRoomNotes, r.ClassRoomNotes); err != nil {
		return m, err
	}

	return m, nil
}

//
// ========== UPDATE / PATCH ==========
//

type UpdateClassRoomRequest struct {
	// Identitas
	ClassRoomName        *string `json:"class_room_name" validate:"omitempty"`
	ClassRoomCode        *string `json:"class_room_code" validate:"omitempty,max=120"`
	ClassRoomSlug        *string `json:"class_room_slug" validate:"omitempty,max=50"`
	ClassRoomLocation    *string `json:"class_room_location" validate:"omitempty,max=500"`
	ClassRoomCapacity    *int    `json:"class_room_capacity" validate:"omitempty,min=0"`
	ClassRoomDescription *string `json:"class_room_description" validate:"omitempty"`

	// Karakteristik
	ClassRoomIsVirtual *bool `json:"class_room_is_virtual" validate:"omitempty"`
	ClassRoomIsActive  *bool `json:"class_room_is_active" validate:"omitempty"`

	// Image (opsional)
	ClassRoomImageURL                *string    `json:"class_room_image_url" validate:"omitempty,url"`
	ClassRoomImageObjectKey          *string    `json:"class_room_image_object_key" validate:"omitempty"`
	ClassRoomImageURLOld             *string    `json:"class_room_image_url_old" validate:"omitempty,url"`
	ClassRoomImageObjectKeyOld       *string    `json:"class_room_image_object_key_old" validate:"omitempty"`
	ClassRoomImageDeletePendingUntil *time.Time `json:"class_room_image_delete_pending_until" validate:"omitempty"`

	// ONLINE FIELDS
	ClassRoomPlatform  *string `json:"class_room_platform" validate:"omitempty,oneof=zoom google_meet microsoft_teams other"`
	ClassRoomJoinURL   *string `json:"class_room_join_url" validate:"omitempty,url"`
	ClassRoomMeetingID *string `json:"class_room_meeting_id" validate:"omitempty"`
	ClassRoomPasscode  *string `json:"class_room_passcode" validate:"omitempty"`

	// JSONB ergonomis → pointer slice (nil=skip)
	ClassRoomFeatures *[]string    `json:"class_room_features" validate:"omitempty,dive,printascii"`
	ClassRoomSchedule *[]AnyObject `json:"class_room_schedule" validate:"omitempty,dive"`
	ClassRoomNotes    *[]AnyObject `json:"class_room_notes" validate:"omitempty,dive"`

	// Clear (set ke kosong/NULL sesuai kolom)
	Clear []string `json:"__clear,omitempty" validate:"omitempty,dive,oneof=class_room_code class_room_slug class_room_location class_room_capacity class_room_description class_room_image_url class_room_image_object_key class_room_image_url_old class_room_image_object_key_old class_room_image_delete_pending_until class_room_platform class_room_join_url class_room_meeting_id class_room_passcode class_room_features class_room_schedule class_room_notes"`
}

// Mutasi in-place ke model
func (r UpdateClassRoomRequest) ApplyPatch(m *model.ClassRoomModel) error {
	// Identitas
	if r.ClassRoomName != nil {
		m.ClassRoomName = *r.ClassRoomName
	}
	if r.ClassRoomCode != nil {
		m.ClassRoomCode = r.ClassRoomCode
	}
	if r.ClassRoomSlug != nil {
		m.ClassRoomSlug = r.ClassRoomSlug
	}
	if r.ClassRoomLocation != nil {
		m.ClassRoomLocation = r.ClassRoomLocation
	}
	if r.ClassRoomCapacity != nil {
		m.ClassRoomCapacity = r.ClassRoomCapacity
	}
	if r.ClassRoomDescription != nil {
		m.ClassRoomDescription = r.ClassRoomDescription
	}

	// Flags
	if r.ClassRoomIsVirtual != nil {
		m.ClassRoomIsVirtual = *r.ClassRoomIsVirtual
	}
	if r.ClassRoomIsActive != nil {
		m.ClassRoomIsActive = *r.ClassRoomIsActive
	}

	// Image
	if r.ClassRoomImageURL != nil {
		m.ClassRoomImageURL = r.ClassRoomImageURL
	}
	if r.ClassRoomImageObjectKey != nil {
		m.ClassRoomImageObjectKey = r.ClassRoomImageObjectKey
	}
	if r.ClassRoomImageURLOld != nil {
		m.ClassRoomImageURLOld = r.ClassRoomImageURLOld
	}
	if r.ClassRoomImageObjectKeyOld != nil {
		m.ClassRoomImageObjectKeyOld = r.ClassRoomImageObjectKeyOld
	}
	if r.ClassRoomImageDeletePendingUntil != nil {
		m.ClassRoomImageDeletePendingUntil = r.ClassRoomImageDeletePendingUntil
	}

	// Online fields
	if r.ClassRoomPlatform != nil {
		m.ClassRoomPlatform = r.ClassRoomPlatform
	}
	if r.ClassRoomJoinURL != nil {
		m.ClassRoomJoinURL = r.ClassRoomJoinURL
	}
	if r.ClassRoomMeetingID != nil {
		m.ClassRoomMeetingID = r.ClassRoomMeetingID
	}
	if r.ClassRoomPasscode != nil {
		m.ClassRoomPasscode = r.ClassRoomPasscode
	}

	// JSONB
	if r.ClassRoomFeatures != nil {
		if err := setJSONFromStrings(&m.ClassRoomFeatures, *r.ClassRoomFeatures); err != nil {
			return err
		}
	}
	if r.ClassRoomSchedule != nil {
		if err := setJSONFromAnySlice(&m.ClassRoomSchedule, *r.ClassRoomSchedule); err != nil {
			return err
		}
	}
	if r.ClassRoomNotes != nil {
		if err := setJSONFromAnySlice(&m.ClassRoomNotes, *r.ClassRoomNotes); err != nil {
			return err
		}
	}

	// Clear
	for _, col := range r.Clear {
		switch col {
		case "class_room_code":
			m.ClassRoomCode = nil
		case "class_room_slug":
			m.ClassRoomSlug = nil
		case "class_room_location":
			m.ClassRoomLocation = nil
		case "class_room_capacity":
			m.ClassRoomCapacity = nil
		case "class_room_description":
			m.ClassRoomDescription = nil
		case "class_room_image_url":
			m.ClassRoomImageURL = nil
		case "class_room_image_object_key":
			m.ClassRoomImageObjectKey = nil
		case "class_room_image_url_old":
			m.ClassRoomImageURLOld = nil // jika field berbeda nama, sesuaikan; di model: ClassRoomImageURLOld
		case "class_room_image_object_key_old":
			m.ClassRoomImageObjectKeyOld = nil
		case "class_room_image_delete_pending_until":
			m.ClassRoomImageDeletePendingUntil = nil
		case "class_room_platform":
			m.ClassRoomPlatform = nil
		case "class_room_join_url":
			m.ClassRoomJoinURL = nil
		case "class_room_meeting_id":
			m.ClassRoomMeetingID = nil
		case "class_room_passcode":
			m.ClassRoomPasscode = nil
		case "class_room_features":
			m.ClassRoomFeatures = datatypes.JSON([]byte("[]"))
		case "class_room_schedule":
			m.ClassRoomSchedule = datatypes.JSON([]byte("[]"))
		case "class_room_notes":
			m.ClassRoomNotes = datatypes.JSON([]byte("[]"))
		}
	}

	return nil
}

//
// ========== RESPONSE ==========
//

type ClassRoomResponse struct {
	// Inti
	ClassRoomID       uuid.UUID `json:"class_room_id"`
	ClassRoomMasjidID uuid.UUID `json:"class_room_masjid_id"`

	// Identitas
	ClassRoomName        string  `json:"class_room_name"`
	ClassRoomCode        *string `json:"class_room_code,omitempty"`
	ClassRoomSlug        *string `json:"class_room_slug,omitempty"`
	ClassRoomLocation    *string `json:"class_room_location,omitempty"`
	ClassRoomCapacity    *int    `json:"class_room_capacity,omitempty"`
	ClassRoomDescription *string `json:"class_room_description,omitempty"`

	// Karakteristik
	ClassRoomIsVirtual bool `json:"class_room_is_virtual"`
	ClassRoomIsActive  bool `json:"class_room_is_active"`

	// Image
	ClassRoomImageURL                *string    `json:"class_room_image_url,omitempty"`
	ClassRoomImageObjectKey          *string    `json:"class_room_image_object_key,omitempty"`
	ClassRoomImageURLOld             *string    `json:"class_room_image_url_old,omitempty"`
	ClassRoomImageObjectKeyOld       *string    `json:"class_room_image_object_key_old,omitempty"`
	ClassRoomImageDeletePendingUntil *time.Time `json:"class_room_image_delete_pending_until,omitempty"`

	// Online
	ClassRoomPlatform  *string `json:"class_room_platform,omitempty"`
	ClassRoomJoinURL   *string `json:"class_room_join_url,omitempty"`
	ClassRoomMeetingID *string `json:"class_room_meeting_id,omitempty"`
	ClassRoomPasscode  *string `json:"class_room_passcode,omitempty"`

	// JSONB, ergonomis
	ClassRoomFeatures []string    `json:"class_room_features"`
	ClassRoomSchedule []AnyObject `json:"class_room_schedule"`
	ClassRoomNotes    []AnyObject `json:"class_room_notes"`

	// Audit
	ClassRoomCreatedAt string `json:"class_room_created_at"`
	ClassRoomUpdatedAt string `json:"class_room_updated_at"`
}

func ToClassRoomResponse(m model.ClassRoomModel) ClassRoomResponse {
	return ClassRoomResponse{
		ClassRoomID:                      m.ClassRoomID,
		ClassRoomMasjidID:                m.ClassRoomMasjidID,
		ClassRoomName:                    m.ClassRoomName,
		ClassRoomCode:                    m.ClassRoomCode,
		ClassRoomSlug:                    m.ClassRoomSlug,
		ClassRoomLocation:                m.ClassRoomLocation,
		ClassRoomCapacity:                m.ClassRoomCapacity,
		ClassRoomDescription:             m.ClassRoomDescription,
		ClassRoomIsVirtual:               m.ClassRoomIsVirtual,
		ClassRoomIsActive:                m.ClassRoomIsActive,
		ClassRoomImageURL:                m.ClassRoomImageURL,
		ClassRoomImageObjectKey:          m.ClassRoomImageObjectKey,
		ClassRoomImageURLOld:             m.ClassRoomImageURLOld,
		ClassRoomImageObjectKeyOld:       m.ClassRoomImageObjectKeyOld,
		ClassRoomImageDeletePendingUntil: m.ClassRoomImageDeletePendingUntil,
		ClassRoomPlatform:                m.ClassRoomPlatform,
		ClassRoomJoinURL:                 m.ClassRoomJoinURL,
		ClassRoomMeetingID:               m.ClassRoomMeetingID,
		ClassRoomPasscode:                m.ClassRoomPasscode,
		ClassRoomFeatures:                mustStringsFromJSON(m.ClassRoomFeatures),
		ClassRoomSchedule:                mustObjectsFromJSON(m.ClassRoomSchedule),
		ClassRoomNotes:                   mustObjectsFromJSON(m.ClassRoomNotes),
		ClassRoomCreatedAt:               m.ClassRoomCreatedAt.Format(time.RFC3339),
		ClassRoomUpdatedAt:               m.ClassRoomUpdatedAt.Format(time.RFC3339),
	}
}

//
// ========== helpers ==========
//

func setJSONFromStrings(dst *datatypes.JSON, arr []string) error {
	if len(arr) == 0 {
		*dst = datatypes.JSON([]byte("[]"))
		return nil
	}
	b, err := json.Marshal(arr)
	if err != nil {
		return err
	}
	*dst = datatypes.JSON(b)
	return nil
}

func setJSONFromAnySlice(dst *datatypes.JSON, arr []AnyObject) error {
	if len(arr) == 0 {
		*dst = datatypes.JSON([]byte("[]"))
		return nil
	}
	b, err := json.Marshal(arr)
	if err != nil {
		return err
	}
	*dst = datatypes.JSON(b)
	return nil
}

func mustStringsFromJSON(j datatypes.JSON) []string {
	if len(j) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(j, &out); err != nil {
		return []string{}
	}
	return out
}

func mustObjectsFromJSON(j datatypes.JSON) []AnyObject {
	if len(j) == 0 {
		return []AnyObject{}
	}
	var out []AnyObject
	if err := json.Unmarshal(j, &out); err != nil {
		return []AnyObject{}
	}
	return out
}
