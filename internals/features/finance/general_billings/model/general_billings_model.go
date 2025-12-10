// file: internals/features/finance/general_billings/model/general_billing.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
   Mirror dari SQL:

   - general_billing_school_id      UUID NOT NULL
   - general_billing_category       general_billing_category NOT NULL
   - general_billing_bill_code      VARCHAR(60) NOT NULL DEFAULT 'SPP'
   - general_billing_code           VARCHAR(60) NULL
   - general_billing_title          TEXT NOT NULL
   - general_billing_desc           TEXT NULL
   - general_billing_class_id       UUID NULL
   - general_billing_section_id     UUID NULL
   - general_billing_term_id        UUID NULL
   - general_billing_month          SMALLINT NULL
   - general_billing_year           SMALLINT NULL
   - general_billing_due_date       DATE NULL
   - general_billing_is_active      BOOLEAN NOT NULL DEFAULT TRUE
   - general_billing_default_amount_idr INT NULL CHECK (>= 0)
   - created_at / updated_at / deleted_at
*/

// Enum mirror untuk general_billing_category
type GeneralBillingCategory string

const (
	GeneralBillingCategoryRegistration GeneralBillingCategory = "registration"
	GeneralBillingCategorySPP          GeneralBillingCategory = "spp"
	GeneralBillingCategoryMassStudent  GeneralBillingCategory = "mass_student"
	GeneralBillingCategoryDonation     GeneralBillingCategory = "donation"
)

type GeneralBillingModel struct {
	GeneralBillingID uuid.UUID `json:"general_billing_id" gorm:"column:general_billing_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// Selalu tenant-scoped (NOT NULL)
	GeneralBillingSchoolID uuid.UUID `json:"general_billing_school_id" gorm:"column:general_billing_school_id;type:uuid;not null"`

	// Kategori & bill code
	GeneralBillingCategory  GeneralBillingCategory `json:"general_billing_category" gorm:"column:general_billing_category;type:general_billing_category;not null"`
	GeneralBillingBillCode  string                 `json:"general_billing_bill_code" gorm:"column:general_billing_bill_code;type:varchar(60);not null;default:'SPP'"`
	GeneralBillingCode      *string                `json:"general_billing_code,omitempty" gorm:"column:general_billing_code;type:varchar(60)"`

	// Basic info
	GeneralBillingTitle string  `json:"general_billing_title" gorm:"column:general_billing_title;type:text;not null"`
	GeneralBillingDesc  *string `json:"general_billing_desc,omitempty" gorm:"column:general_billing_desc;type:text"`

	// Scope akademik (opsional)
	GeneralBillingClassID   *uuid.UUID `json:"general_billing_class_id,omitempty" gorm:"column:general_billing_class_id;type:uuid"`
	GeneralBillingSectionID *uuid.UUID `json:"general_billing_section_id,omitempty" gorm:"column:general_billing_section_id;type:uuid"`
	GeneralBillingTermID    *uuid.UUID `json:"general_billing_term_id,omitempty" gorm:"column:general_billing_term_id;type:uuid"`

	// Periode (opsional, penting untuk SPP)
	GeneralBillingMonth *int16 `json:"general_billing_month,omitempty" gorm:"column:general_billing_month;type:smallint"`
	GeneralBillingYear  *int16 `json:"general_billing_year,omitempty" gorm:"column:general_billing_year;type:smallint"`

	// Jatuh tempo & status
	GeneralBillingDueDate  *time.Time `json:"general_billing_due_date,omitempty" gorm:"column:general_billing_due_date;type:date"`
	GeneralBillingIsActive bool       `json:"general_billing_is_active" gorm:"column:general_billing_is_active;not null;default:true"`

	// Default nominal (boleh NULL)
	GeneralBillingDefaultAmountIDR *int `json:"general_billing_default_amount_idr,omitempty" gorm:"column:general_billing_default_amount_idr;type:int"`

	// Timestamps
	GeneralBillingCreatedAt time.Time  `json:"general_billing_created_at" gorm:"column:general_billing_created_at;type:timestamptz;not null;default:now()"`
	GeneralBillingUpdatedAt time.Time  `json:"general_billing_updated_at" gorm:"column:general_billing_updated_at;type:timestamptz;not null;default:now()"`
	GeneralBillingDeletedAt *time.Time `json:"general_billing_deleted_at,omitempty" gorm:"column:general_billing_deleted_at;type:timestamptz"`
}

func (GeneralBillingModel) TableName() string { return "general_billings" }

/* =========================
   Hooks: refresh updated_at
   ========================= */

func (g *GeneralBillingModel) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().UTC()
	g.GeneralBillingCreatedAt = now
	g.GeneralBillingUpdatedAt = now
	return nil
}

func (g *GeneralBillingModel) BeforeUpdate(tx *gorm.DB) error {
	g.GeneralBillingUpdatedAt = time.Now().UTC()
	return nil
}
