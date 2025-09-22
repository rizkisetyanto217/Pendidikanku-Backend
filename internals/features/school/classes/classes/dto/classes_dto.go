// file: internals/features/school/academics/classes/dto/class_dto.go
package dto

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	model "masjidku_backend/internals/features/school/classes/classes/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/*
=========================================================
PATCH FIELD — tri-state (absent | null | value)
=========================================================
*/
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

func (p PatchFieldClass[T]) Get() (*T, bool) { return p.Value, p.Present }

/*
=========================================================
REQUEST: CREATE — multipart-ready
=========================================================
*/
type CreateClassRequest struct {
	// Wajib
	ClassMasjidID uuid.UUID `json:"class_masjid_id"              form:"class_masjid_id"              validate:"required"`
	ClassParentID uuid.UUID `json:"class_parent_id"              form:"class_parent_id"              validate:"required"`
	ClassSlug     string    `json:"class_slug"                   form:"class_slug"                   validate:"omitempty,min=1,max=160"`

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

	// Mode & Status
	ClassDeliveryMode *string    `json:"class_delivery_mode,omitempty" form:"class_delivery_mode"` // enum
	ClassStatus       *string    `json:"class_status,omitempty"        form:"class_status"`        // enum
	ClassCompletedAt  *time.Time `json:"class_completed_at,omitempty"  form:"class_completed_at"`

	// Image 2-slot
	ClassImageURL                *string    `json:"class_image_url,omitempty"                  form:"class_image_url"`
	ClassImageObjectKey          *string    `json:"class_image_object_key,omitempty"           form:"class_image_object_key"`
	ClassImageURLOld             *string    `json:"class_image_url_old,omitempty"              form:"class_image_url_old"`
	ClassImageObjectKeyOld       *string    `json:"class_image_object_key_old,omitempty"       form:"class_image_object_key_old"`
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
	// enums
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
	// defaults konsisten dengan DB
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

		ClassImageURL:                r.ClassImageURL,
		ClassImageObjectKey:          r.ClassImageObjectKey,
		ClassImageURLOld:             r.ClassImageURLOld,
		ClassImageObjectKeyOld:       r.ClassImageObjectKeyOld,
		ClassImageDeletePendingUntil: r.ClassImageDeletePendingUntil,
	}
	return m
}

/*
=========================================================
REQUEST: PATCH — tri-state
=========================================================
*/
type PatchClassRequest struct {
	ClassSlug PatchFieldClass[string] `json:"class_slug"`

	// ganti parent kelas (wajib non-null kalau dipatch)
	ClassParentID PatchFieldClass[uuid.UUID] `json:"class_parent_id"`

	ClassStartDate            PatchFieldClass[*time.Time] `json:"class_start_date"`
	ClassEndDate              PatchFieldClass[*time.Time] `json:"class_end_date"`
	ClassTermID               PatchFieldClass[*uuid.UUID] `json:"class_term_id"`
	ClassRegistrationOpensAt  PatchFieldClass[*time.Time] `json:"class_registration_opens_at"`
	ClassRegistrationClosesAt PatchFieldClass[*time.Time] `json:"class_registration_closes_at"`

	ClassQuotaTotal PatchFieldClass[*int] `json:"class_quota_total"`
	ClassQuotaTaken PatchFieldClass[int]  `json:"class_quota_taken"`

	ClassRegistrationFeeIDR PatchFieldClass[*int64]  `json:"class_registration_fee_idr"`
	ClassTuitionFeeIDR      PatchFieldClass[*int64]  `json:"class_tuition_fee_idr"`
	ClassBillingCycle       PatchFieldClass[string]  `json:"class_billing_cycle"`
	ClassProviderProductID  PatchFieldClass[*string] `json:"class_provider_product_id"`
	ClassProviderPriceID    PatchFieldClass[*string] `json:"class_provider_price_id"`

	ClassNotes PatchFieldClass[*string] `json:"class_notes"`

	ClassDeliveryMode PatchFieldClass[*string]    `json:"class_delivery_mode"`
	ClassStatus       PatchFieldClass[string]     `json:"class_status"`
	ClassCompletedAt  PatchFieldClass[*time.Time] `json:"class_completed_at"`

	// Image
	ClassImageURL                PatchFieldClass[*string]    `json:"class_image_url"`
	ClassImageObjectKey          PatchFieldClass[*string]    `json:"class_image_object_key"`
	ClassImageURLOld             PatchFieldClass[*string]    `json:"class_image_url_old"`
	ClassImageObjectKeyOld       PatchFieldClass[*string]    `json:"class_image_object_key_old"`
	ClassImageDeletePendingUntil PatchFieldClass[*time.Time] `json:"class_image_delete_pending_until"`
}

func (r *PatchClassRequest) Normalize() {
	// string (single deref)
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

	// *string (double deref)
	normalizePtrStr := func(f *PatchFieldClass[*string], lower bool) {
		if f.Present && f.Value != nil {
			s := strings.TrimSpace(**f.Value)
			if lower {
				s = strings.ToLower(s)
			}
			if s == "" {
				f.Value = nil
			} else {
				**f.Value = s
			}
		}
	}
	normalizePtrStr(&r.ClassDeliveryMode, true)
	normalizePtrStr(&r.ClassNotes, false)
	normalizePtrStr(&r.ClassProviderProductID, false)
	normalizePtrStr(&r.ClassProviderPriceID, false)
}

func (r *PatchClassRequest) Validate() error {
	// registrasi window
	if r.ClassRegistrationOpensAt.Present && r.ClassRegistrationClosesAt.Present &&
		r.ClassRegistrationOpensAt.Value != nil && r.ClassRegistrationClosesAt.Value != nil {
		open := *r.ClassRegistrationOpensAt.Value
		clos := *r.ClassRegistrationClosesAt.Value
		if clos.Before(*open) {
			return errors.New("class_registration_closes_at must be >= class_registration_opens_at")
		}
	}

	// angka non-negatif
	if r.ClassQuotaTotal.Present && r.ClassQuotaTotal.Value != nil && **r.ClassQuotaTotal.Value < 0 {
		return errors.New("class_quota_total must be >= 0")
	}
	if r.ClassQuotaTaken.Present && r.ClassQuotaTaken.Value != nil && *r.ClassQuotaTaken.Value < 0 {
		return errors.New("class_quota_taken must be >= 0")
	}
	if r.ClassRegistrationFeeIDR.Present && r.ClassRegistrationFeeIDR.Value != nil && **r.ClassRegistrationFeeIDR.Value < 0 {
		return errors.New("class_registration_fee_idr must be >= 0")
	}
	if r.ClassTuitionFeeIDR.Present && r.ClassTuitionFeeIDR.Value != nil && **r.ClassTuitionFeeIDR.Value < 0 {
		return errors.New("class_tuition_fee_idr must be >= 0")
	}

	// enums
	if r.ClassBillingCycle.Present && r.ClassBillingCycle.Value != nil {
		switch *r.ClassBillingCycle.Value {
		case model.BillingCycleOneTime, model.BillingCycleMonthly, model.BillingCycleQuarter, model.BillingCycleSemester, model.BillingCycleYearly:
		default:
			return errors.New("invalid class_billing_cycle")
		}
	}
	if r.ClassDeliveryMode.Present && r.ClassDeliveryMode.Value != nil {
		switch **r.ClassDeliveryMode.Value {
		case model.ClassDeliveryModeOffline, model.ClassDeliveryModeOnline, model.ClassDeliveryModeHybrid:
		default:
			return errors.New("invalid class_delivery_mode")
		}
	}
	if r.ClassStatus.Present && r.ClassStatus.Value != nil {
		switch *r.ClassStatus.Value {
		case model.ClassStatusActive, model.ClassStatusInactive, model.ClassStatusCompleted:
		default:
			return errors.New("invalid class_status")
		}
	}

	// parent id guard
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
	// string
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

	// *time.Time
	assignTimePtr := func(dst **time.Time, f PatchFieldClass[*time.Time]) {
		if f.Present {
			if f.Value == nil {
				*dst = nil
			} else {
				*dst = *f.Value
			}
		}
	}
	assignTimePtr(&m.ClassStartDate, r.ClassStartDate)
	assignTimePtr(&m.ClassEndDate, r.ClassEndDate)
	assignTimePtr(&m.ClassRegistrationOpensAt, r.ClassRegistrationOpensAt)
	assignTimePtr(&m.ClassRegistrationClosesAt, r.ClassRegistrationClosesAt)
	assignTimePtr(&m.ClassCompletedAt, r.ClassCompletedAt)
	assignTimePtr(&m.ClassImageDeletePendingUntil, r.ClassImageDeletePendingUntil)

	// *uuid.UUID
	if r.ClassTermID.Present {
		if r.ClassTermID.Value == nil {
			m.ClassTermID = nil
		} else {
			m.ClassTermID = *r.ClassTermID.Value
		}
	}
	// parent id (non-null saat dipatch)
	if r.ClassParentID.Present && r.ClassParentID.Value != nil {
		m.ClassParentID = *r.ClassParentID.Value
	}

	// kuota
	if r.ClassQuotaTotal.Present {
		if r.ClassQuotaTotal.Value == nil {
			m.ClassQuotaTotal = nil
		} else {
			m.ClassQuotaTotal = *r.ClassQuotaTotal.Value
		}
	}
	if r.ClassQuotaTaken.Present && r.ClassQuotaTaken.Value != nil {
		m.ClassQuotaTaken = *r.ClassQuotaTaken.Value
	}

	// pricing
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

	// *string (nullable)
	assignStrPtr := func(dst **string, f PatchFieldClass[*string]) {
		if f.Present {
			if f.Value == nil {
				*dst = nil
			} else {
				*dst = *f.Value
			}
		}
	}
	assignStrPtr(&m.ClassProviderProductID, r.ClassProviderProductID)
	assignStrPtr(&m.ClassProviderPriceID, r.ClassProviderPriceID)
	assignStrPtr(&m.ClassNotes, r.ClassNotes)
	assignStrPtr(&m.ClassDeliveryMode, r.ClassDeliveryMode)
	assignStrPtr(&m.ClassImageURL, r.ClassImageURL)
	assignStrPtr(&m.ClassImageObjectKey, r.ClassImageObjectKey)
	assignStrPtr(&m.ClassImageURLOld, r.ClassImageURLOld)
	assignStrPtr(&m.ClassImageObjectKeyOld, r.ClassImageObjectKeyOld)
}

/*
=========================================================
RESPONSE DTO
=========================================================
*/
// file: internals/features/school/academics/classes/dto/class_dto.go

type ClassResponse struct {
	// === persis urutan yang diminta ===
	ClassID       uuid.UUID `json:"class_id"`
	ClassMasjidID uuid.UUID `json:"class_masjid_id"`
	ClassParentID uuid.UUID `json:"class_parent_id"`

	ClassSlug string `json:"class_slug"`

	ClassStartDate *time.Time `json:"class_start_date,omitempty"`
	ClassEndDate   *time.Time `json:"class_end_date,omitempty"`

	ClassTermID               *uuid.UUID `json:"class_term_id,omitempty"`
	ClassRegistrationOpensAt  *time.Time `json:"class_registration_opens_at,omitempty"`
	ClassRegistrationClosesAt *time.Time `json:"class_registration_closes_at,omitempty"`

	// (tidak ikutkan quota_total agar output sama persis contoh)
	ClassQuotaTaken int `json:"class_quota_taken"`

	ClassRegistrationFeeIDR *int64 `json:"class_registration_fee_idr,omitempty"`
	ClassTuitionFeeIDR      *int64 `json:"class_tuition_fee_idr,omitempty"`

	ClassBillingCycle string  `json:"class_billing_cycle"`
	ClassDeliveryMode *string `json:"class_delivery_mode,omitempty"`
	ClassStatus       string  `json:"class_status"`

	ClassImageURL                *string    `json:"class_image_url,omitempty"`
	ClassImageObjectKey          *string    `json:"class_image_object_key,omitempty"`
	ClassImageURLOld             *string    `json:"class_image_url_old,omitempty"`
	ClassImageObjectKeyOld       *string    `json:"class_image_object_key_old,omitempty"`
	ClassImageDeletePendingUntil *time.Time `json:"class_image_delete_pending_until,omitempty"`

	ClassCreatedAt time.Time `json:"class_created_at"`
	ClassUpdatedAt time.Time `json:"class_updated_at"`
}

func FromModel(m *model.ClassModel) ClassResponse {
	return ClassResponse{
		ClassID:                      m.ClassID,
		ClassMasjidID:                m.ClassMasjidID,
		ClassParentID:                m.ClassParentID,
		ClassSlug:                    m.ClassSlug,
		ClassStartDate:               m.ClassStartDate,
		ClassEndDate:                 m.ClassEndDate,
		ClassTermID:                  m.ClassTermID,
		ClassRegistrationOpensAt:     m.ClassRegistrationOpensAt,
		ClassRegistrationClosesAt:    m.ClassRegistrationClosesAt,
		ClassQuotaTaken:              m.ClassQuotaTaken,
		ClassRegistrationFeeIDR:      m.ClassRegistrationFeeIDR,
		ClassTuitionFeeIDR:           m.ClassTuitionFeeIDR,
		ClassBillingCycle:            m.ClassBillingCycle,
		ClassDeliveryMode:            m.ClassDeliveryMode,
		ClassStatus:                  m.ClassStatus,
		ClassImageURL:                m.ClassImageURL,
		ClassImageObjectKey:          m.ClassImageObjectKey,
		ClassImageURLOld:             m.ClassImageURLOld,
		ClassImageObjectKeyOld:       m.ClassImageObjectKeyOld,
		ClassImageDeletePendingUntil: m.ClassImageDeletePendingUntil,
		ClassCreatedAt:               m.ClassCreatedAt,
		ClassUpdatedAt:               m.ClassUpdatedAt,
	}
}

/*
=========================================================
QUERY / FILTER DTO (untuk list)
=========================================================
*/
type ListClassQuery struct {
	MasjidID     *uuid.UUID `query:"masjid_id"`
	ParentID     *uuid.UUID `query:"parent_id"`
	TermID       *uuid.UUID `query:"term_id"`
	Status       *string    `query:"status"`
	DeliveryMode *string    `query:"delivery_mode"`
	Slug         *string    `query:"slug"`
	Search       *string    `query:"search"`

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

/*
=========================================================
DECODER: dukung JSON & multipart untuk PatchClassRequest
=========================================================
*/

// DecodePatchClassFromRequest:
//   - multipart/form-data:
//     a) jika ada field "payload" -> unmarshal JSON,
//     b) jika tidak ada -> baca form key-value via DecodePatchClassMultipart.
//   - application/json -> BodyParser biasa.
//
// Setelah parse -> Normalize (Validate tetap dilakukan di controller).
func DecodePatchClassFromRequest(c *fiber.Ctx, out *PatchClassRequest) error {
	ct := strings.ToLower(c.Get("Content-Type"))
	if strings.Contains(ct, "multipart/form-data") {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := json.Unmarshal([]byte(s), out); err != nil {
				return err
			}
		} else if err := DecodePatchClassMultipart(c, out); err != nil {
			return err
		}
	} else {
		if err := c.BodyParser(out); err != nil {
			return err
		}
	}
	out.Normalize()
	return nil
}

// DecodePatchClassMultipart: map form key-value -> tri-state.
func DecodePatchClassMultipart(c *fiber.Ctx, r *PatchClassRequest) error {
	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return fiber.ErrBadRequest
	}

	get := func(k string) (string, bool) {
		if vs, ok := form.Value[k]; ok {
			if len(vs) == 0 {
				return "", true
			}
			return vs[0], true
		}
		return "", false
	}

	// --- helpers ---
	parseTime := func(label, s string) (*time.Time, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		if t, e := time.Parse(time.RFC3339, s); e == nil {
			return &t, nil
		}
		if t, e := time.Parse("2006-01-02", s); e == nil {
			return &t, nil
		}
		return nil, fiber.NewError(fiber.StatusBadRequest, label+" format invalid (pakai RFC3339 atau YYYY-MM-DD)")
	}
	parseInt := func(label, s string) (*int, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		x, e := strconv.Atoi(s)
		if e != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, label+" harus int")
		}
		return &x, nil
	}
	parseInt64 := func(label, s string) (*int64, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		x, e := strconv.ParseInt(s, 10, 64)
		if e != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, label+" harus int64")
		}
		return &x, nil
	}
	setStr := func(field *PatchFieldClass[string], key string) {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			field.Value = &v
		}
	}
	setUUIDNonNull := func(field *PatchFieldClass[uuid.UUID], key string, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			if v == "" {
				field.Value = nil // akan ditolak di Validate()
				return nil
			}
			u, e := uuid.Parse(v)
			if e != nil {
				return fiber.NewError(fiber.StatusBadRequest, label+" invalid UUID")
			}
			field.Value = &u
		}
		return nil
	}
	setUUIDPtr := func(field *PatchFieldClass[*uuid.UUID], key, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			if v == "" {
				field.Value = nil
				return nil
			}
			u, e := uuid.Parse(v)
			if e != nil {
				return fiber.NewError(fiber.StatusBadRequest, label+" invalid UUID")
			}
			ptr := new(*uuid.UUID)
			*ptr = &u
			field.Value = ptr
		}
		return nil
	}
	setTimePtr := func(field *PatchFieldClass[*time.Time], key, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			t, e := parseTime(label, v)
			if e != nil {
				return e
			}
			if t == nil {
				field.Value = nil
			} else {
				ptr := new(*time.Time)
				*ptr = t
				field.Value = ptr
			}
		}
		return nil
	}
	setIntPtr := func(field *PatchFieldClass[*int], key, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			if v == "" {
				field.Value = nil
				return nil
			}
			x, e := parseInt(label, v)
			if e != nil {
				return e
			}
			ptr := new(*int)
			*ptr = x
			field.Value = ptr
		}
		return nil
	}
	setInt := func(field *PatchFieldClass[int], key, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			if v == "" {
				field.Value = nil
				return nil
			}
			x, e := strconv.Atoi(v)
			if e != nil {
				return fiber.NewError(fiber.StatusBadRequest, label+" harus int")
			}
			field.Value = &x
		}
		return nil
	}
	setInt64Ptr := func(field *PatchFieldClass[*int64], key, label string) error {
		if v, ok := get(key); ok {
			field.Present = true
			if v == "" {
				field.Value = nil
				return nil
			}
			x, e := parseInt64(label, v)
			if e != nil {
				return e
			}
			ptr := new(*int64)
			*ptr = x
			field.Value = ptr
		}
		return nil
	}
	setStrPtr := func(field *PatchFieldClass[*string], key string) {
		if v, ok := get(key); ok {
			field.Present = true
			v = strings.TrimSpace(v)
			if v == "" {
				field.Value = nil
			} else {
				ptr := new(*string)
				*ptr = &v
				field.Value = ptr
			}
		}
	}

	// string (non-nullable)
	setStr(&r.ClassSlug, "class_slug")
	setStr(&r.ClassBillingCycle, "class_billing_cycle")
	setStr(&r.ClassStatus, "class_status")

	// uuid (non-null saat dipatch)
	if err := setUUIDNonNull(&r.ClassParentID, "class_parent_id", "class_parent_id"); err != nil {
		return err
	}

	// *uuid.UUID (nullable)
	if err := setUUIDPtr(&r.ClassTermID, "class_term_id", "class_term_id"); err != nil {
		return err
	}

	// *time.Time (nullable)
	if err := setTimePtr(&r.ClassStartDate, "class_start_date", "class_start_date"); err != nil {
		return err
	}
	if err := setTimePtr(&r.ClassEndDate, "class_end_date", "class_end_date"); err != nil {
		return err
	}
	if err := setTimePtr(&r.ClassRegistrationOpensAt, "class_registration_opens_at", "class_registration_opens_at"); err != nil {
		return err
	}
	if err := setTimePtr(&r.ClassRegistrationClosesAt, "class_registration_closes_at", "class_registration_closes_at"); err != nil {
		return err
	}
	if err := setTimePtr(&r.ClassCompletedAt, "class_completed_at", "class_completed_at"); err != nil {
		return err
	}
	if err := setTimePtr(&r.ClassImageDeletePendingUntil, "class_image_delete_pending_until", "class_image_delete_pending_until"); err != nil {
		return err
	}

	// *int (nullable) & int
	if err := setIntPtr(&r.ClassQuotaTotal, "class_quota_total", "class_quota_total"); err != nil {
		return err
	}
	if err := setInt(&r.ClassQuotaTaken, "class_quota_taken", "class_quota_taken"); err != nil {
		return err
	}

	// *int64 (nullable)
	if err := setInt64Ptr(&r.ClassRegistrationFeeIDR, "class_registration_fee_idr", "class_registration_fee_idr"); err != nil {
		return err
	}
	if err := setInt64Ptr(&r.ClassTuitionFeeIDR, "class_tuition_fee_idr", "class_tuition_fee_idr"); err != nil {
		return err
	}

	// *string (nullable)
	setStrPtr(&r.ClassDeliveryMode, "class_delivery_mode")
	setStrPtr(&r.ClassNotes, "class_notes")
	setStrPtr(&r.ClassProviderProductID, "class_provider_product_id")
	setStrPtr(&r.ClassProviderPriceID, "class_provider_price_id")
	setStrPtr(&r.ClassImageURL, "class_image_url")
	setStrPtr(&r.ClassImageObjectKey, "class_image_object_key")
	setStrPtr(&r.ClassImageURLOld, "class_image_url_old")
	setStrPtr(&r.ClassImageObjectKeyOld, "class_image_object_key_old")

	return nil
}
