package helper

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func normalizeEndpoint(ep string) string {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		return ep
	}
	if strings.HasPrefix(ep, "http://") || strings.HasPrefix(ep, "https://") {
		return ep
	}
	return "https://" + ep
}

func KeyFromPublicURL(publicURL string) (string, error) {
	u, err := url.Parse(publicURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	key := strings.TrimPrefix(u.Path, "/")
	if key == "" {
		return "", fmt.Errorf("empty key from URL")
	}
	return key, nil
}

func publicURLFromKey(key string) (string, error) {
	endpoint := normalizeEndpoint(os.Getenv("ALI_OSS_ENDPOINT"))
	bucket := os.Getenv("ALI_OSS_BUCKET")
	if endpoint == "" || bucket == "" {
		return "", fmt.Errorf("ALI_OSS_ENDPOINT/ALI_OSS_BUCKET belum di-set")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse endpoint: %w", err)
	}
	host := u.Host
	if host == "" {
		return "", fmt.Errorf("invalid endpoint host")
	}
	return fmt.Sprintf("https://%s.%s/%s", bucket, host, key), nil
}

// Pindahkan objek aktif -> spam/YYYY/MM/DD/HHMMSS__basename
// Return URL tujuan (spam).
func MoveToSpamByPublicURLENV(publicURL string, _ time.Duration) (string, error) {
	endpoint := normalizeEndpoint(os.Getenv("ALI_OSS_ENDPOINT"))
	ak := os.Getenv("ALI_OSS_ACCESS_KEY")
	sk := os.Getenv("ALI_OSS_SECRET_KEY")
	bucketName := os.Getenv("ALI_OSS_BUCKET")

	if endpoint == "" || ak == "" || sk == "" || bucketName == "" {
		return "", fmt.Errorf("ENV wajib: ALI_OSS_ENDPOINT, ALI_OSS_ACCESS_KEY, ALI_OSS_SECRET_KEY, ALI_OSS_BUCKET")
	}

	srcKey, err := KeyFromPublicURL(publicURL)
	if err != nil {
		return "", err
	}
	if srcKey == "" {
		return "", fmt.Errorf("empty src key")
	}

	client, err := oss.New(endpoint, ak, sk)
	if err != nil {
		return "", fmt.Errorf("oss.New: %w", err)
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return "", fmt.Errorf("client.Bucket: %w", err)
	}

	now := time.Now()
	base := path.Base(srcKey)
	dstKey := path.Join(
		"spam",
		now.Format("2006"), now.Format("01"), now.Format("02"),
		fmt.Sprintf("%s__%s", now.Format("150405"), base),
	)

	if _, err := bucket.CopyObject(srcKey, dstKey); err != nil {
		return "", fmt.Errorf("copy %q -> %q: %w", srcKey, dstKey, err)
	}
	_ = bucket.DeleteObject(srcKey) // best-effort

	dstURL, _ := publicURLFromKey(dstKey)
	return dstURL, nil
}
