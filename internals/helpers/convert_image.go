package helper

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
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

	filename := GenerateUniqueFilename(folder, fileHeader.Filename) // tanpanya .webp
	contentType := fileHeader.Header.Get("Content-Type")

	// ✅ Gunakan bucket "image"
	if err := UploadToSupabase("image", filename, contentType, buf); err != nil {
		return "", fmt.Errorf("upload gambar gagal: %w", err)
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/image/%s",
		os.Getenv("SUPABASE_PROJECT_URL"),
		url.PathEscape(filename),
	)

	return publicURL, nil
}


// ✅ Upload image setelah resize + kompresi WebP maksimal 65KB
func UploadImageAsWebPToSupabase(folder string, fileHeader *multipart.FileHeader) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("gagal membuka file gambar: %w", err)
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return "", fmt.Errorf("format gambar tidak dikenali: %w", err)
	}

	resized := imaging.Resize(img, 1080, 0, imaging.Lanczos)

	var buf *bytes.Buffer
	var sizeKB int
	success := false
	for _, quality := range []float32{60, 50, 40, 30, 20} {
		tmp := new(bytes.Buffer)
		if err := webp.Encode(tmp, resized, &webp.Options{Quality: quality}); err != nil {
			continue
		}
		sizeKB = tmp.Len() / 1024
		if sizeKB <= 65 {
			buf = tmp
			success = true
			break
		}
	}
	if !success {
		return "", fmt.Errorf("ukuran gambar setelah kompresi tetap melebihi 65 KB (terkecil %d KB)", sizeKB)
	}

	filename := GenerateUniqueFilename(folder, fileHeader.Filename) + ".webp"

	// ✅ Gunakan bucket "image"
	if err := UploadToSupabase("image", filename, "image/webp", buf); err != nil {
		return "", fmt.Errorf("upload gambar gagal: %w", err)
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/image/%s",
		os.Getenv("SUPABASE_PROJECT_URL"),
		url.PathEscape(filename),
	)

	return publicURL, nil
}

// ✅ Buat nama unik
func GenerateUniqueFilename(folder, originalFilename string) string {
	timestamp := time.Now().Format("20060102")
	uuidStr := uuid.New().String()
	return fmt.Sprintf("%s/%s-%s", folder, timestamp, uuidStr+"-"+originalFilename)
}

func UploadToSupabase(bucket, filename, contentType string, data *bytes.Buffer) error {
	supabaseURL := os.Getenv("SUPABASE_PROJECT_URL")
	supabaseKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")

	if supabaseURL == "" || supabaseKey == "" {
		return fmt.Errorf("SUPABASE_PROJECT_URL atau SUPABASE_SERVICE_ROLE_KEY belum diset")
	}

	// ✅ Simple PUT endpoint untuk upload langsung
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
		return fmt.Errorf("upload gagal status %d: %s", resp.StatusCode, string(body))
	}

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