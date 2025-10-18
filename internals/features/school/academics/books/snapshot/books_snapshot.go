// file: internals/features/library/books/snapshot/book_snapshot.go
package snapshot

import (
	"errors"

	"gorm.io/gorm"

	"github.com/google/uuid"
)

type BookSnapshot struct {
	Title           string
	Author          *string
	Slug            *string
	Publisher       *string
	PublicationYear *int16
	ImageURL        *string
	// bisa ditambah field lain jika perlu
}

func FetchBookSnapshot(tx *gorm.DB, bookID uuid.UUID) (*BookSnapshot, error) {
	if tx == nil {
		return nil, errors.New("nil tx")
	}
	var row struct {
		Title           string  `gorm:"column:book_title"`
		Author          *string `gorm:"column:book_author"`
		Slug            *string `gorm:"column:book_slug"`
		Publisher       *string `gorm:"column:book_publisher"`
		PublicationYear *int16  `gorm:"column:book_publication_year"`
		ImageURL        *string `gorm:"column:book_image_url"`
	}
	if err := tx.Table("books").
		Select(`book_title, book_author, book_slug, book_publisher, book_publication_year, book_image_url`).
		Where("book_id = ?", bookID).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &BookSnapshot{
		Title:           row.Title,
		Author:          row.Author,
		Slug:            row.Slug,
		Publisher:       row.Publisher,
		PublicationYear: row.PublicationYear,
		ImageURL:        row.ImageURL,
	}, nil
}
