// file: internals/features/school/posts/themes/dto/post_theme_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/others/post/model"
)

/* =========================================================
   Helpers
========================================================= */

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

/*
Tri-state field for PATCH:
- Absent : not updated
- null   : set column to NULL
- value  : set to value
*/
type UpdateField[T any] struct {
	set   bool
	null  bool
	value T
}

func (f *UpdateField[T]) UnmarshalJSON(b []byte) error {
	f.set = true
	if string(b) == "null" {
		f.null = true
		var zero T
		f.value = zero
		return nil
	}
	return json.Unmarshal(b, &f.value)
}

func (f UpdateField[T]) ShouldUpdate() bool { return f.set }
func (f UpdateField[T]) IsNull() bool       { return f.set && f.null }
func (f UpdateField[T]) Val() T             { return f.value }

func parseRFC3339Ptr(s string) (*time.Time, error) {
	ss := strings.TrimSpace(s)
	if ss == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, ss)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

/* =========================================================
   CREATE
========================================================= */

type CreatePostThemeRequest struct {
	PostThemeMasjidID uuid.UUID `json:"post_theme_masjid_id" validate:"required"`

	PostThemeKind     string     `json:"post_theme_kind" validate:"required,oneof=announcement material post other"`
	PostThemeParentID *uuid.UUID `json:"post_theme_parent_id" validate:"omitempty,uuid"`

	PostThemeName string `json:"post_theme_name" validate:"required,max=80"`
	PostThemeSlug string `json:"post_theme_slug" validate:"required,max=120"`

	PostThemeColor       *string `json:"post_theme_color" validate:"omitempty,max=20"`
	PostThemeCustomColor *string `json:"post_theme_custom_color" validate:"omitempty,max=20"`
	PostThemeDescription *string `json:"post_theme_description" validate:"omitempty"`

	PostThemeIsActive *bool `json:"post_theme_is_active" validate:"omitempty"`

	// Icon fields (optional)
	PostThemeIconURL                *string `json:"post_theme_icon_url" validate:"omitempty"`
	PostThemeIconObjectKey          *string `json:"post_theme_icon_object_key" validate:"omitempty"`
	PostThemeIconURLOld             *string `json:"post_theme_icon_url_old" validate:"omitempty"`
	PostThemeIconObjectKeyOld       *string `json:"post_theme_icon_object_key_old" validate:"omitempty"`
	PostThemeIconDeletePendingUntil *string `json:"post_theme_icon_delete_pending_until" validate:"omitempty"` // RFC3339
}

var (
	ErrInvalidKind        = errors.New("invalid post_theme_kind (use announcement|material|post|other)")
	ErrInvalidDeleteUntil = errors.New("invalid post_theme_icon_delete_pending_until (use RFC3339)")
)

func (r *CreatePostThemeRequest) ToModel() (*model.PostThemeModel, error) {
	kind := model.PostThemeKind(strings.ToLower(strings.TrimSpace(r.PostThemeKind)))
	if !kind.Valid() {
		return nil, ErrInvalidKind
	}

	isActive := true
	if r.PostThemeIsActive != nil {
		isActive = *r.PostThemeIsActive
	}

	var delUntil *time.Time
	if r.PostThemeIconDeletePendingUntil != nil && strings.TrimSpace(*r.PostThemeIconDeletePendingUntil) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*r.PostThemeIconDeletePendingUntil))
		if err != nil {
			return nil, ErrInvalidDeleteUntil
		}
		delUntil = &t
	}

	m := &model.PostThemeModel{
		PostThemeMasjidID: r.PostThemeMasjidID,

		PostThemeKind:     kind,
		PostThemeParentID: r.PostThemeParentID,

		PostThemeName: strings.TrimSpace(r.PostThemeName),
		PostThemeSlug: strings.TrimSpace(r.PostThemeSlug),

		PostThemeColor:       trimPtr(r.PostThemeColor),
		PostThemeCustomColor: trimPtr(r.PostThemeCustomColor),
		PostThemeDescription: trimPtr(r.PostThemeDescription),

		PostThemeIsActive: isActive,

		PostThemeIconURL:                trimPtr(r.PostThemeIconURL),
		PostThemeIconObjectKey:          trimPtr(r.PostThemeIconObjectKey),
		PostThemeIconURLOld:             trimPtr(r.PostThemeIconURLOld),
		PostThemeIconObjectKeyOld:       trimPtr(r.PostThemeIconObjectKeyOld),
		PostThemeIconDeletePendingUntil: delUntil,
	}
	return m, nil
}

/* =========================================================
   PATCH
========================================================= */

type PatchPostThemeRequest struct {
	PostThemeParentID UpdateField[uuid.UUID] `json:"post_theme_parent_id"`

	PostThemeKind UpdateField[string] `json:"post_theme_kind"` // announcement|material|post|other

	PostThemeName UpdateField[string] `json:"post_theme_name"`
	PostThemeSlug UpdateField[string] `json:"post_theme_slug"`

	PostThemeColor       UpdateField[string] `json:"post_theme_color"`
	PostThemeCustomColor UpdateField[string] `json:"post_theme_custom_color"`
	PostThemeDescription UpdateField[string] `json:"post_theme_description"`

	PostThemeIsActive UpdateField[bool] `json:"post_theme_is_active"`

	PostThemeIconURL                UpdateField[string] `json:"post_theme_icon_url"`
	PostThemeIconObjectKey          UpdateField[string] `json:"post_theme_icon_object_key"`
	PostThemeIconURLOld             UpdateField[string] `json:"post_theme_icon_url_old"`
	PostThemeIconObjectKeyOld       UpdateField[string] `json:"post_theme_icon_object_key_old"`
	PostThemeIconDeletePendingUntil UpdateField[string] `json:"post_theme_icon_delete_pending_until"` // RFC3339 or null
}

func (p *PatchPostThemeRequest) ApplyToModel(m *model.PostThemeModel) error {
	// parent
	if p.PostThemeParentID.ShouldUpdate() {
		if p.PostThemeParentID.IsNull() {
			m.PostThemeParentID = nil
		} else {
			id := p.PostThemeParentID.Val()
			m.PostThemeParentID = &id
		}
	}

	// kind
	if p.PostThemeKind.ShouldUpdate() && !p.PostThemeKind.IsNull() {
		k := model.PostThemeKind(strings.ToLower(strings.TrimSpace(p.PostThemeKind.Val())))
		if !k.Valid() {
			return ErrInvalidKind
		}
		m.PostThemeKind = k
	}

	// name
	if p.PostThemeName.ShouldUpdate() {
		if p.PostThemeName.IsNull() {
			// name NOT NULL di DB → tolak null
			return errors.New("post_theme_name cannot be null")
		}
		m.PostThemeName = strings.TrimSpace(p.PostThemeName.Val())
	}

	// slug
	if p.PostThemeSlug.ShouldUpdate() {
		if p.PostThemeSlug.IsNull() {
			return errors.New("post_theme_slug cannot be null")
		}
		m.PostThemeSlug = strings.TrimSpace(p.PostThemeSlug.Val())
	}

	// color
	if p.PostThemeColor.ShouldUpdate() {
		if p.PostThemeColor.IsNull() {
			m.PostThemeColor = nil
		} else {
			m.PostThemeColor = trimPtr(ptr(p.PostThemeColor.Val()))
		}
	}
	if p.PostThemeCustomColor.ShouldUpdate() {
		if p.PostThemeCustomColor.IsNull() {
			m.PostThemeCustomColor = nil
		} else {
			m.PostThemeCustomColor = trimPtr(ptr(p.PostThemeCustomColor.Val()))
		}
	}
	if p.PostThemeDescription.ShouldUpdate() {
		if p.PostThemeDescription.IsNull() {
			m.PostThemeDescription = nil
		} else {
			m.PostThemeDescription = trimPtr(ptr(p.PostThemeDescription.Val()))
		}
	}

	// is_active (bool, NOT NULL)
	if p.PostThemeIsActive.ShouldUpdate() && !p.PostThemeIsActive.IsNull() {
		m.PostThemeIsActive = p.PostThemeIsActive.Val()
	}

	// icons
	if p.PostThemeIconURL.ShouldUpdate() {
		if p.PostThemeIconURL.IsNull() {
			m.PostThemeIconURL = nil
		} else {
			m.PostThemeIconURL = trimPtr(ptr(p.PostThemeIconURL.Val()))
		}
	}
	if p.PostThemeIconObjectKey.ShouldUpdate() {
		if p.PostThemeIconObjectKey.IsNull() {
			m.PostThemeIconObjectKey = nil
		} else {
			m.PostThemeIconObjectKey = trimPtr(ptr(p.PostThemeIconObjectKey.Val()))
		}
	}
	if p.PostThemeIconURLOld.ShouldUpdate() {
		if p.PostThemeIconURLOld.IsNull() {
			m.PostThemeIconURLOld = nil
		} else {
			m.PostThemeIconURLOld = trimPtr(ptr(p.PostThemeIconURLOld.Val()))
		}
	}
	if p.PostThemeIconObjectKeyOld.ShouldUpdate() {
		if p.PostThemeIconObjectKeyOld.IsNull() {
			m.PostThemeIconObjectKeyOld = nil
		} else {
			m.PostThemeIconObjectKeyOld = trimPtr(ptr(p.PostThemeIconObjectKeyOld.Val()))
		}
	}
	if p.PostThemeIconDeletePendingUntil.ShouldUpdate() {
		if p.PostThemeIconDeletePendingUntil.IsNull() {
			m.PostThemeIconDeletePendingUntil = nil
		} else {
			tp, err := parseRFC3339Ptr(p.PostThemeIconDeletePendingUntil.Val())
			if err != nil {
				return ErrInvalidDeleteUntil
			}
			m.PostThemeIconDeletePendingUntil = tp
		}
	}

	return nil
}

func ptr[T any](v T) *T { return &v }

/* =========================================================
   LIST QUERY
========================================================= */

type ListPostThemesQuery struct {
	// filters
	MasjidID *uuid.UUID `query:"masjid_id" validate:"omitempty,uuid"`
	Kind     *string    `query:"kind"      validate:"omitempty,oneof=announcement material post other"`
	ParentID *uuid.UUID `query:"parent_id" validate:"omitempty,uuid"`
	IsActive *bool      `query:"is_active" validate:"omitempty"`

	// search
	SearchName *string `query:"q" validate:"omitempty,max=120"`

	// pagination + ordering (simple style)
	Limit   int    `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset  int    `query:"offset" validate:"omitempty,min=0"`
	OrderBy string `query:"order_by" validate:"omitempty,oneof=name_asc name_desc slug_asc slug_desc created_at_asc created_at_desc updated_at_asc updated_at_desc"`
}

func (q *ListPostThemesQuery) Normalize() {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.SearchName != nil {
		v := strings.TrimSpace(*q.SearchName)
		if v == "" {
			q.SearchName = nil
		} else {
			q.SearchName = &v
		}
	}
	if q.Kind != nil {
		v := strings.ToLower(strings.TrimSpace(*q.Kind))
		if v == "" {
			q.Kind = nil
		} else {
			q.Kind = &v
		}
	}
	if q.OrderBy == "" {
		q.OrderBy = "created_at_desc"
	}
}

func (q *ListPostThemesQuery) OrderExpr() string {
	switch strings.ToLower(strings.TrimSpace(q.OrderBy)) {
	case "name_asc":
		return "post_theme_name ASC"
	case "name_desc":
		return "post_theme_name DESC"
	case "slug_asc":
		return "post_theme_slug ASC"
	case "slug_desc":
		return "post_theme_slug DESC"
	case "created_at_asc":
		return "post_theme_created_at ASC"
	case "created_at_desc":
		return "post_theme_created_at DESC"
	case "updated_at_asc":
		return "post_theme_updated_at ASC"
	case "updated_at_desc":
		return "post_theme_updated_at DESC"
	default:
		return "post_theme_created_at DESC"
	}
}

/* =========================================================
   RESPONSES
========================================================= */

type PostThemeResponse struct {
	PostThemeID       uuid.UUID `json:"post_theme_id"`
	PostThemeMasjidID uuid.UUID `json:"post_theme_masjid_id"`

	PostThemeKind     string     `json:"post_theme_kind"`
	PostThemeParentID *uuid.UUID `json:"post_theme_parent_id,omitempty"`

	PostThemeName string `json:"post_theme_name"`
	PostThemeSlug string `json:"post_theme_slug"`

	PostThemeColor       *string `json:"post_theme_color,omitempty"`
	PostThemeCustomColor *string `json:"post_theme_custom_color,omitempty"`
	PostThemeDescription *string `json:"post_theme_description,omitempty"`

	PostThemeIsActive bool `json:"post_theme_is_active"`

	PostThemeIconURL                *string    `json:"post_theme_icon_url,omitempty"`
	PostThemeIconObjectKey          *string    `json:"post_theme_icon_object_key,omitempty"`
	PostThemeIconURLOld             *string    `json:"post_theme_icon_url_old,omitempty"`
	PostThemeIconObjectKeyOld       *string    `json:"post_theme_icon_object_key_old,omitempty"`
	PostThemeIconDeletePendingUntil *time.Time `json:"post_theme_icon_delete_pending_until,omitempty"`

	PostThemeCreatedAt time.Time  `json:"post_theme_created_at"`
	PostThemeUpdatedAt time.Time  `json:"post_theme_updated_at"`
	PostThemeDeletedAt *time.Time `json:"post_theme_deleted_at,omitempty"`
}

func FromModel(m *model.PostThemeModel) PostThemeResponse {
	var delAt *time.Time
	if m.PostThemeDeletedAt.Valid {
		t := m.PostThemeDeletedAt.Time
		delAt = &t
	}
	return PostThemeResponse{
		PostThemeID:       m.PostThemeID,
		PostThemeMasjidID: m.PostThemeMasjidID,

		PostThemeKind:     string(m.PostThemeKind),
		PostThemeParentID: m.PostThemeParentID,

		PostThemeName: m.PostThemeName,
		PostThemeSlug: m.PostThemeSlug,

		PostThemeColor:       m.PostThemeColor,
		PostThemeCustomColor: m.PostThemeCustomColor,
		PostThemeDescription: m.PostThemeDescription,

		PostThemeIsActive: m.PostThemeIsActive,

		PostThemeIconURL:                m.PostThemeIconURL,
		PostThemeIconObjectKey:          m.PostThemeIconObjectKey,
		PostThemeIconURLOld:             m.PostThemeIconURLOld,
		PostThemeIconObjectKeyOld:       m.PostThemeIconObjectKeyOld,
		PostThemeIconDeletePendingUntil: m.PostThemeIconDeletePendingUntil,

		PostThemeCreatedAt: m.PostThemeCreatedAt,
		PostThemeUpdatedAt: m.PostThemeUpdatedAt,
		PostThemeDeletedAt: delAt,
	}
}

func FromModels(list []model.PostThemeModel) []PostThemeResponse {
	out := make([]PostThemeResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModel(&list[i]))
	}
	return out
}
