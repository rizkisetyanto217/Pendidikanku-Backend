// file: internals/features/finance/fee_rules/model/fee_rule.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===================== ENUM fee_scope ===================== */

type FeeScope string

const (
	FeeScopeTenant      FeeScope = "tenant"
	FeeScopeClassParent FeeScope = "class_parent"
	FeeScopeClass       FeeScope = "class"
	FeeScopeSection     FeeScope = "section"
	FeeScopeStudent     FeeScope = "student"
	FeeScopeTerm        FeeScope = "term"
)

/* ===================== ENUM general_billing_category ===================== */
/* (mirror enum di DB: general_billing_category) */

type GeneralBillingCategory string

const (
	GeneralBillingCategoryRegistration GeneralBillingCategory = "registration"
	GeneralBillingCategorySPP          GeneralBillingCategory = "spp"
	GeneralBillingCategoryMassStudent  GeneralBillingCategory = "mass_student"
	GeneralBillingCategoryDonation     GeneralBillingCategory = "donation"
)

/* ===================== JSONB AmountOption ===================== */

type AmountOption struct {
	Code   string `json:"code"`   // contoh: "T1"
	Label  string `json:"label"`  // contoh: "Tarif 1"
	Amount int    `json:"amount"` // contoh: 500000
}

/* ===================== MODEL fee_rules ===================== */

type FeeRuleModel struct {
	// PK
	FeeRuleID uuid.UUID `json:"fee_rule_id" gorm:"column:fee_rule_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// Tenant
	FeeRuleSchoolID uuid.UUID `json:"fee_rule_school_id" gorm:"column:fee_rule_school_id;type:uuid;not null"`

	// Scope + Target
	FeeRuleScope           FeeScope   `json:"fee_rule_scope" gorm:"column:fee_rule_scope;type:fee_scope;not null"`
	FeeRuleClassParentID   *uuid.UUID `json:"fee_rule_class_parent_id,omitempty" gorm:"column:fee_rule_class_parent_id;type:uuid"`
	FeeRuleClassID         *uuid.UUID `json:"fee_rule_class_id,omitempty" gorm:"column:fee_rule_class_id;type:uuid"`
	FeeRuleSectionID       *uuid.UUID `json:"fee_rule_section_id,omitempty" gorm:"column:fee_rule_section_id;type:uuid"`
	FeeRuleSchoolStudentID *uuid.UUID `json:"fee_rule_school_student_id,omitempty" gorm:"column:fee_rule_school_student_id;type:uuid"`

	// Periode (pilih: term_id ATAU year+month)
	FeeRuleTermID *uuid.UUID `json:"fee_rule_term_id,omitempty" gorm:"column:fee_rule_term_id;type:uuid"`
	FeeRuleMonth  *int16     `json:"fee_rule_month,omitempty" gorm:"column:fee_rule_month;type:smallint"`
	FeeRuleYear   *int16     `json:"fee_rule_year,omitempty" gorm:"column:fee_rule_year;type:smallint"`

	// Jenis rule (kategori + bill_code)
	FeeRuleCategory GeneralBillingCategory `json:"fee_rule_category" gorm:"column:fee_rule_category;type:general_billing_category;not null"`
	FeeRuleBillCode string                 `json:"fee_rule_bill_code" gorm:"column:fee_rule_bill_code;type:varchar(60);not null;default:'SPP'"`

	// Opsi/label default (single, denorm penanda)
	FeeRuleOptionCode  string  `json:"fee_rule_option_code" gorm:"column:fee_rule_option_code;type:varchar(20);not null;default:'T1'"`
	FeeRuleOptionLabel *string `json:"fee_rule_option_label,omitempty" gorm:"column:fee_rule_option_label;type:varchar(60)"`
	FeeRuleIsDefault   bool    `json:"fee_rule_is_default" gorm:"column:fee_rule_is_default;type:boolean;not null;default:false"`

	// JSONB daftar opsi harga: [{code,label,amount}, ...]
	// serializer:json → []AmountOption <-> JSONB
	FeeRuleAmountOptions []AmountOption `json:"fee_rule_amount_options" gorm:"column:fee_rule_amount_options;type:jsonb;not null;serializer:json"`

	// Effective window
	FeeRuleEffectiveFrom *time.Time `json:"fee_rule_effective_from,omitempty" gorm:"column:fee_rule_effective_from;type:date"`
	FeeRuleEffectiveTo   *time.Time `json:"fee_rule_effective_to,omitempty" gorm:"column:fee_rule_effective_to;type:date"`

	// Catatan
	FeeRuleNote *string `json:"fee_rule_note,omitempty" gorm:"column:fee_rule_note;type:text"`

	// Timestamps (soft delete manual via gorm.DeletedAt)
	FeeRuleCreatedAt time.Time      `json:"fee_rule_created_at" gorm:"column:fee_rule_created_at;type:timestamptz;not null;autoCreateTime"`
	FeeRuleUpdatedAt time.Time      `json:"fee_rule_updated_at" gorm:"column:fee_rule_updated_at;type:timestamptz;not null;autoUpdateTime"`
	FeeRuleDeletedAt gorm.DeletedAt `json:"fee_rule_deleted_at,omitempty" gorm:"column:fee_rule_deleted_at;type:timestamptz;index"`
}

func (FeeRuleModel) TableName() string { return "fee_rules" }
