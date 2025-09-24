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
	AcademicTermAcademicYear string    `json:"academic_term_academic_year" validate:"required,min=4"`
	AcademicTermName         string    `json:"academic_term_name"           validate:"required,oneof=Ganjil Genap Pendek Khusus"`
	AcademicTermStartDate    time.Time `json:"academic_term_start_date"     validate:"required"`
	AcademicTermEndDate      time.Time `json:"academic_term_end_date"       validate:"required,gtefield=AcademicTermStartDate"`
	AcademicTermIsActive     *bool     `json:"academic_term_is_active,omitempty"`
	AcademicTermAngkatan     *int      `json:"academic_term_angkatan,omitempty" validate:"omitempty,gt=0"`

	// Kolom opsional (pointer agar selaras dgn model)
	AcademicTermCode        *string `json:"academic_term_code,omitempty"        validate:"omitempty,min=1,max=24"`
	AcademicTermSlug        *string `json:"academic_term_slug,omitempty"        validate:"omitempty,min=3,max=50"`
	AcademicTermDescription *string `json:"academic_term_description,omitempty" validate:"omitempty"`
}

type AcademicTermUpdateDTO struct {
	AcademicTermAcademicYear *string    `json:"academic_term_academic_year,omitempty" validate:"omitempty,min=4"`
	AcademicTermName         *string    `json:"academic_term_name,omitempty"`
	AcademicTermStartDate    *time.Time `json:"academic_term_start_date,omitempty"`
	AcademicTermEndDate      *time.Time `json:"academic_term_end_date,omitempty"`
	AcademicTermIsActive     *bool      `json:"academic_term_is_active,omitempty"`
	AcademicTermAngkatan     *int       `json:"academic_term_angkatan,omitempty" validate:"omitempty,gt=0"`

	AcademicTermCode        *string `json:"academic_term_code,omitempty"        validate:"omitempty,min=1,max=24"`
	AcademicTermSlug        *string `json:"academic_term_slug,omitempty"        validate:"omitempty,min=3,max=50"`
	AcademicTermDescription *string `json:"academic_term_description,omitempty" validate:"omitempty"`
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
	AcademicTermID           uuid.UUID `json:"academic_term_id"`
	AcademicTermMasjidID     uuid.UUID `json:"academic_term_masjid_id"`
	AcademicTermAcademicYear string    `json:"academic_term_academic_year"`
	AcademicTermName         string    `json:"academic_term_name"`
	AcademicTermStartDate    time.Time `json:"academic_term_start_date"`
	AcademicTermEndDate      time.Time `json:"academic_term_end_date"`
	AcademicTermIsActive     bool      `json:"academic_term_is_active"`

	AcademicTermAngkatan *int `json:"academic_term_angkatan,omitempty"`

	AcademicTermCode        *string `json:"academic_term_code,omitempty"`
	AcademicTermSlug        *string `json:"academic_term_slug,omitempty"`
	AcademicTermDescription *string `json:"academic_term_description,omitempty"`

	// Read-only: diisi DB (generated)
	AcademicTermPeriod *string `json:"academic_term_period,omitempty"`

	AcademicTermCreatedAt time.Time  `json:"academic_term_created_at"`
	AcademicTermUpdatedAt time.Time  `json:"academic_term_updated_at"`
	AcademicTermDeletedAt *time.Time `json:"academic_term_deleted_at,omitempty"`
}

/* ===================== HELPERS ===================== */

func (p *AcademicTermCreateDTO) Normalize() {
	p.AcademicTermAcademicYear = strings.TrimSpace(p.AcademicTermAcademicYear)
	p.AcademicTermName = strings.TrimSpace(p.AcademicTermName)

	// slug: trim + lower
	if p.AcademicTermSlug != nil {
		slug := strings.ToLower(strings.TrimSpace(*p.AcademicTermSlug))
		if slug == "" {
			p.AcademicTermSlug = nil
		} else {
			p.AcademicTermSlug = &slug
		}
	}
	// code: trim
	if p.AcademicTermCode != nil {
		code := strings.TrimSpace(*p.AcademicTermCode)
		if code == "" {
			p.AcademicTermCode = nil
		} else {
			p.AcademicTermCode = &code
		}
	}
	// description: trim (kalau kosong jadikan nil)
	if p.AcademicTermDescription != nil {
		d := strings.TrimSpace(*p.AcademicTermDescription)
		if d == "" {
			p.AcademicTermDescription = nil
		} else {
			p.AcademicTermDescription = &d
		}
	}
}

func (p *AcademicTermCreateDTO) WantsActive() bool {
	return p.AcademicTermIsActive == nil || *p.AcademicTermIsActive
}

func (p *AcademicTermCreateDTO) ToModel(masjidID uuid.UUID) model.AcademicTermModel {
	isActive := true
	if p.AcademicTermIsActive != nil {
		isActive = *p.AcademicTermIsActive
	}
	return model.AcademicTermModel{
		AcademicTermMasjidID:     masjidID,
		AcademicTermAcademicYear: p.AcademicTermAcademicYear,
		AcademicTermName:         p.AcademicTermName,
		AcademicTermStartDate:    p.AcademicTermStartDate,
		AcademicTermEndDate:      p.AcademicTermEndDate,
		AcademicTermIsActive:     isActive,
		AcademicTermAngkatan:     p.AcademicTermAngkatan,
		AcademicTermCode:         p.AcademicTermCode,
		AcademicTermSlug:         p.AcademicTermSlug,
		AcademicTermDescription:  p.AcademicTermDescription,
	}
}

func (u *AcademicTermUpdateDTO) ApplyUpdates(ent *model.AcademicTermModel) {
	if u.AcademicTermAcademicYear != nil {
		ent.AcademicTermAcademicYear = strings.TrimSpace(*u.AcademicTermAcademicYear)
	}
	if u.AcademicTermName != nil {
		ent.AcademicTermName = strings.TrimSpace(*u.AcademicTermName)
	}
	if u.AcademicTermStartDate != nil {
		ent.AcademicTermStartDate = *u.AcademicTermStartDate
	}
	if u.AcademicTermEndDate != nil {
		ent.AcademicTermEndDate = *u.AcademicTermEndDate
	}
	if u.AcademicTermIsActive != nil {
		ent.AcademicTermIsActive = *u.AcademicTermIsActive
	}
	if u.AcademicTermAngkatan != nil {
		ent.AcademicTermAngkatan = u.AcademicTermAngkatan
	}
	// Kolom opsional (pointer): kosong => nil
	if u.AcademicTermCode != nil {
		code := strings.TrimSpace(*u.AcademicTermCode)
		if code == "" {
			ent.AcademicTermCode = nil
		} else {
			ent.AcademicTermCode = &code
		}
	}
	if u.AcademicTermSlug != nil {
		slug := strings.ToLower(strings.TrimSpace(*u.AcademicTermSlug))
		if slug == "" {
			ent.AcademicTermSlug = nil
		} else {
			ent.AcademicTermSlug = &slug
		}
	}
	if u.AcademicTermDescription != nil {
		desc := strings.TrimSpace(*u.AcademicTermDescription)
		if desc == "" {
			ent.AcademicTermDescription = nil
		} else {
			ent.AcademicTermDescription = &desc
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
	if ent.AcademicTermDeletedAt.Valid {
		t := ent.AcademicTermDeletedAt.Time
		deletedAt = &t
	}

	// Langsung pakai pointer dari model untuk code/slug/description/period
	return AcademicTermResponseDTO{
		AcademicTermID:           ent.AcademicTermID,
		AcademicTermMasjidID:     ent.AcademicTermMasjidID,
		AcademicTermAcademicYear: ent.AcademicTermAcademicYear,
		AcademicTermName:         ent.AcademicTermName,
		AcademicTermStartDate:    ent.AcademicTermStartDate,
		AcademicTermEndDate:      ent.AcademicTermEndDate,
		AcademicTermIsActive:     ent.AcademicTermIsActive,
		AcademicTermAngkatan:     ent.AcademicTermAngkatan,

		AcademicTermCode:        ent.AcademicTermCode,
		AcademicTermSlug:        ent.AcademicTermSlug,
		AcademicTermDescription: ent.AcademicTermDescription,

		AcademicTermPeriod:    ent.AcademicTermPeriod,
		AcademicTermCreatedAt: ent.AcademicTermCreatedAt,
		AcademicTermUpdatedAt: ent.AcademicTermUpdatedAt,
		AcademicTermDeletedAt: deletedAt,
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

type OpeningWithClass struct {
	// opening
	ClassTermOpeningsID                    uuid.UUID  `json:"class_term_openings_id"                       gorm:"column:class_term_openings_id"`
	ClassTermOpeningsMasjidID              uuid.UUID  `json:"class_term_openings_masjid_id"                gorm:"column:class_term_openings_masjid_id"`
	ClassTermOpeningsClassID               uuid.UUID  `json:"class_term_openings_class_id"                 gorm:"column:class_term_openings_class_id"`
	ClassTermOpeningsTermID                uuid.UUID  `json:"class_term_openings_term_id"                  gorm:"column:class_term_openings_term_id"`
	ClassTermOpeningsIsOpen                bool       `json:"class_term_openings_is_open"                  gorm:"column:class_term_openings_is_open"`
	ClassTermOpeningsRegistrationOpensAt   *time.Time `json:"class_term_openings_registration_opens_at"    gorm:"column:class_term_openings_registration_opens_at"`
	ClassTermOpeningsRegistrationClosesAt  *time.Time `json:"class_term_openings_registration_closes_at"   gorm:"column:class_term_openings_registration_closes_at"`
	ClassTermOpeningsQuotaTotal            *int       `json:"class_term_openings_quota_total"              gorm:"column:class_term_openings_quota_total"`
	ClassTermOpeningsQuotaTaken            int        `json:"class_term_openings_quota_taken"              gorm:"column:class_term_openings_quota_taken"`
	ClassTermOpeningsFeeOverrideMonthlyIDR *int       `json:"class_term_openings_fee_override_monthly_idr" gorm:"column:class_term_openings_fee_override_monthly_idr"`
	ClassTermOpeningsNotes                 *string    `json:"class_term_openings_notes"                    gorm:"column:class_term_openings_notes"`
	ClassTermOpeningsCreatedAt             time.Time  `json:"class_term_openings_created_at"               gorm:"column:class_term_openings_created_at"`
	ClassTermOpeningsUpdatedAt             *time.Time `json:"class_term_openings_updated_at"               gorm:"column:class_term_openings_updated_at"`
	ClassTermOpeningsDeletedAt             *time.Time `json:"class_term_openings_deleted_at"               gorm:"column:class_term_openings_deleted_at"`

	// class (subset)
	Class struct {
		ClassID          uuid.UUID  `json:"class_id"          gorm:"column:class_id"`
		ClassMasjidID    *uuid.UUID `json:"class_masjid_id"   gorm:"column:class_masjid_id"`
		ClassName        string     `json:"class_name"        gorm:"column:class_name"`
		ClassSlug        string     `json:"class_slug"        gorm:"column:class_slug"`
		ClassDescription *string    `json:"class_description" gorm:"column:class_description"`
		ClassLevel       *string    `json:"class_level"       gorm:"column:class_level"`
		ClassImageURL    *string    `json:"class_image_url"   gorm:"column:class_image_url"`
		ClassIsActive    bool       `json:"class_is_active"   gorm:"column:class_is_active"`
	} `json:"class"`
}
