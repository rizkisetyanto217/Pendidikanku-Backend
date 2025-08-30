// internals/features/lembaga/announcements/announcement/dto/announcement_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	model "masjidku_backend/internals/features/school/announcements/announcement/model"
)

/* ===================== REQUESTS ===================== */

// Create: masjid_id & created_by diambil dari token/context (bukan dari body)
type CreateAnnouncementRequest struct {
	AnnouncementThemeID        *uuid.UUID `json:"announcement_theme_id" validate:"omitempty"`
	AnnouncementClassSectionID *uuid.UUID `json:"announcement_class_section_id" validate:"omitempty"` // NULL = GLOBAL
	AnnouncementTitle          string     `json:"announcement_title" validate:"required,min=3,max=200"`
	AnnouncementDate           string     `json:"announcement_date" validate:"required,datetime=2006-01-02"` // YYYY-MM-DD
	AnnouncementContent        string     `json:"announcement_content" validate:"required,min=3"`
	AnnouncementIsActive       *bool      `json:"announcement_is_active" validate:"omitempty"`
}

// ToModel: builder untuk create (controller akan provide masjidID & createdBy)
func (r CreateAnnouncementRequest) ToModel(masjidID, createdBy uuid.UUID) *model.AnnouncementModel {
	title := strings.TrimSpace(r.AnnouncementTitle)
	content := strings.TrimSpace(r.AnnouncementContent)
	d, _ := time.Parse("2006-01-02", strings.TrimSpace(r.AnnouncementDate))

	m := &model.AnnouncementModel{
		AnnouncementMasjidID:        masjidID,
		AnnouncementCreatedByTeacherID: &createdBy, // Menggunakan teacher_id
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
	return m
}

/* ===================== UPDATE (partial) ===================== */

type UpdateAnnouncementRequest struct {
	AnnouncementThemeID        *uuid.UUID `json:"announcement_theme_id" validate:"omitempty"`
	AnnouncementClassSectionID *uuid.UUID `json:"announcement_class_section_id" validate:"omitempty"` // set NULL → GLOBAL
	AnnouncementTitle          *string    `json:"announcement_title" validate:"omitempty,min=3,max=200"`
	AnnouncementDate           *string    `json:"announcement_date" validate:"omitempty,datetime=2006-01-02"` // YYYY-MM-DD
	AnnouncementContent        *string    `json:"announcement_content" validate:"omitempty,min=3"`
	AnnouncementIsActive       *bool      `json:"announcement_is_active" validate:"omitempty"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
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
	if r.AnnouncementIsActive != nil {
		m.AnnouncementIsActive = *r.AnnouncementIsActive
	}
	// UpdatedAt dikelola otomatis oleh GORM/trigger
}

/* ===================== QUERIES (list) ===================== */

type ListAnnouncementQuery struct {
	Limit         int        `query:"limit"`
	Offset        int        `query:"offset"`
	ThemeID       *uuid.UUID `query:"theme_id"`
	SectionID     *uuid.UUID `query:"section_id"`
	IncludeGlobal *bool      `query:"include_global"`
	OnlyGlobal    *bool      `query:"only_global"`
	HasAttachment *bool      `query:"has_attachment"` // (tidak dipakai di DDL sekarang; biarkan untuk kompatibilitas filter di layer query, atau hapus jika tidak perlu)
	IsActive      *bool      `query:"is_active"`
	DateFrom      *string    `query:"date_from"`
	DateTo        *string    `query:"date_to"`
	Sort          *string    `query:"sort"`
	IncludePusat  *bool      `query:"include_pusat"` // default true
}

/* ===================== RESPONSES ===================== */

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

	AnnouncementTitle    string    `json:"announcement_title"`
	AnnouncementDate     time.Time `json:"announcement_date"`
	AnnouncementContent  string    `json:"announcement_content"`
	AnnouncementIsActive bool      `json:"announcement_is_active"`

	AnnouncementCreatedAt time.Time `json:"announcement_created_at"`
	AnnouncementUpdatedAt time.Time `json:"announcement_updated_at"` // NOT NULL di DDL

	// Info tema yang sudah dipreload
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
		AnnouncementCreatedByUserID: *m.AnnouncementCreatedByTeacherID, // Sesuaikan dengan teacher_id
		AnnouncementTitle:           m.AnnouncementTitle,
		AnnouncementDate:            m.AnnouncementDate,
		AnnouncementContent:         m.AnnouncementContent,
		AnnouncementIsActive:        m.AnnouncementIsActive,
		AnnouncementCreatedAt:       m.AnnouncementCreatedAt,
		AnnouncementUpdatedAt:       m.AnnouncementUpdatedAt,
	}

	// Map relasi Theme jika sudah dipreload
	if m.Theme != nil {
		resp.Theme = &AnnouncementThemeLite{
			ID:    m.Theme.AnnouncementThemesID,
			Name:  m.Theme.AnnouncementThemesName,
			Color: m.Theme.AnnouncementThemesColor,
		}
	}
	return resp
}
