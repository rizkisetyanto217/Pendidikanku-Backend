// internals/features/lembaga/class_subject_books/dto/class_subject_book_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/academics/books/model"

	"github.com/google/uuid"
)

/* =========================================================
   1) REQUESTS
   ========================================================= */

// Create
type CreateClassSubjectBookRequest struct {
	ClassSubjectBooksMasjidID       uuid.UUID `json:"class_subject_books_masjid_id"        validate:"required"`
	ClassSubjectBooksClassSubjectID uuid.UUID `json:"class_subject_books_class_subject_id" validate:"required"`
	ClassSubjectBooksBookID         uuid.UUID `json:"class_subject_books_book_id"          validate:"required"`

	// slug opsional; controller yang normalize + ensure-unique
	ClassSubjectBooksSlug *string `json:"class_subject_books_slug" validate:"omitempty,max=160"`

	ClassSubjectBooksIsActive *bool   `json:"class_subject_books_is_active" validate:"omitempty"`
	ClassSubjectBooksDesc     *string `json:"class_subject_books_desc"      validate:"omitempty,max=2000"`
}

type SectionLite struct {
	ClassSectionsID       uuid.UUID `json:"class_sections_id"`
	ClassSectionsName     string    `json:"class_sections_name"`
	ClassSectionsSlug     string    `json:"class_sections_slug"`
	ClassSectionsCode     *string   `json:"class_sections_code,omitempty"`
	ClassSectionsCapacity *int      `json:"class_sections_capacity,omitempty"`
	ClassSectionsIsActive bool      `json:"class_sections_is_active"`
}

func (r CreateClassSubjectBookRequest) ToModel() model.ClassSubjectBookModel {
	isActive := true
	if r.ClassSubjectBooksIsActive != nil {
		isActive = *r.ClassSubjectBooksIsActive
	}
	var desc *string
	if r.ClassSubjectBooksDesc != nil {
		d := strings.TrimSpace(*r.ClassSubjectBooksDesc)
		if d != "" {
			desc = &d
		}
	}
	var slug *string
	if r.ClassSubjectBooksSlug != nil {
		s := strings.TrimSpace(*r.ClassSubjectBooksSlug)
		if s != "" {
			slug = &s
		}
	}

	return model.ClassSubjectBookModel{
		ClassSubjectBookMasjidID:       r.ClassSubjectBooksMasjidID,
		ClassSubjectBookClassSubjectID: r.ClassSubjectBooksClassSubjectID,
		ClassSubjectBookBookID:         r.ClassSubjectBooksBookID,
		ClassSubjectBookSlug:           slug,
		ClassSubjectBookIsActive:       isActive,
		ClassSubjectBookDesc:           desc,
	}
}

// Update (partial)
type UpdateClassSubjectBookRequest struct {
	ClassSubjectBooksMasjidID       *uuid.UUID `json:"class_subject_books_masjid_id"        validate:"omitempty"`
	ClassSubjectBooksClassSubjectID *uuid.UUID `json:"class_subject_books_class_subject_id" validate:"omitempty"`
	ClassSubjectBooksBookID         *uuid.UUID `json:"class_subject_books_book_id"          validate:"omitempty"`

	// slug opsional; controller yang normalize + ensure-unique (+ exclude diri sendiri)
	ClassSubjectBooksSlug *string `json:"class_subject_books_slug" validate:"omitempty,max=160"`

	ClassSubjectBooksIsActive *bool   `json:"class_subject_books_is_active" validate:"omitempty"`
	ClassSubjectBooksDesc     *string `json:"class_subject_books_desc"      validate:"omitempty,max=2000"`
}

func (r *UpdateClassSubjectBookRequest) Apply(m *model.ClassSubjectBookModel) error {
	if m == nil {
		return errors.New("nil model")
	}
	if r.ClassSubjectBooksMasjidID != nil {
		m.ClassSubjectBookMasjidID = *r.ClassSubjectBooksMasjidID
	}
	if r.ClassSubjectBooksClassSubjectID != nil {
		m.ClassSubjectBookClassSubjectID = *r.ClassSubjectBooksClassSubjectID
	}
	if r.ClassSubjectBooksBookID != nil {
		m.ClassSubjectBookBookID = *r.ClassSubjectBooksBookID
	}
	if r.ClassSubjectBooksIsActive != nil {
		m.ClassSubjectBookIsActive = *r.ClassSubjectBooksIsActive
	}
	if r.ClassSubjectBooksDesc != nil {
		d := strings.TrimSpace(*r.ClassSubjectBooksDesc)
		if d == "" {
			m.ClassSubjectBookDesc = nil
		} else {
			m.ClassSubjectBookDesc = &d
		}
	}
	// slug: normalize di DTO; ensure-unique di controller
	if r.ClassSubjectBooksSlug != nil {
		s := strings.TrimSpace(*r.ClassSubjectBooksSlug)
		if s == "" {
			m.ClassSubjectBookSlug = nil
		} else {
			m.ClassSubjectBookSlug = &s
		}
	}
	// UpdatedAt biar diisi GORM/DB
	return nil
}

/* =========================================================
   2) LIST QUERY
   ========================================================= */

type ListClassSubjectBookQuery struct {
	Limit          *int       `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset         *int       `query:"offset" validate:"omitempty,min=0"`
	ClassSubjectID *uuid.UUID `query:"class_subject_id" validate:"omitempty"`
	BookID         *uuid.UUID `query:"book_id" validate:"omitempty"`
	IsActive       *bool      `query:"is_active" validate:"omitempty"`
	WithDeleted    *bool      `query:"with_deleted" validate:"omitempty"`
	Sort           *string    `query:"sort" validate:"omitempty,oneof=created_at_asc created_at_desc updated_at_asc updated_at_desc"`
	Q              *string    `query:"q" validate:"omitempty,max=100"`
}

/* =========================================================
   3) RESPONSE
   ========================================================= */

// BookURLLite: contoh embed URL buku
type BookURLLite struct {
	BookURLID                 uuid.UUID  `json:"book_url_id"`
	BookURLMasjidID           uuid.UUID  `json:"book_url_masjid_id"`
	BookURLBookID             uuid.UUID  `json:"book_url_book_id"`
	BookURLLabel              *string    `json:"book_url_label,omitempty"`
	BookURLType               string     `json:"book_url_type"`
	BookURLHref               string     `json:"book_url_href"`
	BookURLTrashURL           *string    `json:"book_url_trash_url,omitempty"`
	BookURLDeletePendingUntil *time.Time `json:"book_url_delete_pending_until,omitempty"`
	BookURLCreatedAt          time.Time  `json:"book_url_created_at"`
	BookURLUpdatedAt          time.Time  `json:"book_url_updated_at"`
	BookURLDeletedAt          *time.Time `json:"book_url_deleted_at,omitempty"`
	BookURLIsPrimary    bool      `json:"book_url_is_primary"`
	BookURLOrder        int       `json:"book_url_order"`
	BookURLKind         string    `json:"book_url_kind"`
}

// BookLite (opsional) — dilengkapi daftar URLs
type BookLite struct {
	BooksID       uuid.UUID     `json:"books_id"`
	BooksMasjidID uuid.UUID     `json:"books_masjid_id"`
	BooksTitle    string        `json:"books_title"`
	BooksAuthor   *string       `json:"books_author,omitempty"`
	BooksURL      *string       `json:"books_url,omitempty"`
	BooksImageURL *string       `json:"books_image_url,omitempty"`
	BooksSlug     *string       `json:"books_slug,omitempty"`
	BookURLs      []BookURLLite `json:"book_urls,omitempty"`
}

type ClassSubjectBookResponse struct {
	ClassSubjectBooksID             uuid.UUID `json:"class_subject_books_id"`
	ClassSubjectBooksMasjidID       uuid.UUID `json:"class_subject_books_masjid_id"`
	ClassSubjectBooksClassSubjectID uuid.UUID `json:"class_subject_books_class_subject_id"`
	ClassSubjectBooksBookID         uuid.UUID `json:"class_subject_books_book_id"`

	// slug ikut dibalas
	ClassSubjectBooksSlug *string `json:"class_subject_books_slug,omitempty"`

	ClassSubjectBooksIsActive bool    `json:"class_subject_books_is_active"`
	ClassSubjectBooksDesc     *string `json:"class_subject_books_desc,omitempty"`

	ClassSubjectBooksCreatedAt time.Time  `json:"class_subject_books_created_at"`
	ClassSubjectBooksUpdatedAt time.Time  `json:"class_subject_books_updated_at"` // NOT NULL di model
	ClassSubjectBooksDeletedAt *time.Time `json:"class_subject_books_deleted_at,omitempty"`

	// opsional join
	Book    *BookLite    `json:"book,omitempty"`
	Section *SectionLite `json:"section,omitempty"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type ClassSubjectBookListResponse struct {
	Items      []ClassSubjectBookResponse `json:"items"`
	Pagination Pagination                 `json:"pagination"`
}

/* =========================================================
   4) MAPPERS
   ========================================================= */

func FromModel(m model.ClassSubjectBookModel) ClassSubjectBookResponse {
	var deletedAt *time.Time
	if m.ClassSubjectBookDeletedAt.Valid {
		deletedAt = &m.ClassSubjectBookDeletedAt.Time
	}
	return ClassSubjectBookResponse{
		ClassSubjectBooksID:             m.ClassSubjectBookID,
		ClassSubjectBooksMasjidID:       m.ClassSubjectBookMasjidID,
		ClassSubjectBooksClassSubjectID: m.ClassSubjectBookClassSubjectID,
		ClassSubjectBooksBookID:         m.ClassSubjectBookBookID,
		ClassSubjectBooksSlug:           m.ClassSubjectBookSlug,
		ClassSubjectBooksIsActive:       m.ClassSubjectBookIsActive,
		ClassSubjectBooksDesc:           m.ClassSubjectBookDesc,
		ClassSubjectBooksCreatedAt:      m.ClassSubjectBookCreatedAt,
		ClassSubjectBooksUpdatedAt:      m.ClassSubjectBookUpdatedAt,
		ClassSubjectBooksDeletedAt:      deletedAt,
	}
}

func FromModels(list []model.ClassSubjectBookModel) []ClassSubjectBookResponse {
	out := make([]ClassSubjectBookResponse, 0, len(list))
	for _, it := range list {
		out = append(out, FromModel(it))
	}
	return out
}

// (Opsional) helper kalau controller sudah punya kolom join "books_*"
func WithBook(resp ClassSubjectBookResponse, b *BookLite) ClassSubjectBookResponse {
	resp.Book = b
	return resp
}
