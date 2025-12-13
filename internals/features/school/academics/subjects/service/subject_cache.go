package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrSchoolMismatch = errors.New("school mismatch")

/* =========================================================
   SUBJECT CACHE STRUCT
   Sinkron dengan field cache di:
   - class_subjects:
     - class_subject_subject_name_cache
     - class_subject_subject_code_cache
     - class_subject_subject_slug_cache
     - class_subject_subject_url_cache

   - class_subject_books (opsional, sesuai kolom kamu):
     - class_subject_book_subject_name_cache
     - class_subject_book_subject_code_cache
     - class_subject_book_subject_slug_cache
     - class_subject_book_subject_url_cache

   - class_section_subject_teachers (CSST):
     - csst_subject_name_cache
     - csst_subject_code_cache
     - csst_subject_slug_cache
     (opsional kalau ada: csst_subject_url_cache)
========================================================= */

type SubjectCache struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Code string    `json:"code"`
	Slug string    `json:"slug"`
	URL  *string   `json:"url,omitempty"` // dari subjects.subject_image_url (atau sumber lain)
}

/* =========================================================
   BUILD SUBJECT CACHE (tenant-safe)
========================================================= */

func BuildSubjectCache(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	subjectID uuid.UUID,
) (*SubjectCache, error) {
	if tx == nil {
		return nil, nil
	}

	var row struct {
		SchoolID uuid.UUID
		ID       uuid.UUID
		Name     string
		Code     string
		Slug     string
		URL      *string
	}

	if err := tx.WithContext(ctx).Raw(`
		SELECT
			s.subject_school_id AS school_id,
			s.subject_id        AS id,
			s.subject_name      AS name,
			s.subject_code      AS code,
			s.subject_slug      AS slug,
			s.subject_image_url AS url
		FROM subjects s
		WHERE s.subject_id = ?
		  AND s.subject_deleted_at IS NULL
	`, subjectID).Scan(&row).Error; err != nil {
		return nil, err
	}

	// not found
	if row.SchoolID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	// tenant check
	if row.SchoolID != schoolID {
		return nil, ErrSchoolMismatch
	}

	return &SubjectCache{
		ID:   row.ID,
		Name: strings.TrimSpace(row.Name),
		Code: strings.TrimSpace(row.Code),
		Slug: strings.TrimSpace(row.Slug),
		URL:  nzPtr(row.URL),
	}, nil
}

/* =========================================================
   OPTIONAL: Build cache dari values (tanpa query)
========================================================= */

func BuildSubjectCacheFromValues(
	subjectID uuid.UUID,
	name string,
	code string,
	slug string,
	url *string,
) *SubjectCache {
	return &SubjectCache{
		ID:   subjectID,
		Name: strings.TrimSpace(name),
		Code: strings.TrimSpace(code),
		Slug: strings.TrimSpace(slug),
		URL:  nzPtr(url),
	}
}

/* =========================================================
   Internal helpers
========================================================= */

func trimOrNil(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}

func nzPtr(p *string) *string {
	if p == nil {
		return nil
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return nil
	}
	return &t
}

/* =========================================================
   SYNC A) subject cache -> class_subjects
========================================================= */

func SyncClassSubjectsSubjectCacheFromCache(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	cache *SubjectCache,
) error {
	if tx == nil || cache == nil {
		return nil
	}

	snapName := trimOrNil(cache.Name)
	snapCode := trimOrNil(cache.Code)
	snapSlug := trimOrNil(cache.Slug)
	snapURL := nzPtr(cache.URL)

	patch := map[string]any{
		"class_subject_subject_name_cache": func() any {
			if snapName == nil {
				return gorm.Expr("NULL")
			}
			return *snapName
		}(),
		"class_subject_subject_code_cache": func() any {
			if snapCode == nil {
				return gorm.Expr("NULL")
			}
			return *snapCode
		}(),
		"class_subject_subject_slug_cache": func() any {
			if snapSlug == nil {
				return gorm.Expr("NULL")
			}
			return *snapSlug
		}(),
		"class_subject_subject_url_cache": func() any {
			if snapURL == nil {
				return gorm.Expr("NULL")
			}
			return *snapURL
		}(),
		"class_subject_updated_at": time.Now(), // opsional
	}

	return tx.WithContext(ctx).
		Table("class_subjects").
		Where(`
			class_subject_subject_id = ?
			AND class_subject_school_id = ?
			AND class_subject_deleted_at IS NULL
		`, cache.ID, schoolID).
		Updates(patch).Error
}

/* =========================================================
   SYNC B) subject cache -> class_subject_books
   NOTE: ganti nama kolom kalau beda di DB kamu
========================================================= */

func SyncClassSubjectBooksSubjectCacheFromCache(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	cache *SubjectCache,
) error {
	if tx == nil || cache == nil {
		return nil
	}

	snapName := trimOrNil(cache.Name)
	snapCode := trimOrNil(cache.Code)
	snapSlug := trimOrNil(cache.Slug)
	snapURL := nzPtr(cache.URL)

	patch := map[string]any{
		"class_subject_book_subject_name_cache": func() any {
			if snapName == nil {
				return gorm.Expr("NULL")
			}
			return *snapName
		}(),
		"class_subject_book_subject_code_cache": func() any {
			if snapCode == nil {
				return gorm.Expr("NULL")
			}
			return *snapCode
		}(),
		"class_subject_book_subject_slug_cache": func() any {
			if snapSlug == nil {
				return gorm.Expr("NULL")
			}
			return *snapSlug
		}(),
		"class_subject_book_subject_url_cache": func() any {
			if snapURL == nil {
				return gorm.Expr("NULL")
			}
			return *snapURL
		}(),

		// hapus kalau tabel kamu tidak punya updated_at
		"class_subject_book_updated_at": time.Now(),
	}

	return tx.WithContext(ctx).
		Table("class_subject_books").
		Where(`
			class_subject_book_school_id = ?
			AND class_subject_book_subject_id = ?
			AND class_subject_book_deleted_at IS NULL
		`, schoolID, cache.ID).
		Updates(patch).Error
}

/* =========================================================
   SYNC C) subject cache -> class_section_subject_teachers (CSST)
   Sesuai model CSST kamu:
   - csst_subject_name_cache
   - csst_subject_code_cache
   - csst_subject_slug_cache

   Update baris CSST yang:
   - csst_school_id = schoolID
   - csst_subject_id = subjectID
   - csst_deleted_at IS NULL
========================================================= */

func SyncCSSTSubjectCacheFromCache(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	cache *SubjectCache,
) error {
	if tx == nil || cache == nil {
		return nil
	}

	snapName := trimOrNil(cache.Name)
	snapCode := trimOrNil(cache.Code)
	snapSlug := trimOrNil(cache.Slug)

	patch := map[string]any{
		"csst_subject_name_cache": func() any {
			if snapName == nil {
				return gorm.Expr("NULL")
			}
			return *snapName
		}(),
		"csst_subject_code_cache": func() any {
			if snapCode == nil {
				return gorm.Expr("NULL")
			}
			return *snapCode
		}(),
		"csst_subject_slug_cache": func() any {
			if snapSlug == nil {
				return gorm.Expr("NULL")
			}
			return *snapSlug
		}(),

		// kalau kamu nanti bikin kolom URL di CSST:
		// "csst_subject_url_cache": func() any {
		// 	snapURL := nzPtr(cache.URL)
		// 	if snapURL == nil {
		// 		return gorm.Expr("NULL")
		// 	}
		// 	return *snapURL
		// }(),

		"csst_updated_at": time.Now(), // opsional (kolom kamu ada)
	}

	return tx.WithContext(ctx).
		Table("class_section_subject_teachers").
		Where(`
			csst_school_id = ?
			AND csst_subject_id = ?
			AND csst_deleted_at IS NULL
		`, schoolID, cache.ID).
		Updates(patch).Error
}

/* =========================================================
   WRAPPER: sekali panggil, sync ke semua turunan
========================================================= */

func SyncSubjectCachesEverywhereFromCache(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	cache *SubjectCache,
) error {
	if tx == nil || cache == nil {
		return nil
	}

	// 1) class_subjects
	if err := SyncClassSubjectsSubjectCacheFromCache(ctx, tx, schoolID, cache); err != nil {
		return err
	}

	// 2) class_subject_books
	if err := SyncClassSubjectBooksSubjectCacheFromCache(ctx, tx, schoolID, cache); err != nil {
		return err
	}

	// 3) CSST
	if err := SyncCSSTSubjectCacheFromCache(ctx, tx, schoolID, cache); err != nil {
		return err
	}

	return nil
}

/* =========================================================
   WRAPPER: controller 1-liner dari subject_id
========================================================= */

func SyncSubjectCachesEverywhereFromSubjectID(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	subjectID uuid.UUID,
) error {
	cache, err := BuildSubjectCache(ctx, tx, schoolID, subjectID)
	if err != nil {
		return err
	}
	return SyncSubjectCachesEverywhereFromCache(ctx, tx, schoolID, cache)
}
