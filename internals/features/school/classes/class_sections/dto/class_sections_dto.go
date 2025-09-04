// file: internals/features/school/classes/sections/main/dto/class_section_dto.go
package dto

import (
	"encoding/json"
	"time"

	m "masjidku_backend/internals/features/school/classes/class_sections/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* ===================== Requests ===================== */

// Create: always active when created (server sets IsActive = true)
type CreateClassSectionRequest struct {
	ClassSectionsMasjidID     uuid.UUID       `json:"class_sections_masjid_id"`                 // required
	ClassSectionsClassID      uuid.UUID       `json:"class_sections_class_id"`                  // required
	ClassSectionsTeacherID    *uuid.UUID      `json:"class_sections_teacher_id,omitempty"`      // masjid_teachers.*
	ClassSectionsClassRoomID  *uuid.UUID      `json:"class_sections_class_room_id,omitempty"`   // class_rooms.class_room_id (opsional)
	ClassSectionsSlug         string          `json:"class_sections_slug,omitempty"`            // optional (can be generated server-side)
	ClassSectionsName         string          `json:"class_sections_name"`                      // required
	ClassSectionsCode         *string         `json:"class_sections_code,omitempty"`
	ClassSectionsCapacity     *int            `json:"class_sections_capacity,omitempty"`        // >= 0
	ClassSectionsSchedule     json.RawMessage `json:"class_sections_schedule,omitempty"`        // JSONB blob
}

func (r *CreateClassSectionRequest) ToModel() *m.ClassSectionModel {
	out := &m.ClassSectionModel{
		ClassSectionsClassID:      r.ClassSectionsClassID,
		ClassSectionsMasjidID:     r.ClassSectionsMasjidID,
		ClassSectionsTeacherID:    r.ClassSectionsTeacherID,
		ClassSectionsClassRoomID:  r.ClassSectionsClassRoomID,
		ClassSectionsSlug:         r.ClassSectionsSlug,
		ClassSectionsName:         r.ClassSectionsName,
		ClassSectionsCode:         r.ClassSectionsCode,
		ClassSectionsCapacity:     r.ClassSectionsCapacity,
		ClassSectionsIsActive:     true, // forced on create
		// ClassSectionsTotalStudents default (0)
	}
	if len(r.ClassSectionsSchedule) > 0 {
		out.ClassSectionsSchedule = datatypes.JSON(r.ClassSectionsSchedule)
	}
	return out
}

// Update (PATCH-like at DTO level; controller may enforce PUT semantics)
type UpdateClassSectionRequest struct {
	ClassSectionsMasjidID     *uuid.UUID      `json:"class_sections_masjid_id,omitempty"`
	ClassSectionsClassID      *uuid.UUID      `json:"class_sections_class_id,omitempty"`
	ClassSectionsTeacherID    *uuid.UUID      `json:"class_sections_teacher_id,omitempty"`     // masjid_teachers.*
	ClassSectionsClassRoomID  *uuid.UUID      `json:"class_sections_class_room_id,omitempty"`  // class_rooms.*
	ClassSectionsSlug         *string         `json:"class_sections_slug,omitempty"`
	ClassSectionsName         *string         `json:"class_sections_name,omitempty"`
	ClassSectionsCode         *string         `json:"class_sections_code,omitempty"`
	ClassSectionsCapacity     *int            `json:"class_sections_capacity,omitempty"`       // >= 0
	ClassSectionsSchedule     *json.RawMessage `json:"class_sections_schedule,omitempty"`
	ClassSectionsIsActive     *bool           `json:"class_sections_is_active,omitempty"`
}

func (r *UpdateClassSectionRequest) ApplyToModel(dst *m.ClassSectionModel) {
	if r.ClassSectionsMasjidID != nil {
		dst.ClassSectionsMasjidID = *r.ClassSectionsMasjidID
	}
	if r.ClassSectionsClassID != nil {
		dst.ClassSectionsClassID = *r.ClassSectionsClassID
	}
	if r.ClassSectionsTeacherID != nil {
		dst.ClassSectionsTeacherID = r.ClassSectionsTeacherID
	}
	if r.ClassSectionsClassRoomID != nil {
		dst.ClassSectionsClassRoomID = r.ClassSectionsClassRoomID
	}
	if r.ClassSectionsSlug != nil {
		dst.ClassSectionsSlug = *r.ClassSectionsSlug
	}
	if r.ClassSectionsName != nil {
		dst.ClassSectionsName = *r.ClassSectionsName
	}
	if r.ClassSectionsCode != nil {
		dst.ClassSectionsCode = r.ClassSectionsCode // allow empty string
	}
	if r.ClassSectionsCapacity != nil {
		dst.ClassSectionsCapacity = r.ClassSectionsCapacity
	}
	if r.ClassSectionsSchedule != nil {
		dst.ClassSectionsSchedule = datatypes.JSON(*r.ClassSectionsSchedule)
	}
	if r.ClassSectionsIsActive != nil {
		dst.ClassSectionsIsActive = *r.ClassSectionsIsActive
	}
}

/* ===================== Queries ===================== */

type ListClassSectionQuery struct {
	Limit      int         `query:"limit"`
	Offset     int         `query:"offset"`
	ActiveOnly *bool       `query:"active_only"`
	Search     *string     `query:"search"`      // name/code/slug (controller handles)
	ClassID    *uuid.UUID  `query:"class_id"`    // filter by class
	TeacherID  *uuid.UUID  `query:"teacher_id"`  // filter by teacher (masjid_teachers.*)
	RoomID     *uuid.UUID  `query:"room_id"`     // filter by class_rooms.class_room_id
	Sort       *string     `query:"sort"`        // name_asc|name_desc|created_at_asc|created_at_desc
}

/* ===================== Responses ===================== */

type UserLite struct {
	ID       uuid.UUID `json:"id"`
	UserName string    `json:"user_name"`
	Email    string    `json:"email"`
	IsActive bool      `json:"is_active"`
	FullName string    `json:"full_name"`
}

type ClassSectionResponse struct {
	ClassSectionsID            uuid.UUID       `json:"class_sections_id"`
	ClassSectionsClassID       uuid.UUID       `json:"class_sections_class_id"`
	ClassSectionsMasjidID      uuid.UUID       `json:"class_sections_masjid_id"`
	ClassSectionsTeacherID     *uuid.UUID      `json:"class_sections_teacher_id,omitempty"`
	ClassSectionsClassRoomID   *uuid.UUID      `json:"class_sections_class_room_id,omitempty"`

	ClassSectionsSlug          string          `json:"class_sections_slug"`
	ClassSectionsName          string          `json:"class_sections_name"`
	ClassSectionsCode          *string         `json:"class_sections_code,omitempty"`
	ClassSectionsCapacity      *int            `json:"class_sections_capacity,omitempty"`
	ClassSectionsSchedule      json.RawMessage `json:"class_sections_schedule,omitempty"`

	// Denormalized counter
	ClassSectionsTotalStudents int            `json:"class_sections_total_students"`

	ClassSectionsIsActive      bool           `json:"class_sections_is_active"`
	ClassSectionsCreatedAt     time.Time      `json:"class_sections_created_at"`
	ClassSectionsUpdatedAt     time.Time      `json:"class_sections_updated_at"`
	ClassSectionsDeletedAt     *time.Time     `json:"class_sections_deleted_at,omitempty"`

	Teacher *UserLite `json:"teacher,omitempty"` // enrichment (optional)
}

// builder with teacher name only
func NewClassSectionResponse(src *m.ClassSectionModel, teacherName string) *ClassSectionResponse {
	var deletedAt *time.Time
	if !src.ClassSectionsDeletedAt.Time.IsZero() {
		t := src.ClassSectionsDeletedAt.Time
		deletedAt = &t
	}

	return &ClassSectionResponse{
		ClassSectionsID:            src.ClassSectionsID,
		ClassSectionsClassID:       src.ClassSectionsClassID,
		ClassSectionsMasjidID:      src.ClassSectionsMasjidID,
		ClassSectionsTeacherID:     src.ClassSectionsTeacherID,
		ClassSectionsClassRoomID:   src.ClassSectionsClassRoomID,

		ClassSectionsSlug:          src.ClassSectionsSlug,
		ClassSectionsName:          src.ClassSectionsName,
		ClassSectionsCode:          src.ClassSectionsCode,
		ClassSectionsCapacity:      src.ClassSectionsCapacity,
		ClassSectionsSchedule:      json.RawMessage(src.ClassSectionsSchedule),

		ClassSectionsTotalStudents: src.ClassSectionsTotalStudents,

		ClassSectionsIsActive:      src.ClassSectionsIsActive,
		ClassSectionsCreatedAt:     src.ClassSectionsCreatedAt,
		ClassSectionsUpdatedAt:     src.ClassSectionsUpdatedAt,
		ClassSectionsDeletedAt:     deletedAt,

		Teacher: &UserLite{FullName: teacherName},
	}
}

// builder with full Teacher object (preferred)
func NewClassSectionResponseWithTeacher(src *m.ClassSectionModel, t *UserLite) *ClassSectionResponse {
	teacherName := ""
	if t != nil {
		teacherName = t.FullName
	}
	resp := NewClassSectionResponse(src, teacherName)
	resp.Teacher = t
	return resp
}

func MapClassSectionsWithTeachers(models []m.ClassSectionModel, users map[uuid.UUID]UserLite) []ClassSectionResponse {
	out := make([]ClassSectionResponse, 0, len(models))
	for i := range models {
		row := &models[i]
		var t *UserLite
		if row.ClassSectionsTeacherID != nil {
			if u, ok := users[*row.ClassSectionsTeacherID]; ok {
				uCopy := u
				t = &uCopy
			}
		}
		out = append(out, *NewClassSectionResponseWithTeacher(row, t))
	}
	return out
}
