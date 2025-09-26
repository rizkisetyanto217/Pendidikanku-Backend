// file: internals/features/library/books/model/book_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookModel struct {
	BookID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:book_id" json:"book_id"`
	BookMasjidID uuid.UUID `gorm:"type:uuid;not null;column:book_masjid_id" json:"book_masjid_id"`

	BookTitle  string  `gorm:"type:text;not null;column:book_title" json:"book_title"`
	BookAuthor *string `gorm:"type:text;column:book_author" json:"book_author,omitempty"`
	BookDesc   *string `gorm:"type:text;column:book_desc" json:"book_desc,omitempty"`

	BookSlug *string `gorm:"type:varchar(160);column:book_slug" json:"book_slug,omitempty"`

	BookPublisher       *string `gorm:"type:text;column:book_publisher" json:"book_publisher,omitempty"`
	BookPublicationYear *int16  `gorm:"type:smallint;column:book_publication_year" json:"book_publication_year,omitempty"`

	BookCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:book_created_at" json:"book_created_at"`
	BookUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:book_updated_at" json:"book_updated_at"`
	BookDeletedAt gorm.DeletedAt `gorm:"column:book_deleted_at;index" json:"book_deleted_at,omitempty"`
}

func (BookModel) TableName() string { return "books" }
