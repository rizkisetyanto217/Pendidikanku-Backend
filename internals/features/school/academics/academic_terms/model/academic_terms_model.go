// file: internals/features/school/academics/terms/model/academic_term_model.go
package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AcademicTermModel struct {
	// ============ PK & Tenant ============
	AcademicTermID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:academic_term_id" json:"academic_term_id"`
	AcademicTermMasjidID uuid.UUID `gorm:"type:uuid;not null;column:academic_term_masjid_id" json:"academic_term_masjid_id"`

	// ============ Identitas ============
	// Contoh academic_year: "2026/2027"
	AcademicTermAcademicYear string `gorm:"type:text;not null;column:academic_term_academic_year" json:"academic_term_academic_year"`
	// Contoh name: "Ganjil" | "Genap" | "Pendek"
	AcademicTermName string `gorm:"type:text;not null;column:academic_term_name" json:"academic_term_name"`

	AcademicTermCode *string `gorm:"type:varchar(24);column:academic_term_code" json:"academic_term_code,omitempty"`
	AcademicTermSlug *string `gorm:"type:varchar(50);column:academic_term_slug" json:"academic_term_slug,omitempty"`

	// ============ Periode & metadata ============
	AcademicTermStartDate   time.Time `gorm:"type:timestamptz;not null;column:academic_term_start_date" json:"academic_term_start_date"`
	AcademicTermEndDate     time.Time `gorm:"type:timestamptz;not null;column:academic_term_end_date" json:"academic_term_end_date"`
	AcademicTermIsActive    bool      `gorm:"not null;default:true;column:academic_term_is_active" json:"academic_term_is_active"`
	AcademicTermAngkatan    *int      `gorm:"column:academic_term_angkatan" json:"academic_term_angkatan,omitempty"`
	AcademicTermDescription *string   `gorm:"type:text;column:academic_term_description" json:"academic_term_description,omitempty"`

	// Generated column (half-open daterange [start, end)) — read-only
	AcademicTermPeriod *string `gorm:"->;type:daterange;column:academic_term_period" json:"academic_term_period,omitempty"`

	// ============ Audit / Soft delete ============
	AcademicTermCreatedAt time.Time      `gorm:"type:timestamptz;not null;autoCreateTime;column:academic_term_created_at" json:"academic_term_created_at"`
	AcademicTermUpdatedAt time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime;column:academic_term_updated_at" json:"academic_term_updated_at"`
	AcademicTermDeletedAt gorm.DeletedAt `gorm:"column:academic_term_deleted_at;index" json:"academic_term_deleted_at,omitempty"`
}

func (AcademicTermModel) TableName() string { return "academic_terms" }

// ============ Hooks: validasi & normalisasi ringan ============
func (m *AcademicTermModel) BeforeSave(tx *gorm.DB) error {
	// Mirror CHECK: end >= start
	if m.AcademicTermEndDate.Before(m.AcademicTermStartDate) {
		return errors.New("academic_term_end_date must be >= academic_term_start_date")
	}

	// Trim/normalize string ringan
	m.AcademicTermAcademicYear = strings.TrimSpace(m.AcademicTermAcademicYear)
	m.AcademicTermName = strings.TrimSpace(m.AcademicTermName)

	if m.AcademicTermCode != nil {
		c := strings.TrimSpace(*m.AcademicTermCode)
		if c == "" {
			m.AcademicTermCode = nil
		} else {
			m.AcademicTermCode = &c
		}
	}
	if m.AcademicTermSlug != nil {
		s := strings.TrimSpace(*m.AcademicTermSlug)
		if s == "" {
			m.AcademicTermSlug = nil
		} else {
			// biarkan casing ditentukan service layer; index unik pakai LOWER di SQL
			if len(s) > 50 {
				s = s[:50]
			}
			m.AcademicTermSlug = &s
		}
	}
	if m.AcademicTermDescription != nil {
		d := strings.TrimSpace(*m.AcademicTermDescription)
		if d == "" {
			m.AcademicTermDescription = nil
		} else {
			m.AcademicTermDescription = &d
		}
	}

	return nil
}
