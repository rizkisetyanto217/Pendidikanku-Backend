// file: internals/features/social/posts/model/post_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   Enum: PostKind
========================= */

type PostKind string

const (
	PostKindAnnouncement PostKind = "announcement"
	PostKindMaterial     PostKind = "material"
	PostKindPost         PostKind = "post"
	PostKindOther        PostKind = "other"
)

/* =========================
   Model: Post
========================= */

type Post struct {
	// PK & tenant
	PostID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:post_id" json:"post_id"`
	PostMasjidID uuid.UUID `gorm:"type:uuid;not null;column:post_masjid_id" json:"post_masjid_id"`

	// Jenis
	PostKind PostKind `gorm:"type:varchar(24);not null;column:post_kind" json:"post_kind"`

	// Pengirim & relasi
	IsDKMSender            bool       `gorm:"type:boolean;not null;default:false;column:is_dkm_sender" json:"is_dkm_sender"`
	PostCreatedByTeacherID *uuid.UUID `gorm:"type:uuid;column:post_created_by_teacher_id" json:"post_created_by_teacher_id,omitempty"`
	PostClassSectionID     *uuid.UUID `gorm:"type:uuid;column:post_class_section_id" json:"post_class_section_id,omitempty"`
	PostThemeID            *uuid.UUID `gorm:"type:uuid;column:post_theme_id" json:"post_theme_id,omitempty"`

	// Identitas & isi
	PostSlug    *string   `gorm:"type:varchar(160);column:post_slug" json:"post_slug,omitempty"`
	PostTitle   string    `gorm:"type:varchar(200);not null;column:post_title" json:"post_title"`
	PostDate    time.Time `gorm:"type:date;not null;column:post_date" json:"post_date"`
	PostContent string    `gorm:"type:text;not null;column:post_content" json:"post_content"`

	PostExcerpt *string        `gorm:"type:text;column:post_excerpt" json:"post_excerpt,omitempty"`
	PostMeta    datatypes.JSON `gorm:"type:jsonb;column:post_meta" json:"post_meta,omitempty"`

	// Status & audit
	PostIsActive    bool           `gorm:"type:boolean;not null;default:true;column:post_is_active" json:"post_is_active"`
	PostCreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now();column:post_created_at" json:"post_created_at"`
	PostUpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now();column:post_updated_at" json:"post_updated_at"`
	PostDeletedAt   gorm.DeletedAt `gorm:"column:post_deleted_at" json:"post_deleted_at" swaggertype:"string"`
	PostIsPublished bool           `gorm:"type:boolean;not null;default:false;column:post_is_published" json:"post_is_published"`
	PostPublishedAt *time.Time     `gorm:"type:timestamptz;column:post_published_at" json:"post_published_at,omitempty"`

	// Snapshot audiens saat publish
	PostAudienceSnapshot datatypes.JSON `gorm:"type:jsonb;column:post_audience_snapshot" json:"post_audience_snapshot,omitempty"`

	// Search vector (generated, read-only)
	PostSearch string `gorm:"->;type:tsvector;column:post_search" json:"post_search,omitempty"`

	// Target section (ARRAY UUID, hanya announcement)
	PostSectionIDs []uuid.UUID `gorm:"type:uuid[];column:post_section_ids" json:"post_section_ids"`
}

func (Post) TableName() string { return "posts" }
