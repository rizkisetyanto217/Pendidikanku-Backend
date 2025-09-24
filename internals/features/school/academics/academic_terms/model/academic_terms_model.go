// file: internals/features/school/academics/terms/model/academic_term_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AcademicTermModel struct {
	// PK
	AcademicTermID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:academic_term_id" json:"academic_term_id"`

	// Tenant
	AcademicTermMasjidID uuid.UUID `gorm:"type:uuid;not null;column:academic_term_masjid_id" json:"academic_term_masjid_id"`

	// Identitas
	AcademicTermAcademicYear string  `gorm:"type:text;not null;column:academic_term_academic_year" json:"academic_term_academic_year"` // contoh "2026/2027"
	AcademicTermName         string  `gorm:"type:text;not null;column:academic_term_name" json:"academic_term_name"`                   // "Ganjil" | "Genap" | dst.
	AcademicTermCode         *string `gorm:"type:varchar(24);column:academic_term_code" json:"academic_term_code,omitempty"`
	AcademicTermSlug         *string `gorm:"type:varchar(50);column:academic_term_slug" json:"academic_term_slug,omitempty"`

	// Periode & metadata
	AcademicTermStartDate   time.Time `gorm:"type:timestamptz;not null;column:academic_term_start_date" json:"academic_term_start_date"`
	AcademicTermEndDate     time.Time `gorm:"type:timestamptz;not null;column:academic_term_end_date" json:"academic_term_end_date"`
	AcademicTermIsActive    bool      `gorm:"not null;default:true;column:academic_term_is_active" json:"academic_term_is_active"`
	AcademicTermAngkatan    *int      `gorm:"column:academic_term_angkatan" json:"academic_term_angkatan,omitempty"`
	AcademicTermDescription *string   `gorm:"type:text;column:academic_term_description" json:"academic_term_description,omitempty"`

	// Generated column (read-only) : daterange [start,end)
	AcademicTermPeriod *string `gorm:"->;type:daterange;column:academic_term_period" json:"academic_term_period,omitempty"`

	// Timestamps (dikelola aplikasi)
	AcademicTermCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:academic_term_created_at" json:"academic_term_created_at"`
	AcademicTermUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:academic_term_updated_at" json:"academic_term_updated_at"`
	AcademicTermDeletedAt gorm.DeletedAt `gorm:"column:academic_term_deleted_at;index" json:"academic_term_deleted_at,omitempty"`
}

func (AcademicTermModel) TableName() string { return "academic_terms" }
