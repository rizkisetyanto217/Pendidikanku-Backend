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
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

func getEnv(k string) string { return strings.TrimSpace(os.Getenv(k)) }

func envInt(key string, def int) int {
	if v := getEnv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return def
}
func envFloat(key string, def float32) float32 {
	if v := getEnv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil && f >= 0 {
			return float32(f)
		}
	}
	return def
}

var (
	// batas ukuran uploader di controller (tetap dipakai sebagai guard ringan)
	maxUploadSize = int64(5 * 1024 * 1024)
)

/* =======================================================================
   Konfigurasi WebP (ENV-Driven) + Opsi per-call
======================================================================= */

type WebPOptions struct {
	MaxW         int     // batas lebar (resize keep-aspect)
	MaxH         int     // batas tinggi
	TargetKB     int     // target ukuran; 0 = non-aktif (pakai Quality saja)
	Quality      float32 // default quality saat TargetKB=0 atau initial guess
	MinQ         float32 // min quality utk binary search
	MaxQ         float32 // max quality utk binary search
	ToleranceKB  int     // toleransi di atas target
	Lossless     bool    // jarang dipakai; default false (foto = lossy)
	// + tambahkan di WebPOptions
	MinW       int     // lebar minimum saat iterative downscale
	MinH       int     // tinggi minimum
	ScaleStep  float32 // faktor perkecil tiap iterasi (0<step<1), mis. 0.85 = 85%

}




func defaultWebPOptionsFromEnv() WebPOptions {
	return WebPOptions{
		MaxW:        envInt("IMAGE_WEBP_MAX_W", 1600),
		MaxH:        envInt("IMAGE_WEBP_MAX_H", 1600),
		TargetKB:    envInt("IMAGE_WEBP_TARGET_KB", 0),
		Quality:     envFloat("IMAGE_WEBP_QUALITY", 80),
		MinQ:        envFloat("IMAGE_WEBP_MIN_Q", 45),
		MaxQ:        envFloat("IMAGE_WEBP_MAX_Q", 85),
		ToleranceKB: envInt("IMAGE_WEBP_TOLERANCE_KB", 8),
		Lossless:    false,
		MinW:      envInt("IMAGE_WEBP_MIN_W", 480),
		MinH:      envInt("IMAGE_WEBP_MIN_H", 480),
		ScaleStep: envFloat("IMAGE_WEBP_SCALE_STEP", 0.85),
	}
}

/* =======================================================================
   Decode gambar (jpeg/png/webp) dari []byte dengan sniff MIME
======================================================================= */

func decodeImage(all []byte, filename string) (image.Image, error) {
	if len(all) == 0 {
		return nil, fmt.Errorf("empty file")
	}
	head := all
	if len(head) > 512 {
		head = head[:512]
	}
	ct := http.DetectContentType(head)

	var (
		img image.Image
		err error
	)

	switch {
	case strings.Contains(ct, "jpeg"):
		img, err = jpeg.Decode(bytes.NewReader(all))
	case strings.Contains(ct, "png"):
		img, err = png.Decode(bytes.NewReader(all))
	case strings.Contains(ct, "webp"):
		img, err = webp.Decode(bytes.NewReader(all))
	default:
		// fallback by extension
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".jpg", ".jpeg":
			img, err = jpeg.Decode(bytes.NewReader(all))
		case ".png":
			img, err = png.Decode(bytes.NewReader(all))
		case ".webp":
			img, err = webp.Decode(bytes.NewReader(all))
		default:
			return nil, fmt.Errorf("format tidak didukung: %s / %s", ct, ext)
		}
	}
	return img, err
}

/* =======================================================================
   Resize helper (keep aspect). Pakai CatmullRom (kualitas bagus).
======================================================================= */

func downscaleIfNeeded(src image.Image, maxW, maxH int) image.Image {
	if maxW <= 0 && maxH <= 0 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if (maxW > 0 && w > maxW) || (maxH > 0 && h > maxH) {
		scale := 1.0
		if maxW > 0 {
			scale = math.Min(scale, float64(maxW)/float64(w))
		}
		if maxH > 0 {
			scale = math.Min(scale, float64(maxH)/float64(h))
		}
		nw := int(math.Round(float64(w) * scale))
		nh := int(math.Round(float64(h) * scale))
		if nw < 1 {
			nw = 1
		}
		if nh < 1 {
			nh = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
		return dst
	}
	return src
}

/* =======================================================================
   Encode WebP
   - Jika TargetKB > 0 → binary search quality hingga <= target+tol
   - Jika TargetKB = 0 → encode sekali dengan Quality
======================================================================= */

func encodeToWebP(img image.Image, opt WebPOptions) ([]byte, error) {
	// Lossless langsung encode (jarang dipakai untuk foto)
	if opt.Lossless {
		buf := new(bytes.Buffer)
		if err := webp.Encode(buf, img, &webp.Options{Lossless: true}); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	// Helper: encode dengan quality tertentu
	encodeQ := func(im image.Image, q float32) ([]byte, error) {
		buf := new(bytes.Buffer)
		if err := webp.Encode(buf, im, &webp.Options{Lossless: false, Quality: q}); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	// Tanpa target ukuran → encode sekali
	if opt.TargetKB <= 0 {
		q := opt.Quality
		if q <= 0 {
			q = 80
		}
		return encodeQ(img, q)
	}

	target := opt.TargetKB * 1024
	tol := opt.ToleranceKB * 1024
	if tol <= 0 {
		tol = 8 * 1024
	}
	minQ := opt.MinQ
	maxQ := opt.MaxQ
	if minQ <= 0 { minQ = 45 }
	if maxQ <= 0 { maxQ = 85 }
	if minQ > maxQ { minQ, maxQ = maxQ, minQ }

	minW := opt.MinW
	minH := opt.MinH
	if minW <= 0 { minW = 480 }
	if minH <= 0 { minH = 480 }
	step := opt.ScaleStep
	if step <= 0 || step >= 1 { step = 0.85 }

	// Mulai dari img yang sudah di-downscale awal (di ConvertToWebPWithOptions)
	cur := img
	last := []byte(nil)

	// Ulang sampai masuk target atau mentok minimum size
	for attempt := 0; attempt < 6; attempt++ {
		// ---- Binary search quality pada dimensi saat ini ----
		low, high := minQ, maxQ
		best := []byte(nil)

		for i := 0; i < 8; i++ { // 7–8 iterasi cukup
			q := (low + high) / 2
			data, err := encodeQ(cur, q)
			if err != nil {
				return nil, err
			}
			if len(data) <= target+tol {
				best = data
				high = q // coba kompresi lebih tinggi (q lebih kecil)
			} else {
				low = q // masih gede → turunkan quality
			}
		}
		if best == nil {
			// fallback pakai low
			var err error
			best, err = encodeQ(cur, low)
			if err != nil { return nil, err }
		}
		last = best

		// Sudah pas?
		if len(best) <= target+tol {
			return best, nil
		}

		// Belum pas → perkecil dimensi lagi (iterative downscale)
		b := cur.Bounds()
		cw, ch := b.Dx(), b.Dy()
		if cw <= minW && ch <= minH {
			// Udah mentok minimum — kembalikan hasil terakhir terbaik
			return best, nil
		}

		// Estimasi skala: gunakan sqrt rasio target/actual, lalu kalikan safety 0.95
		ratio := float64(target+tol) / float64(len(best))
		scale := math.Sqrt(ratio) * 0.95
		if scale >= 1.0 {
			scale = float64(step) // fallback kecilin sedikit
		}
		// clamp ke step maksimum (jangan terlalu agresif turun sekaligus)
		if scale > float64(step) {
			scale = float64(step)
		} else if scale < 0.5 {
			scale = 0.5
		}

		nw := int(math.Round(float64(cw) * scale))
		nh := int(math.Round(float64(ch) * scale))
		if nw < minW { nw = minW }
		if nh < minH { nh = minH }
		if nw >= cw && nh >= ch {
			// tidak mengecil → paksa step
			nw = int(float64(cw) * float64(step))
			nh = int(float64(ch) * float64(step))
			if nw < minW { nw = minW }
			if nh < minH { nh = minH }
		}

		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), cur, b, draw.Over, nil)
		cur = dst
	}

	// Fallback: kembalikan hasil terakhir
	return last, nil
}

/* =======================================================================
   API utama untuk re-encode: ConvertToWebP / ConvertToWebPWithOptions
======================================================================= */

// ConvertToWebP (kompat lama): pakai opsi dari ENV (resize + Q/target)
func ConvertToWebP(file multipart.File, filename string) ([]byte, error) {
	opts := defaultWebPOptionsFromEnv()
	return ConvertToWebPWithOptions(file, filename, opts)
}

// ConvertToWebPWithOptions: baca → decode → resize (opsional) → encode webp
func ConvertToWebPWithOptions(file multipart.File, filename string, opts WebPOptions) ([]byte, error) {
	all, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	img, err := decodeImage(all, filename)
	if err != nil {
		return nil, err
	}

	img = downscaleIfNeeded(img, opts.MaxW, opts.MaxH)
	return encodeToWebP(img, opts)
}

/* =======================================================================
   OSS Service (tetap kompat)
======================================================================= */

type OSSService struct {
	Client     *oss.Client
	Bucket     *oss.Bucket
	Endpoint   string
	BucketName string
	Prefix     string // optional: "uploads/"
}

func NewOSSServiceFromEnv(prefix string) (*OSSService, error) {
	endpoint := getEnv("ALI_OSS_ENDPOINT")
	ak := getEnv("ALI_OSS_ACCESS_KEY")
	sk := getEnv("ALI_OSS_SECRET_KEY")
	sts := getEnv("ALI_OSS_SECURITY_TOKEN")
	bucketName := getEnv("ALI_OSS_BUCKET")
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

	// Verifikasi ringan lokasi bucket
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

/* =======================================================================
   Upload helpers
======================================================================= */

// UploadAsWebP: kompat lama → gunakan opsi dari ENV
func (s *OSSService) UploadAsWebP(ctx context.Context, fh *multipart.FileHeader, keyPrefix string) (string, error) {
	return s.UploadAsWebPWithOptions(ctx, fh, keyPrefix, defaultWebPOptionsFromEnv())
}

// UploadAsWebPWithOptions: recompress ke webp sesuai opsi, lalu upload .webp
func (s *OSSService) UploadAsWebPWithOptions(ctx context.Context, fh *multipart.FileHeader, keyPrefix string, opt WebPOptions) (string, error) {
	if fh == nil {
		return "", fmt.Errorf("nil file header")
	}
	if fh.Size > maxUploadSize {
		return "", fmt.Errorf("file too large (max %d bytes)", maxUploadSize)
	}

	src, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer src.Close()

	webpData, err := ConvertToWebPWithOptions(src, fh.Filename, opt)
	if err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "format tidak didukung") {
			return "", fiber.NewError(fiber.StatusUnsupportedMediaType, "Unsupported image format (pakai jpg/png/webp)")
		}
		return "", err
	}

	// ganti ekstensi jadi .webp
	base := strings.TrimSuffix(fh.Filename, filepath.Ext(fh.Filename))
	key := s.buildObjectKey(base + ".webp")
	if keyPrefix != "" {
		key = strings.Trim(keyPrefix, "/") + "/" + key
	}

	opts := []oss.Option{
		oss.WithContext(ctx),
		oss.ContentType("image/webp"),
		oss.ContentDisposition("inline"),
		oss.CacheControl("public, max-age=31536000, immutable"),
	}
	if err := s.Bucket.PutObject(key, bytes.NewReader(webpData), opts...); err != nil {
		return "", err
	}
	return s.PublicURL(key), nil
}

// UploadFromFormFile: upload apa adanya (tanpa recompress)
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

/* =======================================================================
   Update & Delete
======================================================================= */

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

/* =======================================================================
   Public URL & Key utils
======================================================================= */

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

func ExtractKeyFromPublicURL(publicURL string) (string, error) {
	if publicURL == "" {
		return "", fmt.Errorf("empty url")
	}
	if base := strings.TrimSpace(os.Getenv("ALI_OSS_PUBLIC_BASE")); base != "" {
		base = strings.TrimRight(base, "/") + "/"
		if strings.HasPrefix(publicURL, base) {
			return strings.TrimPrefix(publicURL, base), nil
		}
	}
	u := publicURL
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if i := strings.Index(u, "/"); i >= 0 {
		return u[i+1:], nil
	}
	return "", fmt.Errorf("cannot extract key from url: %s", publicURL)
}

/* =======================================================================
   Misc utils
======================================================================= */

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

// detectContentType: tentukan contentType dari ekstensi + sniff 512B, lalu hard-override utk format modern
func detectContentType(src multipart.File, filename string) (string, io.Reader, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	ct := mime.TypeByExtension(ext)

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

	if seeker, ok := src.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
		return ct, src, nil
	}
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

/* =======================================================================
   Path helpers & convenience wrappers (kompat lama)
======================================================================= */

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

// UploadFromFormFileToDir: upload apa adanya (tanpa recompress) ke subdir
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

// UploadImageToOSSScoped: helper praktis ke "masjids/{masjid_id}/{kategori}" (tanpa recompress)
func UploadImageToOSSScoped(masjidID uuid.UUID, kategori string, fh *multipart.FileHeader) (string, error) {
	if masjidID == uuid.Nil {
		return "", fmt.Errorf("masjidID kosong/invalid")
	}
	if strings.TrimSpace(kategori) == "" {
		kategori = "misc"
	}

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

// UploadImageToOSS: helper lama — selalu convert ke WebP
func UploadImageToOSS(ctx context.Context, svc *OSSService, masjidID uuid.UUID, slot string, fh *multipart.FileHeader) (string, error) {
	if fh == nil {
		return "", fiber.NewError(fiber.StatusBadRequest, "File tidak ditemukan")
	}
	if masjidID == uuid.Nil {
		return "", fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak valid")
	}
	if fh.Size > maxUploadSize {
		return "", fiber.NewError(fiber.StatusRequestEntityTooLarge, "Ukuran gambar maksimal 5MB")
	}

	slot = strings.Trim(strings.ToLower(strings.TrimSpace(slot)), "/")
	if slot == "" {
		slot = "default"
	}
	dir := fmt.Sprintf("masjids/%s/images/%s", masjidID.String(), slot)

	url, err := svc.UploadAsWebP(ctx, fh, dir)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "format tidak didukung") {
			return "", fiber.NewError(fiber.StatusUnsupportedMediaType, "Unsupported image format (pakai jpg/png/webp)")
		}
		return "", fiber.NewError(fiber.StatusBadGateway, "Gagal upload ke OSS")
	}
	return url, nil
}

/* =======================================================================
   Delete helpers by public URL
======================================================================= */

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

func (s *OSSService) DeleteManyByPublicURL(ctx context.Context, publicURLs []string) (deleted []string, failed map[string]error) {
	failed = make(map[string]error)
	if len(publicURLs) == 0 {
		return nil, failed
	}

	type item struct{ url, key string }
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

		if _, err := s.Bucket.DeleteObjects(keys, oss.WithContext(context.Background())); err != nil {
			for _, k := range keys {
				u := urlByKey[k]
				failed[u] = fmt.Errorf("delete: %w", err)
			}
			continue
		}
		for _, k := range keys {
			deleted = append(deleted, urlByKey[k])
		}
	}
	return deleted, failed
}

// Convenient wrappers pakai ENV
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

func init() {
	_ = mime.AddExtensionType(".webp", "image/webp")
	_ = mime.AddExtensionType(".avif", "image/avif")
	_ = mime.AddExtensionType(".svg", "image/svg+xml")
}
