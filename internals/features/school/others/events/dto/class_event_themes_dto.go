// file: internals/features/school/events/themes/dto/class_event_theme_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/others/events/model"
)

/* =========================================================
   Shared helpers
   ========================================================= */

var (
	reHexFull = regexp.MustCompile(`^(?i)#?[0-9a-f]{6}$`) // #RRGGBB
)

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

func normalizeHexPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	// allow "FFAABB" or "#FFAABB" → always normalize to "#FFAABB"
	if !reHexFull.MatchString(v) {
		return &v // biar validasi yang nolak; jangan paksa ubah
	}
	if !strings.HasPrefix(v, "#") {
		v = "#" + v
	}
	return &v
}

/* =========================================================
   PatchField (tri-state): absent | null | value
   ========================================================= */

type PatchField[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchField[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
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

func (p PatchField[T]) Get() (*T, bool) { return p.Value, p.Present }

/* =========================================================
   Requests: CREATE
   ========================================================= */

type CreateClassEventThemeRequest struct {
	// MasjidID dipaksa dari controller/path; tidak dari body
	ClassEventThemesCode        string  `json:"class_event_themes_code" validate:"required,max=64"`
	ClassEventThemesName        string  `json:"class_event_themes_name" validate:"required,max=120"`
	ClassEventThemesColor       *string `json:"class_event_themes_color" validate:"omitempty,max=32"`
	ClassEventThemesCustomColor *string `json:"class_event_themes_custom_color" validate:"omitempty,max=16"` // ex: "#FFAABB"
	ClassEventThemesIsActive    *bool   `json:"class_event_themes_is_active" validate:"omitempty"`
}

func (r *CreateClassEventThemeRequest) Normalize() {
	r.ClassEventThemesCode = strings.TrimSpace(r.ClassEventThemesCode)
	r.ClassEventThemesName = strings.TrimSpace(r.ClassEventThemesName)
	r.ClassEventThemesColor = trimPtr(r.ClassEventThemesColor)
	r.ClassEventThemesCustomColor = normalizeHexPtr(r.ClassEventThemesCustomColor)
}

func (r *CreateClassEventThemeRequest) Validate(v *validator.Validate) error {
	if err := v.Struct(r); err != nil {
		return err
	}
	// Custom rules
	if r.ClassEventThemesCustomColor != nil && !reHexFull.MatchString(*r.ClassEventThemesCustomColor) {
		return errors.New("class_event_themes_custom_color must be hex like #RRGGBB")
	}
	return nil
}

func (r *CreateClassEventThemeRequest) ToModel(masjidID uuid.UUID) *model.ClassEventTheme {
	isActive := true
	if r.ClassEventThemesIsActive != nil {
		isActive = *r.ClassEventThemesIsActive
	}
	return &model.ClassEventTheme{
		// kolom explicit sesuai model
		ClassEventThemesMasjidID:    masjidID,
		ClassEventThemesCode:        r.ClassEventThemesCode,
		ClassEventThemesName:        r.ClassEventThemesName,
		ClassEventThemesColor:       r.ClassEventThemesColor,
		ClassEventThemesCustomColor: r.ClassEventThemesCustomColor,
		ClassEventThemesIsActive:    isActive,
	}
}

/* =========================================================
   Requests: PATCH (partial)
   ========================================================= */

type PatchClassEventThemeRequest struct {
	ClassEventThemesCode        PatchField[string] `json:"class_event_themes_code"`
	ClassEventThemesName        PatchField[string] `json:"class_event_themes_name"`
	ClassEventThemesColor       PatchField[string] `json:"class_event_themes_color"`
	ClassEventThemesCustomColor PatchField[string] `json:"class_event_themes_custom_color"`
	ClassEventThemesIsActive    PatchField[bool]   `json:"class_event_themes_is_active"`
}

func (p *PatchClassEventThemeRequest) Normalize() {
	if p.ClassEventThemesCode.Present && p.ClassEventThemesCode.Value != nil {
		v := strings.TrimSpace(*p.ClassEventThemesCode.Value)
		p.ClassEventThemesCode.Value = &v
	}
	if p.ClassEventThemesName.Present && p.ClassEventThemesName.Value != nil {
		v := strings.TrimSpace(*p.ClassEventThemesName.Value)
		p.ClassEventThemesName.Value = &v
	}
	if p.ClassEventThemesColor.Present {
		p.ClassEventThemesColor.Value = trimPtr(p.ClassEventThemesColor.Value)
	}
	if p.ClassEventThemesCustomColor.Present {
		p.ClassEventThemesCustomColor.Value = normalizeHexPtr(p.ClassEventThemesCustomColor.Value)
	}
}

// ValidatePartial: validasi basic utk field yang dikirim saja
func (p *PatchClassEventThemeRequest) ValidatePartial(v *validator.Validate) error {
	// manual per-field agar hanya memvalidasi yang Present
	if p.ClassEventThemesCode.Present && p.ClassEventThemesCode.Value != nil && len(*p.ClassEventThemesCode.Value) > 64 {
		return errors.New("class_event_themes_code max 64 characters")
	}
	if p.ClassEventThemesName.Present && p.ClassEventThemesName.Value != nil && len(*p.ClassEventThemesName.Value) > 120 {
		return errors.New("class_event_themes_name max 120 characters")
	}
	if p.ClassEventThemesColor.Present && p.ClassEventThemesColor.Value != nil && len(*p.ClassEventThemesColor.Value) > 32 {
		return errors.New("class_event_themes_color max 32 characters")
	}
	if p.ClassEventThemesCustomColor.Present && p.ClassEventThemesCustomColor.Value != nil {
		if len(*p.ClassEventThemesCustomColor.Value) > 16 {
			return errors.New("class_event_themes_custom_color max 16 characters")
		}
		if !reHexFull.MatchString(*p.ClassEventThemesCustomColor.Value) {
			return errors.New("class_event_themes_custom_color must be hex like #RRGGBB")
		}
	}
	return nil
}

// ApplyPatch: ubah model in-place sesuai field yang Present
func (p *PatchClassEventThemeRequest) ApplyPatch(m *model.ClassEventTheme) {
	if val, ok := p.ClassEventThemesCode.Get(); ok {
		// code NOT NULL → abaikan nil
		if val != nil {
			m.ClassEventThemesCode = *val
		}
	}
	if val, ok := p.ClassEventThemesName.Get(); ok {
		if val != nil {
			m.ClassEventThemesName = *val
		}
	}
	if val, ok := p.ClassEventThemesColor.Get(); ok {
		// nullable → boleh nil (clear)
		m.ClassEventThemesColor = val
	}
	if val, ok := p.ClassEventThemesCustomColor.Get(); ok {
		// nullable → boleh nil (clear)
		m.ClassEventThemesCustomColor = val
	}
	if val, ok := p.ClassEventThemesIsActive.Get(); ok && val != nil {
		m.ClassEventThemesIsActive = *val
	}
}

/* =========================================================
   Query (list): filter/sort/paging
   ========================================================= */

type ListClassEventThemeQuery struct {
	SearchName *string `query:"q"`         // ILIKE name
	IsActive   *bool   `query:"is_active"` // true/false
	Limit      int     `query:"limit"`     // default di controller
	Offset     int     `query:"offset"`
	OrderBy    string  `query:"order_by"` // "name_asc" | "name_desc" | "created_desc" | "created_asc"
}

func (q *ListClassEventThemeQuery) Normalize() {
	if q.SearchName != nil {
		q.SearchName = trimPtr(q.SearchName)
	}
	if q.OrderBy == "" {
		q.OrderBy = "name_asc"
	}
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
}

func (q *ListClassEventThemeQuery) OrderExpr() string {
	switch strings.ToLower(strings.TrimSpace(q.OrderBy)) {
	case "name_desc":
		return "class_event_themes_name DESC"
	case "created_asc":
		return "class_event_themes_created_at ASC"
	case "created_desc":
		return "class_event_themes_created_at DESC"
	case "name_asc":
		fallthrough
	default:
		return "class_event_themes_name ASC"
	}
}

/* =========================================================
   Response DTO
   ========================================================= */

type ClassEventThemeResponse struct {
	ClassEventThemesID          uuid.UUID `json:"class_event_themes_id"`
	ClassEventThemesMasjidID    uuid.UUID `json:"class_event_themes_masjid_id"`
	ClassEventThemesCode        string    `json:"class_event_themes_code"`
	ClassEventThemesName        string    `json:"class_event_themes_name"`
	ClassEventThemesColor       *string   `json:"class_event_themes_color,omitempty"`
	ClassEventThemesCustomColor *string   `json:"class_event_themes_custom_color,omitempty"`
	ClassEventThemesIsActive    bool      `json:"class_event_themes_is_active"`

	ClassEventThemesCreatedAt string `json:"class_event_themes_created_at"`
	ClassEventThemesUpdatedAt string `json:"class_event_themes_updated_at"`
}

func FromModel(m *model.ClassEventTheme) ClassEventThemeResponse {
	return ClassEventThemeResponse{
		ClassEventThemesID:          m.ClassEventThemesID,
		ClassEventThemesMasjidID:    m.ClassEventThemesMasjidID,
		ClassEventThemesCode:        m.ClassEventThemesCode,
		ClassEventThemesName:        m.ClassEventThemesName,
		ClassEventThemesColor:       m.ClassEventThemesColor,
		ClassEventThemesCustomColor: m.ClassEventThemesCustomColor,
		ClassEventThemesIsActive:    m.ClassEventThemesIsActive,
		ClassEventThemesCreatedAt:   m.ClassEventThemesCreatedAt.Format(time.RFC3339),
		ClassEventThemesUpdatedAt:   m.ClassEventThemesUpdatedAt.Format(time.RFC3339),
	}
}

func FromModels(list []model.ClassEventTheme) []ClassEventThemeResponse {
	out := make([]ClassEventThemeResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModel(&list[i]))
	}
	return out
}
