// file: internals/features/school/subjects/model/subject_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubjectModel struct {
	// PK & tenant
	SubjectID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:subject_id"         json:"subject_id"`
	SubjectMasjidID  uuid.UUID `gorm:"type:uuid;not null;column:subject_masjid_id"                              json:"subject_masjid_id"`

	// Identitas
	SubjectCode string  `gorm:"type:varchar(40);not null;column:subject_code"     json:"subject_code"`
	SubjectName string  `gorm:"type:varchar(120);not null;column:subject_name"    json:"subject_name"`
	SubjectDesc *string `gorm:"type:text;column:subject_desc"                     json:"subject_desc,omitempty"`
	SubjectSlug string  `gorm:"type:varchar(160);not null;column:subject_slug"    json:"subject_slug"`

	// Single image (2-slot + retensi 30 hari)
	SubjectImageURL                   *string    `gorm:"type:text;column:subject_image_url"                    json:"subject_image_url,omitempty"`
	SubjectImageObjectKey             *string    `gorm:"type:text;column:subject_image_object_key"             json:"subject_image_object_key,omitempty"`
	SubjectImageURLOld                *string    `gorm:"type:text;column:subject_image_url_old"                json:"subject_image_url_old,omitempty"`
	SubjectImageObjectKeyOld          *string    `gorm:"type:text;column:subject_image_object_key_old"         json:"subject_image_object_key_old,omitempty"`
	SubjectImageDeletePendingUntil    *time.Time `gorm:"type:timestamptz;column:subject_image_delete_pending_until" json:"subject_image_delete_pending_until,omitempty"`

	// Status & audit
	SubjectIsActive  bool           `gorm:"not null;default:true;column:subject_is_active"   json:"subject_is_active"`
	SubjectCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:subject_created_at" json:"subject_created_at"`
	SubjectUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:subject_updated_at" json:"subject_updated_at"`
	SubjectDeletedAt gorm.DeletedAt `gorm:"column:subject_deleted_at;index"                  json:"subject_deleted_at,omitempty"`
}

func (SubjectModel) TableName() string { return "subjects" }
