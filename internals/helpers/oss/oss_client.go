// internals/helpers/oss_file.go
package helper

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/chai2010/webp"
	"github.com/google/uuid"
)

// ConvertToWebP: decode (jpeg/png/webp) lalu re-encode ke webp
func ConvertToWebP(file multipart.File, filename string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	var img image.Image
	var err error

	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	case ".webp": // sudah webp, tinggal baca langsung
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(file); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("format tidak didukung: %s", ext)
	}
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	// quality bisa 80-95, sesuaikan
	if err := webp.Encode(buf, img, &webp.Options{Lossless: false, Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UploadAsWebP: konversi dulu ke WebP, lalu upload dengan ekstensi .webp
func (s *OSSService) UploadAsWebP(ctx context.Context, fh *multipart.FileHeader, keyPrefix string) (string, error) {
	src, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	webpData, err := ConvertToWebP(src, fh.Filename)
	if err != nil {
		return "", err
	}

	// ganti ekstensi jadi .webp
	base := strings.TrimSuffix(fh.Filename, filepath.Ext(fh.Filename))
	key := s.buildObjectKey(base + ".webp")
	if keyPrefix != "" {
		key = keyPrefix + "/" + key
	}

	// Upload ke OSS
	opts := []oss.Option{
		oss.WithContext(ctx),
		oss.ContentType("image/webp"),
		oss.ContentDisposition("inline"),
	}
	if err := s.Bucket.PutObject(key, bytes.NewReader(webpData), opts...); err != nil {
		return "", err
	}
	return s.PublicURL(key), nil
}


func init() {
	// Daftarkan beberapa MIME yang mungkin belum dikenal environment tertentu
	_ = mime.AddExtensionType(".webp", "image/webp")
	_ = mime.AddExtensionType(".avif", "image/avif")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")
}

// OSSService membungkus client + bucket + info env
type OSSService struct {
	Client     *oss.Client
	Bucket     *oss.Bucket
	Endpoint   string
	BucketName string
	Prefix     string // optional: "uploads/"
}

// NewOSSServiceFromEnv: inisialisasi dari ENV + verifikasi ringan lokasi bucket
// Env: ALI_OSS_ENDPOINT, ALI_OSS_ACCESS_KEY, ALI_OSS_SECRET_KEY, ALI_OSS_SECURITY_TOKEN (opsional), ALI_OSS_BUCKET
func NewOSSServiceFromEnv(prefix string) (*OSSService, error) {
	endpoint := strings.TrimSpace(getEnv("ALI_OSS_ENDPOINT"))
	ak := strings.TrimSpace(getEnv("ALI_OSS_ACCESS_KEY"))
	sk := strings.TrimSpace(getEnv("ALI_OSS_SECRET_KEY"))
	sts := strings.TrimSpace(getEnv("ALI_OSS_SECURITY_TOKEN"))
	bucketName := strings.TrimSpace(getEnv("ALI_OSS_BUCKET"))
	if endpoint == "" || ak == "" || sk == "" || bucketName == "" {
		return nil, fmt.Errorf("missing env: ALI_OSS_ENDPOINT/ACCESS_KEY/SECRET_KEY/BUCKET")
	}

	var (
		client *oss.Client
		err    error
	)
	if sts != "" {
		client, err = oss.New(endpoint, ak, sk, oss.SecurityToken(sts))
	} else {
		client, err = oss.New(endpoint, ak, sk)
	}
	if err != nil {
		return nil, fmt.Errorf("oss.New: %w", err)
	}

	bkt, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("client.Bucket: %w", err)
	}

	// Verifikasi ringan lokasi bucket (boleh dilewati bila ditolak RAM policy)
	if loc, err := client.GetBucketLocation(bucketName); err != nil {
		if se, ok := err.(oss.ServiceError); ok && se.StatusCode == 403 && se.Code == "AccessDenied" {
			log.Printf("[OSS] warn: skip location check due to AccessDenied (bucket=%s). Continuing.", bucketName)
		} else {
			return nil, fmt.Errorf("verify bucket: %w", err)
		}
	} else {
		log.Printf("[OSS] bucket %s location: %s", bucketName, loc)
	}

	return &OSSService{
		Client:     client,
		Bucket:     bkt,
		Endpoint:   endpoint,
		BucketName: bucketName,
		Prefix:     strings.Trim(prefix, "/"),
	}, nil
}

// -------------------- PUBLIC UTIL --------------------

// PublicURL menebak URL publik (untuk bucket public-read) atau override via ALI_OSS_PUBLIC_BASE
func (s *OSSService) PublicURL(key string) string {
	if key == "" {
		return ""
	}
	if base := strings.TrimSpace(os.Getenv("ALI_OSS_PUBLIC_BASE")); base != "" {
		return strings.TrimRight(base, "/") + "/" + key
	}
	if s.Endpoint == "" || s.BucketName == "" {
		return ""
	}
	end := s.Endpoint
	end = strings.TrimPrefix(end, "https://")
	end = strings.TrimPrefix(end, "http://")
	return fmt.Sprintf("https://%s.%s/%s", s.BucketName, end, key)
}

// ExtractKeyFromPublicURL: mengembalikan object key dari URL publik
// - Jika ALI_OSS_PUBLIC_BASE diset, diekstrak relatif terhadap base itu.
// - Jika tidak, fallback pola "https://bucket.endpoint/<key>"
func ExtractKeyFromPublicURL(publicURL string) (string, error) {
	if publicURL == "" {
		return "", fmt.Errorf("empty url")
	}

	// 1) coba dengan ALI_OSS_PUBLIC_BASE
	if base := strings.TrimSpace(os.Getenv("ALI_OSS_PUBLIC_BASE")); base != "" {
		base = strings.TrimRight(base, "/") + "/"
		if strings.HasPrefix(publicURL, base) {
			return strings.TrimPrefix(publicURL, base), nil
		}
	}

	// 2) fallback: hapus scheme lalu ambil path setelah domain
	u := publicURL
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if i := strings.Index(u, "/"); i >= 0 {
		return u[i+1:], nil
	}
	return "", fmt.Errorf("cannot extract key from url: %s", publicURL)
}

// SignURLGET buat link sementara GET
func (s *OSSService) SignURLGET(key string, expire time.Duration) (string, error) {
	return s.Bucket.SignURL(key, oss.HTTPGet, int64(expire.Seconds()))
}

// SignURLPUT buat link sementara PUT (direct upload dari client)
func (s *OSSService) SignURLPUT(key string, expire time.Duration, contentType string) (string, error) {
	opts := []oss.Option{}
	if contentType != "" {
		opts = append(opts, oss.ContentType(contentType))
	}
	return s.Bucket.SignURL(key, oss.HTTPPut, int64(expire.Seconds()), opts...)
}

// Exists cek apakah object ada (HEAD)
func (s *OSSService) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.Bucket.GetObjectMeta(key, oss.WithContext(ctx))
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// -------------------- CREATE (UPLOAD) --------------------

// UploadFromFormFile: upload dari multipart file + set header agar inline dan Content-Type benar.
// Menghasilkan key final dan contentType terdeteksi.
func (s *OSSService) UploadFromFormFile(ctx context.Context, fh *multipart.FileHeader) (string, string, error) {
	if fh == nil {
		return "", "", fmt.Errorf("nil file header")
	}
	key := s.buildObjectKey(fh.Filename)

	src, err := fh.Open()
	if err != nil {
		return "", "", fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	ct, reader, err := detectContentType(src, fh.Filename)
	if err != nil {
		return "", "", err
	}

	opts := []oss.Option{
		oss.WithContext(ctx),
		oss.ContentType(ct),
		oss.ContentDisposition("inline"),
		oss.CacheControl("public, max-age=31536000, immutable"),
	}
	if err := s.Bucket.PutObject(key, reader, opts...); err != nil {
		return "", "", err
	}
	return key, ct, nil
}

// UploadStream: upload dari io.Reader/Seeker (misal bytes.Reader)
func (s *OSSService) UploadStream(ctx context.Context, key string, r io.Reader, contentType string, inline bool, cacheForever bool) error {
	if key == "" {
		return fmt.Errorf("empty key")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	opts := []oss.Option{
		oss.WithContext(ctx),
		oss.ContentType(contentType),
	}
	if inline {
		opts = append(opts, oss.ContentDisposition("inline"))
	}
	if cacheForever {
		opts = append(opts, oss.CacheControl("public, max-age=31536000, immutable"))
	}
	return s.Bucket.PutObject(key, r, opts...)
}

// -------------------- UPDATE --------------------

// UpdateMeta: perbarui metadata (tanpa reupload) via CopyObject+MetaReplace
func (s *OSSService) UpdateMeta(ctx context.Context, key string, newContentType string, inline bool, cacheForever bool) error {
	if key == "" {
		return fmt.Errorf("empty key")
	}
	if newContentType == "" {
		newContentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(key)))
		if newContentType == "" {
			newContentType = "application/octet-stream"
		}
	}
	opts := []oss.Option{
		oss.WithContext(ctx),
		oss.MetadataDirective(oss.MetaReplace),
		oss.ContentType(newContentType),
	}
	if inline {
		opts = append(opts, oss.ContentDisposition("inline"))
	}
	if cacheForever {
		opts = append(opts, oss.CacheControl("public, max-age=31536000, immutable"))
	}
	_, err := s.Bucket.CopyObject(key, key, opts...)
	return err
}

// ReplaceObject: ganti isi object dari sourceKey ke dstKey (server-side copy) + set metadata baru
func (s *OSSService) ReplaceObject(ctx context.Context, dstKey, srcKey string, contentType string, inline bool, cacheForever bool) error {
	if dstKey == "" || srcKey == "" {
		return fmt.Errorf("empty key")
	}
	opts := []oss.Option{
		oss.WithContext(ctx),
		oss.MetadataDirective(oss.MetaReplace),
	}
	if contentType != "" {
		opts = append(opts, oss.ContentType(contentType))
	}
	if inline {
		opts = append(opts, oss.ContentDisposition("inline"))
	}
	if cacheForever {
		opts = append(opts, oss.CacheControl("public, max-age=31536000, immutable"))
	}
	_, err := s.Bucket.CopyObject(srcKey, dstKey, opts...)
	return err
}

// -------------------- DELETE --------------------

func (s *OSSService) DeleteObject(ctx context.Context, key string) error {
	return s.Bucket.DeleteObject(key, oss.WithContext(ctx))
}

func (s *OSSService) DeleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	_, err := s.Bucket.DeleteObjects(keys, oss.WithContext(ctx))
	return err
}

// -------------------- INTERNAL UTILS --------------------

func (s *OSSService) buildObjectKey(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.TrimSuffix(filename, ext)
	if base == "" {
		base = "file"
	}
	ts := time.Now().Format("20060102_150405")
	rand6 := randHex(3)

	prefix := s.Prefix
	if prefix != "" {
		prefix += "/"
	}
	return fmt.Sprintf("%s%s_%s_%s%s", prefix, slugify(base), ts, rand6, ext)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	r := strings.NewReplacer(" ", "-", "_", "-", "—", "-", "–", "-")
	s = r.Replace(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, s)
	if s == "" {
		return "file"
	}
	return s
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// detectContentType: tentukan contentType dari ekstensi + sniff 512B, lalu hard-override utk jenis tertentu
func detectContentType(src multipart.File, filename string) (string, io.Reader, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	ct := mime.TypeByExtension(ext)

	// Sniff 512 byte
	head := make([]byte, 512)
	n, _ := io.ReadFull(io.LimitReader(src, 512), head)
	if seeker, ok := src.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	if n > 0 {
		detected := http.DetectContentType(head[:n])
		if ct == "" || ct == "application/octet-stream" {
			ct = detected
		}
	}

	// Hard override agar tidak jatuh ke octet-stream untuk format gambar modern
	switch ext {
	case ".webp":
		ct = "image/webp"
	case ".avif":
		ct = "image/avif"
	case ".svg":
		ct = "image/svg+xml"
	}

	if ct == "" {
		ct = "application/octet-stream"
	}

	// Jika seekable, kembalikan src setelah reset
	if seeker, ok := src.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
		return ct, src, nil
	}

	// Jika tidak seekable (jarang), gabungkan head + sisa
	combined := append([]byte{}, head[:n]...)
	body, _ := io.ReadAll(src)
	combined = append(combined, body...)
	return ct, bytes.NewReader(combined), nil
}

func isNotFound(err error) bool {
	if e, ok := err.(oss.ServiceError); ok {
		return e.StatusCode == 404
	}
	return false
}

func getEnv(k string) string { return strings.TrimSpace(os.Getenv(k)) }

// -------------------- PATH & SUBDIR HELPERS --------------------

func safePart(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	if s == "" {
		return "unknown"
	}
	return slugify(s)
}

func joinParts(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		clean = append(clean, safePart(p))
	}
	return strings.Join(clean, "/")
}

// UploadFromFormFileToDir: mirip UploadFromFormFile tetapi key berada di sub-direktori `dir`
func (s *OSSService) UploadFromFormFileToDir(ctx context.Context, dir string, fh *multipart.FileHeader) (string, string, error) {
	if fh == nil {
		return "", "", fmt.Errorf("nil file header")
	}

	keyPrefix := s.Prefix
	if keyPrefix != "" {
		keyPrefix += "/"
	}
	dir = strings.Trim(dir, "/")
	if dir != "" {
		keyPrefix += dir + "/"
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	base := strings.TrimSuffix(fh.Filename, ext)
	if base == "" {
		base = "file"
	}
	ts := time.Now().Format("20060102_150405")
	rand6 := randHex(3)
	key := fmt.Sprintf("%s%s_%s_%s%s", keyPrefix, slugify(base), ts, rand6, ext)

	src, err := fh.Open()
	if err != nil {
		return "", "", fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	ct, reader, err := detectContentType(src, fh.Filename)
	if err != nil {
		return "", "", err
	}

	opts := []oss.Option{
		oss.WithContext(ctx),
		oss.ContentType(ct),
		oss.ContentDisposition("inline"),
		oss.CacheControl("public, max-age=31536000, immutable"),
	}
	if err := s.Bucket.PutObject(key, reader, opts...); err != nil {
		return "", "", err
	}
	return key, ct, nil
}

// -------------------- HIGH-LEVEL HELPERS --------------------

// UploadImageToOSSScoped: convenience upload ke path "masjids/{masjid_id}/{kategori}"
// Gunakan NewOSSServiceFromEnv("") agar tersimpan di root bucket (tanpa "uploads")
func UploadImageToOSSScoped(masjidID uuid.UUID, kategori string, fh *multipart.FileHeader) (string, error) {
	if masjidID == uuid.Nil {
		return "", fmt.Errorf("masjidID kosong/invalid")
	}
	if strings.TrimSpace(kategori) == "" {
		kategori = "misc"
	}

	// base prefix kosong → langsung root bucket
	svc, err := NewOSSServiceFromEnv("")
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dir := joinParts("masjids", masjidID.String(), kategori)
	key, _, err := svc.UploadFromFormFileToDir(ctx, dir, fh)
	if err != nil {
		return "", err
	}
	return svc.PublicURL(key), nil
}

// UploadImageToOSS: helper lama (kompatibilitas)
// Inisialisasi OSSService dari env tiap dipanggil (cukup untuk use-case ringan).
func UploadImageToOSS(prefix string, fh *multipart.FileHeader) (string, error) {
	svc, err := NewOSSServiceFromEnv(prefix)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	key, _, err := svc.UploadFromFormFile(ctx, fh)
	if err != nil {
		return "", err
	}
	return svc.PublicURL(key), nil
}

// -------------------- DELETE BY PUBLIC URL (SINGLE & BATCH) --------------------

// DeleteByPublicURL (method): hapus satu objek berdasarkan public URL.
func (s *OSSService) DeleteByPublicURL(ctx context.Context, publicURL string) error {
	if strings.TrimSpace(publicURL) == "" {
		return fmt.Errorf("empty public url")
	}
	key, err := ExtractKeyFromPublicURL(publicURL)
	if err != nil {
		return fmt.Errorf("extract key: %w", err)
	}
	return s.DeleteObject(ctx, key)
}

// DeleteManyByPublicURL (method): hapus banyak objek berdasarkan public URL.
// Akan melakukan chunking ≤ 1000 key per request (batas DeleteObjects OSS).
// Mengembalikan daftar URL yang sukses dihapus dan map error per-URL yang gagal.
func (s *OSSService) DeleteManyByPublicURL(ctx context.Context, publicURLs []string) (deleted []string, failed map[string]error) {
	failed = make(map[string]error)
	if len(publicURLs) == 0 {
		return nil, failed
	}

	// 1) Ubah URL -> key, kumpulkan mapping untuk laporan yang jelas
	type item struct {
		url string
		key string
	}
	items := make([]item, 0, len(publicURLs))
	for _, u := range publicURLs {
		u = strings.TrimSpace(u)
	if u == "" {
			continue
		}
		key, err := ExtractKeyFromPublicURL(u)
		if err != nil {
			failed[u] = fmt.Errorf("extract key: %w", err)
			continue
		}
		items = append(items, item{url: u, key: key})
	}
	if len(items) == 0 {
		return nil, failed
	}

	// 2) Chunking keys (≤1000 per batch)
	const maxChunk = 1000
	for start := 0; start < len(items); start += maxChunk {
		end := start + maxChunk
		if end > len(items) {
			end = len(items)
		}
		chunk := items[start:end]

		keys := make([]string, 0, len(chunk))
		urlByKey := make(map[string]string, len(chunk))
		for _, it := range chunk {
			keys = append(keys, it.key)
			urlByKey[it.key] = it.url
		}

		// 3) DeleteObjects batch
		if _, err := s.Bucket.DeleteObjects(keys, oss.WithContext(ctx)); err != nil {
			// jika ingin lebih granular, bisa parse ServiceError.XML
			for _, k := range keys {
				u := urlByKey[k]
				failed[u] = fmt.Errorf("delete: %w", err)
			}
			continue
		}

		// 4) Tandai berhasil
		for _, k := range keys {
			deleted = append(deleted, urlByKey[k])
		}
	}

	return deleted, failed
}

// -------------------- CONVENIENCE (INISIASI DARI ENV) --------------------

// DeleteByPublicURLENV: helper cepat untuk hapus satu URL tanpa harus bikin service manual.
// Gunakan NewOSSServiceFromEnv("") agar konsisten dengan UploadImageToOSSScoped (root bucket).
func DeleteByPublicURLENV(publicURL string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	svc, err := NewOSSServiceFromEnv("")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return svc.DeleteByPublicURL(ctx, publicURL)
}

// DeleteManyByPublicURLENV: helper cepat batch-delete berdasarkan public URL.
// Mengembalikan daftar yang sukses dan map error per URL.
func DeleteManyByPublicURLENV(publicURLs []string, timeoutPerBatch time.Duration) (deleted []string, failed map[string]error, err error) {
	if timeoutPerBatch <= 0 {
		timeoutPerBatch = 30 * time.Second
	}
	svc, err := NewOSSServiceFromEnv("")
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeoutPerBatch)
	defer cancel()
	deleted, failed = svc.DeleteManyByPublicURL(ctx, publicURLs)
	return deleted, failed, nil
}
