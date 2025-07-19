package constants

import (
	"path/filepath"
	"strings"
)

func DetectFileTypeFromExt(filename string) int {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".mp3", ".wav":
		return 2 // Audio
	case ".doc", ".docx":
		return 3 // DOCX
	case ".pdf":
		return 4 // PDF
	case ".ppt", ".pptx":
		return 5 // PPT
	case ".png", ".jpg", ".jpeg", ".webp":
		return 6 // Image
	default:
		return 99 // Tidak diketahui
	}
}
