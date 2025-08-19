// internals/features/lembaga/class_subject_books/dto/class_subject_book_dto.go
package dto

import (
	"strings"
	"time"

	model "masjidku_backend/internals/features/lembaga/class_books/model"

	"github.com/google/uuid"
)

/* =========================================================
   Helpers
   ========================================================= */

const dateLayout = "2006-01-02"

func parseDatePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	d := strings.TrimSpace(*s)
	if d == "" {
		return nil
	}
	if t, err := time.Parse(dateLayout, d); err == nil {
		// make it date-only (00:00 local)
		tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
		return &tt
	}
	return nil
}

/* =========================================================
   1) REQUESTS
   ========================================================= */

// Create
type CreateClassSubjectBookRequest struct {
	ClassSubjectBooksMasjidID       uuid.UUID `json:"class_subject_books_masjid_id"        validate:"required"`
	ClassSubjectBooksClassSubjectID uuid.UUID `json:"class_subject_books_class_subject_id" validate:"required"`
	ClassSubjectBooksBookID         uuid.UUID `json:"class_subject_books_book_id"          validate:"required"`

	// YYYY-MM-DD (opsional)
	ValidFrom *string `json:"valid_from" validate:"omitempty,datetime=2006-01-02"`
	ValidTo   *string `json:"valid_to"   validate:"omitempty,datetime=2006-01-02"`

	IsPrimary *bool   `json:"is_primary" validate:"omitempty"`
	Notes     *string `json:"notes"      validate:"omitempty"`
}

func (r CreateClassSubjectBookRequest) ToModel() model.ClassSubjectBookModel {
	isPrimary := false
	if r.IsPrimary != nil {
		isPrimary = *r.IsPrimary
	}
	var notes *string
	if r.Notes != nil {
		n := strings.TrimSpace(*r.Notes)
		if n != "" {
			notes = &n
		}
	}

	return model.ClassSubjectBookModel{
		ClassSubjectBooksMasjidID:       r.ClassSubjectBooksMasjidID,
		ClassSubjectBooksClassSubjectID: r.ClassSubjectBooksClassSubjectID,
		ClassSubjectBooksBookID:         r.ClassSubjectBooksBookID,
		ClassSubjectBooksValidFrom:      parseDatePtr(r.ValidFrom),
		ClassSubjectBooksValidTo:        parseDatePtr(r.ValidTo),
		ClassSubjectBooksIsPrimary:      isPrimary,
		ClassSubjectBooksNotes:          notes,
	}
}

// Update (partial)
type UpdateClassSubjectBookRequest struct {
	ClassSubjectBooksMasjidID       *uuid.UUID `json:"class_subject_books_masjid_id"        validate:"omitempty"`
	ClassSubjectBooksClassSubjectID *uuid.UUID `json:"class_subject_books_class_subject_id" validate:"omitempty"`
	ClassSubjectBooksBookID         *uuid.UUID `json:"class_subject_books_book_id"          validate:"omitempty"`

	ValidFrom *string `json:"valid_from" validate:"omitempty,datetime=2006-01-02"`
	ValidTo   *string `json:"valid_to"   validate:"omitempty,datetime=2006-01-02"`

	IsPrimary *bool   `json:"is_primary" validate:"omitempty"`
	Notes     *string `json:"notes"      validate:"omitempty"`
}

func (r *UpdateClassSubjectBookRequest) Apply(m *model.ClassSubjectBookModel) {
	if r.ClassSubjectBooksMasjidID != nil {
		m.ClassSubjectBooksMasjidID = *r.ClassSubjectBooksMasjidID
	}
	if r.ClassSubjectBooksClassSubjectID != nil {
		m.ClassSubjectBooksClassSubjectID = *r.ClassSubjectBooksClassSubjectID
	}
	if r.ClassSubjectBooksBookID != nil {
		m.ClassSubjectBooksBookID = *r.ClassSubjectBooksBookID
	}
	if r.ValidFrom != nil {
		m.ClassSubjectBooksValidFrom = parseDatePtr(r.ValidFrom)
	}
	if r.ValidTo != nil {
		m.ClassSubjectBooksValidTo = parseDatePtr(r.ValidTo)
	}
	if r.IsPrimary != nil {
		m.ClassSubjectBooksIsPrimary = *r.IsPrimary
	}
	if r.Notes != nil {
		n := strings.TrimSpace(*r.Notes)
		if n == "" {
			m.ClassSubjectBooksNotes = nil
		} else {
			m.ClassSubjectBooksNotes = &n
		}
	}
	now := time.Now()
	m.ClassSubjectBooksUpdatedAt = &now
}

/* =========================================================
   2) LIST QUERY
   ========================================================= */

type ListClassSubjectBookQuery struct {
	Limit       *int      `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset      *int      `query:"offset" validate:"omitempty,min=0"`
	ClassSubjectID *uuid.UUID `query:"class_subject_id" validate:"omitempty"`
	BookID      *uuid.UUID `query:"book_id" validate:"omitempty"`
	IsPrimary   *bool     `query:"is_primary" validate:"omitempty"`
	ActiveOn    *string   `query:"active_on" validate:"omitempty,datetime=2006-01-02"` // filter: valid_from <= active_on <= valid_to (null = open)
	WithDeleted *bool     `query:"with_deleted" validate:"omitempty"`
	Sort        *string   `query:"sort" validate:"omitempty,oneof=created_at_asc created_at_desc valid_from_asc valid_from_desc"`
}

/* =========================================================
   3) RESPONSE
   ========================================================= */

type ClassSubjectBookResponse struct {
	ClassSubjectBooksID             uuid.UUID  `json:"class_subject_books_id"`
	ClassSubjectBooksMasjidID       uuid.UUID  `json:"class_subject_books_masjid_id"`
	ClassSubjectBooksClassSubjectID uuid.UUID  `json:"class_subject_books_class_subject_id"`
	ClassSubjectBooksBookID         uuid.UUID  `json:"class_subject_books_book_id"`

	ValidFrom *time.Time `json:"valid_from,omitempty"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`

	IsPrimary bool    `json:"is_primary"`
	Notes     *string `json:"notes,omitempty"`

	ClassSubjectBooksCreatedAt time.Time  `json:"class_subject_books_created_at"`
	ClassSubjectBooksUpdatedAt *time.Time `json:"class_subject_books_updated_at,omitempty"`
	ClassSubjectBooksDeletedAt *time.Time `json:"class_subject_books_deleted_at,omitempty"`
}

// Pagination (reusable)
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
	return ClassSubjectBookResponse{
		ClassSubjectBooksID:             m.ClassSubjectBooksID,
		ClassSubjectBooksMasjidID:       m.ClassSubjectBooksMasjidID,
		ClassSubjectBooksClassSubjectID: m.ClassSubjectBooksClassSubjectID,
		ClassSubjectBooksBookID:         m.ClassSubjectBooksBookID,
		ValidFrom:                       m.ClassSubjectBooksValidFrom,
		ValidTo:                         m.ClassSubjectBooksValidTo,
		IsPrimary:                       m.ClassSubjectBooksIsPrimary,
		Notes:                           m.ClassSubjectBooksNotes,
		ClassSubjectBooksCreatedAt:      m.ClassSubjectBooksCreatedAt,
		ClassSubjectBooksUpdatedAt:      m.ClassSubjectBooksUpdatedAt,
		ClassSubjectBooksDeletedAt:      m.ClassSubjectBooksDeletedAt,
	}
}

func FromModels(list []model.ClassSubjectBookModel) []ClassSubjectBookResponse {
	out := make([]ClassSubjectBookResponse, 0, len(list))
	for _, m := range list {
		out = append(out, FromModel(m))
	}
	return out
}
