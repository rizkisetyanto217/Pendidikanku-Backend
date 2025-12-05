package dto

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"madinahsalam_backend/internals/features/finance/payments/model"
	svc "madinahsalam_backend/internals/features/finance/payments/service"
)

/* =========================================================
   DTO: Bundle registration + payment
========================================================= */

type BundleItem struct {
	ClassID           uuid.UUID `json:"class_id" validate:"required"`
	FeeRuleOptionCode *string   `json:"fee_rule_option_code,omitempty"` // pilih kode option untuk kelas ini
	CustomAmountIDR   *int64    `json:"custom_amount_idr,omitempty"`    // nominal custom utk kelas ini (≥ min option)
	CustomLabel       *string   `json:"custom_label,omitempty"`         // label custom (opsional)
}

type CreateRegistrationAndPaymentRequest struct {
	// Rekomendasi baru (per-kelas):
	Items []BundleItem `json:"items" validate:"omitempty,min=1,dive"`

	// Backward-compat:
	ClassIDs          []uuid.UUID `json:"class_ids" validate:"omitempty,min=1,dive,required"`
	ClassID           uuid.UUID   `json:"class_id"  validate:"-"`
	FeeRuleID         uuid.UUID   `json:"fee_rule_id" validate:"required"`
	FeeRuleOptionCode *string     `json:"fee_rule_option_code,omitempty"` // fallback untuk semua kelas
	CustomAmountIDR   *int64      `json:"custom_amount_idr,omitempty"`    // TOTAL fallback → dibagi rata
	CustomLabel       *string     `json:"custom_label,omitempty"`

	// Pembayaran
	PaymentMethod          *model.PaymentMethod          `json:"payment_method"`
	PaymentGatewayProvider *model.PaymentGatewayProvider `json:"payment_gateway_provider"`
	PaymentExternalID      *string                       `json:"payment_external_id"`

	Customer *svc.CustomerInput `json:"customer,omitempty"`
	Notes    string             `json:"notes,omitempty"`
}

/*
NormalizeItems:
- Kalau Items kosong → isi dari ClassIDs / ClassID lama (backward-compat)
- Rule sama seperti yang tadi di controller
*/
func (r *CreateRegistrationAndPaymentRequest) NormalizeItems() error {
	if len(r.Items) > 0 {
		// sudah diisi eksplisit, tidak diapa-apakan
		return nil
	}

	ids := r.ClassIDs
	if len(ids) == 0 && r.ClassID != uuid.Nil {
		ids = []uuid.UUID{r.ClassID}
	}

	for _, cid := range ids {
		b := BundleItem{ClassID: cid}

		if r.CustomAmountIDR != nil {
			// custom amount global → turun ke tiap item
			b.CustomAmountIDR = r.CustomAmountIDR
			b.CustomLabel = r.CustomLabel
		} else if r.FeeRuleOptionCode != nil && strings.TrimSpace(*r.FeeRuleOptionCode) != "" {
			// option code global → turun ke tiap item
			b.FeeRuleOptionCode = r.FeeRuleOptionCode
		}

		r.Items = append(r.Items, b)
	}

	if len(r.Items) == 0 {
		return fmt.Errorf("items / class_ids wajib")
	}
	return nil
}

/* =========================================================
   Fee rule amount options helper (dipakai controller)
========================================================= */

type FeeRuleAmountOption struct {
	Code    string `json:"code"`
	Label   string `json:"label"`
	Amount  int64  `json:"amount"`
	Default *bool  `json:"default,omitempty"`
}

func FindAmountOptionByCode(opts []FeeRuleAmountOption, code string) *FeeRuleAmountOption {
	c := strings.ToUpper(strings.TrimSpace(code))
	for i := range opts {
		if strings.ToUpper(opts[i].Code) == c {
			return &opts[i]
		}
	}
	return nil
}

func FirstDefaultAmountOption(opts []FeeRuleAmountOption) *FeeRuleAmountOption {
	for i := range opts {
		if opts[i].Default != nil && *opts[i].Default {
			return &opts[i]
		}
	}
	return nil
}

func MinAmountOption(opts []FeeRuleAmountOption) int64 {
	if len(opts) == 0 {
		return 0
	}
	m := opts[0].Amount
	for i := 1; i < len(opts); i++ {
		if opts[i].Amount < m {
			m = opts[i].Amount
		}
	}
	return m
}
