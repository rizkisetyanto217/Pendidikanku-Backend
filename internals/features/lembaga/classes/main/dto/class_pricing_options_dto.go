// file: internals/features/lembaga/classes/main/dto/class_pricing_options.go
package dto

import (
	"errors"
	"strings"
	"time"

	models "masjidku_backend/internals/features/lembaga/classes/main/model"

	"github.com/google/uuid"
)

const (
	PriceTypeOneTime   = "ONE_TIME"
	PriceTypeRecurring = "RECURRING"
)

/* ===================== REQUEST DTO ===================== */

// CREATE
type CreateClassPricingOptionReq struct {
	ClassPricingOptionsClassID          uuid.UUID `json:"class_pricing_options_class_id"           validate:"required"`
	ClassPricingOptionsLabel            string    `json:"class_pricing_options_label"              validate:"required,max=80"`
	ClassPricingOptionsPriceType        string    `json:"class_pricing_options_price_type"         validate:"required,oneof=ONE_TIME RECURRING"`
	ClassPricingOptionsAmountIDR        int       `json:"class_pricing_options_amount_idr"         validate:"required,min=0"`
	ClassPricingOptionsRecurrenceMonths *int      `json:"class_pricing_options_recurrence_months"  validate:"omitempty,oneof=1 3 6 12"`
}

// UPDATE (PATCH)
type UpdateClassPricingOptionReq struct {
	ClassPricingOptionsLabel            *string `json:"class_pricing_options_label"              validate:"omitempty,max=80"`
	ClassPricingOptionsPriceType        *string `json:"class_pricing_options_price_type"         validate:"omitempty,oneof=ONE_TIME RECURRING"`
	ClassPricingOptionsAmountIDR        *int    `json:"class_pricing_options_amount_idr"         validate:"omitempty,min=0"`
	ClassPricingOptionsRecurrenceMonths *int    `json:"class_pricing_options_recurrence_months"  validate:"omitempty,oneof=1 3 6 12"`
}

// PUT (full replace)
type PutClassPricingOptionReq struct {
	ClassPricingOptionsLabel            string `json:"class_pricing_options_label"              validate:"required,max=80"`
	ClassPricingOptionsPriceType        string `json:"class_pricing_options_price_type"         validate:"required,oneof=ONE_TIME RECURRING"`
	ClassPricingOptionsAmountIDR        int    `json:"class_pricing_options_amount_idr"         validate:"required,min=0"`
	ClassPricingOptionsRecurrenceMonths *int   `json:"class_pricing_options_recurrence_months"  validate:"omitempty,oneof=1 3 6 12"`
}

/* ===================== CROSS-FIELD VALIDATION ===================== */

func (r *CreateClassPricingOptionReq) NormalizeAndValidateCombo() error {
	r.ClassPricingOptionsPriceType = strings.ToUpper(strings.TrimSpace(r.ClassPricingOptionsPriceType))
	r.ClassPricingOptionsLabel = strings.TrimSpace(r.ClassPricingOptionsLabel)

	switch r.ClassPricingOptionsPriceType {
	case PriceTypeOneTime:
		if r.ClassPricingOptionsRecurrenceMonths != nil {
			return errors.New("ONE_TIME harus recurrence_months = null")
		}
	case PriceTypeRecurring:
		if r.ClassPricingOptionsRecurrenceMonths == nil {
			return errors.New("RECURRING butuh recurrence_months (1/3/6/12)")
		}
	default:
		return errors.New("price_type tidak valid")
	}
	return nil
}

func (r *UpdateClassPricingOptionReq) NormalizeAndValidateCombo(currType string) error {
	if r.ClassPricingOptionsPriceType != nil {
		*r.ClassPricingOptionsPriceType = strings.ToUpper(strings.TrimSpace(*r.ClassPricingOptionsPriceType))
	}
	if r.ClassPricingOptionsLabel != nil {
		lbl := strings.TrimSpace(*r.ClassPricingOptionsLabel)
		r.ClassPricingOptionsLabel = &lbl
	}

	pt := strings.ToUpper(strings.TrimSpace(currType))
	if r.ClassPricingOptionsPriceType != nil {
		pt = *r.ClassPricingOptionsPriceType
	}

	switch pt {
	case PriceTypeOneTime:
		if r.ClassPricingOptionsRecurrenceMonths != nil {
			return errors.New("ONE_TIME harus recurrence_months = null")
		}
	case PriceTypeRecurring:
		// valid, angka sudah divalidasi via tag
	default:
		return errors.New("price_type tidak valid")
	}
	return nil
}

func (r *PutClassPricingOptionReq) NormalizeAndValidateCombo() error {
	r.ClassPricingOptionsPriceType = strings.ToUpper(strings.TrimSpace(r.ClassPricingOptionsPriceType))
	r.ClassPricingOptionsLabel = strings.TrimSpace(r.ClassPricingOptionsLabel)

	switch r.ClassPricingOptionsPriceType {
	case PriceTypeOneTime:
		if r.ClassPricingOptionsRecurrenceMonths != nil {
			return errors.New("ONE_TIME harus recurrence_months = null")
		}
	case PriceTypeRecurring:
		if r.ClassPricingOptionsRecurrenceMonths == nil {
			return errors.New("RECURRING butuh recurrence_months (1/3/6/12)")
		}
	default:
		return errors.New("price_type tidak valid")
	}
	return nil
}

/* ===================== RESPONSE DTO ===================== */

type ClassPricingOptionResp struct {
	ClassPricingOptionsID              uuid.UUID  `json:"class_pricing_options_id"`
	ClassPricingOptionsClassID         uuid.UUID  `json:"class_pricing_options_class_id"`
	ClassPricingOptionsLabel           string     `json:"class_pricing_options_label"`
	ClassPricingOptionsPriceType       string     `json:"class_pricing_options_price_type"`
	ClassPricingOptionsAmountIDR       int        `json:"class_pricing_options_amount_idr"`
	ClassPricingOptionsRecurrenceMonths *int      `json:"class_pricing_options_recurrence_months,omitempty"`
	ClassPricingOptionsCreatedAt       time.Time  `json:"class_pricing_options_created_at"`
	ClassPricingOptionsUpdatedAt       *time.Time `json:"class_pricing_options_updated_at,omitempty"`
	ClassPricingOptionsDeletedAt       *time.Time `json:"class_pricing_options_deleted_at,omitempty"`
}

/* ===================== MAPPING ===================== */

func (r *CreateClassPricingOptionReq) ToModel() models.ClassPricingOption {
	return models.ClassPricingOption{
		ClassPricingOptionsClassID:          r.ClassPricingOptionsClassID,
		ClassPricingOptionsLabel:            r.ClassPricingOptionsLabel,
		ClassPricingOptionsPriceType:        strings.ToUpper(r.ClassPricingOptionsPriceType),
		ClassPricingOptionsAmountIDR:        r.ClassPricingOptionsAmountIDR,
		ClassPricingOptionsRecurrenceMonths: r.ClassPricingOptionsRecurrenceMonths,
	}
}

func (r *UpdateClassPricingOptionReq) Apply(m *models.ClassPricingOption) {
	if r.ClassPricingOptionsLabel != nil {
		m.ClassPricingOptionsLabel = *r.ClassPricingOptionsLabel
	}
	if r.ClassPricingOptionsPriceType != nil {
		m.ClassPricingOptionsPriceType = strings.ToUpper(*r.ClassPricingOptionsPriceType)
		if m.ClassPricingOptionsPriceType == PriceTypeOneTime {
			m.ClassPricingOptionsRecurrenceMonths = nil
		}
	}
	if r.ClassPricingOptionsAmountIDR != nil {
		m.ClassPricingOptionsAmountIDR = *r.ClassPricingOptionsAmountIDR
	}
	if r.ClassPricingOptionsRecurrenceMonths != nil {
		m.ClassPricingOptionsRecurrenceMonths = r.ClassPricingOptionsRecurrenceMonths
	}
}

func FromModel(m models.ClassPricingOption) ClassPricingOptionResp {
	return ClassPricingOptionResp{
		ClassPricingOptionsID:               m.ClassPricingOptionsID,
		ClassPricingOptionsClassID:          m.ClassPricingOptionsClassID,
		ClassPricingOptionsLabel:            m.ClassPricingOptionsLabel,
		ClassPricingOptionsPriceType:        m.ClassPricingOptionsPriceType,
		ClassPricingOptionsAmountIDR:        m.ClassPricingOptionsAmountIDR,
		ClassPricingOptionsRecurrenceMonths: m.ClassPricingOptionsRecurrenceMonths,
		ClassPricingOptionsCreatedAt:        m.ClassPricingOptionsCreatedAt,
		ClassPricingOptionsUpdatedAt:        m.ClassPricingOptionsUpdatedAt,
		ClassPricingOptionsDeletedAt:        m.ClassPricingOptionsDeletedAt,
	}
}
