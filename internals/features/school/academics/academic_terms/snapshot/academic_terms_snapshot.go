package controller

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	classmodel "masjidku_backend/internals/features/school/classes/classes/model"
)

// Struktur row kecil agar loose-coupling ke tabel academic_terms
type academicTermSnapRow struct {
	Year     string  `gorm:"column:academic_term_academic_year"`
	Name     string  `gorm:"column:academic_term_name"`
	Slug     *string `gorm:"column:academic_term_slug"`
	Angkatan *int    `gorm:"column:academic_term_angkatan"`
}

// Ambil data snapshot term dari DB (guard tenant & soft-delete)
func fetchAcademicTermSnapRow(
	ctx context.Context,
	tx *gorm.DB,
	masjidID uuid.UUID,
	termID uuid.UUID,
) (academicTermSnapRow, error) {
	var tr academicTermSnapRow
	if err := tx.WithContext(ctx).
		Table("academic_terms").
		Select("academic_term_academic_year, academic_term_name, academic_term_slug, academic_term_angkatan").
		Where(
			"academic_term_id = ? AND academic_term_masjid_id = ? AND academic_term_deleted_at IS NULL",
			termID, masjidID,
		).
		Take(&tr).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tr, fiber.NewError(fiber.StatusBadRequest, "Academic term tidak ditemukan di masjid ini")
		}
		return tr, err
	}
	return tr, nil
}

// Apply nilai snapshot term ke model ClassModel
func applyAcademicTermSnapshot(m *classmodel.ClassModel, tr academicTermSnapRow) {
	m.ClassTermAcademicYearSnapshot = &tr.Year
	m.ClassTermNameSnapshot = &tr.Name
	m.ClassTermSlugSnapshot = tr.Slug
	if tr.Angkatan != nil {
		s := strconv.Itoa(*tr.Angkatan)
		m.ClassTermAngkatanSnapshot = &s
	} else {
		m.ClassTermAngkatanSnapshot = nil
	}
}

// Fungsi publik: isi snapshot term ke model (aman utk term nil)
func HydrateAcademicTermSnapshot(
	ctx context.Context,
	tx *gorm.DB,
	masjidID uuid.UUID,
	m *classmodel.ClassModel,
) error {
	if m.ClassTermID == nil {
		m.ClassTermAcademicYearSnapshot = nil
		m.ClassTermNameSnapshot = nil
		m.ClassTermSlugSnapshot = nil
		m.ClassTermAngkatanSnapshot = nil
		return nil
	}
	tr, err := fetchAcademicTermSnapRow(ctx, tx, masjidID, *m.ClassTermID)
	if err != nil {
		return err
	}
	applyAcademicTermSnapshot(m, tr)
	return nil
}
