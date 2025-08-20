// internals/features/lembaga/class_subject_books/dto/class_subject_book_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	model "masjidku_backend/internals/features/lembaga/class_books/model"

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

	// pengganti valid_from/valid_to & notes/is_primary
	ClassSubjectBooksIsActive *bool   `json:"class_subject_books_is_active" validate:"omitempty"`
	ClassSubjectBooksDesc     *string `json:"class_subject_books_desc"      validate:"omitempty,max=2000"`
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

	return model.ClassSubjectBookModel{
		ClassSubjectBooksMasjidID:       r.ClassSubjectBooksMasjidID,
		ClassSubjectBooksClassSubjectID: r.ClassSubjectBooksClassSubjectID,
		ClassSubjectBooksBookID:         r.ClassSubjectBooksBookID,
		ClassSubjectBooksIsActive:       isActive,
		ClassSubjectBooksDesc:           desc,
	}
}

// Update (partial)
type UpdateClassSubjectBookRequest struct {
	// biasanya masjid_id dipaksa dari token di controller; tetap pointer agar mudah di-apply
	ClassSubjectBooksMasjidID       *uuid.UUID `json:"class_subject_books_masjid_id"        validate:"omitempty"`
	ClassSubjectBooksClassSubjectID *uuid.UUID `json:"class_subject_books_class_subject_id" validate:"omitempty"`
	ClassSubjectBooksBookID         *uuid.UUID `json:"class_subject_books_book_id"          validate:"omitempty"`

	ClassSubjectBooksIsActive *bool   `json:"class_subject_books_is_active" validate:"omitempty"`
	ClassSubjectBooksDesc     *string `json:"class_subject_books_desc"      validate:"omitempty,max=2000"`
}

func (r *UpdateClassSubjectBookRequest) Apply(m *model.ClassSubjectBookModel) error {
	if m == nil {
		return errors.New("nil model")
	}
	if r.ClassSubjectBooksMasjidID != nil {
		m.ClassSubjectBooksMasjidID = *r.ClassSubjectBooksMasjidID
	}
	if r.ClassSubjectBooksClassSubjectID != nil {
		m.ClassSubjectBooksClassSubjectID = *r.ClassSubjectBooksClassSubjectID
	}
	if r.ClassSubjectBooksBookID != nil {
		m.ClassSubjectBooksBookID = *r.ClassSubjectBooksBookID
	}
	if r.ClassSubjectBooksIsActive != nil {
		m.ClassSubjectBooksIsActive = *r.ClassSubjectBooksIsActive
	}
	if r.ClassSubjectBooksDesc != nil {
		d := strings.TrimSpace(*r.ClassSubjectBooksDesc)
		if d == "" {
			m.ClassSubjectBooksDesc = nil
		} else {
			m.ClassSubjectBooksDesc = &d
		}
	}
	now := time.Now()
	m.ClassSubjectBooksUpdatedAt = &now
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
	// urutan yang relevan setelah refactor
	Sort *string `query:"sort" validate:"omitempty,oneof=created_at_asc created_at_desc updated_at_asc updated_at_desc"`
	// opsional: pencarian sederhana pada desc
	Q *string `query:"q" validate:"omitempty,max=100"`
}

/* =========================================================
   3) RESPONSE
   ========================================================= */

type ClassSubjectBookResponse struct {
	ClassSubjectBooksID             uuid.UUID  `json:"class_subject_books_id"`
	ClassSubjectBooksMasjidID       uuid.UUID  `json:"class_subject_books_masjid_id"`
	ClassSubjectBooksClassSubjectID uuid.UUID  `json:"class_subject_books_class_subject_id"`
	ClassSubjectBooksBookID         uuid.UUID  `json:"class_subject_books_book_id"`

	ClassSubjectBooksIsActive bool     `json:"class_subject_books_is_active"`
	ClassSubjectBooksDesc     *string  `json:"class_subject_books_desc,omitempty"`

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
	var deletedAt *time.Time
	if m.ClassSubjectBooksDeletedAt.Valid {
		deletedAt = &m.ClassSubjectBooksDeletedAt.Time
	}
	return ClassSubjectBookResponse{
		ClassSubjectBooksID:             m.ClassSubjectBooksID,
		ClassSubjectBooksMasjidID:       m.ClassSubjectBooksMasjidID,
		ClassSubjectBooksClassSubjectID: m.ClassSubjectBooksClassSubjectID,
		ClassSubjectBooksBookID:         m.ClassSubjectBooksBookID,
		ClassSubjectBooksIsActive:       m.ClassSubjectBooksIsActive,
		ClassSubjectBooksDesc:           m.ClassSubjectBooksDesc,
		ClassSubjectBooksCreatedAt:      m.ClassSubjectBooksCreatedAt,
		ClassSubjectBooksUpdatedAt:      m.ClassSubjectBooksUpdatedAt,
		ClassSubjectBooksDeletedAt:      deletedAt,
	}
}

func FromModels(list []model.ClassSubjectBookModel) []ClassSubjectBookResponse {
	out := make([]ClassSubjectBookResponse, 0, len(list))
	for _, m := range list {
		out = append(out, FromModel(m))
	}
	return out
}
