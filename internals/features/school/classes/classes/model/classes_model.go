// models/class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClassModel merepresentasikan tabel `classes`
type ClassModel struct {
	// PK & tenant
	ClassID       uuid.UUID `json:"class_id"        gorm:"column:class_id;type:uuid;default:gen_random_uuid();primaryKey"`
	ClassMasjidID uuid.UUID `json:"class_masjid_id" gorm:"column:class_masjid_id;type:uuid;not null"`

	// Identitas
	ClassName string  `json:"class_name" gorm:"column:class_name;type:varchar(120);not null"`
	ClassSlug string  `json:"class_slug" gorm:"column:class_slug;type:varchar(160);not null"`
	ClassCode *string `json:"class_code,omitempty" gorm:"column:class_code;type:varchar(40)"`

	// Info tambahan
	ClassDescription *string `json:"class_description,omitempty" gorm:"column:class_description;type:text"`
	ClassLevel       *string `json:"class_level,omitempty" gorm:"column:class_level;type:text"`
	ClassImageURL    *string `json:"class_image_url,omitempty" gorm:"column:class_image_url;type:text"`

	// Penghapusan terjadwal
	ClassTrashURL           *string    `json:"class_trash_url,omitempty" gorm:"column:class_trash_url;type:text"`
	ClassDeletePendingUntil *time.Time `json:"class_delete_pending_until,omitempty" gorm:"column:class_delete_pending_until;type:timestamptz"`

	// Mode & status (mode bebas, default di DB: 'tatap muka')
	ClassMode   string `json:"class_mode"    gorm:"column:class_mode;type:varchar;not null"`
	ClassIsActive bool `json:"class_is_active" gorm:"column:class_is_active;not null;default:true"`

	// Timestamps (updated_at ditouch oleh trigger di DB)
	ClassCreatedAt time.Time      `json:"class_created_at" gorm:"column:class_created_at;type:timestamptz;not null;default:now()"`
	ClassUpdatedAt time.Time      `json:"class_updated_at" gorm:"column:class_updated_at;type:timestamptz;not null;default:now()"`
	DeletedAt      gorm.DeletedAt `json:"class_deleted_at,omitempty" gorm:"column:class_deleted_at;type:timestamptz;index"`
}

func (ClassModel) TableName() string {
	return "classes"
}
