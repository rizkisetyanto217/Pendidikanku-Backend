// file: internals/features/school/academics/classes/model/class_parent_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ClassParentModel merepresentasikan tabel class_parents (sesuai DDL)
type ClassParentModel struct {
	// PK & Tenant
	ClassParentID       uuid.UUID `json:"class_parent_id"        gorm:"column:class_parent_id;type:uuid;primaryKey;default:gen_random_uuid()"`
	ClassParentSchoolID uuid.UUID `json:"class_parent_school_id" gorm:"column:class_parent_school_id;type:uuid;not null"`

	// Identitas
	ClassParentName        string  `json:"class_parent_name"        gorm:"column:class_parent_name;type:varchar(120);not null"`
	ClassParentCode        *string `json:"class_parent_code,omitempty"        gorm:"column:class_parent_code;type:varchar(40)"`
	ClassParentSlug        *string `json:"class_parent_slug,omitempty"        gorm:"column:class_parent_slug;type:varchar(160)"`
	ClassParentDescription *string `json:"class_parent_description,omitempty" gorm:"column:class_parent_description;type:text"`

	// Atribut & status
	ClassParentLevel        *int16 `json:"class_parent_level,omitempty"        gorm:"column:class_parent_level"` // 0..100 (cek di DB)
	ClassParentIsActive     bool   `json:"class_parent_is_active"              gorm:"column:class_parent_is_active;not null;default:true"`
	ClassParentTotalClasses int32  `json:"class_parent_total_classes"          gorm:"column:class_parent_total_classes;not null;default:0"`

	// Prasyarat/usia (fleksibel JSONB)
	ClassParentRequirements datatypes.JSONMap `json:"class_parent_requirements" gorm:"column:class_parent_requirements;type:jsonb;not null;default:'{}'"`

	// Single image (2-slot + retensi)
	ClassParentImageURL                *string    `json:"class_parent_image_url,omitempty"                gorm:"column:class_parent_image_url;type:text"`
	ClassParentImageObjectKey          *string    `json:"class_parent_image_object_key,omitempty"          gorm:"column:class_parent_image_object_key;type:text"`
	ClassParentImageURLOld             *string    `json:"class_parent_image_url_old,omitempty"             gorm:"column:class_parent_image_url_old;type:text"`
	ClassParentImageObjectKeyOld       *string    `json:"class_parent_image_object_key_old,omitempty"      gorm:"column:class_parent_image_object_key_old;type:text"`
	ClassParentImageDeletePendingUntil *time.Time `json:"class_parent_image_delete_pending_until,omitempty" gorm:"column:class_parent_image_delete_pending_until;type:timestamptz"`

	// Audit
	ClassParentCreatedAt time.Time      `json:"class_parent_created_at"           gorm:"column:class_parent_created_at;type:timestamptz;not null;default:now();autoCreateTime"`
	ClassParentUpdatedAt time.Time      `json:"class_parent_updated_at"           gorm:"column:class_parent_updated_at;type:timestamptz;not null;default:now();autoUpdateTime"`
	ClassParentDeletedAt gorm.DeletedAt `json:"class_parent_deleted_at,omitempty" gorm:"column:class_parent_deleted_at;type:timestamptz;index"`
}

func (ClassParentModel) TableName() string { return "class_parents" }
