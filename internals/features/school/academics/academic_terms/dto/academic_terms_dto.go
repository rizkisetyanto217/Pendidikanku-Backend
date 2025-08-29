// file: internals/features/lembaga/academics/academic_year/dto/dto.go
package dto

import (
	"strings"
	"time"

	"masjidku_backend/internals/features/school/academics/academic_terms/model"

	"github.com/google/uuid"
)

// =======================
// Request DTO
// =======================

type AcademicTermCreateDTO struct {
	AcademicTermsAcademicYear string    `json:"academic_terms_academic_year" validate:"required,min=4"`
	// Terima hanya 4 opsi ini
	AcademicTermsName         string    `json:"academic_terms_name"          validate:"required,oneof=Ganjil Genap Pendek Khusus"`
	AcademicTermsStartDate    time.Time `json:"academic_terms_start_date"    validate:"required"`
	// gtefield agar sejalan dg DB CHECK (end >= start)
	AcademicTermsEndDate      time.Time `json:"academic_terms_end_date"      validate:"required,gtefield=AcademicTermsStartDate"`
	// pointer: bedakan "tidak dikirim" vs "false"
	AcademicTermsIsActive     *bool     `json:"academic_terms_is_active,omitempty"`
}

type AcademicTermUpdateDTO struct {
	AcademicTermsAcademicYear *string    `json:"academic_terms_academic_year,omitempty" validate:"omitempty,min=4"`
	AcademicTermsName         *string    `json:"academic_terms_name,omitempty"          validate:"omitempty,oneof=Ganjil Genap Pendek Khusus"`
	AcademicTermsStartDate    *time.Time `json:"academic_terms_start_date,omitempty"`
	AcademicTermsEndDate      *time.Time `json:"academic_terms_end_date,omitempty"`
	AcademicTermsIsActive     *bool      `json:"academic_terms_is_active,omitempty"`
}

// (opsional) filter list
type AcademicTermFilterDTO struct {
	// NEW: filter by term ID (query:?id=UUID)
	ID       *string `query:"id"        validate:"omitempty,uuid4"`
	Year     *string `query:"year"      validate:"omitempty,min=4"`
	Name     *string `query:"name"      validate:"omitempty,oneof=Ganjil Genap Pendek Khusus"`
	Active   *bool   `query:"active"    validate:"omitempty"`
	Page     int     `query:"page"      validate:"omitempty,min=1"`
	PageSize int     `query:"page_size" validate:"omitempty,min=1,max=200"`
	SortBy   *string `query:"sort_by"   validate:"omitempty,oneof=created_at updated_at start_date end_date name year"`
	SortDir  *string `query:"sort_dir"  validate:"omitempty,oneof=asc desc"`
}

// =======================
// Response DTO
// =======================

type AcademicTermResponseDTO struct {
	AcademicTermsID           uuid.UUID  `json:"academic_terms_id"`
	AcademicTermsMasjidID     uuid.UUID  `json:"academic_terms_masjid_id"`
	AcademicTermsAcademicYear string     `json:"academic_terms_academic_year"`
	AcademicTermsName         string     `json:"academic_terms_name"`
	AcademicTermsStartDate    time.Time  `json:"academic_terms_start_date"`
	AcademicTermsEndDate      time.Time  `json:"academic_terms_end_date"`
	AcademicTermsIsActive     bool       `json:"academic_terms_is_active"`

	// Read-only: diisi oleh DB (generated column)
	AcademicTermsPeriod *string `json:"academic_terms_period,omitempty"`

	AcademicTermsCreatedAt time.Time  `json:"academic_terms_created_at"`
	AcademicTermsUpdatedAt *time.Time `json:"academic_terms_updated_at,omitempty"`
	AcademicTermsDeletedAt *time.Time `json:"academic_terms_deleted_at,omitempty"`
}

// =======================
// Helpers
// =======================

func (p *AcademicTermCreateDTO) Normalize() {
	p.AcademicTermsAcademicYear = strings.TrimSpace(p.AcademicTermsAcademicYear)
	p.AcademicTermsName = strings.TrimSpace(p.AcademicTermsName)
}

func (p *AcademicTermCreateDTO) WantsActive() bool {
	return p.AcademicTermsIsActive == nil || *p.AcademicTermsIsActive
}

func (p *AcademicTermCreateDTO) ToModel(masjidID uuid.UUID) model.AcademicTermModel {
	isActive := true
	if p.AcademicTermsIsActive != nil {
		isActive = *p.AcademicTermsIsActive // hormati input eksplisit
	}
	return model.AcademicTermModel{
		AcademicTermsMasjidID:     masjidID,
		AcademicTermsAcademicYear: p.AcademicTermsAcademicYear,
		AcademicTermsName:         p.AcademicTermsName,
		AcademicTermsStartDate:    p.AcademicTermsStartDate,
		AcademicTermsEndDate:      p.AcademicTermsEndDate,
		AcademicTermsIsActive:     isActive,
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
}



// Mapper entity -> response
func FromModel(ent model.AcademicTermModel) AcademicTermResponseDTO {
	return AcademicTermResponseDTO{
		AcademicTermsID:           ent.AcademicTermsID,
		AcademicTermsMasjidID:     ent.AcademicTermsMasjidID,
		AcademicTermsAcademicYear: ent.AcademicTermsAcademicYear,
		AcademicTermsName:         ent.AcademicTermsName,
		AcademicTermsStartDate:    ent.AcademicTermsStartDate,
		AcademicTermsEndDate:      ent.AcademicTermsEndDate,
		AcademicTermsIsActive:     ent.AcademicTermsIsActive,
		AcademicTermsPeriod:       ent.AcademicTermsPeriod,
		AcademicTermsCreatedAt:    ent.AcademicTermsCreatedAt,
		AcademicTermsUpdatedAt:    ent.AcademicTermsUpdatedAt,
		AcademicTermsDeletedAt:    ent.AcademicTermsDeletedAt,
	}
}

func FromModels(list []model.AcademicTermModel) []AcademicTermResponseDTO {
	out := make([]AcademicTermResponseDTO, 0, len(list))
	for _, it := range list {
		out = append(out, FromModel(it))
	}
	return out
}

// dto/dto.go
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
}
