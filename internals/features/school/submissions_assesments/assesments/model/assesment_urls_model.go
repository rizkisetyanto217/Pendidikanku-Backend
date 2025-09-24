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

type AssessmentURLModel struct {
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

func (AssessmentURLModel) TableName() string { return "assessment_urls" }
