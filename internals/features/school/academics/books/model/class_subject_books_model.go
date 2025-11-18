// file: internals/features/library/books/model/class_subject_book_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectBookModel struct {
	/* ============ PK & Tenant ============ */
	ClassSubjectBookID       uuid.UUID `gorm:"column:class_subject_book_id;type:uuid;default:gen_random_uuid();primaryKey" json:"class_subject_book_id"`
	ClassSubjectBookSchoolID uuid.UUID `gorm:"column:class_subject_book_school_id;type:uuid;not null;index:idx_csb_school_alive" json:"class_subject_book_school_id"`

	/* ============ FK Relasi ============ */
	ClassSubjectBookClassSubjectID uuid.UUID `gorm:"column:class_subject_book_class_subject_id;type:uuid;not null;index:idx_csb_class_subject_alive" json:"class_subject_book_class_subject_id"`
	ClassSubjectBookBookID         uuid.UUID `gorm:"column:class_subject_book_book_id;type:uuid;not null;index:idx_csb_book_alive" json:"class_subject_book_book_id"`

	/* ============ Identitas & Flags ============ */
	ClassSubjectBookSlug       *string `gorm:"column:class_subject_book_slug;type:varchar(160)" json:"class_subject_book_slug,omitempty"`
	ClassSubjectBookIsPrimary  bool    `gorm:"column:class_subject_book_is_primary;type:boolean;not null;default:false" json:"class_subject_book_is_primary"`
	ClassSubjectBookIsRequired bool    `gorm:"column:class_subject_book_is_required;type:boolean;not null;default:true" json:"class_subject_book_is_required"`
	ClassSubjectBookOrder      *int    `gorm:"column:class_subject_book_order" json:"class_subject_book_order,omitempty"`

	ClassSubjectBookIsActive bool    `gorm:"column:class_subject_book_is_active;not null;default:true;index:idx_csb_active_alive" json:"class_subject_book_is_active"`
	ClassSubjectBookDesc     *string `gorm:"column:class_subject_book_desc;type:text" json:"class_subject_book_desc,omitempty"`

	/* ============ SNAPSHOTS dari books ============ */
	ClassSubjectBookBookTitleSnapshot           *string `gorm:"column:class_subject_book_book_title_snapshot" json:"class_subject_book_book_title_snapshot,omitempty"`
	ClassSubjectBookBookAuthorSnapshot          *string `gorm:"column:class_subject_book_book_author_snapshot" json:"class_subject_book_book_author_snapshot,omitempty"`
	ClassSubjectBookBookSlugSnapshot            *string `gorm:"column:class_subject_book_book_slug_snapshot;type:varchar(160)" json:"class_subject_book_book_slug_snapshot,omitempty"`
	ClassSubjectBookBookPublisherSnapshot       *string `gorm:"column:class_subject_book_book_publisher_snapshot" json:"class_subject_book_book_publisher_snapshot,omitempty"`
	ClassSubjectBookBookPublicationYearSnapshot *int16  `gorm:"column:class_subject_book_book_publication_year_snapshot" json:"class_subject_book_book_publication_year_snapshot,omitempty"`
	ClassSubjectBookBookImageURLSnapshot        *string `gorm:"column:class_subject_book_book_image_url_snapshot" json:"class_subject_book_book_image_url_snapshot,omitempty"`

	/* ============ SNAPSHOTS dari subjects ============ */
	ClassSubjectBookSubjectID   *uuid.UUID `gorm:"column:class_subject_book_subject_id;type:uuid" json:"class_subject_book_subject_id_snapshot,omitempty"`
	ClassSubjectBookSubjectCodeSnapshot *string    `gorm:"column:class_subject_book_subject_code_snapshot;type:varchar(40)" json:"class_subject_book_subject_code_snapshot,omitempty"`
	ClassSubjectBookSubjectNameSnapshot *string    `gorm:"column:class_subject_book_subject_name_snapshot;type:varchar(120)" json:"class_subject_book_subject_name_snapshot,omitempty"`
	ClassSubjectBookSubjectSlugSnapshot *string    `gorm:"column:class_subject_book_subject_slug_snapshot;type:varchar(160)" json:"class_subject_book_subject_slug_snapshot,omitempty"`

	/* ============ Audit ============ */
	ClassSubjectBookCreatedAt time.Time      `gorm:"column:class_subject_book_created_at;type:timestamptz;not null;default:now();autoCreateTime" json:"class_subject_book_created_at"`
	ClassSubjectBookUpdatedAt time.Time      `gorm:"column:class_subject_book_updated_at;type:timestamptz;not null;default:now();autoUpdateTime" json:"class_subject_book_updated_at"`
	ClassSubjectBookDeletedAt gorm.DeletedAt `gorm:"column:class_subject_book_deleted_at;index" json:"class_subject_book_deleted_at,omitempty"`
}

func (ClassSubjectBookModel) TableName() string { return "class_subject_books" }
