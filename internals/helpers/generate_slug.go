package helper

import (
	"regexp"
	"strings"
)

// Precompile regex biar nggak bikin objek baru tiap panggil
var (
	reWhitespace    = regexp.MustCompile(`\s+`)
	reNonAlnumDash  = regexp.MustCompile(`[^a-z0-9-]+`)
	reMultiDash     = regexp.MustCompile(`-+`)
)

// GenerateSlug mengubah string bebas menjadi slug yang lolos regex ^[a-z0-9-]+$
// dan idempotent (dipanggil berulang hasilnya tetap).
func GenerateSlug(s string) string {
	// 1) trim + lower
	s = strings.ToLower(strings.TrimSpace(s))

	// 2) normalisasi separator umum ke '-'
	//    termasuk underscore & en/em dash
	s = strings.NewReplacer(
		"_", "-",
		"–", "-",
		"—", "-",
	).Replace(s)

	// 3) semua whitespace (spasi, tab, newline) → '-'
	s = reWhitespace.ReplaceAllString(s, "-")

	// 4) buang karakter di luar [a-z0-9-] (diganti '-')
	s = reNonAlnumDash.ReplaceAllString(s, "-")

	// 5) rapikan: '---' → '-'
	s = reMultiDash.ReplaceAllString(s, "-")

	// 6) trim '-' di ujung
	s = strings.Trim(s, "-")

	// 7) (opsional) batasi panjang max (ikuti validator kamu, contoh 120)
	const maxLen = 120
	if len(s) > maxLen {
		s = s[:maxLen]
		s = strings.Trim(s, "-") // jaga² kalau terpotong di '-'
	}

	return s
}
