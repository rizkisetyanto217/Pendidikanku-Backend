// internals/features/lembaga/class_books/dto/books_dto.go
package dto

import (
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/class_books/model"

	"github.com/google/uuid"
)

/* =========================
   REQUEST
   ========================= */

type BooksCreateRequest struct {
	BooksMasjidID uuid.UUID `json:"books_masjid_id" form:"books_masjid_id" validate:"required"`
	BooksTitle    string    `json:"books_title"     form:"books_title"     validate:"required,min=1"`

	BooksAuthor   *string `json:"books_author,omitempty"    form:"books_author"    validate:"omitempty,min=1"`
	BooksDesc     *string `json:"books_desc,omitempty"      form:"books_desc"      validate:"omitempty"`
	BooksURL      *string `json:"books_url,omitempty"       form:"books_url"       validate:"omitempty,url"`
	BooksImageURL *string `json:"books_image_url,omitempty" form:"books_image_url" validate:"omitempty,url"`

	// Opsional; controller bisa auto-generate dengan helper.GenerateSlug jika kosong
	BooksSlug *string `json:"books_slug,omitempty" form:"books_slug" validate:"omitempty,max=160"`
}

type BooksUpdateRequest struct {
	BooksTitle    *string `json:"books_title,omitempty"     form:"books_title"     validate:"omitempty,min=1"`
	BooksAuthor   *string `json:"books_author,omitempty"    form:"books_author"    validate:"omitempty,min=1"`
	BooksDesc     *string `json:"books_desc,omitempty"      form:"books_desc"      validate:"omitempty"`
	BooksURL      *string `json:"books_url,omitempty"       form:"books_url"       validate:"omitempty,url"`
	BooksImageURL *string `json:"books_image_url,omitempty" form:"books_image_url" validate:"omitempty,url"`
	BooksSlug     *string `json:"books_slug,omitempty"      form:"books_slug"      validate:"omitempty,max=160"`
}

// Query sederhana untuk listing
type BooksListQuery struct {
	Q        *string `query:"q"`                     // cari di title/author/desc (controller)
	Author   *string `query:"author"`                // filter exact/ilike author (controller)
	HasImage *bool   `query:"has_image"`             // true -> image_url IS NOT NULL
	HasURL   *bool   `query:"has_url"`               // true -> url IS NOT NULL
	OrderBy  *string `query:"order_by"`              // books_title|books_author|created_at
	Sort     *string `query:"sort"`                  // asc|desc
	Limit    *int    `query:"limit"`
	Offset   *int    `query:"offset"`
}


// ==== WITH-USAGES DTO ====

type BookUsageSectionLite struct {
	ClassSectionsID       uuid.UUID `json:"class_sections_id"`
	ClassSectionsName     string    `json:"class_sections_name"`
	ClassSectionsSlug     string    `json:"class_sections_slug"`
	ClassSectionsCode     *string   `json:"class_sections_code,omitempty"`
	ClassSectionsCapacity *int      `json:"class_sections_capacity,omitempty"`
	ClassSectionsIsActive bool      `json:"class_sections_is_active"`
}

type BookUsage struct {
	ClassSubjectBooksID uuid.UUID  `json:"class_subject_books_id"`
	ClassSubjectID      *uuid.UUID `json:"class_subjects_id,omitempty"`
	SubjectsID          *uuid.UUID `json:"subjects_id,omitempty"`
	ClassesID           *uuid.UUID `json:"classes_id,omitempty"`
	Sections            []BookUsageSectionLite `json:"sections"`
}

type BookWithUsagesResponse struct {
	BooksID       uuid.UUID `json:"books_id"`
	BooksMasjidID uuid.UUID `json:"books_masjid_id"`

	BooksTitle    string  `json:"books_title"`
	BooksAuthor   *string `json:"books_author,omitempty"`
	BooksDesc     *string `json:"books_desc,omitempty"`
	BooksURL      *string `json:"books_url,omitempty"`
	BooksImageURL *string `json:"books_image_url,omitempty"`
	BooksSlug     *string `json:"books_slug,omitempty"`

	Usages []BookUsage `json:"usages"`
}

// opsional: query yang sama dengan list biasa, bisa dipakai kembali
// tambahkan with_deleted bila perlu
type BooksWithUsagesListQuery struct {
	Q          *string `query:"q"`
	Author     *string `query:"author"`
	HasImage   *bool   `query:"has_image"`
	HasURL     *bool   `query:"has_url"`
	OrderBy    *string `query:"order_by"` // books_title|books_author|created_at
	Sort       *string `query:"sort"`     // asc|desc
	Limit      *int    `query:"limit"`
	Offset     *int    `query:"offset"`
	WithDeleted *bool  `query:"with_deleted"`
}

/* =========================
   RESPONSE
   ========================= */

type BooksResponse struct {
	BooksID       uuid.UUID `json:"books_id"`
	BooksMasjidID uuid.UUID `json:"books_masjid_id"`

	BooksTitle    string  `json:"books_title"`
	BooksAuthor   *string `json:"books_author,omitempty"`
	BooksDesc     *string `json:"books_desc,omitempty"`
	BooksURL      *string `json:"books_url,omitempty"`
	BooksImageURL *string `json:"books_image_url,omitempty"`
	BooksSlug     *string `json:"books_slug,omitempty"`

	BooksCreatedAt int64  `json:"books_created_at_unix"`
	BooksUpdatedAt *int64 `json:"books_updated_at_unix,omitempty"`
	BooksDeleted   bool   `json:"books_is_deleted"`
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

/* =========================
   NORMALIZER
   ========================= */

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func (r *BooksCreateRequest) Normalize() {
	r.BooksTitle = strings.TrimSpace(r.BooksTitle)
	r.BooksAuthor = trimPtr(r.BooksAuthor)
	r.BooksDesc = trimPtr(r.BooksDesc)
	r.BooksURL = trimPtr(r.BooksURL)
	r.BooksImageURL = trimPtr(r.BooksImageURL)
	r.BooksSlug = trimPtr(r.BooksSlug) // controller boleh GenerateSlug jika nil
}

func (r *BooksUpdateRequest) Normalize() {
	r.BooksTitle = trimPtr(r.BooksTitle)
	r.BooksAuthor = trimPtr(r.BooksAuthor)
	r.BooksDesc = trimPtr(r.BooksDesc)
	r.BooksURL = trimPtr(r.BooksURL)
	r.BooksImageURL = trimPtr(r.BooksImageURL)
	r.BooksSlug = trimPtr(r.BooksSlug)
}

/* =========================
   MAPPER
   ========================= */

func ToBooksResponse(m *model.BooksModel) BooksResponse {
	resp := BooksResponse{
		BooksID:       m.BooksID,
		BooksMasjidID: m.BooksMasjidID,
		BooksTitle:    m.BooksTitle,
		BooksAuthor:   m.BooksAuthor,
		BooksDesc:     m.BooksDesc,
		BooksURL:      m.BooksURL,
		BooksImageURL: m.BooksImageURL,
		BooksSlug:     m.BooksSlug,
		BooksCreatedAt: m.BooksCreatedAt.Unix(),
		BooksDeleted:   !m.BooksDeletedAt.Time.IsZero(),
	}
	if m.BooksUpdatedAt != nil && !m.BooksUpdatedAt.IsZero() {
		u := m.BooksUpdatedAt.Unix()
		resp.BooksUpdatedAt = &u
	}
	return resp
}

func (r *BooksCreateRequest) ToModel() *model.BooksModel {
	now := time.Now()
	return &model.BooksModel{
		BooksMasjidID:  r.BooksMasjidID,
		BooksTitle:     r.BooksTitle,
		BooksAuthor:    r.BooksAuthor,
		BooksDesc:      r.BooksDesc,
		BooksURL:       r.BooksURL,
		BooksImageURL:  r.BooksImageURL,
		BooksSlug:      r.BooksSlug, // controller boleh override dengan GenerateSlug
		BooksCreatedAt: now,
	}
}

func (r *BooksUpdateRequest) ApplyToModel(m *model.BooksModel) {
	if r.BooksTitle != nil {
		m.BooksTitle = *r.BooksTitle
	}
	if r.BooksAuthor != nil {
		m.BooksAuthor = r.BooksAuthor
	}
	if r.BooksDesc != nil {
		m.BooksDesc = r.BooksDesc
	}
	if r.BooksURL != nil {
		m.BooksURL = r.BooksURL
	}
	if r.BooksImageURL != nil {
		m.BooksImageURL = r.BooksImageURL
	}
	if r.BooksSlug != nil {
		m.BooksSlug = r.BooksSlug
	}
	// updated_at akan diisi otomatis oleh GORM (autoUpdateTime)
}
