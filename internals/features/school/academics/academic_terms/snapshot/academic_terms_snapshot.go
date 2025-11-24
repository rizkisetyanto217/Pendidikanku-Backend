// file: internals/features/school/academics/academic_terms/snapshot/academic_term_snapshot_for_class.go
package snapshot

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	classmodel "madinahsalam_backend/internals/features/school/classes/classes/model"
)

/*
Row kecil → longgar terhadap struktur tabel academic_terms
*/
type academicTermSnapRow struct {
	Slug         *string `gorm:"column:academic_term_slug"`
	Name         *string `gorm:"column:academic_term_name"`
	AcademicYear *string `gorm:"column:academic_term_academic_year"`
	Angkatan     *int    `gorm:"column:academic_term_angkatan"`
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func fetchAcademicTermSnapRow(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	termID uuid.UUID,
) (academicTermSnapRow, error) {
	var tr academicTermSnapRow
	if err := tx.WithContext(ctx).
		Table("academic_terms").
		Select("academic_term_slug, academic_term_name, academic_term_academic_year, academic_term_angkatan").
		Where("academic_term_id = ? AND academic_term_school_id = ? AND academic_term_deleted_at IS NULL",
			termID, schoolID).
		Take(&tr).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tr, fiber.NewError(fiber.StatusBadRequest, "Academic term tidak ditemukan di school ini")
		}
		return tr, err
	}
	return tr, nil
}

func applyAcademicTermSnapshotToClass(m *classmodel.ClassModel, tr academicTermSnapRow) {
	m.ClassAcademicTermSlugSnapshot = trimPtr(tr.Slug)
	m.ClassAcademicTermNameSnapshot = trimPtr(tr.Name)
	m.ClassAcademicTermAcademicYearSnapshot = trimPtr(tr.AcademicYear)

	// Di ClassModel: angkatan bertipe *string → convert dari *int bila ada
	if tr.Angkatan != nil {
		s := strconv.Itoa(*tr.Angkatan)
		m.ClassAcademicTermAngkatanSnapshot = &s
	} else {
		m.ClassAcademicTermAngkatanSnapshot = nil
	}
}

func clearAcademicTermSnapshotOnClass(m *classmodel.ClassModel) {
	m.ClassAcademicTermSlugSnapshot = nil
	m.ClassAcademicTermNameSnapshot = nil
	m.ClassAcademicTermAcademicYearSnapshot = nil
	m.ClassAcademicTermAngkatanSnapshot = nil
}

// Fungsi publik: isi snapshot term ke ClassModel
func HydrateAcademicTermSnapshot(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	m *classmodel.ClassModel,
) error {
	// Perhatikan: field FK di model adalah ClassAcademicTermID (*uuid.UUID)
	if m.ClassAcademicTermID == nil || *m.ClassAcademicTermID == uuid.Nil {
		clearAcademicTermSnapshotOnClass(m)
		return nil
	}

	tr, err := fetchAcademicTermSnapRow(ctx, tx, schoolID, *m.ClassAcademicTermID)
	if err != nil {
		return err
	}
	applyAcademicTermSnapshotToClass(m, tr)
	return nil
}
