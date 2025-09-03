package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BooksModel struct {
	BooksID       uuid.UUID      `gorm:"column:books_id;type:uuid;default:gen_random_uuid();primaryKey" json:"books_id"`
	BooksMasjidID uuid.UUID      `gorm:"column:books_masjid_id;type:uuid;not null;index"               json:"books_masjid_id"`

	BooksTitle  string   `gorm:"column:books_title;type:text;not null" json:"books_title"`
	BooksAuthor *string  `gorm:"column:books_author;type:text"         json:"books_author,omitempty"`
	BooksDesc   *string  `gorm:"column:books_desc;type:text"           json:"books_desc,omitempty"`

	// slug nullable; unik per masjid via partial unique index di SQL (uq_books_slug_per_masjid_alive)
	BooksSlug *string `gorm:"column:books_slug;type:varchar(160)" json:"books_slug,omitempty"`

	BooksCreatedAt time.Time      `gorm:"column:books_created_at;autoCreateTime" json:"books_created_at"`
	BooksUpdatedAt time.Time      `gorm:"column:books_updated_at;autoUpdateTime" json:"books_updated_at"`
	BooksDeletedAt gorm.DeletedAt `gorm:"column:books_deleted_at;index"          json:"books_deleted_at,omitempty"`
}

func (BooksModel) TableName() string { return "books" }
