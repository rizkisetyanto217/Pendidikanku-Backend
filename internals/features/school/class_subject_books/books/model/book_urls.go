package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
 * BOOK_URLS (normalize URL)
 * ========================= */
type BookURLModel struct {
	BookURLID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:book_url_id" json:"book_url_id"`
	BookURLMasjidID  uuid.UUID      `gorm:"type:uuid;not null;column:book_url_masjid_id" json:"book_url_masjid_id"`
	BookURLBookID    uuid.UUID      `gorm:"type:uuid;not null;column:book_url_book_id" json:"book_url_book_id"`

	BookURLLabel     *string        `gorm:"type:varchar(120);column:book_url_label" json:"book_url_label,omitempty"`
	BookURLType      string         `gorm:"type:varchar(20);not null;column:book_url_type" json:"book_url_type"` // cover|desc|download|purchase
	BookURLHref      string         `gorm:"type:text;not null;column:book_url_href" json:"book_url_href"`

	// housekeeping optional (ikuti skema SQL)
	BookURLTrashURL            *string    `gorm:"type:text;column:book_url_trash_url" json:"book_url_trash_url,omitempty"`
	BookURLDeletePendingUntil  *time.Time `gorm:"column:book_url_delete_pending_until" json:"book_url_delete_pending_until,omitempty"`

	BookURLCreatedAt time.Time      `gorm:"column:book_url_created_at;autoCreateTime" json:"book_url_created_at"`
	BookURLUpdatedAt time.Time      `gorm:"column:book_url_updated_at;autoUpdateTime" json:"book_url_updated_at"`
	BookURLDeletedAt gorm.DeletedAt `gorm:"column:book_url_deleted_at;index" json:"-"`

	// Relasi balik (opsional)
	// Book *BookModel `gorm:"foreignKey:BookURLBookID;references:BooksID" json:"book,omitempty"`
}

func (BookURLModel) TableName() string { return "book_urls" }