// models/class_model.go
package model

import (
	"time"

	"github.com/google/uuid"
)

// ClassModel merepresentasikan tabel `classes`
type ClassModel struct {
	ClassID              uuid.UUID  `json:"class_id" gorm:"column:class_id;type:uuid;default:gen_random_uuid();primaryKey"`
	ClassMasjidID        *uuid.UUID `json:"class_masjid_id,omitempty" gorm:"column:class_masjid_id;type:uuid"` // FK -> masjids(masjid_id)

	ClassName            string     `json:"class_name" gorm:"column:class_name;type:varchar(120);not null"`
	ClassSlug            string     `json:"class_slug" gorm:"column:class_slug;type:varchar(160);unique;not null"`
	ClassDescription     *string    `json:"class_description,omitempty" gorm:"column:class_description;type:text"`
	ClassLevel           *string    `json:"class_level,omitempty" gorm:"column:class_level;type:text"`

	ClassImageURL        *string    `json:"class_image_url,omitempty" gorm:"column:class_image_url;type:text"`

	// NULL = gratis; >= 0 = tarif per bulan (IDR)
	ClassFeeMonthlyIDR   *int       `json:"class_fee_monthly_idr,omitempty" gorm:"column:class_fee_monthly_idr"`

	ClassIsActive        bool       `json:"class_is_active" gorm:"column:class_is_active;not null;default:true"`

	ClassCreatedAt       time.Time  `json:"class_created_at" gorm:"column:class_created_at;not null;autoCreateTime"`
	ClassUpdatedAt       *time.Time `json:"class_updated_at,omitempty" gorm:"column:class_updated_at"`
	ClassDeletedAt       *time.Time `json:"class_deleted_at,omitempty" gorm:"column:class_deleted_at"`
}

func (ClassModel) TableName() string {
	return "classes"
}
