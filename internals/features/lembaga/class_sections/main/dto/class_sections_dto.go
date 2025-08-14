// internals/features/lembaga/classes/sections/main/dto/class_section_dto.go
package dto

import (
	"encoding/json"
	"time"

	"masjidku_backend/internals/features/lembaga/class_sections/main/model"

	"github.com/google/uuid"
	"gorm.io/datatypes" // ⬅️ tambah ini
)

/* ===================== Requests ===================== */

type CreateClassSectionRequest struct {
	ClassSectionMasjidID *uuid.UUID      `json:"class_sections_masjid_id" validate:"omitempty"`
	ClassSectionClassID  uuid.UUID       `json:"class_sections_class_id" validate:"required"`
	ClassSectionSlug     string          `json:"class_sections_slug" validate:"omitempty,min=1,max=160"` // ⬅️ ganti required -> omitempty
	ClassSectionName     string          `json:"class_sections_name" validate:"required,min=1,max=100"`
	ClassSectionCode     *string         `json:"class_sections_code" validate:"omitempty,max=50"`
	ClassSectionCapacity *int            `json:"class_sections_capacity" validate:"omitempty,gte=0"`
	ClassSectionSchedule json.RawMessage `json:"class_sections_schedule" validate:"omitempty"`
	ClassSectionIsActive *bool           `json:"class_sections_is_active" validate:"omitempty"`
}


func (r *CreateClassSectionRequest) ToModel() *model.ClassSectionModel {
	m := &model.ClassSectionModel{
		ClassID:  r.ClassSectionClassID,
		MasjidID: r.ClassSectionMasjidID,
		Slug:     r.ClassSectionSlug,
		Name:     r.ClassSectionName,
		Code:     r.ClassSectionCode,
		Capacity: r.ClassSectionCapacity,
		IsActive: true, // default
	}
	if len(r.ClassSectionSchedule) > 0 {
    m.Schedule = datatypes.JSON(r.ClassSectionSchedule)  // ⬅️ cast
	}

	if r.ClassSectionIsActive != nil {
		m.IsActive = *r.ClassSectionIsActive
	}
	return m
}

type UpdateClassSectionRequest struct {
	// di-controller: cegah ganti tenant jika perlu
	ClassSectionMasjidID *uuid.UUID       `json:"class_sections_masjid_id" validate:"omitempty"`
	ClassSectionClassID  *uuid.UUID       `json:"class_sections_class_id" validate:"omitempty"`
	ClassSectionSlug     *string          `json:"class_sections_slug" validate:"omitempty,min=1,max=160"`
	ClassSectionName     *string          `json:"class_sections_name" validate:"omitempty,min=1,max=100"`
	ClassSectionCode     *string          `json:"class_sections_code" validate:"omitempty,max=50"`
	ClassSectionCapacity *int             `json:"class_sections_capacity" validate:"omitempty,gte=0"`
	ClassSectionSchedule *json.RawMessage `json:"class_sections_schedule" validate:"omitempty"`
	ClassSectionIsActive *bool            `json:"class_sections_is_active" validate:"omitempty"`
}

func (r *UpdateClassSectionRequest) ApplyToModel(m *model.ClassSectionModel) {
	if r.ClassSectionMasjidID != nil {
		m.MasjidID = r.ClassSectionMasjidID
	}
	if r.ClassSectionClassID != nil {
		m.ClassID = *r.ClassSectionClassID
	}
	if r.ClassSectionSlug != nil {
		m.Slug = *r.ClassSectionSlug
	}
	if r.ClassSectionName != nil {
		m.Name = *r.ClassSectionName
	}
	if r.ClassSectionCode != nil {
		// boleh set ke string kosong; kalau mau null, kirimkan explicit null lewat API dan handle di controller
		m.Code = r.ClassSectionCode
	}
	if r.ClassSectionCapacity != nil {
		m.Capacity = r.ClassSectionCapacity
	}
	if r.ClassSectionSchedule != nil {
    m.Schedule = datatypes.JSON(*r.ClassSectionSchedule) // ⬅️ cast
	}
	if r.ClassSectionIsActive != nil {
		m.IsActive = *r.ClassSectionIsActive
	}
}

/* ===================== Queries ===================== */

type ListClassSectionQuery struct {
	Limit     int        `query:"limit"`
	Offset    int        `query:"offset"`
	ActiveOnly *bool     `query:"active_only"`
	Search    *string    `query:"search"`   // cari by name / code (controller yang handle)
	ClassID   *uuid.UUID `query:"class_id"` // filter by class
	Sort      *string    `query:"sort"`     // name_asc|name_desc|created_at_asc|created_at_desc
}

/* ===================== Responses ===================== */

type ClassSectionResponse struct {
	ClassSectionID       uuid.UUID       `json:"class_sections_id"`
	ClassSectionClassID  uuid.UUID       `json:"class_sections_class_id"`
	ClassSectionMasjidID *uuid.UUID      `json:"class_sections_masjid_id,omitempty"`

	ClassSectionSlug string           `json:"class_sections_slug"`
	ClassSectionName string           `json:"class_sections_name"`
	ClassSectionCode *string          `json:"class_sections_code,omitempty"`
	ClassSectionCapacity *int         `json:"class_sections_capacity,omitempty"`
	ClassSectionSchedule json.RawMessage `json:"class_sections_schedule,omitempty"`

	ClassSectionIsActive bool        `json:"class_sections_is_active"`
	ClassSectionCreatedAt time.Time  `json:"class_sections_created_at"`
	ClassSectionUpdatedAt *time.Time `json:"class_sections_updated_at,omitempty"`
	ClassSectionDeletedAt *time.Time `json:"class_sections_deleted_at,omitempty"`
}

func NewClassSectionResponse(m *model.ClassSectionModel) *ClassSectionResponse {
	return &ClassSectionResponse{
		ClassSectionID:       m.ClassSectionID,
		ClassSectionClassID:  m.ClassID,
		ClassSectionMasjidID: m.MasjidID,

		ClassSectionSlug:     m.Slug,
		ClassSectionName:     m.Name,
		ClassSectionCode:     m.Code,
		ClassSectionCapacity: m.Capacity,
		ClassSectionSchedule: json.RawMessage(m.Schedule),

		ClassSectionIsActive: m.IsActive,
		ClassSectionCreatedAt: m.CreatedAt,
		ClassSectionUpdatedAt: m.UpdatedAt,
		ClassSectionDeletedAt: m.DeletedAt,
	}
}
