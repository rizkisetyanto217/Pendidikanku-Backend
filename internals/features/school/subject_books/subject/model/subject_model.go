// internals/features/lembaga/subjects/main/model/subject_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NOTE:
// - subjects_slug: NOT NULL (string, bukan pointer) sesuai SQL
// - field-field image ditambahkan (nullable → pakai *string / *time.Time)
// - soft delete pakai gorm.DeletedAt (map ke TIMESTAMPTZ)
type SubjectsModel struct {
	SubjectsID       uuid.UUID `gorm:"column:subjects_id;type:uuid;default:gen_random_uuid();primaryKey" json:"subjects_id"`
	SubjectsMasjidID uuid.UUID `gorm:"column:subjects_masjid_id;type:uuid;not null;index" json:"subjects_masjid_id"`

	SubjectsCode string  `gorm:"column:subjects_code;type:varchar(40);not null"  json:"subjects_code"`
	SubjectsName string  `gorm:"column:subjects_name;type:varchar(120);not null" json:"subjects_name"`
	SubjectsDesc *string `gorm:"column:subjects_desc;type:text"                 json:"subjects_desc,omitempty"`

	// SQL: NOT NULL
	SubjectsSlug string `gorm:"column:subjects_slug;type:varchar(160);not null" json:"subjects_slug"`

	// Image (nullable)
	SubjectsImageURL                *string    `gorm:"column:subjects_image_url;type:text"                 json:"subjects_image_url,omitempty"`
	SubjectsImageObjectKey          *string    `gorm:"column:subjects_image_object_key;type:text"          json:"subjects_image_object_key,omitempty"`
	SubjectsImageURLOld             *string    `gorm:"column:subjects_image_url_old;type:text"             json:"subjects_image_url_old,omitempty"`
	SubjectsImageObjectKeyOld       *string    `gorm:"column:subjects_image_object_key_old;type:text"      json:"subjects_image_object_key_old,omitempty"`
	SubjectsImageDeletePendingUntil *time.Time `gorm:"column:subjects_image_delete_pending_until"          json:"subjects_image_delete_pending_until,omitempty"`

	SubjectsIsActive  bool           `gorm:"column:subjects_is_active;not null;default:true" json:"subjects_is_active"`
	SubjectsCreatedAt time.Time      `gorm:"column:subjects_created_at;not null;autoCreateTime" json:"subjects_created_at"`
	SubjectsUpdatedAt time.Time      `gorm:"column:subjects_updated_at;not null;autoUpdateTime" json:"subjects_updated_at"`
	SubjectsDeletedAt gorm.DeletedAt `gorm:"column:subjects_deleted_at;index"                 json:"subjects_deleted_at,omitempty"`
}

func (SubjectsModel) TableName() string { return "subjects" }
