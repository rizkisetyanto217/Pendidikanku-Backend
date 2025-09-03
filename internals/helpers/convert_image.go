package helper

import (
	"bytes"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	// "github.com/chai2010/webp"
	// "github.com/disintegration/imaging"
	"github.com/google/uuid"
)

func UploadImageToSupabase(folder string, fileHeader *multipart.FileHeader) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("gagal membuka file gambar: %w", err)
	}
	defer src.Close()

	// Baca semua isi file
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, src); err != nil {
		return "", fmt.Errorf("gagal membaca file gambar: %w", err)
	}

	// ✅ Validasi ukuran maksimal 300KB
	// const maxSize = 300 * 1024
	// if buf.Len() > maxSize {
	// 	return "", fmt.Errorf("ukuran gambar melebihi 200KB (%dKB)", buf.Len()/1024)
	// }

	filename := GenerateUniqueFilename(folder, fileHeader.Filename)
	contentType := fileHeader.Header.Get("Content-Type")

	// Upload ke Supabase
	if err := UploadToSupabase("image", filename, contentType, buf); err != nil {
		return "", fmt.Errorf("upload gambar gagal: %w", err)
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/image/%s",
		os.Getenv("SUPABASE_PROJECT_URL"),
		url.PathEscape(filename),
	)

	return publicURL, nil
}

// ✅ Buat nama unik
func sanitizeFilename(filename string) string {
	// Hapus karakter selain huruf, angka, titik, dash, underscore
	re := regexp.MustCompile(`[^a-zA-Z0-9.\-_]+`)
	safe := re.ReplaceAllString(filename, "_")
	return safe
}

func GenerateUniqueFilename(folder, originalFilename string) string {
	timestamp := time.Now().Format("20060102")
	uuidStr := uuid.New().String()
	safeFilename := sanitizeFilename(originalFilename)
	return fmt.Sprintf("%s/%s-%s-%s", folder, timestamp, uuidStr, safeFilename)
}

func UploadToSupabase(bucket, filename, contentType string, data *bytes.Buffer) error {
	supabaseURL := os.Getenv("SUPABASE_PROJECT_URL")
	supabaseKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")

	// ✅ DEBUG: cek env
	fmt.Println("📦 Upload to Supabase")
	fmt.Println("🔗 URL:", supabaseURL)
	fmt.Println("🗝️ Key exists:", supabaseKey != "")
	fmt.Println("📁 Bucket:", bucket)
	fmt.Println("📄 Filename:", filename)
	fmt.Println("🧾 Content-Type:", contentType)
	fmt.Println("📏 Size (bytes):", data.Len())

	if supabaseURL == "" || supabaseKey == "" {
		return fmt.Errorf("SUPABASE_PROJECT_URL atau SUPABASE_SERVICE_ROLE_KEY belum diset")
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, filename)

	req, err := http.NewRequest("PUT", url, data)
	if err != nil {
		return fmt.Errorf("gagal membuat request upload: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal mengirim request upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("❌ Upload gagal:", string(body))
		return fmt.Errorf("upload gagal status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Println("✅ Upload sukses ke:", url)
	return nil
}

// ✅ Hapus file dari Supabase
func DeleteFromSupabase(bucket, path string) error {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s",
		os.Getenv("SUPABASE_PROJECT_URL"), bucket, path)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete gagal status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func ExtractSupabasePath(fullURL string) (bucket string, path string, err error) {
	u, err := url.Parse(fullURL)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(u.Path, "/object/public/", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("url tidak valid untuk Supabase public object")
	}

	pathParts := strings.SplitN(parts[1], "/", 2)
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("gagal ekstrak bucket dan path")
	}

	return pathParts[0], pathParts[1], nil
}

// ✅ Ambil path dari URL
func ExtractSupabaseStoragePath(fullURL string) string {
	parts := strings.Split(fullURL, "/storage/v1/object/public/image/")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func UploadFileToSupabase(folder string, fileHeader *multipart.FileHeader) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("gagal membuka file: %w", err)
	}
	defer src.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, src); err != nil {
		return "", fmt.Errorf("gagal membaca isi file: %w", err)
	}

	filename := GenerateUniqueFilename(folder, fileHeader.Filename)
	contentType := fileHeader.Header.Get("Content-Type")

	if err := UploadToSupabase("file", filename, contentType, buf); err != nil {
		return "", fmt.Errorf("upload file gagal: %w", err)
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/file/%s",
		os.Getenv("SUPABASE_PROJECT_URL"),
		url.PathEscape(filename),
	)

	return publicURL, nil
}
