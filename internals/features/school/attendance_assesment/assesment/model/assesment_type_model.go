// file: internals/features/school/assessments/model/assessment_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssessmentTypeModel merepresentasikan tabel `assessment_types`
type AssessmentTypeModel struct {
	// PK
	ID uuid.UUID `json:"assessment_types_id" gorm:"column:assessment_types_id;type:uuid;primaryKey"`

	// Tenant
	MasjidID uuid.UUID `json:"assessment_types_masjid_id" gorm:"column:assessment_types_masjid_id;type:uuid;not null;index:idx_assessment_types_masjid_active,priority:1;uniqueIndex:uq_assessment_types_masjid_key,priority:1"`

	// Business columns
	Key   string `json:"assessment_types_key"  gorm:"column:assessment_types_key;type:varchar(32);not null;uniqueIndex:uq_assessment_types_masjid_key,priority:2"`
	Name  string `json:"assessment_types_name" gorm:"column:assessment_types_name;type:varchar(120);not null"`
	// Gunakan float32/float64. Jika perlu presisi fixed-point, bisa ganti ke shopspring/decimal.
	WeightPercent float32 `json:"assessment_types_weight_percent" gorm:"column:assessment_types_weight_percent;type:numeric(5,2);not null;default:0"`

	IsActive bool `json:"assessment_types_is_active" gorm:"column:assessment_types_is_active;not null;default:true;index:idx_assessment_types_masjid_active,priority:2"`

	// Timestamps
	CreatedAt time.Time      `json:"assessment_types_created_at" gorm:"column:assessment_types_created_at;not null;autoCreateTime"`
	UpdatedAt time.Time      `json:"assessment_types_updated_at" gorm:"column:assessment_types_updated_at;not null;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"assessment_types_deleted_at" gorm:"column:assessment_types_deleted_at;index"`
}

// TableName memastikan nama tabel sesuai DDL
func (AssessmentTypeModel) TableName() string { return "assessment_types" }
