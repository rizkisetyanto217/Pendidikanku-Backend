// file: internals/features/finance/general_billings/model/general_billing.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
SQL acuan (ringkas):
- general_billing_masjid_id UUID NULL (GLOBAL bila NULL)
- general_billing_kind_id   UUID NOT NULL (FK RESTRICT)
- general_billing_code      VARCHAR(60) NULL (unique per-tenant/global via partial index)
- general_billing_title     TEXT NOT NULL
- general_billing_desc      TEXT NULL
- general_billing_due_date  DATE NULL
- general_billing_is_active BOOLEAN NOT NULL DEFAULT TRUE
- general_billing_default_amount_idr INT NULL CHECK (>=0)
- created_at/updated_at/deleted_at
*/

type GeneralBilling struct {
	GeneralBillingID uuid.UUID `json:"general_billing_id" gorm:"column:general_billing_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// NULL = GLOBAL (milik aplikasi), non-NULL = tenant-scoped
	GeneralBillingMasjidID *uuid.UUID `json:"general_billing_masjid_id,omitempty" gorm:"column:general_billing_masjid_id;type:uuid"`

	// kind (ON UPDATE CASCADE, ON DELETE RESTRICT)
	GeneralBillingKindID uuid.UUID `json:"general_billing_kind_id" gorm:"column:general_billing_kind_id;type:uuid;not null;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	// basic fields
	GeneralBillingCode  *string `json:"general_billing_code,omitempty" gorm:"column:general_billing_code;type:varchar(60)"`
	GeneralBillingTitle string  `json:"general_billing_title" gorm:"column:general_billing_title;type:text;not null"`
	GeneralBillingDesc  *string `json:"general_billing_desc,omitempty" gorm:"column:general_billing_desc;type:text"`

	// schedule/flags
	GeneralBillingDueDate  *time.Time `json:"general_billing_due_date,omitempty" gorm:"column:general_billing_due_date;type:date"`
	GeneralBillingIsActive bool       `json:"general_billing_is_active" gorm:"column:general_billing_is_active;not null;default:true"`

	// default amount (nullable INT, CHECK >= 0 ada di DB)
	GeneralBillingDefaultAmountIDR *int `json:"general_billing_default_amount_idr,omitempty" gorm:"column:general_billing_default_amount_idr;type:int"`

	// timestamps (soft delete manual, bukan gorm.DeletedAt)
	GeneralBillingCreatedAt time.Time  `json:"general_billing_created_at" gorm:"column:general_billing_created_at;type:timestamptz;not null;default:now()"`
	GeneralBillingUpdatedAt time.Time  `json:"general_billing_updated_at" gorm:"column:general_billing_updated_at;type:timestamptz;not null;default:now()"`
	GeneralBillingDeletedAt *time.Time `json:"general_billing_deleted_at,omitempty" gorm:"column:general_billing_deleted_at;type:timestamptz"`
}

func (GeneralBilling) TableName() string { return "general_billings" }

/* =========================
   Hooks: refresh updated_at
   ========================= */

func (g *GeneralBilling) BeforeCreate(tx *gorm.DB) error {
	g.GeneralBillingUpdatedAt = time.Now().UTC()
	return nil
}
func (g *GeneralBilling) BeforeUpdate(tx *gorm.DB) error {
	g.GeneralBillingUpdatedAt = time.Now().UTC()
	return nil
}
