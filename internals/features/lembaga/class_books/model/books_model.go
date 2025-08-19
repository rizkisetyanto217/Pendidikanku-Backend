package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BooksModel struct {
	BooksID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:books_id"            json:"books_id"`
	BooksMasjidID      uuid.UUID      `gorm:"type:uuid;not null;index;column:books_masjid_id"                           json:"books_masjid_id"`

	BooksTitle         string         `gorm:"type:text;not null;column:books_title"                                     json:"books_title"`
	BooksAuthor        *string        `gorm:"type:text;column:books_author"                                             json:"books_author,omitempty"`
	BooksEdition       *string        `gorm:"type:text;column:books_edition"                                            json:"books_edition,omitempty"`
	BooksPublisher     *string        `gorm:"type:text;column:books_publisher"                                          json:"books_publisher,omitempty"`
	BooksISBN          *string        `gorm:"type:text;column:books_isbn"                                               json:"books_isbn,omitempty"`
	BooksYear          *int           `gorm:"column:books_year"                                                         json:"books_year,omitempty"`
	BooksURL           *string        `gorm:"type:text;column:books_url"                                                json:"books_url,omitempty"`
	BooksImageURL      *string        `gorm:"type:text;column:books_image_url"                                          json:"books_image_url,omitempty"`
	BooksImageThumbURL *string        `gorm:"type:text;column:books_image_thumb_url"                                    json:"books_image_thumb_url,omitempty"`

	BooksCreatedAt     time.Time      `gorm:"column:books_created_at;autoCreateTime"                                    json:"books_created_at"`
	BooksUpdatedAt     time.Time      `gorm:"column:books_updated_at;autoUpdateTime"                                    json:"books_updated_at"`
	BooksDeletedAt     gorm.DeletedAt `gorm:"column:books_deleted_at;index"                                             json:"books_deleted_at,omitempty"`
}

func (BooksModel) TableName() string { return "books" }
