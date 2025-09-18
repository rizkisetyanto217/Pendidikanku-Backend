// file: internals/features/school/academics/classes/dto/class_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/google/uuid"
)

/* =========================================================
   PATCH FIELD — tri-state (absent | null | value)
   Disamakan dengan class_parents
   ========================================================= */
type PatchFieldClass[T any] struct {
	Present bool
	Value   *T
}

func (p *PatchFieldClass[T]) UnmarshalJSON(b []byte) error {
	p.Present = true
	if string(b) == "null" {
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

// Optional getter
func (p PatchFieldClass[T]) Get() (*T, bool) { return p.Value, p.Present }

/* =========================================================
   REQUEST: CREATE (sinkron dengan DDL & model) — multipart-ready
   ========================================================= */

type CreateClassRequest struct {
	// Wajib
	ClassMasjidID uuid.UUID `json:"class_masjid_id"              form:"class_masjid_id"              validate:"required"`
	ClassParentID uuid.UUID `json:"class_parent_id"              form:"class_parent_id"              validate:"required"`
    ClassSlug string `json:"class_slug" form:"class_slug" validate:"omitempty,min=1,max=160"`

	// Periode
	ClassStartDate *time.Time `json:"class_start_date,omitempty"         form:"class_start_date"`
	ClassEndDate   *time.Time `json:"class_end_date,omitempty"           form:"class_end_date"`

	// Registrasi / Term
	ClassTermID               *uuid.UUID `json:"class_term_id,omitempty"                form:"class_term_id"`
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

	// Catatan
	ClassNotes *string `json:"class_notes,omitempty" form:"class_notes"`

	// Mode & Status (delivery_mode nullable)
	ClassDeliveryMode *string    `json:"class_delivery_mode,omitempty" form:"class_delivery_mode"` // enum
	ClassStatus       *string    `json:"class_status,omitempty"        form:"class_status"`        // enum: active|inactive|completed
	ClassCompletedAt  *time.Time `json:"class_completed_at,omitempty"  form:"class_completed_at"`

	// Image 2-slot (opsional; biasanya diisi via flow upload)
	ClassImageURL                *string    `json:"class_image_url,omitempty"                 form:"class_image_url"`
	ClassImageObjectKey          *string    `json:"class_image_object_key,omitempty"          form:"class_image_object_key"`
	ClassImageURLOld             *string    `json:"class_image_url_old,omitempty"             form:"class_image_url_old"`
	ClassImageObjectKeyOld       *string    `json:"class_image_object_key_old,omitempty"      form:"class_image_object_key_old"`
	ClassImageDeletePendingUntil *time.Time `json:"class_image_delete_pending_until,omitempty" form:"class_image_delete_pending_until"`
}

func (r *CreateClassRequest) Normalize() {
	// slug lower + trim
	r.ClassSlug = strings.TrimSpace(strings.ToLower(r.ClassSlug))

	// enum strings lower
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

	// trim optional strings
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
}

func (r *CreateClassRequest) Validate() error {
	if r.ClassMasjidID == uuid.Nil {
		return errors.New("class_masjid_id required")
	}
	if r.ClassParentID == uuid.Nil {
		return errors.New("class_parent_id required")
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
	// defaults yang konsisten dengan DB
	billing := model.BillingCycleMonthly
	if r.ClassBillingCycle != nil && *r.ClassBillingCycle != "" {
		billing = *r.ClassBillingCycle
	}
	var delivery *string
	if r.ClassDeliveryMode != nil && *r.ClassDeliveryMode != "" {
		d := *r.ClassDeliveryMode
		delivery = &d
	}
	status := model.ClassStatusActive
	if r.ClassStatus != nil && *r.ClassStatus != "" {
		status = *r.ClassStatus
	}

	m := &model.ClassModel{
		ClassMasjidID:             r.ClassMasjidID,
		ClassParentID:             r.ClassParentID,
		ClassSlug:                 r.ClassSlug,
		ClassStartDate:            r.ClassStartDate,
		ClassEndDate:              r.ClassEndDate,
		ClassTermID:               r.ClassTermID,
		ClassRegistrationOpensAt:  r.ClassRegistrationOpensAt,
		ClassRegistrationClosesAt: r.ClassRegistrationClosesAt,
		ClassQuotaTotal:           r.ClassQuotaTotal,
		ClassRegistrationFeeIDR:   r.ClassRegistrationFeeIDR,
		ClassTuitionFeeIDR:        r.ClassTuitionFeeIDR,
		ClassBillingCycle:         billing,
		ClassProviderProductID:    r.ClassProviderProductID,
		ClassProviderPriceID:      r.ClassProviderPriceID,
		ClassNotes:                r.ClassNotes,
		ClassDeliveryMode:         delivery,
		ClassStatus:               status,
		ClassCompletedAt:          r.ClassCompletedAt,

		// Image fields (opsional)
		ClassImageURL:                r.ClassImageURL,
		ClassImageObjectKey:          r.ClassImageObjectKey,
		ClassImageURLOld:             r.ClassImageURLOld,
		ClassImageObjectKeyOld:       r.ClassImageObjectKeyOld,
		ClassImageDeletePendingUntil: r.ClassImageDeletePendingUntil,
	}
	return m
}

/* =========================================================
   REQUEST: PATCH (partial / tri-state) — diseragamkan
   ========================================================= */
type PatchClassRequest struct {
    ClassSlug PatchFieldClass[string] `json:"class_slug"`

    // ⬇️ BARU: dukung ganti parent kelas (wajib non-null kalau dipatch)
    ClassParentID PatchFieldClass[uuid.UUID] `json:"class_parent_id"`

    ClassStartDate            PatchFieldClass[*time.Time] `json:"class_start_date"`
    ClassEndDate              PatchFieldClass[*time.Time] `json:"class_end_date"`
    ClassTermID               PatchFieldClass[*uuid.UUID] `json:"class_term_id"`
    ClassRegistrationOpensAt  PatchFieldClass[*time.Time] `json:"class_registration_opens_at"`
    ClassRegistrationClosesAt PatchFieldClass[*time.Time] `json:"class_registration_closes_at"`

    ClassQuotaTotal PatchFieldClass[*int] `json:"class_quota_total"`
    ClassQuotaTaken PatchFieldClass[int]  `json:"class_quota_taken"`

    ClassRegistrationFeeIDR PatchFieldClass[*int64] `json:"class_registration_fee_idr"`
    ClassTuitionFeeIDR      PatchFieldClass[*int64] `json:"class_tuition_fee_idr"`
    ClassBillingCycle       PatchFieldClass[string] `json:"class_billing_cycle"`
    ClassProviderProductID  PatchFieldClass[*string] `json:"class_provider_product_id"`
    ClassProviderPriceID    PatchFieldClass[*string] `json:"class_provider_price_id"`

    ClassNotes PatchFieldClass[*string] `json:"class_notes"`

    ClassDeliveryMode PatchFieldClass[*string] `json:"class_delivery_mode"`
    ClassStatus       PatchFieldClass[string]  `json:"class_status"`
    ClassCompletedAt  PatchFieldClass[*time.Time] `json:"class_completed_at"`

    // Image (opsional)
    ClassImageURL                PatchFieldClass[*string]    `json:"class_image_url"`
    ClassImageObjectKey          PatchFieldClass[*string]    `json:"class_image_object_key"`
    ClassImageURLOld             PatchFieldClass[*string]    `json:"class_image_url_old"`
    ClassImageObjectKeyOld       PatchFieldClass[*string]    `json:"class_image_object_key_old"`
    ClassImageDeletePendingUntil PatchFieldClass[*time.Time] `json:"class_image_delete_pending_until"`
}


func (r *PatchClassRequest) Normalize() {
	// ---- T=string (Value = *string; deref sekali) ----
	if r.ClassSlug.Present && r.ClassSlug.Value != nil {
		s := strings.TrimSpace(strings.ToLower(*r.ClassSlug.Value))
		r.ClassSlug.Value = &s
	}
	if r.ClassBillingCycle.Present && r.ClassBillingCycle.Value != nil {
		s := strings.ToLower(strings.TrimSpace(*r.ClassBillingCycle.Value))
		r.ClassBillingCycle.Value = &s
	}
	if r.ClassStatus.Present && r.ClassStatus.Value != nil {
		s := strings.ToLower(strings.TrimSpace(*r.ClassStatus.Value))
		r.ClassStatus.Value = &s
	}

	// ---- T=*string (Value = **string; deref dua kali) ----
	if r.ClassDeliveryMode.Present && r.ClassDeliveryMode.Value != nil {
		s := strings.ToLower(strings.TrimSpace(**r.ClassDeliveryMode.Value))
		if s == "" {
			// kosong → anggap clear
			r.ClassDeliveryMode.Value = nil
		} else {
			**r.ClassDeliveryMode.Value = s
		}
	}
	if r.ClassNotes.Present && r.ClassNotes.Value != nil {
		s := strings.TrimSpace(**r.ClassNotes.Value)
		if s == "" {
			r.ClassNotes.Value = nil
		} else {
			**r.ClassNotes.Value = s
		}
	}
	if r.ClassProviderProductID.Present && r.ClassProviderProductID.Value != nil {
		s := strings.TrimSpace(**r.ClassProviderProductID.Value)
		if s == "" {
			r.ClassProviderProductID.Value = nil
		} else {
			**r.ClassProviderProductID.Value = s
		}
	}
	if r.ClassProviderPriceID.Present && r.ClassProviderPriceID.Value != nil {
		s := strings.TrimSpace(**r.ClassProviderPriceID.Value)
		if s == "" {
			r.ClassProviderPriceID.Value = nil
		} else {
			**r.ClassProviderPriceID.Value = s
		}
	}
}

func (r *PatchClassRequest) Validate() error {
	// ---- registrasi window ----
	if r.ClassRegistrationOpensAt.Present && r.ClassRegistrationClosesAt.Present &&
		r.ClassRegistrationOpensAt.Value != nil && r.ClassRegistrationClosesAt.Value != nil {
		open := *r.ClassRegistrationOpensAt.Value   // *time.Time
		clos := *r.ClassRegistrationClosesAt.Value // *time.Time
		if clos.Before(*open) {
			return errors.New("class_registration_closes_at must be >= class_registration_opens_at")
		}
	}

	// ---- angka non-negatif ----
	// ClassQuotaTotal: T=*int -> Value=**int
	if r.ClassQuotaTotal.Present && r.ClassQuotaTotal.Value != nil && **r.ClassQuotaTotal.Value < 0 {
		return errors.New("class_quota_total must be >= 0")
	}
	// ClassQuotaTaken: jika T=int -> Value=*int (sekali deref sudah cukup)
	if r.ClassQuotaTaken.Present && r.ClassQuotaTaken.Value != nil && *r.ClassQuotaTaken.Value < 0 {
		return errors.New("class_quota_taken must be >= 0")
	}
	// Fee: T=*int64 -> Value=**int64
	if r.ClassRegistrationFeeIDR.Present && r.ClassRegistrationFeeIDR.Value != nil && **r.ClassRegistrationFeeIDR.Value < int64(0) {
		return errors.New("class_registration_fee_idr must be >= 0")
	}
	if r.ClassTuitionFeeIDR.Present && r.ClassTuitionFeeIDR.Value != nil && **r.ClassTuitionFeeIDR.Value < int64(0) {
		return errors.New("class_tuition_fee_idr must be >= 0")
	}

	// ---- enums ----
	// billing cycle: T=string -> Value=*string
	if r.ClassBillingCycle.Present && r.ClassBillingCycle.Value != nil {
		switch *r.ClassBillingCycle.Value {
		case model.BillingCycleOneTime, model.BillingCycleMonthly, model.BillingCycleQuarter, model.BillingCycleSemester, model.BillingCycleYearly:
		default:
			return errors.New("invalid class_billing_cycle")
		}
	}
	// delivery mode: T=*string -> Value=**string
	if r.ClassDeliveryMode.Present && r.ClassDeliveryMode.Value != nil {
		switch **r.ClassDeliveryMode.Value {
		case model.ClassDeliveryModeOffline, model.ClassDeliveryModeOnline, model.ClassDeliveryModeHybrid:
		default:
			return errors.New("invalid class_delivery_mode")
		}
	}
	// status: T=string -> Value=*string
	if r.ClassStatus.Present && r.ClassStatus.Value != nil {
		switch *r.ClassStatus.Value {
		case model.ClassStatusActive, model.ClassStatusInactive, model.ClassStatusCompleted:
		default:
			return errors.New("invalid class_status")
		}
	}

	// ⬇️ BARU: class_parent_id tidak boleh null atau uuid.Nil kalau dipatch
    if r.ClassParentID.Present {
        if r.ClassParentID.Value == nil {
            return errors.New("class_parent_id cannot be null")
        }
        if *r.ClassParentID.Value == uuid.Nil {
            return errors.New("class_parent_id is invalid")
        }
    }

	return nil
}


func (r *PatchClassRequest) Apply(m *model.ClassModel) {
	// ---- T=string (Value = *string) ----
	if r.ClassSlug.Present && r.ClassSlug.Value != nil {
		m.ClassSlug = *r.ClassSlug.Value
	}
	if r.ClassBillingCycle.Present && r.ClassBillingCycle.Value != nil {
		if s := strings.TrimSpace(*r.ClassBillingCycle.Value); s != "" {
			m.ClassBillingCycle = s
		}
	}
	if r.ClassStatus.Present && r.ClassStatus.Value != nil {
		if s := strings.TrimSpace(*r.ClassStatus.Value); s != "" {
			m.ClassStatus = s
		}
	}

	// ---- T=*time.Time (Value = **time.Time) ----
	if r.ClassStartDate.Present {
		if r.ClassStartDate.Value == nil {
			m.ClassStartDate = nil
		} else {
			m.ClassStartDate = *r.ClassStartDate.Value
		}
	}
	if r.ClassEndDate.Present {
		if r.ClassEndDate.Value == nil {
			m.ClassEndDate = nil
		} else {
			m.ClassEndDate = *r.ClassEndDate.Value
		}
	}
	if r.ClassRegistrationOpensAt.Present {
		if r.ClassRegistrationOpensAt.Value == nil {
			m.ClassRegistrationOpensAt = nil
		} else {
			m.ClassRegistrationOpensAt = *r.ClassRegistrationOpensAt.Value
		}
	}
	if r.ClassRegistrationClosesAt.Present {
		if r.ClassRegistrationClosesAt.Value == nil {
			m.ClassRegistrationClosesAt = nil
		} else {
			m.ClassRegistrationClosesAt = *r.ClassRegistrationClosesAt.Value
		}
	}
	if r.ClassCompletedAt.Present {
		if r.ClassCompletedAt.Value == nil {
			m.ClassCompletedAt = nil
		} else {
			m.ClassCompletedAt = *r.ClassCompletedAt.Value
		}
	}

	// ---- T=*uuid.UUID (Value = **uuid.UUID) ----
	if r.ClassTermID.Present {
		if r.ClassTermID.Value == nil {
			m.ClassTermID = nil
		} else {
			m.ClassTermID = *r.ClassTermID.Value
		}
	}

	// ⬇️ BARU
    if r.ClassParentID.Present && r.ClassParentID.Value != nil {
        m.ClassParentID = *r.ClassParentID.Value
    }

	// ---- Kuota ----
	// T=*int (Value = **int)
	if r.ClassQuotaTotal.Present {
		if r.ClassQuotaTotal.Value == nil {
			m.ClassQuotaTotal = nil
		} else {
			m.ClassQuotaTotal = *r.ClassQuotaTotal.Value
		}
	}
	// T=int (Value = *int)
	if r.ClassQuotaTaken.Present && r.ClassQuotaTaken.Value != nil {
		m.ClassQuotaTaken = *r.ClassQuotaTaken.Value
	}

	// ---- Pricing ----
	// T=*int64 (Value = **int64)
	if r.ClassRegistrationFeeIDR.Present {
		if r.ClassRegistrationFeeIDR.Value == nil {
			m.ClassRegistrationFeeIDR = nil
		} else {
			m.ClassRegistrationFeeIDR = *r.ClassRegistrationFeeIDR.Value
		}
	}
	if r.ClassTuitionFeeIDR.Present {
		if r.ClassTuitionFeeIDR.Value == nil {
			m.ClassTuitionFeeIDR = nil
		} else {
			m.ClassTuitionFeeIDR = *r.ClassTuitionFeeIDR.Value
		}
	}

	// ---- Optional strings (pointer) → T=*string (Value=**string) ----
	if r.ClassProviderProductID.Present {
		if r.ClassProviderProductID.Value == nil {
			m.ClassProviderProductID = nil
		} else {
			m.ClassProviderProductID = *r.ClassProviderProductID.Value
		}
	}
	if r.ClassProviderPriceID.Present {
		if r.ClassProviderPriceID.Value == nil {
			m.ClassProviderPriceID = nil
		} else {
			m.ClassProviderPriceID = *r.ClassProviderPriceID.Value
		}
	}
	if r.ClassNotes.Present {
		if r.ClassNotes.Value == nil {
			m.ClassNotes = nil
		} else {
			m.ClassNotes = *r.ClassNotes.Value
		}
	}
	// delivery_mode nullable (pointer)
	if r.ClassDeliveryMode.Present {
		if r.ClassDeliveryMode.Value == nil {
			m.ClassDeliveryMode = nil
		} else {
			m.ClassDeliveryMode = *r.ClassDeliveryMode.Value
		}
	}

	// ---- Image fields (semua pointer) → T=*string (Value=**string) ----
	if r.ClassImageURL.Present {
		if r.ClassImageURL.Value == nil {
			m.ClassImageURL = nil
		} else {
			m.ClassImageURL = *r.ClassImageURL.Value
		}
	}
	if r.ClassImageObjectKey.Present {
		if r.ClassImageObjectKey.Value == nil {
			m.ClassImageObjectKey = nil
		} else {
			m.ClassImageObjectKey = *r.ClassImageObjectKey.Value
		}
	}
	if r.ClassImageURLOld.Present {
		if r.ClassImageURLOld.Value == nil {
			m.ClassImageURLOld = nil
		} else {
			m.ClassImageURLOld = *r.ClassImageURLOld.Value
		}
	}
	if r.ClassImageObjectKeyOld.Present {
		if r.ClassImageObjectKeyOld.Value == nil {
			m.ClassImageObjectKeyOld = nil
		} else {
			m.ClassImageObjectKeyOld = *r.ClassImageObjectKeyOld.Value
		}
	}
	if r.ClassImageDeletePendingUntil.Present {
		if r.ClassImageDeletePendingUntil.Value == nil {
			m.ClassImageDeletePendingUntil = nil
		} else {
			m.ClassImageDeletePendingUntil = *r.ClassImageDeletePendingUntil.Value
		}
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
	ClassRegistrationOpensAt  *time.Time `json:"class_registration_opens_at,omitempty"`
	ClassRegistrationClosesAt *time.Time `json:"class_registration_closes_at,omitempty"`

	ClassQuotaTotal *int `json:"class_quota_total,omitempty"`
	ClassQuotaTaken int   `json:"class_quota_taken"`

	ClassRegistrationFeeIDR *int64  `json:"class_registration_fee_idr,omitempty"`
	ClassTuitionFeeIDR      *int64  `json:"class_tuition_fee_idr,omitempty"`
	ClassBillingCycle       string  `json:"class_billing_cycle"`
	ClassProviderProductID  *string `json:"class_provider_product_id,omitempty"`
	ClassProviderPriceID    *string `json:"class_provider_price_id,omitempty"`

	ClassNotes *string `json:"class_notes,omitempty"`

	ClassDeliveryMode *string    `json:"class_delivery_mode,omitempty"` // nullable
	ClassStatus       string     `json:"class_status"`
	ClassCompletedAt  *time.Time `json:"class_completed_at,omitempty"`

	// Image 2-slot
	ClassImageURL                *string    `json:"class_image_url,omitempty"`
	ClassImageObjectKey          *string    `json:"class_image_object_key,omitempty"`
	ClassImageURLOld             *string    `json:"class_image_url_old,omitempty"`
	ClassImageObjectKeyOld       *string    `json:"class_image_object_key_old,omitempty"`
	ClassImageDeletePendingUntil *time.Time `json:"class_image_delete_pending_until,omitempty"`

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

		ClassImageURL:                m.ClassImageURL,
		ClassImageObjectKey:          m.ClassImageObjectKey,
		ClassImageURLOld:             m.ClassImageURLOld,
		ClassImageObjectKeyOld:       m.ClassImageObjectKeyOld,
		ClassImageDeletePendingUntil: m.ClassImageDeletePendingUntil,

		ClassCreatedAt: m.ClassCreatedAt,
		ClassUpdatedAt: m.ClassUpdatedAt,
		ClassDeletedAt: delAt,
	}
}

/* =========================================================
   QUERY / FILTER DTO (untuk list)
   ========================================================= */

type ListClassQuery struct {
	MasjidID     *uuid.UUID `query:"masjid_id"`
	ParentID     *uuid.UUID `query:"parent_id"`
	TermID       *uuid.UUID `query:"term_id"`
	Status       *string    `query:"status"`        // enum: active|inactive|completed
	DeliveryMode *string    `query:"delivery_mode"` // enum; lower-case di repo
	Slug         *string    `query:"slug"`          // exact/ilike di repo
	Search       *string    `query:"search"`        // cari di notes (trigram)

	StartGe    *time.Time `query:"start_ge"`
	StartLe    *time.Time `query:"start_le"`
	RegOpenGe  *time.Time `query:"reg_open_ge"`
	RegCloseLe *time.Time `query:"reg_close_le"`

	CompletedGe *time.Time `query:"completed_ge"`
	CompletedLe *time.Time `query:"completed_le"`

	Limit  int     `query:"limit"  validate:"omitempty,min=1,max=200"`
	Offset int     `query:"offset" validate:"omitempty,min=0"`
	SortBy *string `query:"sort_by"` // created_at|slug|start_date|status|delivery_mode
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
