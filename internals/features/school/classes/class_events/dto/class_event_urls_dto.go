// file: internals/features/school/sessions/events/dto/class_event_urls_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	m "madinahsalam_backend/internals/features/school/classes/class_events/model"
)

/* =========================================================
   Helpers
   ========================================================= */

// RFC3339 time helpers for delete_pending_until
func parseRFC3339Ptr(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

/* =========================================================
   Patch types (tri-state)
   ========================================================= */

func (p *Patch[T]) UnmarshalJSON(b []byte) error {
	p.Set = true
	return json.Unmarshal(b, &p.Value)
}
func (p Patch[T]) IsSetClassUrls() bool { return p.Set }

func (p *PatchNullable[T]) UnmarshalJSON(b []byte) error {
	p.Set = true
	if string(b) == "null" {
		p.Valid = false
		return nil
	}
	p.Valid = true
	return json.Unmarshal(b, &p.Value)
}
func (p PatchNullable[T]) IsSetClassUrls() bool { return p.Set }

/* =========================================================
   1) REQUESTS
   ========================================================= */

// ---------- CREATE ----------
type CreateClassEventURLRequest struct {
	ClassEventURLEventID uuid.UUID `json:"class_event_url_event_id" validate:"required,uuid"`

	ClassEventURLKind  string  `json:"class_event_url_kind"  validate:"required,max=32"`
	ClassEventURLLabel *string `json:"class_event_url_label" validate:"omitempty,max=160"`

	ClassEventURLURL          *string `json:"class_event_url_url"            validate:"omitempty,max=200000"`
	ClassEventURLObjectKey    *string `json:"class_event_url_object_key"     validate:"omitempty,max=200000"`
	ClassEventURLURLOld       *string `json:"class_event_url_url_old"        validate:"omitempty,max=200000"`
	ClassEventURLObjectKeyOld *string `json:"class_event_url_object_key_old" validate:"omitempty,max=200000"`

	// RFC3339 (contoh: 2025-09-23T10:15:00Z)
	ClassEventURLDeletePendingUntil *string `json:"class_event_url_delete_pending_until" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`

	ClassEventURLIsPrimary *bool `json:"class_event_url_is_primary" validate:"omitempty"`
}

var (
	ErrKindEmpty = errors.New("class_event_url_kind cannot be empty")
)

func (r CreateClassEventURLRequest) ToModel(schoolID uuid.UUID) (m.ClassEventURLModel, error) {
	kind := strings.TrimSpace(r.ClassEventURLKind)
	if kind == "" {
		return m.ClassEventURLModel{}, ErrKindEmpty
	}

	delUntil, err := parseRFC3339Ptr(r.ClassEventURLDeletePendingUntil)
	if err != nil {
		return m.ClassEventURLModel{}, err
	}

	model := m.ClassEventURLModel{
		ClassEventURLSchoolID:           schoolID,
		ClassEventURLEventID:            r.ClassEventURLEventID,
		ClassEventURLKind:               kind,
		ClassEventURLLabel:              trimPtr(r.ClassEventURLLabel),
		ClassEventURLURL:                trimPtr(r.ClassEventURLURL),
		ClassEventURLObjectKey:          trimPtr(r.ClassEventURLObjectKey),
		ClassEventURLURLOld:             trimPtr(r.ClassEventURLURLOld),
		ClassEventURLObjectKeyOld:       trimPtr(r.ClassEventURLObjectKeyOld),
		ClassEventURLDeletePendingUntil: delUntil,
		ClassEventURLIsPrimary:          false, // default
	}
	if r.ClassEventURLIsPrimary != nil {
		model.ClassEventURLIsPrimary = *r.ClassEventURLIsPrimary
	}
	return model, nil
}

// ---------- PATCH ----------
type PatchClassEventURLRequest struct {
	ClassEventURLKind  Patch[string]         `json:"class_event_url_kind"`
	ClassEventURLLabel PatchNullable[string] `json:"class_event_url_label"`

	ClassEventURLURL          PatchNullable[string] `json:"class_event_url_url"`
	ClassEventURLObjectKey    PatchNullable[string] `json:"class_event_url_object_key"`
	ClassEventURLURLOld       PatchNullable[string] `json:"class_event_url_url_old"`
	ClassEventURLObjectKeyOld PatchNullable[string] `json:"class_event_url_object_key_old"`

	// RFC3339; null untuk hapus jadwal purge
	ClassEventURLDeletePendingUntil PatchNullable[string] `json:"class_event_url_delete_pending_until"`

	ClassEventURLIsPrimary Patch[bool] `json:"class_event_url_is_primary"`
}

func (p *PatchClassEventURLRequest) Apply(u *m.ClassEventURLModel) error {
	// kind (non-null)
	if p.ClassEventURLKind.IsSetClassUrls() {
		k := strings.TrimSpace(p.ClassEventURLKind.Value)
		if k == "" {
			return ErrKindEmpty
		}
		u.ClassEventURLKind = k
	}

	// label
	if p.ClassEventURLLabel.IsSetClassUrls() {
		if p.ClassEventURLLabel.Valid {
			u.ClassEventURLLabel = trimPtr(&p.ClassEventURLLabel.Value)
		} else {
			u.ClassEventURLLabel = nil
		}
	}

	// storage fields
	if p.ClassEventURLURL.IsSetClassUrls() {
		if p.ClassEventURLURL.Valid {
			u.ClassEventURLURL = trimPtr(&p.ClassEventURLURL.Value)
		} else {
			u.ClassEventURLURL = nil
		}
	}
	if p.ClassEventURLObjectKey.IsSetClassUrls() {
		if p.ClassEventURLObjectKey.Valid {
			u.ClassEventURLObjectKey = trimPtr(&p.ClassEventURLObjectKey.Value)
		} else {
			u.ClassEventURLObjectKey = nil
		}
	}
	if p.ClassEventURLURLOld.IsSetClassUrls() {
		if p.ClassEventURLURLOld.Valid {
			u.ClassEventURLURLOld = trimPtr(&p.ClassEventURLURLOld.Value)
		} else {
			u.ClassEventURLURLOld = nil
		}
	}
	if p.ClassEventURLObjectKeyOld.IsSetClassUrls() {
		if p.ClassEventURLObjectKeyOld.Valid {
			u.ClassEventURLObjectKeyOld = trimPtr(&p.ClassEventURLObjectKeyOld.Value)
		} else {
			u.ClassEventURLObjectKeyOld = nil
		}
	}

	// delete_pending_until
	if p.ClassEventURLDeletePendingUntil.IsSetClassUrls() {
		if p.ClassEventURLDeletePendingUntil.Valid {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(p.ClassEventURLDeletePendingUntil.Value))
			if err != nil {
				return err
			}
			u.ClassEventURLDeletePendingUntil = &t
		} else {
			u.ClassEventURLDeletePendingUntil = nil
		}
	}

	// primary flag
	if p.ClassEventURLIsPrimary.IsSetClassUrls() {
		u.ClassEventURLIsPrimary = p.ClassEventURLIsPrimary.Value
	}

	return nil
}

/* =========================================================
   2) LIST QUERY
   ========================================================= */

type ListClassEventURLQuery struct {
	EventID   *uuid.UUID `query:"event_id"   validate:"omitempty,uuid"`
	Kind      *string    `query:"kind"       validate:"omitempty,max=32"`
	IsPrimary *bool      `query:"is_primary" validate:"omitempty"`

	Q *string `query:"q" validate:"omitempty,max=160"` // cari di label

	// sort: created_at_desc (default)
	// options: created_at_asc|created_at_desc|updated_at_asc|updated_at_desc|kind_asc|kind_desc|primary_desc
	Sort *string `query:"sort" validate:"omitempty,oneof=created_at_asc created_at_desc updated_at_asc updated_at_desc kind_asc kind_desc primary_desc"`

	Limit  int `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int `query:"offset" validate:"omitempty,min=0"`
}

func (q *ListClassEventURLQuery) Normalize() {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Kind != nil {
		k := strings.TrimSpace(*q.Kind)
		if k == "" {
			q.Kind = nil
		} else {
			q.Kind = &k
		}
	}
	if q.Q != nil {
		v := strings.TrimSpace(*q.Q)
		if v == "" {
			q.Q = nil
		} else {
			q.Q = &v
		}
	}
}

/* =========================================================
   3) RESPONSES
   ========================================================= */

type ClassEventURLResponse struct {
	ClassEventURLID       uuid.UUID `json:"class_event_url_id"`
	ClassEventURLSchoolID uuid.UUID `json:"class_event_url_school_id"`
	ClassEventURLEventID  uuid.UUID `json:"class_event_url_event_id"`

	ClassEventURLKind  string  `json:"class_event_url_kind"`
	ClassEventURLLabel *string `json:"class_event_url_label,omitempty"`

	ClassEventURLURL                *string    `json:"class_event_url_url,omitempty"`
	ClassEventURLObjectKey          *string    `json:"class_event_url_object_key,omitempty"`
	ClassEventURLURLOld             *string    `json:"class_event_url_url_old,omitempty"`
	ClassEventURLObjectKeyOld       *string    `json:"class_event_url_object_key_old,omitempty"`
	ClassEventURLDeletePendingUntil *time.Time `json:"class_event_url_delete_pending_until,omitempty"`

	ClassEventURLIsPrimary bool `json:"class_event_url_is_primary"`

	ClassEventURLCreatedAt time.Time  `json:"class_event_url_created_at"`
	ClassEventURLUpdatedAt time.Time  `json:"class_event_url_updated_at"`
	ClassEventURLDeletedAt *time.Time `json:"class_event_url_deleted_at,omitempty"`
}

type ClassEventURLListResponse struct {
	Items      []ClassEventURLResponse `json:"items"`
	Pagination struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Total  int `json:"total"`
	} `json:"pagination"`
}

/* =========================================================
   4) MAPPERS
   ========================================================= */

func FromModelClassEventURL(u m.ClassEventURLModel) ClassEventURLResponse {
	var delAt *time.Time
	if u.ClassEventURLDeletedAt.Valid {
		t := u.ClassEventURLDeletedAt.Time
		delAt = &t
	}
	return ClassEventURLResponse{
		ClassEventURLID:                 u.ClassEventURLID,
		ClassEventURLSchoolID:           u.ClassEventURLSchoolID,
		ClassEventURLEventID:            u.ClassEventURLEventID,
		ClassEventURLKind:               u.ClassEventURLKind,
		ClassEventURLLabel:              u.ClassEventURLLabel,
		ClassEventURLURL:                u.ClassEventURLURL,
		ClassEventURLObjectKey:          u.ClassEventURLObjectKey,
		ClassEventURLURLOld:             u.ClassEventURLURLOld,
		ClassEventURLObjectKeyOld:       u.ClassEventURLObjectKeyOld,
		ClassEventURLDeletePendingUntil: u.ClassEventURLDeletePendingUntil,
		ClassEventURLIsPrimary:          u.ClassEventURLIsPrimary,
		ClassEventURLCreatedAt:          u.ClassEventURLCreatedAt,
		ClassEventURLUpdatedAt:          u.ClassEventURLUpdatedAt,
		ClassEventURLDeletedAt:          delAt,
	}
}

func FromModelsClassEventURL(list []m.ClassEventURLModel) []ClassEventURLResponse {
	out := make([]ClassEventURLResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModelClassEventURL(list[i]))
	}
	return out
}
