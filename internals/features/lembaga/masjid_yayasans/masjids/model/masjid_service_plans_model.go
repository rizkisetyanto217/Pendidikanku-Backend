package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidServicePlan struct {
	MasjidServicePlanID uuid.UUID `gorm:"column:masjid_service_plan_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_service_plan_id"`

	// Uniqueness CI ditangani di DB via generated column + unique index
	MasjidServicePlanCode        string  `gorm:"column:masjid_service_plan_code;size:30;not null" json:"masjid_service_plan_code"`
	MasjidServicePlanName        string  `gorm:"column:masjid_service_plan_name;size:100;not null" json:"masjid_service_plan_name"`
	MasjidServicePlanDescription *string `gorm:"column:masjid_service_plan_description" json:"masjid_service_plan_description,omitempty"`

	// Image (single file, 2-slot + retensi)
	MasjidServicePlanImageURL                *string    `gorm:"column:masjid_service_plan_image_url" json:"masjid_service_plan_image_url,omitempty"`
	MasjidServicePlanImageObjectKey          *string    `gorm:"column:masjid_service_plan_image_object_key" json:"masjid_service_plan_image_object_key,omitempty"`
	MasjidServicePlanImageURLOld             *string    `gorm:"column:masjid_service_plan_image_url_old" json:"masjid_service_plan_image_url_old,omitempty"`
	MasjidServicePlanImageObjectKeyOld       *string    `gorm:"column:masjid_service_plan_image_object_key_old" json:"masjid_service_plan_image_object_key_old,omitempty"`
	MasjidServicePlanImageDeletePendingUntil *time.Time `gorm:"column:masjid_service_plan_image_delete_pending_until" json:"masjid_service_plan_image_delete_pending_until,omitempty"`

	// Limits & pricing
	MasjidServicePlanMaxTeachers  *int     `gorm:"column:masjid_service_plan_max_teachers" json:"masjid_service_plan_max_teachers,omitempty"`
	MasjidServicePlanMaxStudents  *int     `gorm:"column:masjid_service_plan_max_students" json:"masjid_service_plan_max_students,omitempty"`
	MasjidServicePlanMaxStorageMB *int     `gorm:"column:masjid_service_plan_max_storage_mb" json:"masjid_service_plan_max_storage_mb,omitempty"`
	MasjidServicePlanPriceMonthly *float64 `gorm:"column:masjid_service_plan_price_monthly;type:numeric(12,2)" json:"masjid_service_plan_price_monthly,omitempty"`
	MasjidServicePlanPriceYearly  *float64 `gorm:"column:masjid_service_plan_price_yearly;type:numeric(12,2)"  json:"masjid_service_plan_price_yearly,omitempty"`

	// Tema per-plan
	MasjidServicePlanAllowCustomTheme bool `gorm:"column:masjid_service_plan_allow_custom_theme;not null;default:false" json:"masjid_service_plan_allow_custom_theme"`
	MasjidServicePlanMaxCustomThemes  *int `gorm:"column:masjid_service_plan_max_custom_themes" json:"masjid_service_plan_max_custom_themes,omitempty"`

	// Status & audit
	MasjidServicePlanIsActive bool           `gorm:"column:masjid_service_plan_is_active;not null;default:true" json:"masjid_service_plan_is_active"`
	MasjidServicePlanCreatedAt time.Time      `gorm:"column:masjid_service_plan_created_at;autoCreateTime" json:"masjid_service_plan_created_at"`
	MasjidServicePlanUpdatedAt time.Time      `gorm:"column:masjid_service_plan_updated_at;autoUpdateTime" json:"masjid_service_plan_updated_at"`
	MasjidServicePlanDeletedAt gorm.DeletedAt `gorm:"column:masjid_service_plan_deleted_at;index" json:"masjid_service_plan_deleted_at,omitempty"`
}

func (MasjidServicePlan) TableName() string { return "masjid_service_plans" }
