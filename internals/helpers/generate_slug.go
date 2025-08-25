package helper

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"gorm.io/gorm"
)

// ==========================================
// Opsional: batas default panjang slug
// ==========================================
const DefaultSlugMaxLen = 160

// SlugOptions menentukan cara cek keunikan slug di DB.
type SlugOptions struct {
	// Nama tabel di DB, contoh: "classes"
	Table string
	// Nama kolom untuk slug, contoh: "class_slug"
	SlugColumn string

	// Kolom soft-delete (NULL berarti belum terhapus).
	// Contoh: "class_deleted_at" atau "deleted_at".
	// Kosongkan jika tidak pakai soft-delete.
	SoftDeleteColumn string

	// Filter tambahan untuk memastikan unik dalam suatu tenant/scope.
	// Misal: map[string]any{"class_masjid_id": masjidID}
	Filters map[string]any

	// Panjang maksimal slug (termasuk suffix -2, -3, dst).
	// Jika 0, gunakan DefaultSlugMaxLen.
	MaxLen int

	// Base fallback jika input base kosong setelah dinormalisasi.
	// Contoh: "kelas", "post", dll. Wajib diisi agar ada fallback masuk akal.
	DefaultBase string
}

// GenerateSlug menormalkan string menjadi slug:
// - lower-case
// - spasi & non-alnum jadi "-"
// - collapse multiple "-" -> satu "-"
// - trim "-" di kedua ujung
func GenerateSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Ganti semua non-alnum (termasuk spasi) menjadi "-"
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else {
			// jadikan "-"
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := b.String()
	out = strings.Trim(out, "-")

	// Pastikan tidak ada "--" beruntun (guard tambahan)
	reDash := regexp.MustCompile(`-+`)
	out = reDash.ReplaceAllString(out, "-")
	return out
}

// cutToLen memotong string agar panjangnya <= n, lalu trim "-"
func cutToLen(s string, n int) string {
	if n <= 0 {
		return s
	}
	if len(s) <= n {
		return strings.Trim(s, "-")
	}
	return strings.Trim(s[:n], "-")
}

// isTaken mengecek apakah slug candidate sudah ada (case-insensitive),
// dengan memperhitungkan Filters dan SoftDeleteColumn.
func isTaken(db *gorm.DB, opts SlugOptions, candidate string) (bool, error) {
	if opts.Table == "" || opts.SlugColumn == "" {
		return false, errors.New("slug options: table/slug column required")
	}

	q := db.Table(opts.Table).
		Where(fmt.Sprintf("lower(%s) = lower(?)", opts.SlugColumn), candidate)

	// tambahkan filters (tenant/scope)
	for k, v := range opts.Filters {
		q = q.Where(fmt.Sprintf("%s = ?", k), v)
	}

	// soft-delete aware
	if opts.SoftDeleteColumn != "" {
		q = q.Where(fmt.Sprintf("%s IS NULL", opts.SoftDeleteColumn))
	}

	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// GenerateUniqueSlug membuat slug unik berbasis "base" (atau DefaultBase bila kosong),
// unik secara case-insensitive, hanya menghitung data yang belum soft-delete,
// dan unik dalam scope Filters.
//
// Algoritma:
// 1) Coba base dulu.
// 2) Jika bentrok, coba base-2, base-3, ... sampai ketemu atau mencapai batas iterasi.
func GenerateUniqueSlug(db *gorm.DB, opts SlugOptions, base string) (string, error) {
	maxLen := opts.MaxLen
	if maxLen <= 0 {
		maxLen = DefaultSlugMaxLen
	}

	base = strings.TrimSpace(base)
	if base == "" {
		base = opts.DefaultBase
	}
	base = GenerateSlug(base)
	if base == "" {
		// fallback terakhir yang sangat defensif
		if opts.DefaultBase != "" {
			base = GenerateSlug(opts.DefaultBase)
		}
		if base == "" {
			base = "x"
		}
	}

	// jaga panjang awal
	if len(base) > maxLen {
		base = cutToLen(base, maxLen)
		if base == "" {
			base = "x"
		}
	}

	// 1) coba base dulu
	taken, err := isTaken(db, opts, base)
	if err != nil {
		return "", err
	}
	if !taken {
		return base, nil
	}

	// 2) jika bentrok, tambahkan suffix -2, -3, ...
	for i := 2; i < 10000; i++ {
		suf := fmt.Sprintf("-%d", i)
		candidate := base

		if len(candidate)+len(suf) > maxLen {
			cut := maxLen - len(suf)
			if cut < 1 {
				cut = 1
			}
			candidate = cutToLen(candidate, cut)
			if candidate == "" {
				candidate = "x"
			}
		}
		candidate = candidate + suf

		taken, err = isTaken(db, opts, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", errors.New("failed to generate unique slug after many attempts")
}
