// file: internals/services/snapsvc/class_subject_cache.go
package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Struktur JSON yang akan disimpan ke kolom
// class_section_subject_teacher_class_subject_cache (JSONB)
type ClassSubjectCache struct {
	ID        uuid.UUID `json:"id"`
	SchoolID  uuid.UUID `json:"school_id"`
	ParentID  uuid.UUID `json:"parent_id"`
	SubjectID uuid.UUID `json:"subject_id"`

	// Bidang minimal untuk generated columns di CSST
	Name string  `json:"name"`
	Code string  `json:"code"`
	Slug string  `json:"slug"`
	URL  *string `json:"url,omitempty"`

	// Opsional tambahan
	ClassSubjectSlug *string `json:"class_subject_slug,omitempty"`
}

// BuildClassSubjectCache:
// - Validasi tenant & soft-delete pada class_subjects
// - Prefer ambil dari tabel subjects (jika masih ada), fallback ke cache di class_subjects
// - gorm.ErrRecordNotFound bila class_subject tidak ada
// - ErrSchoolMismatch bila tenant beda
func BuildClassSubjectCache(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	classSubjectID uuid.UUID,
) (*ClassSubjectCache, error) {
	var row struct {
		// dari class_subjects
		CSSchoolID       uuid.UUID
		CSID             uuid.UUID
		ParentID         uuid.UUID
		SubjectID        uuid.UUID
		ClassSubjectSlug *string

		// subject live
		SubjName string
		SubjCode string
		SubjSlug string
		SubjURL  *string

		// fallback cache di class_subjects
		SnapName string
		SnapCode string
		SnapSlug string
		SnapURL  *string
	}

	if err := tx.WithContext(ctx).Raw(`
		SELECT
			cs.class_subject_school_id  AS cs_school_id,
			cs.class_subject_id         AS cs_id,
			cs.class_subject_parent_id  AS parent_id,
			cs.class_subject_subject_id AS subject_id,
			cs.class_subject_slug       AS class_subject_slug,

			COALESCE(s.subject_name,  '') AS subj_name,
			COALESCE(s.subject_code,  '') AS subj_code,
			COALESCE(s.subject_slug,  '') AS subj_slug,
			s.subject_image_url            AS subj_url,

			COALESCE(cs.class_subject_subject_name_cache, '') AS snap_name,
			COALESCE(cs.class_subject_subject_code_cache, '') AS snap_code,
			COALESCE(cs.class_subject_subject_slug_cache, '') AS snap_slug,
			cs.class_subject_subject_url_cache               AS snap_url
		FROM class_subjects cs
		LEFT JOIN subjects s
		  ON s.subject_id = cs.class_subject_subject_id
		 AND s.subject_deleted_at IS NULL
		WHERE cs.class_subject_id = ?
		  AND cs.class_subject_deleted_at IS NULL
		LIMIT 1
	`, classSubjectID).Scan(&row).Error; err != nil {
		return nil, err
	}

	// not found
	if row.CSID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	// tenant check
	if row.CSSchoolID != schoolID {
		return nil, ErrSchoolMismatch
	}

	trim := func(s string) string { return strings.TrimSpace(s) }
	// deref *string lalu trim → string
	trimDeref := func(p *string) string {
		if p == nil {
			return ""
		}
		return strings.TrimSpace(*p)
	}
	// normalize pointer string (empty → nil)
	nzPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	// prefer data live dari subjects, fallback ke cache cs
	name := trim(row.SubjName)
	if name == "" {
		name = trim(row.SnapName)
	}
	code := trim(row.SubjCode)
	if code == "" {
		code = trim(row.SnapCode)
	}
	slug := trim(row.SubjSlug)
	if slug == "" {
		slug = trim(row.SnapSlug)
	}
	// URL: kalau dari subject kosong/null → ambil dari cache
	var url *string
	if trimDeref(row.SubjURL) != "" {
		url = row.SubjURL
	} else {
		url = row.SnapURL
	}
	url = nzPtr(url)

	out := &ClassSubjectCache{
		ID:               row.CSID,
		SchoolID:         row.CSSchoolID,
		ParentID:         row.ParentID,
		SubjectID:        row.SubjectID,
		Name:             name,
		Code:             code,
		Slug:             slug,
		URL:              url,
		ClassSubjectSlug: nzPtr(row.ClassSubjectSlug),
	}
	return out, nil
}

// BuildClassSubjectCacheJSON: langsung pulangkan datatypes.JSON tanpa bergantung ToJSONB eksternal.
func BuildClassSubjectCacheJSON(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	classSubjectID uuid.UUID,
) (datatypes.JSON, error) {
	snap, err := BuildClassSubjectCache(ctx, tx, schoolID, classSubjectID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		// secara teori tidak terjadi, tapi amankan
		return datatypes.JSON([]byte("null")), nil
	}
	b, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}
