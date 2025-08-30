// internals/features/lembaga/class_subjects/dto/class_subject_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	booksModel "masjidku_backend/internals/features/school/class_subject_books/books/model"
	csModel "masjidku_backend/internals/features/school/class_subject_books/subject/model"
)

/* =========================================================
   1) REQUEST DTO
   ========================================================= */

type CreateClassSubjectRequest struct {
	MasjidID     uuid.UUID  `json:"class_subjects_masjid_id" validate:"required"`
	ClassID      uuid.UUID  `json:"class_subjects_class_id" validate:"required"`
	SubjectID    uuid.UUID  `json:"class_subjects_subject_id" validate:"required"`
	TermID       *uuid.UUID `json:"class_subjects_term_id" validate:"omitempty"`

	OrderIndex   *int    `json:"class_subjects_order_index" validate:"omitempty,min=0"`
	HoursPerWeek *int    `json:"class_subjects_hours_per_week" validate:"omitempty,min=0"`
	MinScore     *int    `json:"class_subjects_min_passing_score" validate:"omitempty,min=0,max=100"`
	Weight       *int    `json:"class_subjects_weight_on_report" validate:"omitempty,min=0"`
	IsCore       *bool   `json:"class_subjects_is_core" validate:"omitempty"`
	Desc         *string `json:"class_subjects_desc" validate:"omitempty"`
	IsActive     *bool   `json:"class_subjects_is_active" validate:"omitempty"`
}

type UpdateClassSubjectRequest struct {
	MasjidID     *uuid.UUID `json:"class_subjects_masjid_id" validate:"omitempty"`
	ClassID      *uuid.UUID `json:"class_subjects_class_id" validate:"omitempty"`
	SubjectID    *uuid.UUID `json:"class_subjects_subject_id" validate:"omitempty"`
	TermID       *uuid.UUID `json:"class_subjects_term_id" validate:"omitempty"`

	OrderIndex   *int    `json:"class_subjects_order_index" validate:"omitempty,min=0"`
	HoursPerWeek *int    `json:"class_subjects_hours_per_week" validate:"omitempty,min=0"`
	MinScore     *int    `json:"class_subjects_min_passing_score" validate:"omitempty,min=0,max=100"`
	Weight       *int    `json:"class_subjects_weight_on_report" validate:"omitempty,min=0"`
	IsCore       *bool   `json:"class_subjects_is_core" validate:"omitempty"`
	Desc         *string `json:"class_subjects_desc" validate:"omitempty"`
	IsActive     *bool   `json:"class_subjects_is_active" validate:"omitempty"`
}

type ListClassSubjectQuery struct {
	Limit       *int    `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset      *int    `query:"offset" validate:"omitempty,min=0"`
	IsActive    *bool   `query:"is_active" validate:"omitempty"`
	Q           *string `query:"q" validate:"omitempty,max=100"`
	OrderBy     *string `query:"order_by" validate:"omitempty,oneof=order_index created_at updated_at"`
	Sort        *string `query:"sort" validate:"omitempty,oneof=asc desc"`
	WithDeleted *bool   `query:"with_deleted" validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO (basic)
   ========================================================= */

type ClassSubjectResponse struct {
	ID           uuid.UUID  `json:"class_subjects_id"`
	MasjidID     uuid.UUID  `json:"class_subjects_masjid_id"`
	ClassID      uuid.UUID  `json:"class_subjects_class_id"`
	SubjectID    uuid.UUID  `json:"class_subjects_subject_id"`
	TermID       *uuid.UUID `json:"class_subjects_term_id,omitempty"`

	OrderIndex   *int       `json:"class_subjects_order_index,omitempty"`
	HoursPerWeek *int       `json:"class_subjects_hours_per_week,omitempty"`
	MinScore     *int       `json:"class_subjects_min_passing_score,omitempty"`
	Weight       *int       `json:"class_subjects_weight_on_report,omitempty"`
	IsCore       bool       `json:"class_subjects_is_core"`
	Desc         *string    `json:"class_subjects_desc,omitempty"`
	IsActive     bool       `json:"class_subjects_is_active"`
	CreatedAt    time.Time  `json:"class_subjects_created_at"`
	UpdatedAt    *time.Time `json:"class_subjects_updated_at,omitempty"`
	DeletedAt    *time.Time `json:"class_subjects_deleted_at,omitempty"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type ClassSubjectListResponse struct {
	Items      []ClassSubjectResponse `json:"items"`
	Pagination Pagination             `json:"pagination"`
}

/* =========================================================
   3) MAPPERS (basic)
   ========================================================= */

func (r CreateClassSubjectRequest) ToModel() csModel.ClassSubjectModel {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	isCore := false
	if r.IsCore != nil {
		isCore = *r.IsCore
	}

	var desc *string
	if r.Desc != nil {
		d := strings.TrimSpace(*r.Desc)
		if d != "" {
			desc = &d
		}
	}

	return csModel.ClassSubjectModel{
		ClassSubjectsMasjidID:        r.MasjidID,
		ClassSubjectsClassID:         r.ClassID,
		ClassSubjectsSubjectID:       r.SubjectID,
		ClassSubjectsTermID:          r.TermID,
		ClassSubjectsOrderIndex:      r.OrderIndex,
		ClassSubjectsHoursPerWeek:    r.HoursPerWeek,
		ClassSubjectsMinPassingScore: r.MinScore,
		ClassSubjectsWeightOnReport:  r.Weight,
		ClassSubjectsIsCore:          isCore,
		ClassSubjectsDesc:            desc,
		ClassSubjectsIsActive:        isActive,
	}
}

func FromClassSubjectModel(m csModel.ClassSubjectModel) ClassSubjectResponse {
	var deletedAt *time.Time
	if m.ClassSubjectsDeletedAt.Valid {
		t := m.ClassSubjectsDeletedAt.Time
		deletedAt = &t
	}

	return ClassSubjectResponse{
		ID:           m.ClassSubjectsID,
		MasjidID:     m.ClassSubjectsMasjidID,
		ClassID:      m.ClassSubjectsClassID,
		SubjectID:    m.ClassSubjectsSubjectID,
		TermID:       m.ClassSubjectsTermID,
		OrderIndex:   m.ClassSubjectsOrderIndex,
		HoursPerWeek: m.ClassSubjectsHoursPerWeek,
		MinScore:     m.ClassSubjectsMinPassingScore,
		Weight:       m.ClassSubjectsWeightOnReport,
		IsCore:       m.ClassSubjectsIsCore,
		Desc:         m.ClassSubjectsDesc,
		IsActive:     m.ClassSubjectsIsActive,
		CreatedAt:    m.ClassSubjectsCreatedAt,
		UpdatedAt:    m.ClassSubjectsUpdatedAt,
		DeletedAt:    deletedAt,
	}
}

func FromClassSubjectModels(list []csModel.ClassSubjectModel) []ClassSubjectResponse {
	out := make([]ClassSubjectResponse, 0, len(list))
	for _, m := range list {
		out = append(out, FromClassSubjectModel(m))
	}
	return out
}

func (r UpdateClassSubjectRequest) Apply(m *csModel.ClassSubjectModel) {
	if r.MasjidID != nil {
		m.ClassSubjectsMasjidID = *r.MasjidID
	}
	if r.ClassID != nil {
		m.ClassSubjectsClassID = *r.ClassID
	}
	if r.SubjectID != nil {
		m.ClassSubjectsSubjectID = *r.SubjectID
	}
	if r.TermID != nil {
		m.ClassSubjectsTermID = r.TermID
	}
	if r.OrderIndex != nil {
		m.ClassSubjectsOrderIndex = r.OrderIndex
	}
	if r.HoursPerWeek != nil {
		m.ClassSubjectsHoursPerWeek = r.HoursPerWeek
	}
	if r.MinScore != nil {
		m.ClassSubjectsMinPassingScore = r.MinScore
	}
	if r.Weight != nil {
		m.ClassSubjectsWeightOnReport = r.Weight
	}
	if r.IsCore != nil {
		m.ClassSubjectsIsCore = *r.IsCore
	}
	if r.Desc != nil {
		d := strings.TrimSpace(*r.Desc)
		if d == "" {
			m.ClassSubjectsDesc = nil
		} else {
			m.ClassSubjectsDesc = &d
		}
	}
	if r.IsActive != nil {
		m.ClassSubjectsIsActive = *r.IsActive
	}
}

/* =========================================================
   4) NESTED: class_subject_books + book (simple books)
   ========================================================= */

type BookLite struct {
	BooksID     uuid.UUID `json:"books_id"`
	BooksTitle  string    `json:"books_title"`
	BooksAuthor *string   `json:"books_author,omitempty"`
	BooksDesc   *string   `json:"books_desc,omitempty"`
	BooksSlug   *string   `json:"books_slug,omitempty"`
}

func bookLiteFromModel(b booksModel.BooksModel) BookLite {
	return BookLite{
		BooksID:     b.BooksID,
		BooksTitle:  b.BooksTitle,
		BooksAuthor: b.BooksAuthor,
		BooksDesc:   b.BooksDesc,
		BooksSlug:   b.BooksSlug,
	}
}

// Disesuaikan: pakai is_active + desc (tanpa valid_from/valid_to/is_primary/notes)
type ClassSubjectBookWithBook struct {
	ClassSubjectBooksID       uuid.UUID `json:"class_subject_books_id"`
	ClassSubjectBooksIsActive bool      `json:"class_subject_books_is_active"`
	ClassSubjectBooksDesc     *string   `json:"class_subject_books_desc,omitempty"`
	Book                      BookLite  `json:"book"`
}

type ClassSubjectWithBooksResponse struct {
	ClassSubjectResponse
	ClassSubjectBooks []ClassSubjectBookWithBook `json:"class_subject_books"`
}

func NewClassSubjectWithBooksResponse(
	cs csModel.ClassSubjectModel,
	links []booksModel.ClassSubjectBookModel,
	bookByID map[uuid.UUID]booksModel.BooksModel,
) ClassSubjectWithBooksResponse {
	base := FromClassSubjectModel(cs)

	out := make([]ClassSubjectBookWithBook, 0, len(links))
	for _, l := range links {
		if b, ok := bookByID[l.ClassSubjectBooksBookID]; ok {
			out = append(out, ClassSubjectBookWithBook{
				ClassSubjectBooksID:       l.ClassSubjectBooksID,
				ClassSubjectBooksIsActive: l.ClassSubjectBooksIsActive,
				ClassSubjectBooksDesc:     l.ClassSubjectBooksDesc,
				Book:                      bookLiteFromModel(b),
			})
		}
	}

	return ClassSubjectWithBooksResponse{
		ClassSubjectResponse: base,
		ClassSubjectBooks:    out,
	}
}
