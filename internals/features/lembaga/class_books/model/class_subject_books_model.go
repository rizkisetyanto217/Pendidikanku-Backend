// internals/features/lembaga/class_subject_books/model/class_subject_book_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type ClassSubjectBookModel struct {
	// PK
	ClassSubjectBooksID uuid.UUID `json:"class_subject_books_id" gorm:"column:class_subject_books_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// FKs
	ClassSubjectBooksMasjidID       uuid.UUID `json:"class_subject_books_masjid_id"        gorm:"column:class_subject_books_masjid_id;type:uuid;not null"`
	ClassSubjectBooksClassSubjectID uuid.UUID `json:"class_subject_books_class_subject_id" gorm:"column:class_subject_books_class_subject_id;type:uuid;not null"`
	ClassSubjectBooksBookID         uuid.UUID `json:"class_subject_books_book_id"          gorm:"column:class_subject_books_book_id;type:uuid;not null"`

	// Periode pemakaian (nullable)
	ClassSubjectBooksValidFrom *time.Time `json:"valid_from,omitempty" gorm:"column:valid_from;type:date"`
	ClassSubjectBooksValidTo   *time.Time `json:"valid_to,omitempty"   gorm:"column:valid_to;type:date"`

	// Penandaan
	ClassSubjectBooksIsPrimary bool    `json:"is_primary"      gorm:"column:is_primary;not null;default:false"`
	ClassSubjectBooksNotes     *string `json:"notes,omitempty" gorm:"column:notes"`

	// Timestamps
	ClassSubjectBooksCreatedAt time.Time  `json:"class_subject_books_created_at"           gorm:"column:class_subject_books_created_at;not null;default:now()"`
	ClassSubjectBooksUpdatedAt *time.Time `json:"class_subject_books_updated_at,omitempty" gorm:"column:class_subject_books_updated_at"`
	ClassSubjectBooksDeletedAt *time.Time `json:"class_subject_books_deleted_at,omitempty" gorm:"column:class_subject_books_deleted_at;index"`
}

func (ClassSubjectBookModel) TableName() string { return "class_subject_books" }
