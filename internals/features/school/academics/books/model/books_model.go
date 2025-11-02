// file: internals/features/library/books/model/book_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookModel struct {
	BookID       uuid.UUID `gorm:"column:book_id;type:uuid;default:gen_random_uuid();primaryKey" json:"book_id"`
	BookSchoolID uuid.UUID `gorm:"column:book_school_id;type:uuid;not null"                      json:"book_school_id"`

	BookTitle  string  `gorm:"column:book_title;type:text;not null" json:"book_title"`
	BookAuthor *string `gorm:"column:book_author;type:text"         json:"book_author,omitempty"`
	BookDesc   *string `gorm:"column:book_desc;type:text"           json:"book_desc,omitempty"`

	BookSlug *string `gorm:"column:book_slug;type:varchar(160)" json:"book_slug,omitempty"`

	// lokasi file/link (sinkron dengan DDL)
	BookImageURL                *string    `gorm:"column:book_image_url;type:text"                    json:"book_image_url,omitempty"`
	BookImageObjectKey          *string    `gorm:"column:book_image_object_key;type:text"             json:"book_image_object_key,omitempty"`
	BookImageURLOld             *string    `gorm:"column:book_image_url_old;type:text"                json:"book_image_url_old,omitempty"`
	BookImageObjectKeyOld       *string    `gorm:"column:book_image_object_key_old;type:text"         json:"book_image_object_key_old,omitempty"`
	BookImageDeletePendingUntil *time.Time `gorm:"column:book_image_delete_pending_until;type:timestamptz" json:"book_image_delete_pending_until,omitempty"`

	// bibliographic (opsional)
	BookPublisher       *string `gorm:"column:book_publisher;type:text"         json:"book_publisher,omitempty"`
	BookPublicationYear *int16  `gorm:"column:book_publication_year;type:smallint" json:"book_publication_year,omitempty"`

	// timestamps (explicit)
	BookCreatedAt time.Time      `gorm:"column:book_created_at;type:timestamptz;not null;default:now()" json:"book_created_at"`
	BookUpdatedAt time.Time      `gorm:"column:book_updated_at;type:timestamptz;not null;default:now()" json:"book_updated_at"`
	BookDeletedAt gorm.DeletedAt `gorm:"column:book_deleted_at;index"                                   json:"book_deleted_at,omitempty"`
}

func (BookModel) TableName() string { return "books" }
