// file: internals/features/finance/fee_rules/model/fee_rule.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- ENUM fee_scope (ikuti enum di DB) ---------------------------------------
type FeeScope string

const (
	FeeScopeTenant      FeeScope = "tenant"
	FeeScopeClassParent FeeScope = "class_parent"
	FeeScopeClass       FeeScope = "class"
	FeeScopeSection     FeeScope = "section"
	FeeScopeStudent     FeeScope = "student"
)

// --- MODEL fee_rules ---------------------------------------------------------
type FeeRule struct {
	// PK
	FeeRuleID uuid.UUID `json:"fee_rule_id" gorm:"column:fee_rule_id;type:uuid;default:gen_random_uuid();primaryKey"`

	// Tenant
	FeeRuleSchoolID uuid.UUID `json:"fee_rule_school_id" gorm:"column:fee_rule_school_id;type:uuid;not null;index:idx_fee_rules_tenant_scope,priority:1"`

	// Scope + Target
	FeeRuleScope           FeeScope   `json:"fee_rule_scope" gorm:"column:fee_rule_scope;type:fee_scope;not null;index:idx_fee_rules_tenant_scope,priority:2;index:ix_fee_rules_billcode,priority:2"`
	FeeRuleClassParentID   *uuid.UUID `json:"fee_rule_class_parent_id,omitempty" gorm:"column:fee_rule_class_parent_id;type:uuid"`
	FeeRuleClassID         *uuid.UUID `json:"fee_rule_class_id,omitempty" gorm:"column:fee_rule_class_id;type:uuid"`
	FeeRuleSectionID       *uuid.UUID `json:"fee_rule_section_id,omitempty" gorm:"column:fee_rule_section_id;type:uuid"`
	FeeRuleSchoolStudentID *uuid.UUID `json:"fee_rule_school_student_id,omitempty" gorm:"column:fee_rule_school_student_id;type:uuid"`

	// Periode (pilih: term_id ATAU year+month)
	FeeRuleTermID *uuid.UUID `json:"fee_rule_term_id,omitempty" gorm:"column:fee_rule_term_id;type:uuid;index:idx_fee_rules_term"`
	FeeRuleMonth  *int16     `json:"fee_rule_month,omitempty" gorm:"column:fee_rule_month;type:smallint;index:idx_fee_rules_month_year,priority:2"`
	FeeRuleYear   *int16     `json:"fee_rule_year,omitempty" gorm:"column:fee_rule_year;type:smallint;index:idx_fee_rules_month_year,priority:1"`

	// Jenis rule (link ke katalog + denorm code)
	FeeRuleGeneralBillingKindID *uuid.UUID `json:"fee_rule_general_billing_kind_id,omitempty" gorm:"column:fee_rule_general_billing_kind_id;type:uuid;index:ix_fee_rules_gbk"`
	FeeRuleBillCode             string     `json:"fee_rule_bill_code" gorm:"column:fee_rule_bill_code;type:varchar(60);not null;default:'SPP';index:ix_fee_rules_billcode,priority:1"`

	// Opsi/label
	FeeRuleOptionCode  string  `json:"fee_rule_option_code" gorm:"column:fee_rule_option_code;type:varchar(20);not null;default:'T1';index:idx_fee_rules_option_code"` // NOTE: functional index LOWER() dibuat di SQL migrasi
	FeeRuleOptionLabel *string `json:"fee_rule_option_label,omitempty" gorm:"column:fee_rule_option_label;type:varchar(60)"`

	// Default & nominal
	FeeRuleIsDefault bool `json:"fee_rule_is_default" gorm:"column:fee_rule_is_default;type:boolean;not null;default:false;index:idx_fee_rules_is_default"`
	FeeRuleAmountIDR int  `json:"fee_rule_amount_idr" gorm:"column:fee_rule_amount_idr;type:int;not null;index:idx_fee_rules_amount"`

	// Effective window
	FeeRuleEffectiveFrom *time.Time `json:"fee_rule_effective_from,omitempty" gorm:"column:fee_rule_effective_from;type:date"`
	FeeRuleEffectiveTo   *time.Time `json:"fee_rule_effective_to,omitempty" gorm:"column:fee_rule_effective_to;type:date"`

	// Catatan
	FeeRuleNote *string `json:"fee_rule_note,omitempty" gorm:"column:fee_rule_note;type:text"`

	// --- SNAPSHOT kolom GBK (diisi oleh backend) -----------------------------
	FeeRuleGBKCodeSnapshot               *string `json:"fee_rule_gbk_code_snapshot,omitempty" gorm:"column:fee_rule_gbk_code_snapshot;type:varchar(60)"`
	FeeRuleGBKNameSnapshot               *string `json:"fee_rule_gbk_name_snapshot,omitempty" gorm:"column:fee_rule_gbk_name_snapshot;type:text"`
	FeeRuleGBKCategorySnapshot           *string `json:"fee_rule_gbk_category_snapshot,omitempty" gorm:"column:fee_rule_gbk_category_snapshot;type:varchar(20)"`
	FeeRuleGBKIsGlobalSnapshot           *bool   `json:"fee_rule_gbk_is_global_snapshot,omitempty" gorm:"column:fee_rule_gbk_is_global_snapshot;type:boolean"`
	FeeRuleGBKVisibilitySnapshot         *string `json:"fee_rule_gbk_visibility_snapshot,omitempty" gorm:"column:fee_rule_gbk_visibility_snapshot;type:varchar(20)"`
	FeeRuleGBKIsRecurringSnapshot        *bool   `json:"fee_rule_gbk_is_recurring_snapshot,omitempty" gorm:"column:fee_rule_gbk_is_recurring_snapshot;type:boolean"`
	FeeRuleGBKRequiresMonthYearSnapshot  *bool   `json:"fee_rule_gbk_requires_month_year_snapshot,omitempty" gorm:"column:fee_rule_gbk_requires_month_year_snapshot;type:boolean"`
	FeeRuleGBKRequiresOptionCodeSnapshot *bool   `json:"fee_rule_gbk_requires_option_code_snapshot,omitempty" gorm:"column:fee_rule_gbk_requires_option_code_snapshot;type:boolean"`
	FeeRuleGBKDefaultAmountIDRSnapshot   *int    `json:"fee_rule_gbk_default_amount_idr_snapshot,omitempty" gorm:"column:fee_rule_gbk_default_amount_idr_snapshot;type:int"`
	FeeRuleGBKIsActiveSnapshot           *bool   `json:"fee_rule_gbk_is_active_snapshot,omitempty" gorm:"column:fee_rule_gbk_is_active_snapshot;type:boolean"`

	// Timestamps
	FeeRuleCreatedAt time.Time      `json:"fee_rule_created_at" gorm:"column:fee_rule_created_at;type:timestamptz;not null;autoCreateTime"`
	FeeRuleUpdatedAt time.Time      `json:"fee_rule_updated_at" gorm:"column:fee_rule_updated_at;type:timestamptz;not null;autoUpdateTime"`
	FeeRuleDeletedAt gorm.DeletedAt `json:"fee_rule_deleted_at,omitempty" gorm:"column:fee_rule_deleted_at;type:timestamptz;index"`
}

func (FeeRule) TableName() string { return "fee_rules" }
