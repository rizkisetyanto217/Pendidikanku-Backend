// file: internals/features/school/classes/model/class_parent_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ClassParent merepresentasikan tabel class_parents
type ClassParentModel struct {
	// PK & Tenant
	ClassParentID       uuid.UUID `gorm:"column:class_parent_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"class_parent_id"`
    ClassParentMasjidID uuid.UUID `gorm:"column:class_parent_masjid_id;type:uuid;not null;index:idx_class_parents_masjid" json:"class_parent_masjid_id"`

	// Identitas
	ClassParentName        string  `gorm:"column:class_parent_name;type:varchar(120);not null" json:"class_parent_name"`
	ClassParentCode        *string `gorm:"column:class_parent_code;type:varchar(40)" json:"class_parent_code"`
	ClassParentSlug        *string `gorm:"column:class_parent_slug;type:varchar(160)" json:"class_parent_slug"`
	ClassParentDescription *string `gorm:"column:class_parent_description;type:text" json:"class_parent_description"`

	// Atribut & status
	ClassParentLevel        *int16 `gorm:"column:class_parent_level" json:"class_parent_level"` // 0..100
	ClassParentIsActive     bool   `gorm:"column:class_parent_is_active;not null;default:true;index:idx_class_parents_active_alive,where:class_parent_deleted_at IS NULL" json:"class_parent_is_active"`
	ClassParentTotalClasses int32  `gorm:"column:class_parent_total_classes;not null;default:0" json:"class_parent_total_classes"`

	// Prasyarat/usia (fleksibel, JSONB)
	ClassParentRequirements datatypes.JSONMap `gorm:"column:class_parent_requirements;type:jsonb;not null;default:'{}'" json:"class_parent_requirements"`

	// Slot gambar
	ClassParentImageURL                 *string    `gorm:"column:class_parent_image_url" json:"class_parent_image_url"`
	ClassParentImageObjectKey           *string    `gorm:"column:class_parent_image_object_key" json:"class_parent_image_object_key"`
	ClassParentImageURLOld              *string    `gorm:"column:class_parent_image_url_old" json:"class_parent_image_url_old"`
	ClassParentImageObjectKeyOld        *string    `gorm:"column:class_parent_image_object_key_old" json:"class_parent_image_object_key_old"`
	ClassParentImageDeletePendingUntil  *time.Time `gorm:"column:class_parent_image_delete_pending_until;index:idx_class_parents_image_purge_due,where:class_parent_image_object_key_old IS NOT NULL" json:"class_parent_image_delete_pending_until"`

	// Audit
	ClassParentCreatedAt time.Time      `gorm:"column:class_parent_created_at;not null;autoCreateTime" json:"class_parent_created_at"`
	ClassParentUpdatedAt time.Time      `gorm:"column:class_parent_updated_at;not null;autoUpdateTime" json:"class_parent_updated_at"`
	ClassParentDeletedAt gorm.DeletedAt `gorm:"column:class_parent_deleted_at;index" json:"class_parent_deleted_at"`
}

// TableName menetapkan nama tabel
func (ClassParentModel) TableName() string {
	return "class_parents"
}