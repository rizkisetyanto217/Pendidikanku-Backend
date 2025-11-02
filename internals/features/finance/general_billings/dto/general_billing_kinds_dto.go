// file: internals/features/billings/general_billing_kinds/dto/general_billing_kind_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"

	m "schoolku_backend/internals/features/finance/general_billings/model"
)

/* =========================================================
   Response DTO (JSON tag disamakan dengan model)
========================================================= */

type GeneralBillingKindDTO struct {
	ID               uuid.UUID  `json:"general_billing_kind_id"`
	SchoolID         *uuid.UUID `json:"general_billing_kind_school_id,omitempty"` // nullable utk GLOBAL
	Code             string     `json:"general_billing_kind_code"`
	Name             string     `json:"general_billing_kind_name"`
	Desc             *string    `json:"general_billing_kind_desc,omitempty"`
	IsActive         bool       `json:"general_billing_kind_is_active"`
	DefaultAmountIDR *int       `json:"general_billing_kind_default_amount_idr,omitempty"`

	Category   string  `json:"general_billing_kind_category"` // "billing" | "campaign"
	IsGlobal   bool    `json:"general_billing_kind_is_global"`
	Visibility *string `json:"general_billing_kind_visibility,omitempty"` // "public" | "internal" | null

	// Flags pipeline per-siswa
	IsRecurring        bool `json:"general_billing_kind_is_recurring"`
	RequiresMonthYear  bool `json:"general_billing_kind_requires_month_year"`
	RequiresOptionCode bool `json:"general_billing_kind_requires_option_code"`

	CreatedAt time.Time  `json:"general_billing_kind_created_at"`
	UpdatedAt time.Time  `json:"general_billing_kind_updated_at"`
	DeletedAt *time.Time `json:"general_billing_kind_deleted_at,omitempty"`
}

func FromModel(g m.GeneralBillingKind) GeneralBillingKindDTO {
	dto := GeneralBillingKindDTO{
		ID:                 g.GeneralBillingKindID,
		SchoolID:           g.GeneralBillingKindSchoolID,
		Code:               g.GeneralBillingKindCode,
		Name:               g.GeneralBillingKindName,
		Desc:               g.GeneralBillingKindDesc,
		IsActive:           g.GeneralBillingKindIsActive,
		DefaultAmountIDR:   g.GeneralBillingKindDefaultAmountIDR,
		Category:           string(g.GeneralBillingKindCategory),
		IsGlobal:           g.GeneralBillingKindIsGlobal,
		IsRecurring:        g.GeneralBillingKindIsRecurring,
		RequiresMonthYear:  g.GeneralBillingKindRequiresMonthYear,
		RequiresOptionCode: g.GeneralBillingKindRequiresOptionCode,
		CreatedAt:          g.GeneralBillingKindCreatedAt,
		UpdatedAt:          g.GeneralBillingKindUpdatedAt,
		DeletedAt:          g.GeneralBillingKindDeletedAt,
	}
	if g.GeneralBillingKindVisibility != nil {
		v := string(*g.GeneralBillingKindVisibility)
		dto.Visibility = &v
	}
	return dto
}

func FromModelSlice(xs []m.GeneralBillingKind) []GeneralBillingKindDTO {
	out := make([]GeneralBillingKindDTO, 0, len(xs))
	for _, it := range xs {
		out = append(out, FromModel(it))
	}
	return out
}

/* =========================================================
   Create Request (tag JSON disamakan dg model)
========================================================= */

type CreateGeneralBillingKindRequest struct {
	// SchoolID boleh kosong (GLOBAL kind); biasanya di-path dan di-override controller
	SchoolID *uuid.UUID `json:"general_billing_kind_school_id,omitempty"`

	Code             string  `json:"general_billing_kind_code"`
	Name             string  `json:"general_billing_kind_name"`
	Desc             *string `json:"general_billing_kind_desc,omitempty"`
	IsActive         *bool   `json:"general_billing_kind_is_active,omitempty"` // default true
	DefaultAmountIDR *int    `json:"general_billing_kind_default_amount_idr,omitempty"`

	Category   *string `json:"general_billing_kind_category,omitempty"`   // "billing" | "campaign" (default "billing")
	IsGlobal   *bool   `json:"general_billing_kind_is_global,omitempty"`  // default false
	Visibility *string `json:"general_billing_kind_visibility,omitempty"` // "public" | "internal"

	// Flags (default false)
	IsRecurring        *bool `json:"general_billing_kind_is_recurring,omitempty"`
	RequiresMonthYear  *bool `json:"general_billing_kind_requires_month_year,omitempty"`
	RequiresOptionCode *bool `json:"general_billing_kind_requires_option_code,omitempty"`
}

func (r CreateGeneralBillingKindRequest) ToModel() m.GeneralBillingKind {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	// default category "billing"
	cat := m.GBKCategoryBilling
	if r.Category != nil && *r.Category != "" {
		cat = m.GeneralBillingKindCategory(*r.Category)
	}

	var vis *m.GeneralBillingKindVisibility
	if r.Visibility != nil && *r.Visibility != "" {
		v := m.GeneralBillingKindVisibility(*r.Visibility)
		vis = &v
	}

	isGlobal := false
	if r.IsGlobal != nil {
		isGlobal = *r.IsGlobal
	}

	isRecurring := false
	if r.IsRecurring != nil {
		isRecurring = *r.IsRecurring
	}
	requiresMonthYear := false
	if r.RequiresMonthYear != nil {
		requiresMonthYear = *r.RequiresMonthYear
	}
	requiresOptionCode := false
	if r.RequiresOptionCode != nil {
		requiresOptionCode = *r.RequiresOptionCode
	}

	return m.GeneralBillingKind{
		GeneralBillingKindSchoolID:         r.SchoolID,
		GeneralBillingKindCode:             r.Code,
		GeneralBillingKindName:             r.Name,
		GeneralBillingKindDesc:             r.Desc,
		GeneralBillingKindIsActive:         isActive,
		GeneralBillingKindDefaultAmountIDR: r.DefaultAmountIDR,
		GeneralBillingKindCategory:         cat,
		GeneralBillingKindIsGlobal:         isGlobal,
		GeneralBillingKindVisibility:       vis,

		GeneralBillingKindIsRecurring:        isRecurring,
		GeneralBillingKindRequiresMonthYear:  requiresMonthYear,
		GeneralBillingKindRequiresOptionCode: requiresOptionCode,
	}
}

/* =========================================================
   Patch/Update Request (tri-state via pointer)
   (tag JSON disamakan dg model)
========================================================= */

type PatchGeneralBillingKindRequest struct {
	ID uuid.UUID `json:"id"` // biasanya di path; tetap disediakan di body jika perlu

	Code             *string `json:"general_billing_kind_code,omitempty"`
	Name             *string `json:"general_billing_kind_name,omitempty"`
	Desc             *string `json:"general_billing_kind_desc,omitempty"` // "" => clear, nil => no-op
	IsActive         *bool   `json:"general_billing_kind_is_active,omitempty"`
	DefaultAmountIDR *int    `json:"general_billing_kind_default_amount_idr,omitempty"`

	Category   *string `json:"general_billing_kind_category,omitempty"` // "billing" | "campaign"
	IsGlobal   *bool   `json:"general_billing_kind_is_global,omitempty"`
	Visibility *string `json:"general_billing_kind_visibility,omitempty"` // "public" | "internal" | "" => clear

	// Flags (tri-state; nil = no-op)
	IsRecurring        *bool `json:"general_billing_kind_is_recurring,omitempty"`
	RequiresMonthYear  *bool `json:"general_billing_kind_requires_month_year,omitempty"`
	RequiresOptionCode *bool `json:"general_billing_kind_requires_option_code,omitempty"`
}

func (p PatchGeneralBillingKindRequest) ApplyTo(g *m.GeneralBillingKind) {
	if p.Code != nil {
		g.GeneralBillingKindCode = *p.Code
	}
	if p.Name != nil {
		g.GeneralBillingKindName = *p.Name
	}
	if p.Desc != nil {
		if *p.Desc == "" {
			g.GeneralBillingKindDesc = nil
		} else {
			g.GeneralBillingKindDesc = p.Desc
		}
	}
	if p.IsActive != nil {
		g.GeneralBillingKindIsActive = *p.IsActive
	}
	if p.DefaultAmountIDR != nil {
		g.GeneralBillingKindDefaultAmountIDR = p.DefaultAmountIDR
	}
	if p.Category != nil && *p.Category != "" {
		g.GeneralBillingKindCategory = m.GeneralBillingKindCategory(*p.Category)
	}
	if p.IsGlobal != nil {
		g.GeneralBillingKindIsGlobal = *p.IsGlobal
	}
	if p.Visibility != nil {
		if *p.Visibility == "" {
			g.GeneralBillingKindVisibility = nil
		} else {
			v := m.GeneralBillingKindVisibility(*p.Visibility)
			g.GeneralBillingKindVisibility = &v
		}
	}

	// Flags
	if p.IsRecurring != nil {
		g.GeneralBillingKindIsRecurring = *p.IsRecurring
	}
	if p.RequiresMonthYear != nil {
		g.GeneralBillingKindRequiresMonthYear = *p.RequiresMonthYear
	}
	if p.RequiresOptionCode != nil {
		g.GeneralBillingKindRequiresOptionCode = *p.RequiresOptionCode
	}
}

/* =========================================================
   Query/List Request (untouched; tetap pakai query tag singkat)
========================================================= */

type ListGeneralBillingKindsQuery struct {
	SchoolID *uuid.UUID `query:"school_id"`

	Search   string  `query:"search"`     // cari di code/name
	IsActive *bool   `query:"is_active"`  // nil=all
	Category *string `query:"category"`   // "billing" | "campaign"
	IsGlobal *bool   `query:"is_global"`  // true/false
	Visible  *string `query:"visibility"` // "public" | "internal"

	// Filter flags
	IsRecurring        *bool `query:"is_recurring"`
	RequiresMonthYear  *bool `query:"requires_month_year"`
	RequiresOptionCode *bool `query:"requires_option_code"`

	Page        int        `query:"page"`      // default 1
	PageSize    int        `query:"page_size"` // default 20
	Sort        string     `query:"sort"`      // "created_at_desc"(default) | "created_at_asc" | "name_asc" | "name_desc"
	CreatedFrom *time.Time `query:"created_from"`
	CreatedTo   *time.Time `query:"created_to"`
}

/* =========================================================
   Upsert (opsional)
========================================================= */

type UpsertGeneralBillingKindItem struct {
	Code             string  `json:"general_billing_kind_code"`
	Name             string  `json:"general_billing_kind_name"`
	Desc             *string `json:"general_billing_kind_desc,omitempty"`
	IsActive         *bool   `json:"general_billing_kind_is_active,omitempty"`
	DefaultAmountIDR *int    `json:"general_billing_kind_default_amount_idr,omitempty"`

	Category   *string `json:"general_billing_kind_category,omitempty"`
	Visibility *string `json:"general_billing_kind_visibility,omitempty"`
	IsGlobal   *bool   `json:"general_billing_kind_is_global,omitempty"`

	// Flags
	IsRecurring        *bool `json:"general_billing_kind_is_recurring,omitempty"`
	RequiresMonthYear  *bool `json:"general_billing_kind_requires_month_year,omitempty"`
	RequiresOptionCode *bool `json:"general_billing_kind_requires_option_code,omitempty"`
}

type UpsertGeneralBillingKindsRequest struct {
	SchoolID *uuid.UUID                     `json:"general_billing_kind_school_id,omitempty"`
	Items    []UpsertGeneralBillingKindItem `json:"items"`
}
