// file: internals/features/finance/general_billings/dto/general_billing_dto.go
package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	model "madinahsalam_backend/internals/features/finance/general_billings/model"
)

/* =========================================================
   PatchField tri-state (Unset / Null / Set(value))
   ========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"-"`
	Null  bool `json:"-"`
	Value *T   `json:"-"`
}

func (p *PatchField[T]) UnmarshalJSON(b []byte) error {
	p.Set = true
	if string(b) == "null" {
		p.Null = true
		p.Value = nil
		return nil
	}
	var v T
	// use std json to decode generic value
	type alias = T
	var vv alias
	if err := json.Unmarshal(b, &vv); err != nil {
		return err
	}
	v = T(vv)
	p.Value = &v
	return nil
}

/* =========================================================
   REQUEST: Create (school_id optional untuk GLOBAL)
   ========================================================= */

type CreateGeneralBillingRequest struct {
	GeneralBillingSchoolID *uuid.UUID `json:"general_billing_school_id,omitempty"` // NULL = GLOBAL
	GeneralBillingKindID   uuid.UUID  `json:"general_billing_kind_id" validate:"required"`

	GeneralBillingCode  *string `json:"general_billing_code"  validate:"omitempty,max=60"`
	GeneralBillingTitle string  `json:"general_billing_title" validate:"required"`
	GeneralBillingDesc  *string `json:"general_billing_desc"`

	// "YYYY-MM-DD"
	GeneralBillingDueDate *string `json:"general_billing_due_date" validate:"omitempty,datetime=2006-01-02"`

	GeneralBillingIsActive         *bool `json:"general_billing_is_active"`
	GeneralBillingDefaultAmountIDR *int  `json:"general_billing_default_amount_idr" validate:"omitempty,min=0"`
}

func (r *CreateGeneralBillingRequest) ToModel() (*model.GeneralBilling, error) {
	gb := &model.GeneralBilling{
		GeneralBillingSchoolID: r.GeneralBillingSchoolID,
		GeneralBillingKindID:   r.GeneralBillingKindID,
		GeneralBillingCode:     r.GeneralBillingCode,
		GeneralBillingTitle:    r.GeneralBillingTitle,
		GeneralBillingDesc:     r.GeneralBillingDesc,
		GeneralBillingIsActive: true, // default true
	}
	if r.GeneralBillingIsActive != nil {
		gb.GeneralBillingIsActive = *r.GeneralBillingIsActive
	}
	if r.GeneralBillingDefaultAmountIDR != nil {
		gb.GeneralBillingDefaultAmountIDR = r.GeneralBillingDefaultAmountIDR
	}
	if r.GeneralBillingDueDate != nil && *r.GeneralBillingDueDate != "" {
		t, err := parseDateYMD(*r.GeneralBillingDueDate)
		if err != nil {
			return nil, err
		}
		gb.GeneralBillingDueDate = &t
	}
	return gb, nil
}

/* =========================================================
   REQUEST: Patch (Partial Update, tri-state)
   ========================================================= */

type PatchGeneralBillingRequest struct {
	GeneralBillingSchoolID PatchField[uuid.UUID] `json:"general_billing_school_id"` // null => GLOBAL
	GeneralBillingKindID   PatchField[uuid.UUID] `json:"general_billing_kind_id"`

	GeneralBillingCode  PatchField[string] `json:"general_billing_code"`
	GeneralBillingTitle PatchField[string] `json:"general_billing_title"`
	GeneralBillingDesc  PatchField[string] `json:"general_billing_desc"`

	GeneralBillingDueDate PatchField[string] `json:"general_billing_due_date"` // "YYYY-MM-DD"

	GeneralBillingIsActive         PatchField[bool] `json:"general_billing_is_active"`
	GeneralBillingDefaultAmountIDR PatchField[int]  `json:"general_billing_default_amount_idr"`
}

func (p *PatchGeneralBillingRequest) ApplyTo(gb *model.GeneralBilling) error {
	// Tenant & Kind
	if p.GeneralBillingSchoolID.Set {
		if p.GeneralBillingSchoolID.Null {
			gb.GeneralBillingSchoolID = nil
		} else {
			gb.GeneralBillingSchoolID = p.GeneralBillingSchoolID.Value
		}
	}
	if p.GeneralBillingKindID.Set && !p.GeneralBillingKindID.Null {
		gb.GeneralBillingKindID = *p.GeneralBillingKindID.Value
	}

	// Strings
	if p.GeneralBillingCode.Set {
		if p.GeneralBillingCode.Null {
			gb.GeneralBillingCode = nil
		} else {
			gb.GeneralBillingCode = p.GeneralBillingCode.Value
		}
	}
	if p.GeneralBillingTitle.Set && !p.GeneralBillingTitle.Null {
		gb.GeneralBillingTitle = *p.GeneralBillingTitle.Value
	}
	if p.GeneralBillingDesc.Set {
		if p.GeneralBillingDesc.Null {
			gb.GeneralBillingDesc = nil
		} else {
			gb.GeneralBillingDesc = p.GeneralBillingDesc.Value
		}
	}

	// Due date
	if p.GeneralBillingDueDate.Set {
		if p.GeneralBillingDueDate.Null || p.GeneralBillingDueDate.Value == nil || *p.GeneralBillingDueDate.Value == "" {
			gb.GeneralBillingDueDate = nil
		} else {
			t, err := parseDateYMD(*p.GeneralBillingDueDate.Value)
			if err != nil {
				return err
			}
			gb.GeneralBillingDueDate = &t
		}
	}

	// IsActive
	if p.GeneralBillingIsActive.Set && !p.GeneralBillingIsActive.Null {
		gb.GeneralBillingIsActive = *p.GeneralBillingIsActive.Value
	}

	// Default amount
	if p.GeneralBillingDefaultAmountIDR.Set {
		if p.GeneralBillingDefaultAmountIDR.Null {
			gb.GeneralBillingDefaultAmountIDR = nil
		} else {
			gb.GeneralBillingDefaultAmountIDR = p.GeneralBillingDefaultAmountIDR.Value
		}
	}

	return nil
}

/* =========================================================
   RESPONSE
   ========================================================= */

type GeneralBillingResponse struct {
	GeneralBillingID uuid.UUID `json:"general_billing_id"`

	GeneralBillingSchoolID *uuid.UUID `json:"general_billing_school_id,omitempty"` // null = GLOBAL
	GeneralBillingKindID   uuid.UUID  `json:"general_billing_kind_id"`

	GeneralBillingCode  *string `json:"general_billing_code,omitempty"`
	GeneralBillingTitle string  `json:"general_billing_title"`
	GeneralBillingDesc  *string `json:"general_billing_desc,omitempty"`

	GeneralBillingDueDate  *string `json:"general_billing_due_date,omitempty"` // "YYYY-MM-DD"
	GeneralBillingIsActive bool    `json:"general_billing_is_active"`

	GeneralBillingDefaultAmountIDR *int `json:"general_billing_default_amount_idr,omitempty"`

	GeneralBillingCreatedAt time.Time  `json:"general_billing_created_at"`
	GeneralBillingUpdatedAt time.Time  `json:"general_billing_updated_at"`
	GeneralBillingDeletedAt *time.Time `json:"general_billing_deleted_at,omitempty"`
}

func FromModelGeneralBilling(m *model.GeneralBilling) *GeneralBillingResponse {
	// DATE -> "YYYY-MM-DD"
	var due *string
	if m.GeneralBillingDueDate != nil {
		s := m.GeneralBillingDueDate.Format("2006-01-02")
		due = &s
	}
	return &GeneralBillingResponse{
		GeneralBillingID:               m.GeneralBillingID,
		GeneralBillingSchoolID:         m.GeneralBillingSchoolID,
		GeneralBillingKindID:           m.GeneralBillingKindID,
		GeneralBillingCode:             m.GeneralBillingCode,
		GeneralBillingTitle:            m.GeneralBillingTitle,
		GeneralBillingDesc:             m.GeneralBillingDesc,
		GeneralBillingDueDate:          due,
		GeneralBillingIsActive:         m.GeneralBillingIsActive,
		GeneralBillingDefaultAmountIDR: m.GeneralBillingDefaultAmountIDR,
		GeneralBillingCreatedAt:        m.GeneralBillingCreatedAt,
		GeneralBillingUpdatedAt:        m.GeneralBillingUpdatedAt,
		GeneralBillingDeletedAt:        m.GeneralBillingDeletedAt,
	}
}

/* =========================================================
   Utils
   ========================================================= */

func parseDateYMD(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
