// file: internals/features/academics/terms/dto/dto.go
package dto

import (
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/academics/academic_terms/model"

	"github.com/google/uuid"
)

/* ===================== REQUEST DTO ===================== */

type AcademicTermCreateDTO struct {
	AcademicTermsAcademicYear string     `json:"academic_terms_academic_year" validate:"required,min=4"`
	AcademicTermsName         string     `json:"academic_terms_name"          validate:"required,oneof=Ganjil Genap Pendek Khusus"`
	AcademicTermsStartDate    time.Time  `json:"academic_terms_start_date"    validate:"required"`
	AcademicTermsEndDate      time.Time  `json:"academic_terms_end_date"      validate:"required,gtefield=AcademicTermsStartDate"`
	AcademicTermsIsActive     *bool      `json:"academic_terms_is_active,omitempty"`
	AcademicTermsAngkatan     *int       `json:"academic_terms_angkatan,omitempty" validate:"omitempty,gt=0"`

	// Kolom baru (opsional)
	AcademicTermsCode        string  `json:"academic_terms_code,omitempty"        validate:"omitempty,min=1,max=24"`
	AcademicTermsSlug        string  `json:"academic_terms_slug,omitempty"        validate:"omitempty,min=3,max=50"`
	AcademicTermsDescription *string `json:"academic_terms_description,omitempty" validate:"omitempty"`
}

type AcademicTermUpdateDTO struct {
	AcademicTermsAcademicYear *string    `json:"academic_terms_academic_year,omitempty" validate:"omitempty,min=4"`
	AcademicTermsName *string `validate:"omitempty,min=1,max=60"`
	AcademicTermsStartDate    *time.Time `json:"academic_terms_start_date,omitempty"`
	AcademicTermsEndDate      *time.Time `json:"academic_terms_end_date,omitempty"`
	AcademicTermsIsActive     *bool      `json:"academic_terms_is_active,omitempty"`
	AcademicTermsAngkatan     *int       `json:"academic_terms_angkatan,omitempty" validate:"omitempty,gt=0"`

	// Kolom baru (opsional)
	AcademicTermsCode        *string `json:"academic_terms_code,omitempty"        validate:"omitempty,min=1,max=24"`
	AcademicTermsSlug        *string `json:"academic_terms_slug,omitempty"        validate:"omitempty,min=3,max=50"`
	AcademicTermsDescription *string `json:"academic_terms_description,omitempty" validate:"omitempty"`
}

/* ========== LIST/FILTER (query) ========== */

type AcademicTermFilterDTO struct {
	ID       *string `query:"id"        validate:"omitempty,uuid4"`
	Year     *string `query:"year"      validate:"omitempty,min=4"`
	Name     *string `query:"name"      validate:"omitempty,oneof=Ganjil Genap Pendek Khusus"`
	Code     *string `query:"code"      validate:"omitempty,min=1,max=24"`
	Slug     *string `query:"slug"      validate:"omitempty,min=3,max=50"`
	Active   *bool   `query:"active"    validate:"omitempty"`
	Angkatan *int    `query:"angkatan"  validate:"omitempty,gt=0"`

	Page     int     `query:"page"      validate:"omitempty,min=1"`
	PageSize int     `query:"page_size" validate:"omitempty,min=1,max=200"`
	SortBy   *string `query:"sort_by"   validate:"omitempty,oneof=created_at updated_at start_date end_date name year angkatan code slug"`
	SortDir  *string `query:"sort_dir"  validate:"omitempty,oneof=asc desc"`
}

/* ===================== RESPONSE DTO ===================== */

type AcademicTermResponseDTO struct {
	AcademicTermsID           uuid.UUID  `json:"academic_terms_id"`
	AcademicTermsMasjidID     uuid.UUID  `json:"academic_terms_masjid_id"`
	AcademicTermsAcademicYear string     `json:"academic_terms_academic_year"`
	AcademicTermsName         string     `json:"academic_terms_name"`
	AcademicTermsStartDate    time.Time  `json:"academic_terms_start_date"`
	AcademicTermsEndDate      time.Time  `json:"academic_terms_end_date"`
	AcademicTermsIsActive     bool       `json:"academic_terms_is_active"`

	AcademicTermsAngkatan     *int       `json:"academic_terms_angkatan,omitempty"`

	// Kolom baru
	AcademicTermsCode        string  `json:"academic_terms_code,omitempty"`
	AcademicTermsSlug        string  `json:"academic_terms_slug,omitempty"`
	AcademicTermsDescription *string `json:"academic_terms_description,omitempty"`

	// Read-only: diisi DB (generated)
	AcademicTermsPeriod *string `json:"academic_terms_period,omitempty"`

	AcademicTermsCreatedAt time.Time  `json:"academic_terms_created_at"`
	AcademicTermsUpdatedAt time.Time  `json:"academic_terms_updated_at"`
	AcademicTermsDeletedAt *time.Time `json:"academic_terms_deleted_at,omitempty"`
}

/* ===================== HELPERS ===================== */

func (p *AcademicTermCreateDTO) Normalize() {
	p.AcademicTermsAcademicYear = strings.TrimSpace(p.AcademicTermsAcademicYear)
	p.AcademicTermsName = strings.TrimSpace(p.AcademicTermsName)
	p.AcademicTermsCode = strings.TrimSpace(p.AcademicTermsCode)

	// slug: trim + lower
	slug := strings.TrimSpace(p.AcademicTermsSlug)
	if slug != "" {
		slug = strings.ToLower(slug)
		p.AcademicTermsSlug = slug
	}

	// description: trim (kalau kosong jadikan nil)
	if p.AcademicTermsDescription != nil {
		d := strings.TrimSpace(*p.AcademicTermsDescription)
		if d == "" {
			p.AcademicTermsDescription = nil
		} else {
			p.AcademicTermsDescription = &d
		}
	}
}

func (p *AcademicTermCreateDTO) WantsActive() bool {
	return p.AcademicTermsIsActive == nil || *p.AcademicTermsIsActive
}

func (p *AcademicTermCreateDTO) ToModel(masjidID uuid.UUID) model.AcademicTermModel {
	isActive := true
	if p.AcademicTermsIsActive != nil {
		isActive = *p.AcademicTermsIsActive
	}
	return model.AcademicTermModel{
		AcademicTermsMasjidID:     masjidID,
		AcademicTermsAcademicYear: p.AcademicTermsAcademicYear,
		AcademicTermsName:         p.AcademicTermsName,
		AcademicTermsStartDate:    p.AcademicTermsStartDate,
		AcademicTermsEndDate:      p.AcademicTermsEndDate,
		AcademicTermsIsActive:     isActive,
		AcademicTermsAngkatan:     p.AcademicTermsAngkatan,

		// Kolom baru
		AcademicTermsCode:        p.AcademicTermsCode,
		AcademicTermsSlug:        p.AcademicTermsSlug,
		AcademicTermsDescription: derefOrEmpty(p.AcademicTermsDescription),
	}
}

func (u *AcademicTermUpdateDTO) ApplyUpdates(ent *model.AcademicTermModel) {
	if u.AcademicTermsAcademicYear != nil {
		ent.AcademicTermsAcademicYear = strings.TrimSpace(*u.AcademicTermsAcademicYear)
	}
	if u.AcademicTermsName != nil {
		ent.AcademicTermsName = strings.TrimSpace(*u.AcademicTermsName)
	}
	if u.AcademicTermsStartDate != nil {
		ent.AcademicTermsStartDate = *u.AcademicTermsStartDate
	}
	if u.AcademicTermsEndDate != nil {
		ent.AcademicTermsEndDate = *u.AcademicTermsEndDate
	}
	if u.AcademicTermsIsActive != nil {
		ent.AcademicTermsIsActive = *u.AcademicTermsIsActive
	}
	if u.AcademicTermsAngkatan != nil {
		ent.AcademicTermsAngkatan = u.AcademicTermsAngkatan
	}
	// Kolom baru
	if u.AcademicTermsCode != nil {
		ent.AcademicTermsCode = strings.TrimSpace(*u.AcademicTermsCode)
	}
	if u.AcademicTermsSlug != nil {
		slug := strings.TrimSpace(*u.AcademicTermsSlug)
		if slug != "" {
			slug = strings.ToLower(slug)
		}
		ent.AcademicTermsSlug = slug
	}
	if u.AcademicTermsDescription != nil {
		desc := strings.TrimSpace(*u.AcademicTermsDescription)
		if desc == "" {
			ent.AcademicTermsDescription = ""
		} else {
			ent.AcademicTermsDescription = desc
		}
	}
}

func (q *AcademicTermFilterDTO) Normalize() {
	if q.SortDir != nil {
		s := strings.ToLower(strings.TrimSpace(*q.SortDir))
		q.SortDir = &s
	}
	if q.SortBy != nil {
		s := strings.ToLower(strings.TrimSpace(*q.SortBy))
		q.SortBy = &s
	}
	if q.Year != nil {
		s := strings.TrimSpace(*q.Year)
		q.Year = &s
	}
	if q.Name != nil {
		s := strings.TrimSpace(*q.Name)
		q.Name = &s
	}
	if q.Code != nil {
		s := strings.TrimSpace(*q.Code)
		q.Code = &s
	}
	if q.Slug != nil {
		s := strings.ToLower(strings.TrimSpace(*q.Slug))
		q.Slug = &s
	}
}

/* ===================== MAPPERS ===================== */

func FromModel(ent model.AcademicTermModel) AcademicTermResponseDTO {
	var deletedAt *time.Time
	if ent.AcademicTermsDeletedAt.Valid {
		t := ent.AcademicTermsDeletedAt.Time
		deletedAt = &t
	}
	// handle description: kosongkan jadi nil di response
	var desc *string
	if strings.TrimSpace(ent.AcademicTermsDescription) != "" {
		d := ent.AcademicTermsDescription
		desc = &d
	}

	return AcademicTermResponseDTO{
		AcademicTermsID:           ent.AcademicTermsID,
		AcademicTermsMasjidID:     ent.AcademicTermsMasjidID,
		AcademicTermsAcademicYear: ent.AcademicTermsAcademicYear,
		AcademicTermsName:         ent.AcademicTermsName,
		AcademicTermsStartDate:    ent.AcademicTermsStartDate,
		AcademicTermsEndDate:      ent.AcademicTermsEndDate,
		AcademicTermsIsActive:     ent.AcademicTermsIsActive,
		AcademicTermsAngkatan:     ent.AcademicTermsAngkatan,

		AcademicTermsCode:        ent.AcademicTermsCode,
		AcademicTermsSlug:        ent.AcademicTermsSlug,
		AcademicTermsDescription: desc,

		AcademicTermsPeriod:    ent.AcademicTermsPeriod,
		AcademicTermsCreatedAt: ent.AcademicTermsCreatedAt,
		AcademicTermsUpdatedAt: ent.AcademicTermsUpdatedAt,
		AcademicTermsDeletedAt: deletedAt,
	}
}

func FromModels(list []model.AcademicTermModel) []AcademicTermResponseDTO {
	out := make([]AcademicTermResponseDTO, 0, len(list))
	for _, it := range list {
		out = append(out, FromModel(it))
	}
	return out
}

/* ===================== UTIL ===================== */

func derefOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
