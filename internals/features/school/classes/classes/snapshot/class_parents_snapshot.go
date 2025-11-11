// file: internals/features/school/classes/classes/snapshot/class_parent_snapshot.go
package snapshot

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	classmodel "schoolku_backend/internals/features/school/classes/classes/model"
)

/*
Row kecil untuk loose-coupling dengan tabel class_parents
*/
type classParentSnapRow struct {
	Name  string  `gorm:"column:class_parent_name"`
	Code  *string `gorm:"column:class_parent_code"`
	Slug  *string `gorm:"column:class_parent_slug"`
	Level *int    `gorm:"column:class_parent_level"`
	// Kalau ada kolom URL di DB, buka comment ini dan mapping ke model juga:
	// URL   *string `gorm:"column:class_parent_url"`
}

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

func applyClassParentSnapshot(m *classmodel.ClassModel, pr classParentSnapRow) {
	// Wajib: name selalu ada (string)
	name := strings.TrimSpace(pr.Name)
	m.ClassParentNameSnapshot = &name

	// Optional: code & slug bisa nil/empty → trim ke nil
	m.ClassParentCodeSnapshot = trimPtr(pr.Code)
	m.ClassParentSlugSnapshot = trimPtr(pr.Slug)

	// Level: dari *int → *int16 di model
	if pr.Level != nil {
		lv := int16(*pr.Level)
		m.ClassParentLevelSnapshot = &lv
	} else {
		m.ClassParentLevelSnapshot = nil
	}

	// Jika kamu punya kolom URL di DB dan field di model:
	// m.ClassParentURLSnapshot = trimPtr(pr.URL)
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

// Fungsi publik: isi snapshot parent ke model
func HydrateClassParentSnapshot(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	m *classmodel.ClassModel,
) error {
	pr, err := fetchClassParentSnapRow(ctx, tx, schoolID, m.ClassClassParentID)
	if err != nil {
		return err
	}
	applyClassParentSnapshot(m, pr)
	return nil
}
