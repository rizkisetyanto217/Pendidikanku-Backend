// file: internals/features/finance/general_billings/dto/general_billing_dto.go
package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "masjidku_backend/internals/features/finance/general_billings/model"
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
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Value = &v
	return nil
}

/* =========================================================
   REQUEST: Create
   ========================================================= */

type CreateGeneralBillingRequest struct {
	GeneralBillingMasjidID uuid.UUID `json:"general_billing_masjid_id" validate:"required"`
	GeneralBillingKindID   uuid.UUID `json:"general_billing_kind_id"   validate:"required"`

	GeneralBillingCode  *string `json:"general_billing_code"  validate:"omitempty,max=60"`
	GeneralBillingTitle string  `json:"general_billing_title" validate:"required"`
	GeneralBillingDesc  *string `json:"general_billing_desc"`

	GeneralBillingClassID   *uuid.UUID `json:"general_billing_class_id"`
	GeneralBillingSectionID *uuid.UUID `json:"general_billing_section_id"`
	GeneralBillingTermID    *uuid.UUID `json:"general_billing_term_id"`

	// "YYYY-MM-DD"
	GeneralBillingDueDate *string `json:"general_billing_due_date" validate:"omitempty,datetime=2006-01-02"`

	GeneralBillingIsActive         *bool `json:"general_billing_is_active"`
	GeneralBillingDefaultAmountIdr *int  `json:"general_billing_default_amount_idr" validate:"omitempty,min=0"`

	// Snapshots (MINIMAL)
	GeneralBillingKindSnapshot    *model.GeneralBillingKindSnapshotPayload    `json:"general_billing_kind_snapshot"`
	GeneralBillingClassSnapshot   *model.GeneralBillingClassSnapshotPayload   `json:"general_billing_class_snapshot"`
	GeneralBillingSectionSnapshot *model.GeneralBillingSectionSnapshotPayload `json:"general_billing_section_snapshot"`
	GeneralBillingTermSnapshot    *model.GeneralBillingTermSnapshotPayload    `json:"general_billing_term_snapshot"`
}

func (r *CreateGeneralBillingRequest) ToModel() (*model.GeneralBilling, error) {
	gb := &model.GeneralBilling{
		GeneralBillingMasjidID:  r.GeneralBillingMasjidID,
		GeneralBillingKindID:    r.GeneralBillingKindID,
		GeneralBillingCode:      r.GeneralBillingCode,
		GeneralBillingTitle:     r.GeneralBillingTitle,
		GeneralBillingDesc:      r.GeneralBillingDesc,
		GeneralBillingClassID:   r.GeneralBillingClassID,
		GeneralBillingSectionID: r.GeneralBillingSectionID,
		GeneralBillingTermID:    r.GeneralBillingTermID,
		GeneralBillingIsActive:  true, // default true
	}

	if r.GeneralBillingIsActive != nil {
		gb.GeneralBillingIsActive = *r.GeneralBillingIsActive
	}
	if r.GeneralBillingDefaultAmountIdr != nil {
		gb.GeneralBillingDefaultAmountIdr = r.GeneralBillingDefaultAmountIdr
	}
	if r.GeneralBillingDueDate != nil && *r.GeneralBillingDueDate != "" {
		t, err := parseDateYMD(*r.GeneralBillingDueDate)
		if err != nil {
			return nil, err
		}
		gb.GeneralBillingDueDate = &t
	}

	// snapshots → JSONB
	if r.GeneralBillingKindSnapshot != nil {
		if b, err := json.Marshal(r.GeneralBillingKindSnapshot); err == nil {
			gb.GeneralBillingKindSnapshot = datatypes.JSON(b)
		} else {
			return nil, err
		}
	}
	if r.GeneralBillingClassSnapshot != nil {
		if b, err := json.Marshal(r.GeneralBillingClassSnapshot); err == nil {
			gb.GeneralBillingClassSnapshot = datatypes.JSON(b)
		} else {
			return nil, err
		}
	}
	if r.GeneralBillingSectionSnapshot != nil {
		if b, err := json.Marshal(r.GeneralBillingSectionSnapshot); err == nil {
			gb.GeneralBillingSectionSnapshot = datatypes.JSON(b)
		} else {
			return nil, err
		}
	}
	if r.GeneralBillingTermSnapshot != nil {
		if b, err := json.Marshal(r.GeneralBillingTermSnapshot); err == nil {
			gb.GeneralBillingTermSnapshot = datatypes.JSON(b)
		} else {
			return nil, err
		}
	}

	return gb, nil
}

/* =========================================================
   REQUEST: Patch (Partial Update)
   ========================================================= */

type PatchGeneralBillingRequest struct {
	GeneralBillingMasjidID PatchField[uuid.UUID] `json:"general_billing_masjid_id"`
	GeneralBillingKindID   PatchField[uuid.UUID] `json:"general_billing_kind_id"`

	GeneralBillingCode  PatchField[string] `json:"general_billing_code"`
	GeneralBillingTitle PatchField[string] `json:"general_billing_title"`
	GeneralBillingDesc  PatchField[string] `json:"general_billing_desc"`

	GeneralBillingClassID   PatchField[uuid.UUID] `json:"general_billing_class_id"`
	GeneralBillingSectionID PatchField[uuid.UUID] `json:"general_billing_section_id"`
	GeneralBillingTermID    PatchField[uuid.UUID] `json:"general_billing_term_id"`

	GeneralBillingDueDate PatchField[string] `json:"general_billing_due_date"` // "YYYY-MM-DD"

	GeneralBillingIsActive         PatchField[bool] `json:"general_billing_is_active"`
	GeneralBillingDefaultAmountIdr PatchField[int]  `json:"general_billing_default_amount_idr"`

	GeneralBillingKindSnapshot    PatchField[model.GeneralBillingKindSnapshotPayload]    `json:"general_billing_kind_snapshot"`
	GeneralBillingClassSnapshot   PatchField[model.GeneralBillingClassSnapshotPayload]   `json:"general_billing_class_snapshot"`
	GeneralBillingSectionSnapshot PatchField[model.GeneralBillingSectionSnapshotPayload] `json:"general_billing_section_snapshot"`
	GeneralBillingTermSnapshot    PatchField[model.GeneralBillingTermSnapshotPayload]    `json:"general_billing_term_snapshot"`
}

func (p *PatchGeneralBillingRequest) ApplyTo(gb *model.GeneralBilling) error {
	// Tenant & Kind
	if p.GeneralBillingMasjidID.Set && !p.GeneralBillingMasjidID.Null {
		gb.GeneralBillingMasjidID = *p.GeneralBillingMasjidID.Value
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

	// UUID refs
	if p.GeneralBillingClassID.Set {
		if p.GeneralBillingClassID.Null {
			gb.GeneralBillingClassID = nil
		} else {
			gb.GeneralBillingClassID = p.GeneralBillingClassID.Value
		}
	}
	if p.GeneralBillingSectionID.Set {
		if p.GeneralBillingSectionID.Null {
			gb.GeneralBillingSectionID = nil
		} else {
			gb.GeneralBillingSectionID = p.GeneralBillingSectionID.Value
		}
	}
	if p.GeneralBillingTermID.Set {
		if p.GeneralBillingTermID.Null {
			gb.GeneralBillingTermID = nil
		} else {
			gb.GeneralBillingTermID = p.GeneralBillingTermID.Value
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
	if p.GeneralBillingDefaultAmountIdr.Set {
		if p.GeneralBillingDefaultAmountIdr.Null {
			gb.GeneralBillingDefaultAmountIdr = nil
		} else {
			gb.GeneralBillingDefaultAmountIdr = p.GeneralBillingDefaultAmountIdr.Value
		}
	}

	// Snapshots → JSONB
	if p.GeneralBillingKindSnapshot.Set {
		if p.GeneralBillingKindSnapshot.Null {
			gb.GeneralBillingKindSnapshot = nil
		} else if b, err := json.Marshal(p.GeneralBillingKindSnapshot.Value); err == nil {
			gb.GeneralBillingKindSnapshot = datatypes.JSON(b)
		} else {
			return err
		}
	}
	if p.GeneralBillingClassSnapshot.Set {
		if p.GeneralBillingClassSnapshot.Null {
			gb.GeneralBillingClassSnapshot = nil
		} else if b, err := json.Marshal(p.GeneralBillingClassSnapshot.Value); err == nil {
			gb.GeneralBillingClassSnapshot = datatypes.JSON(b)
		} else {
			return err
		}
	}
	if p.GeneralBillingSectionSnapshot.Set {
		if p.GeneralBillingSectionSnapshot.Null {
			gb.GeneralBillingSectionSnapshot = nil
		} else if b, err := json.Marshal(p.GeneralBillingSectionSnapshot.Value); err == nil {
			gb.GeneralBillingSectionSnapshot = datatypes.JSON(b)
		} else {
			return err
		}
	}
	if p.GeneralBillingTermSnapshot.Set {
		if p.GeneralBillingTermSnapshot.Null {
			gb.GeneralBillingTermSnapshot = nil
		} else if b, err := json.Marshal(p.GeneralBillingTermSnapshot.Value); err == nil {
			gb.GeneralBillingTermSnapshot = datatypes.JSON(b)
		} else {
			return err
		}
	}

	return nil
}

/* =========================================================
   RESPONSE
   ========================================================= */

type GeneralBillingResponse struct {
	GeneralBillingID uuid.UUID `json:"general_billing_id"`

	GeneralBillingMasjidID uuid.UUID `json:"general_billing_masjid_id"`
	GeneralBillingKindID   uuid.UUID `json:"general_billing_kind_id"`

	GeneralBillingCode  *string `json:"general_billing_code,omitempty"`
	GeneralBillingTitle string  `json:"general_billing_title"`
	GeneralBillingDesc  *string `json:"general_billing_desc,omitempty"`

	GeneralBillingClassID   *uuid.UUID `json:"general_billing_class_id,omitempty"`
	GeneralBillingSectionID *uuid.UUID `json:"general_billing_section_id,omitempty"`
	GeneralBillingTermID    *uuid.UUID `json:"general_billing_term_id,omitempty"`

	GeneralBillingDueDate  *string `json:"general_billing_due_date,omitempty"` // "YYYY-MM-DD"
	GeneralBillingIsActive bool    `json:"general_billing_is_active"`

	GeneralBillingDefaultAmountIdr *int `json:"general_billing_default_amount_idr,omitempty"`

	GeneralBillingKindSnapshot    *model.GeneralBillingKindSnapshotPayload    `json:"general_billing_kind_snapshot,omitempty"`
	GeneralBillingClassSnapshot   *model.GeneralBillingClassSnapshotPayload   `json:"general_billing_class_snapshot,omitempty"`
	GeneralBillingSectionSnapshot *model.GeneralBillingSectionSnapshotPayload `json:"general_billing_section_snapshot,omitempty"`
	GeneralBillingTermSnapshot    *model.GeneralBillingTermSnapshotPayload    `json:"general_billing_term_snapshot,omitempty"`

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

	resp := &GeneralBillingResponse{
		GeneralBillingID:               m.GeneralBillingID,
		GeneralBillingMasjidID:         m.GeneralBillingMasjidID,
		GeneralBillingKindID:           m.GeneralBillingKindID,
		GeneralBillingCode:             m.GeneralBillingCode,
		GeneralBillingTitle:            m.GeneralBillingTitle,
		GeneralBillingDesc:             m.GeneralBillingDesc,
		GeneralBillingClassID:          m.GeneralBillingClassID,
		GeneralBillingSectionID:        m.GeneralBillingSectionID,
		GeneralBillingTermID:           m.GeneralBillingTermID,
		GeneralBillingDueDate:          due,
		GeneralBillingIsActive:         m.GeneralBillingIsActive,
		GeneralBillingDefaultAmountIdr: m.GeneralBillingDefaultAmountIdr,
		GeneralBillingCreatedAt:        m.GeneralBillingCreatedAt,
		GeneralBillingUpdatedAt:        m.GeneralBillingUpdatedAt,
		GeneralBillingDeletedAt:        m.GeneralBillingDeletedAt,
	}

	// decode snapshots (abaikan error)
	var kind model.GeneralBillingKindSnapshotPayload
	if len(m.GeneralBillingKindSnapshot) > 0 && json.Unmarshal(m.GeneralBillingKindSnapshot, &kind) == nil {
		resp.GeneralBillingKindSnapshot = &kind
	}
	var class model.GeneralBillingClassSnapshotPayload
	if len(m.GeneralBillingClassSnapshot) > 0 && json.Unmarshal(m.GeneralBillingClassSnapshot, &class) == nil {
		resp.GeneralBillingClassSnapshot = &class
	}
	var section model.GeneralBillingSectionSnapshotPayload
	if len(m.GeneralBillingSectionSnapshot) > 0 && json.Unmarshal(m.GeneralBillingSectionSnapshot, &section) == nil {
		resp.GeneralBillingSectionSnapshot = &section
	}
	var term model.GeneralBillingTermSnapshotPayload
	if len(m.GeneralBillingTermSnapshot) > 0 && json.Unmarshal(m.GeneralBillingTermSnapshot, &term) == nil {
		resp.GeneralBillingTermSnapshot = &term
	}

	return resp
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
