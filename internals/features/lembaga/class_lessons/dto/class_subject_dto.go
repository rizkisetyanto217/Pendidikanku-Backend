// internals/features/lembaga/class_subjects/dto/class_subject_dto.go
package dto

import (
	"strings"
	"time"

	model "masjidku_backend/internals/features/lembaga/class_lessons/model"

	"github.com/google/uuid"
)

/* =========================================================
   1) REQUEST DTO
   ========================================================= */

// Create
type CreateClassSubjectRequest struct {
	MasjidID     uuid.UUID `json:"class_subjects_masjid_id" validate:"required"`
	ClassID      uuid.UUID `json:"class_subjects_class_id" validate:"required"`
	SubjectID    uuid.UUID `json:"class_subjects_subject_id" validate:"required"`
	OrderIndex   *int      `json:"class_subjects_order_index" validate:"omitempty,min=0"`
	HoursPerWeek *int      `json:"class_subjects_hours_per_week" validate:"omitempty,min=0"`
	MinScore     *int      `json:"class_subjects_min_passing_score" validate:"omitempty,min=0,max=100"`
	Weight       *int      `json:"class_subjects_weight_on_report" validate:"omitempty,min=0"`
	IsCore       *bool     `json:"class_subjects_is_core" validate:"omitempty"`
	AcademicYear *string   `json:"class_subjects_academic_year" validate:"omitempty"`
	Desc         *string   `json:"class_subjects_desc" validate:"omitempty"`
	IsActive     *bool     `json:"class_subjects_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSubjectRequest struct {
	MasjidID     *uuid.UUID `json:"class_subjects_masjid_id" validate:"omitempty"` // biasanya di-force dari token
	ClassID      *uuid.UUID `json:"class_subjects_class_id" validate:"omitempty"`
	SubjectID    *uuid.UUID `json:"class_subjects_subject_id" validate:"omitempty"`
	OrderIndex   *int       `json:"class_subjects_order_index" validate:"omitempty,min=0"`
	HoursPerWeek *int       `json:"class_subjects_hours_per_week" validate:"omitempty,min=0"`
	MinScore     *int       `json:"class_subjects_min_passing_score" validate:"omitempty,min=0,max=100"`
	Weight       *int       `json:"class_subjects_weight_on_report" validate:"omitempty,min=0"`
	IsCore       *bool      `json:"class_subjects_is_core" validate:"omitempty"`
	AcademicYear *string    `json:"class_subjects_academic_year" validate:"omitempty"`
	Desc         *string    `json:"class_subjects_desc" validate:"omitempty"`
	IsActive     *bool      `json:"class_subjects_is_active" validate:"omitempty"`
}

/*
   List query:
   - Filter by active
   - q (opsional, mis. academic_year)
   - Pagination & sort
   - with_deleted
*/
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
   2) RESPONSE DTO
   ========================================================= */

type ClassSubjectResponse struct {
	ID            uuid.UUID  `json:"class_subjects_id"`
	MasjidID      uuid.UUID  `json:"class_subjects_masjid_id"`
	ClassID       uuid.UUID  `json:"class_subjects_class_id"`
	SubjectID     uuid.UUID  `json:"class_subjects_subject_id"`
	OrderIndex    *int       `json:"class_subjects_order_index,omitempty"`
	HoursPerWeek  *int       `json:"class_subjects_hours_per_week,omitempty"`
	MinScore      *int       `json:"class_subjects_min_passing_score,omitempty"`
	Weight        *int       `json:"class_subjects_weight_on_report,omitempty"`
	IsCore        bool       `json:"class_subjects_is_core"`
	AcademicYear  *string    `json:"class_subjects_academic_year,omitempty"`
	Desc          *string    `json:"class_subjects_desc,omitempty"`
	IsActive      bool       `json:"class_subjects_is_active"`
	CreatedAt     time.Time  `json:"class_subjects_created_at"`
	UpdatedAt     *time.Time `json:"class_subjects_updated_at,omitempty"`
	DeletedAt     *time.Time `json:"class_subjects_deleted_at,omitempty"`
}

// Pagination info
type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// List response
type ClassSubjectListResponse struct {
	Items      []ClassSubjectResponse `json:"items"`
	Pagination Pagination             `json:"pagination"`
}

/* =========================================================
   3) MAPPERS
   ========================================================= */

func (r CreateClassSubjectRequest) ToModel() model.ClassSubjectModel {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	isCore := false
	if r.IsCore != nil {
		isCore = *r.IsCore
	}

	var academicYear *string
	if r.AcademicYear != nil {
		ay := strings.TrimSpace(*r.AcademicYear)
		if ay != "" {
			academicYear = &ay
		}
	}

	var desc *string
	if r.Desc != nil {
		d := strings.TrimSpace(*r.Desc)
		if d != "" {
			desc = &d
		}
	}

	return model.ClassSubjectModel{
		ClassSubjectsMasjidID:     r.MasjidID,
		ClassSubjectsClassID:      r.ClassID,
		ClassSubjectsSubjectID:    r.SubjectID,
		ClassSubjectsOrderIndex:   r.OrderIndex,
		ClassSubjectsHoursPerWeek: r.HoursPerWeek,
		ClassSubjectsMinPassingScore: r.MinScore,
		ClassSubjectsWeightOnReport:  r.Weight,
		ClassSubjectsIsCore:          isCore,
		ClassSubjectsAcademicYear:    academicYear,
		ClassSubjectsDesc:            desc,
		ClassSubjectsIsActive:        isActive,
	}
}

func FromClassSubjectModel(m model.ClassSubjectModel) ClassSubjectResponse {
	return ClassSubjectResponse{
		ID:           m.ClassSubjectsID,
		MasjidID:     m.ClassSubjectsMasjidID,
		ClassID:      m.ClassSubjectsClassID,
		SubjectID:    m.ClassSubjectsSubjectID,
		OrderIndex:   m.ClassSubjectsOrderIndex,
		HoursPerWeek: m.ClassSubjectsHoursPerWeek,
		MinScore:     m.ClassSubjectsMinPassingScore,
		Weight:       m.ClassSubjectsWeightOnReport,
		IsCore:       m.ClassSubjectsIsCore,
		AcademicYear: m.ClassSubjectsAcademicYear,
		Desc:         m.ClassSubjectsDesc,
		IsActive:     m.ClassSubjectsIsActive,
		CreatedAt:    m.ClassSubjectsCreatedAt,
		UpdatedAt:    m.ClassSubjectsUpdatedAt,
		DeletedAt:    m.ClassSubjectsDeletedAt,
	}
}

func FromClassSubjectModels(list []model.ClassSubjectModel) []ClassSubjectResponse {
	out := make([]ClassSubjectResponse, 0, len(list))
	for _, m := range list {
		out = append(out, FromClassSubjectModel(m))
	}
	return out
}

/* =========================================================
   4) APPLY (partial update helper)
   ========================================================= */

func (r UpdateClassSubjectRequest) Apply(m *model.ClassSubjectModel) {
	if r.MasjidID != nil {
		m.ClassSubjectsMasjidID = *r.MasjidID
	}
	if r.ClassID != nil {
		m.ClassSubjectsClassID = *r.ClassID
	}
	if r.SubjectID != nil {
		m.ClassSubjectsSubjectID = *r.SubjectID
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
	if r.AcademicYear != nil {
		ay := strings.TrimSpace(*r.AcademicYear)
		if ay == "" {
			m.ClassSubjectsAcademicYear = nil
		} else {
			m.ClassSubjectsAcademicYear = &ay
		}
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
