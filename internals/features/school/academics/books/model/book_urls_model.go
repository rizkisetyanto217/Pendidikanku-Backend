package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

/*
=========================================
Model: book_urls
=========================================
*/

type BookURLModel struct {
	BookURLID uuid.UUID `json:"book_url_id"                 gorm:"column:book_url_id;type:uuid;primaryKey;not null;default:gen_random_uuid()"`

	BookURLSchoolID uuid.UUID `json:"book_url_school_id"          gorm:"column:book_url_school_id;type:uuid;not null"`
	BookURLBookID   uuid.UUID `json:"book_url_book_id"            gorm:"column:book_url_book_id;type:uuid;not null"`

	BookURLKind string `json:"book_url_kind"               gorm:"column:book_url_kind;type:varchar(24);not null"`

	BookURLHref         *string `json:"book_url_href,omitempty"             gorm:"column:book_url_href;type:text"`
	BookURLObjectKey    *string `json:"book_url_object_key,omitempty"       gorm:"column:book_url_object_key;type:text"`
	BookURLObjectKeyOld *string `json:"book_url_object_key_old,omitempty"   gorm:"column:book_url_object_key_old;type:text"`

	BookURLLabel     *string `json:"book_url_label,omitempty"            gorm:"column:book_url_label;type:varchar(160)"`
	BookURLOrder     int     `json:"book_url_order"                      gorm:"column:book_url_order;not null;default:0"`
	BookURLIsPrimary bool    `json:"book_url_is_primary"                 gorm:"column:book_url_is_primary;not null;default:false"`

	BookURLCreatedAt          time.Time    `json:"book_url_created_at"                 gorm:"column:book_url_created_at;type:timestamptz;not null;default:now()"`
	BookURLUpdatedAt          time.Time    `json:"book_url_updated_at"                 gorm:"column:book_url_updated_at;type:timestamptz;not null;default:now()"`
	BookURLDeletedAt          sql.NullTime `json:"book_url_deleted_at,omitempty"       gorm:"column:book_url_deleted_at;type:timestamptz"`
	BookURLDeletePendingUntil sql.NullTime `json:"book_url_delete_pending_until,omitempty" gorm:"column:book_url_delete_pending_until;type:timestamptz"`
}

func (BookURLModel) TableName() string { return "book_urls" }
