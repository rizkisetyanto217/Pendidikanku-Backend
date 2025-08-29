package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidServicePlan struct {
	MasjidServicePlanID          uuid.UUID      `gorm:"column:masjid_service_plan_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_service_plan_id"`

	MasjidServicePlanCode        string         `gorm:"column:masjid_service_plan_code;size:30;unique;not null" json:"masjid_service_plan_code"`
	MasjidServicePlanName        string         `gorm:"column:masjid_service_plan_name;size:100;not null" json:"masjid_service_plan_name"`
	MasjidServicePlanDescription *string        `gorm:"column:masjid_service_plan_description" json:"masjid_service_plan_description,omitempty"`

	MasjidServicePlanMaxTeachers  *int          `gorm:"column:masjid_service_plan_max_teachers" json:"masjid_service_plan_max_teachers,omitempty"`
	MasjidServicePlanMaxStudents  *int          `gorm:"column:masjid_service_plan_max_students" json:"masjid_service_plan_max_students,omitempty"`
	MasjidServicePlanMaxStorageMB *int          `gorm:"column:masjid_service_plan_max_storage_mb" json:"masjid_service_plan_max_storage_mb,omitempty"`

	MasjidServicePlanAllowCustomDomain    bool `gorm:"column:masjid_service_plan_allow_custom_domain;not null;default:false" json:"masjid_service_plan_allow_custom_domain"`
	MasjidServicePlanAllowCertificates    bool `gorm:"column:masjid_service_plan_allow_certificates;not null;default:false" json:"masjid_service_plan_allow_certificates"`
	MasjidServicePlanAllowPrioritySupport bool `gorm:"column:masjid_service_plan_allow_priority_support;not null;default:false" json:"masjid_service_plan_allow_priority_support"`

	// NUMERIC(12,2) — pakai float64 untuk simple; kalau butuh presisi uang, ganti ke decimal.Decimal
	MasjidServicePlanPriceMonthly *float64 `gorm:"column:masjid_service_plan_price_monthly;type:numeric(12,2)" json:"masjid_service_plan_price_monthly,omitempty"`
	MasjidServicePlanPriceYearly  *float64 `gorm:"column:masjid_service_plan_price_yearly;type:numeric(12,2)"  json:"masjid_service_plan_price_yearly,omitempty"`

	MasjidServicePlanIsActive bool `gorm:"column:masjid_service_plan_is_active;not null;default:true" json:"masjid_service_plan_is_active"`

	MasjidServicePlanCreatedAt time.Time      `gorm:"column:masjid_service_plan_created_at;autoCreateTime" json:"masjid_service_plan_created_at"`
	MasjidServicePlanUpdatedAt time.Time      `gorm:"column:masjid_service_plan_updated_at;autoUpdateTime" json:"masjid_service_plan_updated_at"`
	MasjidServicePlanDeletedAt gorm.DeletedAt `gorm:"column:masjid_service_plan_deleted_at;index" json:"masjid_service_plan_deleted_at,omitempty"`
}

func (MasjidServicePlan) TableName() string {
	return "masjid_service_plans"
}
