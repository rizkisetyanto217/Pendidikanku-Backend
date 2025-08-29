// internals/features/lembaga/announcements/dto/announcement_theme_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/announcements/announcement_thema/model"
	helper "masjidku_backend/internals/helpers"
)

/* ===================== REQUESTS ===================== */

// Create: masjid_id diambil dari token/context, jadi tidak ada di payload
type CreateAnnouncementThemeRequest struct {
	AnnouncementThemesName        string  `json:"announcement_themes_name" validate:"required,min=2,max=80"`
	AnnouncementThemesSlug        string  `json:"announcement_themes_slug" validate:"required,min=2,max=120"`
	AnnouncementThemesColor       *string `json:"announcement_themes_color" validate:"omitempty,hexcolor"` // contoh: #0ea5e9
	AnnouncementThemesDescription *string `json:"announcement_themes_description" validate:"omitempty"`
	AnnouncementThemesIsActive    *bool   `json:"announcement_themes_is_active" validate:"omitempty"`
}

// ToModel: builder untuk create
func (r CreateAnnouncementThemeRequest) ToModel(masjidID uuid.UUID) *model.AnnouncementThemeModel {
	name := strings.TrimSpace(r.AnnouncementThemesName)

	// Safety: rapikan slug lagi (controller juga sudah lakukan)
	slug := helper.GenerateSlug(strings.TrimSpace(r.AnnouncementThemesSlug))
	if slug == "" {
		slug = helper.GenerateSlug(name)
	}

	m := &model.AnnouncementThemeModel{
		AnnouncementThemesMasjidID: masjidID,
		AnnouncementThemesName:     name,
		AnnouncementThemesSlug:     slug,
		AnnouncementThemesIsActive: true, // default true sesuai DDL
	}

	if r.AnnouncementThemesColor != nil {
		c := strings.TrimSpace(*r.AnnouncementThemesColor)
		if c == "" {
			m.AnnouncementThemesColor = nil
		} else {
			m.AnnouncementThemesColor = &c
		}
	}
	if r.AnnouncementThemesDescription != nil {
		d := strings.TrimSpace(*r.AnnouncementThemesDescription)
		if d == "" {
			m.AnnouncementThemesDescription = nil
		} else {
			m.AnnouncementThemesDescription = &d
		}
	}
	if r.AnnouncementThemesIsActive != nil {
		m.AnnouncementThemesIsActive = *r.AnnouncementThemesIsActive
	}

	// CreatedAt/UpdatedAt biarkan diisi otomatis oleh GORM (autoCreateTime/autoUpdateTime)
	return m
}

// Update: semua optional (partial update)
type UpdateAnnouncementThemeRequest struct {
	AnnouncementThemesName        *string `json:"announcement_themes_name" validate:"omitempty,min=2,max=80"`
	AnnouncementThemesSlug        *string `json:"announcement_themes_slug" validate:"omitempty,min=2,max=120"`
	AnnouncementThemesColor       *string `json:"announcement_themes_color" validate:"omitempty,hexcolor"`
	AnnouncementThemesDescription *string `json:"announcement_themes_description" validate:"omitempty"`
	AnnouncementThemesIsActive    *bool   `json:"announcement_themes_is_active" validate:"omitempty"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// ApplyToModel: terapkan hanya field yang dikirim
func (r *UpdateAnnouncementThemeRequest) ApplyToModel(m *model.AnnouncementThemeModel) {
	if r.AnnouncementThemesName != nil {
		m.AnnouncementThemesName = strings.TrimSpace(*r.AnnouncementThemesName)
	}
	if r.AnnouncementThemesSlug != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*r.AnnouncementThemesSlug))
		if s != "" { // jangan set kalau hasil kosong
			m.AnnouncementThemesSlug = s
		}
	}
	if r.AnnouncementThemesColor != nil {
		c := strings.TrimSpace(*r.AnnouncementThemesColor)
		if c == "" {
			m.AnnouncementThemesColor = nil
		} else {
			m.AnnouncementThemesColor = &c
		}
	}
	if r.AnnouncementThemesDescription != nil {
		d := strings.TrimSpace(*r.AnnouncementThemesDescription)
		if d == "" {
			m.AnnouncementThemesDescription = nil
		} else {
			m.AnnouncementThemesDescription = &d
		}
	}
	if r.AnnouncementThemesIsActive != nil {
		m.AnnouncementThemesIsActive = *r.AnnouncementThemesIsActive
	}

	// UpdatedAt biarkan di-handle autoUpdateTime pada Save/Updates
}

/* ===================== QUERIES ===================== */

type ListAnnouncementThemeQuery struct {
	Name     *string `query:"name"`      // ILIKE %name%
	Slug     *string `query:"slug"`      // exact / startswith (di controller)
	IsActive *bool   `query:"is_active"` // true/false
	Limit    int     `query:"limit"`     // default 20
	Offset   int     `query:"offset"`    // default 0
	Sort     *string `query:"sort"`      // "created_at_desc" | "created_at_asc" | "name_asc" | "name_desc"
}

/* ===================== RESPONSES ===================== */

type AnnouncementThemeResponse struct {
	AnnouncementThemesID         uuid.UUID `json:"announcement_themes_id"`
	AnnouncementThemesMasjidID   uuid.UUID `json:"announcement_themes_masjid_id"`

	AnnouncementThemesName       string    `json:"announcement_themes_name"`
	AnnouncementThemesSlug       string    `json:"announcement_themes_slug"`
	AnnouncementThemesColor      *string   `json:"announcement_themes_color,omitempty"`
	AnnouncementThemesDescription *string  `json:"announcement_themes_description,omitempty"`
	AnnouncementThemesIsActive   bool      `json:"announcement_themes_is_active"`

	AnnouncementThemesCreatedAt  time.Time `json:"announcement_themes_created_at"`
	AnnouncementThemesUpdatedAt  time.Time `json:"announcement_themes_updated_at"` // NOT NULL di DDL
}

// Factory response
func NewAnnouncementThemeResponse(m *model.AnnouncementThemeModel) *AnnouncementThemeResponse {
	if m == nil {
		return nil
	}
	return &AnnouncementThemeResponse{
		AnnouncementThemesID:          m.AnnouncementThemesID,
		AnnouncementThemesMasjidID:    m.AnnouncementThemesMasjidID,
		AnnouncementThemesName:        m.AnnouncementThemesName,
		AnnouncementThemesSlug:        m.AnnouncementThemesSlug,
		AnnouncementThemesColor:       m.AnnouncementThemesColor,
		AnnouncementThemesDescription: m.AnnouncementThemesDescription,
		AnnouncementThemesIsActive:    m.AnnouncementThemesIsActive,
		AnnouncementThemesCreatedAt:   m.AnnouncementThemesCreatedAt,
		AnnouncementThemesUpdatedAt:   m.AnnouncementThemesUpdatedAt,
	}
}
