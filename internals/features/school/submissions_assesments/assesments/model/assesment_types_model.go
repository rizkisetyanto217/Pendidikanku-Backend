// file: internals/features/assessments/model/assessment_type_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssessmentTypeModel struct {
	AssessmentTypeID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:assessment_type_id" json:"assessment_type_id"`
	AssessmentTypeSchoolID      uuid.UUID `gorm:"type:uuid;not null;column:assessment_type_school_id" json:"assessment_type_school_id"`
	AssessmentTypeKey           string    `gorm:"type:varchar(32);not null;column:assessment_type_key" json:"assessment_type_key"`
	AssessmentTypeName          string    `gorm:"type:varchar(120);not null;column:assessment_type_name" json:"assessment_type_name"`
	AssessmentTypeWeightPercent float64   `gorm:"type:numeric(5,2);not null;default:0;column:assessment_type_weight_percent" json:"assessment_type_weight_percent"`
	AssessmentTypeIsActive      bool      `gorm:"not null;default:true;column:assessment_type_is_active" json:"assessment_type_is_active"`

	AssessmentTypeCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_type_created_at" json:"assessment_type_created_at"`
	AssessmentTypeUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:assessment_type_updated_at" json:"assessment_type_updated_at"`
	AssessmentTypeDeletedAt gorm.DeletedAt `gorm:"column:assessment_type_deleted_at;index" json:"assessment_type_deleted_at,omitempty"`
}

func (AssessmentTypeModel) TableName() string { return "assessment_types" }
