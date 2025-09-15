// storage/size.go
package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantURL struct {
	ID       uuid.UUID `gorm:"column:url_id"`
	Href     string    `gorm:"column:url_href"`
	FileSize *int64    `gorm:"column:url_file_size_bytes"`
}

func ExtractBucketAndKeyFromHref(href string) (bucket, key string, ok bool) {
	if href == "" { return "", "", false }
	// strip query
	if i := strings.IndexByte(href, '?'); i >= 0 { href = href[:i] }
	rest := strings.TrimPrefix(strings.TrimPrefix(href, "https://"), "http://")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 { return "", "", false }
	host, keyPart := parts[0], parts[1]

	// virtual-host: bucket.xxx.aliyuncs.com/key
	hostParts := strings.Split(host, ".")
	if len(hostParts) >= 4 && strings.Contains(host, "aliyuncs.com") {
		return hostParts[0], keyPart, true
	}
	// path-style: oss-region.aliyuncs.com/bucket/key
	if strings.Contains(host, "aliyuncs.com") {
		if slash := strings.IndexByte(keyPart, '/'); slash > 0 {
			return keyPart[:slash], keyPart[slash+1:], true
		}
	}
	return "", "", false
}

func HeadObjectSize(ossClient *oss.Client, bucket, key string) (int64, error) {
	bkt, err := ossClient.Bucket(bucket)
	if err != nil { return 0, err }

	hdr, err := bkt.GetObjectDetailedMeta(key) // HEAD
	if err != nil { return 0, err }

	// Prefer constant name from SDK; fallback ke literal jika perlu
	cl := hdr.Get(oss.HTTPHeaderContentLength)
	if cl == "" {
		cl = hdr.Get("Content-Length")
	}
	if cl == "" {
		return 0, fmt.Errorf("Content-Length header missing")
	}

	n, err := strconv.ParseInt(cl, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse Content-Length: %w", err)
	}
	return n, nil
}

// EnsureURLSizeByID: kalau belum ada size -> HEAD OSS -> update DB -> return bytes
func EnsureURLSizeByID(db *gorm.DB, ossClient *oss.Client, urlID uuid.UUID, defaultBucket string) (int64, error) {
	var row TenantURL
	if err := db.Raw(`
		SELECT url_id, url_href, url_file_size_bytes
		FROM tenant_entity_urls
		WHERE url_id = ? AND url_deleted_at IS NULL
	`, urlID).Scan(&row).Error; err != nil {
		return 0, fmt.Errorf("db read: %w", err)
	}
	if row.ID == uuid.Nil {
		return 0, fmt.Errorf("url not found")
	}
	if row.FileSize != nil && *row.FileSize > 0 {
		return *row.FileSize, nil
	}

	bucket, key, ok := ExtractBucketAndKeyFromHref(row.Href)
	if !ok {
		if defaultBucket == "" {
			return 0, fmt.Errorf("cannot parse bucket/key from href and defaultBucket empty")
		}
		// NOTE: Kalau kamu simpan object key di DB (disarankan), ambil dari sana.
		return 0, fmt.Errorf("object key unknown; store it or pass it")
	}

	size, err := HeadObjectSize(ossClient, bucket, key)
	if err != nil {
		return 0, fmt.Errorf("oss HEAD: %w", err)
	}

	if err := db.Exec(`
		UPDATE tenant_entity_urls
		SET url_file_size_bytes = ?, url_updated_at = NOW()
		WHERE url_id = ?`, size, urlID).Error; err != nil {
		return 0, fmt.Errorf("db update: %w", err)
	}
	return size, nil
}
