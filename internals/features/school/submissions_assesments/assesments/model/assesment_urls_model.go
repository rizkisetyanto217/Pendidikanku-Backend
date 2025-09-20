// file: internals/features/assessments/assessment_urls/model/assessment_url.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================
 GORM Model — assessment_urls
 - Selaras DDL
 - Soft delete via gorm.DeletedAt dengan kolom custom
 - Scopes umum
 - EnsureAssessmentURLIndexes untuk partial/unique index
=========================================================
*/

type AssessmentURL struct {
	// PK
	AssessmentURLID uuid.UUID `gorm:"column:assessment_url_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// Tenant & owner
	AssessmentURLMasjidID   uuid.UUID `gorm:"column:assessment_url_masjid_id;type:uuid;not null"`
	AssessmentURLAssessment uuid.UUID `gorm:"column:assessment_url_assessment_id;type:uuid;not null"`

	// Jenis/peran aset
	AssessmentURLKind string `gorm:"column:assessment_url_kind;type:varchar(24);not null"`

	// Lokasi file/link
	AssessmentURLHref         *string `gorm:"column:assessment_url_href;type:text"`
	AssessmentURLObjectKey    *string `gorm:"column:assessment_url_object_key;type:text"`
	AssessmentURLObjectKeyOld *string `gorm:"column:assessment_url_object_key_old;type:text"`

	// Tampilan
	AssessmentURLLabel     *string `gorm:"column:assessment_url_label;type:varchar(160)"`
	AssessmentURLOrder     int32   `gorm:"column:assessment_url_order;type:int;not null;default:0"`
	AssessmentURLIsPrimary bool    `gorm:"column:assessment_url_is_primary;type:boolean;not null;default:false"`

	// Audit & retensi
	AssessmentURLCreatedAt          time.Time      `gorm:"column:assessment_url_created_at;type:timestamptz;not null;default:now()"`
	AssessmentURLUpdatedAt          time.Time      `gorm:"column:assessment_url_updated_at;type:timestamptz;not null;default:now()"`
	AssessmentURLDeletedAt          gorm.DeletedAt `gorm:"column:assessment_url_deleted_at;type:timestamptz;index"`
	AssessmentURLDeletePendingUntil *time.Time     `gorm:"column:assessment_url_delete_pending_until;type:timestamptz"`
}

func (AssessmentURL) TableName() string { return "assessment_urls" }

/* =========================================================
   Scopes (chainable)
========================================================= */

func ScopeAssURLLive(db *gorm.DB) *gorm.DB {
	return db.Where("assessment_url_deleted_at IS NULL")
}

func ScopeAssURLByMasjid(masjidID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("assessment_url_masjid_id = ?", masjidID)
	}
}

func ScopeAssURLByAssessment(assessmentID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("assessment_url_assessment_id = ?", assessmentID)
	}
}

func ScopeAssURLByKind(kind string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("assessment_url_kind = ?", kind)
	}
}

// Urutan default: primary dulu, lalu order, lalu created_at
func ScopeAssURLDefaultOrder(db *gorm.DB) *gorm.DB {
	return db.Order("assessment_url_is_primary DESC").
		Order("assessment_url_order ASC").
		Order("assessment_url_created_at ASC")
}

/* =========================================================
   Ensure indexes (idempotent) — partial & unique sesuai DDL
========================================================= */

func EnsureAssessmentURLIndexes(db *gorm.DB) error {
	sqls := []string{
		// Lookup per assessment (live) + urutan tampil
		`CREATE INDEX IF NOT EXISTS ix_ass_urls_by_owner_live
		   ON assessment_urls (
		     assessment_url_assessment_id,
		     assessment_url_kind,
		     assessment_url_is_primary DESC,
		     assessment_url_order,
		     assessment_url_created_at
		   )
		   WHERE assessment_url_deleted_at IS NULL;`,

		// Filter per tenant (live)
		`CREATE INDEX IF NOT EXISTS ix_ass_urls_by_masjid_live
		   ON assessment_urls (assessment_url_masjid_id)
		   WHERE assessment_url_deleted_at IS NULL;`,

		// Satu primary per (assessment, kind) (live)
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_ass_urls_primary_per_kind_alive
		   ON assessment_urls (assessment_url_assessment_id, assessment_url_kind)
		   WHERE assessment_url_deleted_at IS NULL
		     AND assessment_url_is_primary = TRUE;`,

		// Anti-duplikat href per assessment (live)
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_ass_urls_assessment_href_alive
		   ON assessment_urls (assessment_url_assessment_id, assessment_url_href)
		   WHERE assessment_url_deleted_at IS NULL
		     AND assessment_url_href IS NOT NULL;`,

		// Kandidat purge (aktif punya *_old, atau soft-deleted punya object_key)
		`CREATE INDEX IF NOT EXISTS ix_ass_urls_purge_due
		   ON assessment_urls (assessment_url_delete_pending_until)
		   WHERE assessment_url_delete_pending_until IS NOT NULL
		     AND (
		       (assessment_url_deleted_at IS NULL  AND assessment_url_object_key_old IS NOT NULL) OR
		       (assessment_url_deleted_at IS NOT NULL AND assessment_url_object_key     IS NOT NULL)
		     );`,

		// (opsional) BRIN untuk time-scan cepat
		`CREATE INDEX IF NOT EXISTS brin_ass_urls_created_at
		   ON assessment_urls USING BRIN (assessment_url_created_at);`,
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
   Constants Kind (biar konsisten di seluruh layer)
========================================================= */

type AssURLKind string

const (
	AssURLKindImage      AssURLKind = "image"
	AssURLKindVideo      AssURLKind = "video"
	AssURLKindAttachment AssURLKind = "attachment"
	AssURLKindLink       AssURLKind = "link"
	AssURLKindAudio      AssURLKind = "audio"
)

/* =========================================================
   Getters (memudahkan mapping ke DTO tanpa import silang)
========================================================= */

func (m *AssessmentURL) GetID() uuid.UUID           { return m.AssessmentURLID }
func (m *AssessmentURL) GetMasjidID() uuid.UUID     { return m.AssessmentURLMasjidID }
func (m *AssessmentURL) GetAssessmentID() uuid.UUID { return m.AssessmentURLAssessment }
func (m *AssessmentURL) GetKind() string            { return m.AssessmentURLKind }
func (m *AssessmentURL) GetHref() *string           { return m.AssessmentURLHref }
func (m *AssessmentURL) GetObjectKey() *string      { return m.AssessmentURLObjectKey }
func (m *AssessmentURL) GetObjectKeyOld() *string   { return m.AssessmentURLObjectKeyOld }
func (m *AssessmentURL) GetLabel() *string          { return m.AssessmentURLLabel }
func (m *AssessmentURL) GetOrder() int32            { return m.AssessmentURLOrder }
func (m *AssessmentURL) GetIsPrimary() bool         { return m.AssessmentURLIsPrimary }
func (m *AssessmentURL) GetCreatedAt() time.Time    { return m.AssessmentURLCreatedAt }
func (m *AssessmentURL) GetUpdatedAt() time.Time    { return m.AssessmentURLUpdatedAt }
func (m *AssessmentURL) GetDeletedAtPtr() *time.Time {
	if m.AssessmentURLDeletedAt.Valid {
		t := m.AssessmentURLDeletedAt.Time
		return &t
	}
	return nil
}
