// file: internals/features/school/class_rooms/dto/class_room_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	classroomModel "masjidku_backend/internals/features/school/academics/rooms/model" // ← sesuaikan path model

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =======================================================
   OPTIONAL + NULLABLE HELPERS (untuk PATCH tri-state)
   ======================================================= */

type Optional[T any] struct {
	Present bool
	Value   T
}

func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Present = true
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Value = v
	return nil
}

type NullableString struct {
	Valid bool
	Value string
}

func (ns *NullableString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		ns.Valid = false
		ns.Value = ""
		return nil
	}
	ns.Valid = true
	return json.Unmarshal(b, &ns.Value)
}

type NullableInt struct {
	Valid bool
	Value int
}

func (ni *NullableInt) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		ni.Valid = false
		ni.Value = 0
		return nil
	}
	ni.Valid = true
	return json.Unmarshal(b, &ni.Value)
}

/* =======================================================
   REQUEST DTOs (CREATE / UPDATE)
   ======================================================= */

type CreateClassRoomRequest struct {
	ClassRoomName        string         `json:"class_room_name" validate:"required,min=3,max=100"`
	ClassRoomCode        *string        `json:"class_room_code,omitempty" validate:"omitempty,max=50"`
	ClassRoomSlug        *string        `json:"class_room_slug,omitempty" validate:"omitempty,min=3,max=50"`
	ClassRoomLocation    *string        `json:"class_room_location,omitempty" validate:"omitempty,max=100"`
	ClassRoomCapacity    *int           `json:"class_room_capacity,omitempty" validate:"omitempty,min=0"`
	ClassRoomDescription *string        `json:"class_room_description,omitempty" validate:"omitempty"`
	ClassRoomIsVirtual   bool           `json:"class_room_is_virtual"`
	ClassRoomIsActive    bool           `json:"class_room_is_active"`
	ClassRoomFeatures    datatypes.JSON `json:"class_room_features" validate:"omitempty"`
}

type UpdateClassRoomRequest struct {
	ClassRoomName        *string         `json:"class_room_name,omitempty" validate:"omitempty,min=3,max=100"`
	ClassRoomCode        *string         `json:"class_room_code,omitempty" validate:"omitempty,max=50"`
	ClassRoomSlug        *string         `json:"class_room_slug,omitempty" validate:"omitempty,min=3,max=50"`
	ClassRoomLocation    *string         `json:"class_room_location,omitempty" validate:"omitempty,max=100"`
	ClassRoomCapacity    *int            `json:"class_room_capacity,omitempty" validate:"omitempty,min=0"`
	ClassRoomDescription *string         `json:"class_room_description,omitempty" validate:"omitempty"`
	ClassRoomIsVirtual   *bool           `json:"class_room_is_virtual,omitempty"`
	ClassRoomIsActive    *bool           `json:"class_room_is_active,omitempty"`
	ClassRoomFeatures    *datatypes.JSON `json:"class_room_features,omitempty"`
}

/* =======================================================
   PATCH DTO (tri-state)
   ======================================================= */

type PatchClassRoomRequest struct {
	ClassRoomName        Optional[string]         `json:"class_room_name,omitempty"`
	ClassRoomCode        Optional[NullableString] `json:"class_room_code,omitempty"`
	ClassRoomSlug        Optional[NullableString] `json:"class_room_slug,omitempty"`
	ClassRoomLocation    Optional[NullableString] `json:"class_room_location,omitempty"`
	ClassRoomCapacity    Optional[NullableInt]    `json:"class_room_capacity,omitempty"`
	ClassRoomDescription Optional[NullableString] `json:"class_room_description,omitempty"`
	ClassRoomIsVirtual   Optional[bool]           `json:"class_room_is_virtual,omitempty"`
	ClassRoomIsActive    Optional[bool]           `json:"class_room_is_active,omitempty"`
	ClassRoomFeatures    Optional[datatypes.JSON] `json:"class_room_features,omitempty"`
}

func (p *PatchClassRoomRequest) Normalize() {
	if p.ClassRoomName.Present {
		p.ClassRoomName.Value = strings.TrimSpace(p.ClassRoomName.Value)
	}
	if p.ClassRoomCode.Present && p.ClassRoomCode.Value.Valid {
		p.ClassRoomCode.Value.Value = strings.TrimSpace(p.ClassRoomCode.Value.Value)
	}
	if p.ClassRoomSlug.Present && p.ClassRoomSlug.Value.Valid {
		s := strings.ToLower(strings.TrimSpace(p.ClassRoomSlug.Value.Value))
		p.ClassRoomSlug.Value.Value = s
	}
	if p.ClassRoomLocation.Present && p.ClassRoomLocation.Value.Valid {
		p.ClassRoomLocation.Value.Value = strings.TrimSpace(p.ClassRoomLocation.Value.Value)
	}
	if p.ClassRoomDescription.Present && p.ClassRoomDescription.Value.Valid {
		p.ClassRoomDescription.Value.Value = strings.TrimSpace(p.ClassRoomDescription.Value.Value)
	}
}

func (p *PatchClassRoomRequest) BuildUpdateMap() map[string]interface{} {
	up := make(map[string]interface{})

	if p.ClassRoomName.Present {
		up["class_room_name"] = p.ClassRoomName.Value
	}
	if p.ClassRoomCode.Present {
		if p.ClassRoomCode.Value.Valid {
			v := p.ClassRoomCode.Value.Value
			up["class_room_code"] = &v
		} else {
			up["class_room_code"] = nil
		}
	}
	if p.ClassRoomSlug.Present {
		if p.ClassRoomSlug.Value.Valid {
			v := p.ClassRoomSlug.Value.Value
			up["class_room_slug"] = &v
		} else {
			up["class_room_slug"] = nil
		}
	}
	if p.ClassRoomLocation.Present {
		if p.ClassRoomLocation.Value.Valid {
			v := p.ClassRoomLocation.Value.Value
			up["class_room_location"] = &v
		} else {
			up["class_room_location"] = nil
		}
	}
	if p.ClassRoomCapacity.Present {
		if p.ClassRoomCapacity.Value.Valid {
			v := p.ClassRoomCapacity.Value.Value
			up["class_room_capacity"] = &v
		} else {
			up["class_room_capacity"] = nil
		}
	}
	if p.ClassRoomDescription.Present {
		if p.ClassRoomDescription.Value.Valid {
			v := p.ClassRoomDescription.Value.Value
			up["class_room_description"] = &v
		} else {
			up["class_room_description"] = nil
		}
	}
	if p.ClassRoomIsVirtual.Present {
		up["class_room_is_virtual"] = p.ClassRoomIsVirtual.Value
	}
	if p.ClassRoomIsActive.Present {
		up["class_room_is_active"] = p.ClassRoomIsActive.Value
	}
	if p.ClassRoomFeatures.Present {
		up["class_room_features"] = p.ClassRoomFeatures.Value
	}

	return up
}

/* =======================================================
   RESPONSE DTO
   ======================================================= */

type ClassRoomResponse struct {
	ClassRoomID          uuid.UUID      `json:"class_room_id"`
	ClassRoomMasjidID    uuid.UUID      `json:"class_room_masjid_id"`
	ClassRoomName        string         `json:"class_room_name"`
	ClassRoomCode        *string        `json:"class_room_code,omitempty"`
	ClassRoomSlug        *string        `json:"class_room_slug,omitempty"`
	ClassRoomLocation    *string        `json:"class_room_location,omitempty"`
	ClassRoomCapacity    *int           `json:"class_room_capacity,omitempty"`
	ClassRoomDescription *string        `json:"class_room_description,omitempty"`
	ClassRoomIsVirtual   bool           `json:"class_room_is_virtual"`
	ClassRoomIsActive    bool           `json:"class_room_is_active"`
	ClassRoomFeatures    datatypes.JSON `json:"class_room_features"`
	ClassRoomCreatedAt   time.Time      `json:"class_room_created_at"`
	ClassRoomUpdatedAt   time.Time      `json:"class_room_updated_at"`
	ClassRoomDeletedAt   *time.Time     `json:"class_room_deleted_at,omitempty"`
}

func ToClassRoomResponse(m classroomModel.ClassRoomModel) ClassRoomResponse {
	var deletedAt *time.Time
	if m.ClassRoomDeletedAt.Valid {
		deletedAt = &m.ClassRoomDeletedAt.Time
	}

	return ClassRoomResponse{
		ClassRoomID:          m.ClassRoomID,
		ClassRoomMasjidID:    m.ClassRoomMasjidID,
		ClassRoomName:        m.ClassRoomName,
		ClassRoomCode:        m.ClassRoomCode,
		ClassRoomSlug:        m.ClassRoomSlug,
		ClassRoomLocation:    m.ClassRoomLocation,
		ClassRoomCapacity:    m.ClassRoomCapacity,
		ClassRoomDescription: m.ClassRoomDescription,
		ClassRoomIsVirtual:   m.ClassRoomIsVirtual,
		ClassRoomIsActive:    m.ClassRoomIsActive,
		ClassRoomFeatures:    m.ClassRoomFeatures,
		ClassRoomCreatedAt:   m.ClassRoomCreatedAt,
		ClassRoomUpdatedAt:   m.ClassRoomUpdatedAt,
		ClassRoomDeletedAt:   deletedAt,
	}
}

/* =======================================================
   QUERY FILTER DTO
   ======================================================= */

type ListClassRoomsQuery struct {
	Search      string `query:"search"`
	IsActive    *bool  `query:"is_active"`
	IsVirtual   *bool  `query:"is_virtual"`
	HasCodeOnly *bool  `query:"has_code_only"`
	Sort        string `query:"sort"`
	Limit       int    `query:"limit"`
	Offset      int    `query:"offset"`
}

func (q *ListClassRoomsQuery) Normalize() {
	q.Search = strings.TrimSpace(q.Search)
	q.Sort = strings.TrimSpace(strings.ToLower(q.Sort))
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
}

func PtrBool(b bool) *bool { return &b }

/* =======================================================
   NORMALIZER (CREATE/UPDATE)
   ======================================================= */

func (r *CreateClassRoomRequest) Normalize() {
	r.ClassRoomName = strings.TrimSpace(r.ClassRoomName)
	if r.ClassRoomCode != nil {
		c := strings.TrimSpace(*r.ClassRoomCode)
		r.ClassRoomCode = &c
	}
	if r.ClassRoomSlug != nil {
		s := strings.ToLower(strings.TrimSpace(*r.ClassRoomSlug))
		r.ClassRoomSlug = &s
	}
	if r.ClassRoomLocation != nil {
		l := strings.TrimSpace(*r.ClassRoomLocation)
		r.ClassRoomLocation = &l
	}
	if r.ClassRoomDescription != nil {
		d := strings.TrimSpace(*r.ClassRoomDescription)
		r.ClassRoomDescription = &d
	}
}

func (r *UpdateClassRoomRequest) Normalize() {
	if r.ClassRoomName != nil {
		v := strings.TrimSpace(*r.ClassRoomName)
		r.ClassRoomName = &v
	}
	if r.ClassRoomCode != nil {
		v := strings.TrimSpace(*r.ClassRoomCode)
		r.ClassRoomCode = &v
	}
	if r.ClassRoomSlug != nil {
		v := strings.ToLower(strings.TrimSpace(*r.ClassRoomSlug))
		r.ClassRoomSlug = &v
	}
	if r.ClassRoomLocation != nil {
		v := strings.TrimSpace(*r.ClassRoomLocation)
		r.ClassRoomLocation = &v
	}
	if r.ClassRoomDescription != nil {
		v := strings.TrimSpace(*r.ClassRoomDescription)
		r.ClassRoomDescription = &v
	}
}
