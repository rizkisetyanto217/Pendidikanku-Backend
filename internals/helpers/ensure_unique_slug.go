// helper/slug.go
package helper

import (
	"fmt"
	"regexp"

	"gorm.io/gorm"
)

// EnsureUniqueSlug mencari slug unik pada tabel tertentu.
// base → slug dasar (hasil GenerateSlug).
// table → nama tabel, misal "masjids".
// column → nama kolom slug, misal "masjid_slug".
func EnsureUniqueSlug(db *gorm.DB, base, table, column string) (string, error) {
	slug := base

	// fast path: cek slug exact ada/tidak
	var count int64
	if err := db.Table(table).
		Where(fmt.Sprintf("%s = ?", column), slug).
		Count(&count).Error; err != nil {
		return "", err
	}
	if count == 0 {
		return slug, nil
	}

	// cari suffix terbesar
	type row struct{ Slug string }
	var rows []row
	like := base + "%" // slug kita a-z0-9- aman dipakai LIKE
	if err := db.Table(table).
		Select(column + " as slug").
		Where(fmt.Sprintf("%s = ? OR %s LIKE ?", column, column), base, like).
		Find(&rows).Error; err != nil {
		return "", err
	}

	maxN := 1
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `-(\d+)$`)
	for _, r := range rows {
		m := re.FindStringSubmatch(r.Slug)
		if len(m) == 2 {
			var n int
			fmt.Sscanf(m[1], "%d", &n)
			if n > maxN {
				maxN = n
			}
		} else if r.Slug == base {
			if maxN < 1 {
				maxN = 1
			}
		}
	}

	return fmt.Sprintf("%s-%d", base, maxN+1), nil
}
