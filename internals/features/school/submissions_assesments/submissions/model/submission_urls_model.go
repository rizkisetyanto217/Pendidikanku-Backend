// file: internals/features/submissions/submission_urls/model/submission_url.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
  =========================================================
  GORM Model — submission_urls
  - Selaras dgn DDL
  - Soft delete via gorm.DeletedAt pada kolom custom
  - Scopes umum untuk query
  - EnsureSubmissionURLIndexes() membuat partial & unique indexes
  =========================================================
*/

type SubmissionURL struct {
	// PK
	SubmissionURLID uuid.UUID `gorm:"column:submission_url_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// Tenant & owner
	SubmissionURLMasjidID     uuid.UUID `gorm:"column:submission_url_masjid_id;type:uuid;not null"`
	SubmissionURLSubmissionID uuid.UUID `gorm:"column:submission_url_submission_id;type:uuid;not null"`

	// Jenis/peran aset
	SubmissionURLKind string `gorm:"column:submission_url_kind;type:varchar(24);not null"`

	// Lokasi file/link
	SubmissionURLHref         *string `gorm:"column:submission_url_href;type:text"`
	SubmissionURLObjectKey    *string `gorm:"column:submission_url_object_key;type:text"`
	SubmissionURLObjectKeyOld *string `gorm:"column:submission_url_object_key_old;type:text"`

	// Tampilan
	SubmissionURLLabel     *string `gorm:"column:submission_url_label;type:varchar(160)"`
	SubmissionURLOrder     int32   `gorm:"column:submission_url_order;type:int;not null;default:0"`
	SubmissionURLIsPrimary bool    `gorm:"column:submission_url_is_primary;type:boolean;not null;default:false"`

	// Audit & retensi
	SubmissionURLCreatedAt          time.Time      `gorm:"column:submission_url_created_at;type:timestamptz;not null;default:now()"`
	SubmissionURLUpdatedAt          time.Time      `gorm:"column:submission_url_updated_at;type:timestamptz;not null;default:now()"`
	SubmissionURLDeletedAt          gorm.DeletedAt `gorm:"column:submission_url_deleted_at;type:timestamptz;index"`
	SubmissionURLDeletePendingUntil *time.Time     `gorm:"column:submission_url_delete_pending_until;type:timestamptz"`
}

// TableName explicit
func (SubmissionURL) TableName() string {
	return "submission_urls"
}

/* =========================================================
   Scopes umum
========================================================= */

// Baris hidup (belum soft-delete)
func ScopeSubURLLive(db *gorm.DB) *gorm.DB {
	return db.Where("submission_url_deleted_at IS NULL")
}

// Filter per masjid
func ScopeSubURLByMasjid(masjidID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("submission_url_masjid_id = ?", masjidID)
	}
}

// Filter per submission
func ScopeSubURLBySubmission(submissionID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("submission_url_submission_id = ?", submissionID)
	}
}

// Filter per kind
func ScopeSubURLByKind(kind string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("submission_url_kind = ?", kind)
	}
}

// Hanya primary
func ScopeSubURLPrimary(db *gorm.DB) *gorm.DB {
	return db.Where("submission_url_is_primary = TRUE")
}

// Urutan tampil default
func ScopeSubURLDefaultOrder(db *gorm.DB) *gorm.DB {
	return db.Order("submission_url_is_primary DESC").
		Order("submission_url_order ASC").
		Order("submission_url_created_at ASC")
}

/* =========================================================
   Migrations: ensure partial & unique indexes (WHERE ...)
   Pakai Raw SQL (GORM belum full support partial index via tag).
   Idempotent: IF NOT EXISTS.
========================================================= */

func EnsureSubmissionURLIndexes(db *gorm.DB) error {
	sqls := []string{
		// Lookup per submission (live) + urutan tampil
		`CREATE INDEX IF NOT EXISTS ix_sub_urls_by_owner_live
		   ON submission_urls (
		     submission_url_submission_id,
		     submission_url_kind,
		     submission_url_is_primary DESC,
		     submission_url_order,
		     submission_url_created_at
		   )
		   WHERE submission_url_deleted_at IS NULL;`,

		// Filter per tenant (live)
		`CREATE INDEX IF NOT EXISTS ix_sub_urls_by_masjid_live
		   ON submission_urls (submission_url_masjid_id)
		   WHERE submission_url_deleted_at IS NULL;`,

		// Satu primary per (submission, kind) (live only)
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_sub_urls_primary_per_kind_alive
		   ON submission_urls (submission_url_submission_id, submission_url_kind)
		   WHERE submission_url_deleted_at IS NULL
		     AND submission_url_is_primary = TRUE;`,

		// Anti-duplikat href per submission (live only)
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_sub_urls_submission_href_alive
		   ON submission_urls (submission_url_submission_id, submission_url_href)
		   WHERE submission_url_deleted_at IS NULL
		     AND submission_url_href IS NOT NULL;`,

		// Kandidat purge (aktif dgn *_old) atau soft-deleted yg punya object_key
		`CREATE INDEX IF NOT EXISTS ix_sub_urls_purge_due
		   ON submission_urls (submission_url_delete_pending_until)
		   WHERE submission_url_delete_pending_until IS NOT NULL
		     AND (
		       (submission_url_deleted_at IS NULL  AND submission_url_object_key_old IS NOT NULL) OR
		       (submission_url_deleted_at IS NOT NULL AND submission_url_object_key     IS NOT NULL)
		     );`,

		// (opsional) trigram label live — uncomment jika extension tersedia
		// `CREATE EXTENSION IF NOT EXISTS pg_trgm;`,
		// `CREATE INDEX IF NOT EXISTS gin_sub_urls_label_trgm_live
		//    ON submission_urls USING GIN (submission_url_label gin_trgm_ops)
		//    WHERE submission_url_deleted_at IS NULL;`,
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, q := range sqls {
			if err := tx.Exec(q).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

/* =========================================================
   Optional: constants untuk Kind agar konsisten
========================================================= */

type SubURLKind string

const (
	SubURLKindImage      SubURLKind = "image"
	SubURLKindVideo      SubURLKind = "video"
	SubURLKindAttachment SubURLKind = "attachment"
	SubURLKindLink       SubURLKind = "link"
	SubURLKindAudio      SubURLKind = "audio"
)

/* =========================================================
   (Opsional) Getter ringan — berguna jika DTO kamu pakai interface
========================================================= */

func (m *SubmissionURL) GetID() uuid.UUID           { return m.SubmissionURLID }
func (m *SubmissionURL) GetMasjidID() uuid.UUID     { return m.SubmissionURLMasjidID }
func (m *SubmissionURL) GetSubmissionID() uuid.UUID { return m.SubmissionURLSubmissionID }
func (m *SubmissionURL) GetKind() string            { return m.SubmissionURLKind }
func (m *SubmissionURL) GetHref() *string           { return m.SubmissionURLHref }
func (m *SubmissionURL) GetObjectKey() *string      { return m.SubmissionURLObjectKey }
func (m *SubmissionURL) GetObjectKeyOld() *string   { return m.SubmissionURLObjectKeyOld }
func (m *SubmissionURL) GetLabel() *string          { return m.SubmissionURLLabel }
func (m *SubmissionURL) GetOrder() int32            { return m.SubmissionURLOrder }
func (m *SubmissionURL) GetIsPrimary() bool         { return m.SubmissionURLIsPrimary }
func (m *SubmissionURL) GetCreatedAt() time.Time    { return m.SubmissionURLCreatedAt }
func (m *SubmissionURL) GetUpdatedAt() time.Time    { return m.SubmissionURLUpdatedAt }
func (m *SubmissionURL) GetDeletedAtPtr() *time.Time {
	if !m.SubmissionURLDeletedAt.Valid {
		return nil
	}
	t := m.SubmissionURLDeletedAt.Time
	return &t
}
