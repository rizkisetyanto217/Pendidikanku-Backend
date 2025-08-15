// internals/features/lembaga/announcements/model/announcement_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	// import model tema dengan alias agar tidak bentrok
	themeModel "masjidku_backend/internals/features/lembaga/announcements/announcement_thema/model"
)

// internals/features/lembaga/announcements/model/announcement_model.go
type AnnouncementModel struct {
	AnnouncementID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:announcement_id" json:"announcement_id"`
	AnnouncementMasjidID       uuid.UUID      `gorm:"type:uuid;not null;column:announcement_masjid_id" json:"announcement_masjid_id"`

	AnnouncementCreatedByUserID uuid.UUID   `gorm:"type:uuid;not null;column:announcement_created_by_user_id" json:"announcement_created_by_user_id"`
	AnnouncementClassSectionID  *uuid.UUID  `gorm:"type:uuid;column:announcement_class_section_id" json:"announcement_class_section_id,omitempty"`

	AnnouncementThemeID *uuid.UUID `gorm:"type:uuid;column:announcement_theme_id" json:"announcement_theme_id,omitempty"`

	AnnouncementTitle   string    `gorm:"size:200;not null;column:announcement_title" json:"announcement_title"`
	AnnouncementDate    time.Time `gorm:"type:date;not null;column:announcement_date" json:"announcement_date"`
	AnnouncementContent string    `gorm:"type:text;not null;column:announcement_content" json:"announcement_content"`

	AnnouncementAttachmentURL *string       `gorm:"type:text;column:announcement_attachment_url" json:"announcement_attachment_url,omitempty"`
	AnnouncementIsActive      bool          `gorm:"not null;default:true;column:announcement_is_active" json:"announcement_is_active"`

	AnnouncementCreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;column:announcement_created_at" json:"announcement_created_at"`
	AnnouncementUpdatedAt *time.Time     `gorm:"column:announcement_updated_at" json:"announcement_updated_at,omitempty"`
	AnnouncementDeletedAt gorm.DeletedAt `gorm:"column:announcement_deleted_at;index" json:"-"`

	AnnouncementSearch string `gorm:"type:tsvector;column:announcement_search;->" json:"-"`

	// relasi theme tetap:
	Theme *themeModel.AnnouncementThemeModel `gorm:"foreignKey:AnnouncementThemeID,AnnouncementMasjidID;references:AnnouncementThemesID,AnnouncementThemesMasjidID;constraint:OnUpdate:RESTRICT,OnDelete:SET NULL" json:"-"`
}

func (AnnouncementModel) TableName() string { return "announcements" }
