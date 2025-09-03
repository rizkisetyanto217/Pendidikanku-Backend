// internals/features/lembaga/class_books/dto/books_dto.go
package dto

import (
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/subject_books/books/model"

	"github.com/google/uuid"
)

/* =========================
   REQUEST
   ========================= */

type BooksCreateRequest struct {
	BooksMasjidID uuid.UUID `json:"books_masjid_id" form:"books_masjid_id" validate:"required"`
	BooksTitle    string    `json:"books_title"     form:"books_title"     validate:"required,min=1"`

	BooksAuthor *string `json:"books_author,omitempty" form:"books_author" validate:"omitempty,min=1"`
	BooksDesc   *string `json:"books_desc,omitempty"   form:"books_desc"   validate:"omitempty"`

	// Opsional; controller bisa auto-generate dengan helper.GenerateSlug jika kosong
	BooksSlug *string `json:"books_slug,omitempty" form:"books_slug" validate:"omitempty,max=160"`
}

type BooksUpdateRequest struct {
	BooksTitle  *string `json:"books_title,omitempty"  form:"books_title"  validate:"omitempty,min=1"`
	BooksAuthor *string `json:"books_author,omitempty" form:"books_author" validate:"omitempty,min=1"`
	BooksDesc   *string `json:"books_desc,omitempty"   form:"books_desc"   validate:"omitempty"`
	BooksSlug   *string `json:"books_slug,omitempty"   form:"books_slug"   validate:"omitempty,max=160"`
}

// Query sederhana untuk listing
type BooksListQuery struct {
	Q       *string `query:"q"`          // cari di title/author/desc (controller)
	Author  *string `query:"author"`     // filter exact/ilike author (controller)
	OrderBy *string `query:"order_by"`   // books_title|books_author|created_at
	Sort    *string `query:"sort"`       // asc|desc
	Limit   *int    `query:"limit"`
	Offset  *int    `query:"offset"`
}


/* ==== WITH-USAGES DTO ==== */

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

	BooksTitle  string  `json:"books_title"`
	BooksAuthor *string `json:"books_author,omitempty"`
	BooksDesc   *string `json:"books_desc,omitempty"`
	BooksSlug   *string `json:"books_slug,omitempty"`
	BooksURL      *string `json:"books_url,omitempty"`
	BooksImageURL *string `json:"books_image_url,omitempty"`
	Usages []BookUsage `json:"usages"`
}

// opsional: query yang sama dengan list biasa, bisa dipakai kembali
// tambahkan with_deleted bila perlu
type BooksWithUsagesListQuery struct {
	Q           *string `query:"q"`
	Author      *string `query:"author"`
	OrderBy     *string `query:"order_by"` // books_title|books_author|created_at
	Sort        *string `query:"sort"`     // asc|desc
	Limit       *int    `query:"limit"`
	Offset      *int    `query:"offset"`
	WithDeleted *bool   `query:"with_deleted"`
}


/* =========================
   RESPONSE
   ========================= */

// internals/features/lembaga/class_books/dto/books_dto.go

type BooksResponse struct {
	BooksID       uuid.UUID `json:"books_id"`
	BooksMasjidID uuid.UUID `json:"books_masjid_id"`

	BooksTitle  string  `json:"books_title"`
	BooksAuthor *string `json:"books_author,omitempty"`
	BooksDesc   *string `json:"books_desc,omitempty"`
	BooksSlug   *string `json:"books_slug,omitempty"`

	// 🔄 ubah dari *_unix (int64) -> time.Time (RFC3339)
	BooksCreatedAt time.Time `json:"books_created_at"`
	BooksUpdatedAt time.Time `json:"books_updated_at"`
	BooksDeleted   bool      `json:"books_is_deleted"`
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
	r.BooksSlug = trimPtr(r.BooksSlug) // controller boleh GenerateSlug jika nil
}

func (r *BooksUpdateRequest) Normalize() {
	r.BooksTitle = trimPtr(r.BooksTitle)
	r.BooksAuthor = trimPtr(r.BooksAuthor)
	r.BooksDesc = trimPtr(r.BooksDesc)
	r.BooksSlug = trimPtr(r.BooksSlug)
}


/* =========================
   MAPPER
   ========================= */

func ToBooksResponse(m *model.BooksModel) BooksResponse {
	return BooksResponse{
		BooksID:       m.BooksID,
		BooksMasjidID: m.BooksMasjidID,
		BooksTitle:    m.BooksTitle,
		BooksAuthor:   m.BooksAuthor,
		BooksDesc:     m.BooksDesc,
		BooksSlug:     m.BooksSlug,

		// langsung pakai time.Time dari model (Go akan encode RFC3339)
		BooksCreatedAt: m.BooksCreatedAt,
		BooksUpdatedAt: m.BooksUpdatedAt,
		BooksDeleted:   !m.BooksDeletedAt.Time.IsZero(),
	}
}


func (r *BooksCreateRequest) ToModel() *model.BooksModel {
	// biarkan autoCreateTime/autoUpdateTime bekerja; created_at akan diisi DB
	return &model.BooksModel{
		BooksMasjidID: r.BooksMasjidID,
		BooksTitle:    r.BooksTitle,
		BooksAuthor:   r.BooksAuthor,
		BooksDesc:     r.BooksDesc,
		BooksSlug:     r.BooksSlug, // controller boleh override dengan GenerateSlug
		// BooksCreatedAt & BooksUpdatedAt otomatis
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
	if r.BooksSlug != nil {
		m.BooksSlug = r.BooksSlug
	}
	// updated_at akan diisi otomatis oleh GORM (autoUpdateTime)
}

// util kecil berguna jika butuh pointer dari literal
func strptr(s string) *string { return &s }
func nowptr() *time.Time      { t := time.Now(); return &t }
