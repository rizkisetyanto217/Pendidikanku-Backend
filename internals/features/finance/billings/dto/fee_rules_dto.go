// File: internal/dto/dto_and_mapper.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	// ✅ Model FeeRule yang benar (bukan billings)
	fee "madinahsalam_backend/internals/features/finance/billings/model"
)

////////////////////////////////////////////////////////////////////////////////
// ENUMS (samakan dengan enum DB)
////////////////////////////////////////////////////////////////////////////////

type FeeScope string

const (
	FeeScopeTenant      FeeScope = "tenant"
	FeeScopeClassParent FeeScope = "class_parent"
	FeeScopeClass       FeeScope = "class"
	FeeScopeSection     FeeScope = "section"
	FeeScopeStudent     FeeScope = "student"
	FeeScopeTerm        FeeScope = "term"
)

// Mirror enum general_billing_category di DB
type GeneralBillingCategory = fee.GeneralBillingCategory

////////////////////////////////////////////////////////////////////////////////
// COMMON
////////////////////////////////////////////////////////////////////////////////

type Pagination struct {
	Page     int `json:"page" validate:"min=1"`
	PageSize int `json:"page_size" validate:"min=1,max=200"`
}

type PagedResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

////////////////////////////////////////////////////////////////////////////////
// FEE RULES — DTO
////////////////////////////////////////////////////////////////////////////////

// Opsi harga (mirror dari item JSONB {code,label,amount})
type AmountOptionDTO struct {
	Code   string `json:"code" validate:"required,max=20"`
	Label  string `json:"label" validate:"required,max=60"`
	Amount int    `json:"amount" validate:"required,min=1"`
}

// Create
type FeeRuleCreateDTO struct {
	FeeRuleSchoolID uuid.UUID `json:"fee_rule_school_id" validate:"required"`
	FeeRuleScope    FeeScope  `json:"fee_rule_scope" validate:"required,oneof=tenant class_parent class section student term"`

	FeeRuleClassParentID   *uuid.UUID `json:"fee_rule_class_parent_id,omitempty"`
	FeeRuleClassID         *uuid.UUID `json:"fee_rule_class_id,omitempty"`
	FeeRuleSectionID       *uuid.UUID `json:"fee_rule_section_id,omitempty"`
	FeeRuleSchoolStudentID *uuid.UUID `json:"fee_rule_school_student_id,omitempty"`

	// Periode (pilih salah satu: term_id ATAU year+month)
	FeeRuleTermID *uuid.UUID `json:"fee_rule_term_id,omitempty"`
	FeeRuleMonth  *int16     `json:"fee_rule_month,omitempty" validate:"omitempty,min=1,max=12"`
	FeeRuleYear   *int16     `json:"fee_rule_year,omitempty"  validate:"omitempty,min=2000,max=2100"`

	// Kategori + bill_code (tanpa GBK)
	FeeRuleCategory    GeneralBillingCategory `json:"fee_rule_category" validate:"required,oneof=registration spp mass_student donation"`
	FeeRuleBillCode    *string                `json:"fee_rule_bill_code,omitempty" validate:"omitempty,max=60"` // default DB: 'SPP'
	FeeRuleOptionCode  *string                `json:"fee_rule_option_code,omitempty" validate:"omitempty,max=20"`
	FeeRuleOptionLabel *string                `json:"fee_rule_option_label,omitempty" validate:"omitempty,max=60"`

	// Prioritas antar-rule
	FeeRuleIsDefault bool `json:"fee_rule_is_default"`

	// >>> Daftar opsi harga (JSONB)
	FeeRuleAmountOptions []AmountOptionDTO `json:"fee_rule_amount_options" validate:"required,min=1,dive"`

	// Effective window
	FeeRuleEffectiveFrom *time.Time `json:"fee_rule_effective_from,omitempty"`
	FeeRuleEffectiveTo   *time.Time `json:"fee_rule_effective_to,omitempty"`

	FeeRuleNote *string `json:"fee_rule_note,omitempty"`
}

// Update (partial)
type FeeRuleUpdateDTO struct {
	FeeRuleScope           *FeeScope               `json:"fee_rule_scope,omitempty"`
	FeeRuleClassParentID   *uuid.UUID              `json:"fee_rule_class_parent_id,omitempty"`
	FeeRuleClassID         *uuid.UUID              `json:"fee_rule_class_id,omitempty"`
	FeeRuleSectionID       *uuid.UUID              `json:"fee_rule_section_id,omitempty"`
	FeeRuleSchoolStudentID *uuid.UUID              `json:"fee_rule_school_student_id,omitempty"`
	FeeRuleCategory        *GeneralBillingCategory `json:"fee_rule_category,omitempty"`

	FeeRuleTermID *uuid.UUID `json:"fee_rule_term_id,omitempty"`
	FeeRuleMonth  *int16     `json:"fee_rule_month,omitempty"`
	FeeRuleYear   *int16     `json:"fee_rule_year,omitempty"`

	FeeRuleBillCode      *string            `json:"fee_rule_bill_code,omitempty"`
	FeeRuleOptionCode    *string            `json:"fee_rule_option_code,omitempty"`
	FeeRuleOptionLabel   *string            `json:"fee_rule_option_label,omitempty"`
	FeeRuleIsDefault     *bool              `json:"fee_rule_is_default,omitempty"`
	FeeRuleAmountOptions *[]AmountOptionDTO `json:"fee_rule_amount_options,omitempty"`

	FeeRuleEffectiveFrom *time.Time `json:"fee_rule_effective_from,omitempty"`
	FeeRuleEffectiveTo   *time.Time `json:"fee_rule_effective_to,omitempty"`

	FeeRuleNote *string `json:"fee_rule_note,omitempty"`
}

// Response (tanpa GBK snapshot karena sudah dihapus di schema baru)
type FeeRuleResponse struct {
	FeeRuleID              uuid.UUID  `json:"fee_rule_id"`
	FeeRuleSchoolID        uuid.UUID  `json:"fee_rule_school_id"`
	FeeRuleScope           FeeScope   `json:"fee_rule_scope"`
	FeeRuleClassParentID   *uuid.UUID `json:"fee_rule_class_parent_id,omitempty"`
	FeeRuleClassID         *uuid.UUID `json:"fee_rule_class_id,omitempty"`
	FeeRuleSectionID       *uuid.UUID `json:"fee_rule_section_id,omitempty"`
	FeeRuleSchoolStudentID *uuid.UUID `json:"fee_rule_school_student_id,omitempty"`

	FeeRuleTermID *uuid.UUID `json:"fee_rule_term_id,omitempty"`
	FeeRuleMonth  *int16     `json:"fee_rule_month,omitempty"`
	FeeRuleYear   *int16     `json:"fee_rule_year,omitempty"`

	FeeRuleCategory GeneralBillingCategory `json:"fee_rule_category"`
	FeeRuleBillCode string                 `json:"fee_rule_bill_code"`

	FeeRuleOptionCode  *string `json:"fee_rule_option_code,omitempty"`
	FeeRuleOptionLabel *string `json:"fee_rule_option_label,omitempty"`
	FeeRuleIsDefault   bool    `json:"fee_rule_is_default"`

	// >>> Daftar opsi harga (mirror JSONB)
	FeeRuleAmountOptions []AmountOptionDTO `json:"fee_rule_amount_options"`

	FeeRuleEffectiveFrom *time.Time `json:"fee_rule_effective_from,omitempty"`
	FeeRuleEffectiveTo   *time.Time `json:"fee_rule_effective_to,omitempty"`

	FeeRuleNote *string `json:"fee_rule_note,omitempty"`

	FeeRuleCreatedAt time.Time  `json:"fee_rule_created_at"`
	FeeRuleUpdatedAt time.Time  `json:"fee_rule_updated_at"`
	FeeRuleDeletedAt *time.Time `json:"fee_rule_deleted_at,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////
// GENERATE STUDENT BILLS — DTO (tetap)
////////////////////////////////////////////////////////////////////////////////

type AmountStrategyDTO struct {
	Mode string `json:"mode" validate:"required,oneof=rule_fallback_fixed fixed"`

	PreferRule *struct {
		By         string `json:"by" validate:"required,oneof=term ym"`
		OptionCode string `json:"option_code" validate:"required"`
	} `json:"prefer_rule,omitempty"`

	FixedAmountIDR *int `json:"fixed_amount_idr,omitempty" validate:"omitempty,min=0"`
}

type SourceDTO struct {
	Type             string      `json:"type" validate:"required,oneof=class section students"`
	ClassID          *uuid.UUID  `json:"class_id,omitempty"`
	SectionID        *uuid.UUID  `json:"section_id,omitempty"`
	SchoolStudentIDs []uuid.UUID `json:"school_student_ids,omitempty"`
}

type LabelingDTO struct {
	OptionCode  string  `json:"option_code" validate:"required"`
	OptionLabel *string `json:"option_label,omitempty"`
}

type GenerateStudentBillsRequest struct {
	BillBatchID         uuid.UUID `json:"bill_batch_id" validate:"required"`
	StudentBillSchoolID uuid.UUID `json:"student_bill_school_id" validate:"required"`

	Source         SourceDTO         `json:"source" validate:"required"`
	AmountStrategy AmountStrategyDTO `json:"amount_strategy" validate:"required"`
	Labeling       LabelingDTO       `json:"labeling" validate:"required"`

	Filters *struct {
		OnlyActiveStudents bool `json:"only_active_students"`
	} `json:"filters,omitempty"`

	IdempotencyKey *string `json:"idempotency_key,omitempty"`
}

type GenerateStudentBillsResponse struct {
	BillBatchID uuid.UUID `json:"bill_batch_id"`
	Inserted    int       `json:"inserted"`
	Skipped     int       `json:"skipped"`
}

////////////////////////////////////////////////////////////////////////////////
// MAPPERS — Model <-> DTO
////////////////////////////////////////////////////////////////////////////////

func deletedAtToTimePtr(d gorm.DeletedAt) *time.Time {
	if d.Valid {
		t := d.Time
		return &t
	}
	return nil
}

func mapAmountOptionsModelToDTO(src []fee.AmountOption) []AmountOptionDTO {
	if len(src) == 0 {
		return nil
	}
	out := make([]AmountOptionDTO, 0, len(src))
	for _, it := range src {
		out = append(out, AmountOptionDTO{
			Code:   it.Code,
			Label:  it.Label,
			Amount: it.Amount,
		})
	}
	return out
}

func mapAmountOptionsDTOToModel(src []AmountOptionDTO) []fee.AmountOption {
	if len(src) == 0 {
		return nil
	}
	out := make([]fee.AmountOption, 0, len(src))
	for _, it := range src {
		out = append(out, fee.AmountOption{
			Code:   it.Code,
			Label:  it.Label,
			Amount: it.Amount,
		})
	}
	return out
}

// Model -> Response
func ToFeeRuleResponse(m fee.FeeRuleModel) FeeRuleResponse {
	return FeeRuleResponse{
		FeeRuleID:              m.FeeRuleID,
		FeeRuleSchoolID:        m.FeeRuleSchoolID,
		FeeRuleScope:           FeeScope(m.FeeRuleScope),
		FeeRuleClassParentID:   m.FeeRuleClassParentID,
		FeeRuleClassID:         m.FeeRuleClassID,
		FeeRuleSectionID:       m.FeeRuleSectionID,
		FeeRuleSchoolStudentID: m.FeeRuleSchoolStudentID,

		FeeRuleTermID: m.FeeRuleTermID,
		FeeRuleMonth:  m.FeeRuleMonth,
		FeeRuleYear:   m.FeeRuleYear,

		FeeRuleCategory: m.FeeRuleCategory,
		FeeRuleBillCode: m.FeeRuleBillCode,

		FeeRuleOptionCode:  strPtrOrNil(m.FeeRuleOptionCode),
		FeeRuleOptionLabel: m.FeeRuleOptionLabel,
		FeeRuleIsDefault:   m.FeeRuleIsDefault,

		FeeRuleAmountOptions: mapAmountOptionsModelToDTO(m.FeeRuleAmountOptions),

		FeeRuleEffectiveFrom: m.FeeRuleEffectiveFrom,
		FeeRuleEffectiveTo:   m.FeeRuleEffectiveTo,
		FeeRuleNote:          m.FeeRuleNote,

		FeeRuleCreatedAt: m.FeeRuleCreatedAt,
		FeeRuleUpdatedAt: m.FeeRuleUpdatedAt,
		FeeRuleDeletedAt: deletedAtToTimePtr(m.FeeRuleDeletedAt),
	}
}

// CreateDTO -> Model
func FeeRuleCreateDTOToModel(d FeeRuleCreateDTO) fee.FeeRuleModel {
	var billCode string
	if d.FeeRuleBillCode != nil && *d.FeeRuleBillCode != "" {
		billCode = *d.FeeRuleBillCode
	}

	var optCode string
	if d.FeeRuleOptionCode != nil {
		optCode = *d.FeeRuleOptionCode
	}

	return fee.FeeRuleModel{
		FeeRuleSchoolID:        d.FeeRuleSchoolID,
		FeeRuleScope:           fee.FeeScope(d.FeeRuleScope),
		FeeRuleClassParentID:   d.FeeRuleClassParentID,
		FeeRuleClassID:         d.FeeRuleClassID,
		FeeRuleSectionID:       d.FeeRuleSectionID,
		FeeRuleSchoolStudentID: d.FeeRuleSchoolStudentID,

		FeeRuleTermID: d.FeeRuleTermID,
		FeeRuleMonth:  d.FeeRuleMonth,
		FeeRuleYear:   d.FeeRuleYear,

		FeeRuleCategory: d.FeeRuleCategory,
		FeeRuleBillCode: billCode,

		FeeRuleOptionCode:  optCode,
		FeeRuleOptionLabel: d.FeeRuleOptionLabel,
		FeeRuleIsDefault:   d.FeeRuleIsDefault,

		FeeRuleAmountOptions: mapAmountOptionsDTOToModel(d.FeeRuleAmountOptions),

		FeeRuleEffectiveFrom: d.FeeRuleEffectiveFrom,
		FeeRuleEffectiveTo:   d.FeeRuleEffectiveTo,
		FeeRuleNote:          d.FeeRuleNote,
	}
}

// UpdateDTO -> apply ke Model (partial)
func ApplyFeeRuleUpdate(m *fee.FeeRuleModel, d FeeRuleUpdateDTO) {
	if d.FeeRuleScope != nil {
		m.FeeRuleScope = fee.FeeScope(*d.FeeRuleScope)
	}
	if d.FeeRuleClassParentID != nil {
		m.FeeRuleClassParentID = d.FeeRuleClassParentID
	}
	if d.FeeRuleClassID != nil {
		m.FeeRuleClassID = d.FeeRuleClassID
	}
	if d.FeeRuleSectionID != nil {
		m.FeeRuleSectionID = d.FeeRuleSectionID
	}
	if d.FeeRuleSchoolStudentID != nil {
		m.FeeRuleSchoolStudentID = d.FeeRuleSchoolStudentID
	}
	if d.FeeRuleCategory != nil {
		m.FeeRuleCategory = *d.FeeRuleCategory
	}
	if d.FeeRuleTermID != nil {
		m.FeeRuleTermID = d.FeeRuleTermID
	}
	if d.FeeRuleMonth != nil {
		m.FeeRuleMonth = d.FeeRuleMonth
	}
	if d.FeeRuleYear != nil {
		m.FeeRuleYear = d.FeeRuleYear
	}
	if d.FeeRuleBillCode != nil {
		m.FeeRuleBillCode = *d.FeeRuleBillCode
	}
	if d.FeeRuleOptionCode != nil {
		m.FeeRuleOptionCode = *d.FeeRuleOptionCode
	}
	if d.FeeRuleOptionLabel != nil {
		m.FeeRuleOptionLabel = d.FeeRuleOptionLabel
	}
	if d.FeeRuleIsDefault != nil {
		m.FeeRuleIsDefault = *d.FeeRuleIsDefault
	}
	if d.FeeRuleAmountOptions != nil {
		m.FeeRuleAmountOptions = mapAmountOptionsDTOToModel(*d.FeeRuleAmountOptions)
	}
	if d.FeeRuleEffectiveFrom != nil {
		m.FeeRuleEffectiveFrom = d.FeeRuleEffectiveFrom
	}
	if d.FeeRuleEffectiveTo != nil {
		m.FeeRuleEffectiveTo = d.FeeRuleEffectiveTo
	}
	if d.FeeRuleNote != nil {
		m.FeeRuleNote = d.FeeRuleNote
	}
}

// List helper
func ToFeeRuleResponses(list []fee.FeeRuleModel) []FeeRuleResponse {
	out := make([]FeeRuleResponse, 0, len(list))
	for _, v := range list {
		out = append(out, ToFeeRuleResponse(v))
	}
	return out
}

func strPtrOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
