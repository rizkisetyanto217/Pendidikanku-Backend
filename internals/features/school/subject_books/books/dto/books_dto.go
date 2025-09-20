// internals/features/lembaga/class_books/dto/books_dto.go
package dto

import (
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/subject_books/books/model"

	"github.com/google/uuid"
)

/* =========================
   URL DTO (Upsert + Lite)
   ========================= */

func (u *BookURLUpsert) Normalize() {
	u.BookURLKind = strings.TrimSpace(u.BookURLKind)
	if u.BookURLKind == "" {
		u.BookURLKind = "attachment"
	}
	if u.BookURLLabel != nil {
		v := strings.TrimSpace(*u.BookURLLabel)
		if v == "" {
			u.BookURLLabel = nil
		} else {
			u.BookURLLabel = &v
		}
	}
	if u.BookURLHref != nil {
		v := strings.TrimSpace(*u.BookURLHref)
		if v == "" {
			u.BookURLHref = nil
		} else {
			u.BookURLHref = &v
		}
	}
	if u.BookURLObjectKey != nil {
		v := strings.TrimSpace(*u.BookURLObjectKey)
		if v == "" {
			u.BookURLObjectKey = nil
		} else {
			u.BookURLObjectKey = &v
		}
	}
}

/* =========================
   REQUEST
   ========================= */

// Query sederhana untuk listing
type BooksListQuery struct {
	Q       *string `query:"q"`        // cari di title/author/desc (controller)
	Author  *string `query:"author"`   // filter exact/ilike author (controller)
	OrderBy *string `query:"order_by"` // books_title|books_author|created_at
	Sort    *string `query:"sort"`     // asc|desc
	Limit   *int    `query:"limit"`
	Offset  *int    `query:"offset"`
}

/* ==== WITH-USAGES DTO (tetap) ==== */

type BookUsageSectionLite struct {
	ClassSectionsID       uuid.UUID `json:"class_sections_id"`
	ClassSectionsName     string    `json:"class_sections_name"`
	ClassSectionsSlug     string    `json:"class_sections_slug"`
	ClassSectionsCode     *string   `json:"class_sections_code,omitempty"`
	ClassSectionsCapacity *int      `json:"class_sections_capacity,omitempty"`
	ClassSectionsIsActive bool      `json:"class_sections_is_active"`
}

type BookUsage struct {
	ClassSubjectBooksID uuid.UUID              `json:"class_subject_books_id"`
	ClassSubjectID      *uuid.UUID             `json:"class_subjects_id,omitempty"`
	SubjectsID          *uuid.UUID             `json:"subjects_id,omitempty"`
	ClassesID           *uuid.UUID             `json:"classes_id,omitempty"`
	Sections            []BookUsageSectionLite `json:"sections"`
}

type BookWithUsagesResponse struct {
	BooksID       uuid.UUID `json:"books_id"`
	BooksMasjidID uuid.UUID `json:"books_masjid_id"`

	BooksTitle    string  `json:"books_title"`
	BooksAuthor   *string `json:"books_author,omitempty"`
	BooksDesc     *string `json:"books_desc,omitempty"`
	BooksSlug     *string `json:"books_slug,omitempty"`
	BooksURL      *string `json:"books_url,omitempty"`
	BooksImageURL *string `json:"books_image_url,omitempty"`

	Usages []BookUsage `json:"usages"`
}

/* =========================
   RESPONSE (list/detail)
   ========================= */

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
	r.BooksSlug = trimPtr(r.BooksSlug)
	for i := range r.URLs {
		r.URLs[i].Normalize()
	}
}

func (r *BooksUpdateRequest) Normalize() {
	r.BooksTitle = trimPtr(r.BooksTitle)
	r.BooksAuthor = trimPtr(r.BooksAuthor)
	r.BooksDesc = trimPtr(r.BooksDesc)
	r.BooksSlug = trimPtr(r.BooksSlug)
	for i := range r.URLs {
		r.URLs[i].Normalize()
	}
}

/* =========================
   MAPPER
   ========================= */

func (r *BooksCreateRequest) ToModel() *model.BooksModel {
	return &model.BooksModel{
		BooksMasjidID: r.BooksMasjidID,
		BooksTitle:    r.BooksTitle,
		BooksAuthor:   r.BooksAuthor,
		BooksDesc:     r.BooksDesc,
		BooksSlug:     r.BooksSlug,
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
}

// util kecil
func strptr(s string) *string { return &s }
func nowptr() *time.Time      { t := time.Now(); return &t }

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
   URL DTO (Lite & Upsert)
   ========================= */

type BookURLLiteBook struct {
	ID        uuid.UUID `json:"book_url_id"`
	Label     *string   `json:"book_url_label,omitempty"`
	Href      string    `json:"book_url_href"`
	Kind      string    `json:"book_url_kind"`
	IsPrimary bool      `json:"book_url_is_primary"`
	Order     int       `json:"book_url_order"`
}

type BookURLUpsert struct {
	BookURLKind      string  `json:"book_url_kind" validate:"required,min=1,max=24"`
	BookURLLabel     *string `json:"book_url_label,omitempty" validate:"omitempty,max=160"`
	BookURLHref      *string `json:"book_url_href,omitempty" validate:"omitempty,url"`
	BookURLObjectKey *string `json:"book_url_object_key,omitempty" validate:"omitempty"`
	BookURLOrder     int     `json:"book_url_order"`
	BookURLIsPrimary bool    `json:"book_url_is_primary"`
}

/* =========================
   REQUEST
   ========================= */

type BooksCreateRequest struct {
	BooksMasjidID uuid.UUID `json:"books_masjid_id" form:"books_masjid_id" validate:"required"`
	BooksTitle    string    `json:"books_title"     form:"books_title"     validate:"required,min=1"`
	BooksAuthor   *string   `json:"books_author,omitempty" form:"books_author" validate:"omitempty,min=1"`
	BooksDesc     *string   `json:"books_desc,omitempty"   form:"books_desc"   validate:"omitempty"`
	BooksSlug     *string   `json:"books_slug,omitempty"   form:"books_slug"   validate:"omitempty,max=160"`

	// metadata URL opsional
	URLs []BookURLUpsert `json:"urls,omitempty" validate:"omitempty,dive"`
}

type BooksUpdateRequest struct {
	BooksTitle  *string `json:"books_title,omitempty"  form:"books_title"  validate:"omitempty,min=1"`
	BooksAuthor *string `json:"books_author,omitempty" form:"books_author" validate:"omitempty,min=1"`
	BooksDesc   *string `json:"books_desc,omitempty"   form:"books_desc"   validate:"omitempty"`
	BooksSlug   *string `json:"books_slug,omitempty"   form:"books_slug"   validate:"omitempty,max=160"`

	URLs           []BookURLUpsert      `json:"urls,omitempty" validate:"omitempty,dive"`
	DeleteURLIDs   []uuid.UUID          `json:"url_delete_ids,omitempty" validate:"omitempty,dive,uuid4"`
	PrimaryPerKind map[string]uuid.UUID `json:"url_primary_per_kind,omitempty" validate:"omitempty"`
}

/* =========================
   RESPONSE
   ========================= */

type BooksResponse struct {
	BooksID       uuid.UUID `json:"books_id"`
	BooksMasjidID uuid.UUID `json:"books_masjid_id"`

	BooksTitle  string  `json:"books_title"`
	BooksAuthor *string `json:"books_author,omitempty"`
	BooksDesc   *string `json:"books_desc,omitempty"`
	BooksSlug   *string `json:"books_slug,omitempty"`

	BooksCreatedAt time.Time `json:"books_created_at"`
	BooksUpdatedAt time.Time `json:"books_updated_at"`
	BooksDeleted   bool      `json:"books_is_deleted"`

	// ⬇️ tambahkan ini agar controller bisa append URLs ke response
	URLs []BookURLLiteBook `json:"urls,omitempty"`
}

/* =========================
   MAPPER
   ========================= */

func ToBooksResponse(m *model.BooksModel) BooksResponse {
	return BooksResponse{
		BooksID:        m.BooksID,
		BooksMasjidID:  m.BooksMasjidID,
		BooksTitle:     m.BooksTitle,
		BooksAuthor:    m.BooksAuthor,
		BooksDesc:      m.BooksDesc,
		BooksSlug:      m.BooksSlug,
		BooksCreatedAt: m.BooksCreatedAt,
		BooksUpdatedAt: m.BooksUpdatedAt,
		BooksDeleted:   !m.BooksDeletedAt.Time.IsZero(),
	}
}
