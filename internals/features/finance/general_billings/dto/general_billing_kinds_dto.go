// file: internals/features/finance/general_billings/dto/general_billing_kind_dto.go
package dto

import (
	"time"

	m "schoolku_backend/internals/features/finance/general_billings/model"

	"github.com/google/uuid"
)

/* =========================================================
   Response DTO
========================================================= */

type GeneralBillingKindDTO struct {
	ID               uuid.UUID  `json:"general_billing_kind_id"`
	SchoolID         *uuid.UUID `json:"general_billing_kind_school_id,omitempty"` // nullable utk GLOBAL
	Code             string     `json:"general_billing_kind_code"`
	Name             string     `json:"general_billing_kind_name"`
	Desc             *string    `json:"general_billing_kind_desc,omitempty"`
	IsActive         bool       `json:"general_billing_kind_is_active"`
	DefaultAmountIDR *int       `json:"general_billing_kind_default_amount_idr,omitempty"`

	// ⬇️ enum kategori baru: registration | spp | mass_student | donation
	Category   string  `json:"general_billing_kind_category"`
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
   Create / Patch
   (request tetap pakai string agar fleksibel; validasi di controller)
========================================================= */

type CreateGeneralBillingKindRequest struct {
	SchoolID *uuid.UUID `json:"general_billing_kind_school_id,omitempty"`

	Code             string  `json:"general_billing_kind_code"`
	Name             string  `json:"general_billing_kind_name"`
	Desc             *string `json:"general_billing_kind_desc,omitempty"`
	IsActive         *bool   `json:"general_billing_kind_is_active,omitempty"` // default true
	DefaultAmountIDR *int    `json:"general_billing_kind_default_amount_idr,omitempty"`

	// ⬇️ enum baru: "registration" | "spp" | "mass_student" | "donation"
	Category   *string `json:"general_billing_kind_category,omitempty"`
	IsGlobal   *bool   `json:"general_billing_kind_is_global,omitempty"`  // default false
	Visibility *string `json:"general_billing_kind_visibility,omitempty"` // "public" | "internal"

	// Flags (default false – akan dicek oleh constraint DB ck_gbk_flags_match_category)
	IsRecurring        *bool `json:"general_billing_kind_is_recurring,omitempty"`
	RequiresMonthYear  *bool `json:"general_billing_kind_requires_month_year,omitempty"`
	RequiresOptionCode *bool `json:"general_billing_kind_requires_option_code,omitempty"`
}

func (r CreateGeneralBillingKindRequest) ToModel() m.GeneralBillingKind {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	// default kategori di DB = mass_student, tetapi kita set jika diberikan
	cat := m.GeneralBillingKindCategory("mass_student")
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

type PatchGeneralBillingKindRequest struct {
	ID uuid.UUID `json:"id"`

	Code             *string `json:"general_billing_kind_code,omitempty"`
	Name             *string `json:"general_billing_kind_name,omitempty"`
	Desc             *string `json:"general_billing_kind_desc,omitempty"` // "" => clear, nil => no-op
	IsActive         *bool   `json:"general_billing_kind_is_active,omitempty"`
	DefaultAmountIDR *int    `json:"general_billing_kind_default_amount_idr,omitempty"`

	// ⬇️ enum baru: "registration" | "spp" | "mass_student" | "donation"
	Category   *string `json:"general_billing_kind_category,omitempty"`
	IsGlobal   *bool   `json:"general_billing_kind_is_global,omitempty"`
	Visibility *string `json:"general_billing_kind_visibility,omitempty"` // "public" | "internal" | "" => clear

	// Flags
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
   Query/List Request
========================================================= */

type ListGeneralBillingKindsQuery struct {
	SchoolID *uuid.UUID `query:"school_id"`

	Search   string  `query:"search"`     // cari di code/name (case-insensitive)
	IsActive *bool   `query:"is_active"`  // nil=all
	Category *string `query:"category"`   // "registration" | "spp" | "mass_student" | "donation"
	IsGlobal *bool   `query:"is_global"`  // true/false
	Visible  *string `query:"visibility"` // "public" | "internal"

	// Flags
	IsRecurring        *bool `query:"is_recurring"`
	RequiresMonthYear  *bool `query:"requires_month_year"`
	RequiresOptionCode *bool `query:"requires_option_code"`

	// Paging/sort (opsional)
	Page     int    `query:"page"`
	PageSize int    `query:"page_size"`
	Sort     string `query:"sort"`

	// Tanggal; QueryParser aman untuk RFC3339. Controller sediakan fallback YYYY-MM-DD.
	CreatedFrom *time.Time `query:"created_from"`
	CreatedTo   *time.Time `query:"created_to"`
}
