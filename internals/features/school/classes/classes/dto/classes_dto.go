package dto

import (
	"errors"
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

/* =========================================================
   GENERIC: PatchField[T]
   ========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"set"`
	Value T    `json:"value,omitempty"`
}

func (p *PatchField[T]) IsZero() bool {
	return p == nil || !p.Set
}

/* =========================================================
   REQUEST: CREATE (sinkron dengan DDL & model)
   ========================================================= */

type CreateClassRequest struct {
	// Wajib
	ClassMasjidID uuid.UUID `json:"class_masjid_id" validate:"required"`
	ClassParentID uuid.UUID `json:"class_parent_id" validate:"required"`
	ClassSlug     string    `json:"class_slug"      validate:"required,min=1,max=160"`

	// Periode
	ClassStartDate *time.Time `json:"class_start_date,omitempty"`
	ClassEndDate   *time.Time `json:"class_end_date,omitempty"`

	// Registrasi / Term
	ClassTermID               *uuid.UUID `json:"class_term_id,omitempty"`
	ClassIsOpen               *bool      `json:"class_is_open,omitempty"` // default true (kalau nil)
	ClassRegistrationOpensAt  *time.Time `json:"class_registration_opens_at,omitempty"`
	ClassRegistrationClosesAt *time.Time `json:"class_registration_closes_at,omitempty"`

	// Kuota
	ClassQuotaTotal *int `json:"class_quota_total,omitempty"` // nullable
	// ClassQuotaTaken tidak dikirim saat create (default 0 oleh DB)

	// Pricing
	ClassRegistrationFeeIDR *int64  `json:"class_registration_fee_idr,omitempty"`
	ClassTuitionFeeIDR      *int64  `json:"class_tuition_fee_idr,omitempty"`
	ClassBillingCycle       *string `json:"class_billing_cycle,omitempty"` // default "monthly"
	ClassProviderProductID  *string `json:"class_provider_product_id,omitempty"`
	ClassProviderPriceID    *string `json:"class_provider_price_id,omitempty"`

	// Catatan & media
	ClassNotes    *string `json:"class_notes,omitempty"`
	ClassImageURL *string `json:"class_image_url,omitempty"`

	// Mode & status
	ClassDeliveryMode *string `json:"class_delivery_mode,omitempty"` // enum
	ClassIsActive     *bool   `json:"class_is_active,omitempty"`     // default true

	// Trash (opsional)
	ClassTrashURL           *string    `json:"class_trash_url,omitempty"`
	ClassDeletePendingUntil *time.Time `json:"class_delete_pending_until,omitempty"`
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
	if r.ClassIsOpen == nil {
		def := true
		r.ClassIsOpen = &def
	}
	if r.ClassIsActive == nil {
		def := true
		r.ClassIsActive = &def
	}
	if r.ClassNotes != nil {
		s := strings.TrimSpace(*r.ClassNotes)
		if s == "" {
			r.ClassNotes = nil
		} else {
			r.ClassNotes = &s
		}
	}
	if r.ClassImageURL != nil {
		s := strings.TrimSpace(*r.ClassImageURL)
		if s == "" {
			r.ClassImageURL = nil
		} else {
			r.ClassImageURL = &s
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
	if r.ClassTrashURL != nil {
		s := strings.TrimSpace(*r.ClassTrashURL)
		if s == "" {
			r.ClassTrashURL = nil
		} else {
			r.ClassTrashURL = &s
		}
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
	// window (cek ringan, DB juga cek via constraint)
	if r.ClassRegistrationOpensAt != nil && r.ClassRegistrationClosesAt != nil {
		if r.ClassRegistrationClosesAt.Before(*r.ClassRegistrationOpensAt) {
			return errors.New("class_registration_closes_at must be >= class_registration_opens_at")
		}
	}
	// kuota non-neg (DB juga cek)
	if r.ClassQuotaTotal != nil && *r.ClassQuotaTotal < 0 {
		return errors.New("class_quota_total must be >= 0")
	}
	// fee non-neg (DB juga cek)
	if r.ClassRegistrationFeeIDR != nil && *r.ClassRegistrationFeeIDR < 0 {
		return errors.New("class_registration_fee_idr must be >= 0")
	}
	if r.ClassTuitionFeeIDR != nil && *r.ClassTuitionFeeIDR < 0 {
		return errors.New("class_tuition_fee_idr must be >= 0")
	}
	return nil
}

func (r *CreateClassRequest) ToModel() *model.ClassModel {
	m := &model.ClassModel{
		ClassMasjidID: r.ClassMasjidID,
		ClassParentID: r.ClassParentID,
		ClassSlug:     r.ClassSlug,

		ClassStartDate: r.ClassStartDate,
		ClassEndDate:   r.ClassEndDate,

		ClassTermID:               r.ClassTermID,
		ClassIsOpen:               true,
		ClassRegistrationOpensAt:  r.ClassRegistrationOpensAt,
		ClassRegistrationClosesAt: r.ClassRegistrationClosesAt,

		ClassQuotaTotal: r.ClassQuotaTotal,
		// ClassQuotaTaken: default 0 (DB)

		ClassRegistrationFeeIDR: r.ClassRegistrationFeeIDR,
		ClassTuitionFeeIDR:      r.ClassTuitionFeeIDR,
		ClassBillingCycle:       model.BillingCycleMonthly, // default
		ClassProviderProductID:  r.ClassProviderProductID,
		ClassProviderPriceID:    r.ClassProviderPriceID,

		ClassNotes:       r.ClassNotes,
		ClassImageURL:    r.ClassImageURL,
		ClassDeliveryMode: model.ClassDeliveryModeOffline, // default
		ClassIsActive:     true,

		ClassTrashURL:           r.ClassTrashURL,
		ClassDeletePendingUntil: r.ClassDeletePendingUntil,
	}

	if r.ClassIsOpen != nil {
		m.ClassIsOpen = *r.ClassIsOpen
	}
	if r.ClassIsActive != nil {
		m.ClassIsActive = *r.ClassIsActive
	}
	if r.ClassBillingCycle != nil && *r.ClassBillingCycle != "" {
		m.ClassBillingCycle = *r.ClassBillingCycle
	}
	if r.ClassDeliveryMode != nil && *r.ClassDeliveryMode != "" {
		m.ClassDeliveryMode = *r.ClassDeliveryMode
	}

	return m
}

/* =========================================================
   REQUEST: PATCH (partial / tri-state)
   ========================================================= */

type PatchClassRequest struct {
	ClassSlug *PatchField[string] `json:"class_slug,omitempty"`

	ClassStartDate *PatchField[*time.Time] `json:"class_start_date,omitempty"`
	ClassEndDate   *PatchField[*time.Time] `json:"class_end_date,omitempty"`

	ClassTermID               *PatchField[*uuid.UUID] `json:"class_term_id,omitempty"`
	ClassIsOpen               *PatchField[bool]       `json:"class_is_open,omitempty"`
	ClassRegistrationOpensAt  *PatchField[*time.Time] `json:"class_registration_opens_at,omitempty"`
	ClassRegistrationClosesAt *PatchField[*time.Time] `json:"class_registration_closes_at,omitempty"`

	ClassQuotaTotal *PatchField[*int] `json:"class_quota_total,omitempty"`
	ClassQuotaTaken *PatchField[int]  `json:"class_quota_taken,omitempty"`

	ClassRegistrationFeeIDR *PatchField[*int64] `json:"class_registration_fee_idr,omitempty"`
	ClassTuitionFeeIDR      *PatchField[*int64] `json:"class_tuition_fee_idr,omitempty"`
	ClassBillingCycle       *PatchField[string]  `json:"class_billing_cycle,omitempty"`
	ClassProviderProductID  *PatchField[*string] `json:"class_provider_product_id,omitempty"`
	ClassProviderPriceID    *PatchField[*string] `json:"class_provider_price_id,omitempty"`

	ClassNotes    *PatchField[*string] `json:"class_notes,omitempty"`
	ClassImageURL *PatchField[*string] `json:"class_image_url,omitempty"`

	ClassDeliveryMode *PatchField[string] `json:"class_delivery_mode,omitempty"`
	ClassIsActive     *PatchField[bool]   `json:"class_is_active,omitempty"`

	ClassTrashURL           *PatchField[*string]    `json:"class_trash_url,omitempty"`
	ClassDeletePendingUntil *PatchField[*time.Time] `json:"class_delete_pending_until,omitempty"`
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
	if r.ClassNotes != nil && r.ClassNotes.Set && r.ClassNotes.Value != nil {
		s := strings.TrimSpace(*r.ClassNotes.Value)
		if s == "" {
			r.ClassNotes.Value = nil
		} else {
			r.ClassNotes.Value = &s
		}
	}
	if r.ClassImageURL != nil && r.ClassImageURL.Set && r.ClassImageURL.Value != nil {
		s := strings.TrimSpace(*r.ClassImageURL.Value)
		if s == "" {
			r.ClassImageURL.Value = nil
		} else {
			r.ClassImageURL.Value = &s
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
	if r.ClassTrashURL != nil && r.ClassTrashURL.Set && r.ClassTrashURL.Value != nil {
		s := strings.TrimSpace(*r.ClassTrashURL.Value)
		if s == "" {
			r.ClassTrashURL.Value = nil
		} else {
			r.ClassTrashURL.Value = &s
		}
	}
}

func (r *PatchClassRequest) Validate() error {
	// registrasi window
	if r.ClassRegistrationOpensAt != nil && r.ClassRegistrationOpensAt.Set &&
		r.ClassRegistrationClosesAt != nil && r.ClassRegistrationClosesAt.Set &&
		r.ClassRegistrationOpensAt.Value != nil && r.ClassRegistrationClosesAt.Value != nil &&
		r.ClassRegistrationClosesAt.Value.Before(*r.ClassRegistrationOpensAt.Value) {
		return errors.New("class_registration_closes_at must be >= class_registration_opens_at")
	}
	// kuota non-neg
	if r.ClassQuotaTotal != nil && r.ClassQuotaTotal.Set && r.ClassQuotaTotal.Value != nil && *r.ClassQuotaTotal.Value < 0 {
		return errors.New("class_quota_total must be >= 0")
	}
	if r.ClassQuotaTaken != nil && r.ClassQuotaTaken.Set && r.ClassQuotaTaken.Value < 0 {
		return errors.New("class_quota_taken must be >= 0")
	}
	// fee non-neg
	if r.ClassRegistrationFeeIDR != nil && r.ClassRegistrationFeeIDR.Set && r.ClassRegistrationFeeIDR.Value != nil && *r.ClassRegistrationFeeIDR.Value < 0 {
		return errors.New("class_registration_fee_idr must be >= 0")
	}
	if r.ClassTuitionFeeIDR != nil && r.ClassTuitionFeeIDR.Set && r.ClassTuitionFeeIDR.Value != nil && *r.ClassTuitionFeeIDR.Value < 0 {
		return errors.New("class_tuition_fee_idr must be >= 0")
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
	if r.ClassImageURL != nil && r.ClassImageURL.Set {
		m.ClassImageURL = r.ClassImageURL.Value
	}
	if r.ClassDeliveryMode != nil && r.ClassDeliveryMode.Set {
		m.ClassDeliveryMode = r.ClassDeliveryMode.Value
	}
	if r.ClassIsActive != nil && r.ClassIsActive.Set {
		m.ClassIsActive = r.ClassIsActive.Value
	}
	if r.ClassTrashURL != nil && r.ClassTrashURL.Set {
		m.ClassTrashURL = r.ClassTrashURL.Value
	}
	if r.ClassDeletePendingUntil != nil && r.ClassDeletePendingUntil.Set {
		m.ClassDeletePendingUntil = r.ClassDeletePendingUntil.Value
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
	ClassImageURL *string `json:"class_image_url,omitempty"`

	ClassDeliveryMode string `json:"class_delivery_mode"`
	ClassIsActive     bool   `json:"class_is_active"`

	ClassTrashURL           *string    `json:"class_trash_url,omitempty"`
	ClassDeletePendingUntil *time.Time `json:"class_delete_pending_until,omitempty"`

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
		ClassImageURL:             m.ClassImageURL,
		ClassDeliveryMode:         m.ClassDeliveryMode,
		ClassIsActive:             m.ClassIsActive,
		ClassTrashURL:             m.ClassTrashURL,
		ClassDeletePendingUntil:   m.ClassDeletePendingUntil,
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
	IsActive     *bool      `query:"is_active"`
	DeliveryMode *string    `query:"delivery_mode"` // enum; lower-case di repo
	Slug         *string    `query:"slug"`          // exact/ilike di repo
	Search       *string    `query:"search"`        // cari di notes (trigram)
	StartGe      *time.Time `query:"start_ge"`
	StartLe      *time.Time `query:"start_le"`
	RegOpenGe    *time.Time `query:"reg_open_ge"`
	RegCloseLe   *time.Time `query:"reg_close_le"`

	Limit  int     `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	SortBy *string `query:"sort_by"`  // created_at|slug|start_date|is_open|is_active|delivery_mode
	Order  *string `query:"order"`    // asc|desc
}

func (q *ListClassQuery) Normalize() {
	if q.DeliveryMode != nil {
		x := strings.ToLower(strings.TrimSpace(*q.DeliveryMode))
		q.DeliveryMode = &x
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
