// internals/features/lembaga/class_subject_books/model/class_subject_book_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectBookModel struct {
	// PK
	ClassSubjectBooksID uuid.UUID `json:"class_subject_books_id" gorm:"column:class_subject_books_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// FKs
	ClassSubjectBooksMasjidID       uuid.UUID `json:"class_subject_books_masjid_id"        gorm:"column:class_subject_books_masjid_id;type:uuid;not null;index:idx_csb_masjid"`
	ClassSubjectBooksClassSubjectID uuid.UUID `json:"class_subject_books_class_subject_id" gorm:"column:class_subject_books_class_subject_id;type:uuid;not null;index:idx_csb_class_subject"`
	ClassSubjectBooksBookID         uuid.UUID `json:"class_subject_books_book_id"          gorm:"column:class_subject_books_book_id;type:uuid;not null;index:idx_csb_book"`

	// SLUG (opsional; unik per tenant saat alive)
	ClassSubjectBooksSlug *string `json:"class_subject_books_slug,omitempty" gorm:"column:class_subject_books_slug;type:varchar(160);index:uq_csb_slug_per_tenant_alive,unique,expression:lower(class_subject_books_slug),where:class_subject_books_deleted_at IS NULL"`

	// Status & desc
	ClassSubjectBooksIsActive bool    `json:"class_subject_books_is_active" gorm:"column:class_subject_books_is_active;not null;default:true;index:idx_csb_active_alive,where:class_subject_books_deleted_at IS NULL"`
	ClassSubjectBooksDesc     *string `json:"class_subject_books_desc,omitempty" gorm:"column:class_subject_books_desc"`

	// Timestamps (UPDATED_AT NOT NULL di SQL → pakai time.Time, bukan pointer)
	ClassSubjectBooksCreatedAt time.Time      `json:"class_subject_books_created_at" gorm:"column:class_subject_books_created_at;type:timestamptz;not null;autoCreateTime"`
	ClassSubjectBooksUpdatedAt *time.Time      `json:"class_subject_books_updated_at" gorm:"column:class_subject_books_updated_at;type:timestamptz;not null;autoUpdateTime"`
	ClassSubjectBooksDeletedAt gorm.DeletedAt `json:"class_subject_books_deleted_at,omitempty" gorm:"column:class_subject_books_deleted_at;index"`

	// (Opsional) Relasi — uncomment & sesuaikan import path jika mau preload
	// ClassSubject ClassSubjectModel `gorm:"foreignKey:ClassSubjectBooksClassSubjectID;references:ClassSubjectsID;constraint:OnDelete:CASCADE" json:"-"`
	// Book         BooksModel        `gorm:"foreignKey:ClassSubjectBooksBookID;references:BooksID;constraint:OnDelete:RESTRICT"     json:"-"`
}

func (ClassSubjectBookModel) TableName() string { return "class_subject_books" }
