package dto

import (
	"strings"
	"time"

	// TODO: sesuaikan path model package-nya di project kamu
	"masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

/* =========================================================
   REQUESTS
========================================================= */

// CREATE
type CreateClassParentRequest struct {
	ClassParentMasjidID        uuid.UUID `json:"class_parent_masjid_id" validate:"required"`
	ClassParentName            string    `json:"class_parent_name" validate:"required,max=120"`
	ClassParentCode            string    `json:"class_parent_code" validate:"omitempty,max=40"`
	ClassParentDescription     string    `json:"class_parent_description" validate:"omitempty"`
	ClassParentLevel           *int16    `json:"class_parent_level" validate:"omitempty,gte=0,lte=100"`
	ClassParentImageURL        string    `json:"class_parent_image_url" validate:"omitempty,url"`
	ClassParentIsActive        *bool     `json:"class_parent_is_active" validate:"omitempty"`
	ClassParentTrashURL        string    `json:"class_parent_trash_url" validate:"omitempty,url"`
	ClassParentDeletePendingAt *time.Time `json:"class_parent_delete_pending_until" validate:"omitempty"`
}

// UPDATE / PATCH (semua pointer → optional)
type UpdateClassParentRequest struct {
	ClassParentName            *string    `json:"class_parent_name" validate:"omitempty,max=120"`
	ClassParentCode            *string    `json:"class_parent_code" validate:"omitempty,max=40"`
	ClassParentDescription     *string    `json:"class_parent_description" validate:"omitempty"`
	ClassParentLevel           *int16     `json:"class_parent_level" validate:"omitempty,gte=0,lte=100"`
	ClassParentImageURL        *string    `json:"class_parent_image_url" validate:"omitempty,url"`
	ClassParentIsActive        *bool      `json:"class_parent_is_active" validate:"omitempty"`
	ClassParentTrashURL        *string    `json:"class_parent_trash_url" validate:"omitempty,url"`
	ClassParentDeletePendingAt *time.Time `json:"class_parent_delete_pending_until" validate:"omitempty"`
}

// LIST QUERY (untuk controller List)
type ListClassParentQuery struct {
	Q         string     `query:"q"`                        // cari name/code/description
	MasjidID  *uuid.UUID `query:"masjid_id"`               // filter tenant
	Active    *bool      `query:"active"`                  // true/false
	LevelMin  *int       `query:"level_min"`               // 0..100
	LevelMax  *int       `query:"level_max"`               // 0..100
	Limit     int        `query:"limit"`                   // default handled di controller
	Offset    int        `query:"offset"`                  // default handled di controller
	CreatedGt *time.Time `query:"created_gt"`             // optional range
	CreatedLt *time.Time `query:"created_lt"`
}

/* =========================================================
   RESPONSES
========================================================= */

type ClassParentResponse struct {
	ClassParentID              uuid.UUID  `json:"class_parent_id"`
	ClassParentMasjidID        uuid.UUID  `json:"class_parent_masjid_id"`

	ClassParentName            string     `json:"class_parent_name"`
	ClassParentCode            string     `json:"class_parent_code"`
	ClassParentDescription     string     `json:"class_parent_description"`
	ClassParentLevel           *int16     `json:"class_parent_level"`
	ClassParentImageURL        string     `json:"class_parent_image_url"`

	ClassParentIsActive        bool       `json:"class_parent_is_active"`

	ClassParentTrashURL        string     `json:"class_parent_trash_url"`
	ClassParentDeletePendingAt *time.Time `json:"class_parent_delete_pending_until"`

	ClassParentCreatedAt       string     `json:"class_parent_created_at"`
	ClassParentUpdatedAt       string     `json:"class_parent_updated_at"`
}

type PaginationMeta struct {
	Total       int64  `json:"total"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	NextOffset  int    `json:"next_offset"`
	PrevOffset  int    `json:"prev_offset"`
	Returned    int    `json:"returned"`
	ServerTime  string `json:"server_time"`
}

type ClassParentListResponse struct {
	Data []ClassParentResponse `json:"data"`
	Meta PaginationMeta        `json:"meta"`
}

/* =========================================================
   HELPERS (converter & patch applier)
========================================================= */

// Converter: model → response
func ToClassParentResponse(m model.ClassParentModel) ClassParentResponse {
	return ClassParentResponse{
		ClassParentID:              m.ClassParentID,
		ClassParentMasjidID:        m.ClassParentMasjidID,

		ClassParentName:            m.ClassParentName,
		ClassParentCode:            m.ClassParentCode,
		ClassParentDescription:     m.ClassParentDescription,
		ClassParentLevel:           m.ClassParentLevel,
		ClassParentImageURL:        m.ClassParentImageURL,

		ClassParentIsActive:        m.ClassParentIsActive,

		ClassParentTrashURL:        m.ClassParentTrashURL,
		ClassParentDeletePendingAt: m.ClassParentDeletePendingUntil,

		ClassParentCreatedAt:       m.ClassParentCreatedAt.Format(time.RFC3339),
		ClassParentUpdatedAt:       m.ClassParentUpdatedAt.Format(time.RFC3339),
	}
}

func ToClassParentResponses(ms []model.ClassParentModel) []ClassParentResponse {
	out := make([]ClassParentResponse, 0, len(ms))
	for _, m := range ms {
		out = append(out, ToClassParentResponse(m))
	}
	return out
}

// Builder meta untuk list
func NewPaginationMeta(total int64, limit, offset, returned int) PaginationMeta {
	prev := offset - limit
	if prev < 0 {
		prev = 0
	}
	return PaginationMeta{
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		NextOffset: offset + limit,
		PrevOffset: prev,
		Returned:   returned,
		ServerTime: time.Now().Format(time.RFC3339),
	}
}

// Create → Model
func (r *CreateClassParentRequest) ToModel() model.ClassParentModel {
	m := model.ClassParentModel{
		ClassParentMasjidID:        r.ClassParentMasjidID,
		ClassParentName:            strings.TrimSpace(r.ClassParentName),
		ClassParentCode:            strings.TrimSpace(r.ClassParentCode),
		ClassParentDescription:     strings.TrimSpace(r.ClassParentDescription),
		ClassParentLevel:           r.ClassParentLevel,
		ClassParentImageURL:        strings.TrimSpace(r.ClassParentImageURL),
		ClassParentIsActive:        true, // default DB juga true
		ClassParentTrashURL:        strings.TrimSpace(r.ClassParentTrashURL),
		ClassParentDeletePendingUntil: r.ClassParentDeletePendingAt,
	}

	if r.ClassParentIsActive != nil {
		m.ClassParentIsActive = *r.ClassParentIsActive
	}
	return m
}

// PATCH applier (Update) → terapkan field yang terisi saja
func (r *UpdateClassParentRequest) ApplyPatch(m *model.ClassParentModel) {
	if r.ClassParentName != nil {
		m.ClassParentName = strings.TrimSpace(*r.ClassParentName)
	}
	if r.ClassParentCode != nil {
		m.ClassParentCode = strings.TrimSpace(*r.ClassParentCode)
	}
	if r.ClassParentDescription != nil {
		m.ClassParentDescription = strings.TrimSpace(*r.ClassParentDescription)
	}
	if r.ClassParentLevel != nil {
		m.ClassParentLevel = r.ClassParentLevel
	}
	if r.ClassParentImageURL != nil {
		m.ClassParentImageURL = strings.TrimSpace(*r.ClassParentImageURL)
	}
	if r.ClassParentIsActive != nil {
		m.ClassParentIsActive = *r.ClassParentIsActive
	}
	if r.ClassParentTrashURL != nil {
		m.ClassParentTrashURL = strings.TrimSpace(*r.ClassParentTrashURL)
	}
	if r.ClassParentDeletePendingAt != nil {
		m.ClassParentDeletePendingUntil = r.ClassParentDeletePendingAt
	}
}
