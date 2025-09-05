// internals/features/lembaga/announcements/model/announcement_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	themeModel "masjidku_backend/internals/features/school/announcements/announcement_thema/model"
)

type AnnouncementModel struct {
	AnnouncementID               uuid.UUID      `gorm:"column:announcement_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"announcement_id"`
	AnnouncementMasjidID         uuid.UUID      `gorm:"column:announcement_masjid_id;type:uuid;not null" json:"announcement_masjid_id"`

	// Ganti dari user_id ke teacher_id sesuai SQL
	AnnouncementCreatedByTeacherID *uuid.UUID     `gorm:"column:announcement_created_by_teacher_id;type:uuid" json:"announcement_created_by_teacher_id"`

	AnnouncementClassSectionID     *uuid.UUID     `gorm:"column:announcement_class_section_id;type:uuid" json:"announcement_class_section_id,omitempty"`

	// Relasi tema (tenant-safe via composite FK)
	AnnouncementThemeID            *uuid.UUID     `gorm:"column:announcement_theme_id;type:uuid" json:"announcement_theme_id,omitempty"`

	AnnouncementTitle              string         `gorm:"column:announcement_title;type:varchar(200);not null" json:"announcement_title"`
	AnnouncementDate               time.Time      `gorm:"column:announcement_date;type:date;not null" json:"announcement_date"`
	AnnouncementContent            string         `gorm:"column:announcement_content;type:text;not null" json:"announcement_content"`

	AnnouncementIsActive           bool           `gorm:"column:announcement_is_active;not null;default:true" json:"announcement_is_active"`

	AnnouncementCreatedAt          time.Time      `gorm:"column:announcement_created_at;type:timestamptz;not null;autoCreateTime" json:"announcement_created_at"`
	AnnouncementUpdatedAt          time.Time      `gorm:"column:announcement_updated_at;type:timestamptz;not null;autoUpdateTime" json:"announcement_updated_at"`
	AnnouncementDeletedAt          gorm.DeletedAt `gorm:"column:announcement_deleted_at;index" json:"-"`

	// Generated column (read-only)
	AnnouncementSearch             string         `gorm:"column:announcement_search;type:tsvector;->" json:"-"`

	// Relasi tema (tenant-safe composite FK)
	Theme                         *themeModel.AnnouncementThemeModel `gorm:"foreignKey:AnnouncementThemeID,AnnouncementMasjidID;references:AnnouncementThemesID,AnnouncementThemesMasjidID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	
}

func (AnnouncementModel) TableName() string { return "announcements" }
