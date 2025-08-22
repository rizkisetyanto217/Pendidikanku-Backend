// file: internals/features/academics/terms/model/academic_term_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// AcademicTermModel merepresentasikan tabel `academic_terms`
type AcademicTermModel struct {
	AcademicTermsID            uuid.UUID  `json:"academic_terms_id"             gorm:"column:academic_terms_id;type:uuid;default:gen_random_uuid();primaryKey"`
	AcademicTermsMasjidID      uuid.UUID  `json:"academic_terms_masjid_id"      gorm:"column:academic_terms_masjid_id;type:uuid;not null"`
	AcademicTermsAcademicYear  string     `json:"academic_terms_academic_year"  gorm:"column:academic_terms_academic_year;type:text;not null"` // contoh: "2025/2026"
	AcademicTermsName          string     `json:"academic_terms_name"           gorm:"column:academic_terms_name;type:text;not null"`          // "Ganjil" | "Genap" | dst.

	AcademicTermsStartDate     time.Time  `json:"academic_terms_start_date"     gorm:"column:academic_terms_start_date;type:timestamp;not null"`
	AcademicTermsEndDate       time.Time  `json:"academic_terms_end_date"       gorm:"column:academic_terms_end_date;type:timestamp;not null"`
	AcademicTermsIsActive      bool       `json:"academic_terms_is_active"      gorm:"column:academic_terms_is_active;not null;default:true"`

	// generated daterange, read-only untuk GORM
	AcademicTermsPeriod        *string    `json:"academic_terms_period,omitempty" gorm:"column:academic_terms_period;type:daterange;->"`

	AcademicTermsCreatedAt     time.Time  `json:"academic_terms_created_at"     gorm:"column:academic_terms_created_at;type:timestamp;autoCreateTime"`
	AcademicTermsUpdatedAt     *time.Time `json:"academic_terms_updated_at,omitempty" gorm:"column:academic_terms_updated_at;type:timestamp"`
	AcademicTermsDeletedAt     *time.Time `json:"academic_terms_deleted_at,omitempty" gorm:"column:academic_terms_deleted_at;type:timestamp;index"`
}

// TableName memastikan GORM pakai nama tabel yang tepat.
func (AcademicTermModel) TableName() string {
	return "academic_terms"
}
