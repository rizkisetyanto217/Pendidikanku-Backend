// file: internals/features/school/class_rooms/dto/class_room_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	classroomsModel "masjidku_backend/internals/features/school/schedule_daily_rooms/rooms/model"

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
	ClassRoomsName      string         `json:"class_rooms_name" validate:"required,min=3,max=100"`
	ClassRoomsCode      *string        `json:"class_rooms_code,omitempty" validate:"omitempty,max=50"`
	ClassRoomsLocation  *string        `json:"class_rooms_location,omitempty" validate:"omitempty,max=100"`
	ClassRoomsFloor     *int           `json:"class_rooms_floor,omitempty" validate:"omitempty"`
	ClassRoomsCapacity  *int           `json:"class_rooms_capacity,omitempty" validate:"omitempty,min=0"`
	ClassRoomsIsVirtual bool           `json:"class_rooms_is_virtual"`
	ClassRoomsIsActive  bool           `json:"class_rooms_is_active"`
	ClassRoomsFeatures  datatypes.JSON `json:"class_rooms_features" validate:"omitempty"`
}

type UpdateClassRoomRequest struct {
	ClassRoomsName      *string         `json:"class_rooms_name,omitempty" validate:"omitempty,min=3,max=100"`
	ClassRoomsCode      *string         `json:"class_rooms_code,omitempty" validate:"omitempty,max=50"`
	ClassRoomsLocation  *string         `json:"class_rooms_location,omitempty" validate:"omitempty,max=100"`
	ClassRoomsFloor     *int            `json:"class_rooms_floor,omitempty" validate:"omitempty"`
	ClassRoomsCapacity  *int            `json:"class_rooms_capacity,omitempty" validate:"omitempty,min=0"`
	ClassRoomsIsVirtual *bool           `json:"class_rooms_is_virtual,omitempty"`
	ClassRoomsIsActive  *bool           `json:"class_rooms_is_active,omitempty"`
	ClassRoomsFeatures  *datatypes.JSON `json:"class_rooms_features,omitempty"`
}

/* =======================================================
   PATCH DTO (tri-state)
   ======================================================= */

type PatchClassRoomRequest struct {
	ClassRoomsName      Optional[string]         `json:"class_rooms_name,omitempty"`
	ClassRoomsCode      Optional[NullableString] `json:"class_rooms_code,omitempty"`
	ClassRoomsLocation  Optional[NullableString] `json:"class_rooms_location,omitempty"`
	ClassRoomsFloor     Optional[NullableInt]    `json:"class_rooms_floor,omitempty"`
	ClassRoomsCapacity  Optional[NullableInt]    `json:"class_rooms_capacity,omitempty"`
	ClassRoomsIsVirtual Optional[bool]           `json:"class_rooms_is_virtual,omitempty"`
	ClassRoomsIsActive  Optional[bool]           `json:"class_rooms_is_active,omitempty"`
	ClassRoomsFeatures  Optional[datatypes.JSON] `json:"class_rooms_features,omitempty"`
}

func (p *PatchClassRoomRequest) Normalize() {
	if p.ClassRoomsName.Present {
		p.ClassRoomsName.Value = strings.TrimSpace(p.ClassRoomsName.Value)
	}
	if p.ClassRoomsCode.Present && p.ClassRoomsCode.Value.Valid {
		p.ClassRoomsCode.Value.Value = strings.TrimSpace(p.ClassRoomsCode.Value.Value)
	}
	if p.ClassRoomsLocation.Present && p.ClassRoomsLocation.Value.Valid {
		p.ClassRoomsLocation.Value.Value = strings.TrimSpace(p.ClassRoomsLocation.Value.Value)
	}
}

func (p *PatchClassRoomRequest) BuildUpdateMap() map[string]interface{} {
	up := make(map[string]interface{})

	if p.ClassRoomsName.Present {
		up["class_rooms_name"] = p.ClassRoomsName.Value
	}
	if p.ClassRoomsCode.Present {
		if p.ClassRoomsCode.Value.Valid {
			v := p.ClassRoomsCode.Value.Value
			up["class_rooms_code"] = &v
		} else {
			up["class_rooms_code"] = nil
		}
	}
	if p.ClassRoomsLocation.Present {
		if p.ClassRoomsLocation.Value.Valid {
			v := p.ClassRoomsLocation.Value.Value
			up["class_rooms_location"] = &v
		} else {
			up["class_rooms_location"] = nil
		}
	}
	if p.ClassRoomsFloor.Present {
		if p.ClassRoomsFloor.Value.Valid {
			v := p.ClassRoomsFloor.Value.Value
			up["class_rooms_floor"] = &v
		} else {
			up["class_rooms_floor"] = nil
		}
	}
	if p.ClassRoomsCapacity.Present {
		if p.ClassRoomsCapacity.Value.Valid {
			v := p.ClassRoomsCapacity.Value.Value
			up["class_rooms_capacity"] = &v
		} else {
			up["class_rooms_capacity"] = nil
		}
	}
	if p.ClassRoomsIsVirtual.Present {
		up["class_rooms_is_virtual"] = p.ClassRoomsIsVirtual.Value
	}
	if p.ClassRoomsIsActive.Present {
		up["class_rooms_is_active"] = p.ClassRoomsIsActive.Value
	}
	if p.ClassRoomsFeatures.Present {
		up["class_rooms_features"] = p.ClassRoomsFeatures.Value
	}

	return up
}

/* =======================================================
   RESPONSE DTO
   ======================================================= */

type ClassRoomResponse struct {
	ClassRoomID         uuid.UUID      `json:"class_room_id"`
	ClassRoomsMasjidID  uuid.UUID      `json:"class_rooms_masjid_id"`
	ClassRoomsName      string         `json:"class_rooms_name"`
	ClassRoomsCode      *string        `json:"class_rooms_code,omitempty"`
	ClassRoomsLocation  *string        `json:"class_rooms_location,omitempty"`
	ClassRoomsFloor     *int           `json:"class_rooms_floor,omitempty"`
	ClassRoomsCapacity  *int           `json:"class_rooms_capacity,omitempty"`
	ClassRoomsIsVirtual bool           `json:"class_rooms_is_virtual"`
	ClassRoomsIsActive  bool           `json:"class_rooms_is_active"`
	ClassRoomsFeatures  datatypes.JSON `json:"class_rooms_features"`
	ClassRoomsCreatedAt time.Time      `json:"class_rooms_created_at"`
	ClassRoomsUpdatedAt time.Time      `json:"class_rooms_updated_at"`
	ClassRoomsDeletedAt *time.Time     `json:"class_rooms_deleted_at,omitempty"`
}

func ToClassRoomResponse(m classroomsModel.ClassRoomModel) ClassRoomResponse {
	var deletedAt *time.Time
	if m.ClassRoomsDeletedAt.Valid {
		deletedAt = &m.ClassRoomsDeletedAt.Time
	}

	return ClassRoomResponse{
		ClassRoomID:         m.ClassRoomID,
		ClassRoomsMasjidID:  m.ClassRoomsMasjidID,
		ClassRoomsName:      m.ClassRoomsName,
		ClassRoomsCode:      m.ClassRoomsCode,
		ClassRoomsLocation:  m.ClassRoomsLocation,
		ClassRoomsFloor:     m.ClassRoomsFloor,
		ClassRoomsCapacity:  m.ClassRoomsCapacity,
		ClassRoomsIsVirtual: m.ClassRoomsIsVirtual,
		ClassRoomsIsActive:  m.ClassRoomsIsActive,
		ClassRoomsFeatures:  m.ClassRoomsFeatures,
		ClassRoomsCreatedAt: m.ClassRoomsCreatedAt,
		ClassRoomsUpdatedAt: m.ClassRoomsUpdatedAt,
		ClassRoomsDeletedAt: deletedAt,
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
	r.ClassRoomsName = strings.TrimSpace(r.ClassRoomsName)
	if r.ClassRoomsCode != nil {
		c := strings.TrimSpace(*r.ClassRoomsCode)
		r.ClassRoomsCode = &c
	}
	if r.ClassRoomsLocation != nil {
		l := strings.TrimSpace(*r.ClassRoomsLocation)
		r.ClassRoomsLocation = &l
	}
}

func (r *UpdateClassRoomRequest) Normalize() {
	if r.ClassRoomsName != nil {
		v := strings.TrimSpace(*r.ClassRoomsName)
		r.ClassRoomsName = &v
	}
	if r.ClassRoomsCode != nil {
		v := strings.TrimSpace(*r.ClassRoomsCode)
		r.ClassRoomsCode = &v
	}
	if r.ClassRoomsLocation != nil {
		v := strings.TrimSpace(*r.ClassRoomsLocation)
		r.ClassRoomsLocation = &v
	}
}
