// file: internals/features/social/posts/model/post_url_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
post_urls
- Soft delete: post_url_deleted_at
- Retensi 2-slot object key + delete_pending_until
- Primary flag unik per (post, kind) sudah dijaga index partial di DB
*/

type PostURL struct {
	// PK & tenant
	PostURLID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:post_url_id" json:"post_url_id"`
	PostURLMasjidID uuid.UUID `gorm:"type:uuid;not null;column:post_url_masjid_id" json:"post_url_masjid_id"`
	PostURLPostID   uuid.UUID `gorm:"type:uuid;not null;column:post_url_post_id" json:"post_url_post_id"`

	// Jenis & lokasi file/link
	PostURLKind         string  `gorm:"type:varchar(24);not null;column:post_url_kind" json:"post_url_kind"`
	PostURLHref         *string `gorm:"type:text;column:post_url_href" json:"post_url_href,omitempty"`
	PostURLObjectKey    *string `gorm:"type:text;column:post_url_object_key" json:"post_url_object_key,omitempty"`
	PostURLObjectKeyOld *string `gorm:"type:text;column:post_url_object_key_old" json:"post_url_object_key_old,omitempty"`

	// Label & urutan
	PostURLLabel     *string `gorm:"type:varchar(160);column:post_url_label" json:"post_url_label,omitempty"`
	PostURLOrder     int     `gorm:"type:int;not null;default:0;column:post_url_order" json:"post_url_order"`
	PostURLIsPrimary bool    `gorm:"type:boolean;not null;default:false;column:post_url_is_primary" json:"post_url_is_primary"`

	// Audit & soft delete
	PostURLCreatedAt          time.Time      `gorm:"type:timestamptz;not null;default:now();column:post_url_created_at" json:"post_url_created_at"`
	PostURLUpdatedAt          time.Time      `gorm:"type:timestamptz;not null;default:now();column:post_url_updated_at" json:"post_url_updated_at"`
	PostURLDeletedAt          gorm.DeletedAt `gorm:"column:post_url_deleted_at" json:"post_url_deleted_at" swaggertype:"string"`
	PostURLDeletePendingUntil *time.Time     `gorm:"type:timestamptz;column:post_url_delete_pending_until" json:"post_url_delete_pending_until,omitempty"`
}

func (PostURL) TableName() string { return "post_urls" }
