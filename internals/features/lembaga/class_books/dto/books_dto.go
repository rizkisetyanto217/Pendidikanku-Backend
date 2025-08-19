// internals/features/lembaga/class_books/dto/books_dto.go
package dto

import (
	"strings"
	"time"

	"masjidku_backend/internals/features/lembaga/class_books/model"

	"github.com/google/uuid"
)

// ======================================================
// REQUEST
// ======================================================

type BooksCreateRequest struct {
	BooksMasjidID      uuid.UUID `json:"books_masjid_id" validate:"required"`
	BooksTitle         string    `json:"books_title" validate:"required,min=1"`
	BooksAuthor        *string   `json:"books_author,omitempty" validate:"omitempty,min=1"`
	BooksEdition       *string   `json:"books_edition,omitempty" validate:"omitempty,min=1"`
	BooksPublisher     *string   `json:"books_publisher,omitempty" validate:"omitempty,min=1"`
	BooksISBN          *string   `json:"books_isbn,omitempty" validate:"omitempty,ascii,max=32"`
	BooksYear          *int      `json:"books_year,omitempty" validate:"omitempty,gte=1000,lte=3000"`
	BooksURL           *string   `json:"books_url,omitempty" validate:"omitempty,url"`
	BooksImageURL      *string   `json:"books_image_url,omitempty" validate:"omitempty,url"`
	BooksImageThumbURL *string   `json:"books_image_thumb_url,omitempty" validate:"omitempty,url"`
}

type BooksUpdateRequest struct {
	BooksTitle         *string `json:"books_title,omitempty" validate:"omitempty,min=1"`
	BooksAuthor        *string `json:"books_author,omitempty" validate:"omitempty,min=1"`
	BooksEdition       *string `json:"books_edition,omitempty" validate:"omitempty,min=1"`
	BooksPublisher     *string `json:"books_publisher,omitempty" validate:"omitempty,min=1"`
	BooksISBN          *string `json:"books_isbn,omitempty" validate:"omitempty,ascii,max=32"`
	BooksYear          *int    `json:"books_year,omitempty" validate:"omitempty,gte=1000,lte=3000"`
	BooksURL           *string `json:"books_url,omitempty" validate:"omitempty,url"`
	BooksImageURL      *string `json:"books_image_url,omitempty" validate:"omitempty,url"`
	BooksImageThumbURL *string `json:"books_image_thumb_url,omitempty" validate:"omitempty,url"`
}

type BooksListQuery struct {
	Q         *string `query:"q"`
	Publisher *string `query:"publisher"`
	YearMin   *int    `query:"year_min"`
	YearMax   *int    `query:"year_max"`
	HasImage  *bool   `query:"has_image"`
	OrderBy   *string `query:"order_by"` // books_title|books_year|created_at
	Sort      *string `query:"sort"`     // asc|desc
	Limit     *int    `query:"limit"`
	Offset    *int    `query:"offset"`
}

// ======================================================
// RESPONSE
// ======================================================

type BooksResponse struct {
	BooksID            uuid.UUID  `json:"books_id"`
	BooksMasjidID      uuid.UUID  `json:"books_masjid_id"`

	BooksTitle         string     `json:"books_title"`
	BooksAuthor        *string    `json:"books_author,omitempty"`
	BooksEdition       *string    `json:"books_edition,omitempty"`
	BooksPublisher     *string    `json:"books_publisher,omitempty"`
	BooksISBN          *string    `json:"books_isbn,omitempty"`
	BooksYear          *int       `json:"books_year,omitempty"`
	BooksURL           *string    `json:"books_url,omitempty"`
	BooksImageURL      *string    `json:"books_image_url,omitempty"`
	BooksImageThumbURL *string    `json:"books_image_thumb_url,omitempty"`

	BooksCreatedAt     int64      `json:"books_created_at_unix"`
	BooksUpdatedAt     *int64     `json:"books_updated_at_unix,omitempty"`
	BooksDeleted       bool       `json:"books_is_deleted"`
}

type PageInfo struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type BooksListResponse struct {
	Data []BooksResponse `json:"data"`
	Page PageInfo        `json:"page"`
}

// ======================================================
// NORMALIZER
// ======================================================

func (r *BooksCreateRequest) Normalize() {
	r.BooksTitle = strings.TrimSpace(r.BooksTitle)
	if r.BooksEdition != nil {
		ed := strings.TrimSpace(*r.BooksEdition)
		r.BooksEdition = &ed
	}
}
func (r *BooksUpdateRequest) Normalize() {
	if r.BooksTitle != nil {
		t := strings.TrimSpace(*r.BooksTitle)
		r.BooksTitle = &t
	}
	if r.BooksEdition != nil {
		ed := strings.TrimSpace(*r.BooksEdition)
		r.BooksEdition = &ed
	}
}

// ======================================================
// MAPPER
// ======================================================

func ToBooksResponse(m *model.BooksModel) BooksResponse {
	resp := BooksResponse{
		BooksID:            m.BooksID,
		BooksMasjidID:      m.BooksMasjidID,
		BooksTitle:         m.BooksTitle,
		BooksAuthor:        m.BooksAuthor,
		BooksEdition:       m.BooksEdition,
		BooksPublisher:     m.BooksPublisher,
		BooksISBN:          m.BooksISBN,
		BooksYear:          m.BooksYear,
		BooksURL:           m.BooksURL,
		BooksImageURL:      m.BooksImageURL,
		BooksImageThumbURL: m.BooksImageThumbURL,
		BooksCreatedAt:     m.BooksCreatedAt.Unix(),
		BooksDeleted:       !m.BooksDeletedAt.Time.IsZero(),
	}
	if !m.BooksUpdatedAt.IsZero() {
		u := m.BooksUpdatedAt.Unix()
		resp.BooksUpdatedAt = &u
	}
	return resp
}

func (r *BooksCreateRequest) ToModel() *model.BooksModel {
	now := time.Now()
	return &model.BooksModel{
		BooksMasjidID:      r.BooksMasjidID,
		BooksTitle:         r.BooksTitle,
		BooksAuthor:        r.BooksAuthor,
		BooksEdition:       r.BooksEdition,
		BooksPublisher:     r.BooksPublisher,
		BooksISBN:          r.BooksISBN,
		BooksYear:          r.BooksYear,
		BooksURL:           r.BooksURL,
		BooksImageURL:      r.BooksImageURL,
		BooksImageThumbURL: r.BooksImageThumbURL,
		BooksCreatedAt:     now,
	}
}

func (r *BooksUpdateRequest) ApplyToModel(m *model.BooksModel) {
	if r.BooksTitle != nil {
		m.BooksTitle = *r.BooksTitle
	}
	if r.BooksAuthor != nil {
		m.BooksAuthor = r.BooksAuthor
	}
	if r.BooksEdition != nil {
		m.BooksEdition = r.BooksEdition
	}
	if r.BooksPublisher != nil {
		m.BooksPublisher = r.BooksPublisher
	}
	if r.BooksISBN != nil {
		m.BooksISBN = r.BooksISBN
	}
	if r.BooksYear != nil {
		m.BooksYear = r.BooksYear
	}
	if r.BooksURL != nil {
		m.BooksURL = r.BooksURL
	}
	if r.BooksImageURL != nil {
		m.BooksImageURL = r.BooksImageURL
	}
	if r.BooksImageThumbURL != nil {
		m.BooksImageThumbURL = r.BooksImageThumbURL
	}
}
