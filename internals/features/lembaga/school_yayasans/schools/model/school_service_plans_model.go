package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchoolServicePlan struct {
	SchoolServicePlanID uuid.UUID `gorm:"column:school_service_plan_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"school_service_plan_id"`

	// Uniqueness CI ditangani di DB via generated column + unique index
	SchoolServicePlanCode        string  `gorm:"column:school_service_plan_code;size:30;not null" json:"school_service_plan_code"`
	SchoolServicePlanName        string  `gorm:"column:school_service_plan_name;size:100;not null" json:"school_service_plan_name"`
	SchoolServicePlanDescription *string `gorm:"column:school_service_plan_description" json:"school_service_plan_description,omitempty"`

	// Image (single file, 2-slot + retensi)
	SchoolServicePlanImageURL                *string    `gorm:"column:school_service_plan_image_url" json:"school_service_plan_image_url,omitempty"`
	SchoolServicePlanImageObjectKey          *string    `gorm:"column:school_service_plan_image_object_key" json:"school_service_plan_image_object_key,omitempty"`
	SchoolServicePlanImageURLOld             *string    `gorm:"column:school_service_plan_image_url_old" json:"school_service_plan_image_url_old,omitempty"`
	SchoolServicePlanImageObjectKeyOld       *string    `gorm:"column:school_service_plan_image_object_key_old" json:"school_service_plan_image_object_key_old,omitempty"`
	SchoolServicePlanImageDeletePendingUntil *time.Time `gorm:"column:school_service_plan_image_delete_pending_until" json:"school_service_plan_image_delete_pending_until,omitempty"`

	// Limits & pricing
	SchoolServicePlanMaxTeachers  *int     `gorm:"column:school_service_plan_max_teachers" json:"school_service_plan_max_teachers,omitempty"`
	SchoolServicePlanMaxStudents  *int     `gorm:"column:school_service_plan_max_students" json:"school_service_plan_max_students,omitempty"`
	SchoolServicePlanMaxStorageMB *int     `gorm:"column:school_service_plan_max_storage_mb" json:"school_service_plan_max_storage_mb,omitempty"`
	SchoolServicePlanPriceMonthly *float64 `gorm:"column:school_service_plan_price_monthly;type:numeric(12,2)" json:"school_service_plan_price_monthly,omitempty"`
	SchoolServicePlanPriceYearly  *float64 `gorm:"column:school_service_plan_price_yearly;type:numeric(12,2)"  json:"school_service_plan_price_yearly,omitempty"`

	// Tema per-plan
	SchoolServicePlanAllowCustomTheme bool `gorm:"column:school_service_plan_allow_custom_theme;not null;default:false" json:"school_service_plan_allow_custom_theme"`
	SchoolServicePlanMaxCustomThemes  *int `gorm:"column:school_service_plan_max_custom_themes" json:"school_service_plan_max_custom_themes,omitempty"`

	// Status & audit
	SchoolServicePlanIsActive  bool           `gorm:"column:school_service_plan_is_active;not null;default:true" json:"school_service_plan_is_active"`
	SchoolServicePlanCreatedAt time.Time      `gorm:"column:school_service_plan_created_at;autoCreateTime" json:"school_service_plan_created_at"`
	SchoolServicePlanUpdatedAt time.Time      `gorm:"column:school_service_plan_updated_at;autoUpdateTime" json:"school_service_plan_updated_at"`
	SchoolServicePlanDeletedAt gorm.DeletedAt `gorm:"column:school_service_plan_deleted_at;index" json:"school_service_plan_deleted_at,omitempty"`
}

func (SchoolServicePlan) TableName() string { return "school_service_plans" }
