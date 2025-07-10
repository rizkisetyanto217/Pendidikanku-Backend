package helpers

import "strings"

// ✅ Hilangkan spasi depan-belakang
func sanitizeInput(s string) string {
	return strings.TrimSpace(s)
}

// ✅ Hilangkan spasi + ubah ke huruf kecil (untuk email/domain)
func sanitizeLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
