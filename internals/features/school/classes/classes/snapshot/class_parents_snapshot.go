package controller

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	classmodel "schoolku_backend/internals/features/school/classes/classes/model"
)

// Struktur row kecil agar loose-coupling ke tabel class_parents
type classParentSnapRow struct {
	Name  string  `gorm:"column:class_parent_name"`
	Code  *string `gorm:"column:class_parent_code"`
	Slug  *string `gorm:"column:class_parent_slug"`
	Level *int    `gorm:"column:class_parent_level"`
	// Jika kamu punya kolom URL:
	// URL *string `gorm:"column:class_parent_url"`
}

// Ambil data snapshot parent dari DB (guard tenant & soft-delete)
func fetchClassParentSnapRow(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	parentID uuid.UUID,
) (classParentSnapRow, error) {
	var pr classParentSnapRow
	if err := tx.WithContext(ctx).
		Table("class_parents").
		Select("class_parent_name, class_parent_code, class_parent_slug, class_parent_level").
		Where(
			"class_parent_id = ? AND class_parent_school_id = ? AND class_parent_deleted_at IS NULL",
			parentID, schoolID,
		).
		Take(&pr).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pr, fiber.NewError(fiber.StatusBadRequest, "Class parent tidak ditemukan di school ini")
		}
		return pr, err
	}
	return pr, nil
}

// Apply nilai snapshot parent ke model ClassModel
func applyClassParentSnapshot(m *classmodel.ClassModel, pr classParentSnapRow) {
	m.ClassParentNameSnapshot = &pr.Name
	m.ClassParentCodeSnapshot = pr.Code
	m.ClassParentSlugSnapshot = pr.Slug
	if pr.Level != nil {
		lv := int16(*pr.Level)
		m.ClassParentLevelSnapshot = &lv
	} else {
		m.ClassParentLevelSnapshot = nil
	}
	// m.ClassParentURLSnapshot = pr.URL // kalau kolom URL tersedia
}

// Fungsi publik: isi snapshot parent ke model
func HydrateClassParentSnapshot(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	m *classmodel.ClassModel,
) error {
	pr, err := fetchClassParentSnapRow(ctx, tx, schoolID, m.ClassParentID)
	if err != nil {
		return err
	}
	applyClassParentSnapshot(m, pr)
	return nil
}
