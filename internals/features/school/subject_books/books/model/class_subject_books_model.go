// internals/features/lembaga/class_subject_books/model/class_subject_book_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectBookModel struct {
	// PK
	ClassSubjectBooksID uuid.UUID `json:"class_subject_books_id" gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_subject_books_id"`

	// FKs
	ClassSubjectBooksMasjidID       uuid.UUID `json:"class_subject_books_masjid_id"        gorm:"type:uuid;not null;column:class_subject_books_masjid_id;index:idx_csb_masjid"`
	ClassSubjectBooksClassSubjectID uuid.UUID `json:"class_subject_books_class_subject_id" gorm:"type:uuid;not null;column:class_subject_books_class_subject_id;index:idx_csb_class_subject"`
	ClassSubjectBooksBookID         uuid.UUID `json:"class_subject_books_book_id"          gorm:"type:uuid;not null;column:class_subject_books_book_id;index:idx_csb_book"`

	// Status aktif (pengganti valid_from/valid_to)
	ClassSubjectBooksIsActive bool     `json:"class_subject_books_is_active" gorm:"not null;default:true;column:class_subject_books_is_active;index:idx_csb_active"`
	// Deskripsi (pengganti notes)
	ClassSubjectBooksDesc     *string  `json:"class_subject_books_desc,omitempty" gorm:"column:class_subject_books_desc"`

	// Timestamps
	ClassSubjectBooksCreatedAt time.Time      `json:"class_subject_books_created_at"           gorm:"column:class_subject_books_created_at;autoCreateTime"`
	ClassSubjectBooksUpdatedAt *time.Time     `json:"class_subject_books_updated_at,omitempty" gorm:"column:class_subject_books_updated_at;autoUpdateTime"`
	ClassSubjectBooksDeletedAt gorm.DeletedAt `json:"class_subject_books_deleted_at,omitempty" gorm:"column:class_subject_books_deleted_at;index"`
}

func (ClassSubjectBookModel) TableName() string { return "class_subject_books" }
