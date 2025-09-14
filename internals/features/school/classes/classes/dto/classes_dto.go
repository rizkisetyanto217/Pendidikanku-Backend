package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

/* ===================== PatchField ===================== */

type PatchField[T any] struct {
	Set   bool `json:"set"`
	Value T    `json:"value,omitempty"`
}

func (p *PatchField[T]) IsZero() bool {
	return p == nil || !p.Set
}

// >>> WAJIB pakai nama ini (tanpa angka 2)
func (p *PatchField[T]) UnmarshalJSON(b []byte) error {
	type env struct {
		Set   *bool            `json:"set"`
		Value *json.RawMessage `json:"value"`
	}
	if len(b) > 0 && b[0] == '{' {
		var e env
		if err := json.Unmarshal(b, &e); err == nil {
			if e.Set != nil {
				p.Set = *e.Set
			} else {
				p.Set = true
			}
			if e.Value != nil {
				var v T
				if err := json.Unmarshal(*e.Value, &v); err != nil {
					return err
				}
				p.Value = v
			}
			return nil
		}
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	p.Set = true
	p.Value = v
	return nil
}

// >>> Biar form-data juga bisa: "online", "1500003", "true", dst.
func (p *PatchField[T]) UnmarshalText(b []byte) error {
	quoted, _ := json.Marshal(string(b))
	var v T
	if err := json.Unmarshal(quoted, &v); err != nil {
		if err2 := json.Unmarshal(b, &v); err2 != nil {
			return err
		}
	}
	p.Set = true
	p.Value = v
	return nil
}

/* =========================================================
   REQUEST: CREATE (sinkron dengan DDL & model)
   ========================================================= */

type CreateClassRequest struct {
	// Wajib
	ClassMasjidID uuid.UUID `json:"class_masjid_id"              form:"class_masjid_id"              validate:"required"`
	ClassParentID uuid.UUID `json:"class_parent_id"              form:"class_parent_id"              validate:"required"`
	ClassSlug     string    `json:"class_slug"                   form:"class_slug"                   validate:"required,min=1,max=160"`

	// Periode
	ClassStartDate *time.Time `json:"class_start_date,omitempty"         form:"class_start_date"`
	ClassEndDate   *time.Time `json:"class_end_date,omitempty"           form:"class_end_date"`

	// Registrasi / Term
	ClassTermID               *uuid.UUID `json:"class_term_id,omitempty"                form:"class_term_id"`
	ClassIsOpen               *bool      `json:"class_is_open,omitempty"                form:"class_is_open"`
	ClassRegistrationOpensAt  *time.Time `json:"class_registration_opens_at,omitempty"  form:"class_registration_opens_at"`
	ClassRegistrationClosesAt *time.Time `json:"class_registration_closes_at,omitempty" form:"class_registration_closes_at"`

	// Kuota
	ClassQuotaTotal *int `json:"class_quota_total,omitempty" form:"class_quota_total"`

	// Pricing
	ClassRegistrationFeeIDR *int64  `json:"class_registration_fee_idr,omitempty" form:"class_registration_fee_idr"`
	ClassTuitionFeeIDR      *int64  `json:"class_tuition_fee_idr,omitempty"      form:"class_tuition_fee_idr"`
	ClassBillingCycle       *string `json:"class_billing_cycle,omitempty"         form:"class_billing_cycle"`
	ClassProviderProductID  *string `json:"class_provider_product_id,omitempty"   form:"class_provider_product_id"`
	ClassProviderPriceID    *string `json:"class_provider_price_id,omitempty"     form:"class_provider_price_id"`

	// Catatan & media
	ClassNotes    *string `json:"class_notes,omitempty"     form:"class_notes"`

	// Mode & Status (baru)
	ClassDeliveryMode *string    `json:"class_delivery_mode,omitempty" form:"class_delivery_mode"` // enum
	ClassStatus       *string    `json:"class_status,omitempty"        form:"class_status"`        // enum: active|inactive|completed
	ClassCompletedAt  *time.Time `json:"class_completed_at,omitempty"  form:"class_completed_at"`

}

func (r *CreateClassRequest) Normalize() {
	r.ClassSlug = strings.TrimSpace(strings.ToLower(r.ClassSlug))

	if r.ClassBillingCycle != nil {
		x := strings.ToLower(strings.TrimSpace(*r.ClassBillingCycle))
		r.ClassBillingCycle = &x
	}
	if r.ClassDeliveryMode != nil {
		x := strings.ToLower(strings.TrimSpace(*r.ClassDeliveryMode))
		r.ClassDeliveryMode = &x
	}
	if r.ClassStatus != nil {
		x := strings.ToLower(strings.TrimSpace(*r.ClassStatus))
		r.ClassStatus = &x
	}
	if r.ClassIsOpen == nil {
		def := true
		r.ClassIsOpen = &def
	}
	// bersihkan string opsional
	if r.ClassNotes != nil {
		s := strings.TrimSpace(*r.ClassNotes)
		if s == "" {
			r.ClassNotes = nil
		} else {
			r.ClassNotes = &s
		}
	}

	if r.ClassProviderProductID != nil {
		s := strings.TrimSpace(*r.ClassProviderProductID)
		if s == "" {
			r.ClassProviderProductID = nil
		} else {
			r.ClassProviderProductID = &s
		}
	}
	if r.ClassProviderPriceID != nil {
		s := strings.TrimSpace(*r.ClassProviderPriceID)
		if s == "" {
			r.ClassProviderPriceID = nil
		} else {
			r.ClassProviderPriceID = &s
		}
	}


	// Jika status completed → auto close pendaftaran (selaras constraint DB)
	if r.ClassStatus != nil && *r.ClassStatus == model.ClassStatusCompleted {
		f := false
		r.ClassIsOpen = &f
		// CompletedAt boleh diisi, kalau kosong biarin nil (DB tidak paksa)
	}
}

func (r *CreateClassRequest) Validate() error {
	if r.ClassMasjidID == uuid.Nil {
		return errors.New("class_masjid_id required")
	}
	if r.ClassParentID == uuid.Nil {
		return errors.New("class_parent_id required")
	}
	if r.ClassSlug == "" {
		return errors.New("class_slug required")
	}
	if r.ClassRegistrationOpensAt != nil && r.ClassRegistrationClosesAt != nil &&
		r.ClassRegistrationClosesAt.Before(*r.ClassRegistrationOpensAt) {
		return errors.New("class_registration_closes_at must be >= class_registration_opens_at")
	}
	if r.ClassQuotaTotal != nil && *r.ClassQuotaTotal < 0 {
		return errors.New("class_quota_total must be >= 0")
	}
	if r.ClassRegistrationFeeIDR != nil && *r.ClassRegistrationFeeIDR < 0 {
		return errors.New("class_registration_fee_idr must be >= 0")
	}
	if r.ClassTuitionFeeIDR != nil && *r.ClassTuitionFeeIDR < 0 {
		return errors.New("class_tuition_fee_idr must be >= 0")
	}
	// enum guards (soft)
	if r.ClassBillingCycle != nil {
		switch *r.ClassBillingCycle {
		case model.BillingCycleOneTime, model.BillingCycleMonthly, model.BillingCycleQuarter, model.BillingCycleSemester, model.BillingCycleYearly:
		default:
			return errors.New("invalid class_billing_cycle")
		}
	}
	if r.ClassDeliveryMode != nil {
		switch *r.ClassDeliveryMode {
		case model.ClassDeliveryModeOffline, model.ClassDeliveryModeOnline, model.ClassDeliveryModeHybrid:
		default:
			return errors.New("invalid class_delivery_mode")
		}
	}
	if r.ClassStatus != nil {
		switch *r.ClassStatus {
		case model.ClassStatusActive, model.ClassStatusInactive, model.ClassStatusCompleted:
		default:
			return errors.New("invalid class_status")
		}
	}
	return nil
}

func (r *CreateClassRequest) ToModel() *model.ClassModel {
	m := &model.ClassModel{
		ClassMasjidID:             r.ClassMasjidID,
		ClassParentID:             r.ClassParentID,
		ClassSlug:                 r.ClassSlug,
		ClassStartDate:            r.ClassStartDate,
		ClassEndDate:              r.ClassEndDate,
		ClassTermID:               r.ClassTermID,
		ClassIsOpen:               true, // default
		ClassRegistrationOpensAt:  r.ClassRegistrationOpensAt,
		ClassRegistrationClosesAt: r.ClassRegistrationClosesAt,
		ClassQuotaTotal:           r.ClassQuotaTotal,
		ClassRegistrationFeeIDR:   r.ClassRegistrationFeeIDR,
		ClassTuitionFeeIDR:        r.ClassTuitionFeeIDR,
		ClassBillingCycle:         model.BillingCycleMonthly, // default DB
		ClassProviderProductID:    r.ClassProviderProductID,
		ClassProviderPriceID:      r.ClassProviderPriceID,
		ClassNotes:                r.ClassNotes,
		ClassDeliveryMode:         model.ClassDeliveryModeOffline, // default app-side
		ClassStatus:               model.ClassStatusActive,         // default DB
		ClassCompletedAt:          r.ClassCompletedAt,
	}
	if r.ClassIsOpen != nil {
		m.ClassIsOpen = *r.ClassIsOpen
	}
	if r.ClassBillingCycle != nil && *r.ClassBillingCycle != "" {
		m.ClassBillingCycle = *r.ClassBillingCycle
	}
	if r.ClassDeliveryMode != nil && *r.ClassDeliveryMode != "" {
		m.ClassDeliveryMode = *r.ClassDeliveryMode
	}
	if r.ClassStatus != nil && *r.ClassStatus != "" {
		m.ClassStatus = *r.ClassStatus
		// safety: jika completed tapi lupa close
		if m.ClassStatus == model.ClassStatusCompleted {
			m.ClassIsOpen = false
		}
	}
	return m
}

/* =========================================================
   REQUEST: PATCH (partial / tri-state)
   ========================================================= */

type PatchClassRequest struct {
	ClassSlug *PatchField[string] `json:"class_slug,omitempty"                    form:"class_slug"`

	ClassStartDate *PatchField[*time.Time] `json:"class_start_date,omitempty"          form:"class_start_date"`
	ClassEndDate   *PatchField[*time.Time] `json:"class_end_date,omitempty"            form:"class_end_date"`

	ClassTermID               *PatchField[*uuid.UUID] `json:"class_term_id,omitempty"                form:"class_term_id"`
	ClassIsOpen               *PatchField[bool]       `json:"class_is_open,omitempty"                form:"class_is_open"`
	ClassRegistrationOpensAt  *PatchField[*time.Time] `json:"class_registration_opens_at,omitempty"  form:"class_registration_opens_at"`
	ClassRegistrationClosesAt *PatchField[*time.Time] `json:"class_registration_closes_at,omitempty" form:"class_registration_closes_at"`

	ClassQuotaTotal *PatchField[*int] `json:"class_quota_total,omitempty" form:"class_quota_total"`
	ClassQuotaTaken *PatchField[int]  `json:"class_quota_taken,omitempty" form:"class_quota_taken"`

	ClassRegistrationFeeIDR *PatchField[*int64] `json:"class_registration_fee_idr,omitempty" form:"class_registration_fee_idr"`
	ClassTuitionFeeIDR      *PatchField[*int64] `json:"class_tuition_fee_idr,omitempty"      form:"class_tuition_fee_idr"`
	ClassBillingCycle       *PatchField[string] `json:"class_billing_cycle,omitempty"         form:"class_billing_cycle"`
	ClassProviderProductID  *PatchField[*string] `json:"class_provider_product_id,omitempty"  form:"class_provider_product_id"`
	ClassProviderPriceID    *PatchField[*string] `json:"class_provider_price_id,omitempty"    form:"class_provider_price_id"`

	ClassNotes    *PatchField[*string] `json:"class_notes,omitempty"     form:"class_notes"`

	ClassDeliveryMode *PatchField[string]     `json:"class_delivery_mode,omitempty" form:"class_delivery_mode"`
	ClassStatus       *PatchField[string]     `json:"class_status,omitempty"        form:"class_status"`
	ClassCompletedAt  *PatchField[*time.Time] `json:"class_completed_at,omitempty"  form:"class_completed_at"`

}

func (r *PatchClassRequest) Normalize() {
	if r.ClassSlug != nil && r.ClassSlug.Set {
		r.ClassSlug.Value = strings.TrimSpace(strings.ToLower(r.ClassSlug.Value))
	}
	if r.ClassBillingCycle != nil && r.ClassBillingCycle.Set {
		r.ClassBillingCycle.Value = strings.ToLower(strings.TrimSpace(r.ClassBillingCycle.Value))
	}
	if r.ClassDeliveryMode != nil && r.ClassDeliveryMode.Set {
		r.ClassDeliveryMode.Value = strings.ToLower(strings.TrimSpace(r.ClassDeliveryMode.Value))
	}
	if r.ClassStatus != nil && r.ClassStatus.Set {
		r.ClassStatus.Value = strings.ToLower(strings.TrimSpace(r.ClassStatus.Value))
	}
	if r.ClassNotes != nil && r.ClassNotes.Set && r.ClassNotes.Value != nil {
		s := strings.TrimSpace(*r.ClassNotes.Value)
		if s == "" {
			r.ClassNotes.Value = nil
		} else {
			r.ClassNotes.Value = &s
		}
	}
	if r.ClassProviderProductID != nil && r.ClassProviderProductID.Set && r.ClassProviderProductID.Value != nil {
		s := strings.TrimSpace(*r.ClassProviderProductID.Value)
		if s == "" {
			r.ClassProviderProductID.Value = nil
		} else {
			r.ClassProviderProductID.Value = &s
		}
	}
	if r.ClassProviderPriceID != nil && r.ClassProviderPriceID.Set && r.ClassProviderPriceID.Value != nil {
		s := strings.TrimSpace(*r.ClassProviderPriceID.Value)
		if s == "" {
			r.ClassProviderPriceID.Value = nil
		} else {
			r.ClassProviderPriceID.Value = &s
		}
	}

}

func (r *PatchClassRequest) Validate() error {
	if r.ClassRegistrationOpensAt != nil && r.ClassRegistrationOpensAt.Set &&
		r.ClassRegistrationClosesAt != nil && r.ClassRegistrationClosesAt.Set &&
		r.ClassRegistrationOpensAt.Value != nil && r.ClassRegistrationClosesAt.Value != nil &&
		r.ClassRegistrationClosesAt.Value.Before(*r.ClassRegistrationOpensAt.Value) {
		return errors.New("class_registration_closes_at must be >= class_registration_opens_at")
	}
	if r.ClassQuotaTotal != nil && r.ClassQuotaTotal.Set && r.ClassQuotaTotal.Value != nil && *r.ClassQuotaTotal.Value < 0 {
		return errors.New("class_quota_total must be >= 0")
	}
	if r.ClassQuotaTaken != nil && r.ClassQuotaTaken.Set && r.ClassQuotaTaken.Value < 0 {
		return errors.New("class_quota_taken must be >= 0")
	}
	if r.ClassRegistrationFeeIDR != nil && r.ClassRegistrationFeeIDR.Set && r.ClassRegistrationFeeIDR.Value != nil && *r.ClassRegistrationFeeIDR.Value < 0 {
		return errors.New("class_registration_fee_idr must be >= 0")
	}
	if r.ClassTuitionFeeIDR != nil && r.ClassTuitionFeeIDR.Set && r.ClassTuitionFeeIDR.Value != nil && *r.ClassTuitionFeeIDR.Value < 0 {
		return errors.New("class_tuition_fee_idr must be >= 0")
	}
	// enums
	if r.ClassBillingCycle != nil && r.ClassBillingCycle.Set {
		switch r.ClassBillingCycle.Value {
		case model.BillingCycleOneTime, model.BillingCycleMonthly, model.BillingCycleQuarter, model.BillingCycleSemester, model.BillingCycleYearly:
		default:
			return errors.New("invalid class_billing_cycle")
		}
	}
	if r.ClassDeliveryMode != nil && r.ClassDeliveryMode.Set {
		switch r.ClassDeliveryMode.Value {
		case model.ClassDeliveryModeOffline, model.ClassDeliveryModeOnline, model.ClassDeliveryModeHybrid:
		default:
			return errors.New("invalid class_delivery_mode")
		}
	}
	if r.ClassStatus != nil && r.ClassStatus.Set {
		switch r.ClassStatus.Value {
		case model.ClassStatusActive, model.ClassStatusInactive, model.ClassStatusCompleted:
		default:
			return errors.New("invalid class_status")
		}
	}
	return nil
}

func (r *PatchClassRequest) Apply(m *model.ClassModel) {
	if r.ClassSlug != nil && r.ClassSlug.Set {
		m.ClassSlug = r.ClassSlug.Value
	}
	if r.ClassStartDate != nil && r.ClassStartDate.Set {
		m.ClassStartDate = r.ClassStartDate.Value
	}
	if r.ClassEndDate != nil && r.ClassEndDate.Set {
		m.ClassEndDate = r.ClassEndDate.Value
	}
	if r.ClassTermID != nil && r.ClassTermID.Set {
		m.ClassTermID = r.ClassTermID.Value
	}
	if r.ClassIsOpen != nil && r.ClassIsOpen.Set {
		m.ClassIsOpen = r.ClassIsOpen.Value
	}
	if r.ClassRegistrationOpensAt != nil && r.ClassRegistrationOpensAt.Set {
		m.ClassRegistrationOpensAt = r.ClassRegistrationOpensAt.Value
	}
	if r.ClassRegistrationClosesAt != nil && r.ClassRegistrationClosesAt.Set {
		m.ClassRegistrationClosesAt = r.ClassRegistrationClosesAt.Value
	}
	if r.ClassQuotaTotal != nil && r.ClassQuotaTotal.Set {
		m.ClassQuotaTotal = r.ClassQuotaTotal.Value
	}
	if r.ClassQuotaTaken != nil && r.ClassQuotaTaken.Set {
		m.ClassQuotaTaken = r.ClassQuotaTaken.Value
	}
	if r.ClassRegistrationFeeIDR != nil && r.ClassRegistrationFeeIDR.Set {
		m.ClassRegistrationFeeIDR = r.ClassRegistrationFeeIDR.Value
	}
	if r.ClassTuitionFeeIDR != nil && r.ClassTuitionFeeIDR.Set {
		m.ClassTuitionFeeIDR = r.ClassTuitionFeeIDR.Value
	}
	if r.ClassBillingCycle != nil && r.ClassBillingCycle.Set {
		m.ClassBillingCycle = r.ClassBillingCycle.Value
	}
	if r.ClassProviderProductID != nil && r.ClassProviderProductID.Set {
		m.ClassProviderProductID = r.ClassProviderProductID.Value
	}
	if r.ClassProviderPriceID != nil && r.ClassProviderPriceID.Set {
		m.ClassProviderPriceID = r.ClassProviderPriceID.Value
	}
	if r.ClassNotes != nil && r.ClassNotes.Set {
		m.ClassNotes = r.ClassNotes.Value
	}

	if r.ClassDeliveryMode != nil && r.ClassDeliveryMode.Set {
		m.ClassDeliveryMode = r.ClassDeliveryMode.Value
	}
	if r.ClassStatus != nil && r.ClassStatus.Set {
		m.ClassStatus = r.ClassStatus.Value
		// selaras constraint DB: jika completed → is_open = false (kecuali user sudah set explicit)
		if m.ClassStatus == model.ClassStatusCompleted && (r.ClassIsOpen == nil || (r.ClassIsOpen != nil && !r.ClassIsOpen.Set)) {
			m.ClassIsOpen = false
		}
	}
	if r.ClassCompletedAt != nil && r.ClassCompletedAt.Set {
		m.ClassCompletedAt = r.ClassCompletedAt.Value
	}

}

/* =========================================================
   RESPONSE DTO
   ========================================================= */

type ClassResponse struct {
	ClassID       uuid.UUID `json:"class_id"`
	ClassMasjidID uuid.UUID `json:"class_masjid_id"`
	ClassParentID uuid.UUID `json:"class_parent_id"`

	ClassSlug string `json:"class_slug"`

	ClassStartDate *time.Time `json:"class_start_date,omitempty"`
	ClassEndDate   *time.Time `json:"class_end_date,omitempty"`

	ClassTermID               *uuid.UUID `json:"class_term_id,omitempty"`
	ClassIsOpen               bool       `json:"class_is_open"`
	ClassRegistrationOpensAt  *time.Time `json:"class_registration_opens_at,omitempty"`
	ClassRegistrationClosesAt *time.Time `json:"class_registration_closes_at,omitempty"`

	ClassQuotaTotal *int `json:"class_quota_total,omitempty"`
	ClassQuotaTaken int   `json:"class_quota_taken"`

	ClassRegistrationFeeIDR *int64  `json:"class_registration_fee_idr,omitempty"`
	ClassTuitionFeeIDR      *int64  `json:"class_tuition_fee_idr,omitempty"`
	ClassBillingCycle       string  `json:"class_billing_cycle"`
	ClassProviderProductID  *string `json:"class_provider_product_id,omitempty"`
	ClassProviderPriceID    *string `json:"class_provider_price_id,omitempty"`

	ClassNotes    *string `json:"class_notes,omitempty"`

	ClassDeliveryMode string     `json:"class_delivery_mode"`
	ClassStatus       string     `json:"class_status"`
	ClassCompletedAt  *time.Time `json:"class_completed_at,omitempty"`

	ClassCreatedAt time.Time  `json:"class_created_at"`
	ClassUpdatedAt time.Time  `json:"class_updated_at"`
	ClassDeletedAt *time.Time `json:"class_deleted_at,omitempty"`
}

func FromModel(m *model.ClassModel) ClassResponse {
	var delAt *time.Time
	if m.ClassDeletedAt.Valid {
		t := m.ClassDeletedAt.Time
		delAt = &t
	}
	return ClassResponse{
		ClassID:                   m.ClassID,
		ClassMasjidID:             m.ClassMasjidID,
		ClassParentID:             m.ClassParentID,
		ClassSlug:                 m.ClassSlug,
		ClassStartDate:            m.ClassStartDate,
		ClassEndDate:              m.ClassEndDate,
		ClassTermID:               m.ClassTermID,
		ClassIsOpen:               m.ClassIsOpen,
		ClassRegistrationOpensAt:  m.ClassRegistrationOpensAt,
		ClassRegistrationClosesAt: m.ClassRegistrationClosesAt,
		ClassQuotaTotal:           m.ClassQuotaTotal,
		ClassQuotaTaken:           m.ClassQuotaTaken,
		ClassRegistrationFeeIDR:   m.ClassRegistrationFeeIDR,
		ClassTuitionFeeIDR:        m.ClassTuitionFeeIDR,
		ClassBillingCycle:         m.ClassBillingCycle,
		ClassProviderProductID:    m.ClassProviderProductID,
		ClassProviderPriceID:      m.ClassProviderPriceID,
		ClassNotes:                m.ClassNotes,
		ClassDeliveryMode:         m.ClassDeliveryMode,
		ClassStatus:               m.ClassStatus,
		ClassCompletedAt:          m.ClassCompletedAt,
		ClassCreatedAt:            m.ClassCreatedAt,
		ClassUpdatedAt:            m.ClassUpdatedAt,
		ClassDeletedAt:            delAt,
	}
}

/* =========================================================
   QUERY / FILTER DTO (untuk list)
   ========================================================= */

type ListClassQuery struct {
	MasjidID     *uuid.UUID `query:"masjid_id"`
	ParentID     *uuid.UUID `query:"parent_id"`
	TermID       *uuid.UUID `query:"term_id"`
	IsOpen       *bool      `query:"is_open"`
	Status       *string    `query:"status"`        // enum: active|inactive|completed
	DeliveryMode *string    `query:"delivery_mode"` // enum; lower-case di repo
	Slug         *string    `query:"slug"`          // exact/ilike di repo
	Search       *string    `query:"search"`        // cari di notes (trigram)

	StartGe      *time.Time `query:"start_ge"`
	StartLe      *time.Time `query:"start_le"`
	RegOpenGe    *time.Time `query:"reg_open_ge"`
	RegCloseLe   *time.Time `query:"reg_close_le"`

	// opsional filter untuk completed range
	CompletedGe *time.Time `query:"completed_ge"`
	CompletedLe *time.Time `query:"completed_le"`

	Limit  int     `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	SortBy *string `query:"sort_by"` // created_at|slug|start_date|is_open|status|delivery_mode
	Order  *string `query:"order"`   // asc|desc
}

func (q *ListClassQuery) Normalize() {
	if q.DeliveryMode != nil {
		x := strings.ToLower(strings.TrimSpace(*q.DeliveryMode))
		q.DeliveryMode = &x
	}
	if q.Status != nil {
		x := strings.ToLower(strings.TrimSpace(*q.Status))
		q.Status = &x
	}
	if q.Slug != nil {
		x := strings.TrimSpace(strings.ToLower(*q.Slug))
		q.Slug = &x
	}
	if q.SortBy != nil {
		x := strings.TrimSpace(strings.ToLower(*q.SortBy))
		q.SortBy = &x
	}
	if q.Order != nil {
		x := strings.TrimSpace(strings.ToLower(*q.Order))
		if x != "asc" && x != "desc" {
			x = "desc"
		}
		q.Order = &x
	}
}
