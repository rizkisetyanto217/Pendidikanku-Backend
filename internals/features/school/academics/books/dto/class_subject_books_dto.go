// file: internals/features/lembaga/class_subject_books/dto/class_subject_book_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	model "madinahsalam_backend/internals/features/school/academics/books/model"

	"github.com/google/uuid"
)

/* =========================================================
   1) REQUESTS
========================================================= */

// Create
type CreateClassSubjectBookRequest struct {
	ClassSubjectBookSchoolID  uuid.UUID `json:"class_subject_book_school_id" validate:"required"`
	ClassSubjectBookSubjectID uuid.UUID `json:"class_subject_book_subject_id" validate:"required"`
	ClassSubjectBookBookID    uuid.UUID `json:"class_subject_book_book_id"    validate:"required"`

	// opsional; controller yang normalize + ensure-unique (alive-only)
	ClassSubjectBookSlug *string `json:"class_subject_book_slug" validate:"omitempty,max=160"`

	// flags & ordering
	ClassSubjectBookIsPrimary  *bool `json:"class_subject_book_is_primary"  validate:"omitempty"`
	ClassSubjectBookIsRequired *bool `json:"class_subject_book_is_required" validate:"omitempty"`
	ClassSubjectBookOrder      *int  `json:"class_subject_book_order"       validate:"omitempty"`

	// default true kalau tidak dikirim
	ClassSubjectBookIsActive *bool   `json:"class_subject_book_is_active" validate:"omitempty"`
	ClassSubjectBookDesc     *string `json:"class_subject_book_desc"      validate:"omitempty,max=2000"`
}

func (r *CreateClassSubjectBookRequest) Normalize() {
	if r.ClassSubjectBookSlug != nil {
		s := strings.TrimSpace(*r.ClassSubjectBookSlug)
		if s == "" {
			r.ClassSubjectBookSlug = nil
		} else {
			r.ClassSubjectBookSlug = &s
		}
	}
	if r.ClassSubjectBookDesc != nil {
		d := strings.TrimSpace(*r.ClassSubjectBookDesc)
		if d == "" {
			r.ClassSubjectBookDesc = nil
		} else {
			r.ClassSubjectBookDesc = &d
		}
	}
}

func (r CreateClassSubjectBookRequest) ToModel() model.ClassSubjectBookModel {
	isActive := true
	if r.ClassSubjectBookIsActive != nil {
		isActive = *r.ClassSubjectBookIsActive
	}

	isPrimary := false
	if r.ClassSubjectBookIsPrimary != nil {
		isPrimary = *r.ClassSubjectBookIsPrimary
	}

	isRequired := true
	if r.ClassSubjectBookIsRequired != nil {
		isRequired = *r.ClassSubjectBookIsRequired
	}

	return model.ClassSubjectBookModel{
		ClassSubjectBookSchoolID:  r.ClassSubjectBookSchoolID,
		ClassSubjectBookSubjectID: r.ClassSubjectBookSubjectID,
		ClassSubjectBookBookID:    r.ClassSubjectBookBookID,

		ClassSubjectBookSlug:       r.ClassSubjectBookSlug,
		ClassSubjectBookIsPrimary:  isPrimary,
		ClassSubjectBookIsRequired: isRequired,
		ClassSubjectBookOrder:      r.ClassSubjectBookOrder,

		ClassSubjectBookIsActive: isActive,
		ClassSubjectBookDesc:     r.ClassSubjectBookDesc,
	}
}

// Update (partial)
type UpdateClassSubjectBookRequest struct {
	ClassSubjectBookSchoolID  *uuid.UUID `json:"class_subject_book_school_id"  validate:"omitempty"`
	ClassSubjectBookSubjectID *uuid.UUID `json:"class_subject_book_subject_id" validate:"omitempty"`
	ClassSubjectBookBookID    *uuid.UUID `json:"class_subject_book_book_id"    validate:"omitempty"`

	// controller yang ensure-unique (alive-only)
	ClassSubjectBookSlug *string `json:"class_subject_book_slug" validate:"omitempty,max=160"`

	ClassSubjectBookIsPrimary  *bool `json:"class_subject_book_is_primary"  validate:"omitempty"`
	ClassSubjectBookIsRequired *bool `json:"class_subject_book_is_required" validate:"omitempty"`
	ClassSubjectBookOrder      *int  `json:"class_subject_book_order"       validate:"omitempty"`

	ClassSubjectBookIsActive *bool   `json:"class_subject_book_is_active" validate:"omitempty"`
	ClassSubjectBookDesc     *string `json:"class_subject_book_desc"      validate:"omitempty,max=2000"`
}

func (r *UpdateClassSubjectBookRequest) Apply(m *model.ClassSubjectBookModel) error {
	if m == nil {
		return errors.New("nil model")
	}
	if r.ClassSubjectBookSchoolID != nil {
		m.ClassSubjectBookSchoolID = *r.ClassSubjectBookSchoolID
	}
	if r.ClassSubjectBookSubjectID != nil {
		m.ClassSubjectBookSubjectID = *r.ClassSubjectBookSubjectID
	}
	if r.ClassSubjectBookBookID != nil {
		// Perubahan book_id → controller akan refresh cache buku
		m.ClassSubjectBookBookID = *r.ClassSubjectBookBookID
	}
	if r.ClassSubjectBookIsActive != nil {
		m.ClassSubjectBookIsActive = *r.ClassSubjectBookIsActive
	}
	if r.ClassSubjectBookIsPrimary != nil {
		m.ClassSubjectBookIsPrimary = *r.ClassSubjectBookIsPrimary
	}
	if r.ClassSubjectBookIsRequired != nil {
		m.ClassSubjectBookIsRequired = *r.ClassSubjectBookIsRequired
	}
	if r.ClassSubjectBookOrder != nil {
		m.ClassSubjectBookOrder = r.ClassSubjectBookOrder
	}
	if r.ClassSubjectBookDesc != nil {
		d := strings.TrimSpace(*r.ClassSubjectBookDesc)
		if d == "" {
			m.ClassSubjectBookDesc = nil
		} else {
			m.ClassSubjectBookDesc = &d
		}
	}
	if r.ClassSubjectBookSlug != nil {
		s := strings.TrimSpace(*r.ClassSubjectBookSlug)
		if s == "" {
			m.ClassSubjectBookSlug = nil
		} else {
			m.ClassSubjectBookSlug = &s
		}
	}
	return nil
}

/* =========================================================
   2) LIST QUERY
========================================================= */

type ListClassSubjectBookQuery struct {
	Limit       *int       `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset      *int       `query:"offset" validate:"omitempty,min=0"`
	SubjectID   *uuid.UUID `query:"subject_id" validate:"omitempty"`
	BookID      *uuid.UUID `query:"book_id"    validate:"omitempty"`
	IsActive    *bool      `query:"is_active"  validate:"omitempty"`
	IsPrimary   *bool      `query:"is_primary" validate:"omitempty"`
	IsRequired  *bool      `query:"is_required" validate:"omitempty"`
	WithDeleted *bool      `query:"with_deleted" validate:"omitempty"`

	// q: cari di slug relasi & judul buku cache & nama/slug subject cache (LOWER LIKE/TRGM)
	Q *string `query:"q" validate:"omitempty,max=100"`

	// created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
	Sort *string `query:"sort" validate:"omitempty,oneof=created_at_asc created_at_desc updated_at_asc updated_at_desc"`
}

/* =========================================================
   3) RESPONSE
========================================================= */

// (Opsional) embed URL buku kalau controller melakukan join ke book_urls
type BookURLLite struct {
	BookURLID                 uuid.UUID  `json:"book_url_id"`
	BookURLSchoolID           uuid.UUID  `json:"book_url_school_id"`
	BookURLBookID             uuid.UUID  `json:"book_url_book_id"`
	BookURLLabel              *string    `json:"book_url_label,omitempty"`
	BookURLHref               string     `json:"book_url_href"`
	BookURLObjectKey          *string    `json:"book_url_object_key,omitempty"`
	BookURLIsPrimary          bool       `json:"book_url_is_primary"`
	BookURLKind               string     `json:"book_url_kind"`
	BookURLOrder              int        `json:"book_url_order"`
	BookURLDeletePendingUntil *time.Time `json:"book_url_delete_pending_until,omitempty"`
	BookURLCreatedAt          time.Time  `json:"book_url_created_at"`
	BookURLUpdatedAt          time.Time  `json:"book_url_updated_at"`
	BookURLDeletedAt          *time.Time `json:"book_url_deleted_at,omitempty"`
}

// (Opsional) ringkasan buku asli bila controller melakukan join langsung ke books
type BookLite struct {
	BookID        uuid.UUID     `json:"book_id"`
	BookSchoolID  uuid.UUID     `json:"book_school_id"`
	BookTitle     string        `json:"book_title"`
	BookAuthor    *string       `json:"book_author,omitempty"`
	BookSlug      *string       `json:"book_slug,omitempty"`
	BookImageURL  *string       `json:"book_image_url,omitempty"`
	BookPublisher *string       `json:"book_publisher,omitempty"`
	BookYear      *int16        `json:"book_publication_year,omitempty"`
	URLs          []BookURLLite `json:"urls,omitempty"`
}

// (Opsional) ringkasan subject asli bila controller join ke subjects
type SubjectLite struct {
	SubjectID       uuid.UUID `json:"subject_id"`
	SubjectSchoolID uuid.UUID `json:"subject_school_id"`
	SubjectCode     string    `json:"subject_code"`
	SubjectName     string    `json:"subject_name"`
	SubjectSlug     string    `json:"subject_slug"`
}

// Response utama relasi + cache buku/subject (dibekukan via service/controller)
type ClassSubjectBookResponse struct {
	ClassSubjectBookID        uuid.UUID `json:"class_subject_book_id"`
	ClassSubjectBookSchoolID  uuid.UUID `json:"class_subject_book_school_id"`
	ClassSubjectBookSubjectID uuid.UUID `json:"class_subject_book_subject_id"`
	ClassSubjectBookBookID    uuid.UUID `json:"class_subject_book_book_id"`

	ClassSubjectBookSlug *string `json:"class_subject_book_slug,omitempty"`

	ClassSubjectBookIsPrimary  bool    `json:"class_subject_book_is_primary"`
	ClassSubjectBookIsRequired bool    `json:"class_subject_book_is_required"`
	ClassSubjectBookOrder      *int    `json:"class_subject_book_order,omitempty"`
	ClassSubjectBookIsActive   bool    `json:"class_subject_book_is_active"`
	ClassSubjectBookDesc       *string `json:"class_subject_book_desc,omitempty"`

	// caches dari books
	ClassSubjectBookBookTitleCache           *string `json:"class_subject_book_book_title_cache,omitempty"`
	ClassSubjectBookBookAuthorCache          *string `json:"class_subject_book_book_author_cache,omitempty"`
	ClassSubjectBookBookSlugCache            *string `json:"class_subject_book_book_slug_cache,omitempty"`
	ClassSubjectBookBookPublisherCache       *string `json:"class_subject_book_book_publisher_cache,omitempty"`
	ClassSubjectBookBookPublicationYearCache *int16  `json:"class_subject_book_book_publication_year_cache,omitempty"`
	ClassSubjectBookBookImageURLCache        *string `json:"class_subject_book_book_image_url_cache,omitempty"`

	// caches subject
	ClassSubjectBookSubjectCodeCache *string `json:"class_subject_book_subject_code_cache,omitempty"`
	ClassSubjectBookSubjectNameCache *string `json:"class_subject_book_subject_name_cache,omitempty"`
	ClassSubjectBookSubjectSlugCache *string `json:"class_subject_book_subject_slug_cache,omitempty"`

	ClassSubjectBookCreatedAt time.Time  `json:"class_subject_book_created_at"`
	ClassSubjectBookUpdatedAt time.Time  `json:"class_subject_book_updated_at"`
	ClassSubjectBookDeletedAt *time.Time `json:"class_subject_book_deleted_at,omitempty"`

	// opsional join
	Book    *BookLite    `json:"book,omitempty"`
	Subject *SubjectLite `json:"subject,omitempty"`
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
		ClassSubjectBookID:        m.ClassSubjectBookID,
		ClassSubjectBookSchoolID:  m.ClassSubjectBookSchoolID,
		ClassSubjectBookSubjectID: m.ClassSubjectBookSubjectID,
		ClassSubjectBookBookID:    m.ClassSubjectBookBookID,

		ClassSubjectBookSlug:       m.ClassSubjectBookSlug,
		ClassSubjectBookIsPrimary:  m.ClassSubjectBookIsPrimary,
		ClassSubjectBookIsRequired: m.ClassSubjectBookIsRequired,
		ClassSubjectBookOrder:      m.ClassSubjectBookOrder,
		ClassSubjectBookIsActive:   m.ClassSubjectBookIsActive,
		ClassSubjectBookDesc:       m.ClassSubjectBookDesc,

		ClassSubjectBookCreatedAt: m.ClassSubjectBookCreatedAt,
		ClassSubjectBookUpdatedAt: m.ClassSubjectBookUpdatedAt,
		ClassSubjectBookDeletedAt: deletedAt,

		// caches buku
		ClassSubjectBookBookTitleCache:           m.ClassSubjectBookBookTitleCache,
		ClassSubjectBookBookAuthorCache:          m.ClassSubjectBookBookAuthorCache,
		ClassSubjectBookBookSlugCache:            m.ClassSubjectBookBookSlugCache,
		ClassSubjectBookBookPublisherCache:       m.ClassSubjectBookBookPublisherCache,
		ClassSubjectBookBookPublicationYearCache: m.ClassSubjectBookBookPublicationYearCache,
		ClassSubjectBookBookImageURLCache:        m.ClassSubjectBookBookImageURLCache,

		// caches subject
		ClassSubjectBookSubjectCodeCache: m.ClassSubjectBookSubjectCodeCache,
		ClassSubjectBookSubjectNameCache: m.ClassSubjectBookSubjectNameCache,
		ClassSubjectBookSubjectSlugCache: m.ClassSubjectBookSubjectSlugCache,
	}
}

func FromModels(list []model.ClassSubjectBookModel) []ClassSubjectBookResponse {
	out := make([]ClassSubjectBookResponse, 0, len(list))
	for _, it := range list {
		out = append(out, FromModel(it))
	}
	return out
}
