// file: internals/features/billings/general_billing_kinds/model/general_billing_kind.go
package model

import (
	"time"

	"github.com/google/uuid"
)

/* =========================
   Enums (Go-side, opsional)
   ========================= */

type GeneralBillingKindCategory string

const (
	GBKCategoryBilling  GeneralBillingKindCategory = "billing"
	GBKCategoryCampaign GeneralBillingKindCategory = "campaign"
)

type GeneralBillingKindVisibility string

const (
	GBKVisibilityPublic   GeneralBillingKindVisibility = "public"
	GBKVisibilityInternal GeneralBillingKindVisibility = "internal"
)

/* =========================
   Model
   ========================= */

type GeneralBillingKind struct {
	GeneralBillingKindID uuid.UUID `json:"general_billing_kind_id" gorm:"column:general_billing_kind_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// NULLABLE: global kind (operasional aplikasi) tidak punya masjid_id
	GeneralBillingKindMasjidID *uuid.UUID `json:"general_billing_kind_masjid_id,omitempty" gorm:"column:general_billing_kind_masjid_id;type:uuid"`

	GeneralBillingKindCode string  `json:"general_billing_kind_code" gorm:"column:general_billing_kind_code;type:varchar(60);not null"`
	GeneralBillingKindName string  `json:"general_billing_kind_name" gorm:"column:general_billing_kind_name;type:text;not null"`
	GeneralBillingKindDesc *string `json:"general_billing_kind_desc,omitempty" gorm:"column:general_billing_kind_desc;type:text"`

	GeneralBillingKindIsActive bool `json:"general_billing_kind_is_active" gorm:"column:general_billing_kind_is_active;not null;default:true"`

	// INT di DB; nullable + CHECK >= 0 (ditangani di DB)
	GeneralBillingKindDefaultAmountIDR *int `json:"general_billing_kind_default_amount_idr,omitempty" gorm:"column:general_billing_kind_default_amount_idr;type:int"`

	// category (DEFAULT 'billing' di DB), visibility nullable, is_global DEFAULT false
	GeneralBillingKindCategory   GeneralBillingKindCategory    `json:"general_billing_kind_category" gorm:"column:general_billing_kind_category;type:varchar(20);default:'billing'"`
	GeneralBillingKindIsGlobal   bool                          `json:"general_billing_kind_is_global"  gorm:"column:general_billing_kind_is_global;not null;default:false"`
	GeneralBillingKindVisibility *GeneralBillingKindVisibility `json:"general_billing_kind_visibility,omitempty" gorm:"column:general_billing_kind_visibility;type:varchar(20)"`

	// Flags pipeline per-siswa (baru di SQL)
	GeneralBillingKindIsRecurring        bool `json:"general_billing_kind_is_recurring"         gorm:"column:general_billing_kind_is_recurring;not null;default:false"`
	GeneralBillingKindRequiresMonthYear  bool `json:"general_billing_kind_requires_month_year"  gorm:"column:general_billing_kind_requires_month_year;not null;default:false"`
	GeneralBillingKindRequiresOptionCode bool `json:"general_billing_kind_requires_option_code" gorm:"column:general_billing_kind_requires_option_code;not null;default:false"`

	GeneralBillingKindCreatedAt time.Time  `json:"general_billing_kind_created_at" gorm:"column:general_billing_kind_created_at;type:timestamptz;not null;default:now()"`
	GeneralBillingKindUpdatedAt time.Time  `json:"general_billing_kind_updated_at" gorm:"column:general_billing_kind_updated_at;type:timestamptz;not null;default:now()"`
	GeneralBillingKindDeletedAt *time.Time `json:"general_billing_kind_deleted_at,omitempty" gorm:"column:general_billing_kind_deleted_at;type:timestamptz"`
}

func (GeneralBillingKind) TableName() string { return "general_billing_kinds" }

