// file: internals/features/school/classes/dto/class_parent_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	m "masjidku_backend/internals/features/school/classes/classes/model"
)

/* =========================================================
   PATCH FIELD — tri-state (absent | null | value)
   ========================================================= */

type PatchFieldClassParent[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchFieldClassParent[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	// "null" → explicit null
	if string(b) == "null" {
		p.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

// Getter aman (opsional)
func (p PatchFieldClassParent[T]) Get() (*T, bool) { return p.Value, p.Present }

/* =========================================================
   CREATE REQUEST / RESPONSE
   ========================================================= */

type ClassParentCreateRequest struct {
	// Wajib
	ClassParentMasjidID uuid.UUID `json:"class_parent_masjid_id" validate:"required"`
	ClassParentName     string    `json:"class_parent_name" validate:"required,min=1,max=120"`

	// Opsional
	ClassParentCode        *string             `json:"class_parent_code" validate:"omitempty,max=40"`
	ClassParentSlug        *string             `json:"class_parent_slug" validate:"omitempty,max=160"`
	ClassParentDescription *string             `json:"class_parent_description"`
	ClassParentLevel       *int16              `json:"class_parent_level" validate:"omitempty,min=0,max=100"`
	ClassParentIsActive    *bool               `json:"class_parent_is_active"` // default true di DB
	ClassParentTotalClasses *int32             `json:"class_parent_total_classes" validate:"omitempty,min=0"`
	ClassParentRequirements datatypes.JSONMap  `json:"class_parent_requirements"` // default {}
	// Slot image (opsional—biasanya diisi saat upload selesai)
	ClassParentImageURL          *string `json:"class_parent_image_url"`
	ClassParentImageObjectKey    *string `json:"class_parent_image_object_key"`
}

func (r *ClassParentCreateRequest) Normalize() {
	trimPtr := func(p **string, lower bool) {
		if p == nil || *p == nil {
			return
		}
		v := strings.TrimSpace(**p)
		if v == "" {
			*p = nil
			return
		}
		if lower {
			v = strings.ToLower(v)
		}
		*p = &v
	}
	trimPtr(&r.ClassParentCode, false)
	trimPtr(&r.ClassParentSlug, true)
	// name wajib, trim saja
	r.ClassParentName = strings.TrimSpace(r.ClassParentName)
}

func (r ClassParentCreateRequest) ToModel() *m.ClassParentModel {
	now := time.Now()
	cp := &m.ClassParentModel{
		ClassParentMasjidID: r.ClassParentMasjidID,
		ClassParentName:     r.ClassParentName,
		ClassParentCode:     r.ClassParentCode,
		ClassParentSlug:     r.ClassParentSlug,
		ClassParentDescription: r.ClassParentDescription,
		ClassParentLevel:       r.ClassParentLevel,
		ClassParentRequirements: r.ClassParentRequirements,
		ClassParentCreatedAt:    now,
		ClassParentUpdatedAt:    now,
	}
	if r.ClassParentIsActive != nil {
		cp.ClassParentIsActive = *r.ClassParentIsActive
	} else {
		cp.ClassParentIsActive = true // default di sisi app (DB juga default true)
	}
	if r.ClassParentTotalClasses != nil {
		cp.ClassParentTotalClasses = *r.ClassParentTotalClasses
	}
	// image (opsional)
	cp.ClassParentImageURL = r.ClassParentImageURL
	cp.ClassParentImageObjectKey = r.ClassParentImageObjectKey
	return cp
}

type ClassParentResponse struct {
	ClassParentID              uuid.UUID         `json:"class_parent_id"`
	ClassParentMasjidID        uuid.UUID         `json:"class_parent_masjid_id"`
	ClassParentName            string            `json:"class_parent_name"`
	ClassParentCode            *string           `json:"class_parent_code"`
	ClassParentSlug            *string           `json:"class_parent_slug"`
	ClassParentDescription     *string           `json:"class_parent_description"`
	ClassParentLevel           *int16            `json:"class_parent_level"`
	ClassParentIsActive        bool              `json:"class_parent_is_active"`
	ClassParentTotalClasses    int32             `json:"class_parent_total_classes"`
	ClassParentRequirements    datatypes.JSONMap `json:"class_parent_requirements"`

	ClassParentImageURL                 *string    `json:"class_parent_image_url"`
	ClassParentImageObjectKey           *string    `json:"class_parent_image_object_key"`
	ClassParentImageURLOld              *string    `json:"class_parent_image_url_old"`
	ClassParentImageObjectKeyOld        *string    `json:"class_parent_image_object_key_old"`
	ClassParentImageDeletePendingUntil  *time.Time `json:"class_parent_image_delete_pending_until"`

	ClassParentCreatedAt time.Time  `json:"class_parent_created_at"`
	ClassParentUpdatedAt time.Time  `json:"class_parent_updated_at"`
	ClassParentDeletedAt *time.Time `json:"class_parent_deleted_at,omitempty"`
}

func FromModelClassParent(cp *m.ClassParentModel) ClassParentResponse {
	var deletedAt *time.Time
	if cp.ClassParentDeletedAt.Valid {
		t := cp.ClassParentDeletedAt.Time
		deletedAt = &t
	}
	return ClassParentResponse{
		ClassParentID:               cp.ClassParentID,
		ClassParentMasjidID:         cp.ClassParentMasjidID,
		ClassParentName:             cp.ClassParentName,
		ClassParentCode:             cp.ClassParentCode,
		ClassParentSlug:             cp.ClassParentSlug,
		ClassParentDescription:      cp.ClassParentDescription,
		ClassParentLevel:            cp.ClassParentLevel,
		ClassParentIsActive:         cp.ClassParentIsActive,
		ClassParentTotalClasses:     cp.ClassParentTotalClasses,
		ClassParentRequirements:     cp.ClassParentRequirements,
		ClassParentImageURL:         cp.ClassParentImageURL,
		ClassParentImageObjectKey:   cp.ClassParentImageObjectKey,
		ClassParentImageURLOld:      cp.ClassParentImageURLOld,
		ClassParentImageObjectKeyOld: cp.ClassParentImageObjectKeyOld,
		ClassParentImageDeletePendingUntil: cp.ClassParentImageDeletePendingUntil,
		ClassParentCreatedAt: cp.ClassParentCreatedAt,
		ClassParentUpdatedAt: cp.ClassParentUpdatedAt,
		ClassParentDeletedAt: deletedAt,
	}
}
// =========================================================
// PATCH REQUEST — tri-state
// =========================================================

type ClassParentPatchRequest struct {
	// NOTE: gunakan PatchFieldClassParent (Value *T)
	ClassParentName         PatchFieldClassParent[string]             `json:"class_parent_name"`          // min=1,max=120
	ClassParentCode         PatchFieldClassParent[*string]            `json:"class_parent_code"`          // null → clear
	ClassParentSlug         PatchFieldClassParent[*string]            `json:"class_parent_slug"`          // null → clear
	ClassParentDescription  PatchFieldClassParent[*string]            `json:"class_parent_description"`   // null → clear
	ClassParentLevel        PatchFieldClassParent[*int16]             `json:"class_parent_level"`         // 0..100 atau null
	ClassParentIsActive     PatchFieldClassParent[*bool]              `json:"class_parent_is_active"`     // bool atau null (abaikan jika nil)
	ClassParentTotalClasses PatchFieldClassParent[*int32]             `json:"class_parent_total_classes"` // >=0 atau null
	ClassParentRequirements PatchFieldClassParent[datatypes.JSONMap]  `json:"class_parent_requirements"`  // null → {}

	// Image (opsional)
	ClassParentImageURL        PatchFieldClassParent[*string] `json:"class_parent_image_url"`
	ClassParentImageObjectKey  PatchFieldClassParent[*string] `json:"class_parent_image_object_key"`

	// Old / delete pending
	ClassParentImageURLOld             PatchFieldClassParent[*string]    `json:"class_parent_image_url_old"`
	ClassParentImageObjectKeyOld       PatchFieldClassParent[*string]    `json:"class_parent_image_object_key_old"`
	ClassParentImageDeletePendingUntil PatchFieldClassParent[*time.Time] `json:"class_parent_image_delete_pending_until"`
}

func (p *ClassParentPatchRequest) Normalize() {
	// trim untuk **string (karena beberapa field bertipe T=*string → Value **string)
	trim := func(s **string, lower bool) {
		if s == nil || *s == nil {
			return
		}
		v := strings.TrimSpace(**s)
		if v == "" {
			*s = nil
			return
		}
		if lower {
			v = strings.ToLower(v)
		}
		*s = &v
	}

	// name: Value *string
	if p.ClassParentName.Present && p.ClassParentName.Value != nil {
		v := strings.TrimSpace(*p.ClassParentName.Value)
		p.ClassParentName.Value = &v // kosong boleh, nanti divalidasi layer atas jika perlu
	}

	// pointer string → **string
	if p.ClassParentCode.Present { trim(p.ClassParentCode.Value, false) }
	if p.ClassParentSlug.Present { trim(p.ClassParentSlug.Value, true) }
	if p.ClassParentDescription.Present { trim(p.ClassParentDescription.Value, false) }
	if p.ClassParentImageURL.Present { trim(p.ClassParentImageURL.Value, false) }
	if p.ClassParentImageObjectKey.Present { trim(p.ClassParentImageObjectKey.Value, false) }
	if p.ClassParentImageURLOld.Present { trim(p.ClassParentImageURLOld.Value, false) }
	if p.ClassParentImageObjectKeyOld.Present { trim(p.ClassParentImageObjectKeyOld.Value, false) }
}

func (p ClassParentPatchRequest) Apply(cp *m.ClassParentModel) {
	// name (*string → value-nya string)
	if p.ClassParentName.Present && p.ClassParentName.Value != nil {
		cp.ClassParentName = *p.ClassParentName.Value
	}

	// code (**string)
	if p.ClassParentCode.Present {
		if p.ClassParentCode.Value == nil {
			cp.ClassParentCode = nil
		} else {
			cp.ClassParentCode = *p.ClassParentCode.Value
		}
	}

	// slug (**string)
	if p.ClassParentSlug.Present {
		if p.ClassParentSlug.Value == nil {
			cp.ClassParentSlug = nil
		} else {
			cp.ClassParentSlug = *p.ClassParentSlug.Value
		}
	}

	// description (**string)
	if p.ClassParentDescription.Present {
		if p.ClassParentDescription.Value == nil {
			cp.ClassParentDescription = nil
		} else {
			cp.ClassParentDescription = *p.ClassParentDescription.Value
		}
	}

	// level (**int16) → field model *int16
	if p.ClassParentLevel.Present {
		if p.ClassParentLevel.Value == nil {
			cp.ClassParentLevel = nil
		} else {
			cp.ClassParentLevel = *p.ClassParentLevel.Value
		}
	}

	// is_active (**bool) → field model bool
	if p.ClassParentIsActive.Present && p.ClassParentIsActive.Value != nil {
		cp.ClassParentIsActive = **p.ClassParentIsActive.Value
	}

	// total_classes (**int32) → field model int32
	if p.ClassParentTotalClasses.Present {
		if p.ClassParentTotalClasses.Value == nil {
			cp.ClassParentTotalClasses = 0
		} else {
			cp.ClassParentTotalClasses = **p.ClassParentTotalClasses.Value
		}
	}

	// requirements (*datatypes.JSONMap)
	if p.ClassParentRequirements.Present {
		if p.ClassParentRequirements.Value == nil {
			cp.ClassParentRequirements = datatypes.JSONMap{}
		} else {
			cp.ClassParentRequirements = *p.ClassParentRequirements.Value
		}
	}

	// image fields (**string)
	if p.ClassParentImageURL.Present {
		if p.ClassParentImageURL.Value == nil {
			cp.ClassParentImageURL = nil
		} else {
			cp.ClassParentImageURL = *p.ClassParentImageURL.Value
		}
	}
	if p.ClassParentImageObjectKey.Present {
		if p.ClassParentImageObjectKey.Value == nil {
			cp.ClassParentImageObjectKey = nil
		} else {
			cp.ClassParentImageObjectKey = *p.ClassParentImageObjectKey.Value
		}
	}

	// old / delete_pending (**string / **time.Time)
	if p.ClassParentImageURLOld.Present {
		if p.ClassParentImageURLOld.Value == nil {
			cp.ClassParentImageURLOld = nil
		} else {
			cp.ClassParentImageURLOld = *p.ClassParentImageURLOld.Value
		}
	}
	if p.ClassParentImageObjectKeyOld.Present {
		if p.ClassParentImageObjectKeyOld.Value == nil {
			cp.ClassParentImageObjectKeyOld = nil
		} else {
			cp.ClassParentImageObjectKeyOld = *p.ClassParentImageObjectKeyOld.Value
		}
	}
	if p.ClassParentImageDeletePendingUntil.Present {
		if p.ClassParentImageDeletePendingUntil.Value == nil {
			cp.ClassParentImageDeletePendingUntil = nil
		} else {
			cp.ClassParentImageDeletePendingUntil = *p.ClassParentImageDeletePendingUntil.Value
		}
	}

	cp.ClassParentUpdatedAt = time.Now()
}


// =========================================================
// LIST QUERY + HELPERS (responses & pagination)
// =========================================================

type ListClassParentQuery struct {
	Limit     int        `query:"limit"`
	Offset    int        `query:"offset"`
	Q         string     `query:"q"`
	Active    *bool      `query:"active"`
	LevelMin  *int16     `query:"level_min"`
	LevelMax  *int16     `query:"level_max"`
	CreatedGt *time.Time `query:"created_gt"`
	CreatedLt *time.Time `query:"created_lt"`
}

// Konversi slice model → slice response
func ToClassParentResponses(rows []m.ClassParentModel) []ClassParentResponse {
	out := make([]ClassParentResponse, 0, len(rows))
	for i := range rows {
		out = append(out, FromModelClassParent(&rows[i]))
	}
	return out
}

// Pagination metadata standar
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Limit      int   `json:"limit"`
	Offset     int   `json:"offset"`
	Count      int   `json:"count"`
	NextOffset *int  `json:"next_offset,omitempty"`
	PrevOffset *int  `json:"prev_offset,omitempty"`
	HasMore    bool  `json:"has_more"`
}

func NewPaginationMeta(total int64, limit, offset, count int) PaginationMeta {
	if limit <= 0 {
		limit = 20
	}
	meta := PaginationMeta{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		Count:   count,
		HasMore: int64(offset+count) < total,
	}
	if offset > 0 {
		prev := offset - limit
		if prev < 0 {
			prev = 0
		}
		meta.PrevOffset = &prev
	}
	if meta.HasMore {
		next := offset + count
		meta.NextOffset = &next
	}
	return meta
}
