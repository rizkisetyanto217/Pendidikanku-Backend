// file: internals/features/lembaga/class_subjects/dto/class_subject_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	linkModel "schoolku_backend/internals/features/school/academics/books/model"
	csModel "schoolku_backend/internals/features/school/academics/subjects/model"
)

/* =========================================================
   0) helpers
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

func timePtrOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func intToInt16Ptr(p *int) *int16 {
	if p == nil {
		return nil
	}
	v := int16(*p)
	return &v
}

func int16ToIntPtr(p *int16) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

/* =========================================================
   1) REQUEST DTO
   ========================================================= */

type CreateClassSubjectRequest struct {
	SchoolID  uuid.UUID `json:"class_subject_school_id"  validate:"required"`
	ParentID  uuid.UUID `json:"class_subject_parent_id"  validate:"required"`
	SubjectID uuid.UUID `json:"class_subject_subject_id" validate:"required"`

	// slug
	Slug *string `json:"class_subject_slug" validate:"omitempty,max=160"`

	// kurikulum
	OrderIndex   *int    `json:"class_subject_order_index"       validate:"omitempty,min=0"`
	HoursPerWeek *int    `json:"class_subject_hours_per_week"    validate:"omitempty,min=0"`
	MinScore     *int    `json:"class_subject_min_passing_score" validate:"omitempty,min=0,max=100"`
	Weight       *int    `json:"class_subject_weight_on_report"  validate:"omitempty,min=0"`
	IsCore       *bool   `json:"class_subject_is_core"           validate:"omitempty"`
	Desc         *string `json:"class_subject_desc"              validate:"omitempty"`

	// bobot penilaian (0..100)
	WeightAssignment     *int `json:"class_subject_weight_assignment"       validate:"omitempty,min=0,max=100"`
	WeightQuiz           *int `json:"class_subject_weight_quiz"             validate:"omitempty,min=0,max=100"`
	WeightMid            *int `json:"class_subject_weight_mid"              validate:"omitempty,min=0,max=100"`
	WeightFinal          *int `json:"class_subject_weight_final"            validate:"omitempty,min=0,max=100"`
	MinAttendancePercent *int `json:"class_subject_min_attendance_percent"  validate:"omitempty,min=0,max=100"`

	IsActive *bool `json:"class_subject_is_active" validate:"omitempty"`
}

type UpdateClassSubjectRequest struct {
	SchoolID  *uuid.UUID `json:"class_subject_school_id"  validate:"omitempty"`
	ParentID  *uuid.UUID `json:"class_subject_parent_id"  validate:"omitempty"`
	SubjectID *uuid.UUID `json:"class_subject_subject_id" validate:"omitempty"`

	// slug
	Slug *string `json:"class_subject_slug" validate:"omitempty,max=160"`

	// kurikulum
	OrderIndex   *int    `json:"class_subject_order_index"       validate:"omitempty,min=0"`
	HoursPerWeek *int    `json:"class_subject_hours_per_week"    validate:"omitempty,min=0"`
	MinScore     *int    `json:"class_subject_min_passing_score" validate:"omitempty,min=0,max=100"`
	Weight       *int    `json:"class_subject_weight_on_report"  validate:"omitempty,min=0"`
	IsCore       *bool   `json:"class_subject_is_core"           validate:"omitempty"`
	Desc         *string `json:"class_subject_desc"              validate:"omitempty"`

	// bobot penilaian
	WeightAssignment     *int `json:"class_subject_weight_assignment"       validate:"omitempty,min=0,max=100"`
	WeightQuiz           *int `json:"class_subject_weight_quiz"             validate:"omitempty,min=0,max=100"`
	WeightMid            *int `json:"class_subject_weight_mid"              validate:"omitempty,min=0,max=100"`
	WeightFinal          *int `json:"class_subject_weight_final"            validate:"omitempty,min=0,max=100"`
	MinAttendancePercent *int `json:"class_subject_min_attendance_percent"  validate:"omitempty,min=0,max=100"`

	IsActive *bool `json:"class_subject_is_active" validate:"omitempty"`
}

type ListClassSubjectQuery struct {
	Limit       *int    `query:"limit"         validate:"omitempty,min=1,max=200"`
	Offset      *int    `query:"offset"        validate:"omitempty,min=0"`
	IsActive    *bool   `query:"is_active"     validate:"omitempty"`
	Q           *string `query:"q"             validate:"omitempty,max=100"`
	OrderBy     *string `query:"order_by"      validate:"omitempty,oneof=order_index created_at updated_at"`
	Sort        *string `query:"sort"          validate:"omitempty,oneof=asc desc"`
	WithDeleted *bool   `query:"with_deleted"  validate:"omitempty"`
	// (opsional) filter tambahan
	SchoolID  *uuid.UUID `query:"school_id"     validate:"omitempty"`
	ParentID  *uuid.UUID `query:"parent_id"     validate:"omitempty"`
	SubjectID *uuid.UUID `query:"subject_id"   validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO (basic + snapshots)
   ========================================================= */

type ClassSubjectResponse struct {
	ID        uuid.UUID `json:"class_subject_id"`
	SchoolID  uuid.UUID `json:"class_subject_school_id"`
	ParentID  uuid.UUID `json:"class_subject_parent_id"`
	SubjectID uuid.UUID `json:"class_subject_subject_id"`

	// slug
	Slug *string `json:"class_subject_slug,omitempty"`

	// kurikulum
	OrderIndex   *int    `json:"class_subject_order_index,omitempty"`
	HoursPerWeek *int    `json:"class_subject_hours_per_week,omitempty"`
	MinScore     *int    `json:"class_subject_min_passing_score,omitempty"`
	Weight       *int    `json:"class_subject_weight_on_report,omitempty"`
	IsCore       bool    `json:"class_subject_is_core"`
	Desc         *string `json:"class_subject_desc,omitempty"`

	// bobot (int untuk JSON, disimpan sebagai SMALLINT di DB)
	WeightAssignment     *int `json:"class_subject_weight_assignment,omitempty"`
	WeightQuiz           *int `json:"class_subject_weight_quiz,omitempty"`
	WeightMid            *int `json:"class_subject_weight_mid,omitempty"`
	WeightFinal          *int `json:"class_subject_weight_final,omitempty"`
	MinAttendancePercent *int `json:"class_subject_min_attendance_percent,omitempty"`

	// ============ Snapshots: subjects ============
	SubjectNameSnapshot *string `json:"class_subject_subject_name_snapshot,omitempty"`
	SubjectCodeSnapshot *string `json:"class_subject_subject_code_snapshot,omitempty"`
	SubjectSlugSnapshot *string `json:"class_subject_subject_slug_snapshot,omitempty"`
	SubjectURLSnapshot  *string `json:"class_subject_subject_url_snapshot,omitempty"`

	// ============ Snapshots: class_parent ============
	ParentCodeSnapshot  *string `json:"class_subject_parent_code_snapshot,omitempty"`
	ParentSlugSnapshot  *string `json:"class_subject_parent_slug_snapshot,omitempty"`
	ParentLevelSnapshot *int16  `json:"class_subject_parent_level_snapshot,omitempty"`
	ParentURLSnapshot   *string `json:"class_subject_parent_url_snapshot,omitempty"`
	ParentNameSnapshot  *string `json:"class_subject_parent_name_snapshot,omitempty"`

	// status & timestamps
	IsActive  bool       `json:"class_subject_is_active"`
	CreatedAt time.Time  `json:"class_subject_created_at"`
	UpdatedAt *time.Time `json:"class_subject_updated_at,omitempty"`
	DeletedAt *time.Time `json:"class_subject_deleted_at,omitempty"`
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

	return csModel.ClassSubjectModel{
		ClassSubjectSchoolID:  r.SchoolID,
		ClassSubjectParentID:  r.ParentID,
		ClassSubjectSubjectID: r.SubjectID,

		// slug (trim)
		ClassSubjectSlug: trimPtr(r.Slug),

		// kurikulum
		ClassSubjectOrderIndex:      r.OrderIndex,
		ClassSubjectHoursPerWeek:    r.HoursPerWeek,
		ClassSubjectMinPassingScore: r.MinScore,
		ClassSubjectWeightOnReport:  r.Weight,
		ClassSubjectIsCore:          isCore,
		ClassSubjectDesc:            trimPtr(r.Desc),

		// bobot (konversi ke *int16)
		ClassSubjectWeightAssignment:     intToInt16Ptr(r.WeightAssignment),
		ClassSubjectWeightQuiz:           intToInt16Ptr(r.WeightQuiz),
		ClassSubjectWeightMid:            intToInt16Ptr(r.WeightMid),
		ClassSubjectWeightFinal:          intToInt16Ptr(r.WeightFinal),
		ClassSubjectMinAttendancePercent: intToInt16Ptr(r.MinAttendancePercent),

		// status
		ClassSubjectIsActive: isActive,
	}
}

func FromClassSubjectModel(m csModel.ClassSubjectModel) ClassSubjectResponse {
	var deletedAt *time.Time
	if m.ClassSubjectDeletedAt.Valid {
		t := m.ClassSubjectDeletedAt.Time
		deletedAt = &t
	}

	return ClassSubjectResponse{
		ID:        m.ClassSubjectID,
		SchoolID:  m.ClassSubjectSchoolID,
		ParentID:  m.ClassSubjectParentID,
		SubjectID: m.ClassSubjectSubjectID,

		Slug: m.ClassSubjectSlug,

		OrderIndex:   m.ClassSubjectOrderIndex,
		HoursPerWeek: m.ClassSubjectHoursPerWeek,
		MinScore:     m.ClassSubjectMinPassingScore,
		Weight:       m.ClassSubjectWeightOnReport,
		IsCore:       m.ClassSubjectIsCore,
		Desc:         m.ClassSubjectDesc,

		// bobot (konversi *int16 → *int untuk JSON)
		WeightAssignment:     int16ToIntPtr(m.ClassSubjectWeightAssignment),
		WeightQuiz:           int16ToIntPtr(m.ClassSubjectWeightQuiz),
		WeightMid:            int16ToIntPtr(m.ClassSubjectWeightMid),
		WeightFinal:          int16ToIntPtr(m.ClassSubjectWeightFinal),
		MinAttendancePercent: int16ToIntPtr(m.ClassSubjectMinAttendancePercent),

		// snapshots: subjects
		SubjectNameSnapshot: m.ClassSubjectSubjectNameSnapshot,
		SubjectCodeSnapshot: m.ClassSubjectSubjectCodeSnapshot,
		SubjectSlugSnapshot: m.ClassSubjectSubjectSlugSnapshot,
		SubjectURLSnapshot:  m.ClassSubjectSubjectURLSnapshot,

		// snapshots: class_parent
		ParentCodeSnapshot:  m.ClassSubjectParentCodeSnapshot,
		ParentSlugSnapshot:  m.ClassSubjectParentSlugSnapshot,
		ParentLevelSnapshot: m.ClassSubjectParentLevelSnapshot,
		ParentURLSnapshot:   m.ClassSubjectParentURLSnapshot,
		ParentNameSnapshot:  m.ClassSubjectParentNameSnapshot,

		IsActive:  m.ClassSubjectIsActive,
		CreatedAt: m.ClassSubjectCreatedAt,
		UpdatedAt: timePtrOrNil(m.ClassSubjectUpdatedAt),
		DeletedAt: deletedAt,
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
	if r.SchoolID != nil {
		m.ClassSubjectSchoolID = *r.SchoolID
	}
	if r.ParentID != nil {
		m.ClassSubjectParentID = *r.ParentID
	}
	if r.SubjectID != nil {
		m.ClassSubjectSubjectID = *r.SubjectID
	}
	if r.Slug != nil {
		m.ClassSubjectSlug = trimPtr(r.Slug)
	}

	if r.OrderIndex != nil {
		m.ClassSubjectOrderIndex = r.OrderIndex
	}
	if r.HoursPerWeek != nil {
		m.ClassSubjectHoursPerWeek = r.HoursPerWeek
	}
	if r.MinScore != nil {
		m.ClassSubjectMinPassingScore = r.MinScore
	}
	if r.Weight != nil {
		m.ClassSubjectWeightOnReport = r.Weight
	}
	if r.IsCore != nil {
		m.ClassSubjectIsCore = *r.IsCore
	}
	if r.Desc != nil {
		m.ClassSubjectDesc = trimPtr(r.Desc)
	}

	// bobot (*int → *int16)
	if r.WeightAssignment != nil {
		m.ClassSubjectWeightAssignment = intToInt16Ptr(r.WeightAssignment)
	}
	if r.WeightQuiz != nil {
		m.ClassSubjectWeightQuiz = intToInt16Ptr(r.WeightQuiz)
	}
	if r.WeightMid != nil {
		m.ClassSubjectWeightMid = intToInt16Ptr(r.WeightMid)
	}
	if r.WeightFinal != nil {
		m.ClassSubjectWeightFinal = intToInt16Ptr(r.WeightFinal)
	}
	if r.MinAttendancePercent != nil {
		m.ClassSubjectMinAttendancePercent = intToInt16Ptr(r.MinAttendancePercent)
	}

	if r.IsActive != nil {
		m.ClassSubjectIsActive = *r.IsActive
	}
}

/* =========================================================
   4) NESTED: class_subject_books + book (simple books)
   ========================================================= */

type BookLite struct {
	BookID     uuid.UUID `json:"book_id"`
	BookTitle  string    `json:"book_title"`
	BookAuthor *string   `json:"book_author,omitempty"`
	BookDesc   *string   `json:"book_desc,omitempty"`
	BookSlug   *string   `json:"book_slug,omitempty"`
}

func bookLiteFromModel(b linkModel.BookModel) BookLite {
	return BookLite{
		BookID:     b.BookID,
		BookTitle:  b.BookTitle,
		BookAuthor: b.BookAuthor,
		BookDesc:   b.BookDesc,
		BookSlug:   b.BookSlug,
	}
}

// Disesuaikan: pakai is_active + desc dengan penamaan singular
type ClassSubjectBookWithBook struct {
	ClassSubjectBookID       uuid.UUID `json:"class_subject_book_id"`
	ClassSubjectBookIsActive bool      `json:"class_subject_book_is_active"`
	ClassSubjectBookDesc     *string   `json:"class_subject_book_desc,omitempty"`
	Book                     BookLite  `json:"book"`
}

type ClassSubjectWithBooksResponse struct {
	ClassSubjectResponse
	ClassSubjectBooks []ClassSubjectBookWithBook `json:"class_subject_books"`
}

func NewClassSubjectWithBooksResponse(
	cs csModel.ClassSubjectModel,
	links []linkModel.ClassSubjectBookModel,
	bookByID map[uuid.UUID]linkModel.BookModel,
) ClassSubjectWithBooksResponse {
	base := FromClassSubjectModel(cs)

	out := make([]ClassSubjectBookWithBook, 0, len(links))
	for _, l := range links {
		if b, ok := bookByID[l.ClassSubjectBookBookID]; ok {
			out = append(out, ClassSubjectBookWithBook{
				ClassSubjectBookID:       l.ClassSubjectBookID,
				ClassSubjectBookIsActive: l.ClassSubjectBookIsActive,
				ClassSubjectBookDesc:     l.ClassSubjectBookDesc,
				Book:                     bookLiteFromModel(b),
			})
		}
	}

	return ClassSubjectWithBooksResponse{
		ClassSubjectResponse: base,
		ClassSubjectBooks:    out,
	}
}
