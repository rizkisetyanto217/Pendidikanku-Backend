// file: internals/features/library/books/snapshot/book_snapshot.go
package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================================================
   Single Book Snapshot (live from books)
   ========================================================= */

type BookSnapshot struct {
	Title           string
	Author          *string
	Slug            *string
	Publisher       *string
	PublicationYear *int16
	ImageURL        *string
	// tambah field lain jika perlu
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
		Where("book_id = ? AND book_deleted_at IS NULL", bookID).
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

/* =========================================================
   Class-Subject Books Snapshot (list → JSONB array)
   - Kumpulkan daftar buku aktif untuk 1 class_subject
   - COALESCE ke snapshot di CSB kalau ada, fallback ke books.*
   ========================================================= */

// struktur item untuk array snapshot (tidak diexport, supaya bebas ubah internal)
type bookListSnap struct {
	CSBID     uuid.UUID  `json:"csb_id"`
	BookID    uuid.UUID  `json:"book_id"`
	Title     string     `json:"title"`
	Author    *string    `json:"author,omitempty"`
	Slug      *string    `json:"slug,omitempty"`
	ImageURL  *string    `json:"image_url,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// tambahkan is_primary jika kamu punya kolom itu di CSB:
	// IsPrimary bool `json:"is_primary"`
}

// BuildBooksSnapshotJSON mengembalikan JSONB array berisi daftar buku aktif
// yang dipakai oleh class_subject tertentu (tenant-aware). Hasil cocok untuk
// disimpan di kolom class_section_subject_teacher_books_snapshot.
func BuildBooksSnapshotJSON(
	_ context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	classSubjectID uuid.UUID,
) (datatypes.JSON, error) {

	// Langsung pakai target struct-nya
	type row = bookListSnap

	var rows []row

	err := tx.
		Table("class_subject_books AS csb").
		Select(`
			csb.class_subject_book_id                                    AS csb_id,
			csb.class_subject_book_book_id                               AS book_id,
			COALESCE(csb.class_subject_book_book_title_snapshot, b.book_title)         AS title,
			COALESCE(csb.class_subject_book_book_author_snapshot, b.book_author)       AS author,
			COALESCE(csb.class_subject_book_book_slug_snapshot, b.book_slug)           AS slug,
			COALESCE(csb.class_subject_book_book_image_url_snapshot, b.book_image_url) AS image_url,
			csb.class_subject_book_created_at                             AS created_at
		`).
		Joins(`
			JOIN books b
			  ON b.book_id = csb.class_subject_book_book_id
			 AND b.book_school_id = csb.class_subject_book_school_id
			 AND b.book_deleted_at IS NULL
		`).
		Where(`
			csb.class_subject_book_school_id = ?
			AND csb.class_subject_book_class_subject_id = ?
			AND csb.class_subject_book_deleted_at IS NULL
			AND csb.class_subject_book_is_active = TRUE
		`, schoolID, classSubjectID).
		Order("csb.class_subject_book_created_at DESC").
		Scan(&rows).Error
	if err != nil {
		return datatypes.JSON{}, err
	}

	// rows sudah []bookListSnap, tinggal marshal
	b, err := json.Marshal(rows)
	if err != nil {
		return datatypes.JSON{}, err
	}
	return datatypes.JSON(b), nil
}
