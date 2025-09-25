// file: internals/features/school/posts/themes/model/post_theme_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
   ENUM: PostThemeKind
   ========================================================= */

type PostThemeKind string

const (
	PostThemeKindAnnouncement PostThemeKind = "announcement"
	PostThemeKindMaterial     PostThemeKind = "material"
	PostThemeKindPost         PostThemeKind = "post"
	PostThemeKindOther        PostThemeKind = "other"
)

func (k PostThemeKind) Valid() bool {
	switch k {
	case PostThemeKindAnnouncement, PostThemeKindMaterial, PostThemeKindPost, PostThemeKindOther:
		return true
	default:
		return false
	}
}

/* =========================================================
   MODEL: post_themes
   ========================================================= */

type PostThemeModel struct {
	// PK & tenant
	PostThemeID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:post_theme_id" json:"post_theme_id"`
	PostThemeMasjidID uuid.UUID `gorm:"type:uuid;not null;column:post_theme_masjid_id" json:"post_theme_masjid_id"`

	// Identitas tema
	PostThemeKind PostThemeKind `gorm:"type:varchar(24);not null;column:post_theme_kind" json:"post_theme_kind"`

	// Hierarki (self-reference; nullable)
	PostThemeParentID *uuid.UUID `gorm:"type:uuid;column:post_theme_parent_id" json:"post_theme_parent_id,omitempty"`

	// Nama & slug
	PostThemeName string `gorm:"type:varchar(80);not null;column:post_theme_name" json:"post_theme_name"`
	PostThemeSlug string `gorm:"type:varchar(120);not null;column:post_theme_slug" json:"post_theme_slug"`

	// Warna & deskripsi (nullable)
	PostThemeColor       *string `gorm:"type:varchar(20);column:post_theme_color" json:"post_theme_color,omitempty"`
	PostThemeCustomColor *string `gorm:"type:varchar(20);column:post_theme_custom_color" json:"post_theme_custom_color,omitempty"`
	PostThemeDescription *string `gorm:"type:text;column:post_theme_description" json:"post_theme_description,omitempty"`

	PostThemeIsActive bool `gorm:"type:boolean;not null;default:true;column:post_theme_is_active" json:"post_theme_is_active"`

	// Icon (2-slot + retensi)
	PostThemeIconURL                *string    `gorm:"type:text;column:post_theme_icon_url" json:"post_theme_icon_url,omitempty"`
	PostThemeIconObjectKey          *string    `gorm:"type:text;column:post_theme_icon_object_key" json:"post_theme_icon_object_key,omitempty"`
	PostThemeIconURLOld             *string    `gorm:"type:text;column:post_theme_icon_url_old" json:"post_theme_icon_url_old,omitempty"`
	PostThemeIconObjectKeyOld       *string    `gorm:"type:text;column:post_theme_icon_object_key_old" json:"post_theme_icon_object_key_old,omitempty"`
	PostThemeIconDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:post_theme_icon_delete_pending_until" json:"post_theme_icon_delete_pending_until,omitempty"`

	// Audit
	PostThemeCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:post_theme_created_at" json:"post_theme_created_at"`
	PostThemeUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:post_theme_updated_at" json:"post_theme_updated_at"`
	PostThemeDeletedAt gorm.DeletedAt `gorm:"column:post_theme_deleted_at;index" json:"post_theme_deleted_at,omitempty"`

	// Relasi self-reference (opsional)
	Parent   *PostThemeModel  `gorm:"foreignKey:PostThemeParentID;references:PostThemeID" json:"parent,omitempty"`
	Children []PostThemeModel `gorm:"foreignKey:PostThemeParentID;references:PostThemeID" json:"children,omitempty"`
}

func (PostThemeModel) TableName() string { return "post_themes" }
