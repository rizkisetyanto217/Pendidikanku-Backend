package helper

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
)

var (
	reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
	reHyphen   = regexp.MustCompile(`-+`)
)

// Slugify mengubah teks bebas jadi slug [a-z0-9-], hilangkan diakritik,
// kompres "-", trim ujung, enforce maxLen (default 100 jika <=0), fallback "item".
func Slugify(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 100
	}
	s = strings.ToLower(strings.TrimSpace(s))

	// Strip diakritik (é → e, dll)
	var buf []rune
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) { // mark nonspacing
			continue
		}
		buf = append(buf, r)
	}
	s = string(buf)

	// Keep [a-z0-9-]
	s = reNonAlnum.ReplaceAllString(s, "-")
	s = reHyphen.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	if s == "" {
		s = "item"
	}
	// Hard-limit panjang
	if utf8.RuneCountInString(s) > maxLen {
		rs := []rune(s)
		s = string(rs[:maxLen])
		s = strings.Trim(s, "-")
	}
	if s == "" {
		s = "item"
	}
	return s
}

// EnsureUniqueSlugCI memastikan slug unik (case-insensitive) di satu tabel/kolom.
// scopeFn boleh nil; kalau tidak nil, dipakai untuk menambah WHERE (mis. tenant).
// maxLen untuk menjaga total panjang saat menambah suffix "-2", "-3", dst.
// Contoh scopeFn: func(q *gorm.DB)*gorm.DB { return q.Where("masjid_id = ?", mid) }
func EnsureUniqueSlugCI(
	ctx context.Context,
	db *gorm.DB,
	table string,
	column string,
	baseSlug string,
	scopeFn func(*gorm.DB) *gorm.DB,
	maxLen int,
) (string, error) {
	if maxLen <= 0 {
		maxLen = 100
	}
	slug := baseSlug
	lower := strings.ToLower(slug)

	// Coba beberapa kali dengan suffix -2, -3, ... lalu fallback random pendek.
	for i := 0; i < 25; i++ {
		q := db.WithContext(ctx).Table(table)
		if scopeFn != nil {
			q = scopeFn(q)
		}

		var count int64
		// CASE-INSENSITIVE check
		if err := q.Where(fmt.Sprintf("LOWER(%s) = ?", column), lower).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return slug, nil
		}

		suffix := fmt.Sprintf("-%d", i+2)
		slug = trimForSuffix(baseSlug, suffix, maxLen) + suffix
		lower = strings.ToLower(slug)
	}

	// Fallback: random pendek berbasis waktu
	r := fmt.Sprintf("-%x", (time.Now().UnixNano() & 0xffff))
	slug = trimForSuffix(baseSlug, r, maxLen) + r
	return slug, nil
}

// trimForSuffix memotong base agar base+suffix <= maxLen, lalu trim '-' di ujung.
func trimForSuffix(base, suffix string, maxLen int) string {
	if maxLen <= 0 {
		return base
	}
	need := len(suffix)
	if need >= maxLen {
		// kasus ekstrim; kembalikan minimal satu huruf 'x'
		return "x"
	}
	// Pastikan base tak melebihi batas.
	rs := []rune(base)
	keep := maxLen - need
	if keep < 1 {
		keep = 1
	}
	if len(rs) > keep {
		rs = rs[:keep]
	}
	out := strings.Trim(string(rs), "-")
	if out == "" {
		out = "x"
	}
	return out
}

// SuggestSlugFromName util kecil: slugify nama dengan batas default 100.
func SuggestSlugFromName(name string) string {
	return Slugify(name, 100)
}
