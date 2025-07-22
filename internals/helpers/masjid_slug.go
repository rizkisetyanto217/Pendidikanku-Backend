package helper

import (
	"regexp"
	"strings"
)

func GenerateSlug(name string) string {
	// Ubah ke huruf kecil
	slug := strings.ToLower(name)

	// Ganti spasi dan underscore jadi tanda hubung
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Hapus karakter non-alfanumerik dan -
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = reg.ReplaceAllString(slug, "")

	// Hapus tanda hubung dobel
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	// Trim tanda hubung di depan dan belakang
	slug = strings.Trim(slug, "-")

	return slug
}
