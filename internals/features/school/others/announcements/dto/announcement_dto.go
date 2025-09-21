// internals/features/lembaga/announcements/dto/announcement_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/others/announcements/model"
)

/* ===================== Utils ===================== */

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

/* ===================== URL SUB-PAYLOAD (untuk Create/Update) ===================== */

type AnnouncementURLUpsert struct {
	AnnouncementURLKind      string  `json:"announcement_url_kind" validate:"required,max=24"`
	AnnouncementURLHref      *string `json:"announcement_url_href"`
	AnnouncementURLObjectKey *string `json:"announcement_url_object_key"`
	AnnouncementURLLabel     *string `json:"announcement_url_label" validate:"omitempty,max=160"`
	AnnouncementURLOrder     int     `json:"announcement_url_order"`
	AnnouncementURLIsPrimary bool    `json:"announcement_url_is_primary"`
}

func (u *AnnouncementURLUpsert) Normalize() {
	u.AnnouncementURLKind = strings.TrimSpace(u.AnnouncementURLKind)
	if u.AnnouncementURLKind == "" {
		u.AnnouncementURLKind = "attachment"
	}
	if u.AnnouncementURLLabel != nil {
		lbl := strings.TrimSpace(*u.AnnouncementURLLabel)
		if lbl == "" {
			u.AnnouncementURLLabel = nil
		} else {
			u.AnnouncementURLLabel = &lbl
		}
	}
	if u.AnnouncementURLHref != nil {
		h := strings.TrimSpace(*u.AnnouncementURLHref)
		if h == "" {
			u.AnnouncementURLHref = nil
		} else {
			u.AnnouncementURLHref = &h
		}
	}
	if u.AnnouncementURLObjectKey != nil {
		ok := strings.TrimSpace(*u.AnnouncementURLObjectKey)
		if ok == "" {
			u.AnnouncementURLObjectKey = nil
		} else {
			u.AnnouncementURLObjectKey = &ok
		}
	}
}

/* ===================== REQUESTS ===================== */

type CreateAnnouncementRequest struct {
	AnnouncementThemeID        *uuid.UUID `json:"announcement_theme_id" validate:"omitempty,uuid"`
	AnnouncementClassSectionID *uuid.UUID `json:"announcement_class_section_id" validate:"omitempty,uuid"` // NULL = GLOBAL
	AnnouncementTitle          string     `json:"announcement_title" validate:"required,min=3,max=200"`
	AnnouncementDate           string     `json:"announcement_date" validate:"required,datetime=2006-01-02"` // YYYY-MM-DD
	AnnouncementContent        string     `json:"announcement_content" validate:"required,min=3"`
	AnnouncementIsActive       *bool      `json:"announcement_is_active" validate:"omitempty"`
	AnnouncementSlug           *string    `json:"announcement_slug" validate:"omitempty,max=160"` // opsional; biasanya digenerate controller

	// (opsional) creator; biasanya diisi dari token controller (teacher)
	AnnouncementCreatedByTeacherID *uuid.UUID `json:"announcement_created_by_teacher_id" validate:"omitempty,uuid"`

	// Lampiran metadata opsional
	URLs []AnnouncementURLUpsert `json:"urls" validate:"omitempty,dive"`
}

func (r CreateAnnouncementRequest) ToModel(masjidID uuid.UUID) *model.AnnouncementModel {
	title := strings.TrimSpace(r.AnnouncementTitle)
	content := strings.TrimSpace(r.AnnouncementContent)

	var d time.Time
	if ds := strings.TrimSpace(r.AnnouncementDate); ds != "" {
		d, _ = time.Parse("2006-01-02", ds)
	}

	m := &model.AnnouncementModel{
		AnnouncementMasjidID:           masjidID,
		AnnouncementThemeID:            r.AnnouncementThemeID,
		AnnouncementClassSectionID:     r.AnnouncementClassSectionID,
		AnnouncementTitle:              title,
		AnnouncementDate:               d,
		AnnouncementContent:            content,
		AnnouncementIsActive:           true,                             // default aktif
		AnnouncementCreatedByTeacherID: r.AnnouncementCreatedByTeacherID, // controller boleh override dari token
		AnnouncementSlug:               trimPtr(r.AnnouncementSlug),      // controller nanti ensure-unique
	}
	if r.AnnouncementIsActive != nil {
		m.AnnouncementIsActive = *r.AnnouncementIsActive
	}
	return m
}

/* ===================== UPDATE (partial) ===================== */

type UpdateAnnouncementRequest struct {
	AnnouncementThemeID            *uuid.UUID `json:"announcement_theme_id" validate:"omitempty,uuid"`
	AnnouncementClassSectionID     *uuid.UUID `json:"announcement_class_section_id" validate:"omitempty,uuid"`
	AnnouncementTitle              *string    `json:"announcement_title" validate:"omitempty,min=3,max=200"`
	AnnouncementDate               *string    `json:"announcement_date" validate:"omitempty,datetime=2006-01-02"`
	AnnouncementContent            *string    `json:"announcement_content" validate:"omitempty,min=3"`
	AnnouncementIsActive           *bool      `json:"announcement_is_active" validate:"omitempty"`
	AnnouncementSlug               *string    `json:"announcement_slug" validate:"omitempty,max=160"` // opsional; controller akan normalize+ensure-unique
	AnnouncementCreatedByTeacherID *uuid.UUID `json:"announcement_created_by_teacher_id" validate:"omitempty,uuid"`

	// URL opsional
	URLs           []AnnouncementURLUpsert `json:"urls" validate:"omitempty,dive"`
	DeleteURLIDs   []uuid.UUID             `json:"delete_url_ids" validate:"omitempty,dive,uuid"`
	PrimaryPerKind map[string]uuid.UUID    `json:"primary_per_kind" validate:"omitempty"`
}

func (r *UpdateAnnouncementRequest) ApplyToModel(m *model.AnnouncementModel) {
	if r.AnnouncementThemeID != nil {
		m.AnnouncementThemeID = r.AnnouncementThemeID
	}
	// Section: nil berarti GLOBAL
	if r.AnnouncementClassSectionID != nil || r.AnnouncementClassSectionID == nil {
		m.AnnouncementClassSectionID = r.AnnouncementClassSectionID
	}
	if r.AnnouncementTitle != nil {
		m.AnnouncementTitle = strings.TrimSpace(*r.AnnouncementTitle)
	}
	if r.AnnouncementDate != nil {
		if dt := strings.TrimSpace(*r.AnnouncementDate); dt != "" {
			if parsed, err := time.Parse("2006-01-02", dt); err == nil {
				m.AnnouncementDate = parsed
			}
		}
	}
	if r.AnnouncementContent != nil {
		m.AnnouncementContent = strings.TrimSpace(*r.AnnouncementContent)
	}
	if r.AnnouncementIsActive != nil {
		m.AnnouncementIsActive = *r.AnnouncementIsActive
	}
	// Slug: normalize di DTO; ensure-unique di controller
	if r.AnnouncementSlug != nil {
		m.AnnouncementSlug = trimPtr(r.AnnouncementSlug)
	}
	// (opsional) Creator override — biasanya dari token
	if r.AnnouncementCreatedByTeacherID != nil {
		m.AnnouncementCreatedByTeacherID = r.AnnouncementCreatedByTeacherID
	}
}

/* ===================== QUERIES (list) ===================== */

type ListAnnouncementQuery struct {
	Limit         int        `query:"limit"`
	Offset        int        `query:"offset"`
	ThemeID       *uuid.UUID `query:"theme_id"`
	SectionID     *uuid.UUID `query:"section_id"`
	IncludeGlobal *bool      `query:"include_global"`
	OnlyGlobal    *bool      `query:"only_global"`
	HasAttachment *bool      `query:"has_attachment"`
	IsActive      *bool      `query:"is_active"`
	DateFrom      *string    `query:"date_from"`
	DateTo        *string    `query:"date_to"`
	Sort          *string    `query:"sort"`
	IncludePusat  *bool      `query:"include_pusat"`

	// raw include param, e.g. "theme,urls"
	Include string `query:"include"`
}

// Helper parse include di level DTO
func (q ListAnnouncementQuery) WantTheme() bool {
	return hasInclude(q.Include, "theme", "themes", "announcement_theme")
}
func (q ListAnnouncementQuery) WantURLs() bool {
	return hasInclude(q.Include, "urls", "attachments", "announcement_urls")
}

func hasInclude(raw string, keys ...string) bool {
	if raw == "" {
		return false
	}
	raw = strings.ToLower(strings.TrimSpace(raw))
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		for _, k := range keys {
			if p == k {
				return true
			}
		}
	}
	return false
}

/* ===================== RESPONSES ===================== */

type AnnouncementThemeLite struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name,omitempty"`
	Color *string   `json:"color,omitempty"`
}

// Diperluas agar lebih berguna saat include=urls
type AnnouncementURLLite struct {
	ID             uuid.UUID `json:"id"`
	Label          *string   `json:"label,omitempty"`
	AnnouncementID uuid.UUID `json:"announcement_id"`
	Href           string    `json:"href"`

	// NEW (opsional)
	Kind      *string `json:"kind,omitempty"`
	Order     *int    `json:"order,omitempty"`
	IsPrimary *bool   `json:"is_primary,omitempty"`
}

type AnnouncementResponse struct {
	AnnouncementID                 uuid.UUID  `json:"announcement_id"`
	AnnouncementMasjidID           uuid.UUID  `json:"announcement_masjid_id"`
	AnnouncementThemeID            *uuid.UUID `json:"announcement_theme_id,omitempty"`
	AnnouncementClassSectionID     *uuid.UUID `json:"announcement_class_section_id,omitempty"`
	AnnouncementCreatedByTeacherID *uuid.UUID `json:"announcement_created_by_teacher_id,omitempty"`

	AnnouncementSlug     *string   `json:"announcement_slug,omitempty"`
	AnnouncementTitle    string    `json:"announcement_title"`
	AnnouncementDate     time.Time `json:"announcement_date"`
	AnnouncementContent  string    `json:"announcement_content"`
	AnnouncementIsActive bool      `json:"announcement_is_active"`

	AnnouncementCreatedAt time.Time `json:"announcement_created_at"`
	AnnouncementUpdatedAt time.Time `json:"announcement_updated_at"`

	Theme *AnnouncementThemeLite `json:"theme,omitempty"`
	Urls  []*AnnouncementURLLite `json:"urls,omitempty"`
}

/* ===================== Builders & Attach helpers ===================== */

func NewAnnouncementResponse(m *model.AnnouncementModel) *AnnouncementResponse {
	if m == nil {
		return nil
	}
	resp := &AnnouncementResponse{
		AnnouncementID:                 m.AnnouncementID,
		AnnouncementMasjidID:           m.AnnouncementMasjidID,
		AnnouncementThemeID:            m.AnnouncementThemeID,
		AnnouncementClassSectionID:     m.AnnouncementClassSectionID,
		AnnouncementCreatedByTeacherID: m.AnnouncementCreatedByTeacherID,

		AnnouncementSlug:     m.AnnouncementSlug,
		AnnouncementTitle:    m.AnnouncementTitle,
		AnnouncementDate:     m.AnnouncementDate,
		AnnouncementContent:  m.AnnouncementContent,
		AnnouncementIsActive: m.AnnouncementIsActive,

		AnnouncementCreatedAt: m.AnnouncementCreatedAt,
		AnnouncementUpdatedAt: m.AnnouncementUpdatedAt,
	}
	// Theme akan terisi jika controller preload / batch-load & assign ke m.Theme
	if m.Theme.AnnouncementThemesID != uuid.Nil {
		resp.Theme = &AnnouncementThemeLite{
			ID:    m.Theme.AnnouncementThemesID,
			Name:  m.Theme.AnnouncementThemesName,
			Color: m.Theme.AnnouncementThemesColor,
		}
	}
	return resp
}

// AttachTheme: helper optional (kalau theme di-load terpisah)
func (r *AnnouncementResponse) AttachTheme(th *model.AnnouncementThemeModel) *AnnouncementResponse {
	if r == nil || th == nil || th.AnnouncementThemesID == uuid.Nil {
		return r
	}
	r.Theme = &AnnouncementThemeLite{
		ID:    th.AnnouncementThemesID,
		Name:  th.AnnouncementThemesName,
		Color: th.AnnouncementThemesColor,
	}
	return r
}

// AttachURLs: convert rows -> URL lites, isi ke response
func (r *AnnouncementResponse) AttachURLs(rows []model.AnnouncementURLModel) *AnnouncementResponse {
	if r == nil || len(rows) == 0 {
		return r
	}
	out := make([]*AnnouncementURLLite, 0, len(rows))
	for i := range rows {
		// hanya kirim yang punya Href (kompat FE lama)
		if rows[i].AnnouncementURLHref == nil || strings.TrimSpace(*rows[i].AnnouncementURLHref) == "" {
			continue
		}
		l := &AnnouncementURLLite{
			ID:             rows[i].AnnouncementURLId,
			Label:          rows[i].AnnouncementURLLabel,
			AnnouncementID: rows[i].AnnouncementURLAnnouncementId,
			Href:           *rows[i].AnnouncementURLHref,
		}
		// field baru opsional
		if k := strings.TrimSpace(rows[i].AnnouncementURLKind); k != "" {
			l.Kind = &rows[i].AnnouncementURLKind
		}
		if rows[i].AnnouncementURLOrder != 0 {
			o := rows[i].AnnouncementURLOrder
			l.Order = &o
		}
		if rows[i].AnnouncementURLIsPrimary {
			b := true
			l.IsPrimary = &b
		}
		out = append(out, l)
	}
	if len(out) > 0 {
		r.Urls = out
	}
	return r
}
