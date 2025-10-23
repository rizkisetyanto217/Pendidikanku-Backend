// File: internal/dto/dto_and_mapper.go
package dto

import (
	"time"

	"github.com/google/uuid"

	// Ganti dengan path model kamu
	billing "masjidku_backend/internals/features/finance/billings/model"
)

////////////////////////////////////////////////////////////////////////////////
// ENUMS
////////////////////////////////////////////////////////////////////////////////

type FeeScope string

const (
	FeeScopeTenant      FeeScope = "tenant"
	FeeScopeClassParent FeeScope = "class_parent"
	FeeScopeClass       FeeScope = "class"
	FeeScopeSection     FeeScope = "section"
	FeeScopeStudent     FeeScope = "student"
)

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

// Create
type FeeRuleCreateDTO struct {
	FeeRuleMasjidID uuid.UUID `json:"fee_rule_masjid_id" validate:"required"`
	FeeRuleScope    FeeScope  `json:"fee_rule_scope" validate:"required,oneof=tenant class_parent class section student"`

	FeeRuleClassParentID   *uuid.UUID `json:"fee_rule_class_parent_id,omitempty"`
	FeeRuleClassID         *uuid.UUID `json:"fee_rule_class_id,omitempty"`
	FeeRuleSectionID       *uuid.UUID `json:"fee_rule_section_id,omitempty"`
	FeeRuleMasjidStudentID *uuid.UUID `json:"fee_rule_masjid_student_id,omitempty"`

	FeeRuleTermID *uuid.UUID `json:"fee_rule_term_id,omitempty"`
	FeeRuleMonth  *int16     `json:"fee_rule_month,omitempty" validate:"omitempty,min=1,max=12"`
	FeeRuleYear   *int16     `json:"fee_rule_year,omitempty"  validate:"omitempty,min=2000,max=2100"`

	FeeRuleOptionCode  string  `json:"fee_rule_option_code" validate:"required,max=20"` // SPP/REG/BOOK/UNIFORM/...
	FeeRuleOptionLabel *string `json:"fee_rule_option_label,omitempty" validate:"omitempty,max=60"`

	FeeRuleIsDefault bool `json:"fee_rule_is_default"`
	FeeRuleAmountIDR int  `json:"fee_rule_amount_idr" validate:"required,min=0"`

	FeeRuleEffectiveFrom *time.Time `json:"fee_rule_effective_from,omitempty"`
	FeeRuleEffectiveTo   *time.Time `json:"fee_rule_effective_to,omitempty"`

	FeeRuleNote *string `json:"fee_rule_note,omitempty"`
}

// Update (partial)
type FeeRuleUpdateDTO struct {
	FeeRuleScope           *FeeScope  `json:"fee_rule_scope,omitempty"`
	FeeRuleClassParentID   *uuid.UUID `json:"fee_rule_class_parent_id,omitempty"`
	FeeRuleClassID         *uuid.UUID `json:"fee_rule_class_id,omitempty"`
	FeeRuleSectionID       *uuid.UUID `json:"fee_rule_section_id,omitempty"`
	FeeRuleMasjidStudentID *uuid.UUID `json:"fee_rule_masjid_student_id,omitempty"`

	FeeRuleTermID *uuid.UUID `json:"fee_rule_term_id,omitempty"`
	FeeRuleMonth  *int16     `json:"fee_rule_month,omitempty"`
	FeeRuleYear   *int16     `json:"fee_rule_year,omitempty"`

	FeeRuleOptionCode  *string `json:"fee_rule_option_code,omitempty"`
	FeeRuleOptionLabel *string `json:"fee_rule_option_label,omitempty"`

	FeeRuleIsDefault *bool `json:"fee_rule_is_default,omitempty"`
	FeeRuleAmountIDR *int  `json:"fee_rule_amount_idr,omitempty"`

	FeeRuleEffectiveFrom *time.Time `json:"fee_rule_effective_from,omitempty"`
	FeeRuleEffectiveTo   *time.Time `json:"fee_rule_effective_to,omitempty"`

	FeeRuleNote *string `json:"fee_rule_note,omitempty"`
}

// Response
type FeeRuleResponse struct {
	FeeRuleID              uuid.UUID  `json:"fee_rule_id"`
	FeeRuleMasjidID        uuid.UUID  `json:"fee_rule_masjid_id"`
	FeeRuleScope           FeeScope   `json:"fee_rule_scope"`
	FeeRuleClassParentID   *uuid.UUID `json:"fee_rule_class_parent_id,omitempty"`
	FeeRuleClassID         *uuid.UUID `json:"fee_rule_class_id,omitempty"`
	FeeRuleSectionID       *uuid.UUID `json:"fee_rule_section_id,omitempty"`
	FeeRuleMasjidStudentID *uuid.UUID `json:"fee_rule_masjid_student_id,omitempty"`

	FeeRuleTermID *uuid.UUID `json:"fee_rule_term_id,omitempty"`
	FeeRuleMonth  *int16     `json:"fee_rule_month,omitempty"`
	FeeRuleYear   *int16     `json:"fee_rule_year,omitempty"`

	FeeRuleOptionCode  string  `json:"fee_rule_option_code"`
	FeeRuleOptionLabel *string `json:"fee_rule_option_label,omitempty"`

	FeeRuleIsDefault bool `json:"fee_rule_is_default"`
	FeeRuleAmountIDR int  `json:"fee_rule_amount_idr"`

	FeeRuleEffectiveFrom *time.Time `json:"fee_rule_effective_from,omitempty"`
	FeeRuleEffectiveTo   *time.Time `json:"fee_rule_effective_to,omitempty"`

	FeeRuleNote *string `json:"fee_rule_note,omitempty"`

	FeeRuleCreatedAt time.Time  `json:"fee_rule_created_at"`
	FeeRuleUpdatedAt time.Time  `json:"fee_rule_updated_at"`
	FeeRuleDeletedAt *time.Time `json:"fee_rule_deleted_at,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////
// GENERATE STUDENT BILLS — DTO
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
	Type             string      `json:"type" validate:"required,oneof=class students"`
	ClassID          *uuid.UUID  `json:"class_id,omitempty"`
	MasjidStudentIDs []uuid.UUID `json:"masjid_student_ids,omitempty"`
}

type LabelingDTO struct {
	OptionCode  string  `json:"option_code" validate:"required"` // SPP/REG/BOOK/UNIFORM/...
	OptionLabel *string `json:"option_label,omitempty"`
}

type GenerateStudentBillsRequest struct {
	BillBatchID         uuid.UUID `json:"bill_batch_id" validate:"required"`
	StudentBillMasjidID uuid.UUID `json:"student_bill_masjid_id" validate:"required"`

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

// ---------------------- FeeRule ----------------------

// Model -> Response
func ToFeeRuleResponse(m billing.FeeRule) FeeRuleResponse {
	return FeeRuleResponse{
		FeeRuleID:              m.FeeRuleID,
		FeeRuleMasjidID:        m.FeeRuleMasjidID,
		FeeRuleScope:           FeeScope(m.FeeRuleScope),
		FeeRuleClassParentID:   m.FeeRuleClassParentID,
		FeeRuleClassID:         m.FeeRuleClassID,
		FeeRuleSectionID:       m.FeeRuleSectionID,
		FeeRuleMasjidStudentID: m.FeeRuleMasjidStudentID,
		FeeRuleTermID:          m.FeeRuleTermID,
		FeeRuleMonth:           m.FeeRuleMonth,
		FeeRuleYear:            m.FeeRuleYear,
		FeeRuleOptionCode:      m.FeeRuleOptionCode,
		FeeRuleOptionLabel:     m.FeeRuleOptionLabel,
		FeeRuleIsDefault:       m.FeeRuleIsDefault,
		FeeRuleAmountIDR:       m.FeeRuleAmountIDR,
		FeeRuleEffectiveFrom:   m.FeeRuleEffectiveFrom,
		FeeRuleEffectiveTo:     m.FeeRuleEffectiveTo,
		FeeRuleNote:            m.FeeRuleNote,
		FeeRuleCreatedAt:       m.FeeRuleCreatedAt,
		FeeRuleUpdatedAt:       m.FeeRuleUpdatedAt,
		FeeRuleDeletedAt:       toPtrTimeFromDeletedAt(m.FeeRuleDeletedAt),
	}
}

// CreateDTO -> Model
func FeeRuleCreateDTOToModel(d FeeRuleCreateDTO) billing.FeeRule {
	return billing.FeeRule{
		FeeRuleMasjidID:        d.FeeRuleMasjidID,
		FeeRuleScope:           billing.FeeScope(d.FeeRuleScope),
		FeeRuleClassParentID:   d.FeeRuleClassParentID,
		FeeRuleClassID:         d.FeeRuleClassID,
		FeeRuleSectionID:       d.FeeRuleSectionID,
		FeeRuleMasjidStudentID: d.FeeRuleMasjidStudentID,
		FeeRuleTermID:          d.FeeRuleTermID,
		FeeRuleMonth:           d.FeeRuleMonth,
		FeeRuleYear:            d.FeeRuleYear,
		FeeRuleOptionCode:      d.FeeRuleOptionCode,
		FeeRuleOptionLabel:     d.FeeRuleOptionLabel,
		FeeRuleIsDefault:       d.FeeRuleIsDefault,
		FeeRuleAmountIDR:       d.FeeRuleAmountIDR,
		FeeRuleEffectiveFrom:   d.FeeRuleEffectiveFrom,
		FeeRuleEffectiveTo:     d.FeeRuleEffectiveTo,
		FeeRuleNote:            d.FeeRuleNote,
	}
}

// UpdateDTO -> Model (apply partial)
func ApplyFeeRuleUpdate(m *billing.FeeRule, d FeeRuleUpdateDTO) {
	if d.FeeRuleScope != nil {
		m.FeeRuleScope = billing.FeeScope(*d.FeeRuleScope)
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
	if d.FeeRuleMasjidStudentID != nil {
		m.FeeRuleMasjidStudentID = d.FeeRuleMasjidStudentID
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
	if d.FeeRuleOptionCode != nil {
		m.FeeRuleOptionCode = *d.FeeRuleOptionCode
	}
	if d.FeeRuleOptionLabel != nil {
		m.FeeRuleOptionLabel = d.FeeRuleOptionLabel
	}
	if d.FeeRuleIsDefault != nil {
		m.FeeRuleIsDefault = *d.FeeRuleIsDefault
	}
	if d.FeeRuleAmountIDR != nil {
		m.FeeRuleAmountIDR = *d.FeeRuleAmountIDR
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

// ---------------------- BillBatch ----------------------

// Helpers list mapping
func ToFeeRuleResponses(list []billing.FeeRule) []FeeRuleResponse {
	out := make([]FeeRuleResponse, 0, len(list))
	for _, v := range list {
		out = append(out, ToFeeRuleResponse(v))
	}
	return out
}
