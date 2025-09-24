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

	model "masjidku_backend/internals/features/school/classes/class_events/model"
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
	// allow "FFAABB" or "#FFAABB" → normalize to "#FFAABB" jika valid
	if !reHexFull.MatchString(v) {
		return &v // biarkan validator yang menolak jika format salah
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
	ClassEventThemeCode        string  `json:"class_event_theme_code" validate:"required,max=64"`
	ClassEventThemeName        string  `json:"class_event_theme_name" validate:"required,max=120"`
	ClassEventThemeColor       *string `json:"class_event_theme_color" validate:"omitempty,max=32"`
	ClassEventThemeCustomColor *string `json:"class_event_theme_custom_color" validate:"omitempty,max=16"` // ex: "#FFAABB"
	ClassEventThemeIsActive    *bool   `json:"class_event_theme_is_active" validate:"omitempty"`
}

func (r *CreateClassEventThemeRequest) Normalize() {
	r.ClassEventThemeCode = strings.TrimSpace(r.ClassEventThemeCode)
	r.ClassEventThemeName = strings.TrimSpace(r.ClassEventThemeName)
	r.ClassEventThemeColor = trimPtr(r.ClassEventThemeColor)
	r.ClassEventThemeCustomColor = normalizeHexPtr(r.ClassEventThemeCustomColor)
}

func (r *CreateClassEventThemeRequest) Validate(v *validator.Validate) error {
	if err := v.Struct(r); err != nil {
		return err
	}
	// Custom rule untuk hex
	if r.ClassEventThemeCustomColor != nil && !reHexFull.MatchString(*r.ClassEventThemeCustomColor) {
		return errors.New("class_event_theme_custom_color must be hex like #RRGGBB")
	}
	return nil
}

func (r *CreateClassEventThemeRequest) ToModel(masjidID uuid.UUID) *model.ClassEventThemeModel {
	isActive := true
	if r.ClassEventThemeIsActive != nil {
		isActive = *r.ClassEventThemeIsActive
	}
	return &model.ClassEventThemeModel{
		ClassEventThemeMasjidID:    masjidID,
		ClassEventThemeCode:        r.ClassEventThemeCode,
		ClassEventThemeName:        r.ClassEventThemeName,
		ClassEventThemeColor:       r.ClassEventThemeColor,
		ClassEventThemeCustomColor: r.ClassEventThemeCustomColor,
		ClassEventThemeIsActive:    isActive,
	}
}

/* =========================================================
   Requests: PATCH (partial)
   ========================================================= */

type PatchClassEventThemeRequest struct {
	ClassEventThemeCode        PatchField[string] `json:"class_event_theme_code"`
	ClassEventThemeName        PatchField[string] `json:"class_event_theme_name"`
	ClassEventThemeColor       PatchField[string] `json:"class_event_theme_color"`
	ClassEventThemeCustomColor PatchField[string] `json:"class_event_theme_custom_color"`
	ClassEventThemeIsActive    PatchField[bool]   `json:"class_event_theme_is_active"`
}

func (p *PatchClassEventThemeRequest) Normalize() {
	if p.ClassEventThemeCode.Present && p.ClassEventThemeCode.Value != nil {
		v := strings.TrimSpace(*p.ClassEventThemeCode.Value)
		p.ClassEventThemeCode.Value = &v
	}
	if p.ClassEventThemeName.Present && p.ClassEventThemeName.Value != nil {
		v := strings.TrimSpace(*p.ClassEventThemeName.Value)
		p.ClassEventThemeName.Value = &v
	}
	if p.ClassEventThemeColor.Present {
		p.ClassEventThemeColor.Value = trimPtr(p.ClassEventThemeColor.Value)
	}
	if p.ClassEventThemeCustomColor.Present {
		p.ClassEventThemeCustomColor.Value = normalizeHexPtr(p.ClassEventThemeCustomColor.Value)
	}
}

// ValidatePartial: validasi hanya field yang dikirim
func (p *PatchClassEventThemeRequest) ValidatePartial(_ *validator.Validate) error {
	if p.ClassEventThemeCode.Present && p.ClassEventThemeCode.Value != nil && len(*p.ClassEventThemeCode.Value) > 64 {
		return errors.New("class_event_theme_code max 64 characters")
	}
	if p.ClassEventThemeName.Present && p.ClassEventThemeName.Value != nil && len(*p.ClassEventThemeName.Value) > 120 {
		return errors.New("class_event_theme_name max 120 characters")
	}
	if p.ClassEventThemeColor.Present && p.ClassEventThemeColor.Value != nil && len(*p.ClassEventThemeColor.Value) > 32 {
		return errors.New("class_event_theme_color max 32 characters")
	}
	if p.ClassEventThemeCustomColor.Present && p.ClassEventThemeCustomColor.Value != nil {
		if len(*p.ClassEventThemeCustomColor.Value) > 16 {
			return errors.New("class_event_theme_custom_color max 16 characters")
		}
		if !reHexFull.MatchString(*p.ClassEventThemeCustomColor.Value) {
			return errors.New("class_event_theme_custom_color must be hex like #RRGGBB")
		}
	}
	return nil
}

// ApplyPatch: ubah model in-place sesuai field yang Present
func (p *PatchClassEventThemeRequest) ApplyPatch(m *model.ClassEventThemeModel) {
	if val, ok := p.ClassEventThemeCode.Get(); ok {
		if val != nil { // NOT NULL
			m.ClassEventThemeCode = *val
		}
	}
	if val, ok := p.ClassEventThemeName.Get(); ok {
		if val != nil { // NOT NULL
			m.ClassEventThemeName = *val
		}
	}
	if val, ok := p.ClassEventThemeColor.Get(); ok {
		// nullable → boleh nil (clear)
		m.ClassEventThemeColor = val
	}
	if val, ok := p.ClassEventThemeCustomColor.Get(); ok {
		// nullable → boleh nil (clear)
		m.ClassEventThemeCustomColor = val
	}
	if val, ok := p.ClassEventThemeIsActive.Get(); ok && val != nil {
		m.ClassEventThemeIsActive = *val
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
		return "class_event_theme_name DESC"
	case "created_asc":
		return "class_event_theme_created_at ASC"
	case "created_desc":
		return "class_event_theme_created_at DESC"
	case "name_asc":
		fallthrough
	default:
		return "class_event_theme_name ASC"
	}
}

/* =========================================================
   Response DTO
   ========================================================= */

type ClassEventThemeResponse struct {
	ClassEventThemeID          uuid.UUID `json:"class_event_theme_id"`
	ClassEventThemeMasjidID    uuid.UUID `json:"class_event_theme_masjid_id"`
	ClassEventThemeCode        string    `json:"class_event_theme_code"`
	ClassEventThemeName        string    `json:"class_event_theme_name"`
	ClassEventThemeColor       *string   `json:"class_event_theme_color,omitempty"`
	ClassEventThemeCustomColor *string   `json:"class_event_theme_custom_color,omitempty"`
	ClassEventThemeIsActive    bool      `json:"class_event_theme_is_active"`

	ClassEventThemeCreatedAt string `json:"class_event_theme_created_at"`
	ClassEventThemeUpdatedAt string `json:"class_event_theme_updated_at"`
}

func FromModel(m *model.ClassEventThemeModel) ClassEventThemeResponse {
	return ClassEventThemeResponse{
		ClassEventThemeID:          m.ClassEventThemeID,
		ClassEventThemeMasjidID:    m.ClassEventThemeMasjidID,
		ClassEventThemeCode:        m.ClassEventThemeCode,
		ClassEventThemeName:        m.ClassEventThemeName,
		ClassEventThemeColor:       m.ClassEventThemeColor,
		ClassEventThemeCustomColor: m.ClassEventThemeCustomColor,
		ClassEventThemeIsActive:    m.ClassEventThemeIsActive,
		ClassEventThemeCreatedAt:   m.ClassEventThemeCreatedAt.Format(time.RFC3339),
		ClassEventThemeUpdatedAt:   m.ClassEventThemeUpdatedAt.Format(time.RFC3339),
	}
}

func FromModels(list []model.ClassEventThemeModel) []ClassEventThemeResponse {
	out := make([]ClassEventThemeResponse, 0, len(list))
	for i := range list {
		out = append(out, FromModel(&list[i]))
	}
	return out
}
