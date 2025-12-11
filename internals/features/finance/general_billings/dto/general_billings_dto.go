// file: internals/features/finance/general_billings/dto/general_billing_dto.go
package dto

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	model "madinahsalam_backend/internals/features/finance/general_billings/model"
	"madinahsalam_backend/internals/helpers/dbtime"
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
   Utils
========================================================= */

func parseDateYMD(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func intToInt16Ptr(p *int) *int16 {
	if p == nil {
		return nil
	}
	v := int16(*p)
	return &v
}

func int16PtrToIntPtr(p *int16) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

/* =========================================================
   REQUEST: Create
   - school_id WAJIB (nanti bisa dioverride dari token oleh controller)
========================================================= */

type CreateGeneralBillingRequest struct {
	GeneralBillingSchoolID uuid.UUID `json:"general_billing_school_id" validate:"required"`

	// kategori & bill code
	GeneralBillingCategory model.GeneralBillingCategory `json:"general_billing_category" validate:"required"` // registration|spp|mass_student|donation
	GeneralBillingBillCode string                       `json:"general_billing_bill_code" validate:"omitempty,max=60"`

	GeneralBillingCode  *string `json:"general_billing_code"  validate:"omitempty,max=60"`
	GeneralBillingTitle string  `json:"general_billing_title" validate:"required"`
	GeneralBillingDesc  *string `json:"general_billing_desc"`

	// Scope akademik (opsional)
	GeneralBillingClassID   *uuid.UUID `json:"general_billing_class_id"`
	GeneralBillingSectionID *uuid.UUID `json:"general_billing_section_id"`
	GeneralBillingTermID    *uuid.UUID `json:"general_billing_term_id"`

	// Periode (opsional, tapi biasanya wajib untuk SPP)
	GeneralBillingMonth *int `json:"general_billing_month" validate:"omitempty,min=1,max=12"`
	GeneralBillingYear  *int `json:"general_billing_year"  validate:"omitempty,min=2000,max=2100"`

	// "YYYY-MM-DD"
	GeneralBillingDueDate *string `json:"general_billing_due_date" validate:"omitempty,datetime=2006-01-02"`

	GeneralBillingIsActive         *bool `json:"general_billing_is_active"`
	GeneralBillingDefaultAmountIDR *int  `json:"general_billing_default_amount_idr" validate:"omitempty,min=0"`
}

func (r *CreateGeneralBillingRequest) ToModel() (*model.GeneralBillingModel, error) {
	gb := &model.GeneralBillingModel{
		GeneralBillingSchoolID: r.GeneralBillingSchoolID,
		GeneralBillingCategory: r.GeneralBillingCategory,
		// kalau kosong, default "SPP"
		GeneralBillingBillCode: func() string {
			if r.GeneralBillingBillCode == "" {
				return "SPP"
			}
			return r.GeneralBillingBillCode
		}(),
		GeneralBillingCode:  r.GeneralBillingCode,
		GeneralBillingTitle: r.GeneralBillingTitle,
		GeneralBillingDesc:  r.GeneralBillingDesc,

		GeneralBillingClassID:   r.GeneralBillingClassID,
		GeneralBillingSectionID: r.GeneralBillingSectionID,
		GeneralBillingTermID:    r.GeneralBillingTermID,

		GeneralBillingMonth: intToInt16Ptr(r.GeneralBillingMonth),
		GeneralBillingYear:  intToInt16Ptr(r.GeneralBillingYear),

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
	// school_id sengaja TIDAK dibuka di patch (tenant tidak boleh diganti)
	// tapi kalau mau, bisa tambahkan PatchField[uuid.UUID] di sini.

	GeneralBillingCategory PatchField[model.GeneralBillingCategory] `json:"general_billing_category"`
	GeneralBillingBillCode PatchField[string]                       `json:"general_billing_bill_code"`

	GeneralBillingCode  PatchField[string] `json:"general_billing_code"`
	GeneralBillingTitle PatchField[string] `json:"general_billing_title"`
	GeneralBillingDesc  PatchField[string] `json:"general_billing_desc"`

	GeneralBillingClassID   PatchField[uuid.UUID] `json:"general_billing_class_id"`
	GeneralBillingSectionID PatchField[uuid.UUID] `json:"general_billing_section_id"`
	GeneralBillingTermID    PatchField[uuid.UUID] `json:"general_billing_term_id"`

	GeneralBillingMonth PatchField[int] `json:"general_billing_month"`
	GeneralBillingYear  PatchField[int] `json:"general_billing_year"`

	GeneralBillingDueDate PatchField[string] `json:"general_billing_due_date"` // "YYYY-MM-DD"

	GeneralBillingIsActive         PatchField[bool] `json:"general_billing_is_active"`
	GeneralBillingDefaultAmountIDR PatchField[int]  `json:"general_billing_default_amount_idr"`
}

func (p *PatchGeneralBillingRequest) ApplyTo(gb *model.GeneralBillingModel) error {
	// Category
	if p.GeneralBillingCategory.Set && !p.GeneralBillingCategory.Null && p.GeneralBillingCategory.Value != nil {
		gb.GeneralBillingCategory = *p.GeneralBillingCategory.Value
	}

	// Bill code
	if p.GeneralBillingBillCode.Set {
		if p.GeneralBillingBillCode.Null {
			gb.GeneralBillingBillCode = "" // boleh dikosongkan, nanti bisa ditimpa di controller jika perlu
		} else if p.GeneralBillingBillCode.Value != nil {
			gb.GeneralBillingBillCode = *p.GeneralBillingBillCode.Value
		}
	}

	// Strings
	if p.GeneralBillingCode.Set {
		if p.GeneralBillingCode.Null {
			gb.GeneralBillingCode = nil
		} else {
			gb.GeneralBillingCode = p.GeneralBillingCode.Value
		}
	}
	if p.GeneralBillingTitle.Set && !p.GeneralBillingTitle.Null && p.GeneralBillingTitle.Value != nil {
		gb.GeneralBillingTitle = *p.GeneralBillingTitle.Value
	}
	if p.GeneralBillingDesc.Set {
		if p.GeneralBillingDesc.Null {
			gb.GeneralBillingDesc = nil
		} else {
			gb.GeneralBillingDesc = p.GeneralBillingDesc.Value
		}
	}

	// Scope akademik
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

	// Periode: month/year
	if p.GeneralBillingMonth.Set {
		if p.GeneralBillingMonth.Null || p.GeneralBillingMonth.Value == nil {
			gb.GeneralBillingMonth = nil
		} else {
			gb.GeneralBillingMonth = intToInt16Ptr(p.GeneralBillingMonth.Value)
		}
	}
	if p.GeneralBillingYear.Set {
		if p.GeneralBillingYear.Null || p.GeneralBillingYear.Value == nil {
			gb.GeneralBillingYear = nil
		} else {
			gb.GeneralBillingYear = intToInt16Ptr(p.GeneralBillingYear.Value)
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
	if p.GeneralBillingIsActive.Set && !p.GeneralBillingIsActive.Null && p.GeneralBillingIsActive.Value != nil {
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

	GeneralBillingSchoolID uuid.UUID                    `json:"general_billing_school_id"`
	GeneralBillingCategory model.GeneralBillingCategory `json:"general_billing_category"`
	GeneralBillingBillCode string                       `json:"general_billing_bill_code"`

	GeneralBillingCode  *string `json:"general_billing_code,omitempty"`
	GeneralBillingTitle string  `json:"general_billing_title"`
	GeneralBillingDesc  *string `json:"general_billing_desc,omitempty"`

	GeneralBillingClassID   *uuid.UUID `json:"general_billing_class_id,omitempty"`
	GeneralBillingSectionID *uuid.UUID `json:"general_billing_section_id,omitempty"`
	GeneralBillingTermID    *uuid.UUID `json:"general_billing_term_id,omitempty"`

	GeneralBillingMonth *int `json:"general_billing_month,omitempty"`
	GeneralBillingYear  *int `json:"general_billing_year,omitempty"`

	GeneralBillingDueDate  *string `json:"general_billing_due_date,omitempty"` // "YYYY-MM-DD"
	GeneralBillingIsActive bool    `json:"general_billing_is_active"`

	GeneralBillingDefaultAmountIDR *int `json:"general_billing_default_amount_idr,omitempty"`

	GeneralBillingCreatedAt time.Time  `json:"general_billing_created_at"`
	GeneralBillingUpdatedAt time.Time  `json:"general_billing_updated_at"`
	GeneralBillingDeletedAt *time.Time `json:"general_billing_deleted_at,omitempty"`
}

func FromModelGeneralBilling(c *fiber.Ctx, m *model.GeneralBillingModel) *GeneralBillingResponse {
	// Konversi created/updated/deleted ke timezone sekolah
	createdAt := dbtime.ToSchoolTime(c, m.GeneralBillingCreatedAt)
	updatedAt := dbtime.ToSchoolTime(c, m.GeneralBillingUpdatedAt)
	deletedAt := dbtime.ToSchoolTimePtr(c, m.GeneralBillingDeletedAt)

	// Due date: tetap string "YYYY-MM-DD" tapi berdasarkan waktu di timezone sekolah
	var due *string
	if m.GeneralBillingDueDate != nil {
		localDue := dbtime.ToSchoolTime(c, *m.GeneralBillingDueDate)
		s := localDue.Format("2006-01-02")
		due = &s
	}

	return &GeneralBillingResponse{
		GeneralBillingID:       m.GeneralBillingID,
		GeneralBillingSchoolID: m.GeneralBillingSchoolID,
		GeneralBillingCategory: m.GeneralBillingCategory,
		GeneralBillingBillCode: m.GeneralBillingBillCode,

		GeneralBillingCode:  m.GeneralBillingCode,
		GeneralBillingTitle: m.GeneralBillingTitle,
		GeneralBillingDesc:  m.GeneralBillingDesc,

		GeneralBillingClassID:   m.GeneralBillingClassID,
		GeneralBillingSectionID: m.GeneralBillingSectionID,
		GeneralBillingTermID:    m.GeneralBillingTermID,

		GeneralBillingMonth: int16PtrToIntPtr(m.GeneralBillingMonth),
		GeneralBillingYear:  int16PtrToIntPtr(m.GeneralBillingYear),

		GeneralBillingDueDate:          due,
		GeneralBillingIsActive:         m.GeneralBillingIsActive,
		GeneralBillingDefaultAmountIDR: m.GeneralBillingDefaultAmountIDR,

		GeneralBillingCreatedAt: createdAt,
		GeneralBillingUpdatedAt: updatedAt,
		GeneralBillingDeletedAt: deletedAt,
	}
}
