// internals/features/lembaga/subjects/main/dto/subject_dto.go
package dto

import (
	"strings"
	"time"

	subjectModel "masjidku_backend/internals/features/school/subject_books/subject/model"

	"github.com/google/uuid"
)

/* =========================================================
   1) REQUEST DTO
   ========================================================= */

// Create
type CreateSubjectRequest struct {
	MasjidID uuid.UUID `json:"subjects_masjid_id" validate:"required"`
	Code     string    `json:"subjects_code" validate:"required,max=40"`
	Name     string    `json:"subjects_name" validate:"required,max=120"`
	Slug     *string   `json:"subjects_slug" validate:"omitempty,max=160"`
	Desc     *string   `json:"subjects_desc" validate:"omitempty"`
	IsActive *bool     `json:"subjects_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateSubjectRequest struct {
	MasjidID *uuid.UUID `json:"subjects_masjid_id" validate:"omitempty"` // akan di-force di controller
	Code     *string    `json:"subjects_code" validate:"omitempty,max=40"`
	Name     *string    `json:"subjects_name" validate:"omitempty,max=120"`
	Slug     *string    `json:"subjects_slug" validate:"omitempty,max=160"`
	Desc     *string    `json:"subjects_desc" validate:"omitempty"`
	IsActive *bool      `json:"subjects_is_active" validate:"omitempty"`
}

/*
List query:
- Filter by active
- q untuk pencarian sederhana (code/name/slug)
- Pagination & sort sederhana
- with_deleted: jika true, sertakan baris yang soft-deleted (default: false)
*/
type ListSubjectQuery struct {
	Limit       *int    `query:"limit" validate:"omitempty,min=1,max=200"`
	Offset      *int    `query:"offset" validate:"omitempty,min=0"`
	IsActive    *bool   `query:"is_active" validate:"omitempty"`
	Q           *string `query:"q" validate:"omitempty,max=100"`
	OrderBy     *string `query:"order_by" validate:"omitempty,oneof=code name slug created_at updated_at"`
	Sort        *string `query:"sort" validate:"omitempty,oneof=asc desc"`
	WithDeleted *bool   `query:"with_deleted" validate:"omitempty"`
}

/* =========================================================
   2) RESPONSE DTO
   ========================================================= */

type SubjectResponse struct {
	ID        uuid.UUID  `json:"subjects_id"`
	MasjidID  uuid.UUID  `json:"subjects_masjid_id"`
	Code      string     `json:"subjects_code"`
	Name      string     `json:"subjects_name"`
	Slug      *string    `json:"subjects_slug,omitempty"`
	Desc      *string    `json:"subjects_desc,omitempty"`
	IsActive  bool       `json:"subjects_is_active"`
	CreatedAt time.Time  `json:"subjects_created_at"`
	UpdatedAt time.Time  `json:"subjects_updated_at"` // <- non-pointer, selaras model NOT NULL
	DeletedAt *time.Time `json:"subjects_deleted_at,omitempty"`
}

// List response + meta
type SubjectListResponse struct {
	Items []SubjectResponse `json:"items"`
	Meta  ListMeta          `json:"meta"`
}

type ListMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

/* =========================================================
   3) MAPPERS
   ========================================================= */

func (r CreateSubjectRequest) ToModel() subjectModel.SubjectsModel {
	// Trim string agar bersih
	code := strings.TrimSpace(r.Code)
	name := strings.TrimSpace(r.Name)

	var desc *string
	if r.Desc != nil {
		if d := strings.TrimSpace(*r.Desc); d != "" {
			desc = &d
		}
	}

	// slug opsional
	var slug *string
	if r.Slug != nil {
		if s := strings.TrimSpace(*r.Slug); s != "" {
			slug = &s
		}
	}

	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	return subjectModel.SubjectsModel{
		SubjectsMasjidID: r.MasjidID,
		SubjectsCode:     code,
		SubjectsName:     name,
		SubjectsDesc:     desc,
		SubjectsSlug:     slug,
		SubjectsIsActive: isActive,
	}
}

func FromSubjectModel(m subjectModel.SubjectsModel) SubjectResponse {
	var deletedAt *time.Time
	if m.SubjectsDeletedAt.Valid {
		deletedAt = &m.SubjectsDeletedAt.Time
	}
	return SubjectResponse{
		ID:        m.SubjectsID,
		MasjidID:  m.SubjectsMasjidID,
		Code:      m.SubjectsCode,
		Name:      m.SubjectsName,
		Slug:      m.SubjectsSlug,
		Desc:      m.SubjectsDesc,
		IsActive:  m.SubjectsIsActive,
		CreatedAt: m.SubjectsCreatedAt,
		UpdatedAt: m.SubjectsUpdatedAt,
		DeletedAt: deletedAt, // sekarang aman
	}
}


func FromSubjectModels(models []subjectModel.SubjectsModel) []SubjectResponse {
	out := make([]SubjectResponse, 0, len(models))
	for _, m := range models {
		out = append(out, FromSubjectModel(m))
	}
	return out
}

/* =========================================================
   4) APPLY (partial update helper)
   ========================================================= */

func (r UpdateSubjectRequest) Apply(m *subjectModel.SubjectsModel) {
	if r.MasjidID != nil {
		m.SubjectsMasjidID = *r.MasjidID // biasanya di-force di controller
	}
	if r.Code != nil {
		m.SubjectsCode = strings.TrimSpace(*r.Code)
	}
	if r.Name != nil {
		m.SubjectsName = strings.TrimSpace(*r.Name)
	}
	if r.Slug != nil {
		s := strings.TrimSpace(*r.Slug)
		if s == "" {
			m.SubjectsSlug = nil
		} else {
			m.SubjectsSlug = &s
		}
	}
	if r.Desc != nil {
		d := strings.TrimSpace(*r.Desc)
		if d == "" {
			m.SubjectsDesc = nil
		} else {
			m.SubjectsDesc = &d
		}
	}
	if r.IsActive != nil {
		m.SubjectsIsActive = *r.IsActive
	}
}
