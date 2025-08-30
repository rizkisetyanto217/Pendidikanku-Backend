// internals/features/school/classes/sections/main/dto/class_section_dto.go
package dto

import (
	"encoding/json"
	"time"

	"masjidku_backend/internals/features/school/classes/class_sections/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ===================== Requests =====================

// Create: Always active when created (doesn't accept active flag from client)
type CreateClassSectionRequest struct {
	ClassSectionsMasjidID  *uuid.UUID      `json:"class_sections_masjid_id" validate:"omitempty"`
	ClassSectionsClassID   uuid.UUID       `json:"class_sections_class_id" validate:"required"`
	ClassSectionsTeacherID *uuid.UUID      `json:"class_sections_teacher_id" validate:"omitempty"`  // TeacherID refers to masjid_teachers
	ClassSectionsSlug      string          `json:"class_sections_slug" validate:"omitempty,min=1,max=160"` // slug can be generated server-side
	ClassSectionsName      string          `json:"class_sections_name" validate:"required,min=1,max=100"`
	ClassSectionsCode      *string         `json:"class_sections_code" validate:"omitempty,max=50"`
	ClassSectionsCapacity  *int            `json:"class_sections_capacity" validate:"omitempty,gte=0"`
	ClassSectionsSchedule  json.RawMessage `json:"class_sections_schedule" validate:"omitempty"`
}

func (r *CreateClassSectionRequest) ToModel() *model.ClassSectionModel {
	m := &model.ClassSectionModel{
		ClassSectionsClassID:   r.ClassSectionsClassID,
		ClassSectionsMasjidID:  r.ClassSectionsMasjidID,
		ClassSectionsTeacherID: r.ClassSectionsTeacherID,  // TeacherID refers to masjid_teachers
		ClassSectionsSlug:      r.ClassSectionsSlug,
		ClassSectionsName:      r.ClassSectionsName,
		ClassSectionsCode:      r.ClassSectionsCode,
		ClassSectionsCapacity:  r.ClassSectionsCapacity,
		ClassSectionsIsActive:  true, // always active when created
	}
	if len(r.ClassSectionsSchedule) > 0 {
		m.ClassSectionsSchedule = datatypes.JSON(r.ClassSectionsSchedule)
	}
	return m
}

// Update: Allows changing active flag and other fields
type UpdateClassSectionRequest struct {
	ClassSectionsMasjidID  *uuid.UUID       `json:"class_sections_masjid_id" validate:"omitempty"`
	ClassSectionsClassID   *uuid.UUID       `json:"class_sections_class_id" validate:"omitempty"`
	ClassSectionsTeacherID *uuid.UUID       `json:"class_sections_teacher_id" validate:"omitempty"` // TeacherID refers to masjid_teachers
	ClassSectionsSlug      *string          `json:"class_sections_slug" validate:"omitempty,min=1,max=160"`
	ClassSectionsName      *string          `json:"class_sections_name" validate:"omitempty,min=1,max=100"`
	ClassSectionsCode      *string          `json:"class_sections_code" validate:"omitempty,max=50"`
	ClassSectionsCapacity  *int             `json:"class_sections_capacity" validate:"omitempty,gte=0"`
	ClassSectionsSchedule  *json.RawMessage `json:"class_sections_schedule" validate:"omitempty"`
	ClassSectionsIsActive  *bool            `json:"class_sections_is_active" validate:"omitempty"`
}

func (r *UpdateClassSectionRequest) ApplyToModel(m *model.ClassSectionModel) {
	if r.ClassSectionsMasjidID != nil {
		m.ClassSectionsMasjidID = r.ClassSectionsMasjidID
	}
	if r.ClassSectionsClassID != nil {
		m.ClassSectionsClassID = *r.ClassSectionsClassID
	}
	if r.ClassSectionsTeacherID != nil {
		m.ClassSectionsTeacherID = r.ClassSectionsTeacherID  // TeacherID refers to masjid_teachers
	}
	if r.ClassSectionsSlug != nil {
		m.ClassSectionsSlug = *r.ClassSectionsSlug
	}
	if r.ClassSectionsName != nil {
		m.ClassSectionsName = *r.ClassSectionsName
	}
	if r.ClassSectionsCode != nil {
		m.ClassSectionsCode = r.ClassSectionsCode // can be empty string
	}
	if r.ClassSectionsCapacity != nil {
		m.ClassSectionsCapacity = r.ClassSectionsCapacity
	}
	if r.ClassSectionsSchedule != nil {
		m.ClassSectionsSchedule = datatypes.JSON(*r.ClassSectionsSchedule)
	}
	if r.ClassSectionsIsActive != nil {
		m.ClassSectionsIsActive = *r.ClassSectionsIsActive
	}
}

// ===================== Queries =====================

type ListClassSectionQuery struct {
	Limit      int        `query:"limit"`
	Offset     int        `query:"offset"`
	ActiveOnly *bool      `query:"active_only"`
	Search     *string    `query:"search"`     // search by name/code/slug (controller handle)
	ClassID    *uuid.UUID `query:"class_id"`   // filter by class
	TeacherID  *uuid.UUID `query:"teacher_id"` // filter by teacher
	Sort       *string    `query:"sort"`       // name_asc|name_desc|created_at_asc|created_at_desc
}

// ===================== Responses =====================

// Add FullName to UserLite struct
type UserLite struct {
	ID        uuid.UUID `json:"id"`
	UserName  string    `json:"user_name"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"is_active"`
	FullName  string    `json:"full_name"` // Add FullName field
}


type ClassSectionResponse struct {
	ClassSectionsID        uuid.UUID      `json:"class_sections_id"`
	ClassSectionsClassID   uuid.UUID      `json:"class_sections_class_id"`
	ClassSectionsMasjidID  *uuid.UUID     `json:"class_sections_masjid_id,omitempty"`
	ClassSectionsTeacherID *uuid.UUID     `json:"class_sections_teacher_id,omitempty"`

	ClassSectionsSlug     string          `json:"class_sections_slug"`
	ClassSectionsName     string          `json:"class_sections_name"`
	ClassSectionsCode     *string         `json:"class_sections_code,omitempty"`
	ClassSectionsCapacity *int            `json:"class_sections_capacity,omitempty"`
	ClassSectionsSchedule json.RawMessage `json:"class_sections_schedule,omitempty"`

	ClassSectionsIsActive  bool       `json:"class_sections_is_active"`
	ClassSectionsCreatedAt time.Time  `json:"class_sections_created_at"`
	ClassSectionsUpdatedAt *time.Time `json:"class_sections_updated_at,omitempty"`
	ClassSectionsDeletedAt *time.Time `json:"class_sections_deleted_at,omitempty"`

	Teacher *UserLite `json:"teacher,omitempty"` // enrichment (optional)
}

// NewClassSectionResponse creates a new ClassSectionResponse from ClassSectionModel
// Create the response with teacher name
func NewClassSectionResponse(m *model.ClassSectionModel, teacherName string) *ClassSectionResponse {
	return &ClassSectionResponse{
		ClassSectionsID:        m.ClassSectionsID,
		ClassSectionsClassID:   m.ClassSectionsClassID,
		ClassSectionsMasjidID:  m.ClassSectionsMasjidID,
		ClassSectionsTeacherID: m.ClassSectionsTeacherID,

		ClassSectionsSlug:     m.ClassSectionsSlug,
		ClassSectionsName:     m.ClassSectionsName,
		ClassSectionsCode:     m.ClassSectionsCode,
		ClassSectionsCapacity: m.ClassSectionsCapacity,
		ClassSectionsSchedule: json.RawMessage(m.ClassSectionsSchedule),

		ClassSectionsIsActive:  m.ClassSectionsIsActive,
		ClassSectionsCreatedAt: m.ClassSectionsCreatedAt,
		ClassSectionsUpdatedAt: m.ClassSectionsUpdatedAt,
		ClassSectionsDeletedAt: m.ClassSectionsDeletedAt,
		Teacher:                &UserLite{FullName: teacherName}, // assign teacher name
	}
}

// Update this method to ensure teacherName is passed from user data
func NewClassSectionResponseWithTeacher(m *model.ClassSectionModel, t *UserLite) *ClassSectionResponse {
	teacherName := ""
	if t != nil {
		teacherName = t.FullName // Use the teacher's full name from UserLite
	}
	return NewClassSectionResponse(m, teacherName) // pass teacherName
}


func MapClassSectionsWithTeachers(models []model.ClassSectionModel, users map[uuid.UUID]UserLite) []ClassSectionResponse {
	out := make([]ClassSectionResponse, 0, len(models))
	for i := range models {
		m := &models[i]
		var t *UserLite
		if m.ClassSectionsTeacherID != nil {
			if u, ok := users[*m.ClassSectionsTeacherID]; ok {
				uCopy := u
				t = &uCopy
			}
		}
		resp := NewClassSectionResponseWithTeacher(m, t)
		out = append(out, *resp)
	}
	return out
}
