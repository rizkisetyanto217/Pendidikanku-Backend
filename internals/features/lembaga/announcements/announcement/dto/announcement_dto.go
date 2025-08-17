package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/lembaga/announcements/announcement/model"
)

/* ===================== REQUESTS ===================== */

// Create: masjid_id & created_by diambil dari token/context (bukan dari body)
type CreateAnnouncementRequest struct {
	AnnouncementThemeID        *uuid.UUID `json:"announcement_theme_id" validate:"omitempty"`
	AnnouncementClassSectionID *uuid.UUID `json:"announcement_class_section_id" validate:"omitempty"` // NULL = GLOBAL (tampil ke semua user masjid)
	AnnouncementTitle          string     `json:"announcement_title" validate:"required,min=3,max=200"`
	AnnouncementDate           string     `json:"announcement_date" validate:"required,datetime=2006-01-02"` // YYYY-MM-DD
	AnnouncementContent        string     `json:"announcement_content" validate:"required,min=3"`
	AnnouncementAttachmentURL  *string    `json:"announcement_attachment_url" validate:"omitempty"`
	AnnouncementIsActive       *bool      `json:"announcement_is_active" validate:"omitempty"`
}

// ToModel: builder untuk create (controller akan provide masjidID & createdBy)
func (r CreateAnnouncementRequest) ToModel(masjidID, createdBy uuid.UUID) *model.AnnouncementModel {
	title := strings.TrimSpace(r.AnnouncementTitle)
	content := strings.TrimSpace(r.AnnouncementContent)
	d, _ := time.Parse("2006-01-02", strings.TrimSpace(r.AnnouncementDate))

	m := &model.AnnouncementModel{
		AnnouncementMasjidID:        masjidID,
		AnnouncementCreatedByUserID: createdBy,
		AnnouncementThemeID:         r.AnnouncementThemeID,
		AnnouncementClassSectionID:  r.AnnouncementClassSectionID,
		AnnouncementTitle:           title,
		AnnouncementDate:            d,
		AnnouncementContent:         content,
		AnnouncementIsActive:        true, // default aktif
	}

	if r.AnnouncementIsActive != nil {
		m.AnnouncementIsActive = *r.AnnouncementIsActive
	}
	if r.AnnouncementAttachmentURL != nil {
		u := strings.TrimSpace(*r.AnnouncementAttachmentURL)
		if u == "" {
			m.AnnouncementAttachmentURL = nil
		} else {
			m.AnnouncementAttachmentURL = &u
		}
	}
	return m
}

/* ===================== UPDATE (partial) ===================== */

type UpdateAnnouncementRequest struct {
	AnnouncementThemeID        *uuid.UUID `json:"announcement_theme_id" validate:"omitempty"`
	AnnouncementClassSectionID *uuid.UUID `json:"announcement_class_section_id" validate:"omitempty"` // set NULL → GLOBAL
	AnnouncementTitle          *string    `json:"announcement_title" validate:"omitempty,min=3,max=200"`
	AnnouncementDate           *string    `json:"announcement_date" validate:"omitempty,datetime=2006-01-02"` // YYYY-MM-DD
	AnnouncementContent        *string    `json:"announcement_content" validate:"omitempty,min=3"`
	AnnouncementAttachmentURL  *string    `json:"announcement_attachment_url" validate:"omitempty"`
	AnnouncementIsActive       *bool      `json:"announcement_is_active" validate:"omitempty"`
}

// ApplyToModel: terapkan hanya field yang dikirim
func (r *UpdateAnnouncementRequest) ApplyToModel(m *model.AnnouncementModel) {
	if r.AnnouncementThemeID != nil {
		m.AnnouncementThemeID = r.AnnouncementThemeID
	}
	if r.AnnouncementClassSectionID != nil {
		m.AnnouncementClassSectionID = r.AnnouncementClassSectionID // boleh set ke nil → GLOBAL
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
	if r.AnnouncementAttachmentURL != nil {
		u := strings.TrimSpace(*r.AnnouncementAttachmentURL)
		if u == "" {
			m.AnnouncementAttachmentURL = nil
		} else {
			m.AnnouncementAttachmentURL = &u
		}
	}
	if r.AnnouncementIsActive != nil {
		m.AnnouncementIsActive = *r.AnnouncementIsActive
	}

	now := time.Now()
	m.AnnouncementUpdatedAt = &now
}

/* ===================== QUERIES (list) ===================== */

// internals/features/lembaga/announcements/announcement/dto/list_query.go
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

  // NEW:
  IncludePusat  *bool      `query:"include_pusat"` // default true
}


// internals/features/lembaga/announcements/announcement/dto/announcement_response.go

type AnnouncementThemeLite struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name,omitempty"`
	Color *string   `json:"color,omitempty"`
}

type AnnouncementResponse struct {
	AnnouncementID              uuid.UUID  `json:"announcement_id"`
	AnnouncementMasjidID        uuid.UUID  `json:"announcement_masjid_id"`
	AnnouncementThemeID         *uuid.UUID `json:"announcement_theme_id,omitempty"`
	AnnouncementClassSectionID  *uuid.UUID `json:"announcement_class_section_id,omitempty"`
	AnnouncementCreatedByUserID uuid.UUID  `json:"announcement_created_by_user_id"`

	AnnouncementTitle         string     `json:"announcement_title"`
	AnnouncementDate          time.Time  `json:"announcement_date"`
	AnnouncementContent       string     `json:"announcement_content"`
	AnnouncementAttachmentURL *string    `json:"announcement_attachment_url,omitempty"`
	AnnouncementIsActive      bool       `json:"announcement_is_active"`

	AnnouncementCreatedAt time.Time  `json:"announcement_created_at"`
	AnnouncementUpdatedAt *time.Time `json:"announcement_updated_at,omitempty"`

	// NEW: info tema yang sudah dipreload
	Theme *AnnouncementThemeLite `json:"theme,omitempty"`
}

// Factory response
func NewAnnouncementResponse(m *model.AnnouncementModel) *AnnouncementResponse {
	if m == nil {
		return nil
	}
	resp := &AnnouncementResponse{
		AnnouncementID:              m.AnnouncementID,
		AnnouncementMasjidID:        m.AnnouncementMasjidID,
		AnnouncementThemeID:         m.AnnouncementThemeID,
		AnnouncementClassSectionID:  m.AnnouncementClassSectionID,
		AnnouncementCreatedByUserID: m.AnnouncementCreatedByUserID,
		AnnouncementTitle:           m.AnnouncementTitle,
		AnnouncementDate:            m.AnnouncementDate,
		AnnouncementContent:         m.AnnouncementContent,
		AnnouncementAttachmentURL:   m.AnnouncementAttachmentURL,
		AnnouncementIsActive:        m.AnnouncementIsActive,
		AnnouncementCreatedAt:       m.AnnouncementCreatedAt,
		AnnouncementUpdatedAt:       m.AnnouncementUpdatedAt,
	}

	// Map relasi Theme jika sudah dipreload
	if m.Theme != nil {
		resp.Theme = &AnnouncementThemeLite{
			ID:    m.Theme.AnnouncementThemesID,     // sesuaikan dengan field model tema kamu
			Name:  m.Theme.AnnouncementThemesName,   // sesuaikan
			Color: m.Theme.AnnouncementThemesColor,  // opsional, sesuaikan
		}
	}
	return resp
}
