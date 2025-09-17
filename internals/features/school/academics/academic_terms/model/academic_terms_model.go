// file: internals/features/academics/terms/model/academic_term_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AcademicTermModel struct {
	AcademicTermsID       uuid.UUID      `json:"academic_terms_id"        gorm:"column:academic_terms_id;type:uuid;default:gen_random_uuid();primaryKey"`
	AcademicTermsMasjidID uuid.UUID      `json:"academic_terms_masjid_id" gorm:"column:academic_terms_masjid_id;type:uuid;not null"`

	AcademicTermsAcademicYear string    `json:"academic_terms_academic_year" gorm:"column:academic_terms_academic_year;type:text;not null"`
	AcademicTermsName         string    `json:"academic_terms_name"          gorm:"column:academic_terms_name;type:text;not null"`

	AcademicTermsStartDate time.Time `json:"academic_terms_start_date" gorm:"column:academic_terms_start_date;type:timestamptz;not null"`
	AcademicTermsEndDate   time.Time `json:"academic_terms_end_date"   gorm:"column:academic_terms_end_date;type:timestamptz;not null"`
	AcademicTermsIsActive  bool      `json:"academic_terms_is_active"  gorm:"column:academic_terms_is_active;not null;default:true"`

	// Kolom yang sebelumnya belum ada di model:
	AcademicTermsCode string  `json:"academic_terms_code,omitempty" gorm:"column:academic_terms_code;type:varchar(24)"`
	AcademicTermsSlug string  `json:"academic_terms_slug,omitempty" gorm:"column:academic_terms_slug;type:varchar(50)"`
	AcademicTermsDescription string `json:"academic_terms_description,omitempty" gorm:"column:academic_terms_description;type:text"`

	// Nullable
	AcademicTermsAngkatan *int `json:"academic_terms_angkatan,omitempty" gorm:"column:academic_terms_angkatan;type:int"`

	// Generated (read-only). Biarkan string jika tidak butuh tipe range native.
	AcademicTermsPeriod *string `json:"academic_terms_period,omitempty" gorm:"column:academic_terms_period;type:daterange;->"`

	AcademicTermsCreatedAt time.Time      `json:"academic_terms_created_at" gorm:"column:academic_terms_created_at;type:timestamptz;not null;autoCreateTime"`
	AcademicTermsUpdatedAt time.Time      `json:"academic_terms_updated_at" gorm:"column:academic_terms_updated_at;type:timestamptz;not null;autoUpdateTime"`
	AcademicTermsDeletedAt gorm.DeletedAt `json:"academic_terms_deleted_at,omitempty" gorm:"column:academic_terms_deleted_at;type:timestamptz;index"`
}

func (AcademicTermModel) TableName() string { return "academic_terms" }
