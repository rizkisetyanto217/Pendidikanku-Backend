// file: internals/features/school/academics/books/dto/book_dto.go
package dto

import (
	"strings"
	"time"

	model "madinahsalam_backend/internals/features/school/academics/books/model"
	classSubjectDTO "madinahsalam_backend/internals/features/school/academics/subjects/dto"

	"github.com/google/uuid"
)

/* =========================================================
   QUERY (LIST)
========================================================= */

type BooksListQuery struct {
	Q       *string `query:"q"`        // cari di title/author/desc (controller yang handle)
	Author  *string `query:"author"`   // filter exact/ilike author
	OrderBy *string `query:"order_by"` // book_title|book_author|created_at
	Sort    *string `query:"sort"`     // asc|desc
	Limit   *int    `query:"limit"`
	Offset  *int    `query:"offset"`
	// jika mau tampilkan yang soft-deleted
	WithDeleted *bool `query:"with_deleted"`
}

/* =========================================================
   URL DTO (opsional, bila ada tabel/entitas URL terpisah)
========================================================= */

type BookURLUpsert struct {
	BookURLKind      string  `json:"book_url_kind" validate:"required,min=1,max=24"` // attachment|image|link|video|other
	BookURLLabel     *string `json:"book_url_label,omitempty" validate:"omitempty,max=160"`
	BookURLHref      *string `json:"book_url_href,omitempty"  validate:"omitempty,url"`
	BookURLObjectKey *string `json:"book_url_object_key,omitempty" validate:"omitempty"`
	BookURLOrder     int     `json:"book_url_order"`
	BookURLIsPrimary bool    `json:"book_url_is_primary"`
}

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

/* =========================================================
   REQUEST
========================================================= */

type BookCreateRequest struct {
	BookSchoolID    uuid.UUID `json:"book_school_id"      form:"book_school_id"      validate:"required"`
	BookTitle       string    `json:"book_title"          form:"book_title"          validate:"required,min=1"`
	BookAuthor      *string   `json:"book_author,omitempty"  form:"book_author"      validate:"omitempty,min=1"`
	BookDesc        *string   `json:"book_desc,omitempty"    form:"book_desc"        validate:"omitempty"`
	BookSlug        *string   `json:"book_slug,omitempty"    form:"book_slug"        validate:"omitempty,max=160"`
	BookPurchaseURL *string   `json:"book_purchase_url,omitempty" form:"book_purchase_url" validate:"omitempty,url"`
}

type BookUpdateRequest struct {
	BookTitle       *string `json:"book_title,omitempty"        form:"book_title"        validate:"omitempty,min=1"`
	BookAuthor      *string `json:"book_author,omitempty"       form:"book_author"       validate:"omitempty,min=1"`
	BookDesc        *string `json:"book_desc,omitempty"         form:"book_desc"         validate:"omitempty"`
	BookSlug        *string `json:"book_slug,omitempty"         form:"book_slug"         validate:"omitempty,max=160"`
	BookPurchaseURL *string `json:"book_purchase_url,omitempty" form:"book_purchase_url" validate:"omitempty,url"`

	// opsional untuk sinkron URL
	URLs           []BookURLUpsert      `json:"urls,omitempty" validate:"omitempty,dive"`
	DeleteURLIDs   []uuid.UUID          `json:"url_delete_ids,omitempty" validate:"omitempty,dive,uuid4"`
	PrimaryPerKind map[string]uuid.UUID `json:"url_primary_per_kind,omitempty" validate:"omitempty"`
}

/* =========================================================
   NORMALIZER
========================================================= */

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

func (r *BookCreateRequest) Normalize() {
	r.BookTitle = strings.TrimSpace(r.BookTitle)
	r.BookAuthor = trimPtr(r.BookAuthor)
	r.BookDesc = trimPtr(r.BookDesc)
	r.BookSlug = trimPtr(r.BookSlug)
	r.BookPurchaseURL = trimPtr(r.BookPurchaseURL)
}

func (r *BookUpdateRequest) Normalize() {
	r.BookTitle = trimPtr(r.BookTitle)
	r.BookAuthor = trimPtr(r.BookAuthor)
	r.BookDesc = trimPtr(r.BookDesc)
	r.BookSlug = trimPtr(r.BookSlug)
	r.BookPurchaseURL = trimPtr(r.BookPurchaseURL)
	for i := range r.URLs {
		r.URLs[i].Normalize()
	}
}

/* =========================================================
   RESPONSE (FULL)
========================================================= */

type BookResponse struct {
	BookID       uuid.UUID `json:"book_id"`
	BookSchoolID uuid.UUID `json:"book_school_id"`

	BookTitle  string  `json:"book_title"`
	BookAuthor *string `json:"book_author,omitempty"`
	BookDesc   *string `json:"book_desc,omitempty"`
	BookSlug   *string `json:"book_slug,omitempty"`

	BookImageURL       *string `json:"book_image_url,omitempty"`
	BookImageObjectKey *string `json:"book_image_object_key,omitempty"`

	// link pembelian
	BookPurchaseURL *string `json:"book_purchase_url,omitempty"`

	// publisher & tahun terbit
	BookPublisher       *string `json:"book_publisher,omitempty"`
	BookPublicationYear *int16  `json:"book_publication_year,omitempty"`

	// versi lama image (GC)
	BookImageURLOld             *string    `json:"book_image_url_old,omitempty"`
	BookImageObjectKeyOld       *string    `json:"book_image_object_key_old,omitempty"`
	BookImageDeletePendingUntil *time.Time `json:"book_image_delete_pending_until,omitempty"`

	BookCreatedAt time.Time `json:"book_created_at"`
	BookUpdatedAt time.Time `json:"book_updated_at"`
	BookIsDeleted bool      `json:"book_is_deleted"`

	// nested mapel (kalau diminta)
	ClassSubjectBooks []BookClassSubjectItem `json:"class_subject_books,omitempty"`
}

type PageInfo struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type BooksListResponse struct {
	Data []BookResponse `json:"data"`
	Page PageInfo       `json:"page"`
}

/* =========================================================
   MAPPER ⇄ MODEL
========================================================= */

func ToBookResponse(m *model.BookModel) BookResponse {
	return BookResponse{
		BookID:       m.BookID,
		BookSchoolID: m.BookSchoolID,

		BookTitle:  m.BookTitle,
		BookAuthor: m.BookAuthor,
		BookDesc:   m.BookDesc,
		BookSlug:   m.BookSlug,

		BookImageURL:       m.BookImageURL,
		BookImageObjectKey: m.BookImageObjectKey,

		BookPurchaseURL: m.BookPurchaseURL,

		BookPublisher:       m.BookPublisher,
		BookPublicationYear: m.BookPublicationYear,

		BookImageURLOld:             m.BookImageURLOld,
		BookImageObjectKeyOld:       m.BookImageObjectKeyOld,
		BookImageDeletePendingUntil: m.BookImageDeletePendingUntil,

		BookCreatedAt: m.BookCreatedAt,
		BookUpdatedAt: m.BookUpdatedAt,
		BookIsDeleted: !m.BookDeletedAt.Time.IsZero(),
	}
}

func (r *BookCreateRequest) ToModel() *model.BookModel {
	return &model.BookModel{
		BookSchoolID:    r.BookSchoolID,
		BookTitle:       r.BookTitle,
		BookAuthor:      r.BookAuthor,
		BookDesc:        r.BookDesc,
		BookSlug:        r.BookSlug,
		BookPurchaseURL: r.BookPurchaseURL,
	}
}

func (r *BookUpdateRequest) ApplyToModel(m *model.BookModel) {
	if r.BookTitle != nil {
		m.BookTitle = *r.BookTitle
	}
	if r.BookAuthor != nil {
		m.BookAuthor = r.BookAuthor
	}
	if r.BookDesc != nil {
		m.BookDesc = r.BookDesc
	}
	if r.BookSlug != nil {
		m.BookSlug = r.BookSlug
	}
	if r.BookPurchaseURL != nil {
		m.BookPurchaseURL = r.BookPurchaseURL
	}
}

/* =========================================================
   WITH-USAGES (opsional, untuk detail penggunaan)
========================================================= */

type BookUsageSectionLite struct {
	ClassSectionID   uuid.UUID `json:"class_section_id"`
	ClassSectionName string    `json:"class_section_name"`
	ClassSectionSlug string    `json:"class_section_slug"`
	ClassSectionCode *string   `json:"class_section_code,omitempty"`

	// konsisten dengan class_sections: quota_total
	ClassSectionQuotaTotal *int `json:"class_section_quota_total,omitempty"`

	// enum: active | inactive | completed
	ClassSectionStatus string `json:"class_section_status"`
}

type BookUsage struct {
	ClassSubjectBookID uuid.UUID              `json:"class_subject_book_id"`
	ClassSubjectID     *uuid.UUID             `json:"class_subject_id,omitempty"`
	SubjectID          *uuid.UUID             `json:"subject_id,omitempty"`
	ClassID            *uuid.UUID             `json:"class_id,omitempty"`
	Sections           []BookUsageSectionLite `json:"sections"`
}

type BookWithUsagesResponse struct {
	BookID       uuid.UUID `json:"book_id"`
	BookSchoolID uuid.UUID `json:"book_school_id"`

	BookTitle    string  `json:"book_title"`
	BookAuthor   *string `json:"book_author,omitempty"`
	BookDesc     *string `json:"book_desc,omitempty"`
	BookSlug     *string `json:"book_slug,omitempty"`
	BookURL      *string `json:"book_url,omitempty"`       // bila ada URL utama
	BookImageURL *string `json:"book_image_url,omitempty"` // bila ada gambar

	// link pembelian khusus
	BookPurchaseURL *string `json:"book_purchase_url,omitempty"`

	Usages []BookUsage `json:"usages"`
}

/* =========================
   LIST QUERY (with usages)
========================= */

type BooksWithUsagesListQuery struct {
	Q           *string `query:"q"`
	Author      *string `query:"author"`
	OrderBy     *string `query:"order_by"` // books_title|books_author|created_at
	Sort        *string `query:"sort"`     // asc|desc
	Limit       *int    `query:"limit"`
	Offset      *int    `query:"offset"`
	WithDeleted *bool   `query:"with_deleted"`
}

/* =========================================================
   COMPACT DTO
========================================================= */

// BookClassSubjectItem: pivot + compact class_subject
type BookClassSubjectItem struct {
	ClassSubjectBookID         uuid.UUID `json:"class_subject_book_id"`
	ClassSubjectBookIsPrimary  bool      `json:"class_subject_book_is_primary"`
	ClassSubjectBookIsRequired bool      `json:"class_subject_book_is_required"`
	ClassSubjectBookOrder      *int      `json:"class_subject_book_order,omitempty"`

	// detail class_subject (compact: sudah ada caches subject + class_parent)
	ClassSubject classSubjectDTO.ClassSubjectCompactResponse `json:"class_subject"`
}

// BookCompact dipakai sebagai bentuk ringkas di tempat lain
// (misal embed di materials, CSST, dsb)
type BookCompact struct {
	BookID       uuid.UUID `json:"book_id"`
	BookSchoolID uuid.UUID `json:"book_school_id"`

	BookTitle  string  `json:"book_title"`
	BookAuthor *string `json:"book_author,omitempty"`
	BookDesc   *string `json:"book_desc,omitempty"`
	BookSlug   *string `json:"book_slug,omitempty"`

	BookImageURL    *string `json:"book_image_url,omitempty"`
	BookPurchaseURL *string `json:"book_purchase_url,omitempty"`

	BookCreatedAt time.Time `json:"book_created_at"`
	BookUpdatedAt time.Time `json:"book_updated_at"`
	BookIsDeleted bool      `json:"book_is_deleted"`

	// daftar mapel lengkap (class_subject compact)
	ClassSubjectBooks []BookClassSubjectItem `json:"class_subject_books,omitempty"`
}

func ToBookCompact(m *model.BookModel) BookCompact {
	return BookCompact{
		BookID:       m.BookID,
		BookSchoolID: m.BookSchoolID,

		BookTitle:  m.BookTitle,
		BookAuthor: m.BookAuthor,
		BookDesc:   m.BookDesc,
		BookSlug:   m.BookSlug,

		BookImageURL:    m.BookImageURL,
		BookPurchaseURL: m.BookPurchaseURL,

		BookCreatedAt: m.BookCreatedAt,
		BookUpdatedAt: m.BookUpdatedAt,
		BookIsDeleted: !m.BookDeletedAt.Time.IsZero(),
	}
}

func ToBookCompactList(models []*model.BookModel) []BookCompact {
	out := make([]BookCompact, 0, len(models))
	for _, m := range models {
		if m == nil {
			continue
		}
		out = append(out, ToBookCompact(m))
	}
	return out
}
