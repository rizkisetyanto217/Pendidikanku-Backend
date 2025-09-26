// file: internals/features/library/books/model/class_subject_book_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectBookModel struct {
	ClassSubjectBookID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_subject_book_id" json:"class_subject_book_id"`
	ClassSubjectBookMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_subject_book_masjid_id" json:"class_subject_book_masjid_id"`

	ClassSubjectBookClassSubjectID uuid.UUID `gorm:"type:uuid;not null;column:class_subject_book_class_subject_id" json:"class_subject_book_class_subject_id"`
	ClassSubjectBookBookID         uuid.UUID `gorm:"type:uuid;not null;column:class_subject_book_book_id" json:"class_subject_book_book_id"`

	ClassSubjectBookSlug     *string `gorm:"type:varchar(160);column:class_subject_book_slug" json:"class_subject_book_slug,omitempty"`
	ClassSubjectBookIsActive bool    `gorm:"not null;default:true;column:class_subject_book_is_active" json:"class_subject_book_is_active"`
	ClassSubjectBookDesc     *string `gorm:"type:text;column:class_subject_book_desc" json:"class_subject_book_desc,omitempty"`

	ClassSubjectBookCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_subject_book_created_at" json:"class_subject_book_created_at"`
	ClassSubjectBookUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_subject_book_updated_at" json:"class_subject_book_updated_at"`
	ClassSubjectBookDeletedAt gorm.DeletedAt `gorm:"column:class_subject_book_deleted_at;index" json:"class_subject_book_deleted_at,omitempty"`
}

func (ClassSubjectBookModel) TableName() string { return "class_subject_books" }
