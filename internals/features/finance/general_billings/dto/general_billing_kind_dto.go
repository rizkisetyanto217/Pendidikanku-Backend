// file: internals/features/billings/general_billing_kinds/dto/general_billing_kind_dto.go
package dto

import (
	"time"

	"github.com/google/uuid"

	m "masjidku_backend/internals/features/finance/general_billings/model"
)

/* =========================================================
   Response DTO
========================================================= */

type GeneralBillingKindDTO struct {
	ID               uuid.UUID  `json:"id"`
	MasjidID         *uuid.UUID `json:"masjid_id,omitempty"` // nullable untuk GLOBAL kind
	Code             string     `json:"code"`
	Name             string     `json:"name"`
	Desc             *string    `json:"desc,omitempty"`
	IsActive         bool       `json:"is_active"`
	DefaultAmountIDR *int       `json:"default_amount_idr,omitempty"`

	Category   string  `json:"category"`             // "billing" | "campaign"
	IsGlobal   bool    `json:"is_global"`            // true jika global kind
	Visibility *string `json:"visibility,omitempty"` // "public" | "internal" | null

	// Flags pipeline per-siswa (baru)
	IsRecurring        bool `json:"is_recurring"`
	RequiresMonthYear  bool `json:"requires_month_year"`
	RequiresOptionCode bool `json:"requires_option_code"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func FromModel(g m.GeneralBillingKind) GeneralBillingKindDTO {
	dto := GeneralBillingKindDTO{
		ID:                 g.GeneralBillingKindID,
		MasjidID:           g.GeneralBillingKindMasjidID,
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
   Create Request
========================================================= */

type CreateGeneralBillingKindRequest struct {
	// MasjidID boleh kosong (GLOBAL kind)
	MasjidID *uuid.UUID `json:"masjid_id,omitempty"`

	Code             string  `json:"code"` // unik per-tenant (alive) atau unik global bila masjid_id null
	Name             string  `json:"name"`
	Desc             *string `json:"desc,omitempty"`
	IsActive         *bool   `json:"is_active,omitempty"` // default true
	DefaultAmountIDR *int    `json:"default_amount_idr,omitempty"`

	// Baru (sesuai SQL)
	Category   *string `json:"category,omitempty"`   // "billing" | "campaign" (default "billing")
	IsGlobal   *bool   `json:"is_global,omitempty"`  // default false
	Visibility *string `json:"visibility,omitempty"` // "public" | "internal"

	// Flags pipeline per-siswa (baru; default false)
	IsRecurring        *bool `json:"is_recurring,omitempty"`
	RequiresMonthYear  *bool `json:"requires_month_year,omitempty"`
	RequiresOptionCode *bool `json:"requires_option_code,omitempty"`
}

func (r CreateGeneralBillingKindRequest) ToModel() m.GeneralBillingKind {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	// category default "billing"
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
		GeneralBillingKindMasjidID:         r.MasjidID,
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
========================================================= */

type PatchGeneralBillingKindRequest struct {
	ID uuid.UUID `json:"id"` // biasanya di path param; disediakan juga di body jika perlu

	Code             *string `json:"code,omitempty"`
	Name             *string `json:"name,omitempty"`
	Desc             *string `json:"desc,omitempty"` // "" untuk clear, nil untuk no-op
	IsActive         *bool   `json:"is_active,omitempty"`
	DefaultAmountIDR *int    `json:"default_amount_idr,omitempty"`

	// Baru (opsional; sebaiknya dibatasi untuk admin)
	Category   *string `json:"category,omitempty"` // "billing" | "campaign"
	IsGlobal   *bool   `json:"is_global,omitempty"`
	Visibility *string `json:"visibility,omitempty"` // "public" | "internal" | "" => clear

	// Flags (tri-state; nil = no-op)
	IsRecurring        *bool `json:"is_recurring,omitempty"`
	RequiresMonthYear  *bool `json:"requires_month_year,omitempty"`
	RequiresOptionCode *bool `json:"requires_option_code,omitempty"`
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
   Query/List Request
========================================================= */

type ListGeneralBillingKindsQuery struct {
	// MasjidID optional: kosong = list global kinds atau gabungan (tergantung endpoint-mu)
	MasjidID *uuid.UUID `query:"masjid_id"`

	Search   string  `query:"search"`     // cari di code/name
	IsActive *bool   `query:"is_active"`  // nil=all
	Category *string `query:"category"`   // "billing" | "campaign"
	IsGlobal *bool   `query:"is_global"`  // true/false
	Visible  *string `query:"visibility"` // "public" | "internal"

	// Filter flags (baru)
	IsRecurring        *bool `query:"is_recurring"`
	RequiresMonthYear  *bool `query:"requires_month_year"`
	RequiresOptionCode *bool `query:"requires_option_code"`

	Page     int    `query:"page"`      // default 1
	PageSize int    `query:"page_size"` // default 20
	Sort     string `query:"sort"`      // "created_at_desc"(default) | "created_at_asc" | "name_asc" | "name_desc"

	CreatedFrom *time.Time `query:"created_from"`
	CreatedTo   *time.Time `query:"created_to"`
}

/* =========================================================
   Upsert (opsional)
========================================================= */

type UpsertGeneralBillingKindItem struct {
	Code             string  `json:"code"`
	Name             string  `json:"name"`
	Desc             *string `json:"desc,omitempty"`
	IsActive         *bool   `json:"is_active,omitempty"`
	DefaultAmountIDR *int    `json:"default_amount_idr,omitempty"`

	Category   *string `json:"category,omitempty"`   // "billing" | "campaign"
	Visibility *string `json:"visibility,omitempty"` // "public" | "internal"
	IsGlobal   *bool   `json:"is_global,omitempty"`

	// Flags (opsional)
	IsRecurring        *bool `json:"is_recurring,omitempty"`
	RequiresMonthYear  *bool `json:"requires_month_year,omitempty"`
	RequiresOptionCode *bool `json:"requires_option_code,omitempty"`
}

type UpsertGeneralBillingKindsRequest struct {
	// MasjidID boleh null untuk upsert GLOBAL items
	MasjidID *uuid.UUID                     `json:"masjid_id,omitempty"`
	Items    []UpsertGeneralBillingKindItem `json:"items"`
}
