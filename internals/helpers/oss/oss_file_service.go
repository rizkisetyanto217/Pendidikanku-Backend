package helper

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/*
BlobService adalah facade upload/hapus yang seragam untuk controller.

Tambahan penting:
- UploadRawToDirWithKey(ctx, dir, fh) -> (publicURL, objectKey, contentType, err)
  => dipakai untuk aset GLOBAL (tidak terkait masjid_id), mis. gambar Service Plan.
- TryGetImageFile/GetImageFile util untuk ambil file multipart dari berbagai nama field.
*/

type BlobService interface {
	// ----------- existing (tetap kompat) -----------
	UploadImage(ctx context.Context, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (publicURL string, err error)
	UploadAny(ctx context.Context, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (publicURL string, err error)
	UploadRawToDir(ctx context.Context, dir string, fh *multipart.FileHeader) (publicURL, contentType string, err error)

	DeleteByPublicURL(ctx context.Context, publicURL string) error
	DeleteManyByPublicURL(ctx context.Context, publicURLs []string) (deleted []string, failed map[string]error, err error)
	MoveToSpam(ctx context.Context, publicURL string) (spamURL string, err error)

	// ----------- new -----------
	// Mengembalikan juga objectKey untuk disimpan di DB.
	UploadRawToDirWithKey(ctx context.Context, dir string, fh *multipart.FileHeader) (publicURL, objectKey, contentType string, err error)
}

// --------------------------------------------------
// Implementasi berbasis Aliyun OSS (OSSService)
// --------------------------------------------------

type OSSBlobService struct {
	svc *OSSService
}

// Buat instance dari ENV. prefix opsional (contoh: "uploads/")
func NewOSSBlobServiceFromEnv(prefix string) (*OSSBlobService, error) {
	s, err := NewOSSServiceFromEnv(prefix)
	if err != nil {
		return nil, err
	}
	return &OSSBlobService{svc: s}, nil
}

func (b *OSSBlobService) UploadImage(ctx context.Context, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (string, error) {
	if fh == nil {
		return "", fiber.NewError(fiber.StatusBadRequest, "File tidak ditemukan")
	}
	if masjidID == uuid.Nil {
		return "", fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak valid")
	}
	url, err := UploadImageToOSS(ctx, b.svc, masjidID, slot, fh) // re-encode → WebP
	if err != nil {
		return "", err
	}
	return url, nil
}

func (b *OSSBlobService) UploadAny(ctx context.Context, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (string, error) {
	if fh == nil {
		return "", fiber.NewError(fiber.StatusBadRequest, "File tidak ditemukan")
	}
	if masjidID == uuid.Nil {
		return "", fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak valid")
	}
	url, err := UploadAnyToOSS(ctx, b.svc, masjidID, slot, fh) // image→WebP, non-image→raw
	if err != nil {
		return "", err
	}
	return url, nil
}

func (b *OSSBlobService) UploadRawToDir(ctx context.Context, dir string, fh *multipart.FileHeader) (string, string, error) {
	if fh == nil {
		return "", "", fiber.NewError(fiber.StatusBadRequest, "File tidak ditemukan")
	}
	key, ct, err := b.svc.UploadFromFormFileToDir(ctx, dir, fh) // raw upload ke subdir bebas
	if err != nil {
		return "", "", fiber.NewError(fiber.StatusBadGateway, "Gagal upload ke OSS")
	}
	return b.svc.PublicURL(key), ct, nil
}

// NEW: versi yang juga mengembalikan objectKey
func (b *OSSBlobService) UploadRawToDirWithKey(ctx context.Context, dir string, fh *multipart.FileHeader) (string, string, string, error) {
	if fh == nil {
		return "", "", "", fiber.NewError(fiber.StatusBadRequest, "File tidak ditemukan")
	}
	key, ct, err := b.svc.UploadFromFormFileToDir(ctx, dir, fh)
	if err != nil {
		return "", "", "", fiber.NewError(fiber.StatusBadGateway, "Gagal upload ke OSS")
	}
	return b.svc.PublicURL(key), key, ct, nil
}

func (b *OSSBlobService) DeleteByPublicURL(ctx context.Context, publicURL string) error {
	if strings.TrimSpace(publicURL) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "URL kosong")
	}
	if err := b.svc.DeleteByPublicURL(ctx, publicURL); err != nil {
		return fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("Gagal hapus object: %v", err))
	}
	return nil
}

func (b *OSSBlobService) DeleteManyByPublicURL(ctx context.Context, publicURLs []string) ([]string, map[string]error, error) {
	if len(publicURLs) == 0 {
		return nil, map[string]error{}, nil
	}
	deleted, failed := b.svc.DeleteManyByPublicURL(ctx, publicURLs)
	return deleted, failed, nil
}

func (b *OSSBlobService) MoveToSpam(ctx context.Context, publicURL string) (string, error) {
	if strings.TrimSpace(publicURL) == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, "URL kosong")
	}
	spamURL, err := MoveToSpamByPublicURLENV(publicURL, 0)
	if err != nil {
		return "", fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("Gagal memindahkan ke spam: %v", err))
	}
	return spamURL, nil
}

// --------------------------------------------------
// Helper kecil untuk controller
// --------------------------------------------------

// IsMultipart menilai request multipart/form-data
func IsMultipart(c *fiber.Ctx) bool {
	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	return strings.HasPrefix(ct, "multipart/form-data")
}

// Nama-nama field umum untuk upload gambar
var defaultImageFields = []string{
	"image", "file", "photo", "picture",
	"masjid_service_plan_image", // khusus case service plan
}

// GetImageFile mencari file dari beberapa kemungkinan field form.
// Jika tidak ada file, kembalikan (nil, nil) supaya controller bisa fallback.
func GetImageFile(c *fiber.Ctx, fieldNames ...string) (*multipart.FileHeader, error) {
	if !IsMultipart(c) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Gunakan multipart/form-data")
	}
	names := fieldNames
	if len(names) == 0 {
		names = defaultImageFields
	}
	for _, fn := range names {
		if fh, err := c.FormFile(fn); err == nil && fh != nil {
			return fh, nil
		}
	}
	return nil, nil
}

// TryGetImageFile hanyalah alias yang lebih “friendly”
func TryGetImageFile(c *fiber.Ctx) (*multipart.FileHeader, error) {
	return GetImageFile(c)
}

// --------------------------------------------------
// Mock untuk unit test
// --------------------------------------------------

type MockBlobService struct {
	UploadImageFn              func(ctx context.Context, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (string, error)
	UploadAnyFn                func(ctx context.Context, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (string, error)
	UploadRawToDirFn           func(ctx context.Context, dir string, fh *multipart.FileHeader) (string, string, error)
	UploadRawToDirWithKeyFn    func(ctx context.Context, dir string, fh *multipart.FileHeader) (string, string, string, error)
	DeleteByPublicURLFn        func(ctx context.Context, publicURL string) error
	DeleteManyByPublicURLFn    func(ctx context.Context, publicURLs []string) ([]string, map[string]error, error)
	MoveToSpamFn               func(ctx context.Context, publicURL string) (string, error)
}

func (m *MockBlobService) UploadImage(ctx context.Context, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (string, error) {
	if m.UploadImageFn == nil { return "", errors.New("not implemented") }
	return m.UploadImageFn(ctx, masjidID, slot, fh)
}
func (m *MockBlobService) UploadAny(ctx context.Context, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (string, error) {
	if m.UploadAnyFn == nil { return "", errors.New("not implemented") }
	return m.UploadAnyFn(ctx, masjidID, slot, fh)
}
func (m *MockBlobService) UploadRawToDir(ctx context.Context, dir string, fh *multipart.FileHeader) (string, string, error) {
	if m.UploadRawToDirFn == nil { return "", "", errors.New("not implemented") }
	return m.UploadRawToDirFn(ctx, dir, fh)
}
func (m *MockBlobService) UploadRawToDirWithKey(ctx context.Context, dir string, fh *multipart.FileHeader) (string, string, string, error) {
	if m.UploadRawToDirWithKeyFn == nil { return "", "", "", errors.New("not implemented") }
	return m.UploadRawToDirWithKeyFn(ctx, dir, fh)
}
func (m *MockBlobService) DeleteByPublicURL(ctx context.Context, publicURL string) error {
	if m.DeleteByPublicURLFn == nil { return errors.New("not implemented") }
	return m.DeleteByPublicURLFn(ctx, publicURL)
}
func (m *MockBlobService) DeleteManyByPublicURL(ctx context.Context, publicURLs []string) ([]string, map[string]error, error) {
	if m.DeleteManyByPublicURLFn == nil { return nil, nil, errors.New("not implemented") }
	return m.DeleteManyByPublicURLFn(ctx, publicURLs)
}
func (m *MockBlobService) MoveToSpam(ctx context.Context, publicURL string) (string, error) {
	if m.MoveToSpamFn == nil { return "", errors.New("not implemented") }
	return m.MoveToSpamFn(ctx, publicURL)
}

// --------------------------------------------------
// Retensi trash (for OLD image rotation)
// --------------------------------------------------

func TrashRetention() time.Duration {
	days := getEnvInt("RETENTION_DAYS", 30)
	return time.Duration(days) * 24 * time.Hour
}

func (b *OSSBlobService) Retention() time.Duration {
	return TrashRetention()
}
