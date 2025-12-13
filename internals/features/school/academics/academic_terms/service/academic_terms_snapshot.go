// file: internals/features/school/academics/academic_terms/service/term_cache.go
package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	termModel "madinahsalam_backend/internals/features/school/academics/academic_terms/model"
	classmodel "madinahsalam_backend/internals/features/school/classes/classes/model"
)

/* =========================================================
   Helper: display name "name + year" (anti dobel)
========================================================= */

func buildTermDisplayName(name string, ay string) string {
	n := strings.TrimSpace(name)
	y := strings.TrimSpace(ay)
	if y == "" {
		return n
	}

	ln := strings.ToLower(n)
	ly := strings.ToLower(y)

	if strings.HasSuffix(ln, " "+ly) || ln == ly {
		return n
	}
	return strings.TrimSpace(n + " " + y)
}

/* =========================================================
   A) HYDRATE cache ke struct ClassModel (dipakai Create/Patch Class)
========================================================= */

func HydrateAcademicTermCache(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	m *classmodel.ClassModel,
) error {
	if tx == nil || m == nil {
		return nil
	}

	// kalau class gak punya term, bersihin cache
	if m.ClassAcademicTermID == nil || *m.ClassAcademicTermID == uuid.Nil {
		m.ClassAcademicTermAcademicYearCache = nil
		m.ClassAcademicTermNameCache = nil
		m.ClassAcademicTermSlugCache = nil
		m.ClassAcademicTermAngkatanCache = nil
		return nil
	}

	type row struct {
		Year     *string `gorm:"column:academic_term_academic_year"`
		Name     string  `gorm:"column:academic_term_name"`
		Slug     *string `gorm:"column:academic_term_slug"`
		Angkatan *int16  `gorm:"column:academic_term_angkatan"`
	}
	var r row

	if err := tx.WithContext(ctx).
		Table("academic_terms").
		Select("academic_term_academic_year, academic_term_name, academic_term_slug, academic_term_angkatan").
		Where("academic_term_id = ? AND academic_term_school_id = ? AND academic_term_deleted_at IS NULL",
			*m.ClassAcademicTermID, schoolID).
		Take(&r).Error; err != nil {
		return err
	}

	// year cache
	if r.Year != nil {
		y := strings.TrimSpace(*r.Year)
		if y == "" {
			m.ClassAcademicTermAcademicYearCache = nil
		} else {
			m.ClassAcademicTermAcademicYearCache = &y
		}
	} else {
		m.ClassAcademicTermAcademicYearCache = nil
	}

	// name cache: diset display "name + year" biar konsisten
	ay := ""
	if r.Year != nil {
		ay = strings.TrimSpace(*r.Year)
	}
	display := buildTermDisplayName(r.Name, ay)
	if strings.TrimSpace(display) == "" {
		m.ClassAcademicTermNameCache = nil
	} else {
		m.ClassAcademicTermNameCache = &display
	}

	// slug cache
	m.ClassAcademicTermSlugCache = r.Slug

	// angkatan cache: classes pakai *string
	if r.Angkatan != nil {
		tmp := strconv.Itoa(int(*r.Angkatan))
		m.ClassAcademicTermAngkatanCache = &tmp
	} else {
		m.ClassAcademicTermAngkatanCache = nil
	}

	return nil
}

func (s *AcademicTermCacheService) RefreshClassSectionsForTerm(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	term *termModel.AcademicTermModel,
) error {
	if tx == nil || term == nil {
		return nil
	}

	termDisplayName := buildTermDisplayName(
		term.AcademicTermName,
		term.AcademicTermAcademicYear,
	)

	var angkatanInt *int
	if term.AcademicTermAngkatan != nil {
		tmp := int(*term.AcademicTermAngkatan)
		angkatanInt = &tmp
	}

	now := time.Now()

	sql := `
UPDATE class_sections cs
SET
  class_section_academic_term_academic_year_cache = ?,
  class_section_academic_term_name_cache          = ?,
  class_section_academic_term_slug_cache          = ?,
  class_section_academic_term_angkatan_cache      = ?,
  class_section_updated_at                        = ?
WHERE
  cs.class_section_school_id = ?
  AND cs.class_section_academic_term_id = ?
  AND cs.class_section_deleted_at IS NULL;
`

	return tx.WithContext(ctx).Exec(
		sql,
		term.AcademicTermAcademicYear,
		termDisplayName,
		term.AcademicTermSlug,
		angkatanInt,
		now,
		schoolID,
		term.AcademicTermID,
	).Error
}

func (s *AcademicTermCacheService) RefreshCSSTForTerm(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	term *termModel.AcademicTermModel,
) error {
	if tx == nil || term == nil {
		return nil
	}

	termDisplayName := buildTermDisplayName(
		term.AcademicTermName,
		term.AcademicTermAcademicYear,
	)

	// CSST cache angkatan bertipe *int (sesuai model kamu)
	var angkatanInt *int
	if term.AcademicTermAngkatan != nil {
		tmp := int(*term.AcademicTermAngkatan)
		angkatanInt = &tmp
	}

	now := time.Now()

	sql := `
UPDATE class_section_subject_teachers csst
SET
  csst_academic_year_cache          = ?,
  csst_academic_term_name_cache     = ?,
  csst_academic_term_slug_cache     = ?,
  csst_academic_term_angkatan_cache = ?,
  csst_updated_at                  = ?
WHERE
  csst.csst_school_id = ?
  AND csst.csst_academic_term_id = ?
  AND csst.csst_deleted_at IS NULL;
`

	return tx.WithContext(ctx).Exec(
		sql,
		term.AcademicTermAcademicYear,
		termDisplayName,
		term.AcademicTermSlug,
		angkatanInt,
		now,
		schoolID,
		term.AcademicTermID,
	).Error
}

/* =========================================================
   B) MASS REFRESH cache di tabel classes (dipakai Patch AcademicTerm)
========================================================= */

type AcademicTermCacheService struct{}

func NewAcademicTermCacheService() *AcademicTermCacheService {
	return &AcademicTermCacheService{}
}

func (s *AcademicTermCacheService) RefreshClassesForTerm(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	term *termModel.AcademicTermModel,
) error {
	if tx == nil || term == nil {
		return nil
	}

	termDisplayName := buildTermDisplayName(term.AcademicTermName, term.AcademicTermAcademicYear)

	// classes cache angkatan bertipe *string
	var angkatanStr *string
	if term.AcademicTermAngkatan != nil {
		tmp := strconv.Itoa(int(*term.AcademicTermAngkatan))
		angkatanStr = &tmp
	}

	now := time.Now()

	sql := `
UPDATE classes c
SET
  class_academic_term_academic_year_cache = ?,
  class_academic_term_name_cache          = ?,
  class_academic_term_slug_cache          = ?,
  class_academic_term_angkatan_cache      = ?,

  class_name = CASE
    WHEN COALESCE(TRIM(cp.class_parent_name), '') = '' THEN NULL
    WHEN COALESCE(TRIM(?), '') = '' THEN cp.class_parent_name
    ELSE cp.class_parent_name || ' — ' || ?
  END,

  class_updated_at = ?
FROM class_parents cp
WHERE
  c.class_school_id = ?
  AND c.class_academic_term_id = ?
  AND c.class_deleted_at IS NULL
  AND cp.class_parent_id = c.class_class_parent_id
  AND cp.class_parent_deleted_at IS NULL
  AND cp.class_parent_school_id = ?;
`

	return tx.WithContext(ctx).Exec(
		sql,
		term.AcademicTermAcademicYear,
		termDisplayName,
		term.AcademicTermSlug,
		angkatanStr,

		termDisplayName,
		termDisplayName,

		now,
		schoolID,
		term.AcademicTermID,
		schoolID,
	).Error
}
